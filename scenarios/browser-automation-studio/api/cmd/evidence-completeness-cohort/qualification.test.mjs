import assert from 'node:assert/strict';
import { EventEmitter } from 'node:events';
import { PassThrough } from 'node:stream';
import { mkdtemp, mkdir, readFile, rm, stat, writeFile } from 'node:fs/promises';
import { writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

import { expectedTests, runQualification, sourceFiles } from './qualification.mjs';

const scriptPath = fileURLToPath(new URL('./qualification.mjs', import.meta.url));
const executionWriterTests = expectedTests.slice(0, 3);
const retentionTests = expectedTests.slice(3);

class FakeChild extends EventEmitter {
  constructor(options = {}) {
    super();
    const { stdout = '', stderr = '', exitCode = 0, signal = null, autoClose = true } =
      typeof options === 'string' ? { stdout: options } : options;
    this.stdout = new PassThrough();
    this.stderr = new PassThrough();
    this.pid = 4242;
    this.closed = false;
    this.exitCode = exitCode;
    this.signal = signal;
    this.output = stdout;
    this.errorOutput = stderr;
    setImmediate(() => {
      if (autoClose) this.closeNormally();
      else {
        this.stdout.end(this.output);
        this.stderr.end(this.errorOutput);
      }
    });
  }

  closeNormally() {
    if (this.closed) return;
    this.stdout.end(this.output);
    this.stderr.end(this.errorOutput);
    this.closed = true;
    this.emit('close', this.exitCode, this.signal);
  }

  stop(signal) {
    this.stopSignal = signal;
    if (this.closed) return;
    this.stdout.end();
    this.stderr.end();
    this.closed = true;
    setImmediate(() => this.emit('close', null, signal));
  }
}

async function fixture(t) {
  const root = await mkdtemp(join(tmpdir(), 'bas-evidence-completeness-'));
  const outputDir = join(root, 'private-output');
  await mkdir(outputDir);
  for (const source of sourceFiles) {
    const path = join(root, source);
    await mkdir(dirname(path), { recursive: true });
    await writeFile(path, source === 'api/cmd/evidence-completeness-cohort/qualification.mjs'
      ? await readFile(scriptPath)
      : `fixture source: ${source}\n`);
  }
  t.after(async () => rm(root, { recursive: true, force: true }));
  return { root, outputDir };
}

function eventLines(tests, action = 'pass') {
  return tests.map((name) => JSON.stringify({ Test: name, Action: action })).join('\n') + '\n';
}

function dependencies({
  testRuns = [],
  identities = ['managed-build-a', 'managed-build-a'],
  onTestStart,
} = {}) {
  const observed = { portCalls: [], fetchCalls: [], goCalls: [], children: [], stopCalls: [] };
  return {
    observed,
    dependencies: {
      env: { PATH: '/fixture-only' },
      spawnSyncCommand(command, args, options) {
        observed.portCalls.push({ command, args, options });
        return {
          status: 0,
          stdout: JSON.stringify({
            success: true,
            scenario: 'browser-automation-studio',
            portName: 'API_PORT',
            port: 43123,
          }),
          stderr: '',
        };
      },
      async fetcher(url, options) {
        observed.fetchCalls.push({ url, options });
        const identity = identities[Math.min(observed.fetchCalls.length - 1, identities.length - 1)];
        return { ok: true, async json() { return { status: 'healthy', build_identity: identity }; } };
      },
      spawnCommand(command, args, options) {
        observed.goCalls.push({ command, args, options });
        onTestStart?.(command, args, options);
        const output = testRuns.length ? testRuns.shift() : '';
        const child = new FakeChild(output);
        observed.children.push(child);
        return child;
      },
      stopProcessGroup(child, signal) {
        observed.stopCalls.push(signal);
        child.stop(signal);
      },
    },
  };
}

async function run(fixtureValue, injected, overrides = {}) {
  return runQualification({
    outputDir: fixtureValue.outputDir,
    scenarioRoot: fixtureValue.root,
    outputMountPath: fixtureValue.outputDir,
    dependencies: injected.dependencies,
    ...overrides,
  });
}

test('qualification records exactly the four passing owner assertions with stable source and build identities', async (t) => {
  const paths = await fixture(t);
  const injected = dependencies({
    testRuns: [eventLines(executionWriterTests), eventLines(retentionTests)],
  });

  const { report, failure } = await run(paths, injected);

  assert.equal(failure, undefined);
  assert.equal(report.result, 'passed');
  assert.deepEqual(report.testOutput.tests.map(({ name }) => name), expectedTests);
  assert.equal(report.testOutput.tests.length, 4);
  assert.equal(report.sourceMap.files.length, sourceFiles.length);
  assert.equal(report.managedBuild.before.identity, 'managed-build-a');
  assert.equal(report.managedBuild.after.identity, 'managed-build-a');
  assert.equal(injected.observed.goCalls.length, 2);
  assert.deepEqual(injected.observed.goCalls.map(({ command, args }) => [command, ...args]), [
    ['go', 'test', '-json', './automation/execution-writer', '-run', `^(${executionWriterTests.join('|')})$`, '-count=1'],
    ['go', 'test', '-json', './services/retention', '-run', `^(${retentionTests.join('|')})$`, '-count=1'],
  ]);
  assert.deepEqual(injected.observed.portCalls.map(({ command, args }) => [command, ...args]), [
    ['vrooli', 'scenario', 'port', 'browser-automation-studio', 'API_PORT', '--json'],
    ['vrooli', 'scenario', 'port', 'browser-automation-studio', 'API_PORT', '--json'],
  ]);
  assert.deepEqual(injected.observed.fetchCalls.map(({ url }) => url), [
    'http://127.0.0.1:43123/health',
    'http://127.0.0.1:43123/health',
  ]);
  assert.ok((await stat(join(paths.outputDir, 'owner-qualification.json'))).size > 0);
});

for (const defect of [
  { name: 'missing assertion', output: eventLines(expectedTests.slice(0, 2), 'pass'), error: /owner assertion did not pass exactly as selected/ },
  { name: 'skipped assertion', output: eventLines(expectedTests, 'pass').replace(`"Test":"${expectedTests[0]}","Action":"pass"`, `"Test":"${expectedTests[0]}","Action":"skip"`), error: /owner assertion did not pass exactly as selected/ },
  { name: 'failed test process', output: eventLines(expectedTests.slice(0, 3), 'pass'), exitCode: 1, error: /go test failed/ },
]) {
  test(`qualification refuses ${defect.name}`, async (t) => {
    const paths = await fixture(t);
    const injected = dependencies({ testRuns: [defect.output, eventLines(retentionTests)] });
    if (defect.exitCode !== undefined) {
      injected.dependencies.spawnCommand = (command, args, options) => {
        const child = new FakeChild({ stdout: defect.output, exitCode: defect.exitCode });
        injected.observed.goCalls.push({ command, args, options });
        injected.observed.children.push(child);
        return child;
      };
    }

    const { report, failure } = await run(paths, injected);

    assert.ok(failure, 'defective owner run must fail closed');
    assert.match(report.error, defect.error);
    assert.equal(report.result, 'failed');
  });
}

test('qualification refuses when a declared source changes during owner checks', async (t) => {
  const paths = await fixture(t);
  const changed = join(paths.root, 'api/services/retention/retention.go');
  const injected = dependencies({
    testRuns: [eventLines(executionWriterTests), eventLines(retentionTests)],
    onTestStart: () => writeFileSync(changed, 'changed during qualification\n'),
  });

  const { report, failure } = await run(paths, injected);

  assert.ok(failure);
  assert.match(report.error, /declared source map changed while owner assertions ran/);
  assert.equal(report.result, 'failed');
});

test('qualification refuses when the managed build identity changes during owner checks', async (t) => {
  const paths = await fixture(t);
  const injected = dependencies({
    testRuns: [eventLines(executionWriterTests), eventLines(retentionTests)],
    identities: ['managed-build-before', 'managed-build-after'],
  });

  const { report, failure } = await run(paths, injected);

  assert.ok(failure);
  assert.match(report.error, /managed BAS build identity changed during owner assertions/);
  assert.equal(report.managedBuild.before.identity, 'managed-build-before');
  assert.equal(report.managedBuild.after.identity, 'managed-build-after');
});

test('qualification stops and drains an over-limit child without writing oversized logs', async (t) => {
  const paths = await fixture(t);
  const oversized = Buffer.alloc(16 * 1024 * 1024, 0x78);
  const injected = dependencies();
  injected.dependencies.spawnCommand = (command, args, options) => {
    injected.observed.goCalls.push({ command, args, options });
    const child = new FakeChild({ stdout: oversized, autoClose: false });
    injected.observed.children.push(child);
    return child;
  };

  const { report, failure } = await run(paths, injected);

  assert.ok(failure);
  assert.match(report.error, /combined go test logs exceeded/);
  assert.ok(injected.observed.stopCalls.includes('SIGTERM'));
  assert.ok(injected.observed.children[0].closed, 'child must be drained before the run returns');
  const stdout = await stat(join(paths.outputDir, 'owner-tests.stdout.jsonl'));
  const stderr = await stat(join(paths.outputDir, 'owner-tests.stderr.log'));
  assert.ok(stdout.size + stderr.size <= (16 * 1024 * 1024) - (64 * 1024));
});

test('qualification keeps the fixed Test Genie mount as the production default', async (t) => {
  const paths = await fixture(t);
  const injected = dependencies();

  await assert.rejects(
    runQualification({
      outputDir: paths.outputDir,
      scenarioRoot: paths.root,
      dependencies: injected.dependencies,
    }),
    /expected Test Genie private output mount \/dev\/shm\/tg-output/,
  );
  assert.equal(injected.observed.portCalls.length, 0, 'refusal must happen before external work');
});
