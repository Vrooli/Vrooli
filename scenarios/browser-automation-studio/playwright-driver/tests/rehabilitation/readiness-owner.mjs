import { createHash, randomUUID } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, renameSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { setTimeout as delay } from 'node:timers/promises';

const SCRIPT_PATH = fileURLToPath(import.meta.url);
const SCENARIO_ROOT = path.resolve(path.dirname(SCRIPT_PATH), '../../..');
const DEFAULT_API_URL = 'http://127.0.0.1:17116';
const DEFAULT_DRIVER_URL = 'http://127.0.0.1:24485';
const CONNECT_EXECUTE_ADHOC = '/browser_automation_studio.v1.WorkflowsService/ExecuteAdhocWorkflow';
const CONNECT_GET_EXECUTION = '/browser_automation_studio.v1.ExecutionsService/GetExecution';
const TARGET_SAMPLES = 100;
const TARGET_P95_MS = 1000;
const HOLD_SESSION_MS = 1500;

export function nearestRankPercentile(values, percentile) {
  if (!Array.isArray(values) || values.length === 0) return null;
  if (!(percentile > 0 && percentile <= 1)) throw new Error('percentile must be in (0, 1]');
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.ceil(percentile * sorted.length) - 1];
}

export function assessWarmCohort(attempts, requiredSamples = TARGET_SAMPLES) {
  const complete = attempts.length === requiredSamples && attempts.every((attempt) => attempt.status === 'passed');
  const values = attempts.filter((attempt) => attempt.status === 'passed').map((attempt) => attempt.latency_ms);
  const p95 = complete ? nearestRankPercentile(values, 0.95) : null;
  return {
    complete,
    sample_count: attempts.length,
    successful_count: values.length,
    p95_ms: p95,
    within_band: complete && p95 <= TARGET_P95_MS,
  };
}

export function sessionsAreIdle(snapshot) {
  return Number(snapshot?.summary?.active) === 0 && Number(snapshot?.summary?.total) === 0;
}

function cliArgs(argv) {
  const values = new Map();
  for (let i = 0; i < argv.length; i += 1) {
    if (argv[i].startsWith('--')) values.set(argv[i].slice(2), argv[i + 1]);
  }
  return {
    apiUrl: values.get('api-url') || process.env.BAS_API_URL || DEFAULT_API_URL,
    driverUrl: values.get('driver-url') || process.env.BAS_DRIVER_URL || DEFAULT_DRIVER_URL,
    output: values.get('output') || '',
    count: Number(values.get('count') || TARGET_SAMPLES),
  };
}

async function getJson(url, options = {}) {
  const response = await fetch(url, { ...options, signal: AbortSignal.timeout(15_000) });
  const raw = await response.text();
  let body;
  try {
    body = JSON.parse(raw);
  } catch {
    body = { raw };
  }
  if (!response.ok) throw new Error(`${options.method || 'GET'} ${url} returned ${response.status}: ${JSON.stringify(body).slice(0, 500)}`);
  return body;
}

function hashFile(relativePath) {
  const contents = readFileSync(path.join(SCENARIO_ROOT, relativePath));
  return createHash('sha256').update(contents).digest('hex');
}

function outputPath(requested) {
  if (requested) return path.resolve(requested);
  const id = randomUUID();
  return path.join(SCENARIO_ROOT, '.vrooli/runtime/rehabilitation-evidence', `readiness-warm-${new Date().toISOString().replaceAll(':', '-')}-${id}.json`);
}

function atomicWrite(filePath, body) {
  const temporaryPath = `${filePath}.tmp`;
  writeFileSync(temporaryPath, `${JSON.stringify(body, null, 2)}\n`, { flag: 'w' });
  renameSync(temporaryPath, filePath);
}

async function listSessions(driverUrl) {
  const snapshot = await getJson(`${driverUrl}/observability/sessions`);
  if (!Array.isArray(snapshot.sessions) || !snapshot.summary) throw new Error('driver session snapshot is missing its sessions or summary');
  return snapshot;
}

function readinessFlow() {
  return {
    metadata: {
      name: 'BAS rehabilitation warm readiness probe',
      description: 'Local about:blank tab readiness measurement; no external site or account effects.',
      executionMode: 'mutating',
    },
    nodes: [
      { id: 'open-local-tab', action: { type: 'ACTION_TYPE_NAVIGATE', navigate: { url: 'about:blank' } } },
      { id: 'hold-tab-for-observation', action: { type: 'ACTION_TYPE_WAIT', wait: { durationMs: HOLD_SESSION_MS } } },
    ],
    edges: [{ id: 'open-to-hold', source: 'open-local-tab', target: 'hold-tab-for-observation', type: 'WORKFLOW_EDGE_TYPE_SMOOTHSTEP' }],
  };
}

async function executeAdhoc(apiUrl, index) {
  return getJson(`${apiUrl}${CONNECT_EXECUTE_ADHOC}`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      flowDefinition: readinessFlow(),
      waitForCompletion: false,
      parameters: { projectRoot: SCENARIO_ROOT },
      metadata: { name: `readiness-warm-${index}` },
    }),
  });
}

async function getExecution(apiUrl, executionId) {
  return getJson(`${apiUrl}${CONNECT_GET_EXECUTION}`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ executionId }),
  });
}

function sessionForOwner(snapshot, executionId) {
  return snapshot.sessions.find((session) => session.owner_execution_id === executionId) || null;
}

function usableTab(session) {
  return session && session.page_count >= 1 && session.current_url === 'about:blank'
    && !['failed', 'closed', 'closing'].includes(String(session.phase).toLowerCase());
}

async function awaitUsableTab(apiUrl, driverUrl, executionId, startedAt, attempt) {
  const deadline = Date.now() + 5000;
  let checks = 0;
  while (Date.now() < deadline) {
    const snapshot = await listSessions(driverUrl);
    checks += 1;
    const ownerSession = sessionForOwner(snapshot, executionId);
    if (usableTab(ownerSession)) {
      return { latency_ms: performance.now() - startedAt, observer_checks: checks, session: ownerSession };
    }

    const foreign = snapshot.sessions.filter((session) => session.owner_execution_id !== executionId);
    if (foreign.length > 0) throw new Error(`unrelated session appeared during sample (${foreign.length} active)`);

    if (checks % 10 === 0) {
      const execution = await getExecution(apiUrl, executionId);
      const status = execution.execution?.status;
      if (status && status !== 'EXECUTION_STATUS_RUNNING' && status !== 'EXECUTION_STATUS_PENDING') {
        throw new Error(`execution became ${status} before a usable tab appeared: ${execution.execution?.error || 'no error detail'}`);
      }
    }
    await delay(10);
  }
  throw new Error(`no usable owned tab observed within 5000 ms for attempt ${attempt}`);
}

async function awaitOwnerCleanup(driverUrl, executionId) {
  const deadline = Date.now() + 7000;
  while (Date.now() < deadline) {
    const snapshot = await listSessions(driverUrl);
    if (!sessionForOwner(snapshot, executionId)) return true;
    await delay(25);
  }
  return false;
}

function sourceDigests() {
  const paths = [
    'api/automation/executor/simple_executor.go',
    'api/automation/session/session.go',
    'playwright-driver/src/session/manager.ts',
    'playwright-driver/src/session/browser-manager.ts',
    'playwright-driver/tests/rehabilitation/readiness-owner.mjs',
  ];
  return Object.fromEntries(paths.map((sourcePath) => [sourcePath, hashFile(sourcePath)]));
}

async function measure(apiUrl, driverUrl, index) {
  const before = await listSessions(driverUrl);
  if (!sessionsAreIdle(before)) {
    return { attempt: index, status: 'not_run', active_sessions_before: before.summary.active, reason: 'refused_non_idle_driver' };
  }

  const startedAt = performance.now();
  let executionId;
  let observation;
  try {
    const response = await executeAdhoc(apiUrl, index);
    executionId = response.executionId;
    if (!executionId) throw new Error(`adhoc response omitted executionId: ${JSON.stringify(response)}`);
    observation = await awaitUsableTab(apiUrl, driverUrl, executionId, startedAt, index);
    const cleanup = await awaitOwnerCleanup(driverUrl, executionId);
    if (!cleanup) throw new Error(`owned session ${executionId} remained active after its observation workflow`);
    const after = await listSessions(driverUrl);
    if (!sessionsAreIdle(after)) throw new Error(`unrelated session appeared after sample (${after.summary.active} active)`);
    return {
      attempt: index,
      status: 'passed',
      execution_id: executionId,
      latency_ms: observation.latency_ms,
      observer_checks: observation.observer_checks,
      session_phase: observation.session.phase,
      page_count: observation.session.page_count,
      current_url: observation.session.current_url,
      active_sessions_after: after.summary.active,
    };
  } catch (error) {
    if (executionId) await awaitOwnerCleanup(driverUrl, executionId).catch(() => false);
    return {
      attempt: index,
      status: 'failed',
      execution_id: executionId,
      latency_ms: null,
      error: error instanceof Error ? error.message : String(error),
    };
  }
}

async function main() {
  const args = cliArgs(process.argv.slice(2));
  if (!Number.isInteger(args.count) || args.count < 1 || args.count > 100) throw new Error('--count must be an integer from 1 to 100');
  const output = outputPath(args.output);
  mkdirSync(path.dirname(output), { recursive: true });
  if (existsSync(output)) throw new Error(`receipt already exists: ${output}`);

  const apiHealth = await getJson(`${args.apiUrl}/api/v1/health`);
  const driverHealth = await getJson(`${args.driverUrl}/health`);
  if (apiHealth.readiness !== true || driverHealth.ready !== true) throw new Error('managed BAS API and driver must both be healthy');
  const initialSessions = await listSessions(args.driverUrl);
  const receipt = {
    schema_version: 1,
    evidence_kind: 'readiness_warm_owner_cohort',
    outcome_id: 'readiness',
    status: 'running',
    operation_id: `readiness-warm-${randomUUID()}`,
    started_at: new Date().toISOString(),
    contract: {
      id: 'bas-rehabilitation-v1',
      sha256: hashFile('docs/internal/REFRACTOR_CONTRACT.json'),
      row: 'readiness',
      band: 'Warm usable tab p95 <=1 s over 100 trials; cold usable browser p95 <=5 s over 30 trials per required platform.',
    },
    identity: {
      managed_build_identity: apiHealth.build_identity,
      driver_build_identity: driverHealth.build_identity || null,
      source_sha256: sourceDigests(),
      platform: `${os.platform()}-${os.arch()}`,
      cpu_model: os.cpus()[0]?.model || null,
      logical_cpu_count: os.cpus().length,
      total_memory_bytes: os.totalmem(),
      node_version: process.version,
      browser_version: driverHealth.browser?.version || null,
    },
    environment: {
      api_url: args.apiUrl,
      driver_url: args.driverUrl,
      fixture: 'about:blank',
      external_site_load_included: false,
      viewport: { width: 1280, height: 720, device_scale_factor: 1 },
      initial_active_sessions: initialSessions.summary.active,
    },
    method: {
      clock: 'Node performance.now from before the API ExecuteAdhocWorkflow Connect request until the driver reports the matching execution with one about:blank page',
      ready_predicate: 'matching owner_execution_id, page_count >= 1, current_url == about:blank, nonterminal session phase',
      observation_interval_ms: 'driver session checks every 10 ms, execution failure checks every 10 observations',
      workflow: 'Navigate to about:blank, then wait 1500 ms to hold the usable tab open for observation; await normal execution cleanup before next sample.',
      cohort: 'one declared warmup and 100 sequential warm trials; refuse any trial when the driver has unrelated active sessions',
      warmup: null,
      limitations: ['Linux local managed candidate only; cold-start cohort and other required platforms are not measured.', 'This receipt is the warm half of the readiness contract and does not alone qualify the row.'],
    },
    attempts: [],
    summary: null,
  };

  const initialIdle = sessionsAreIdle(initialSessions);
  if (!initialIdle) {
    receipt.status = 'not_run';
    receipt.method.limitations.push(`Driver had ${initialSessions.summary.active} active session(s) at preflight; no session was touched.`);
    atomicWrite(output, receipt);
    console.log(JSON.stringify({ status: receipt.status, output, active_sessions: initialSessions.summary.active }));
    return 2;
  }

  atomicWrite(output, receipt);
  const warmup = await measure(args.apiUrl, args.driverUrl, 0);
  receipt.method.warmup = warmup;
  atomicWrite(output, receipt);
  if (warmup.status !== 'passed') {
    receipt.status = 'failed';
    receipt.summary = assessWarmCohort([] , args.count);
    atomicWrite(output, receipt);
    console.log(JSON.stringify({ status: receipt.status, output, warmup }));
    return 1;
  }

  for (let index = 1; index <= args.count; index += 1) {
    const attempt = await measure(args.apiUrl, args.driverUrl, index);
    receipt.attempts.push(attempt);
    receipt.summary = assessWarmCohort(receipt.attempts, args.count);
    atomicWrite(output, receipt);
    if (attempt.status === 'not_run') break;
  }
  receipt.status = receipt.summary.within_band ? 'passed' : 'failed';
  receipt.completed_at = new Date().toISOString();
  receipt.summary = assessWarmCohort(receipt.attempts, args.count);
  atomicWrite(output, receipt);
  console.log(JSON.stringify({ status: receipt.status, output, ...receipt.summary }));
  return receipt.status === 'passed' ? 0 : 1;
}

if (process.argv[1] && path.resolve(process.argv[1]) === SCRIPT_PATH) {
  main().then((code) => { process.exitCode = code; }).catch((error) => {
    console.error(error instanceof Error ? error.message : String(error));
    process.exitCode = 2;
  });
}
