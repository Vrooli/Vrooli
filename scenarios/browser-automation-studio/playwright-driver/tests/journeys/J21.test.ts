import { mkdtemp, rm } from 'node:fs/promises';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { spawn, type ChildProcess } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { chromium } from 'rebrowser-playwright';
import type { AppTargetSpec, StartSessionRequest, StartSessionResponse, ValidationContextSpec } from '../../src/types';
import { action, closeLease, driver, env, fixtureAfter, fixtureState, releaseLease, runAction, type Lease, withJourneyResources } from './support';

type Renderer = { id: string; type: string; url: string; title: string };

async function allocatePort(): Promise<number> {
  const server = createServer();
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const address = server.address();
  await new Promise<void>((resolve) => server.close(() => resolve()));
  if (!address || typeof address === 'string') throw new Error('Could not allocate a CDP port');
  return address.port;
}

async function waitForRenderer(endpoint: string): Promise<Renderer> {
  const deadline = Date.now() + 15000;
  let lastError: unknown;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(`${endpoint}/json/list`, { signal: AbortSignal.timeout(1000) });
      if (!response.ok) throw new Error(`CDP returned HTTP ${response.status}`);
      const rows: unknown = await response.json();
      if (!Array.isArray(rows)) throw new Error('CDP renderer list was not an array');
      const page = rows.find((row): row is Renderer =>
        Boolean(row && typeof row === 'object' && (row as Renderer).type === 'page' &&
          typeof (row as Renderer).id === 'string' && typeof (row as Renderer).url === 'string' &&
          (row as Renderer).url.startsWith(env.fixture))
      );
      if (page) return page;
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolve) => setTimeout(resolve, 100));
  }
  throw new Error(`Fixture Chromium did not expose a renderer: ${String(lastError ?? 'timeout')}`);
}

async function stopProcess(child: ChildProcess): Promise<void> {
  const pid = child.pid;
  if (!pid || child.exitCode !== null) return;
  try {
    process.kill(-pid, 'SIGTERM');
  } catch (error) {
    if ((error as NodeJS.ErrnoException).code !== 'ESRCH') throw error;
  }
  await new Promise<void>((resolve) => {
    const timer = setTimeout(() => {
      try { process.kill(-pid, 'SIGKILL'); } catch { /* Process exited during shutdown. */ }
      resolve();
    }, 3000);
    child.once('exit', () => { clearTimeout(timer); resolve(); });
  });
}

describe('[REQ:BAS-RH-J21] released managed target admission', () => {
  it('reattaches the exact released target and refuses missing or mismatched validation context', () => withJourneyResources(async (resources) => {
    const port = await allocatePort();
    const endpoint = `http://127.0.0.1:${port}`;
    const profileDir = await mkdtemp(path.join(tmpdir(), 'bas-j21-cdp-'));
    let child: ChildProcess | undefined;
    let first: Lease | undefined;
    let second: Lease | undefined;
    resources.add('stop J21 fixture Chromium', async () => {
      if (child) await stopProcess(child);
      await rm(profileDir, { recursive: true, force: true });
    });

    child = spawn(chromium.executablePath(), [
      '--no-sandbox', '--headless=new', '--disable-gpu', '--no-first-run',
      '--remote-debugging-address=127.0.0.1', `--remote-debugging-port=${port}`,
      `--user-data-dir=${profileDir}`, `${env.fixture}/`,
    ], { detached: true, stdio: 'ignore' });
    const renderer = await waitForRenderer(endpoint);
    const target: AppTargetSpec = {
      target_id: `j21-target-${randomUUID()}`,
      cdp_endpoint: endpoint,
      renderer_id: renderer.id,
      renderer_url: renderer.url,
      renderer_title: renderer.title,
      scenario_name: 'browser-automation-studio',
      artifact_digest: 'sha256:bas-j21-linux-fixture',
      context_id: `j21-context-${randomUUID()}`,
      cdp_transport: 'loopback-authenticated',
    };
    const validation: ValidationContextSpec = {
      context_id: target.context_id,
      scenario_name: target.scenario_name,
      artifact_digest: target.artifact_digest,
      target_id: target.target_id,
      workflow_id: 'bas-rh-j21/linux-managed-session',
      profile_id: 'normal',
      isolation_lease_id: `j21-isolation-${randomUUID()}`,
    };
    const closedLeases = new WeakSet<Lease>();
    const start = (executionId: string, appTarget = target, context: ValidationContextSpec | null = validation): StartSessionRequest => ({
      execution_id: executionId,
      workflow_id: randomUUID(),
      session_options: {
        viewport: { width: 640, height: 480 },
        reuse_mode: 'reuse',
        frame_scale: 'css',
        labels: { journey: 'J21-released-managed-target' },
        app_target: appTarget,
        validation_context: context ?? undefined,
      },
    });
    const asLease = (response: StartSessionResponse, executionId: string): Lease => ({
      sessionId: response.session_id,
      leaseId: response.lease_id,
      executionId,
      sequence: 0,
    });
    const expectRejectedStart = async (request: StartSessionRequest, message: string): Promise<void> => {
      const response = await fetch(`${env.driver}/session/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(request),
        signal: AbortSignal.timeout(10000),
      });
      const body = await response.text();
      expect(response.status).toBe(400);
      expect(body).toContain(message);
      expect(body).not.toContain('session_id');
    };

    const before = await fixtureState();
    const firstExecution = randomUUID();
    const initial = await driver<StartSessionResponse>('/session/start', start(firstExecution));
    first = asLease(initial, firstExecution);
    resources.add('close J21 initial target lease', async () => {
      if (!closedLeases.has(first as Lease)) {
        await closeLease(first as Lease);
        closedLeases.add(first as Lease);
      }
    });
    await runAction(first, action('click', { selector: '#counter' }));
    const clicked = await fixtureAfter('effect', before.effects.length);
    expect(clicked.effects).toHaveLength(before.effects.length + 1);
    await releaseLease(first);

    const secondExecution = randomUUID();
    const resumed = await driver<StartSessionResponse>('/session/start', start(secondExecution));
    second = asLease(resumed, secondExecution);
    resources.add('close J21 resumed target lease', async () => {
      if (!closedLeases.has(second as Lease)) {
        await closeLease(second as Lease);
        closedLeases.add(second as Lease);
      }
    });
    expect(resumed.reused).toBeUndefined();
    expect(resumed.session_id).not.toBe(initial.session_id);
    expect(resumed.lease_id).not.toBe(initial.lease_id);
    await runAction(second, action('click', { selector: '#counter' }));

    await expectRejectedStart(start(secondExecution, target, null), 'Electron validation context is required');
    await expectRejectedStart(start(secondExecution, target, { ...validation, isolation_lease_id: '' }), 'Electron validation context requires an isolation lease');
    await expectRejectedStart(start(secondExecution, target, { ...validation, context_id: 'wrong-context' }), 'does not match target context');
    await expectRejectedStart(start(secondExecution, target, { ...validation, target_id: 'wrong-target' }), 'does not match session identity');

    await closeLease(first);
    closedLeases.add(first);
    await closeLease(second);
    closedLeases.add(second);
  }));
});
