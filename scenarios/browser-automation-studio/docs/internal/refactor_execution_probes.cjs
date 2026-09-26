/* Actual run route, instruction pipeline, caches, telemetry collectors and manager
 * phase methods. Browser handlers, capture I/O, middleware and proto codec are
 * synthetic seams. Feed JSON from refactor_execution_probes.go as argv[2].
 * No browser, HTTP socket, real session or production evidence is touched.
 * Exit 0 means diagnostics completed; inspect expected_behavior_met.
 */
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const { EventEmitter } = require('node:events');
const { createRequire } = require('node:module');
const root = path.resolve(__dirname, '../..');
const req = createRequire(path.join(root, 'playwright-driver/package.json'));
const ts = req('typescript');
if (!process.argv[2]) throw new Error('Pass the Go probe output JSON path as argv[2]');
const goBytes = fs.readFileSync(process.argv[2]);
const go = JSON.parse(goBytes);
const hashes = {}, results = [];
const add = (id, expected, actual, met) => results.push({ id, expected, actual, expected_behavior_met: met });
function load(p, mocks) {
  const relative = `playwright-driver/src/${p}.ts`, filename = path.join(root, relative);
  const source = fs.readFileSync(filename, 'utf8');
  hashes[relative] = crypto.createHash('sha256').update(source).digest('hex');
  const code = ts.transpileModule(source, { fileName: filename, compilerOptions: {
    module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true,
  } }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(code, { module, exports: module.exports, console, Date, Buffer, setInterval, clearInterval,
    require(name) { if (!Object.hasOwn(mocks, name)) throw new Error(`Unexpected dependency ${p}: ${name}`); return mocks[name]; },
  }, { filename });
  return module.exports;
}
const silent = new Proxy({}, { get: () => () => {} });
const utils = { logger: silent, metrics: new Proxy({}, { get: () => silent }), scopedLog: (_, text) => text,
  LogContext: {}, normalizeConsoleLogType: value => value, SessionNotFoundError: Error, InvalidInstructionError: Error };
const constants = { MAX_CONSOLE_ENTRIES: 100, MAX_NETWORK_EVENTS: 100, MAX_EXECUTED_INSTRUCTIONS_PER_SESSION: 1000 };
const machine = load('session/state-machine', { '../utils': utils });
const managerModule = load('session/manager', {
  'node:path': path, '../utils': utils, './context-builder': {}, uuid: {}, '../recording': {},
  '../service-worker': {}, '../infra': {}, './browser-manager': {}, './audio': {}, './audio/device-evidence': {},
  './state-machine': machine, './session-decisions': {}, './diagnostic-logger': {}, '../instrumentation': {},
  '../tracing': {}, './session-inspection': {}, './session-reset': {}, './session-teardown': {}, './electron-target': {},
});
const collectors = load('telemetry/collector', { '../utils': utils, '../constants': constants });
const outcomes = { buildStepOutcome: x => ({ success: x.result.success, failure: x.result.error, consoleLogs: x.consoleLogs, networkEvents: x.networkEvents }), toDriverOutcome: x => x };
const proto = {
  // Codec is a deliberate seam, not a generated-schema validation claim.
  parseProtoLenient: (_schema, x) => x,
  toHandlerInstruction: x => ({ ...x, nodeId: x.nodeId ?? x.node_id }),
  getActionType: x => String(x.action?.type ?? 'navigate'),
};
function fixture(handler) {
  const stats = { handler_calls: 0, screenshots: 0, dom_captures: 0, console_sessions: 0 };
  const telemetry = load('telemetry/orchestrator', {
    './collector': collectors,
    './screenshot': { captureScreenshot: async () => { stats.screenshots++; return { base64: 'synthetic' }; } },
    './dom': { captureDOMSnapshot: async () => { stats.dom_captures++; return { html: '<synthetic/>' }; } },
    './element-context': {},
  });
  const execution = load('execution/instruction-executor', {
    '../proto': proto, '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb': { ScreenshotCapturePolicy: { NEVER: 1, ON_FAILURE: 2 } },
    '../telemetry': telemetry,
    '../outcome': outcomes,
    '../utils': utils, '../instrumentation': { resolveInstrumentation: () => ({}), safeInvoke: async () => {} },
  });
  const route = load('routes/session-run', {
    '../middleware': {
      parseJsonBody: async r => r.body,
      sendJson(res, status, body) { res.status = status; res.body = body; },
      sendError(res, error) { res.status = 500; res.body = { error: error.message }; },
    }, '../execution': execution, '../outcome': outcomes, 'node:crypto': crypto,
    '../utils': utils, '../proto': proto, '../constants': constants,
  });
  const page = new EventEmitter(); page.url = () => 'https://fixture.invalid'; page.isClosed = () => false;
  page.context = () => ({ newCDPSession: async () => {
    const cdp = new EventEmitter(); stats.console_sessions++;
    cdp.send = async () => ({});
    cdp.detach = async () => { stats.console_sessions--; };
    page.consoleSession = cdp; return cdp;
  } });
  const session = { id: 'synthetic-session', ownerExecutionId: 'current-owner', leaseId: 'current-lease', spec: {},
    page, context: {}, phase: 'ready', instructionCount: 0, lastUsedAt: new Date(), instructionReceipts: new Map(), lastInstructionSequence: 0 };
  // Actual manager methods; construction/browser startup are outside this fixture.
  const manager = Object.create(managerModule.SessionManager.prototype);
  manager.sessions = new Map([[session.id, session]]); manager.instrumentation = {};
  const config = { telemetry: { console: { enabled: true, maxEntries: 100 }, network: { enabled: true, maxEvents: 100 },
    screenshot: { enabled: true }, dom: { enabled: true } } };
  const registry = { getHandler: () => ({ execute: async (...args) => { stats.handler_calls++; return handler ? handler(stats.handler_calls, page, ...args) : { success: true }; } }) };
  async function run(body, headers = {}) {
    const res = { set statusCode(value) { this.status = value; }, setHeader() {}, end(value) { this.body = JSON.parse(value); } }; await route.handleSessionRun({ body, headers }, res, session.id, manager, registry, config, silent, utils.metrics); return res;
  }
  return { run, session, manager, stats, page, dispose: () => {} };
}
const body = (index = 0, url = 'https://fixture.invalid/a') => ({ execution_id: 'current-owner', lease_id: 'current-lease', operation_sequence: index + 1, invocation_id: `visit-${index}`, attempt: 1, instruction: { index, node_id: 'same-node', action: { type: 1, navigate: { url } } } });
const consoleEvent = { type: 'error', args: [{ value: 'synthetic failure context' }], stackTrace: { callFrames: [{ url: 'https://fixture.invalid', lineNumber: 1, columnNumber: 1 }] } };

(async () => {
  for (const phase of ['initializing', 'resetting', 'closing', 'executing']) {
    const f = fixture(); f.session.phase = phase;
    const res = await f.run(body());
    add(`run-admission-${phase}`, 'Only ready/recording sessions admit a new instruction',
      { phase_before: phase, status: res.status, handler_calls: f.stats.handler_calls, phase_after: f.session.phase }, f.stats.handler_calls === 0);
    f.dispose();
  }
  const stale = fixture();
  const staleRes = await stale.run({ ...body(), execution_id: 'old-owner', lease_id: 'expired-lease' });
  add('run-stale-lease', 'A command from a previous execution lease cannot mutate the current session',
    { current_owner: stale.session.ownerExecutionId, supplied_owner: 'old-owner', status: staleRes.status, handler_calls: stale.stats.handler_calls }, stale.stats.handler_calls === 0);
  stale.dispose();

  for (const name of ['retry', 'loop']) {
    const packets = go.wire_packets[name]; if (packets.length !== 2) throw new Error(`Expected two real Go ${name} requests`);
    const f = fixture(call => name === 'retry' && call === 1 ? { success: false, error: { retryable: true, message: 'synthetic transient failure' } } : { success: true });
    f.session.ownerExecutionId = packets[0].body.execution_id;
    f.session.leaseId = packets[0].body.lease_id;
    const responses = [];
    for (const packet of packets) responses.push(await f.run(packet.body));
    add(`go-to-driver-${name}`, name === 'retry' ? 'An explicitly new retry attempt can recover from a completed transient failure' : 'Two loop iterations perform their browser handler twice',
      { requests: packets.length, handler_calls: f.stats.handler_calls, response_successes: responses.map(r => r.body.success) }, f.stats.handler_calls === 2 && responses[1].body.success);
    f.dispose();
  }

  const duplicate = fixture(); await duplicate.run(body()); await duplicate.run(body());
  add('transport-repeat-control', 'Retransmitting one operation returns its result without repeating the effect',
    { handler_calls: duplicate.stats.handler_calls }, duplicate.stats.handler_calls === 1); duplicate.dispose();
  const distinct = fixture(); await distinct.run(body(0)); await distinct.run(body(1));
  add('distinct-step-control', 'Distinct step identities execute separately', { handler_calls: distinct.stats.handler_calls }, distinct.stats.handler_calls === 2); distinct.dispose();

  const changed = fixture(); await changed.run(body(0, 'https://fixture.invalid/a')); const changedRes = await changed.run(body(0, 'https://fixture.invalid/b'));
  add('changed-payload-same-identity', 'A reused invocation identity with changed payload produces an explicit conflict',
    { handler_calls: changed.stats.handler_calls, status: changedRes.status, returned_success: changedRes.body.success }, changedRes.status === 409); changed.dispose();

  const cached = fixture(); await cached.run(body(0), { 'x-idempotency-key': 'current-lease:1' });
  cached.session.ownerExecutionId = 'next-owner'; cached.session.leaseId = 'next-lease'; cached.session.instructionReceipts.clear(); cached.session.lastInstructionSequence = 0;
  const cacheRes = await cached.run({ ...body(1), execution_id: 'next-owner', lease_id: 'next-lease' }, { 'x-idempotency-key': 'current-lease:1' });
  add('idempotency-cache-lease-and-payload', 'Cache hits are scoped to lease and validated request identity',
    { current_owner: cached.session.ownerExecutionId, handler_calls: cached.stats.handler_calls, status: cacheRes.status, error: cacheRes.body.error, returned_success: cacheRes.body.success }, cacheRes.body.error === 'X-Idempotency-Key must match lease_id:operation_sequence' && cached.stats.handler_calls === 1);
  cached.dispose();

  let effects = 0;
  const uncertain = fixture(call => {
    effects++;
    if (call === 1) throw new Error('synthetic failure after a simulated effect');
    return { success: true };
  });
  const first = await uncertain.run(body(), { 'x-idempotency-key': 'current-lease:1' });
  const second = await uncertain.run(body(), { 'x-idempotency-key': 'current-lease:1' });
  add('retry-after-uncertain-effect', 'Retransmission of an uncertain operation does not silently repeat its effect',
    { first_status: first.status, second_status: second.status, simulated_effects: effects, handler_calls: uncertain.stats.handler_calls }, effects === 1);
  uncertain.dispose();

  for (const throws of [false, true]) {
    const f = fixture((_count, page) => {
      page.consoleSession.emit('Runtime.consoleAPICalled', { ...consoleEvent, timestamp: Date.now() });
      if (throws) throw new Error('synthetic unexpected handler throw');
      return { success: false, error: { message: 'synthetic returned failure', retryable: false } };
    });
    const res = await f.run(body());
    const logs = res.body.consoleLogs ?? [];
    add(throws ? 'thrown-handler-failure-evidence' : 'returned-handler-failure-control', 'A failed instruction retains available console context and attempts failure capture',
      { status: res.status, screenshots: f.stats.screenshots, dom_captures: f.stats.dom_captures, returned_console_logs: logs.length, remaining_console_listeners: f.page.consoleSession.listenerCount('Runtime.consoleAPICalled'), remaining_console_sessions: f.stats.console_sessions, cached_instructions: f.session.instructionReceipts.size },
      logs.length === 1 && f.stats.screenshots === 1 && f.page.consoleSession.listenerCount('Runtime.consoleAPICalled') === 0 && f.stats.console_sessions === 0);
    f.dispose();
  }

  console.log(JSON.stringify({ schema_version: 1, observed_at: new Date().toISOString(),
    scope: 'Actual run route, executor pipeline, cache, manager phase methods and telemetry collectors. Synthetic browser handlers/capture, middleware and proto-codec seams. Go packet replay composes wire identity behavior but is not a live HTTP/browser E2E test.',
    go_probe_output_sha256: crypto.createHash('sha256').update(goBytes).digest('hex'), source_sha256: hashes, results }, null, 2));
})().catch(error => { console.error(error); process.exitCode = 1; });
