#!/usr/bin/env node
import { execFile } from 'node:child_process';
import { createHash, randomUUID } from 'node:crypto';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join, relative, resolve } from 'node:path';
import { promisify } from 'node:util';
import { fileURLToPath } from 'node:url';
import { validateObservation } from './validation.mjs';

const execFileAsync = promisify(execFile);
const scenarioRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const uiRoot = join(scenarioRoot, 'ui');
const driverRoot = join(scenarioRoot, 'playwright-driver');
const evidenceRoot = join(scenarioRoot, '.vrooli/runtime/rehabilitation-evidence');
const api = process.env.BAS_REHAB_LIVE_API_BASE || 'http://127.0.0.1:17116/api/v1';
const ui = process.env.BAS_REHAB_LIVE_UI_BASE || 'http://127.0.0.1:21794';
const apiURL = new URL(api);
const uiURL = new URL(ui);
if (apiURL.protocol !== 'http:' || !['127.0.0.1', 'localhost', '[::1]'].includes(apiURL.hostname)
    || uiURL.protocol !== 'http:' || !['127.0.0.1', 'localhost', '[::1]'].includes(uiURL.hostname)) {
  throw new Error('Motion qualification only accepts loopback BAS API and UI URLs');
}

const sourceFiles = [
  'docs/internal/REFRACTOR_CONTRACT.json',
  '.vrooli/test-genie.json',
  '.vrooli/program-runtime/setpoint-read.py',
  'api/automation/driver/types.go',
  'api/automation/driver/client_payload_test.go',
  'api/handlers/record_mode_types.go',
  'api/handlers/record_mode_test.go',
  'api/handlers/testutil_mock_services.go',
  'api/internal/motionqualification/receipt.go',
  'api/internal/motionqualification/receipt_test.go',
  'api/internal/testutil/testutil.go',
  'api/handlers/profilevalidation/provider.go',
  'api/handlers/profilevalidation/provider_test.go',
  'api/config/config.go',
  'api/websocket/hub.go',
  'api/websocket/hub_test.go',
  'api/cmd/motion-cohort/qualification.mjs',
  'api/cmd/motion-cohort/validation.mjs',
  'playwright-driver/src/frame-streaming/strategies/cdp-screencast.ts',
  'playwright-driver/tests/integration/motion-qualification.test.ts',
  'playwright-driver/tests/unit/frame-streaming/cdp-screencast-strategy.test.ts',
  'ui/src/domains/recording/capture/useFrameStream.ts',
  'ui/src/domains/recording/capture/useFrameStream.test.ts',
];

async function sha256(path) {
  return createHash('sha256').update(await readFile(path)).digest('hex');
}

async function buildIdentity() {
  const healthURL = new URL(api);
  healthURL.pathname = `${healthURL.pathname.replace(/\/api\/v1\/?$/, '').replace(/\/$/, '')}/health`;
  const response = await fetch(healthURL, { signal: AbortSignal.timeout(5000) });
  if (!response.ok) throw new Error(`BAS health returned ${response.status}`);
  const health = await response.json();
  const identity = String(health.build_identity || '').trim();
  if (!identity) throw new Error('BAS health returned no build_identity');
  return identity;
}

async function writeReceipt(observation, observationPath, combinedLogPath, build, receiptPath) {
  const sourceSHA256 = Object.fromEntries(await Promise.all(sourceFiles.map(async (path) => [path, await sha256(join(scenarioRoot, path))])));
  const artifact = async (path) => ({
    path: relative(scenarioRoot, path).replaceAll('\\', '/'),
    sha256: await sha256(path),
  });
  const receipt = {
    schemaVersion: 1,
    contractRow: 'motion',
    result: 'passed',
    managedBuildIdentity: build,
    contractSha256: sourceSHA256['docs/internal/REFRACTOR_CONTRACT.json'],
    sourceSha256: sourceSHA256,
    baseline: { ...observation.baseline, durationMs: Math.round(observation.baseline.durationMs) },
    slowReader: { ...observation.slowReader, durationMs: Math.round(observation.slowReader.durationMs) },
    artifacts: [await artifact(observationPath), await artifact(combinedLogPath)],
  };
  await writeFile(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, { mode: 0o600, flag: 'wx' });
  return receipt;
}

function summarizeCohort(cohort) {
  const summary = { ...cohort };
  delete summary.samples;
  return summary;
}

function summarizeReceipt(receiptPath, receipt) {
  return {
    result: receipt.result,
    buildIdentity: receipt.managedBuildIdentity,
    receipt: relative(scenarioRoot, receiptPath),
    baseline: summarizeCohort(receipt.baseline),
    slowReader: summarizeCohort(receipt.slowReader),
  };
}

if (process.argv[2] === 'assemble-existing') {
  const observationPath = resolve(scenarioRoot, process.argv[3] || '');
  const combinedLogPath = resolve(scenarioRoot, process.argv[4] || '');
  if (!process.argv[3] || !process.argv[4]) {
    throw new Error('usage: qualification.mjs assemble-existing <observation.json> <combined-owner.log>');
  }
  const observation = JSON.parse(await readFile(observationPath, 'utf8'));
  const combinedLog = await readFile(combinedLogPath, 'utf8');
  const relayMetric = combinedLog.match(/BAS_MOTION_RELAY_QUEUE max_queued_bytes=(\d+) cap_bytes=(\d+) dropped=(\d+)/);
  const focusedTests = [
    'TestBroadcastBinaryFrameDropsAtByteBudget',
    'TestUpdateStreamSettingsPreservesFractionalCurrentFPS',
    'TestUpdateStreamSettings_Success',
  ];
  const relayTestsPassed = focusedTests.every((name) =>
    new RegExp(`"Action":"pass".*"Test":"${name}"`).test(combinedLog)
  );
  if (
    !/Test Files\s+1 passed/.test(combinedLog) ||
    !/Tests\s+\d+ passed/.test(combinedLog) ||
    /Tests\s+\d+ failed/.test(combinedLog) ||
    !relayTestsPassed ||
    !relayMetric ||
    !/PASS tests\/unit\/frame-streaming\/cdp-screencast-strategy\.test\.ts/.test(combinedLog) ||
    !/Tests:\s+\d+ passed,\s+\d+ total/.test(combinedLog) ||
    !/PASS tests\/integration\/motion-qualification\.test\.ts/.test(combinedLog) ||
    !/Test Suites:\s+1 passed,\s+1 total/.test(combinedLog) ||
    !/Tests:\s+1 passed,\s+1 total/.test(combinedLog)
  ) {
    throw new Error('retained motion owner logs do not prove every focused check passed');
  }
  if (Number(relayMetric[1]) <= 0 || Number(relayMetric[1]) > Number(relayMetric[2])) {
    throw new Error('retained relay log omitted or exceeded its queued-byte measurement');
  }
  observation.slowReader.maxApiQueueBytes = Number(relayMetric[1]);
  observation.slowReader.apiQueueBudgetBytes = Number(relayMetric[2]);
  validateObservation(observation, 12 * 1024 * 1024 + 4 * 1024);
  const build = await buildIdentity();
  const id = `${new Date().toISOString().replaceAll(':', '-')}-${randomUUID()}`;
  const receiptPath = join(evidenceRoot, `motion-receipt-reassembled-${id}.json`);
  const receipt = await writeReceipt(observation, observationPath, combinedLogPath, build, receiptPath);
  process.stdout.write(`${JSON.stringify(summarizeReceipt(receiptPath, receipt), null, 2)}\n`);
  process.exit(0);
}

await mkdir(evidenceRoot, { recursive: true });
const id = `${new Date().toISOString().replaceAll(':', '-')}-${randomUUID()}`;
const observationPath = join(evidenceRoot, `motion-live-owner-${id}.json`);
const unitLogPath = join(evidenceRoot, `motion-owner-ui-tests-${id}.log`);
const strategyLogPath = join(evidenceRoot, `motion-owner-cdp-strategy-tests-${id}.log`);
const relayLogPath = join(evidenceRoot, `motion-owner-relay-tests-${id}.log`);
const liveLogPath = join(evidenceRoot, `motion-owner-live-test-${id}.log`);
const combinedLogPath = join(evidenceRoot, `motion-owner-tests-${id}.log`);
const receiptPath = join(evidenceRoot, `motion-receipt-${id}.json`);
const buildBefore = await buildIdentity();

let unitRun;
try {
  unitRun = await execFileAsync('pnpm', [
    'exec', 'vitest', 'run', 'src/domains/recording/capture/useFrameStream.test.ts', '--reporter=dot',
  ], { cwd: uiRoot, timeout: 120_000, maxBuffer: 8 * 1024 * 1024, env: process.env });
} catch (error) {
  const output = `${error.stdout || ''}${error.stderr || ''}`;
  await writeFile(unitLogPath, output, { mode: 0o600 });
  process.stderr.write(output);
  throw new Error(`bounded UI frame-decoder owner failed: ${error.message}`);
}
const unitOutput = `${unitRun.stdout || ''}${unitRun.stderr || ''}`;
await writeFile(unitLogPath, unitOutput, { mode: 0o600 });
if (!/Test Files\s+1 passed/.test(unitOutput) || !/Tests\s+\d+ passed/.test(unitOutput) || /Tests\s+\d+ failed/.test(unitOutput)) {
  throw new Error('UI frame-decoder test output did not report a fully passing owner file');
}

let strategyRun;
try {
  strategyRun = await execFileAsync('pnpm', [
    'exec', 'jest', 'tests/unit/frame-streaming/cdp-screencast-strategy.test.ts',
    '--runInBand', '--coverage=false', '--silent=false',
  ], { cwd: driverRoot, timeout: 120_000, maxBuffer: 12 * 1024 * 1024, env: process.env });
} catch (error) {
  const output = `${error.stdout || ''}${error.stderr || ''}`;
  await writeFile(strategyLogPath, output, { mode: 0o600 });
  await writeFile(combinedLogPath, `${unitOutput}\n${output}`, { mode: 0o600 });
  process.stderr.write(output);
  throw new Error(`bounded CDP strategy owner failed: ${error.message}`);
}
const strategyOutput = `${strategyRun.stdout || ''}${strategyRun.stderr || ''}`;
await writeFile(strategyLogPath, strategyOutput, { mode: 0o600 });
if (!/PASS tests\/unit\/frame-streaming\/cdp-screencast-strategy\.test\.ts/.test(strategyOutput)
    || !/Tests:\s+\d+ passed,\s+\d+ total/.test(strategyOutput)
    || /Tests:\s+\d+ failed/.test(strategyOutput)) {
  await writeFile(combinedLogPath, `${unitOutput}\n${strategyOutput}`, { mode: 0o600 });
  throw new Error('CDP strategy unit tests did not report a fully passing owner file');
}

let relayRun;
try {
  relayRun = await execFileAsync('go', [
    'test', '-json', './websocket', './automation/driver', './handlers',
    '-run', '^(TestBroadcastBinaryFrameDropsAtByteBudget|TestUpdateStreamSettingsPreservesFractionalCurrentFPS|TestUpdateStreamSettings_Success)$',
    '-count=1',
  ], { cwd: join(scenarioRoot, 'api'), timeout: 120_000, maxBuffer: 12 * 1024 * 1024,
    env: { ...process.env, GOTOOLCHAIN: 'local', GOPROXY: 'off' } });
} catch (error) {
  const output = `${error.stdout || ''}${error.stderr || ''}`;
  await writeFile(relayLogPath, output, { mode: 0o600 });
  await writeFile(combinedLogPath, `${unitOutput}\n${strategyOutput}\n${output}`, { mode: 0o600 });
  process.stderr.write(output);
  throw new Error(`API relay byte-budget owner failed: ${error.message}`);
}
const relayOutput = `${relayRun.stdout || ''}${relayRun.stderr || ''}`;
await writeFile(relayLogPath, relayOutput, { mode: 0o600 });
const relayEvents = relayOutput.split(/\r?\n/).filter(Boolean).map((line) => JSON.parse(line));
const focusedTests = [
  'TestBroadcastBinaryFrameDropsAtByteBudget',
  'TestUpdateStreamSettingsPreservesFractionalCurrentFPS',
  'TestUpdateStreamSettings_Success',
];
if (focusedTests.some((name) => !relayEvents.some((event) => event.Action === 'pass' && event.Test === name))) {
  await writeFile(combinedLogPath, `${unitOutput}\n${strategyOutput}\n${relayOutput}`, { mode: 0o600 });
  throw new Error(`focused relay/driver/API tests did not all report passing: ${focusedTests.join(', ')}`);
}
const relayMetric = relayEvents.map((event) => event.Output || '').join('').match(/BAS_MOTION_RELAY_QUEUE max_queued_bytes=(\d+) cap_bytes=(\d+) dropped=(\d+)/);
if (!relayMetric || Number(relayMetric[1]) <= 0 || Number(relayMetric[1]) > Number(relayMetric[2])) {
  throw new Error('API relay test omitted or exceeded its queued-byte measurement');
}

let liveRun;
try {
  liveRun = await execFileAsync('pnpm', [
    'exec', 'jest', 'tests/integration/motion-qualification.test.ts', '--runInBand', '--coverage=false', '--silent=false',
  ], {
    cwd: driverRoot,
    timeout: 390_000,
    maxBuffer: 24 * 1024 * 1024,
    env: {
      ...process.env,
      BAS_REHAB_LIVE_API_BASE: api,
      BAS_REHAB_LIVE_UI_BASE: ui,
      BAS_MOTION_OBSERVATION_PATH: observationPath,
      BAS_JEST_VERBOSE_LOGS: '1',
    },
  });
} catch (error) {
  const output = `${error.stdout || ''}${error.stderr || ''}`;
  await writeFile(liveLogPath, output, { mode: 0o600 });
  await writeFile(combinedLogPath, `${unitOutput}\n${strategyOutput}\n${relayOutput}\n${output}`, { mode: 0o600 });
  process.stderr.write(output);
  throw new Error(`managed motion owner failed: ${error.message}`);
}
const liveOutput = `${liveRun.stdout || ''}${liveRun.stderr || ''}`;
await writeFile(liveLogPath, liveOutput, { mode: 0o600 });
await writeFile(combinedLogPath, `${unitOutput}\n${strategyOutput}\n${relayOutput}\n${liveOutput}`, { mode: 0o600 });
const observation = JSON.parse(await readFile(observationPath, 'utf8'));
observation.slowReader.maxApiQueueBytes = Number(relayMetric[1]);
observation.slowReader.apiQueueBudgetBytes = Number(relayMetric[2]);
validateObservation(observation, 12 * 1024 * 1024 + 4 * 1024);

const buildAfter = await buildIdentity();
if (buildAfter !== buildBefore) throw new Error(`managed BAS build changed during motion owner: ${buildBefore} -> ${buildAfter}`);
const receipt = await writeReceipt(observation, observationPath, combinedLogPath, buildBefore, receiptPath);
process.stdout.write(`${JSON.stringify(summarizeReceipt(receiptPath, receipt), null, 2)}\n`);
