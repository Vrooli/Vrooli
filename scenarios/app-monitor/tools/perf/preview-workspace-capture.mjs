#!/usr/bin/env node

import fs from 'node:fs/promises';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { performance } from 'node:perf_hooks';
import { fileURLToPath } from 'node:url';
import { createAppMonitorBASClientFromEnv, APP_MONITOR_URL } from './bas-perf-client.mjs';

const DEFAULT_PANES = 8;
const DEFAULT_OUT = '/tmp/app-monitor/perf/preview-workspace-stable.json';
const TASK_REVISION = 'app-monitor.preview-workspace.performance-capture.v1';
const ORACLE_REVISION = 'app-monitor.preview-workspace.metrics.v1';
const settle = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

export function parseArgs(argv) {
  const args = { panes: DEFAULT_PANES, out: DEFAULT_OUT, trace: null };
  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const [key, inlineValue] = arg.startsWith('--') ? arg.split('=', 2) : [arg, undefined];
    const value = inlineValue ?? argv[index + 1];
    if (inlineValue === undefined && key.startsWith('--')) index += 1;
    if (key === '--panes' && value) args.panes = Math.max(1, Math.min(12, Number.parseInt(value, 10) || DEFAULT_PANES));
    else if (key === '--out' && value) args.out = value;
    else if (key === '--trace' && value) args.trace = value;
    else throw new Error(`unsupported argument: ${key}`);
  }
  return args;
}

function safeSessionId(value) {
  if (!/^[A-Za-z0-9_-]{1,128}$/.test(value ?? '')) throw new Error('BAS returned an invalid session ID');
  return value;
}

function checkBinding(response, { sessionId, captureId, phase, runId } = {}) {
  const binding = response?.binding;
  if (!binding || binding.sessionId !== sessionId || binding.phase !== phase ||
      (captureId && binding.captureId !== captureId) ||
      (runId && binding.runId !== runId) ||
      binding.taskRevision !== TASK_REVISION || binding.oracleRevision !== ORACLE_REVISION ||
      !binding.captureId || !binding.runId || !binding.profileRevision || !binding.observedAt) {
    throw new Error(`BAS returned an invalid ${phase} capture binding`);
  }
  return binding;
}

function appMonitorCounters(metrics, panesRequested) {
  if (!metrics || metrics.contract !== ORACLE_REVISION ||
      !Number.isSafeInteger(metrics.iframeCount) || metrics.iframeCount < 0 ||
      !Number.isSafeInteger(metrics.paneCount) || metrics.paneCount !== panesRequested ||
      !Number.isSafeInteger(metrics.tabCardCount) || metrics.tabCardCount < 0 ||
      !Number.isSafeInteger(metrics.profilerMarkCount) || metrics.profilerMarkCount < 0 ||
      !Number.isSafeInteger(metrics.workspaceStorageBytes) || metrics.workspaceStorageBytes < 0) {
    throw new Error('App Monitor returned incomplete or invalid workspace metrics');
  }
  return {
    iframes: metrics.iframeCount,
    panes: metrics.paneCount,
    tabCards: metrics.tabCardCount,
    profilerMarks: metrics.profilerMarkCount,
    storageBytes: metrics.workspaceStorageBytes,
  };
}

async function addConfiguredPanes(client, sessionId, paneCount) {
  if (paneCount > 0) {
    await client.task(sessionId, 'navigate', {
      url: `${APP_MONITOR_URL}?workspaceAppId=scenario-1&workspaceMode=replace-focused`,
    });
  }
  for (let index = 1; index < paneCount; index += 1) {
    const appId = index % 2 === 0 ? 'scenario-1' : 'scenario-2';
    await client.task(sessionId, 'navigate', {
      url: `${APP_MONITOR_URL}?workspaceAppId=${appId}&workspaceMode=add-pane`,
    });
  }
}

export async function runCapture({ argv = [], client = createAppMonitorBASClientFromEnv(), fsImpl = fs, log = console.log } = {}) {
  const args = parseArgs(argv);
  await fsImpl.mkdir(path.dirname(args.out), { recursive: true });
  if (args.trace) await fsImpl.mkdir(path.dirname(args.trace), { recursive: true });

  let sessionId = '';
  let captureId = '';
  let began = false;
  let closed = false;
  try {
    const opened = await client.openSession({ viewportWidth: 1440, viewportHeight: 1000 });
    sessionId = safeSessionId(opened.sessionId);
    const started = performance.now();

    await addConfiguredPanes(client, sessionId, args.panes);
    await settle(500);
    const beginResponse = await client.capture(sessionId, 'begin');
    const beginBinding = checkBinding(beginResponse, { sessionId, phase: 'begin' });
    captureId = beginBinding.captureId;
    began = true;

    // Match the former capture's remount and settle windows without running
    // expressions inside the page.
    await client.task(sessionId, 'navigate', { url: APP_MONITOR_URL });
    await settle(1000);
    const afterMountResponse = await client.capture(sessionId, 'after_mount', captureId);
    checkBinding(afterMountResponse, { sessionId, captureId, phase: 'after_mount', runId: beginBinding.runId });
    const afterMount = appMonitorCounters(afterMountResponse.metrics, args.panes);

    await client.task(sessionId, 'scroll', { x: 900, y: 800, delta_y: 1600 });
    await settle(1000);
    const afterScrollResponse = await client.capture(sessionId, 'after_scroll', captureId);
    checkBinding(afterScrollResponse, { sessionId, captureId, phase: 'after_scroll', runId: beginBinding.runId });
    const afterScroll = appMonitorCounters(afterScrollResponse.metrics, args.panes);

    await client.task(sessionId, 'keyboard', {
      key: 'k', modifiers: [process.platform === 'darwin' ? 'meta' : 'ctrl'],
    });
    await settle(500);
    await client.task(sessionId, 'fill', { selector: 'input[placeholder*="Search" i]', value: 'scenario' });
    await settle(500);
    const afterPickerResponse = await client.capture(sessionId, 'after_picker', captureId);
    checkBinding(afterPickerResponse, { sessionId, captureId, phase: 'after_picker', runId: beginBinding.runId });
    const afterPicker = appMonitorCounters(afterPickerResponse.metrics, args.panes);

    const finishResponse = await client.capture(sessionId, 'finish', captureId);
    began = false;
    const finishBinding = checkBinding(finishResponse, { sessionId, captureId, phase: 'finish', runId: beginBinding.runId });
    const trace = finishResponse.trace ?? null;
    let traceBytes = 0;
    if (args.trace) {
      if (!trace || trace.captureId !== captureId || trace.sessionId !== sessionId ||
          trace.contract !== 'app-monitor.preview-workspace.trace.v1' ||
          !Number.isSafeInteger(trace.byteLength) || trace.byteLength <= 0 ||
          Buffer.byteLength(trace.traceJson ?? '') !== trace.byteLength ||
          createHash('sha256').update(trace.traceJson).digest('hex') !== trace.sha256) {
        throw new Error('BAS returned an invalid bounded trace artifact');
      }
      traceBytes = trace.byteLength;
      await fsImpl.writeFile(args.trace, trace.traceJson, 'utf8');
    }

    const result = {
      capturedAt: new Date().toISOString(),
      url: APP_MONITOR_URL,
      panesRequested: args.panes,
      elapsedMs: Math.round(performance.now() - started),
      tracePath: args.trace,
      traceBytes,
      traceSHA256: args.trace && trace ? trace.sha256 : null,
      counters: { afterMount, afterScroll, afterPicker },
      errors: null,
      errorCollection: { status: 'unknown', reason: 'console exception text is not exposed by the typed Agent Access contract' },
      capture: {
        captureId, sessionId, runId: beginBinding.runId,
        taskRevision: finishBinding.taskRevision,
        oracleRevision: finishBinding.oracleRevision,
        profileRevision: finishBinding.profileRevision,
        phases: ['begin', 'after_mount', 'after_scroll', 'after_picker', 'finish'],
      },
    };
    await fsImpl.writeFile(args.out, `${JSON.stringify(result, null, 2)}\n`, 'utf8');
    log(JSON.stringify(result, null, 2));
    return result;
  } finally {
    if (sessionId && !closed) {
      try {
        if (began) await client.capture(sessionId, 'finish', captureId).catch(() => undefined);
      } finally {
        await client.closeSession(sessionId);
        closed = true;
      }
    }
  }
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  runCapture({ argv: process.argv.slice(2) }).catch((error) => {
    console.error(`[preview-workspace-capture] ${error.message || 'capture failed'}`);
    process.exitCode = 1;
  });
}
