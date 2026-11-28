#!/usr/bin/env node

// Browser-paced Dictation Studio qualification using Chrome's WAV-backed fake
// microphone and the product's metadata-only stream diagnostic global.
// No npm dependencies are required (Node 21+ provides WebSocket).
// Example: node scripts/browser-paced-dictation.mjs --attempts 30 --url http://127.0.0.1:23345/dictation-studio
// Recovery probes require an observed reconnect/replay transition and a
// complete, accounted final result. `--offline-after-partial-ms` closes the
// active product WebSocket while CDP network emulation is offline. Set
// `--offline-attempt all` to inject the interruption on every cohort turn.

import { createHash } from 'node:crypto';
import { mkdtemp, readFile, rm, writeFile } from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { spawn } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const args = new Map();
for (let i = 2; i < process.argv.length; i += 1) {
  const key = process.argv[i];
  if (!key.startsWith('--')) throw new Error(`unexpected argument: ${key}`);
  const next = process.argv[i + 1];
  if (next === undefined || next.startsWith('--')) args.set(key.slice(2), true);
  else args.set(key.slice(2), process.argv[++i]);
}
const get = (key, fallback) => args.get(key) ?? fallback;
if (typeof WebSocket !== 'function') throw new Error('Node 21 or newer is required for the built-in WebSocket client');
const attempts = Number(get('attempts', '30'));
if (!Number.isInteger(attempts) || attempts < 1 || attempts > 1000) throw new Error('--attempts must be 1..1000');
const offlineDurationMs = Number(get('offline-after-partial-ms', '0'));
if (!Number.isFinite(offlineDurationMs) || offlineDurationMs < 0 || offlineDurationMs > 30000) {
  throw new Error('--offline-after-partial-ms must be 0..30000');
}
const offlineAttemptArg = String(get('offline-attempt', '1'));
const offlineAttempt = offlineAttemptArg === 'all' ? 'all' : Number(offlineAttemptArg);
if (offlineAttempt !== 'all' && (!Number.isInteger(offlineAttempt) || offlineAttempt < 0 || offlineAttempt > attempts)) {
  throw new Error('--offline-attempt must be all or 0..attempts (0 disables offline injection)');
}
const injectOfflineOnAttempt = (index) => offlineAttempt === 'all' || offlineAttempt === index;
const interruptAfterPartial = args.has('interrupt-after-partial');
const baseUrl = get('url', process.env.AUDIO_TOOLS_UI_URL ?? 'http://127.0.0.1:23345/dictation-studio');
const fixture = path.resolve(get('fixture', fileURLToPath(new URL('../bas/fixtures/dictation-reference.wav', import.meta.url))));
const expectedProvider = get('provider', 'sherpa-stt');
const chrome = get('chrome', process.env.CHROME_BIN ?? '/usr/bin/google-chrome');
const scenarioRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const output = path.resolve(get('output', path.join(scenarioRoot, 'coverage', `browser-paced-${new Date().toISOString().replaceAll(':', '').replaceAll('.', '')}.json`)));
const fixtureBytes = await readFile(fixture);
const fixtureSha256 = createHash('sha256').update(fixtureBytes).digest('hex');
const profile = await mkdtemp(path.join(os.tmpdir(), 'audio-tools-paced-chrome-'));
let browser;
let socket;
let nextId = 0;
const pending = new Map();
const pageErrors = [];
let chromeStderr = '';

function cdp(method, params = {}, sessionId) {
  const id = ++nextId;
  const message = { id, method, params };
  if (sessionId) message.sessionId = sessionId;
  return new Promise((resolve, reject) => {
    pending.set(id, { resolve, reject });
    socket.send(JSON.stringify(message));
  });
}

function evaluate(expression, sessionId, timeout = 30000) {
  return cdp('Runtime.evaluate', {
    expression,
    awaitPromise: true,
    returnByValue: true,
    userGesture: true,
    timeout,
  }, sessionId).then((result) => {
    if (result.exceptionDetails) throw new Error(result.exceptionDetails.text ?? 'page evaluation failed');
    return result.result?.value;
  });
}

async function waitForChrome() {
  const deadline = Date.now() + 15000;
  while (Date.now() < deadline) {
    if (browser?.exitCode !== null && browser?.exitCode !== undefined) {
      throw new Error(`isolated Chrome exited with code ${browser.exitCode}${chromeStderr ? `: ${chromeStderr.trim()}` : ''}`);
    }
    try {
      const [portLine] = (await readFile(path.join(profile, 'DevToolsActivePort'), 'utf8')).split('\n');
      const port = Number(portLine);
      if (Number.isInteger(port) && port > 0 && port <= 65535) {
        return await (await fetch(`http://127.0.0.1:${port}/json/version`)).json();
      }
    }
    catch { await delay(100); }
  }
  throw new Error(`isolated Chrome did not open its debugging endpoint within 15 seconds${chromeStderr ? `: ${chromeStderr.trim()}` : ''}`);
}

async function runTurn(sessionId, index) {
  const isOfflineAttempt = injectOfflineOnAttempt(index);
  const turnOfflineDurationMs = isOfflineAttempt ? offlineDurationMs : 0;
  const startedAt = new Date().toISOString();
  const started = await evaluate('performance.now()', sessionId);
  const previousSession = await evaluate('globalThis.__VROOLI_AUDIO_STREAM_DIAGNOSTIC__?.latest?.sessionId ?? null', sessionId);
  await evaluate(`document.querySelector('[data-testid="dictation-record-start"]').click()`, sessionId);
  const observed = await evaluate(`(async (previousSession) => {
    const deadline = performance.now() + 15000;
    let firstVisibleAtMs = null;
    let visible = false;
    while (performance.now() < deadline) {
      const diag = globalThis.__VROOLI_AUDIO_STREAM_DIAGNOSTIC__?.latest;
      if (!diag || diag.sessionId === previousSession) { await new Promise(resolve => setTimeout(resolve, 20)); continue; }
      const text = document.querySelector('[data-testid="dictation-interim-transcript"]')?.textContent?.trim() ?? '';
      if (text) { visible = true; firstVisibleAtMs ??= performance.now(); }
      if (diag?.firstPartialAtMs != null && visible && diag.visibleRevisionCount >= 2) {
        return { firstVisibleAtMs, firstSignalAtMs: diag.firstSignalAtMs ?? null, visibleRevisionCount: diag.visibleRevisionCount, sessionId: diag.sessionId };
      }
      if (document.querySelector('[data-testid="dictation-record-error"]')) throw new Error('recorder error: ' + document.querySelector('[data-testid="dictation-record-error"]').textContent);
      await new Promise(resolve => setTimeout(resolve, 20));
    }
    throw new Error('timed out before first visible partial and second rendered revision');
  })(${JSON.stringify(previousSession)})`, sessionId, 20000);
  const closeProductSocket = async () => evaluate(`(() => {
      const socket = globalThis.__AUDIO_TOOLS_TEST_WEBSOCKET__;
      if (!socket || socket.readyState !== WebSocket.OPEN) return false;
      socket.close(4001, 'audio-tools deterministic recovery probe');
      return true;
    })()`, sessionId);
  let socketClosed = false;
  if (turnOfflineDurationMs > 0) {
    await cdp('Network.enable', {}, sessionId);
    await cdp('Network.emulateNetworkConditions', {
      offline: true, latency: 0, downloadThroughput: 0, uploadThroughput: 0,
    }, sessionId);
    try {
      // CDP offline emulation does not close an established WebSocket. Close
      // the product socket while offline so reconnect must span the outage.
      socketClosed = await closeProductSocket();
      if (!socketClosed) throw new Error('could not close the active product WebSocket while offline');
      await delay(turnOfflineDurationMs);
    } finally {
      await cdp('Network.emulateNetworkConditions', {
        offline: false, latency: 0, downloadThroughput: -1, uploadThroughput: -1,
      }, sessionId);
    }
  } else if (interruptAfterPartial) {
    socketClosed = await closeProductSocket();
    if (!socketClosed) throw new Error('could not close the active product WebSocket after the first visible partial');
  }
  // Let the fake microphone continue at 1x for a short, fixed window after
  // two rendered revisions. This keeps tail accounting exercised without
  // requiring the whole looping fixture on every attempt.
  await delay(1200);
  await evaluate(`document.querySelector('[data-testid="dictation-record-start"]').click()`, sessionId);
  const terminal = await evaluate(`(async (sessionId) => {
    const deadline = performance.now() + 15000;
    while (performance.now() < deadline) {
      const diag = globalThis.__VROOLI_AUDIO_STREAM_DIAGNOSTIC__?.latest;
      if (diag?.sessionId === sessionId && ['completed', 'failed', 'cancelled'].includes(diag.state)) return diag;
      await new Promise(resolve => setTimeout(resolve, 25));
    }
    throw new Error('timed out awaiting terminal stream diagnostic');
  })(${JSON.stringify(observed.sessionId)})`, sessionId, 20000);
  const visibleFinal = await evaluate(`Boolean(document.querySelector('[data-testid="dictation-final-transcript"]')?.textContent?.trim())`, sessionId);
  const complete = terminal.state === 'completed' && terminal.terminalReason === 'final';
  const accounted = terminal.capturedSamples === terminal.committedSamples
    && terminal.capturedSequence === terminal.sentSequence
    && terminal.sentSequence === terminal.processedSequence
    && terminal.doneSent;
  const visibleGapsComplete = Array.isArray(terminal.visibleUpdateGapsMs)
    && terminal.visibleUpdateGapsMs.length === terminal.visibleRevisionCount - 1
    && terminal.visibleUpdateGapsMs.every((gap) => Number.isFinite(gap) && gap >= 0);
  const recoveryStatuses = ['reconnecting', 'replaying', 'reconnect_exhausted', 'buffered_recovery', 'buffered_recovery_completed'];
  const interruptionInjected = socketClosed;
  const injectedInterruptionObserved = !interruptionInjected
    || (terminal.statusCodes ?? []).some((code) => recoveryStatuses.includes(code));
  const ok = complete && accounted && observed.visibleRevisionCount >= 2 && visibleGapsComplete && visibleFinal && !terminal.errorCodes?.length && injectedInterruptionObserved;
  return {
    attempt: index,
    startedAt,
    finishedAt: new Date().toISOString(),
    sessionId: terminal.sessionId,
    state: terminal.state,
    terminalReason: terminal.terminalReason ?? null,
    providerId: terminal.providerId ?? null,
    providerTier: terminal.providerTier ?? null,
    modelId: terminal.modelId ?? null,
    signalToFirstVisiblePartialMs: observed.firstSignalAtMs != null ? observed.firstVisibleAtMs - observed.firstSignalAtMs : null,
    signalToPartialDiagnosticMs: terminal.firstPartialLatencyMs ?? null,
    firstPartialAvailableAudioMs: terminal.firstPartialAvailableAudioMs ?? null,
    firstPartialExcessLatencyMs: terminal.firstPartialExcessLatencyMs ?? null,
    startToCaptureReadyMs: terminal.startToCaptureReadyMs ?? null,
    stopToFinalTailMs: terminal.stopToFinalTailMs ?? null,
    capturedSequence: terminal.capturedSequence,
    sentSequence: terminal.sentSequence,
    processedSequence: terminal.processedSequence,
    capturedSamples: terminal.capturedSamples,
    committedSamples: terminal.committedSamples,
    retainedBytes: terminal.retainedBytes,
    durability: terminal.durability,
    visibleRevisionCount: terminal.visibleRevisionCount,
    visibleUpdateGapsMs: terminal.visibleUpdateGapsMs,
    visibleGapsComplete,
    injectedOfflineDurationMs: turnOfflineDurationMs,
    injectedOfflineAttempt: isOfflineAttempt && turnOfflineDurationMs > 0,
    injectedWebSocketClose: socketClosed,
    injectedInterruptionObserved,
    visiblePartialObserved: true,
    visibleFinalObserved: visibleFinal,
    doneSent: terminal.doneSent,
    statusCodes: terminal.statusCodes,
    errorCodes: terminal.errorCodes,
    clickToFirstVisiblePartialMs: observed.firstVisibleAtMs - started,
    startedAtMs: started,
    passed: ok && (expectedProvider === 'any' || terminal.providerId === expectedProvider),
    failure: !complete
      ? `terminal ${terminal.state} (${terminal.terminalReason ?? 'no terminal reason'})`
      : expectedProvider !== 'any' && terminal.providerId !== expectedProvider
      ? `expected provider ${expectedProvider}, received ${terminal.providerId ?? 'unknown'}`
      : turnOfflineDurationMs > 0 && !injectedInterruptionObserved
        ? 'Chrome offline emulation did not produce a provider reconnect or replay transition'
      : undefined,
  };
}

function percentile(values, p) {
  if (!values.length) return null;
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.ceil(p * sorted.length) - 1];
}

function summary(values) {
  const finite = values.filter(Number.isFinite);
  return finite.length ? {
    n: finite.length,
    p50: percentile(finite, .5),
    p95_nearest_rank: percentile(finite, .95),
    max: Math.max(...finite),
    min: Math.min(...finite),
  } : { n: 0, p50: null, p95_nearest_rank: null, max: null, min: null };
}

try {
  browser = spawn(chrome, [
    '--headless=new', '--no-sandbox', '--disable-dev-shm-usage', '--no-first-run',
    '--disable-background-networking', '--use-fake-device-for-media-stream',
    '--use-fake-ui-for-media-stream', `--use-file-for-fake-audio-capture=${fixture}`,
    `--user-data-dir=${profile}`, '--remote-debugging-port=0', 'about:blank',
  ], { stdio: ['ignore', 'ignore', 'pipe'] });
  browser.stderr.setEncoding('utf8');
  browser.stderr.on('data', (chunk) => {
    chromeStderr = `${chromeStderr}${chunk}`.slice(-4000);
  });
  const version = await waitForChrome();
  socket = new WebSocket(version.webSocketDebuggerUrl);
  await new Promise((resolve, reject) => {
    socket.addEventListener('open', resolve, { once: true });
    socket.addEventListener('error', reject, { once: true });
  });
  socket.addEventListener('message', (event) => {
    const message = JSON.parse(event.data);
    if (message.method === 'Runtime.exceptionThrown') pageErrors.push(message.params.exceptionDetails?.text ?? 'uncaught page exception');
    if (message.id && pending.has(message.id)) {
      const p = pending.get(message.id); pending.delete(message.id);
      message.error ? p.reject(new Error(message.error.message)) : p.resolve(message.result);
    }
  });
  const { targetId } = await cdp('Target.createTarget', { url: 'about:blank' });
  const { sessionId } = await cdp('Target.attachToTarget', { targetId, flatten: true });
  await cdp('Runtime.enable', {}, sessionId);
  await cdp('Page.enable', {}, sessionId);
  await cdp('Page.addScriptToEvaluateOnNewDocument', { source: `(() => {
    const NativeWebSocket = window.WebSocket;
    window.WebSocket = new Proxy(NativeWebSocket, {
      construct(target, args) {
        const socket = Reflect.construct(target, args, target);
        if (String(args[0]).includes('/ws/stt/transcribe-stream-v2')) {
          globalThis.__AUDIO_TOOLS_TEST_WEBSOCKET__ = socket;
        }
        return socket;
      },
    });
  })();` }, sessionId);
  await cdp('Page.navigate', { url: baseUrl }, sessionId);
  await evaluate(`(async () => {
    const end = performance.now() + 30000;
    while (performance.now() < end) {
      if (document.querySelector('[data-testid="dictation-record-start"]')) return true;
      await new Promise(resolve => setTimeout(resolve, 50));
    }
    throw new Error('Dictation Studio recorder did not render at ' + location.href);
  })()`, sessionId, 35000);
  const startedAt = new Date().toISOString();
  let apiHealth = null;
  try {
    const response = await fetch(new URL('/health', baseUrl));
    if (response.ok) apiHealth = await response.json();
  } catch { /* build identity is useful context, not a run prerequisite */ }
  const runs = [];
  for (let i = 1; i <= attempts; i += 1) {
    try { runs.push(await runTurn(sessionId, i)); }
    catch (error) { runs.push({ attempt: i, passed: false, state: 'failed', finishedAt: new Date().toISOString(), failure: String(error) }); }
    if (i < attempts) await delay(250);
  }
  const successes = runs.filter((run) => run.passed);
  const allAccounted = successes.filter((run) => run.capturedSamples === run.committedSamples).length;
  const allProcessed = successes.filter((run) => run.capturedSequence === run.sentSequence && run.sentSequence === run.processedSequence).length;
  const doneSent = successes.filter((run) => run.doneSent).length;
  const revisionsCovered = successes.filter((run) => run.visibleRevisionCount >= 2).length;
  const gapsCovered = successes.filter((run) => run.visibleGapsComplete).length;
  const firstPartials = successes.filter((run) => run.visiblePartialObserved).length;
  const finalPartials = successes.filter((run) => run.visibleFinalObserved).length;
  const updateGaps = successes.flatMap((run) => run.visibleUpdateGapsMs ?? []);
  const finishedAt = new Date().toISOString();
  const fixtureEvidence = path.relative(scenarioRoot, fixture);
  const report = {
    schemaVersion: 1,
    runId: `browser-paced-${startedAt.replaceAll(':', '').replaceAll('.', '')}`,
    observedAt: { startedAt, finishedAt },
    qualification: 'diagnostic_only',
    cohort: {
      consumer: 'audio-tools Dictation Studio',
      profile: 'synthetic browser fake microphone',
      browser: version.Browser,
      host: `${os.platform()}-${os.arch()}`,
      capture: 'Chrome file-backed fake microphone, paced by browser media device',
      injectedInterruption: offlineDurationMs > 0 && offlineAttempt !== 0
        ? `Product WebSocket closed with code 4001 while Chrome was offline for ${offlineDurationMs} ms after the first visible partial on ${offlineAttempt === 'all' ? 'every attempt' : `attempt ${offlineAttempt}`}`
      : interruptAfterPartial ? 'Product WebSocket closed with code 4001 after the first visible partial' : null,
      route: baseUrl,
      providerId: successes[0]?.providerId ?? null,
      fixture: path.basename(fixture),
      fixturePath: fixtureEvidence.startsWith('..') ? fixture : fixtureEvidence,
      fixtureSha256,
    },
    buildIdentity: { api: apiHealth?.build_identity ?? apiHealth?.upstream?.build_identity ?? null },
    attemptCount: attempts,
    completedFinalCount: successes.filter((run) => run.state === 'completed' && run.terminalReason === 'final').length,
    failedOrIncompleteCount: attempts - successes.length,
    latencyMs: {
      signalToFirstVisiblePartial: summary(successes.map((run) => run.signalToFirstVisiblePartialMs)),
      signalToPartialDiagnostic: summary(successes.map((run) => run.signalToPartialDiagnosticMs)),
      clickToFirstVisiblePartial: summary(successes.map((run) => run.clickToFirstVisiblePartialMs)),
      postOnsetAudioAvailableAtFirstPartial: summary(successes.map((run) => run.firstPartialAvailableAudioMs)),
      excessAfterAvailableAudio: summary(successes.map((run) => run.firstPartialExcessLatencyMs)),
      clickToCaptureReady: summary(successes.map((run) => run.startToCaptureReadyMs)),
      stopToFinalTail: summary(successes.map((run) => run.stopToFinalTailMs)),
      visibleUpdateGap: summary(updateGaps),
    },
    accounting: {
      attemptsWithCapturedSamplesEqualCommittedSamples: allAccounted,
      attemptsWithAllSentSequenceProcessed: allProcessed,
      attemptsWithDoneSent: doneSent,
      attemptsWithAtLeastTwoVisibleRevisions: revisionsCovered,
      attemptsWithCompleteVisibleUpdateGaps: gapsCovered,
      attemptsWithFirstVisiblePartial: firstPartials,
      attemptsWithVisibleFinal: finalPartials,
      maxVisibleRevisionCount: Math.max(0, ...successes.map((run) => run.visibleRevisionCount ?? 0)),
      allPageErrors: pageErrors,
    },
    limitations: [
      'Synthetic media browser run; not a physical microphone/device qualification.',
      'Repeated WAV fixture supports timing and cursor accounting only; no independent speech-quality score was calculated.',
      'One cohort does not establish performance on other hosts, profiles, cold starts, or routes.',
      ...(interruptAfterPartial ? ['The active product WebSocket is closed with code 4001 after the first visible partial. This exercises client reconnect/replay behavior but is not a physical network or browser/device recovery qualification.'] : []),
      ...(offlineDurationMs > 0 ? ['The product WebSocket is explicitly closed while Chrome is offline because CDP offline emulation alone does not interrupt an established WebSocket. Reconnect/replay must be observed after connectivity returns. This is not a physical network or browser/device recovery qualification.'] : []),
    ],
    attempts: runs,
  };
  await writeFile(output, `${JSON.stringify(report, null, 2)}\n`);
  console.log(JSON.stringify({ output, attemptCount: attempts, completedCount: report.completedFinalCount, failureCount: report.failedOrIncompleteCount, firstVisiblePartialMs: report.latencyMs.signalToFirstVisiblePartial }, null, 2));
  if (successes.length !== attempts || pageErrors.length) process.exitCode = 1;
} finally {
  socket?.close();
  if (browser && browser.exitCode === null) {
    browser.kill('SIGTERM');
    await Promise.race([new Promise((resolve) => browser.once('exit', resolve)), delay(3000)]);
    if (browser.exitCode === null) {
      browser.kill('SIGKILL');
      await new Promise((resolve) => browser.once('exit', resolve));
    }
  }
  await rm(profile, { recursive: true, force: true, maxRetries: 20, retryDelay: 100 });
}
