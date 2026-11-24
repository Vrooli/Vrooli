import { createHash, createPublicKey, generateKeyPairSync } from 'node:crypto';
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import type { SessionState } from '../../../src/types/session';
import {
  installManagedExtension,
  loadManagedExtension,
  reloadManagedExtension,
  unloadManagedExtension,
  validateManagedExtension,
  inventoryManagedExtensions,
} from '../../../src/session/managed-extensions';

const stagingRoot = join(tmpdir(), 'vrooli-bas-managed-extensions');

function extensionID(key: string): string {
  const der = Buffer.from(key, 'base64');
  const digest = createHash('sha256').update(der).digest();
  const alphabet = 'abcdefghijklmnop';
  return [...digest.subarray(0, 16)].map((byte) => alphabet[byte >> 4] + alphabet[byte & 0x0f]).join('');
}

describe('managed extension operations', () => {
  let directory = '';
  let artifact: {
    extension_id: string;
    artifact_ref: string;
    artifact_sha256: string;
    manifest_sha256: string;
    version: string;
    permissions: string[];
    host_permissions: string[];
    permission_probe: 'chrome.permissions.getAll.v1';
  };
  let session: SessionState;
  let send: jest.Mock;
  let newPage: jest.Mock;
  let installed: boolean;
  let manifestBytes: Buffer;

  beforeAll(async () => {
    const { publicKey } = generateKeyPairSync('rsa', { modulusLength: 2048 });
    const key = publicKey.export({ format: 'der', type: 'spki' }).toString('base64');
    const manifest = {
      manifest_version: 3,
      name: 'Test extension',
      version: '1.0.0',
      key,
      permissions: ['storage'],
      optional_permissions: ['tabs'],
      host_permissions: ['https://example.test/*'],
      optional_host_permissions: [],
    };
    manifestBytes = Buffer.from(JSON.stringify(manifest));
    const manifestHash = createHash('sha256').update(manifestBytes).digest('hex');
    artifact = {
      extension_id: extensionID(key),
      artifact_ref: 'build-1',
      artifact_sha256: 'a'.repeat(64),
      manifest_sha256: manifestHash,
      version: '1.0.0',
      permissions: ['storage', 'tabs'],
      host_permissions: ['https://example.test/*'],
      permission_probe: 'chrome.permissions.getAll.v1',
    };
    const artifactDirectory = join(stagingRoot, artifact.artifact_sha256);
    await mkdir(artifactDirectory, { recursive: true });
    directory = await mkdtemp(join(artifactDirectory, 'test-'));
    await writeFile(join(directory, 'manifest.json'), manifestBytes);
  });

  beforeEach(async () => {
    await writeFile(join(directory, 'manifest.json'), manifestBytes);
    installed = false;
    send = jest.fn(async (method: string, params?: Record<string, unknown>) => {
      if (method === 'Extensions.loadUnpacked') {
        expect(params).toEqual({ path: resolve(directory), enableInIncognito: true });
        installed = true;
        return { id: artifact.extension_id };
      }
      if (method === 'Extensions.uninstall') {
        installed = false;
        return {};
      }
      if (method === 'Extensions.getExtensions') {
        return {
          extensions: installed
            ? [{
                id: artifact.extension_id,
                name: 'Test extension',
                version: artifact.version,
                path: resolve(directory),
                enabled: true,
              }]
            : [],
        };
      }
      throw new Error(`unexpected CDP method ${method}`);
    });
    const browser = {
      newBrowserCDPSession: jest.fn(async () => ({ send, detach: jest.fn(async () => undefined) })),
    };
    const locator = {
      waitFor: jest.fn(async () => undefined),
      evaluate: jest.fn(async () => ({
        status: 'ready',
        text: JSON.stringify({
          protocol: 'vrooli.extension-permissions.v1',
          extension_id: artifact.extension_id,
          version: artifact.version,
          permissions: ['storage'],
          host_permissions: ['https://example.test/*'],
          observed_at: new Date().toISOString(),
          source: 'chrome.permissions.getAll',
        }),
      })),
    };
    const page = {
      goto: jest.fn(async (url: string) => {
        expect(url).toBe(`chrome-extension://${artifact.extension_id}/__vrooli_bas/permission-report.html`);
      }),
      locator: jest.fn(() => locator),
      close: jest.fn(async () => undefined),
    };
    newPage = jest.fn(async () => page);
    session = {
      id: 'driver-session',
      externalTarget: false,
      phase: 'ready',
      instructionInFlight: false,
      extensionOperationInFlight: false,
      spec: { permission_profile: {
        profile_id: 'profile-1',
        extension_install: true,
        extension_load: true,
        extension_access: true,
        extension_permissions: {
          [artifact.extension_id]: ['storage', 'tabs', 'https://example.test/*'],
        },
      } },
      browser,
      context: { newPage },
    } as unknown as SessionState;
  });

  afterAll(async () => {
    await rm(directory, { recursive: true, force: true });
  });

  it('installs, reloads and validates only the session-bound verified artifact', async () => {
    const installedExtension = await installManagedExtension(session, directory, artifact);
    expect(installedExtension.id).toBe(artifact.extension_id);
    expect(send).toHaveBeenCalledWith('Extensions.loadUnpacked', {
      path: resolve(directory), enableInIncognito: true,
    });

    await reloadManagedExtension(session, directory, artifact);
    expect(send).toHaveBeenCalledWith('Extensions.uninstall', { id: artifact.extension_id });

    const validation = await validateManagedExtension(session, directory, artifact);
    expect(validation.enabled).toBe(true);
  });

  it('loads idempotently and unloads only the exact staged extension', async () => {
    await installManagedExtension(session, directory, artifact);
    const before = send.mock.calls.length;
    await loadManagedExtension(session, directory, artifact);
    expect(send.mock.calls.slice(before).map(([method]) => method)).toEqual(['Extensions.getExtensions']);
    await unloadManagedExtension(session, directory, artifact);
    expect(send).toHaveBeenCalledWith('Extensions.uninstall', { id: artifact.extension_id });
  });

  it('fails closed when the session lacks extension install permission or the staged manifest changes', async () => {
    const denied = {
      ...session,
      spec: { permission_profile: { extension_install: false } },
    } as SessionState;
    await expect(installManagedExtension(denied, directory, artifact)).rejects.toThrow('extension_install');
    expect(send).not.toHaveBeenCalled();

    await writeFile(join(directory, 'manifest.json'), '{"manifest_version":3}');
    await expect(installManagedExtension(session, directory, artifact)).rejects.toThrow('manifest digest mismatch');
    expect(send).not.toHaveBeenCalled();
  });

  it('enforces approval mode independently before extension side effects', async () => {
    session.spec.permission_profile!.approval = 'every_action';
    await expect(installManagedExtension(session, directory, artifact)).rejects.toThrow('approval receipt');
    await expect(loadManagedExtension(session, directory, artifact)).rejects.toThrow('approval receipt');
    await expect(inventoryManagedExtensions(session)).rejects.toThrow('approval receipt');
    expect(send).not.toHaveBeenCalled();

    session.spec.permission_profile!.approval = 'safety_pass';
    await expect(installManagedExtension(session, directory, artifact)).rejects.toThrow('approval receipt');
    expect(send).not.toHaveBeenCalled();

    // Safety-pass allows a lower-impact lifecycle operation; the exact
    // extension ID and declared permissions remain profile scoped.
    await loadManagedExtension(session, directory, artifact);
    expect(send).toHaveBeenCalledWith('Extensions.loadUnpacked', {
      path: resolve(directory), enableInIncognito: true,
    });
  });

  it('rechecks declared permissions in the admitted profile before CDP and filters inventory to approved IDs', async () => {
    const deniedArtifact = { ...artifact, permissions: [...artifact.permissions, 'debugger'] };
    await expect(installManagedExtension(session, directory, deniedArtifact)).rejects.toThrow(/not admitted/);
    expect(send).not.toHaveBeenCalled();

    send.mockImplementation(async (method: string) => {
      if (method === 'Extensions.getExtensions') return {
        extensions: [
          { id: artifact.extension_id, name: 'Approved', version: '1.0.0', enabled: true, path: resolve(directory) },
          { id: 'abcdefghijklmnopabcdefghijklmnop', name: 'Other', version: '1', enabled: true, path: '' },
        ],
      };
      throw new Error(`unexpected CDP method ${method}`);
    });
    const entries = await inventoryManagedExtensions(session);
    expect(entries.map((entry) => entry.id)).toEqual([artifact.extension_id]);
    expect(entries[0].permissions).toBeUndefined();
    expect(entries[0].host_permissions).toBeUndefined();
  });

  it('queries the artifact-owned permission page in a temporary page on the exact session', async () => {
    installed = true;
    const entries = await inventoryManagedExtensions(session, [{ directory_path: directory, artifact }], session.id, 'profile-1');
    expect(entries).toHaveLength(1);
    expect(entries[0].permissions).toEqual(['storage']);
    expect(entries[0].host_permissions).toEqual(['https://example.test/*']);
    expect(entries[0].permission_evidence).toMatchObject({ status: 'effective', source: 'chrome.permissions.getAll.v1', scope: 'grant_set_only', per_tab_site_access: 'not_measured' });
    expect((entries[0].permission_evidence as { observed_at?: string }).observed_at).toBeTruthy();
    expect(newPage).toHaveBeenCalledTimes(1);
  });

  it('keeps a stale permission page unavailable and denies a profile mismatch', async () => {
    installed = true;
    const page = await session.context.newPage();
    const report = page.locator('#report');
    jest.mocked(report.evaluate).mockResolvedValueOnce({ status: 'ready', text: JSON.stringify({
      protocol: 'vrooli.extension-permissions.v1', extension_id: artifact.extension_id,
      version: artifact.version, permissions: ['storage'], host_permissions: ['https://example.test/*'],
      observed_at: new Date(Date.now() - 120_000).toISOString(), source: 'chrome.permissions.getAll',
    }) });
    const stale = await inventoryManagedExtensions(session, [{ directory_path: directory, artifact }], session.id, 'profile-1');
    expect(stale[0].permission_evidence).toEqual({ status: 'unavailable', source: 'chromium.Extensions.getExtensions', scope: 'unavailable', per_tab_site_access: 'not_measured' });
    await expect(inventoryManagedExtensions(session, [{ directory_path: directory, artifact }], session.id, 'other-profile'))
      .rejects.toThrow('does not match this session and admitted profile');
  });
});
