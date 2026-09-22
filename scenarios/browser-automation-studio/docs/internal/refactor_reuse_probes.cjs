/* Investigation only. Execute actual start/manager/reuse/reset/teardown owners
 * against synthetic contexts, profile markers, artifact paths and HTTP objects.
 * No browser, socket, profile, artifact file or service is opened or changed.
 * Exit 0 means observations completed; inspect expected_behavior_met.
 */
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const crypto = require('node:crypto');
const { createRequire } = require('node:module');
const root = path.resolve(__dirname, '../..');
const ts = createRequire(path.join(root, 'playwright-driver/package.json'))('typescript');
const sources = {}, results = [];
const record = (id, expected, actual, met) => results.push({ id, expected, actual, expected_behavior_met: met });
function driver(name, mocks = {}) {
  const relative = `playwright-driver/src/${name}.ts`;
  const filename = path.join(root, relative), source = fs.readFileSync(filename, 'utf8');
  sources[relative] = crypto.createHash('sha256').update(source).digest('hex');
  const code = ts.transpileModule(source, { fileName: filename, compilerOptions: {
    module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022, esModuleInterop: true,
  } }).outputText;
  const module = { exports: {} };
  vm.runInNewContext(code, { module, exports: module.exports, crypto, URL,
    process: { env: {} }, console,
    require(dependency) {
      if (!Object.hasOwn(mocks, dependency)) throw new Error(`Unexpected dependency ${relative}: ${dependency}`);
      return mocks[dependency];
    },
  }, { filename });
  return module.exports;
}
const silent = new Proxy({}, { get: () => () => {} });
const utils = { logger: silent, metrics: new Proxy({}, { get: () => silent }),
  scopedLog: (_, text) => text, LogContext: {}, ResourceLimitError: Error,
  SessionNotFoundError: Error, InvalidInstructionError: Error, PlaywrightDriverError: Error };
const decisions = driver('session/session-decisions');
const machine = driver('session/state-machine', { '../utils': utils });
const inspection = driver('session/session-inspection', { './session-decisions': decisions });
const guard = driver('infra/in-flight-guard', { '../utils': utils });
const registry = driver('infra/session-cleanup-registry', { '../utils': utils });
const reset = driver('session/session-reset', { '../infra': registry, '../utils': utils });
const moves = [], traceStops = [];
const teardown = driver('session/session-teardown', {
  'node:fs': { constants: fs.constants },
  'node:fs/promises': { async stat() { return { isFile: () => true }; }, async access() {}, async mkdir() {}, async rename(from, to) { moves.push({ from, to }); } },
  'node:path': path, '../recording': { removeRecordingBuffer() {} }, '../utils': utils,
});
const appTarget = driver('session/electron-target', { 'node:path': path });
const artifactPaths = driver('session/artifact-paths', { path });
function page(viewport, video = false) {
  return { closed: false, isClosed() { return this.closed; }, on() {}, viewportSize: () => viewport,
    async goto() {}, async evaluate() {}, async unroute() {}, async close() { this.closed = true; },
    video: () => video ? { async path() { return '/synthetic/a/videos/random.webm'; } } : null };
}
const profile = name => ({ fingerprint: { locale: name === 'a' ? 'en-US' : 'fr-FR' },
  proxy: { enabled: true, server: `http://proxy-${name}.invalid:8080` } });
const storage = name => ({ cookies: [{ name: 'synthetic-identity', value: name, domain: 'fixture.invalid', path: '/' }], origins: [] });
const spec = (owner = 'a', changes = {}) => ({ execution_id: owner, workflow_id: `workflow-${owner}`,
  viewport: { width: 1280, height: 720 }, labels: { fake_microphone: '' }, reuse_mode: 'reuse',
  browser_profile: profile(owner), storage_state: storage(owner), ...changes });
let builds = 0;
function contextFor(request, video = false) {
  const p = page(request.viewport, video);
  return { marker: request.storage_state.cookies[0].value, originalProfile: request.browser_profile, p,
    async newPage() { return p; }, async close() {}, async clearCookies() { this.marker = null; },
    async clearPermissions() {}, tracing: { async stop(options) { traceStops.push(options.path); } } };
}
class Pipeline {
  async initialize() {}
  async verifyPipeline() { return { scriptLoaded: true, scriptReady: true, inMainContext: true }; }
}
const { SessionManager } = driver('session/manager', {
  'node:path': path, '../utils': utils,
  './context-builder': { async buildContext(_browser, request) {
    builds++;
    return { context: contextFor(request), actualViewport: { ...request.viewport, source: 'requested' },
      serviceWorkerController: { async enable() {} } };
  } },
  uuid: { v4: crypto.randomUUID }, '../recording': { RecordingPipelineManager: Pipeline },
  '../service-worker': {}, '../infra': guard, './browser-manager': {}, './audio': {}, './audio/device-evidence': {},
  './state-machine': machine, './session-decisions': decisions, './diagnostic-logger': { setupDiagnosticLogging() {} },
  '../instrumentation': { resolveInstrumentation: () => ({}), safeInvoke: async () => {} },
  '../tracing': {}, './session-inspection': inspection, './session-reset': reset,
  './session-teardown': teardown, './electron-target': appTarget,
});
const manager = () => new SessionManager({ session: { maxConcurrent: 8, idleTimeoutMs: 1000 } }, {
  async getHostAudioCapability() { return {}; }, async getAudioStrategy() { return 'native'; },
  async getBrowser() { return {}; },
  async connectOverCDP() { throw new Error('Probe must not reach a CDP connection'); },
});
function seed(m, { evidence = false, released = true, external = false } = {}) {
  const request = spec('a', evidence ? { required_capabilities: { video: true, har: true, tracing: true },
    artifact_paths: { root: '/synthetic/a' } } : {});
  const context = contextFor(request, evidence);
  const paths = artifactPaths.resolveArtifactPaths(request.artifact_paths, request.required_capabilities, 'a');
  const session = { id: 'synthetic-session', spec: request, ownerExecutionId: 'a', leaseId: 'lease-a',
    phase: 'ready', createdAt: new Date(), lastUsedAt: new Date(), context, page: context.p, pages: [context.p],
    pageIdMap: new Map([['first', context.p]]), pageToIdMap: new WeakMap([[context.p, 'first']]),
    frameStack: [], activeMocks: new Map(), executedInstructions: new Map(), instructionCount: 3,
    tracing: evidence, video: evidence, ...paths,
    externalTarget: external, browser: { async close() {} } };
  m.sessions.set(session.id, session);
  if (released) m.releaseExecutionLease(session.id, 'a', 'lease-a');
  return session;
}
const { handleSessionStart } = driver('routes/session-start', {
  '../middleware': { async parseJsonBody(req) { return req.body; },
    sendJson(res, status, body) { Object.assign(res, { status, body }); },
    sendError(res, error) { Object.assign(res, { status: 'error', error: error.message }); } },
  '../utils': utils, '../frame-streaming': { startFrameStreaming() { throw new Error('Unexpected stream start'); } },
});
async function start(m, request) {
  const res = {};
  await handleSessionStart({ body: request, headers: {} }, res, m, {});
  return res;
}
const target = kind => ({ target_kind: kind, target_id: 'target-b', cdp_endpoint: 'http://127.0.0.1:65530',
  cdp_transport: 'loopback-authenticated', renderer_id: 'renderer-b', renderer_url: 'http://fixture.invalid/',
  scenario_name: 'fixture-b', artifact_digest: 'synthetic-digest-b', context_id: 'context-b' });

async function isolationProbes() {
  for (const reuse_mode of ['reuse', 'clean']) {
    const m = manager(), old = seed(m), before = builds;
    const response = await start(m, spec('b', { reuse_mode, viewport: { width: 390, height: 844 } }));
    record(`${reuse_mode}-profile-isolation`, 'A different requested identity is applied in an isolated context or rejected', {
      status: response.status, same_context: m.sessions.get(response.body?.session_id)?.context === old.context,
      requested_identity: 'b', context_identity: old.context.marker,
      spec_identity_after: old.spec.storage_state.cookies[0].value, context_builds: builds - before,
    }, response.status !== 200 || old.context.marker === 'b');
    record(`${reuse_mode}-context-settings`, 'Accepted viewport, locale and proxy agree with effective context settings', {
      status: response.status, requested_viewport: { width: 390, height: 844 }, actual_viewport: response.body?.actual_viewport,
      requested_profile: profile('b'), effective_profile: old.context.originalProfile,
    }, response.status !== 200 || (old.context.originalProfile.fingerprint.locale === 'fr-FR' && old.page.viewportSize().width === 390));
  }
  const fresh = manager(), old = seed(fresh), response = await start(fresh, spec('b', { reuse_mode: 'fresh' }));
  const created = fresh.sessions.get(response.body?.session_id);
  record('fresh-context-control', 'Fresh request allocates a distinct context and passes the requested identity to its builder',
    { status: response.status, distinct_context: created?.context !== old.context, identity: created?.context.marker },
    response.status === 200 && created?.context !== old.context && created?.context.marker === 'b');
  for (const [id, options, changes] of [
    ['unreleased-lease-control', { released: false }, {}],
    ['different-label-control', {}, { labels: { fake_microphone: 'different.wav' } }],
  ]) {
    const m = manager(), s = seed(m, options), r = await start(m, spec('b', changes));
    record(id, 'An unreleased or differently labelled session is not transferred',
      { status: r.status, original_owner: s.ownerExecutionId, same_session: r.body?.session_id === s.id },
      r.status === 200 && s.ownerExecutionId === 'a' && r.body?.session_id !== s.id);
  }
  const m = manager(), s = seed(m), r = await start(m, spec('b', { browser_profile: profile('a'), storage_state: storage('a') }));
  record('compatible-reuse-control', 'Compatible released reuse transfers the lease and preserves the intended context',
    { status: r.status, reused: r.body?.reused, new_lease: r.body?.lease_id !== 'lease-a', identity: s.context.marker },
    r.status === 200 && r.body?.reused === true && r.body?.lease_id !== 'lease-a' && s.context.marker === 'a');
}

async function evidenceProbes() {
  const m = manager(), s = seed(m, { evidence: true });
  const request = spec('b', { browser_profile: profile('a'), storage_state: storage('a'),
    required_capabilities: { video: true, har: true, tracing: true }, artifact_paths: { root: '/synthetic/b' } });
  const response = await start(m, request);
  const wanted = artifactPaths.resolveArtifactPaths(request.artifact_paths, request.required_capabilities, 'b');
  const oldPaths = { tracePath: s.tracePath, harPath: s.harPath, videoDir: s.videoDir };
  record('reuse-artifact-destination', 'New execution evidence uses its requested destination and an explicit capture boundary',
    { status: response.status, requested: wanted, effective: oldPaths, current_owner: s.ownerExecutionId },
    response.status !== 200 || Object.entries(wanted).every(([k, v]) => s[k] === v));
  const closed = await m.closeSessionForLease(s.id, 'b', response.body.lease_id);
  record('reuse-artifact-close-receipt', 'The new owner close receipt and video name do not mix execution identities',
    { receipt: closed, trace_stop_paths: traceStops, simulated_video_moves: moves },
    closed.tracePath === wanted.tracePath && closed.harPath === wanted.harPath && closed.videoPaths.every(p => p.startsWith('/synthetic/b/')));

  const bare = manager(), bareSession = seed(bare), enabled = await start(bare, request);
  record('reuse-enables-required-evidence', 'A newly required capture capability starts collecting or admission fails',
    { status: enabled.status, requested: request.required_capabilities, effective_tracing: bareSession.tracing,
      effective_video: bareSession.video, effective_har: !!bareSession.harPath },
    enabled.status !== 200 || (bareSession.tracing && bareSession.video && !!bareSession.harPath));
  const invalid = await start(manager(), spec('b', { required_capabilities: { video: true } }));
  record('artifact-root-validation-control', 'The route rejects required recording without an artifact root', invalid,
    invalid.status === 'error' && invalid.error.includes('artifact_paths.root'));
}

async function targetProbes() {
  for (const kind of ['electron', 'android-webview']) {
    const request = spec('b', { app_target: target(kind) }); // Missing validation context must be rejected.
    const fresh = await start(manager(), { ...request, reuse_mode: 'fresh' });
    record(`${kind}-fresh-validation-control`, 'Fresh external target admission requires its validation context', fresh,
      fresh.status === 'error' && fresh.error === 'Electron validation context is required');
    const m = manager(), s = seed(m), reused = await start(m, request);
    record(`${kind}-reuse-validation`, 'Released reuse also enforces target identity and validation context',
      { status: reused.status, requested_target: kind, actual_external_target: s.externalTarget,
        spec_target_after: s.spec.app_target?.target_kind, retained_managed_context: m.sessions.get(reused.body?.session_id)?.context === s.context },
      reused.status !== 200);
  }
  const m = manager(), s = seed(m, { external: true });
  s.spec.app_target = target('electron');
  const r = await start(m, spec('b'));
  record('external-to-managed-reuse', 'A managed-browser request cannot silently inherit an external app renderer',
    { status: r.status, external_target_after: s.externalTarget, spec_target_after: s.spec.app_target ?? null },
    r.status !== 200 || !s.externalTarget);
}

(async () => {
  await isolationProbes(); await evidenceProbes(); await targetProbes();
  console.log(JSON.stringify({ schema_version: 1, observed_at: new Date().toISOString(),
    scope: 'Actual start route, manager, decision/reset/teardown and app-target validation; synthetic HTTP, contexts, profile markers, capture handles, filesystem and context-builder seam. No live browser/profile/artifact access.',
    source_sha256: sources, results }, null, 2));
})().catch(error => { console.error(error); process.exitCode = 1; });
