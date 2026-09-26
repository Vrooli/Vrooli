/* Isolated investigation probes. No browser/service starts or real user data.
 * Run: node scenarios/browser-automation-studio/docs/internal/refactor_probes.cjs
 * expected_behavior_met=false records a defect, not a passing regression test.
 * Installed dependencies are reused; no installation or product edits occur.
 */
const path = require('node:path');
const vm = require('node:vm');
const { createRequire } = require('node:module');
const { EventEmitter } = require('node:events');
const driver = path.resolve(__dirname, '../../playwright-driver');
const req = createRequire(path.join(driver, 'package.json'));
req('ts-node').register({ transpileOnly: true, project: path.join(driver, 'tsconfig.json') });
// Stub only the logging facade to avoid importing unrelated generated ESM
// contracts through utils/index. The network probe does not collect console logs.
const Module = require('node:module');
const silent = new Proxy({}, { get: () => () => {} });
const load = (p, stubs = {}) => {
  const original = Module._load;
  Module._load = function(name, parent, ...args) {
    if (parent?.filename.startsWith(driver + '/src/') && Object.hasOwn(stubs, name)) return stubs[name];
    if (parent?.filename.startsWith(driver + '/src/') && /^\.\.?\/(?:.*\/)?utils$/.test(name)) {
      return { logger: silent, scopedLog: (_, message) => message, LogContext: { RECORDING: 'recording' } };
    }
    return original.call(this, name, parent, ...args);
  };
  try { return req(path.join(driver, 'src', p)); } finally { Module._load = original; }
};
const results = [];
const record = (id, expected, actual, met) => results.push({ id, expected, actual, expected_behavior_met: met });
const drain = async () => { for (let i = 0; i < 8; i++) await Promise.resolve(); };

async function networkProbe() {
  const { NetworkCollector } = load('telemetry/collector.ts');
  const providerReq = createRequire(req.resolve('rebrowser-playwright'));
  const core = path.dirname(providerReq.resolve('playwright-core'));
  const { Request } = req(path.join(core, 'lib/client/network.js'));
  const makeRequest = url => Object.assign(Object.create(Request.prototype), {
    url: () => url, method: () => 'GET', resourceType: () => 'fetch',
  });
  const a = makeRequest('https://fixture.invalid/a'), b = makeRequest('https://fixture.invalid/b');
  const page = new EventEmitter();
  const collector = new NetworkCollector(page);
  page.emit('request', a); page.emit('request', b);
  page.emit('response', { request: () => a, status: () => 201, ok: () => true });
  page.emit('response', { request: () => b, status: () => 202, ok: () => true, url: b.url });
  const events = collector.getEvents().map(e => ({url: e.url, status: e.status}));
  record('network-request-identity', 'Two responses retain their own request URL and status',
    { request_strings: [String(a), String(b)], distinct_request_objects: a !== b, events },
    events.length === 2 && events[0].url === a.url() && events[1].url === b.url());
  collector.clear();
  for (const [request, status] of [[a, 201], [b, 202]]) {
    page.emit('request', request);
    page.emit('response', { request: () => request, status: () => status, ok: () => true, url: request.url });
  }
  const sequential = collector.getEvents().map(e => ({ url: e.url, status: e.status }));
  record('network-sequential-control', 'The same fixture without overlapping requests preserves two correct responses',
    sequential, sequential.length === 2 && sequential[0].url === a.url() && sequential[1].url === b.url());
  collector.dispose();
}

async function poolProbe() {
  const { BrowserPool } = load('session/browser-pool.ts');
  const pool = new BrowserPool();
  let rejectFirst, launchCount = 0;
  const launch = () => {
    launchCount++;
    if (launchCount === 1) return new Promise((_, reject) => { rejectFirst = reject; });
    return Promise.resolve({ id: launchCount, isConnected: () => true });
  };
  const pending = [1, 2, 3].map(() => pool.getOrLaunch('same-key', launch));
  rejectFirst(new Error('synthetic first launch failure'));
  const outcomes = await Promise.allSettled(pending);
  const successes = outcomes.filter(x => x.status === 'fulfilled').map(x => x.value.id);
  const closed = [];
  await pool.closeAll(async (_, browser) => closed.push(browser.id));
  record('browser-pool-retry-coalescing', 'Waiting callers share one bounded retry and its browser is closed by the pool',
    { launchCount, successful_browser_ids: successes, closed_browser_ids: closed },
    launchCount === 2 && successes.length === 3 && new Set(successes).size === 1 &&
      closed.length === 1 && closed[0] === successes[0]);

  const shutdownPool = new BrowserPool();
  let finishLaunch;
  const inFlight = shutdownPool.getOrLaunch('late', () => new Promise(resolve => { finishLaunch = resolve; }));
  const closedOnShutdown = [];
  const shutdown = shutdownPool.closeAll(async (_, browser) => { closedOnShutdown.push(browser); });
  const lateBrowser = { id: 'late', isConnected: () => true };
  finishLaunch(lateBrowser);
  const [requestOutcome] = await Promise.allSettled([inFlight]);
  await shutdown;
  record('browser-pool-shutdown-during-launch', 'Shutdown owns and closes a concurrently completing launch',
    { request_status: requestOutcome.status, live_browser_after_close: !!shutdownPool.get('late'), closed_ids: closedOnShutdown.map(browser => browser.id) },
    requestOutcome.status === 'rejected' && !shutdownPool.get('late') && closedOnShutdown.length === 1 && closedOnShutdown[0] === lateBrowser);
}

async function directFrameProbe() {
  const { DirectFrameServer } = load('frame-streaming/websocket/server.ts');
  const server = new DirectFrameServer(0);
  // Call actual connection/broadcast methods with synthetic sockets. No port is opened.
  server.isRunning = true;
  const makeSocket = () => Object.assign(new EventEmitter(), {
    readyState: 1, bufferedAmount: 64 * 1024 * 1024, sent: [],
    send(data) { this.sent.push(data); }, close() {},
  });
  const anonymous = makeSocket(), wrong = makeSocket(), matching = makeSocket();
  server.handleConnection(anonymous, { url: '/frames', headers: { origin: 'https://fixture.invalid' } });
  server.handleConnection(wrong, { url: '/frames?session_id=other', headers: {} });
  server.handleConnection(matching, { url: '/frames?session_id=target', headers: {} });
  for (let i = 0; i < 10; i++) server.broadcast(Buffer.from('synthetic-frame'), 'target');
  const binaryCount = socket => socket.sent.filter(Buffer.isBuffer).length;
  record('direct-frame-subscription', 'A connection without a session receives no session frames',
    { anonymous_frames: binaryCount(anonymous), wrong_session_frames: binaryCount(wrong), matching_frames: binaryCount(matching) },
    binaryCount(anonymous) === 0 && binaryCount(wrong) === 0);
  record('direct-frame-backpressure', 'A socket already holding 64 MiB cannot accumulate ten more frame sends',
    { initial_buffered_bytes: matching.bufferedAmount, additional_frames: binaryCount(matching) }, binaryCount(matching) <= 1);
}

async function inputOrderingProbe() {
  // Synthetic HTTP parsing and page I/O; the input handler itself is unchanged.
  const { handleRecordInput } = load('routes/record-mode/recording-input.ts', {
    '../../middleware': { parseJsonBody: async request => request.body, sendJson() {}, sendError(_, error) { throw error; } },
    '../../frame-streaming': { updateFrameStreamViewport() {} },
  });
  const effects = [];
  let resumeDown, enteredDown;
  const downEntered = new Promise(resolve => { enteredDown = resolve; });
  const page = { mouse: {
    move: async x => { if (x === 1) { enteredDown(); await new Promise(resolve => { resumeDown = resolve; }); } },
    down: async () => { effects.push('down'); }, up: async () => { effects.push('up'); },
  } };
  const manager = { getSession: () => ({ page }) };
  const down = handleRecordInput({ body: { type: 'pointer', action: 'down', x: 1 } }, {}, 'synthetic', manager, {});
  await downEntered;
  await handleRecordInput({ body: { type: 'pointer', action: 'up', x: 2 } }, {}, 'synthetic', manager, {});
  resumeDown(); await down;
  record('concurrent-input-order', 'A down request started before up applies down before up',
    { applied_order: effects }, effects.join(',') === 'down,up');
}

function recorderFixture(fetchImpl) {
  const handlers = new Map(), timers = new Map(), storage = new Map();
  let timerId = 0;
  const target = name => ({
    addEventListener(event, fn) { const key = name + ':' + event; handlers.set(key, [...(handlers.get(key) || []), fn]); },
    removeEventListener() {},
  });
  const field = {
    tagName: 'INPUT', id: 'fixture-input', type: 'text', value: '', disabled: false,
    className: '', classList: [], attributes: [], textContent: '', innerText: '',
    parentElement: null, parentNode: null, previousElementSibling: null, children: [],
    getAttribute(name) { return name === 'type' ? this.type : null; },
    getBoundingClientRect() { return { x: 0, y: 0, width: 100, height: 30 }; }, closest() { return null; },
  };
  const document = Object.assign(target('document'), {
    body: field, documentElement: field, querySelectorAll: () => [field], querySelector: () => field,
  });
  class FixedDate extends Date { static now() { return 10000; } }
  const context = Object.assign(target('window'), {
    document, console: { log() {}, warn() {}, error() {} }, Date: FixedDate, URL,
    location: { href: 'https://fixture.invalid/' }, history: { pushState() {}, replaceState() {} },
    getComputedStyle: () => ({ display: 'block', visibility: 'visible', opacity: '1' }),
    CSS: { escape: s => s }, Node: { ELEMENT_NODE: 1 },
    sessionStorage: { getItem: k => storage.get(k) || null, setItem: (k, v) => storage.set(k, v), removeItem: k => storage.delete(k) },
    setTimeout(fn) { timers.set(++timerId, fn); return timerId; }, clearTimeout(id) { timers.delete(id); },
    setInterval() { return 999; }, clearInterval() {}, fetch: fetchImpl,
  });
  context.window = context;
  vm.createContext(context);
  const { generateRecordingInitScript } = load('recording/capture/init-script-generator.ts');
  vm.runInContext(generateRecordingInitScript(), context);
  if (!context.__vrooli_recording_ready) throw new Error(context.__vrooli_recording_init_error || 'recorder did not initialize');
  const emit = (where, event, data) => (handlers.get(where + ':' + event) || []).forEach(fn => fn(data));
  return {
    context, field, storage,
    input(value) { field.value = value; emit('document', 'input', { target: field }); },
    flush() { const pending = [...timers.values()]; timers.clear(); pending.forEach(fn => fn()); },
    stop() { emit('window', 'message', { source: context, data: { type: '__VROOLI_RECORDING_CONTROL__', action: 'stop' } }); },
    pending() { return JSON.parse(storage.get('__vrooli_pending_events__') || '[]'); },
  };
}

async function recorderProbes() {
  const sent = [];
  const f = recorderFixture((url, options) => { sent.push(JSON.parse(options.body)); return Promise.resolve({ ok: true, status: 200 }); });
  for (const text of ['a', 'ab', '']) { f.input(text); f.flush(); await drain(); }
  record('recorder-clear-field', 'Input snapshots include the final empty value',
    { captured_texts: sent.map(e => e.payload.text) }, sent.at(-1)?.payload.text === '');
  f.field.type = 'password'; f.input('SYNTHETIC-NOT-A-REAL-SECRET'); f.flush(); await drain();
  const passwordEvent = sent.at(-1);
  record('recorder-password', 'A password sentinel is absent from raw recording transport',
    { transported_literal: passwordEvent.payload.text, input_type: passwordEvent.elementMeta.attributes.type },
    !JSON.stringify(passwordEvent).includes('SYNTHETIC-NOT-A-REAL-SECRET'));
  const beforeStop = sent.length;
  f.field.type = 'text'; f.input('last-edit-before-stop'); f.stop(); f.flush(); await drain();
  record('recorder-stop-flush', 'Stopping capture preserves a buffered final edit',
    { new_events_after_stop: sent.length - beforeStop }, sent.length > beforeStop);

  const rejected = recorderFixture(() => Promise.resolve({ ok: false, status: 500 }));
  rejected.input('must-remain-pending'); rejected.flush(); await drain();
  record('recorder-http-error-ack', 'An HTTP 500 does not acknowledge or discard a pending event',
    { pending_count: rejected.pending().length, success_count: rejected.context.__vrooli_recording_telemetry.eventsSendSuccess },
    rejected.pending().length === 1);

  const completions = [];
  const colliding = recorderFixture(() => new Promise(resolve => completions.push(resolve)));
  colliding.input('first'); colliding.flush(); colliding.input('second'); colliding.flush();
  const pendingBefore = colliding.pending().length;
  completions[0]({ ok: true, status: 200 }); await drain();
  record('recorder-timestamp-identity', 'Acknowledging one of two same-millisecond events retains the other',
    { pending_before: pendingBefore, pending_after_one_ack: colliding.pending().length }, colliding.pending().length === 1);
  completions[1]({ ok: true, status: 200 }); await drain();
}

(async () => {
  for (const probe of [networkProbe, poolProbe, directFrameProbe, inputOrderingProbe, recorderProbes]) {
    try { await probe(); } catch (e) { results.push({ probe: probe.name, probe_error: e.stack }); }
  }
  console.log(JSON.stringify({ schema_version: 1, observed_at: new Date().toISOString(), scope: 'isolated actual-module probes with synthetic I/O; not a browser E2E or certification run', results }, null, 2));
  if (results.some(r => r.probe_error)) process.exitCode = 2;
})();
