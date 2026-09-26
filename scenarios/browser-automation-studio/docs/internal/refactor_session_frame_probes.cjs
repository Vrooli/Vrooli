/* Investigation only: actual TypeScript owners with synthetic dependency seams.
 * No browser, service, socket, profile or production file is opened/mutated.
 * Run with node. Exit 0 means probes completed, not that the product passed.
 * A small explicit hook scheduler models render/effect cleanup, not React/OS timing.
 */
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const { createRequire } = require('node:module');
const root = path.resolve(__dirname, '../..');
const req = createRequire(path.join(root, 'playwright-driver/package.json'));
const ts = req('typescript');
const sources = {};
const results = [];
const record = (id, expected, actual, met) => results.push({ id, expected, actual, expected_behavior_met: met });
const drain = async () => { for (let i = 0; i < 30; i++) await Promise.resolve(); };
function evaluate(relative, mocks, globals = {}) {
  const filename = path.join(root, relative), source = fs.readFileSync(filename, 'utf8');
  sources[relative] = crypto.createHash('sha256').update(source).digest('hex');
  const code = ts.transpileModule(source, { fileName: filename, compilerOptions: {
    module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true,
  } }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(code, { module, exports: module.exports,
    require(name) { if (!Object.hasOwn(mocks, name)) throw new Error(`Unexpected dependency ${relative}: ${name}`); return mocks[name]; },
    console, crypto, process: { env: {} }, ...globals,
  }, { filename });
  return module.exports;
}
const driver = (p, mocks = {}) => evaluate(`playwright-driver/src/${p}.ts`, mocks);
const silent = new Proxy({}, { get: () => () => {} });
const utils = { logger: silent, metrics: new Proxy({}, { get: () => silent }), scopedLog: (_, text) => text,
  LogContext: {}, ResourceLimitError: Error, SessionNotFoundError: Error };
const decisions = driver('session/session-decisions');
const machine = driver('session/state-machine', { '../utils': utils });
const inspection = driver('session/session-inspection', { './session-decisions': decisions });
const guard = driver('infra/in-flight-guard', { '../utils': utils });
const registry = driver('infra/session-cleanup-registry', { '../utils': utils });
const reset = driver('session/session-reset', { '../infra': registry, '../utils': utils });
const teardown = driver('session/session-teardown', {
  'node:fs': { constants: fs.constants },
  'node:fs/promises': { mkdir() { throw new Error('Unexpected filesystem call'); }, rename() { throw new Error('Unexpected filesystem call'); } },
  'node:path': path, '../recording': { removeRecordingBuffer() {} }, '../utils': utils,
});
const page = id => ({ id, closed: false, isClosed() { return this.closed; }, on() {}, viewportSize: () => ({ width: 1280, height: 720 }),
  async goto() {}, async evaluate() {}, async unroute() {}, async close() { this.closed = true; } });
const session = (id = 'synthetic-session') => {
  const initial = page('first');
  return { id, spec: { execution_id: 'owner', labels: { pool: 'fixture' } }, ownerExecutionId: 'owner', leaseId: 'lease',
    phase: 'ready', createdAt: new Date(), lastUsedAt: new Date(), page: initial, pages: [initial],
    pageIdMap: new Map([['first', initial]]), pageToIdMap: new WeakMap([[initial, 'first']]),
    frameStack: [], activeMocks: new Map(), executedInstructions: new Map(), instructionCount: 0,
    context: { async clearCookies() {}, async clearPermissions() {} } };
};
class Pipeline {
  async initialize() {}
  async verifyPipeline() { return { scriptLoaded: true, scriptReady: true, inMainContext: true }; }
}
let contextCount = 0;
const { SessionManager } = driver('session/manager', {
  'node:path': path, '../utils': utils,
  './context-builder': { async buildContext() {
    contextCount++;
    return { context: { async newPage() { return page('fresh'); }, async close() {} },
      actualViewport: { width: 1280, height: 720, source: 'requested' },
      serviceWorkerController: { async enable() {} } };
  } },
  uuid: { v4: crypto.randomUUID }, '../recording': { RecordingPipelineManager: Pipeline },
  '../service-worker': {}, '../infra': guard, './browser-manager': {}, './audio': {}, './audio/device-evidence': {},
  './state-machine': machine, './session-decisions': decisions, './diagnostic-logger': { setupDiagnosticLogging() {} },
  '../instrumentation': { resolveInstrumentation: () => ({}), safeInvoke: async () => {} },
  '../tracing': {}, './session-inspection': inspection, './session-reset': reset,
  './session-teardown': teardown, './electron-target': {},
});
const manager = () => new SessionManager({ session: { maxConcurrent: 1, idleTimeoutMs: 1000 } }, {
  async getHostAudioCapability() { return {}; }, async getAudioStrategy() { return 'native'; },
  async getBrowser() { return {}; },
});

async function sessionProbes() {
  const m = manager();
  const admitted = await Promise.allSettled(['a', 'b'].map(execution_id => m.startSession({ execution_id, reuse_mode: 'fresh' })));
  record('admission-distinct-ids', 'maxConcurrent=1 admits at most one concurrent new session',
    { fulfilled: admitted.filter(r => r.status === 'fulfilled').length, sessions: m.sessions.size, contexts_created: contextCount }, m.sessions.size <= 1);
  const control = manager(), before = contextCount;
  const same = await Promise.all([1, 2].map(() => control.startSession({ execution_id: 'same', reuse_mode: 'fresh' })));
  record('admission-same-id-control', 'Concurrent same-execution starts share one session',
    { same_session: same[0].sessionId === same[1].sessionId, contexts_created: contextCount - before }, same[0].sessionId === same[1].sessionId && contextCount - before === 1);

  const retryManager = manager(), active = session();
  active.phase = 'executing';
  retryManager.sessions.set(active.id, active);
  const beforeAccepts = retryManager.canAcceptInstructions(active.id);
  const retry = await retryManager.startSession({ execution_id: 'owner', reuse_mode: 'reuse' });
  record('active-instruction-start-retry', 'A start retry does not make a still-executing session accept another instruction',
    { same_lease: retry.leaseId === 'lease', phase_after_retry: active.phase, accepted_before: beforeAccepts, accepts_after: retryManager.canAcceptInstructions(active.id) }, !retryManager.canAcceptInstructions(active.id));

  const ready = session('ready-control');
  retryManager.sessions.clear(); retryManager.sessions.set(ready.id, ready);
  const readyRetry = await retryManager.startSession({ execution_id: 'owner', reuse_mode: 'reuse' });
  record('ready-start-retry-control', 'A ready session retry preserves identity and lease',
    { session_id: readyRetry.sessionId, lease_id: readyRetry.leaseId }, readyRetry.sessionId === ready.id && readyRetry.leaseId === ready.leaseId);

  const clean = session('second-tab'), second = page('second');
  clean.pages.push(second); clean.page = second; clean.currentPageIndex = 1;
  clean.pageIdMap.set('second', second); clean.pageToIdMap.set(second, 'second');
  const first = clean.pages[0];
  await reset.resetSessionState(clean);
  record('clean-active-second-tab', 'Successful reset retains an open current page and closes discarded pages',
    { current_page_closed: clean.page.closed, first_page_closed: first.closed, retained_page_ids: clean.pages.map(p => p.id), page_map_ids: [...clean.pageIdMap.keys()], phase: clean.phase }, !clean.page.closed && first.closed);

  const failing = session('reset-fault');
  failing.page.goto = async () => { throw new Error('synthetic navigation failure'); };
  let error;
  try { await reset.resetSessionState(failing); } catch (e) { error = e.message; }
  const cleanup = decisions.shouldCleanupSession(failing, 1000, Date.now() + 3600000);
  record('failed-reset-lifecycle', 'A failed reset exits resetting into a recoverable or explicitly failed state',
    { error, phase: failing.phase, idle_cleanup_eligible_one_hour_later: cleanup }, failing.phase !== 'resetting');
  const external = session('external-control'); external.externalTarget = {};
  let refused = false;
  try { await reset.resetSessionState(external); } catch { refused = true; }
  record('external-target-reset-control', 'Reset refuses an owner-controlled target without changing its phase',
    { refused, phase: external.phase }, refused && external.phase === 'ready');

  const closing = manager(), broken = session('teardown-fault');
  let pageCloseAttempts = 0, contextCloseAttempts = 0, traceAttempts = 0;
  broken.page.video = () => null;
  broken.page.close = async () => { pageCloseAttempts++; throw new Error('synthetic page close failure'); };
  broken.context.close = async () => { contextCloseAttempts++; throw new Error('synthetic context close failure'); };
  broken.tracing = true; broken.tracePath = '/synthetic/never-written-trace.zip';
  broken.context.tracing = { async stop() { traceAttempts++; throw new Error('synthetic trace flush failure'); } };
  closing.sessions.set(broken.id, broken);
  let receipt, status;
  const closeRoute = driver('routes/session-close', {
    '../utils': utils, './session-run': { clearSessionIdempotencyCache() {} },
    '../handlers/download': { clearSessionDownloadCache() {} },
    '../middleware': {
      parseJsonBody: async () => ({ execution_id: 'owner', lease_id: 'lease' }),
      sendJson(_res, code, body) { status = code; receipt = body; },
      sendError(_res, error) { status = 500; receipt = { success: false, error: error.message }; },
    },
  });
  await closeRoute.handleSessionClose({}, {}, broken.id, closing);
  record('teardown-failure-receipt', 'Failed browser cleanup and trace flush remain owned and appear in the close result',
    { page_close_attempts: pageCloseAttempts, context_close_attempts: contextCloseAttempts, trace_attempts: traceAttempts,
      page_still_open: !broken.page.closed, sessions_tracked: closing.sessions.size, status, close_result: receipt },
    closing.sessions.size > 0 && receipt.success !== true);

  const normalOwner = manager();
  const normal = session('normal-close'); let contextClosed = false;
  normal.page.video = () => null; normal.context.close = async () => { contextClosed = true; };
  normalOwner.sessions.set(normal.id, normal);
  await closeRoute.handleSessionClose({}, {}, normal.id, normalOwner);
  record('normal-close-control', 'Successful cleanup closes owned resources and removes the session',
    { page_closed: normal.page.closed, context_closed: contextClosed, tracked_sessions: normalOwner.sessions.size, status, close_result: receipt },
    normal.page.closed && contextClosed && normalOwner.sessions.size === 0 && receipt.success === true);
}

function frameHarness() {
  let cursor = 0, cells = [], effects = [], timerID = 0, now = 1700000000000;
  const timers = new Map(), rafs = new Map(), sockets = [], decodes = [], draws = [], polls = [];
  const sameDeps = (a, b) => a && b && a.length === b.length && a.every((x, i) => Object.is(x, b[i]));
  const react = {
    useRef(value) { const i = cursor++; return cells[i] ??= { current: value }; },
    useState(value) { const i = cursor++; cells[i] ??= { value }; return [cells[i].value, value => { cells[i].value = value; }]; },
    useMemo(fn, deps) { const i = cursor++; if (!sameDeps(cells[i]?.deps, deps)) cells[i] = { deps, value: fn() }; return cells[i].value; },
    useCallback(fn, deps) { return react.useMemo(() => fn, deps); },
    useEffect(fn, deps) { const i = cursor++; if (!sameDeps(cells[i]?.deps, deps)) { effects.push({ i, fn, previous: cells[i]?.cleanup }); cells[i] = { deps }; } },
  };
  const ctx = { drawImage(bitmap) { draws.push(bitmap.id); }, clearRect() {} };
  const canvas = { width: 1280, height: 720, getContext: () => ctx };
  const store = { isValidated: false, setFrameDimensions() {}, setDisplayDimensions() {} };
  const stats = { stats: {}, recordFrame() {}, reset() {} };
  class Socket {
    constructor(url) { this.url = url; sockets.push(this); }
    close() { this.closed = true; } // async close-event timing is outside this probe
  }
  class Clock extends Date { static now() { return now; } }
  const exports = evaluate('ui/src/domains/recording/capture/useFrameStream.ts', {
    react, '@/config': { getConfig: async () => ({ API_URL: 'https://fixture.invalid' }) },
    '@/contexts/WebSocketContext': { useWebSocket() {} }, '../hooks/useFrameStats': { useFrameStats: () => stats },
    '@utils/latencyLogger': { LatencyLogger: class { record() {} getSampleCount() { return 1; } } },
    '../stores': { useSessionStore: selector => selector(store) },
  }, {
    Date: Clock, ArrayBuffer, DataView, Blob, performance, atob, WebSocket: Socket,
    document: { hidden: false, addEventListener() {}, removeEventListener() {}, createElement: () => canvas },
    fetch: async url => {
      if (url === '/config') return { ok: true, json: async () => ({ playwrightDriverPort: 24485 }) };
      polls.push(url); return { status: 304 };
    },
    setTimeout(fn) { const id = ++timerID; timers.set(id, fn); return id; }, clearTimeout(id) { timers.delete(id); },
    requestAnimationFrame(fn) { const id = ++timerID; rafs.set(id, fn); return id; }, cancelAnimationFrame(id) { rafs.delete(id); },
    createImageBitmap(blob) { return new Promise(resolve => decodes.push({ blob, resolve })); },
  });
  function render(sessionId = 'session-a') {
    cursor = 0; effects = [];
    const hook = exports.useFrameStream({ sessionId, pageId: 'page-a' }); hook.canvasRef.current = canvas;
    for (const e of effects) e.previous?.();
    for (const e of effects) cells[e.i].cleanup = e.fn();
    return hook;
  }
  function send(time = now) {
    now = time;
    const data = new ArrayBuffer(9); new DataView(data).setBigInt64(0, BigInt(now));
    sockets.at(-1).onmessage({ data });
  }
  async function resolve(index, id) {
    const bitmap = { id, width: 1280, height: 720, closed: false, close() { this.closed = true; } };
    decodes[index].resolve(bitmap); await drain(); return bitmap;
  }
  function paint() { for (const [id, fn] of [...rafs]) { rafs.delete(id); fn(); } }
  function unmount() { for (const cell of cells) cell?.cleanup?.(); }
  return { render, send, resolve, paint, unmount, sockets, decodes, draws, polls, timers, rafs };
}

async function connectedHarness() { const h = frameHarness(); h.render(); await drain(); h.sockets[0].onopen(); h.render(); await drain(); return h; }
async function frameProbes() {
  const noFrames = await connectedHarness();
  record('connected-without-frames-fallback', 'Polling continues after socket opens until a frame has arrived',
    { polling_calls: noFrames.polls.length, scheduled_fallback_polls: noFrames.timers.size, binary_frames: noFrames.decodes.length }, noFrames.timers.size > 0);
  noFrames.unmount();

  const burst = await connectedHarness();
  for (let i = 0; i < 100; i++) burst.send(1700000000000 + i);
  record('decode-admission-bound', 'Slow decoding uses bounded active work plus latest pending frame, independent of burst length',
    { frames: 100, unresolved_decodes: burst.decodes.length }, burst.decodes.length <= 2);
  burst.unmount();
  for (let i = 0; i < burst.decodes.length; i++) await burst.resolve(i, `burst-${i}`);
  burst.paint();

  const collision = await connectedHarness();
  collision.send(1700000000000); collision.send(1700000000000);
  await collision.resolve(1, 'newer'); collision.paint();
  await collision.resolve(0, 'older'); collision.paint();
  record('same-millisecond-frame-order', 'Older frame cannot overwrite newer frame after out-of-order decode',
    { drawn_order: collision.draws }, collision.draws.at(-1) === 'newer');
  collision.unmount();

  const sequence = await connectedHarness();
  sequence.send(1700000000000); sequence.send(1700000000001);
  await sequence.resolve(1, 'newer'); sequence.paint();
  const discarded = await sequence.resolve(0, 'older'); sequence.paint();
  record('distinct-timestamp-frame-control', 'Older frame with a distinct timestamp is discarded and closed',
    { drawn_order: sequence.draws, older_bitmap_closed: discarded.closed }, sequence.draws.join() === 'newer' && discarded.closed);
  sequence.unmount();

  const switched = await connectedHarness(); switched.send();
  switched.render('session-b'); await drain();
  await switched.resolve(0, 'old-session'); switched.paint();
  record('decode-after-session-switch', 'Old-session decode cannot draw into the new session canvas',
    { drawn_after_switch: switched.draws }, switched.draws.length === 0);
  switched.unmount();

  const unmounted = await connectedHarness(); unmounted.send(); unmounted.unmount();
  const lateBitmap = await unmounted.resolve(0, 'late');
  record('decode-after-unmount', 'An in-flight decode finishing after cleanup is closed without scheduling a new paint',
    { scheduled_paints_after_cleanup: unmounted.rafs.size, bitmap_closed: lateBitmap.closed }, unmounted.rafs.size === 0 && lateBitmap.closed);
  unmounted.paint(); unmounted.unmount();

  const lateConnect = frameHarness(); lateConnect.render(); lateConnect.unmount(); await drain();
  record('config-resolution-after-unmount', 'An unfinished config request cannot create a socket after effect cleanup',
    { socket_count: lateConnect.sockets.length, open_sockets: lateConnect.sockets.filter(s => !s.closed).length }, lateConnect.sockets.every(s => s.closed));
  lateConnect.unmount();
}

(async () => {
  await sessionProbes(); await frameProbes();
  console.log(JSON.stringify({ schema_version: 1, observed_at: new Date().toISOString(),
    scope: 'Actual manager/guard/decisions/reset and UI frame hook; synthetic browser/context/pages, hook scheduler, socket, clock, decoder and canvas. No browser or React-renderer certification.',
    source_sha256: sources, results }, null, 2));
})().catch(error => { console.error(error); process.exitCode = 1; });
