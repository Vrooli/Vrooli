import { createHash, createPublicKey } from 'node:crypto';
import { realpath, readFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { basename, isAbsolute, join, relative, resolve, sep } from 'node:path';
import type { SessionState } from '../types/session';

const stagingRoot = resolve(tmpdir(), 'vrooli-bas-managed-extensions');

export interface ManagedExtensionArtifact {
  extension_id: string;
  artifact_ref: string;
  artifact_sha256: string;
  manifest_sha256: string;
  version: string;
  permissions: string[];
  host_permissions: string[];
  permission_probe?: 'chrome.permissions.getAll.v1';
}

export interface ManagedExtensionInventoryArtifact {
  directory_path: string;
  artifact: ManagedExtensionArtifact;
}

export interface ManagedExtensionInventoryEntry {
  id: string;
  name: string;
  version: string;
  enabled: boolean;
  path: string;
  /** Present only when a driver owner has an explicit effective-grant source. */
  permissions?: string[];
  /** Present only when a driver owner has an explicit effective-grant source. */
  host_permissions?: string[];
  permission_evidence?:
    | { status: 'unavailable'; source: 'chromium.Extensions.getExtensions'; scope: 'unavailable'; per_tab_site_access: 'not_measured' }
    | { status: 'effective'; source: 'chrome.permissions.getAll.v1'; scope: 'grant_set_only'; per_tab_site_access: 'not_measured'; observed_at: string };
}

interface ExtensionManifest {
  manifest_version: number;
  name: string;
  version: string;
  key: string;
  permissions?: string[];
  optional_permissions?: string[];
  host_permissions?: string[];
  optional_host_permissions?: string[];
}

interface ExtensionCDPSession {
  // Extensions.* is experimental and not in every pinned Playwright protocol
  // schema; keep that boundary explicit while using the session owner's CDP.
  send<T = Record<string, unknown>>(method: string, params?: Record<string, unknown>): Promise<T>;
  detach(): Promise<void>;
}

interface BrowserWithCDP {
  newBrowserCDPSession(): Promise<ExtensionCDPSession>;
}

interface CDPExtensionInfo {
  id: string;
  name: string;
  version: string;
  path: string;
  enabled: boolean;
  // Chromium Extensions.getExtensions currently omits these. Preserve them if
  // a future owner API supplies live grants; never synthesize them from manifest.
  permissions?: string[];
  host_permissions?: string[];
}

function uniqueSorted(values: unknown): string[] {
  if (!Array.isArray(values) || !values.every((value) => typeof value === 'string')) {
    throw new Error('extension manifest permission list is invalid');
  }
  return [...new Set(values.map((value) => value.trim()).filter(Boolean))].sort();
}

function chromeExtensionId(encodedKey: string): string {
  const der = Buffer.from(encodedKey, 'base64');
  if (der.length === 0 || der.toString('base64') !== encodedKey) {
    throw new Error('extension manifest key is not canonical base64');
  }
  const publicKey = createPublicKey({ key: der, format: 'der', type: 'spki' });
  if (publicKey.asymmetricKeyType !== 'rsa') throw new Error('extension manifest key must be RSA');
  const digest = createHash('sha256').update(der).digest();
  const alphabet = 'abcdefghijklmnop';
  let result = '';
  for (const byte of digest.subarray(0, 16)) {
    result += alphabet.charAt(byte >> 4) + alphabet.charAt(byte & 0x0f);
  }
  return result;
}

async function verifyStagedManifest(
  directory: string,
  artifact: ManagedExtensionArtifact
): Promise<ExtensionManifest> {
  if (!isAbsolute(directory)) throw new Error('extension staging directory must be absolute');
  const resolved = await realpath(directory);
  if (resolved !== resolve(directory)) throw new Error('extension staging directory must not contain symlinked path components');
  const relativeToRoot = relative(stagingRoot, resolved);
  if (!relativeToRoot || relativeToRoot.startsWith(`..${sep}`) || relativeToRoot === '..' || isAbsolute(relativeToRoot)) {
    throw new Error('extension staging directory is outside the BAS managed extension store');
  }
  if (!basename(relativeToRoot.split(sep)[0] ?? '').match(/^[a-f0-9]{64}$/)) {
    throw new Error('extension staging directory does not use a digest-scoped owner directory');
  }
  if (relativeToRoot.split(sep)[0] !== artifact.artifact_sha256) {
    throw new Error('extension staging directory does not match the verified artifact digest');
  }
  const manifestBytes = await readFile(join(resolved, 'manifest.json'));
  const digest = createHash('sha256').update(manifestBytes).digest('hex');
  if (digest !== artifact.manifest_sha256) throw new Error('staged extension manifest digest mismatch');
  const manifest = JSON.parse(manifestBytes.toString('utf8')) as ExtensionManifest;
  if (
    manifest.manifest_version !== 3 ||
    typeof manifest.name !== 'string' ||
    typeof manifest.version !== 'string' ||
    typeof manifest.key !== 'string' ||
    manifest.version !== artifact.version ||
    chromeExtensionId(manifest.key) !== artifact.extension_id
  ) {
    throw new Error('staged extension manifest does not match the verified artifact identity');
  }
  const permissions = uniqueSorted([
    ...uniqueSorted(manifest.permissions ?? []),
    ...uniqueSorted(manifest.optional_permissions ?? []),
  ]);
  const hostPermissions = uniqueSorted([
    ...uniqueSorted(manifest.host_permissions ?? []),
    ...uniqueSorted(manifest.optional_host_permissions ?? []),
  ]);
  if (
    JSON.stringify(permissions) !== JSON.stringify([...artifact.permissions].sort()) ||
    JSON.stringify(hostPermissions) !== JSON.stringify([...artifact.host_permissions].sort())
  ) {
    throw new Error('staged extension manifest permission set differs from its verified receipt');
  }
  return manifest;
}

function requireManagedExtensionSession(
  session: SessionState,
  capability: 'extension_install' | 'extension_load' | 'extension_access' = 'extension_install',
  approvalClass: 'install' | 'lifecycle' | 'read' = 'lifecycle'
): void {
  if (session.externalTarget || session.spec.app_target) {
    throw new Error('extensions are unsupported on externally controlled browser targets');
  }
  if (session.spec.permission_profile?.[capability] !== true) {
    throw new Error(`${capability} capability was not admitted for this BAS session`);
  }
  const approval = session.spec.permission_profile?.approval ?? 'none';
  if (approval === 'every_action' || (approval === 'safety_pass' && approvalClass === 'install')) {
    throw new Error('managed extension route requires an approval receipt that this driver route does not support');
  }
  if (session.phase !== 'ready' || session.instructionInFlight) {
    throw new Error('BAS session is busy and cannot perform an extension operation');
  }
}

function requireArtifactPermissions(session: SessionState, artifact: ManagedExtensionArtifact): void {
  const allowlist = session.spec.permission_profile?.extension_permissions;
  const approved = allowlist?.[artifact.extension_id];
  if (!approved) throw new Error('extension ID is not allowlisted by this BAS session');
  for (const permission of [...artifact.permissions, ...artifact.host_permissions]) {
    if (!approved.includes(permission)) {
      throw new Error(`extension permission is not admitted by this BAS session: ${permission}`);
    }
  }
}

async function openExtensionCDP(session: SessionState): Promise<ExtensionCDPSession> {
  const browser = session.browser as unknown as BrowserWithCDP;
  if (typeof browser.newBrowserCDPSession !== 'function') {
    throw new Error('browser-level CDP is unavailable for this BAS session');
  }
  return browser.newBrowserCDPSession();
}

async function getInstalledExtensions(cdp: ExtensionCDPSession): Promise<CDPExtensionInfo[]> {
  const response = await cdp.send<{ extensions?: CDPExtensionInfo[] }>('Extensions.getExtensions');
  if (!Array.isArray(response.extensions)) throw new Error('browser extension inventory is unavailable');
  return response.extensions;
}

interface PermissionReport {
  protocol: string;
  extension_id: string;
  version: string;
  permissions: string[];
  host_permissions: string[];
  observed_at: string;
  source: string;
}

async function readArtifactPermissionReport(
  session: SessionState,
  directory: string,
  artifact: ManagedExtensionArtifact
): Promise<{ permissions: string[]; host_permissions: string[]; observed_at: string } | undefined> {
  if (artifact.permission_probe !== 'chrome.permissions.getAll.v1') return undefined;
  const manifest = await verifyStagedManifest(directory, artifact);
  const profile = session.spec.permission_profile;
  if (!profile || !profile.extension_permissions?.[artifact.extension_id]) return undefined;
  const page = await session.context.newPage();
  try {
    await page.goto(`chrome-extension://${artifact.extension_id}/__vrooli_bas/permission-report.html`, {
      waitUntil: 'domcontentloaded', timeout: 5000,
    });
    const locator = page.locator('#report');
    await locator.waitFor({ state: 'attached', timeout: 5000 });
    const element = await locator.evaluate((node) => ({
      status: (node as HTMLElement).dataset.status ?? '',
      text: node.textContent ?? '',
    }));
    if (element.status !== 'ready' || element.text.length === 0 || element.text.length > 16_384) return undefined;
    const report = JSON.parse(element.text) as Partial<PermissionReport>;
    if (
      report.protocol !== 'vrooli.extension-permissions.v1' ||
      report.extension_id !== artifact.extension_id ||
      report.version !== manifest.version ||
      report.source !== 'chrome.permissions.getAll' ||
      typeof report.observed_at !== 'string'
    ) return undefined;
    const observedAtMs = Date.parse(report.observed_at);
    const ageMs = Date.now() - observedAtMs;
    if (!Number.isFinite(observedAtMs) || ageMs < -5000 || ageMs > 60_000) return undefined;
    const permissions = uniqueSorted(report.permissions);
    const hostPermissions = uniqueSorted(report.host_permissions);
    const declaredPermissions = new Set(artifact.permissions);
    const declaredHosts = new Set(artifact.host_permissions);
    if (
      permissions.some((permission) => !declaredPermissions.has(permission)) ||
      hostPermissions.some((permission) => !declaredHosts.has(permission)) ||
      permissions.some((permission) => !profile.extension_permissions?.[artifact.extension_id]?.includes(permission)) ||
      hostPermissions.some((permission) => !profile.extension_permissions?.[artifact.extension_id]?.includes(permission))
    ) return undefined;
    return { permissions, host_permissions: hostPermissions, observed_at: new Date(observedAtMs).toISOString() };
  } catch {
    return undefined;
  } finally {
    await page.close().catch(() => undefined);
  }
}

export async function installManagedExtension(
  session: SessionState,
  directory: string,
  artifact: ManagedExtensionArtifact
): Promise<ManagedExtensionInventoryEntry> {
  requireManagedExtensionSession(session, 'extension_install', 'install');
  requireArtifactPermissions(session, artifact);
  const manifest = await verifyStagedManifest(directory, artifact);
  const cdp = await openExtensionCDP(session);
  try {
    const result = await cdp.send<{ id?: string }>('Extensions.loadUnpacked', {
      path: resolve(directory),
      enableInIncognito: true,
    });
    if (result.id !== artifact.extension_id) throw new Error('browser installed an unexpected extension id');
    const installed = (await getInstalledExtensions(cdp)).find((extension) => extension.id === result.id);
    if (!installed?.enabled || installed.version !== artifact.version) {
      throw new Error('browser did not report the verified extension as enabled');
    }
    return { ...installed, path: resolve(directory), name: manifest.name };
  } finally {
    await cdp.detach().catch(() => undefined);
  }
}

export async function loadManagedExtension(
  session: SessionState,
  directory: string,
  artifact: ManagedExtensionArtifact
): Promise<ManagedExtensionInventoryEntry> {
  requireManagedExtensionSession(session, 'extension_load');
  requireArtifactPermissions(session, artifact);
  const manifest = await verifyStagedManifest(directory, artifact);
  const cdp = await openExtensionCDP(session);
  try {
    const existing = (await getInstalledExtensions(cdp)).find((extension) => extension.id === artifact.extension_id);
    if (existing) {
      if (!existing.enabled || existing.version !== artifact.version || resolve(existing.path) !== resolve(directory)) {
        throw new Error('installed extension does not match the requested verified artifact');
      }
      return { ...existing, path: resolve(directory), name: manifest.name };
    }
    const result = await cdp.send<{ id?: string }>('Extensions.loadUnpacked', {
      path: resolve(directory),
      enableInIncognito: true,
    });
    if (result.id !== artifact.extension_id) throw new Error('browser loaded an unexpected extension id');
    const installed = (await getInstalledExtensions(cdp)).find((extension) => extension.id === result.id);
    if (!installed?.enabled || installed.version !== artifact.version) {
      throw new Error('browser did not report the verified extension as enabled');
    }
    return { ...installed, path: resolve(directory), name: manifest.name };
  } finally {
    await cdp.detach().catch(() => undefined);
  }
}

export async function reloadManagedExtension(
  session: SessionState,
  directory: string,
  artifact: ManagedExtensionArtifact
): Promise<ManagedExtensionInventoryEntry> {
  requireManagedExtensionSession(session, 'extension_load');
  requireArtifactPermissions(session, artifact);
  const manifest = await verifyStagedManifest(directory, artifact);
  const cdp = await openExtensionCDP(session);
  try {
    const before = (await getInstalledExtensions(cdp)).find((extension) => extension.id === artifact.extension_id);
    if (!before || !before.enabled || resolve(before.path) !== resolve(directory)) {
      throw new Error('verified extension is not installed from the expected artifact directory');
    }
    await cdp.send('Extensions.uninstall', { id: artifact.extension_id });
    const installed = await cdp.send<{ id?: string }>('Extensions.loadUnpacked', {
      path: resolve(directory),
      enableInIncognito: true,
    });
    if (installed.id !== artifact.extension_id) throw new Error('reloaded extension id does not match the artifact');
    const after = (await getInstalledExtensions(cdp)).find((extension) => extension.id === installed.id);
    if (!after?.enabled || after.version !== artifact.version) {
      throw new Error('browser did not report the verified extension as enabled after reload');
    }
    return { ...after, path: resolve(directory), name: manifest.name };
  } finally {
    await cdp.detach().catch(() => undefined);
  }
}

export async function unloadManagedExtension(
  session: SessionState,
  directory: string,
  artifact: ManagedExtensionArtifact
): Promise<void> {
  requireManagedExtensionSession(session, 'extension_load');
  requireArtifactPermissions(session, artifact);
  await verifyStagedManifest(directory, artifact);
  const cdp = await openExtensionCDP(session);
  try {
    const installed = (await getInstalledExtensions(cdp)).find((extension) => extension.id === artifact.extension_id);
    if (!installed || resolve(installed.path) !== resolve(directory)) {
      throw new Error('refusing to unload an extension from a different artifact directory');
    }
    await cdp.send('Extensions.uninstall', { id: artifact.extension_id });
    if ((await getInstalledExtensions(cdp)).some((extension) => extension.id === artifact.extension_id)) {
      throw new Error('browser still reports the extension after uninstall');
    }
  } finally {
    await cdp.detach().catch(() => undefined);
  }
}

export async function validateManagedExtension(
  session: SessionState,
  directory: string,
  artifact: ManagedExtensionArtifact
): Promise<ManagedExtensionInventoryEntry> {
  requireManagedExtensionSession(session, 'extension_load');
  requireArtifactPermissions(session, artifact);
  const manifest = await verifyStagedManifest(directory, artifact);
  const cdp = await openExtensionCDP(session);
  try {
    const installed = (await getInstalledExtensions(cdp)).find((extension) => extension.id === artifact.extension_id);
    if (
      !installed?.enabled ||
      installed.version !== artifact.version ||
      resolve(installed.path) !== resolve(directory)
    ) {
      throw new Error('browser extension inventory does not match the verified installed artifact');
    }
    return { ...installed, path: resolve(directory), name: manifest.name };
  } finally {
    await cdp.detach().catch(() => undefined);
  }
}

export async function inventoryManagedExtensions(
  session: SessionState,
  inventoryArtifacts: ManagedExtensionInventoryArtifact[] = [],
  requestSessionID = session.id,
  requestProfileID = session.spec.permission_profile?.profile_id ?? ''
): Promise<ManagedExtensionInventoryEntry[]> {
  requireManagedExtensionSession(session, 'extension_access', 'read');
  const allowlist = session.spec.permission_profile?.extension_permissions;
  if (!allowlist || Object.keys(allowlist).length === 0) {
    throw new Error('extension inventory requires an explicit extension ID allowlist');
  }
  if (requestSessionID !== session.id || !requestProfileID || requestProfileID !== session.spec.permission_profile?.profile_id) {
    throw new Error('extension inventory request does not match this session and admitted profile');
  }
  const artifactsByID = new Map<string, ManagedExtensionInventoryArtifact>();
  for (const candidate of inventoryArtifacts) {
    if (!candidate || !candidate.artifact || artifactsByID.has(candidate.artifact.extension_id)) {
      throw new Error('extension inventory artifact set is invalid or ambiguous');
    }
    requireArtifactPermissions(session, candidate.artifact);
    await verifyStagedManifest(candidate.directory_path, candidate.artifact);
    artifactsByID.set(candidate.artifact.extension_id, candidate);
  }
  const cdp = await openExtensionCDP(session);
  try {
    const entries: ManagedExtensionInventoryEntry[] = [];
    for (const extension of await getInstalledExtensions(cdp)) {
      if (!allowlist[extension.id]) continue;
      const candidate = artifactsByID.get(extension.id);
      const exactArtifact = Boolean(candidate && extension.enabled && extension.version === candidate.artifact.version && resolve(extension.path) === resolve(candidate.directory_path));
      const report = exactArtifact && candidate
        ? await readArtifactPermissionReport(session, candidate.directory_path, candidate.artifact)
        : undefined;
      entries.push({
        ...extension,
        // Effective permissions come only from the artifact-owned same-session
        // extension page. Extensions.getExtensions alone never supplies grants.
        permissions: report?.permissions,
        host_permissions: report?.host_permissions,
        permission_evidence: report
          ? { status: 'effective', source: 'chrome.permissions.getAll.v1', scope: 'grant_set_only', per_tab_site_access: 'not_measured', observed_at: report.observed_at }
          : { status: 'unavailable', source: 'chromium.Extensions.getExtensions', scope: 'unavailable', per_tab_site_access: 'not_measured' },
        path: extension.path.startsWith(stagingRoot) ? extension.path : '',
      });
    }
    return entries;
  } finally {
    await cdp.detach().catch(() => undefined);
  }
}
