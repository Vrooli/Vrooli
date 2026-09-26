#!/usr/bin/env node
import { createHash, randomUUID } from 'node:crypto';
import { execFile } from 'node:child_process';
import { createServer } from 'node:http';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, resolve } from 'node:path';
import { promisify } from 'node:util';
import { setTimeout as delay } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

const execFileAsync = promisify(execFile);
const scenarioRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const api = process.env.BAS_API_URL || 'http://127.0.0.1:17116';
const driver = process.env.BAS_DRIVER_URL || 'http://127.0.0.1:24485';
// One extra interval is margin for scheduler and sample overhead: 61 samples
// at 1s took 59,988ms on a real owner run and correctly failed the 60s gate.
const samplesWanted = 62;
const intervalMs = 1000;
const maxIdlePssKiB = 300 * 1024;
const maxFixturePssKiB = 1024 * 1024;
const maxIdleCPUPercent = 2;
const sourceFiles = [
  'docs/internal/REFRACTOR_CONTRACT.json',
  'api/cmd/resource-budget-cohort/qualification.mjs',
  'api/automation/driver/client.go',
  'playwright-driver/src/routes/session-start.ts',
  'playwright-driver/src/session/manager.ts',
];
const observations = [];
let fixtureServer;
let fixturePort = 0;
let managedSession;

async function json(url, body) {
  const response = await fetch(url, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'application/json', 'Connect-Protocol-Version': '1' },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(30000),
  });
  const value = await response.json();
  if (!response.ok) throw new Error(`${url}: HTTP ${response.status}: ${JSON.stringify(value)}`);
  return value;
}

function sha256(data) {
  return createHash('sha256').update(data).digest('hex');
}

async function shaFile(path) {
  return sha256(await readFile(resolve(scenarioRoot, path)));
}

async function managedPIDs() {
  const { stdout } = await execFileAsync('vrooli', ['scenario', 'status', 'browser-automation-studio', '--json'], {
    cwd: resolve(scenarioRoot, '../..'), timeout: 15000, maxBuffer: 2 * 1024 * 1024,
  });
  const status = JSON.parse(stdout);
  const apiProcess = status.runtime?.process_records?.find((entry) => entry.step === 'start-api');
  if (!apiProcess?.pid) throw new Error('managed BAS status has no start-api process');
  const { stdout: listeners } = await execFileAsync('ss', ['-ltnp'], { timeout: 5000, maxBuffer: 1024 * 1024 });
  const port = new URL(driver).port || '80';
  const driverLine = listeners.split('\n').find((line) => line.includes(`:${port} `) || line.includes(`:${port}\t`));
  const driverPid = Number(driverLine?.match(/pid=(\d+)/)?.[1] || 0);
  if (!driverPid) throw new Error('cannot identify the managed driver process listening on port 24485');
  return { api: Number(apiProcess.pid), driver: driverPid };
}

async function readPssKiB(pid) {
  const text = await readFile(`/proc/${pid}/smaps_rollup`, 'utf8');
  const match = text.match(/^Pss:\s+(\d+) kB$/m);
  if (!match) throw new Error(`PSS unavailable for process ${pid}`);
  return Number(match[1]);
}

async function readCpuTicks(pid) {
  const text = await readFile(`/proc/${pid}/stat`, 'utf8');
  const rest = text.slice(text.lastIndexOf(')') + 2).trim().split(/\s+/);
  return Number(rest[11]) + Number(rest[12]);
}

async function readDriverHealth() {
  const health = await json(`${driver}/health`);
  if (health.ready !== true && health.status !== 'ok') throw new Error('managed driver is not ready');
  return health;
}

async function fixtureBrowserPids(driverPid) {
  const { stdout } = await execFileAsync('ps', ['-eo', 'pid=,ppid=,comm='], { timeout: 5000, maxBuffer: 1024 * 1024 });
  const processes = stdout.trim().split('\n').map((line) => {
    const match = line.trim().match(/^(\d+)\s+(\d+)\s+(.+)$/);
    return match ? { pid: Number(match[1]), ppid: Number(match[2]), comm: match[3] } : null;
  }).filter(Boolean);
  const descendants = new Set([driverPid]);
  let changed = true;
  while (changed) {
    changed = false;
    for (const process of processes) {
      if (descendants.has(process.ppid) && !descendants.has(process.pid)) {
        descendants.add(process.pid);
        changed = true;
      }
    }
  }
  return processes.filter((process) => descendants.has(process.pid) && /chrome|chromium/i.test(process.comm))
    .map((process) => process.pid);
}

async function sampleIdle(pids) {
  const ticksPerSecond = Number((await execFileAsync('getconf', ['CLK_TCK'], { timeout: 5000 })).stdout.trim());
  const samples = [];
  let prior = null;
  for (let index = 0; index < samplesWanted; index += 1) {
    const started = performance.now();
    const health = await readDriverHealth();
    const apiHealth = await json(`${api}/health`);
    if (apiHealth.status !== 'healthy' || apiHealth.build_identity !== buildBefore) {
      throw new Error('managed BAS API became unhealthy or changed build during resource sampling');
    }
    if (health.sessions !== 0 || health.active_recordings !== 0) {
      throw new Error(`idle resource sample invalid: managed driver has ${health.sessions} sessions and ${health.active_recordings} recordings`);
    }
    const pss = { api: await readPssKiB(pids.api), driver: await readPssKiB(pids.driver) };
    const ticks = { api: await readCpuTicks(pids.api), driver: await readCpuTicks(pids.driver) };
    const observedAt = new Date().toISOString();
    const elapsedSeconds = prior ? (performance.now() - prior.monotonicMs) / 1000 : null;
    const cpuPercent = prior ? {
      api: ((ticks.api - prior.ticks.api) / ticksPerSecond / elapsedSeconds) * 100,
      driver: ((ticks.driver - prior.ticks.driver) / ticksPerSecond / elapsedSeconds) * 100,
    } : null;
    samples.push({ observedAt, pssKiB: pss, combinedPssKiB: pss.api + pss.driver, cpuPercent, sessions: health.sessions, activeRecordings: health.active_recordings });
    prior = { ticks, monotonicMs: performance.now() };
    const remaining = intervalMs - (performance.now() - started);
    if (index < samplesWanted - 1 && remaining > 0) await delay(remaining);
  }
  return samples;
}

async function beginFixtureBrowser(driverPid) {
  fixtureServer = createServer((_request, response) => {
    response.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8', 'Cache-Control': 'no-store' });
    response.end('<!doctype html><title>BAS resource fixture</title><main id="ready">fixture ready</main>');
  });
  await new Promise((resolveListen, reject) => {
    fixtureServer.once('error', reject);
    fixtureServer.listen(0, '127.0.0.1', resolveListen);
  });
  fixturePort = fixtureServer.address().port;
  const executionId = randomUUID();
  const opened = await json(`${driver}/session/start`, {
    execution_id: executionId,
    workflow_id: 'resource-budget-fixture',
    viewport: { width: 1280, height: 800 },
    reuse_mode: 'fresh',
    base_url: `http://127.0.0.1:${fixturePort}`,
    required_capabilities: {},
  });
  managedSession = { executionId, sessionId: opened.session_id, leaseId: opened.lease_id };
  const navigated = await json(`${driver}/session/${opened.session_id}/run`, {
    execution_id: executionId,
    lease_id: opened.lease_id,
    operation_sequence: 1,
    invocation_id: randomUUID(),
    attempt: 1,
    instruction: {
      index: 0,
      node_id: 'resource-budget-fixture-navigation',
      action: { type: 'ACTION_TYPE_NAVIGATE', navigate: { url: `http://127.0.0.1:${fixturePort}/`, waitUntil: 'domcontentloaded' } },
    },
  });
  if (navigated.success === false || navigated.status === 'failed') throw new Error(`fixture browser navigation failed: ${JSON.stringify(navigated)}`);
  await delay(1000);
  const browserPids = await fixtureBrowserPids(driverPid);
  if (browserPids.length === 0) throw new Error('no Chromium descendant belongs to the managed driver fixture session');
  const processPss = await Promise.all(browserPids.map(async (pid) => ({ pid, pssKiB: await readPssKiB(pid) })));
  return { url: `http://127.0.0.1:${fixturePort}/`, browserPids: processPss, browserPssKiB: processPss.reduce((sum, item) => sum + item.pssKiB, 0), fixtureShellPid: process.pid, fixtureShellPssKiB: await readPssKiB(process.pid) };
}

async function cleanup() {
  if (managedSession) {
    try {
      await json(`${driver}/session/${managedSession.sessionId}/close`, {
        execution_id: managedSession.executionId,
        lease_id: managedSession.leaseId,
      });
      observations.push({ cleanup: 'fixture-session-closed' });
    } finally {
      managedSession = undefined;
    }
  }
  if (fixtureServer) {
    fixtureServer.closeAllConnections();
    await new Promise((resolveClose) => fixtureServer.close(resolveClose));
    fixtureServer = undefined;
  }
}

let buildBefore = '';
const runID = new Date().toISOString().replace(/[:.]/g, '-');
let outputPath = process.env.BAS_RESOURCE_BUDGET_RECEIPT || resolve(scenarioRoot, `.vrooli/runtime/rehabilitation-evidence/resource-budget-w189-receipt-${runID}.json`);
let rawPath = process.env.BAS_RESOURCE_BUDGET_RAW || resolve(scenarioRoot, `.vrooli/runtime/rehabilitation-evidence/resource-budget-owner-w189-${runID}.json`);
let failure;
let measurement;
try {
  if (process.platform !== 'linux') throw new Error(`Linux /proc collector cannot measure ${process.platform}; Windows private memory must be reported by its native owner`);
  const apiHealth = await json(`${api}/health`);
  buildBefore = apiHealth.build_identity;
  if (apiHealth.status !== 'healthy' || !buildBefore) throw new Error('managed BAS API is not healthy or has no build identity');
  const pids = await managedPIDs();
  const initialDriver = await readDriverHealth();
  if (initialDriver.sessions !== 0 || initialDriver.active_recordings !== 0) throw new Error('managed BAS driver must be idle before sampling');
  const idleSamples = await sampleIdle(pids);
  const fixture = await beginFixtureBrowser(pids.driver);
  await cleanup();
  const finalHealth = await readDriverHealth();
  const afterHealth = await json(`${api}/health`);
  const cpu = idleSamples.slice(1).map((sample) => (sample.cpuPercent.api || 0) + (sample.cpuPercent.driver || 0));
  const sortedCPU = [...cpu].sort((a, b) => a - b);
  const p95Index = Math.min(sortedCPU.length - 1, Math.ceil(sortedCPU.length * 0.95) - 1);
  const maxCombinedPssKiB = Math.max(...idleSamples.map((sample) => sample.combinedPssKiB));
  const averageCPUPercent = cpu.reduce((sum, value) => sum + value, 0) / cpu.length;
  const p95CPUPercent = sortedCPU[p95Index];
  const fixtureAndShellPssKiB = fixture.browserPssKiB + fixture.fixtureShellPssKiB;
  if (afterHealth.build_identity !== buildBefore || finalHealth.sessions !== 0 || finalHealth.active_recordings !== 0) {
    throw new Error('build identity changed or the fixture session did not clean up');
  }
  const sourceSHA256 = Object.fromEntries(await Promise.all(sourceFiles.map(async (path) => [path, await shaFile(path)])));
  const startedAt = idleSamples[0].observedAt;
  const endedAt = idleSamples.at(-1).observedAt;
  const durationMs = Date.parse(endedAt) - Date.parse(startedAt);
  const passed = idleSamples.length === samplesWanted && durationMs >= 60000 && maxCombinedPssKiB <= maxIdlePssKiB &&
    averageCPUPercent < maxIdleCPUPercent && p95CPUPercent < maxIdleCPUPercent && fixtureAndShellPssKiB <= maxFixturePssKiB;
  measurement = {
    schemaVersion: 1,
    contractRow: 'resource-budget',
    result: passed ? 'passed' : 'failed',
    managedBuildIdentityBefore: buildBefore,
    managedBuildIdentityAfter: afterHealth.build_identity,
    platform: { os: process.platform, arch: process.arch, windowsPrivateMemory: { status: 'not_measured', reason: 'This owner uses Linux /proc; run the Windows private-memory comparison on a Windows BAS candidate.' } },
    processes: pids,
    idleSampling: { startedAt, endedAt, durationMs, sampleIntervalMs: intervalMs, sampleCount: idleSamples.length, sessionsThroughout: 0, recordingsThroughout: 0, maxCombinedPssKiB, maxAllowedCombinedPssKiB: maxIdlePssKiB, averageCombinedCPUPercent: averageCPUPercent, p95CombinedCPUPercent: p95CPUPercent, maxCombinedCPUPercent: Math.max(...cpu), samples: idleSamples },
    fixtureBrowserAndShell: { url: fixture.url, browserProcessCount: fixture.browserPids.length, browserPids: fixture.browserPids, browserPssKiB: fixture.browserPssKiB, shellPid: fixture.fixtureShellPid, shellPssKiB: fixture.fixtureShellPssKiB, combinedPssKiB: fixtureAndShellPssKiB, maxAllowedCombinedPssKiB: maxFixturePssKiB },
    cleanup: { fixtureSessionClosed: finalHealth.sessions === 0 && finalHealth.active_recordings === 0, apiHealthy: afterHealth.status === 'healthy' },
    source_sha256: sourceSHA256,
  };
  observations.push({ capturedAt: new Date().toISOString(), outcome: measurement.result, maxCombinedPssKiB, averageCPUPercent, p95CPUPercent, fixtureAndShellPssKiB, buildBefore });
} catch (error) {
  failure = error instanceof Error ? error : new Error(String(error));
} finally {
  try { await cleanup(); } catch (error) { failure ||= error instanceof Error ? error : new Error(String(error)); }
}
if (failure) {
  console.error(failure.message);
  process.exitCode = 1;
} else {
  await mkdir(dirname(rawPath), { recursive: true });
  await writeFile(rawPath, JSON.stringify(measurement, null, 2) + '\n');
  const ownerArtifact = { path: rawPath.slice(scenarioRoot.length + 1), sha256: sha256(await readFile(rawPath)) };
  measurement.owner_artifact = ownerArtifact;
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, JSON.stringify(measurement, null, 2) + '\n');
  console.log(JSON.stringify({ receipt: outputPath, rawArtifact: ownerArtifact, buildIdentity: buildBefore, result: measurement.result, idlePssKiB: measurement.idleSampling.maxCombinedPssKiB, averageCPUPercent: measurement.idleSampling.averageCombinedCPUPercent, p95CPUPercent: measurement.idleSampling.p95CombinedCPUPercent, fixtureAndShellPssKiB: measurement.fixtureBrowserAndShell.combinedPssKiB, windowsPrivateMemory: measurement.platform.windowsPrivateMemory.status }));
  if (measurement.result !== 'passed') process.exitCode = 1;
}
