#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { open, readFile, realpath, stat } from 'node:fs/promises';
import { dirname, isAbsolute, join, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';
import { once } from 'node:events';

const defaultScenarioRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../..');
const outputMount = '/dev/shm/tg-output';
const maximumOutputBytes = 16 * 1024 * 1024;
const maximumLogBytes = maximumOutputBytes - 64 * 1024;
const commandTimeoutMs = 8 * 60 * 1000;
const portTimeoutMs = 10 * 1000;
const healthTimeoutMs = 5 * 1000;

export const sourceFiles = [
  'api/automation/execution-writer/file_writer.go',
  'api/automation/execution-writer/file_writer_test.go',
  'api/automation/execution-writer/external_artifacts.go',
  'api/automation/execution-writer/external_artifacts_test.go',
  'api/services/evidence/policy.go',
  'api/services/evidence/har.go',
  'api/services/retention/retention.go',
  'api/services/retention/activity.go',
  'api/services/retention/fs.go',
  'api/services/retention/retention_test.go',
  'api/go.mod',
  'api/go.sum',
  'api/cmd/evidence-completeness-cohort/qualification.mjs',
  '.vrooli/test-genie.json',
  'docs/internal/TESTING.md',
];

const suites = [
  {
    tests: [
      'TestScreenshotAndOutcomeWriteFailuresBothSurvive',
      'TestInlineTelemetryRemainsAttributableWhenSnapshotStorageFails',
      'TestExternalArtifactsRejectMissingOrUncommittedEvidence',
    ],
    packages: ['./automation/execution-writer'],
  },
  {
    tests: ['TestActiveEvidenceRefusesDeletionUntilExportFinishes'],
    packages: ['./services/retention'],
  },
];
export const expectedTests = suites.flatMap((suite) => suite.tests);

function assertOutputMount(value, expectedMount = outputMount) {
  if (value !== expectedMount) {
    throw new Error(`expected Test Genie private output mount ${expectedMount}`);
  }
  return value;
}

async function hashSources(scenarioRoot) {
  const entries = [];
  for (const file of sourceFiles) {
    const absolute = resolve(scenarioRoot, file);
    const actual = await realpath(absolute);
    const rel = relative(scenarioRoot, actual);
    if (!rel || rel === '..' || rel.startsWith(`..${sep}`) || isAbsolute(rel)) {
      throw new Error(`source path escapes BAS scenario: ${file}`);
    }
    const info = await stat(actual);
    if (!info.isFile()) throw new Error(`source input is not a regular file: ${file}`);
    const content = await readFile(actual);
    const digest = createHash('sha256').update(content).digest('hex');
    entries.push({ path: file, sizeBytes: info.size, sha256: digest });
  }
  entries.sort((a, b) => a.path.localeCompare(b.path));
  const map = createHash('sha256');
  for (const entry of entries) map.update(`${entry.path}\0${entry.sizeBytes}\0${entry.sha256}\n`);
  return { digest: `sha256:${map.digest('hex')}`, files: entries };
}

function resolvePort(scenarioRoot, { spawnSyncCommand = spawnSync, env = process.env } = {}) {
  const result = spawnSyncCommand('vrooli', ['scenario', 'port', 'browser-automation-studio', 'API_PORT', '--json'], {
    cwd: scenarioRoot,
    env,
    encoding: 'utf8',
    timeout: portTimeoutMs,
    maxBuffer: 64 * 1024,
  });
  if (result.error) throw new Error(`scenario port lookup failed: ${result.error.message}`);
  if (result.status !== 0) {
    throw new Error(`scenario port lookup failed (${result.status}): ${(result.stderr || '').trim()}`);
  }
  let portResult;
  try {
    portResult = JSON.parse(result.stdout);
  } catch (error) {
    throw new Error(`scenario port lookup returned invalid JSON: ${error.message}`);
  }
  if (portResult.success !== true || portResult.scenario !== 'browser-automation-studio' ||
      portResult.portName !== 'API_PORT' || !Number.isInteger(portResult.port) ||
      portResult.port < 1 || portResult.port > 65535) {
    throw new Error(`scenario port lookup did not resolve BAS API_PORT: ${JSON.stringify(portResult)}`);
  }
  return portResult.port;
}

async function liveBuildIdentity(port, { fetcher = fetch } = {}) {
  const response = await fetcher(`http://127.0.0.1:${port}/health`, {
    signal: AbortSignal.timeout(healthTimeoutMs),
  });
  if (!response.ok) throw new Error(`BAS health returned HTTP ${response.status}`);
  const health = await response.json();
  const identity = String(health.build_identity || '').trim();
  if (!identity) throw new Error('BAS health returned no build_identity');
  return { identity, status: String(health.status || '') };
}

function parseEventLine(line, passed, failed, skipped) {
  if (!line) return;
  let event;
  try {
    event = JSON.parse(line);
  } catch (error) {
    throw new Error(`go test emitted an invalid JSON event: ${error.message}`);
  }
  if (!expectedTests.includes(event.Test)) return;
  if (event.Action === 'pass') passed.add(event.Test);
  if (event.Action === 'fail') failed.add(event.Test);
  if (event.Action === 'skip') skipped.add(event.Test);
}

function signalProcessGroup(child, signal) {
  if (!child.pid) return;
  try {
    process.kill(-child.pid, signal);
  } catch (error) {
    if (error.code !== 'ESRCH') throw error;
  }
}

async function runTests(outputDir, scenarioRoot, {
  spawnCommand = spawn,
  env = process.env,
  stopProcessGroup = signalProcessGroup,
} = {}) {
  const stdoutPath = join(outputDir, 'owner-tests.stdout.jsonl');
  const stderrPath = join(outputDir, 'owner-tests.stderr.log');
  const stdoutFile = await open(stdoutPath, 'wx', 0o600);
  const stderrFile = await open(stderrPath, 'wx', 0o600);
  const passed = new Set();
  const failed = new Set();
  const skipped = new Set();
  const packageExitCodes = [];
  let totalBytes = 0;
  let stdoutCarry = '';
  let outputError = null;
  let stdoutPosition = 0;
  let stderrPosition = 0;
  let stdoutQueue = Promise.resolve();
  let stderrQueue = Promise.resolve();
  const deadline = Date.now() + commandTimeoutMs;

  try {
    for (const suite of suites) {
      const selector = `^(${suite.tests.join('|')})$`;
      const remainingMs = deadline - Date.now();
      if (remainingMs <= 0) throw new Error(`owner assertions exceeded ${commandTimeoutMs}ms`);
      const child = spawnCommand('go', ['test', '-json', ...suite.packages, '-run', selector, '-count=1'], {
        cwd: join(scenarioRoot, 'api'),
        env,
        detached: true,
        stdio: ['ignore', 'pipe', 'pipe'],
      });
      let spawnError = null;
      child.on('error', (error) => { spawnError = error; });
      let timedOut = false;
      const timeoutTimer = setTimeout(() => {
        timedOut = true;
        stopProcessGroup(child, 'SIGTERM');
        setTimeout(() => stopProcessGroup(child, 'SIGKILL'), 3000).unref();
      }, remainingMs);

      const consume = async (chunk, streamName) => {
        if (outputError) return;
        totalBytes += chunk.length;
        if (totalBytes > maximumLogBytes) {
          outputError = new Error(`combined go test logs exceeded ${maximumLogBytes} bytes`);
          stopProcessGroup(child, 'SIGTERM');
          setTimeout(() => stopProcessGroup(child, 'SIGKILL'), 3000).unref();
          return;
        }
        const destination = streamName === 'stdout' ? stdoutFile : stderrFile;
        let written = 0;
        const position = streamName === 'stdout' ? stdoutPosition : stderrPosition;
        while (written < chunk.length) {
          const result = await destination.write(chunk, written, chunk.length - written, position + written);
          if (result.bytesWritten <= 0) throw new Error(`could not write complete ${streamName} test output`);
          written += result.bytesWritten;
        }
        if (streamName === 'stdout') stdoutPosition += written;
        else stderrPosition += written;
        if (streamName === 'stdout') {
          stdoutCarry += chunk.toString('utf8');
          const lines = stdoutCarry.split(/\r?\n/);
          stdoutCarry = lines.pop() || '';
          for (const line of lines) parseEventLine(line, passed, failed, skipped);
        }
      };
      child.stdout.on('data', (chunk) => {
        stdoutQueue = stdoutQueue.then(() => consume(chunk, 'stdout'));
        stdoutQueue.catch((error) => { outputError = error; stopProcessGroup(child, 'SIGTERM'); });
      });
      child.stderr.on('data', (chunk) => {
        stderrQueue = stderrQueue.then(() => consume(chunk, 'stderr'));
        stderrQueue.catch((error) => { outputError = error; stopProcessGroup(child, 'SIGTERM'); });
      });
      const [exitCode, signal] = await once(child, 'close');
      await Promise.all([stdoutQueue, stderrQueue]);
      clearTimeout(timeoutTimer);
      if (stdoutCarry) {
        parseEventLine(stdoutCarry, passed, failed, skipped);
        stdoutCarry = '';
      }
      if (outputError) throw outputError;
      if (spawnError) throw new Error(`could not start go test: ${spawnError.message}`);
      if (timedOut) throw new Error(`go test exceeded ${commandTimeoutMs}ms`);
      if (signal || exitCode !== 0) throw new Error(`go test failed (${signal || exitCode})`);
      packageExitCodes.push({ packages: suite.packages, exitCode });
    }
  } finally {
    await stdoutFile.close();
    await stderrFile.close();
  }

  for (const name of expectedTests) {
    if (!passed.has(name) || failed.has(name) || skipped.has(name)) {
      throw new Error(`owner assertion did not pass exactly as selected: ${name}`);
    }
  }
  return {
    stdoutPath,
    stderrPath,
    stdoutSha256: await hashFile(stdoutPath),
    stderrSha256: await hashFile(stderrPath),
    outputBytes: totalBytes,
    packageExitCodes,
    tests: expectedTests.map((name) => ({ name, result: 'passed' })),
  };
}

async function hashFile(path) {
  const { createReadStream } = await import('node:fs');
  const digest = createHash('sha256');
  for await (const chunk of createReadStream(path)) digest.update(chunk);
  return digest.digest('hex');
}

// The optional dependencies and roots keep this exact production runner
// testable with disposable fixtures. The CLI entrypoint below supplies only
// the fixed Test Genie output mount and real process/network bindings.
export async function runQualification({
  outputDir,
  scenarioRoot = defaultScenarioRoot,
  outputMountPath = outputMount,
  dependencies = {},
} = {}) {
  const {
    spawnCommand = spawn,
    spawnSyncCommand = spawnSync,
    fetcher = fetch,
    env = process.env,
    stopProcessGroup = signalProcessGroup,
  } = dependencies;
  assertOutputMount(outputDir, outputMountPath);
  const reportPath = join(outputDir, 'owner-qualification.json');
  const report = {
    schemaVersion: 1,
    contractRow: 'evidence-completeness',
    result: 'failed',
    sourceMap: null,
    managedBuild: { before: null, after: null },
    testOutput: null,
    error: null,
  };
  let failure;
  try {
    const sourceBefore = await hashSources(scenarioRoot);
    report.sourceMap = sourceBefore;
    const portBefore = resolvePort(scenarioRoot, { spawnSyncCommand, env });
    const buildBefore = await liveBuildIdentity(portBefore, { fetcher });
    report.managedBuild.before = { port: portBefore, ...buildBefore };

    report.testOutput = await runTests(outputDir, scenarioRoot, { spawnCommand, env, stopProcessGroup });

    const portAfter = resolvePort(scenarioRoot, { spawnSyncCommand, env });
    const buildAfter = await liveBuildIdentity(portAfter, { fetcher });
    report.managedBuild.after = { port: portAfter, ...buildAfter };
    const sourceAfter = await hashSources(scenarioRoot);
    if (JSON.stringify(sourceAfter) !== JSON.stringify(sourceBefore)) {
      throw new Error('declared source map changed while owner assertions ran');
    }
    if (portAfter !== portBefore || buildAfter.identity !== buildBefore.identity) {
      throw new Error(`managed BAS build identity changed during owner assertions: ${buildBefore.identity} -> ${buildAfter.identity}`);
    }
    report.result = 'passed';
  } catch (error) {
    failure = error;
    report.error = String(error?.stack || error);
  }

  const reportFile = await open(reportPath, 'wx', 0o600);
  try {
    await reportFile.writeFile(`${JSON.stringify(report, null, 2)}\n`);
  } finally {
    await reportFile.close();
  }

  return { report, failure };
}

async function main() {
  const outputDir = process.argv[2];
  const result = await runQualification({ outputDir });
  const { report, failure } = result;
  if (failure) {
    process.stderr.write(`${failure.message || failure}\n`);
    process.exitCode = 1;
    return;
  }
  process.stdout.write(JSON.stringify({
    result: report.result,
    sourceMapDigest: report.sourceMap.digest,
    managedBuildIdentity: report.managedBuild.before.identity,
    ownerAssertions: report.testOutput.tests.length,
    outputBytes: report.testOutput.outputBytes,
    report: join(outputDir, 'owner-qualification.json'),
  }) + '\n');
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main().catch((error) => {
    process.stderr.write(`${error?.stack || error}\n`);
    process.exitCode = 1;
  });
}
