import { test } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createRequire } from 'node:module';
import ts from 'typescript';

const require = createRequire(import.meta.url);
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const sourcePath = path.resolve(__dirname, '../src/stores/executionEventProcessor.ts');
const sourceCode = fs.readFileSync(sourcePath, 'utf8');

if (typeof globalThis.crypto?.randomUUID !== 'function') {
  globalThis.crypto = {
    randomUUID: () => 'test-id',
  };
}

const transpiled = ts.transpileModule(sourceCode, {
  compilerOptions: {
    module: ts.ModuleKind.CommonJS,
    target: ts.ScriptTarget.ES2020,
    esModuleInterop: true,
  },
});

const moduleExports = {};
const moduleObj = { exports: moduleExports };
const compiled = new Function('exports', 'require', 'module', '__filename', '__dirname', transpiled.outputText);
compiled(moduleExports, require, moduleObj, sourcePath, path.dirname(sourcePath));

const { processExecutionEvent } = moduleObj.exports;

const noop = () => {};

test('processExecutionEvent records heartbeat and progress updates', () => {
  const recorded = [];
  const progress = [];

  const handlers = {
    updateExecutionStatus: noop,
    updateProgress: (value, label) => {
      progress.push({ value, label });
    },
    addLog: noop,
    addScreenshot: noop,
    recordHeartbeat: (step, elapsedMs) => {
      recorded.push({ step, elapsedMs });
    },
  };

  processExecutionEvent(
    handlers,
    {
      type: 'step.heartbeat',
      execution_id: 'exec-1',
      workflow_id: 'wf-1',
      step_type: 'click',
      payload: { elapsed_ms: 900 },
    },
    {
      fallbackProgress: 62,
      fallbackTimestamp: new Date().toISOString(),
    },
  );

  assert.equal(progress.length, 1);
  assert.equal(progress[0].value, 62);
  assert.equal(progress[0].label, 'click');

  assert.equal(recorded.length, 1);
  assert.equal(recorded[0].step, 'click');
  assert.equal(recorded[0].elapsedMs, 900);
});
