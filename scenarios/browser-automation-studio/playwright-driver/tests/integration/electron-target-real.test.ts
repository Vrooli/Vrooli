import { mkdtemp, rm } from 'node:fs/promises';
import { createConnection, createServer } from 'node:net';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { spawn, type ChildProcess } from 'node:child_process';
import { playwrightProvider } from '../../src/playwright';
import { selectElectronPage, verifyElectronRenderer } from '../../src/session/electron-target';
import type { ElectronTargetSpec } from '../../src/types';

type DiscoveredRenderer = {
  id: string;
  type: 'page';
  url: string;
  title: string;
  webSocketDebuggerUrl: string;
};

const HELLO_DESKTOP_RENDERER_PREFIX = 'http://127.0.0.1';
const integrationArtifact = process.env.VROOLI_ELECTRON_INTEGRATION_APP;
const runRealElectronIntegration = Boolean(integrationArtifact && process.env.DISPLAY);

async function allocateLoopbackPort(): Promise<number> {
  const server = createServer();
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', () => resolve());
  });
  const address = server.address();
  await new Promise<void>((resolve) => server.close(() => resolve()));
  if (!address || typeof address === 'string') throw new Error('failed to allocate a loopback port');
  return address.port;
}

async function waitForRenderer(endpoint: string, timeoutMs = 20000): Promise<DiscoveredRenderer> {
  const deadline = Date.now() + timeoutMs;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${endpoint}/json/list`, { signal: AbortSignal.timeout(1000) });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const payload: unknown = await response.json();
      if (!Array.isArray(payload)) throw new Error('non-list CDP payload');
      const pages = payload.filter((entry): entry is DiscoveredRenderer => {
        if (!entry || typeof entry !== 'object') return false;
        const candidate = entry as Partial<DiscoveredRenderer>;
        return (
          candidate.type === 'page' &&
          candidate.url.startsWith(HELLO_DESKTOP_RENDERER_PREFIX) &&
          typeof candidate.id === 'string' &&
          typeof candidate.url === 'string' &&
          typeof candidate.title === 'string' &&
          typeof candidate.webSocketDebuggerUrl === 'string'
        );
      });
      if (pages.length === 1) return pages[0] as DiscoveredRenderer;
      if (pages.length > 1) throw new Error(`expected one renderer, found ${pages.length}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`Electron renderer did not become attachable: ${String(lastError ?? 'timeout')}`);
}

async function terminateProcess(child: ChildProcess): Promise<void> {
  const pid = child.pid;
  if (!pid || child.exitCode !== null) return;
  try {
    process.kill(-pid, 'SIGTERM');
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error;
  }
  await new Promise<void>((resolve) => {
    const timer = setTimeout(() => {
      try {
        process.kill(-pid, 'SIGKILL');
      } catch {
        // The process may have exited between the two signals.
      }
      resolve();
    }, 3000);
    child.once('exit', () => {
      clearTimeout(timer);
      resolve();
    });
  });
}

async function waitForPortRelease(port: number, timeoutMs = 3000): Promise<boolean> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const released = await new Promise<boolean>((resolve) => {
      const socket = createConnection({ host: '127.0.0.1', port });
      socket.once('connect', () => {
        socket.destroy();
        resolve(false);
      });
      socket.once('error', () => resolve(true));
    });
    if (released) return true;
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  return false;
}

describe('Electron target attachment (real AppImage)', () => {
  const describeIfEnabled = runRealElectronIntegration ? describe : describe.skip;

  describeIfEnabled('through the BAS CDP seam', () => {
    let browser: Awaited<ReturnType<typeof playwrightProvider.chromium.connectOverCDP>> | undefined;
    let child: ChildProcess | undefined;
    let profileDir: string | undefined;

    afterEach(async () => {
      if (child) await terminateProcess(child);
      if (browser) await browser.close().catch(() => undefined);
      if (profileDir) await rm(profileDir, { recursive: true, force: true });
    }, 15000);

    it('attaches to the controlled renderer, asserts DOM, and detaches cleanly', async () => {
      const port = await allocateLoopbackPort();
      const endpoint = `http://127.0.0.1:${port}`;
      profileDir = await mkdtemp(path.join(tmpdir(), 'vrooli-electron-bas-'));
      child = spawn(
        integrationArtifact as string,
        [
          '--no-sandbox',
          `--remote-debugging-address=127.0.0.1`,
          `--remote-debugging-port=${port}`,
          `--user-data-dir=${profileDir}`,
        ],
        {
          detached: true,
          stdio: 'ignore',
          env: { ...process.env, APPIMAGE_EXTRACT_AND_RUN: '1' },
        }
      );

      const renderer = await waitForRenderer(endpoint);
      const target: ElectronTargetSpec = {
        target_id: 'real-electron-target',
        cdp_endpoint: endpoint,
        renderer_id: renderer.id,
        renderer_url: renderer.url,
        renderer_title: renderer.title,
        scenario_name: 'hello-desktop',
        artifact_digest: 'sha256:hello-desktop-integration',
        context_id: 'real-electron-context',
        cdp_transport: 'loopback-authenticated',
      };

      await verifyElectronRenderer(target);
      browser = await playwrightProvider.chromium.connectOverCDP(endpoint);
      const contexts = browser.contexts();
      expect(contexts).toHaveLength(1);
      const pages = contexts[0]?.pages() ?? [];
      const page = await selectElectronPage(pages, target);
      expect(await page.title()).toEqual(expect.any(String));
      expect(await page.locator('h1').textContent()).toBe('Hello Desktop');

      await terminateProcess(child as ChildProcess);
      child = undefined;
      expect(await waitForPortRelease(port)).toBe(true);
    }, 60000);
  });
});
