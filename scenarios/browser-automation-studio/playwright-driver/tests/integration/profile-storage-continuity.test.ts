import * as http from 'node:http';
import type { AddressInfo } from 'node:net';
import { randomUUID } from 'node:crypto';
import type { Page, Frame } from 'rebrowser-playwright';
import { mkdtemp, rm, stat, writeFile } from 'node:fs/promises';
import { pathToFileURL } from 'node:url';
import { ActionType } from '../../src/proto/recording';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { SessionManager } from '../../src/session/manager';
import { handleSessionReset } from '../../src/routes/session-reset';
import type { SessionSpec } from '../../src/types';
import { createTestConfig } from '../helpers';

type IdentityState = {
  cookie: string;
  localStorage: string | null;
  indexedDB: string | null;
};

type ResetReply = {
  error?: { code?: string };
  success?: boolean;
  phase?: string;
};

const required = <T>(value: T | undefined | null, label: string): T => {
  if (value === undefined || value === null) throw new Error(`${label} was not available`);
  return value;
};

// [REQ:BAS-RH-J01] Observe authentication independently of the captured snapshot.
describe('profile authentication continuity', () => {
  let server: http.Server;
  let manager: SessionManager;
  let origin: string;

  beforeAll(async () => {
    server = http.createServer((request, response) => {
      const sessionId = request.url?.match(/^\/session\/(.+)\/reset$/)?.[1];
      if (sessionId) {
        void handleSessionReset(request, response, sessionId, manager);
        return;
      }
      if (request.url === '/worker.js') {
        response.writeHead(200, { 'Content-Type': 'text/javascript' });
        response.end("self.addEventListener('install', () => self.skipWaiting()); self.addEventListener('activate', event => event.waitUntil(self.clients.claim()));");
        return;
      }
      if (request.url === '/frame-host') {
        response.writeHead(200, { 'Content-Type': 'text/html' });
        response.end(`<title>Cross-site frame</title><iframe src="${origin.replace('127.0.0.1', 'localhost')}/third"></iframe>`);
        return;
      }
      response.writeHead(200, { 'Content-Type': 'text/html' });
      response.end('<!doctype html><title>Local profile identity fixture</title><button id="capture" onclick="this.dataset.clicked=String(Number(this.dataset.clicked || 0)+1)">Capture</button>');
    });
    await new Promise<void>((resolve) => server.listen(0, '127.0.0.1', resolve));
    origin = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
    manager = new SessionManager(createTestConfig());
  });

  afterAll(async () => {
    await manager?.shutdown();
    server?.closeAllConnections();
    await new Promise<void>((resolve, reject) => server?.close((error) => error ? reject(error) : resolve()));
  });

  async function start(
    storageState?: SessionSpec['storage_state']
  ): Promise<Awaited<ReturnType<SessionManager['startSession']>> & { page: Page }> {
    const session = await manager.startSession({
      execution_id: randomUUID(),
      workflow_id: 'synthetic-profile-continuity',
      base_url: origin,
      viewport: { width: 640, height: 480 },
      reuse_mode: 'fresh',
      required_capabilities: {},
      storage_state: storageState,
    });
    const page = manager.getSession(session.sessionId).page;
    await page.goto(origin);
    return { ...session, page };
  }

  async function writeIdentity(page: Page | Frame, identity: string): Promise<void> {
    await page.evaluate(async (value) => {
      document.cookie = `fixture_identity=${value}; Path=/; SameSite=Lax`;
      localStorage.setItem('fixture_identity', value);
      const database = await new Promise<IDBDatabase>((resolve, reject) => {
        const request = indexedDB.open('fixture-auth', 1);
        request.onupgradeneeded = (): void => { request.result.createObjectStore('tokens'); };
        request.onsuccess = (): void => resolve(request.result);
        request.onerror = (): void => reject(request.error);
      });
      await new Promise<void>((resolve, reject) => {
        const transaction = database.transaction('tokens', 'readwrite');
        transaction.objectStore('tokens').put({ identity: value }, 'current');
        transaction.oncomplete = (): void => resolve();
        transaction.onerror = (): void => reject(transaction.error);
      });
      database.close();
    }, identity);
  }

  async function readIdentity(page: Page | Frame): Promise<IdentityState> {
    return page.evaluate(async (): Promise<IdentityState> => {
      const database = await new Promise<IDBDatabase>((resolve, reject) => {
        const request = indexedDB.open('fixture-auth', 1);
        request.onupgradeneeded = (): void => { request.result.createObjectStore('tokens'); };
        request.onsuccess = (): void => resolve(request.result);
        request.onerror = (): void => reject(request.error);
      });
      const identity = await new Promise<string | null>((resolve, reject) => {
        const request = database.transaction('tokens').objectStore('tokens').get('current');
        request.onsuccess = (): void => {
          const result = request.result as { identity?: unknown } | undefined;
          resolve(typeof result?.identity === 'string' ? result.identity : null);
        };
        request.onerror = (): void => reject(request.error);
      });
      database.close();
      return { cookie: document.cookie, localStorage: localStorage.getItem('fixture_identity'), indexedDB: identity };
    });
  }

  it('restores authentication after close while a second profile stays distinct', async () => {
    const first = await start();
    await writeIdentity(first.page, 'alpha');
    const second = await start();
    await writeIdentity(second.page, 'beta');
    const snapshot = await manager.getStorageState(first.sessionId);
    await manager.closeSession(first.sessionId);
    const restored = await start(snapshot);

    expect(await readIdentity(restored.page)).toEqual({
      cookie: 'fixture_identity=alpha', localStorage: 'alpha', indexedDB: 'alpha',
    });
    expect(await readIdentity(second.page)).toEqual({
      cookie: 'fixture_identity=beta', localStorage: 'beta', indexedDB: 'beta',
    });
  }, 30000);

  it.each(['stale', 'missing', 'released'] as const)(
    'preserves browser identity when reset credentials are %s',
    async (credentialCase) => {
      const firstOwner = randomUUID();
      const labels = { resetFixture: randomUUID() };
      const spec: SessionSpec = {
        execution_id: firstOwner, workflow_id: 'synthetic-reset-ownership',
        base_url: origin, viewport: { width: 640, height: 480 },
        reuse_mode: 'fresh', required_capabilities: {}, labels,
      };
      const first = await manager.startSession(spec);
      try {
        expect(manager.releaseExecutionLease(first.sessionId, firstOwner, first.leaseId)).toBe(true);
        const nextOwner = randomUUID();
        const next = await manager.startSession({ ...spec, execution_id: nextOwner, reuse_mode: 'reuse' });
        expect(next.sessionId).toBe(first.sessionId);
        expect(next.leaseId).not.toBe(first.leaseId);
        const session = manager.peekSession(next.sessionId);
        await session.page.goto(origin);
        await writeIdentity(session.page, 'new-owner');
        if (credentialCase === 'released') {
          expect(manager.releaseExecutionLease(next.sessionId, nextOwner, next.leaseId)).toBe(true);
        }
        const lastUsedAt = session.lastUsedAt;
        const credentials = credentialCase === 'missing' ? {} : credentialCase === 'stale'
          ? { execution_id: firstOwner, lease_id: first.leaseId }
          : { execution_id: nextOwner, lease_id: next.leaseId };
        const response = await fetch(`${origin}/session/${next.sessionId}/reset`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(credentials),
        });
        const reply = (await response.json()) as ResetReply;
        expect({ status: response.status, code: reply.error?.code }).toEqual({
          status: credentialCase === 'missing' ? 400 : 404,
          code: credentialCase === 'missing' ? 'INVALID_INSTRUCTION' : 'SESSION_NOT_FOUND',
        });
        expect(session.page.url()).toBe(`${origin}/`);
        expect(session.phase).toBe('ready');
        expect(session.lastUsedAt).toBe(lastUsedAt);
        expect(await readIdentity(session.page)).toEqual({
          cookie: 'fixture_identity=new-owner', localStorage: 'new-owner', indexedDB: 'new-owner',
        });
      } finally {
        await manager.closeSession(first.sessionId);
      }
    }, 30000,
  );


  it('resets every owned origin without application requests or changing another context', async () => {
    const owner = randomUUID();
    const importedOrigin = 'http://imported.invalid';
    const remoteOrigin = origin.replace('127.0.0.1', 'localhost');
    const cacheOnlyOrigin = origin.replace('127.0.0.1', '127.0.0.2');
    const first = await manager.startSession({
      execution_id: owner, workflow_id: 'synthetic-clean-reset', base_url: origin,
      viewport: { width: 640, height: 480 }, reuse_mode: 'fresh', required_capabilities: {},
      storage_state: { cookies: [], origins: [{ origin: importedOrigin, localStorage: [{ name: 'imported', value: 'must-clear' }] }] },
    });
    const unrelated = await start();
    const session = manager.peekSession(first.sessionId);
    const primary = session.page;
    try {
      await writeIdentity(unrelated.page, 'unrelated');
      for (const url of [origin, remoteOrigin]) {
        await primary.goto(url);
        await writeIdentity(primary, 'clear-me');
        await primary.evaluate(async () => {
          sessionStorage.setItem('reset-session', 'must-clear');
          await (await caches.open('reset-cache')).put('/cached', new Response('must-clear'));
          await navigator.serviceWorker.register('/worker.js');
          await navigator.serviceWorker.ready;
        });
      }
      await primary.goto(`${origin}/frame-host`, { timeout: 5000 });
      const partitionedFrame = required(primary.frames()[1], 'partitioned frame');
      await partitionedFrame.waitForLoadState();
      await writeIdentity(partitionedFrame, 'partitioned-frame');
      await partitionedFrame.evaluate(() => sessionStorage.setItem('frame-session', 'must-clear'));
      const extra = await session.context.newPage();
      await extra.goto(origin);
      session.page = extra;
      await extra.route(`${cacheOnlyOrigin}/**`, route => route.fulfill({ contentType: 'text/html', body: '<title>Cache-only origin</title>' }));
      await extra.goto(cacheOnlyOrigin);
      await extra.evaluate(async () => { await (await caches.open('cache-only')).put('/value', new Response('must-clear')); });
      const closedOrigin = 'http://closed.invalid';
      await extra.route(`${closedOrigin}/**`, route => route.fulfill({ contentType: 'text/html', body: '<title>Closed origin</title>' }));
      await extra.goto(closedOrigin);
      await writeIdentity(extra, 'closed-page');
      await extra.close();
      const active = await session.context.newPage();
      await active.goto(origin);
      const activeId = randomUUID();
      session.pageBindings.register(active, activeId);
      session.page = active;
      const sequence = 9;
      session.lastInstructionSequence = sequence;
      session.instructionReceipts?.set(sequence, { fingerprint: 'old', response: '{}' });
      let applicationRequests = 0;
      const observe = (request: http.IncomingMessage): void => {
        if (request.url === '/' || request.url === '/worker.js') applicationRequests++;
      };
      server.on('request', observe);
      let response: Response;
      try {
        response = await fetch(`${origin}/session/${first.sessionId}/reset`, {
          method: 'POST', headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ execution_id: owner, lease_id: first.leaseId }),
        });
        const reply = (await response.json()) as ResetReply;
        expect({ status: response.status, reply }).toEqual({ status: 200, reply: { success: true, phase: 'ready' } });
      } finally {
        server.off('request', observe);
      }
      expect(applicationRequests).toBe(0);
      expect(session.context.pages()).toEqual([active]);
      expect(session.page).toBe(active);
      expect(primary.isClosed()).toBe(true);
      expect(active.isClosed()).toBe(false);
      expect(session.pageBindings.ids()).toHaveLength(1);
      expect(session.pageBindings.getPage(activeId)).toBe(active);
      expect(session.pageBindings.getId(active)).toBe(activeId);
      expect(active.url()).toBe('about:blank');
      expect(session.phase).toBe('ready');
      expect(session.lastInstructionSequence).toBe(sequence);
      expect(session.instructionReceipts?.size).toBe(0);
      for (const url of [origin, remoteOrigin, importedOrigin, closedOrigin, cacheOnlyOrigin]) {
        await active.route(`${url}/**`, route => route.fulfill({ contentType: 'text/html', body: '<title>Storage oracle</title>' }));
        await active.goto(url);
        expect(await active.evaluate(async () => ({
          cookie: document.cookie, localStorage: Object.keys(localStorage), sessionStorage: Object.keys(sessionStorage),
          indexedDB: (await indexedDB.databases()).map(db => db.name), caches: globalThis.caches ? await caches.keys() : [],
          serviceWorkers: navigator.serviceWorker ? (await navigator.serviceWorker.getRegistrations()).length : 0,
        }))).toEqual({ cookie: '', localStorage: [], sessionStorage: [], indexedDB: [], caches: [], serviceWorkers: 0 });
      }
      await active.route(`${origin}/frame-oracle`, route => route.fulfill({
        contentType: 'text/html', body: `<iframe src="${remoteOrigin}/third"></iframe>`,
      }));
      await active.goto(`${origin}/frame-oracle`, { timeout: 5000 });
      const oracleFrame = required(active.frames()[1], 'oracle frame');
      await oracleFrame.waitForLoadState();
      expect(await oracleFrame.evaluate(async () => ({
        local: Object.keys(localStorage), session: Object.keys(sessionStorage), databases: await indexedDB.databases(),
      }))).toEqual({ local: [], session: [], databases: [] });
      expect(await readIdentity(unrelated.page)).toEqual({
        cookie: 'fixture_identity=unrelated', localStorage: 'unrelated', indexedDB: 'unrelated',
      });
    } finally {
      await manager.closeSession(first.sessionId);
      await manager.closeSession(unrelated.sessionId);
    }
  }, 30000);


  it('preserves primary-page capture and can record again after reset', async () => {
    const root = await mkdtemp(join(tmpdir(), 'bas-reset-capture-'));
    const owned = new SessionManager(createTestConfig({ telemetry: { har: { enabled: true }, tracing: { enabled: true } } }));
    try {
      const { sessionId } = await owned.startSession({
        execution_id: randomUUID(), workflow_id: 'synthetic-reset-capture', base_url: origin,
        viewport: { width: 640, height: 480 }, reuse_mode: 'fresh',
        required_capabilities: { video: true, tracing: true, har: true }, artifact_paths: { root },
      });
      const session = owned.peekSession(sessionId);
      const context = session.context;
      const page = session.page;
      await page.goto(origin);
      await owned.resetSession(sessionId);
      expect(session.context).toBe(context);
      expect(session.page).toBe(page);
      await page.goto(origin);
      const captured: Array<ActionType | undefined> = [];
      const pipelineManager = required(session.pipelineManager, 'profile pipeline manager');
      await pipelineManager.startRecording({
        sessionId, recordingId: randomUUID(), onEntry: entry => { captured.push(entry.action?.type); },
      });
      await page.click('#capture');
      await pipelineManager.stopRecording();
      expect(await page.getAttribute('#capture', 'data-clicked')).toBe('1');
      expect(captured).toContain(ActionType.CLICK);
      expect(page.video()).not.toBeNull();
      const result = await owned.closeSession(sessionId);
      expect(result.videoPaths).toHaveLength(1);
      expect(result.tracePath).toBeTruthy();
      expect(result.harPath).toBeTruthy();
      const tracePath = required(result.tracePath, 'trace path');
      const harPath = required(result.harPath, 'HAR path');
      for (const file of [...result.videoPaths, tracePath, harPath]) {
        expect((await stat(file)).size).toBeGreaterThan(0);
      }
    } finally {
      await owned.shutdown();
      await rm(root, { recursive: true, force: true });
    }
  }, 30000);


  it('clears file-origin storage without affecting another context on the same files', async () => {
    const root = await mkdtemp(join(tmpdir(), 'bas-reset-file-'));
    const urls = [pathToFileURL(join(root, 'first.html')).href, pathToFileURL(join(root, 'second.html')).href];
    await Promise.all(['first.html', 'second.html'].map(name => writeFile(join(root, name), '<title>Owned file fixture</title>')));
    const first = await start();
    const control = await start();
    try {
      for (const url of urls) {
        await first.page.goto(url);
        await first.page.evaluate(() => { localStorage.setItem('file', 'clear'); sessionStorage.setItem('file', 'clear'); });
      }
      await control.page.goto(required(urls[0], 'first file URL'));
      await control.page.evaluate(() => { localStorage.setItem('file', 'preserve'); sessionStorage.setItem('file', 'preserve'); });
      await manager.resetSession(first.sessionId);
      for (const url of urls) {
        await first.page.goto(url);
        expect(await first.page.evaluate(() => [localStorage.getItem('file'), sessionStorage.getItem('file')])).toEqual([null, null]);
      }
      expect(await control.page.evaluate(() => [localStorage.getItem('file'), sessionStorage.getItem('file')])).toEqual(['preserve', 'preserve']);
    } finally {
      await manager.closeSession(first.sessionId);
      await manager.closeSession(control.sessionId);
      await rm(root, { recursive: true, force: true });
    }
  }, 30000);

});
