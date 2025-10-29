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

const exportsObj = {};
const moduleObj = { exports: exportsObj };
const compiled = new Function('exports', 'require', 'module', '__filename', '__dirname', transpiled.outputText);
compiled(exportsObj, require, moduleObj, sourcePath, path.dirname(sourcePath));

const { processExecutionEvent } = moduleObj.exports;

const noop = () => {};

test('processExecutionEvent logs assertion details', () => {
  const logs = [];

  const handlers = {
    updateExecutionStatus: noop,
    updateProgress: noop,
    addLog: (entry) => {
      logs.push(entry);
    },
    addScreenshot: noop,
    recordHeartbeat: noop,
  };

  processExecutionEvent(
    handlers,
    {
      type: 'step.completed',
      execution_id: 'exec-1',
      workflow_id: 'wf-1',
      step_type: 'assert',
      payload: {
        assertion: {
          selector: '#status',
          mode: 'exists',
          success: false,
          message: 'Selector not found',
        },
      },
    },
    { fallbackTimestamp: new Date().toISOString() },
  );

  assert.equal(logs.length, 2);
  assert.match(logs[0].message, /completed/);
  assert.match(logs[1].message, /Assertion #status failed: Selector not found/);
  assert.equal(logs[1].level, 'error');
});

test('processExecutionEvent appends retry information to step logs', () => {
  const logs = [];

  const handlers = {
    updateExecutionStatus: noop,
    updateProgress: noop,
    addLog: (entry) => logs.push(entry),
    addScreenshot: noop,
    recordHeartbeat: noop,
  };

  processExecutionEvent(
    handlers,
    {
      type: 'step.failed',
      execution_id: 'exec-1',
      workflow_id: 'wf-1',
      step_type: 'click',
      payload: {
        retry_attempt: 3,
        retry_max_attempts: 5,
      },
    },
    { fallbackTimestamp: new Date().toISOString() },
  );

  assert.equal(logs.length, 1);
  assert.match(logs[0].message, /attempt 3\/5/);
  assert.equal(logs[0].level, 'error');
});

test('processExecutionEvent notes DOM snapshots when present', () => {
  const logs = [];

  const handlers = {
    updateExecutionStatus: noop,
    updateProgress: noop,
    addLog: (entry) => logs.push(entry),
    addScreenshot: noop,
    recordHeartbeat: noop,
  };

  processExecutionEvent(
    handlers,
    {
      type: 'step.completed',
      execution_id: 'exec-2',
      workflow_id: 'wf-2',
      step_type: 'screenshot',
      step_node_id: 'node-xyz',
      payload: {
        dom_snapshot_preview: '<html>Preview</html>',
      },
    },
    { fallbackTimestamp: new Date().toISOString() },
  );

  assert.equal(logs.length, 2);
  assert.match(logs[1].message, /DOM snapshot captured/);
});
