/* Investigation only: actual frame manager/CDP strategy with controlled async
 * browser, clock, socket and collector seams. No browser or socket is opened.
 * Exit 0 means probes completed; inspect expected_behavior_met.
 */
const fs = require('node:fs'), path = require('node:path'), vm = require('node:vm');
const crypto = require('node:crypto');
const { createRequire } = require('node:module');
const root = path.resolve(__dirname, '../..');
const ts = createRequire(path.join(root, 'playwright-driver/package.json'))('typescript');
const sources = {}, results = [];
const record = (id, expected, actual, met) => results.push({ id, expected, actual, expected_behavior_met: met });
const drain = async () => { for (let i = 0; i < 35; i++) await Promise.resolve(); };
const deferred = () => { let resolve, reject; const promise = new Promise((a, b) => { resolve = a; reject = b; }); return { promise, resolve, reject }; };
const silent = new Proxy({}, { get: () => () => {} });
const utils = { logger: silent, metrics: new Proxy({}, { get: () => silent }), scopedLog: (_, s) => s, LogContext: {} };
function driver(name, mocks = {}, globals = {}) {
  const relative = `playwright-driver/src/${name}.ts`, filename = path.join(root, relative);
  const source = fs.readFileSync(filename, 'utf8');
  sources[relative] = crypto.createHash('sha256').update(source).digest('hex');
  const code = ts.transpileModule(source, { fileName: filename, compilerOptions: {
    module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true,
  } }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(code, { module, exports: module.exports, Buffer, console,
    require(dep) { if (!Object.hasOwn(mocks, dep)) throw new Error(`Unexpected dependency ${relative}: ${dep}`); return mocks[dep]; },
    ...globals,
  }, { filename });
  return module.exports;
}
function managerHarness(strategy) {
  const sockets = [];
  const manager = driver('frame-streaming/manager', {
    '../utils': utils, '../config': { loadConfig: () => ({ performance: { enabled: false }, frameStreaming: { useScreencast: true, fallbackToPolling: false } }) },
    '../performance': { PerfCollector: { fromConfig: () => ({ recordFrame() {}, recordSkipped() {}, shouldLogSummary: () => false }) } },
    './strategies': { createCdpScreencastStrategy: () => strategy, createPollingStrategy: () => ({ name: 'polling' }) },
    './websocket': { buildWebSocketUrl: () => 'ws://synthetic.invalid', createWebSocketConnectionManager: () => {
      const s = { closed: false, connect() {}, close() { this.closed = true; },
        isReady: () => true, getWebSocket: () => ({ readyState: 1, send() {} }) };
      sockets.push(s); return s;
    } },
  });
  return { ...manager, sockets, begin() { manager.startFrameStreaming('fixture', { getSession: () => ({ page: {} }) }, { callbackUrl: 'http://synthetic.invalid', quality: 65, fps: 30 }); } };
}
function fakeHandle(stopGate) {
  return { active: true, stops: 0, getFrameCount: () => 0, isActive() { return this.active; },
    async stop() { this.stops++; if (stopGate) await stopGate.promise; this.active = false; } };
}
async function managerProbes() {
  const gate = deferred(), handles = [];
  const h = managerHarness({ name: 'cdp-screencast', isSupported: async () => true,
    async start() { const handle = fakeHandle(handles.length === 0 ? gate : null); handles.push(handle); return handle; } });
  h.begin(); await drain(); h.begin(); await drain();
  const trackedBefore = !!h.getFrameStreamSettings('fixture');
  gate.resolve(); await drain();
  record('replacement-old-stop-deletes-new-owner', 'Finishing old stream cleanup preserves tracking of its replacement',
    { tracked_before_old_stop: trackedBefore, tracked_after_old_stop: !!h.getFrameStreamSettings('fixture'),
      replacement_active: handles[1].active, replacement_socket_open: !h.sockets[1].closed }, !!h.getFrameStreamSettings('fixture'));
  await h.stopFrameStreaming('fixture');
  record('replacement-remains-stoppable', 'A stop request reaches every retained replacement resource',
    { replacement_stop_calls: handles[1].stops, replacement_active: handles[1].active, socket_closed: h.sockets[1].closed },
    !handles[1].active && h.sockets[1].closed);
  await handles[1].stop(); h.sockets[1].close(); // Synthetic harness cleanup only.

  const startGate = deferred(), late = fakeHandle();
  const pending = managerHarness({ name: 'cdp-screencast', isSupported: async () => true, start: () => startGate.promise });
  pending.begin(); await drain(); await pending.stopFrameStreaming('fixture');
  startGate.resolve(late); await drain();
  record('start-completes-after-stop', 'A capture start completing after stop is cancelled or immediately disposed',
    { tracked: !!pending.getFrameStreamSettings('fixture'), late_handle_active: late.active, late_stop_calls: late.stops }, !late.active);
  await late.stop();

  const failure = managerHarness({ name: 'cdp-screencast', isSupported: async () => true, async start() { throw new Error('synthetic start failure'); } });
  failure.begin(); await drain();
  record('failed-start-socket-cleanup', 'Failed capture startup closes its connection as well as removing its registry entry',
    { tracked: !!failure.getFrameStreamSettings('fixture'), socket_closed: failure.sockets[0].closed }, failure.sockets[0].closed);
  failure.sockets[0].close();

  const normal = fakeHandle(), control = managerHarness({ name: 'cdp-screencast', isSupported: async () => true, start: async () => normal });
  control.begin(); await drain(); await control.stopFrameStreaming('fixture');
  record('settled-start-stop-control', 'A settled stream stop closes its handle/socket and removes tracking',
    { active: normal.active, socket_closed: control.sockets[0].closed, tracked: !!control.getFrameStreamSettings('fixture') },
    !normal.active && control.sockets[0].closed && !control.getFrameStreamSettings('fixture'));
}

function cdpHarness() {
  const intervals = new Map(), timeouts = new Map(), sessions = [], sent = [], stats = [];
  let timerId = 0, nextSessionGate = null, ready = true, now = 1000;
  const createSession = () => {
    const calls = [], handlers = new Map();
    const s = { calls, handlers, detached: false, on: (name, cb) => handlers.set(name, cb),
      async send(name, params) { calls.push({ name, params }); }, async detach() { this.detached = true; },
      emit(data, id = 1) { handlers.get('Page.screencastFrame')?.({ data: Buffer.from(data).toString('base64'), sessionId: id, metadata: {} }); } };
    sessions.push(s); return s;
  };
  const makePage = label => ({ label, size: { width: 1280, height: 720 }, isClosed: () => false,
    viewportSize() { return this.size; }, async setViewportSize(size) { this.size = size; },
    context: () => ({ browser: () => ({ browserType: () => ({ name: () => 'chromium' }) }),
      async newCDPSession() {
        const s = createSession(), gate = nextSessionGate; nextSessionGate = null;
        if (gate) await gate.promise;
        return s;
      } }),
  });
  let currentPage = makePage('a');
  const interfaces = driver('frame-streaming/strategies/interface');
  const { CdpScreencastStrategy } = driver('frame-streaming/strategies/cdp-screencast', {
    '../../utils': utils, './interface': interfaces,
  }, {
    performance: { now: () => now }, Date, global: {},
    setInterval(cb, ms) { const id = ++timerId; intervals.set(id, { cb, ms }); return id; },
    clearInterval(id) { intervals.delete(id); },
    setTimeout(cb, ms) { const id = ++timerId; timeouts.set(id, { cb, ms }); return id; },
    clearTimeout(id) { timeouts.delete(id); },
  });
  const strategy = new CdpScreencastStrategy();
  return { strategy, sessions, sent, stats, intervals, timeouts,
    provider: () => currentPage,
    ws: { isReady: () => ready, getWebSocket: () => ({ readyState: ready ? 1 : 0, send: b => sent.push(b) }) },
    reporter: { onFrameSent: s => stats.push(s), onFrameSkipped() {} },
    config: { sessionId: 'fixture', quality: 65, targetFps: 30, scale: 'css', includePerfHeaders: false },
    ready(value) { ready = value; }, clock(value) { now = value; },
    delayNextSession() { nextSessionGate = deferred(); return nextSessionGate; },
    switchPage() { currentPage = makePage('b'); },
    async tick() { for (const { cb } of [...intervals.values()]) cb(); await drain(); },
    payloads() { return sent.map(b => b.subarray(8).toString()); },
  };
}
async function cdpProbes() {
  const h = cdpHarness(), handle = await h.strategy.start(h.provider, h.config, h.ws, h.reporter);
  h.ready(false); h.sessions[0].emit('initial-frame'); await drain();
  h.ready(true); await h.tick();
  record('ready-without-new-paint-buffer-flush', 'A retained initial frame becomes deliverable when transport is ready without requiring another page paint',
    { frames_sent_after_ready: h.sent.length, acknowledged: h.sessions[0].calls.filter(c => c.name === 'Page.screencastFrameAck').length }, h.sent.length === 1);
  h.sessions[0].emit('next-frame', 2); await drain();
  record('next-paint-flush-control', 'A later compositor frame delivers current content through the ready transport',
    { payload_order: h.payloads() }, h.payloads().at(-1) === 'next-frame');
  await handle.stop();

  const cross = cdpHarness(), crossHandle = await cross.strategy.start(cross.provider, cross.config, cross.ws, cross.reporter);
  cross.ready(false); cross.sessions[0].emit('old-page-pending'); await drain();
  cross.switchPage(); await cross.tick(); cross.ready(true);
  cross.sessions[1].emit('new-page-current'); await drain();
  record('buffer-survives-page-generation', 'Pending old-page bytes are discarded before the new page can publish frames',
    { payload_order: cross.payloads() }, !cross.payloads().includes('old-page-pending'));
  await crossHandle.stop();

  const race = cdpHarness(), raceHandle = await race.strategy.start(race.provider, race.config, race.ws, race.reporter);
  const gate = race.delayNextSession(), resizing = raceHandle.updateViewport(1400, 800);
  await drain(); await raceHandle.stop(); gate.resolve(); await resizing;
  record('cdp-restart-completes-after-stop', 'A resize restart cannot acquire an active CDP screencast after stream stop',
    { handle_active: raceHandle.isActive(), new_session_detached: race.sessions[1].detached,
      new_start_calls: race.sessions[1].calls.filter(c => c.name === 'Page.startScreencast').length },
    race.sessions[1].detached || !race.sessions[1].calls.some(c => c.name === 'Page.startScreencast'));
  await race.sessions[1].detach();

  const controls = cdpHarness(), good = await controls.strategy.start(controls.provider, controls.config, controls.ws, controls.reporter);
  await good.updateViewport(1400, 800);
  record('settled-resize-control', 'A settled resize starts capture with the new dimensions and detaches the old CDP session',
    { old_detached: controls.sessions[0].detached, viewport: controls.provider().viewportSize(),
      start: controls.sessions[1].calls.find(c => c.name === 'Page.startScreencast')?.params },
    controls.sessions[0].detached && controls.sessions[1].calls.some(c => c.name === 'Page.startScreencast' && c.params.maxWidth === 1400));
  await good.stop();
  record('settled-cdp-stop-control', 'Settled stop detaches CDP and removes the page monitor',
    { active: good.isActive(), detached: controls.sessions[1].detached, intervals: controls.intervals.size },
    !good.isActive() && controls.sessions[1].detached && controls.intervals.size === 0);
}
async function settingsProbes() {
  const c = cdpHarness();
  let handle;
  const wrapped = { name: 'cdp-screencast', isSupported: async () => true,
    async start(_page, config) { handle = await c.strategy.start(c.provider, config, c.ws, c.reporter); return handle; } };
  const m = managerHarness(wrapped); m.begin(); await drain();
  const changed = m.updateFrameStreamSettings('fixture', { quality: 20, fps: 1, perfMode: true });
  const reported = m.getFrameStreamSettings('fixture');
  record('quality-update-reports-before-application', 'A successful quality update is applied or explicitly reported as pending',
    { changed, reported_quality: reported.quality,
      applied_start_qualities: c.sessions.flatMap(s => s.calls.filter(call => call.name === 'Page.startScreencast').map(call => call.params.quality)) },
    c.sessions.some(s => s.calls.some(call => call.name === 'Page.startScreencast' && call.params.quality === 20)));
  for (let i = 0; i < 60; i++) { c.clock(1000 + i * 1000 / 60); c.sessions[0].emit(`frame-${i}`, i + 1); await drain(); }
  record('fps-setting-not-effective', 'A reported FPS limit is enforced or explicitly reported unsupported by the active strategy',
    { reported_fps: reported.fps, reported_current_fps: reported.currentFps, frames_in_one_simulated_second: c.sent.length,
      update_target_fps_supported: typeof handle.updateTargetFps === 'function' }, c.sent.length <= 2);
  record('perf-header-setting-not-effective', 'Enabling performance headers changes outgoing framing before success is reported',
    { reported_perf_mode: reported.perfMode, first_payload_uses_timestamp_format: c.payloads()[0] === 'frame-0' },
    c.payloads()[0] !== 'frame-0');
  await handle.updateViewport(1500, 850);
  record('quality-applies-on-restart-control', 'The retained quality setting takes effect at the next screencast restart',
    { restart_quality: c.sessions[1].calls.find(call => call.name === 'Page.startScreencast')?.params.quality },
    c.sessions[1].calls.some(call => call.name === 'Page.startScreencast' && call.params.quality === 20));
  await m.stopFrameStreaming('fixture');
}
(async () => {
  await managerProbes(); await cdpProbes(); await settingsProbes();
  console.log(JSON.stringify({ schema_version: 1, observed_at: new Date().toISOString(),
    scope: 'Actual frame manager and CDP strategy with synthetic browser/CDP, collector, socket and deterministic clock/scheduler. No live latency or memory measurement.',
    source_sha256: sources, results }, null, 2));
})().catch(error => { console.error(error); process.exitCode = 1; });
