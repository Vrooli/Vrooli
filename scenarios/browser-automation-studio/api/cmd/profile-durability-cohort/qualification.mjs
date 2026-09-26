#!/usr/bin/env node
import { createHash, randomUUID } from 'node:crypto';
import { createServer } from 'node:http';
import { mkdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, relative, resolve, sep } from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { fileURLToPath } from 'node:url';

const [stage, outputArg] = process.argv.slice(2);
if (!['seed', 'verify', 'assemble'].includes(stage) || !outputArg) {
  throw new Error('usage: qualification.mjs <seed|verify|assemble> <output-directory>');
}
const output = resolve(outputArg);
const api = process.env.BAS_API_URL || 'http://127.0.0.1:17116';
const rpc = '/browser_automation_studio.v1.session_profiles.SessionProfilesService/';
const sourcePath = fileURLToPath(import.meta.url);
const contractPath = resolve(dirname(sourcePath), '../../../docs/internal/REFRACTOR_CONTRACT.json');
const contractHash = createHash('sha256').update(await readFile(contractPath)).digest('hex');
const sourceHash = createHash('sha256').update(await readFile(sourcePath)).digest('hex');
const samples = [];
const observations = [];
const liveSessions = new Set();
const results = [];
const cleanup = [];
const createdProfiles = [];
let keepSeededProfiles = false;
let fixture;
let fixturePort;
let nextObservation;

async function call(path, body, method = 'POST') {
  const response = await fetch(api + path, {
    method,
    headers: { 'Content-Type': 'application/json', 'Connect-Protocol-Version': '1' },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(30000),
  });
  const data = await response.json();
  if (!response.ok) throw new Error(`${path}: ${response.status}: ${JSON.stringify(data)}`);
  return data;
}

function makeFixture(port = 0) {
  fixture = createServer(async (request, response) => {
    if (request.url === '/ready' && request.method === 'POST') {
      let body = '';
      for await (const chunk of request) body += chunk;
      observations.push(JSON.parse(body));
      nextObservation?.();
      nextObservation = undefined;
      response.writeHead(204).end();
      return;
    }
    if (!request.url?.startsWith('/?')) {
      response.writeHead(404).end();
      return;
    }
    const url = new URL(request.url, `http://127.0.0.1:${fixturePort || 1}`);
    const identity = url.searchParams.get('identity');
    const mode = url.searchParams.get('mode');
    response.writeHead(200, { 'Content-Type': 'text/html', 'Cache-Control': 'no-store' });
    response.end(`<!doctype html><title>Profile durability fixture</title><script>
      (async () => {
        try {
          const identity = ${JSON.stringify(identity)};
          const mode = ${JSON.stringify(mode)};
          if (mode === 'seed') {
            document.cookie = 'fixture_identity=' + identity + '; Path=/; SameSite=Lax';
            localStorage.setItem('fixture_identity', identity);
          }
          const db = await new Promise((resolve, reject) => {
            const request = indexedDB.open('fixture-auth', 1);
            request.onupgradeneeded = () => request.result.createObjectStore('identity');
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
          });
          const indexed = await new Promise((resolve, reject) => {
            const tx = db.transaction('identity', mode === 'seed' ? 'readwrite' : 'readonly');
            const store = tx.objectStore('identity');
            if (mode === 'seed') store.put(identity, 'current');
            const request = store.get('current');
            request.onsuccess = () => resolve(request.result ?? null);
            request.onerror = () => reject(request.error);
          });
          db.close();
          await fetch('/ready', { method: 'POST', body: JSON.stringify({
            identity, cookie: document.cookie, localStorage: localStorage.getItem('fixture_identity'), indexedDB: indexed,
          }) });
        } catch (error) {
          await fetch('/ready', { method: 'POST', body: JSON.stringify({ error: String(error) }) });
        }
      })();
    </script>`);
  });
  return new Promise((resolve, reject) => {
    fixture.once('error', reject);
    fixture.listen(port, '127.0.0.1', () => {
      fixturePort = fixture.address().port;
      resolve();
    });
  });
}

function expectIdentity(observation, identity) {
  return observation !== undefined && observation.error === undefined
    && observation?.cookie.includes(`fixture_identity=${identity}`)
    && observation?.localStorage === identity
    && observation?.indexedDB === identity;
}

function waitForObservation() {
  const index = observations.length;
  return new Promise((resolve, reject) => {
    if (observations.length > index) {
      resolve(observations[index]);
      return;
    }
    const timer = setTimeout(() => reject(new Error('fixture observation timed out')), 10000);
    nextObservation = () => {
      clearTimeout(timer);
      resolve(observations[index]);
    };
  });
}

async function createProfile(label) {
  const result = await call(rpc + 'Create', { name: `BAS profile durability ${label} ${randomUUID()}` });
  if (!result.profile?.id) throw new Error('profile creation returned no profile id');
  createdProfiles.push(result.profile.id);
  return result.profile.id;
}

async function profile(profileId) {
  const result = await call(rpc + 'List', {});
  const match = result.profiles?.find((candidate) => candidate.id === profileId);
  if (!match) throw new Error(`profile ${profileId} disappeared`);
  return match;
}

async function openSession(profileId, identity, mode) {
  const url = `http://127.0.0.1:${fixturePort}/?identity=${encodeURIComponent(identity)}&mode=${mode}`;
  const result = await call('/api/v1/recordings/live/session', {
    session_profile_id: profileId, initial_url: url, restore_tabs: false,
    viewport_width: 640, viewport_height: 480,
  });
  const sessionId = result.sessionId ?? result.session_id;
  if (!sessionId) throw new Error('session creation returned no id');
  liveSessions.add(sessionId);
  return sessionId;
}

async function closeSession(sessionId) {
  await call(`/api/v1/recordings/live/session/${sessionId}/close`, {});
  liveSessions.delete(sessionId);
}

async function waitForAutomaticCheckpoint(profileId) {
  const started = performance.now();
  const deadline = started + 5000;
  while (performance.now() <= deadline) {
    const value = await profile(profileId);
    const observed = performance.now();
    const elapsedMs = observed - started;
    samples.push({ elapsedMs, hasStorageState: value.hasStorageState === true });
    if (value.hasStorageState === true) {
      return { observedAfterMs: elapsedMs, sampleCount: samples.length, passed: elapsedMs <= 5000 };
    }
    const remaining = deadline - performance.now();
    if (remaining <= 0) break;
    await delay(Math.min(50, remaining));
  }
  return { observedAfterMs: null, sampleCount: samples.length, passed: false };
}

async function seed(manifest) {
  const alpha = manifest?.identities?.alpha || randomUUID();
  const beta = manifest?.identities?.beta || randomUUID();
  const alphaProfile = manifest?.profiles?.alpha || await createProfile('alpha');
  const betaProfile = manifest?.profiles?.beta || await createProfile('beta');
  const alphaReady = waitForObservation();
  const alphaSession = await openSession(alphaProfile, alpha, 'seed');
  const seedObservation = await Promise.race([alphaReady, delay(10000).then(() => { throw new Error('alpha seed timed out'); })]);
  const writePassed = expectIdentity(seedObservation, alpha);
  results.push({ check: 'alpha cookie/localStorage/IndexedDB fixture writes', passed: writePassed });
  if (!writePassed) throw new Error(`alpha seed mismatch: ${JSON.stringify(seedObservation)}`);

  const checkpoint = await waitForAutomaticCheckpoint(alphaProfile);
  results.push({ check: 'automatic checkpoint visible within 5000ms of committed fixture writes', ...checkpoint });
  if (!checkpoint.passed) throw new Error('automatic checkpoint was not visible within 5000ms');
  await closeSession(alphaSession);

  const alphaReopenReady = waitForObservation();
  const alphaReopened = await openSession(alphaProfile, alpha, 'read');
  const alphaRead = await alphaReopenReady;
  const alphaReopenPassed = expectIdentity(alphaRead, alpha);
  results.push({ check: 'alpha identity survives close/reopen', passed: alphaReopenPassed });
  if (!alphaReopenPassed) throw new Error(`alpha close/reopen mismatch: ${JSON.stringify(alphaRead)}`);
  await closeSession(alphaReopened);

  const betaReady = waitForObservation();
  const betaSession = await openSession(betaProfile, beta, 'seed');
  const betaRead = await betaReady;
  const betaSeedPassed = expectIdentity(betaRead, beta);
  results.push({ check: 'beta identity seeds independently', passed: betaSeedPassed });
  if (!betaSeedPassed) throw new Error(`beta seed mismatch: ${JSON.stringify(betaRead)}`);
  await closeSession(betaSession);

  const alphaAgainReady = waitForObservation();
  const alphaAgain = await openSession(alphaProfile, alpha, 'read');
  const alphaAgainRead = await alphaAgainReady;
  const alphaIsolated = expectIdentity(alphaAgainRead, alpha);
  results.push({ check: 'beta profile activity does not replace alpha identity', passed: alphaIsolated });
  if (!alphaIsolated) throw new Error(`alpha isolation mismatch: ${JSON.stringify(alphaAgainRead)}`);
  await closeSession(alphaAgain);

  const saved = {
    profiles: { alpha: alphaProfile, beta: betaProfile },
    identities: { alpha, beta }, fixturePort,
  };
  await writeFile(`${output}/manifest.json`, JSON.stringify(saved, null, 2) + '\n');
  keepSeededProfiles = true;
  return saved;
}

async function verify(manifest) {
  for (const identityName of ['alpha', 'beta']) {
    const id = manifest.profiles[identityName];
    const identity = manifest.identities[identityName];
    const observed = waitForObservation();
    const sessionId = await openSession(id, identity, 'read');
    const read = await observed;
    const passed = expectIdentity(read, identity);
    results.push({ check: `${identityName} identity survives managed API/driver restart`, passed });
    if (!passed) throw new Error(`${identityName} restart mismatch: ${JSON.stringify(read)}`);
    await closeSession(sessionId);
  }
}

async function assembleCohort() {
  const seedPath = resolve(output, 'seed-receipt.json');
  const verifyPath = resolve(output, 'verify-receipt.json');
  const [seedReceipt, verifyReceipt] = await Promise.all([
    readFile(seedPath, 'utf8').then(JSON.parse),
    readFile(verifyPath, 'utf8').then(JSON.parse),
  ]);
  const live = await call('/health', undefined, 'GET');
  const build = seedReceipt.beforeHealth?.build_identity;
  if (!build || live.build_identity !== build) {
    throw new Error('live managed build does not match the seed owner receipt');
  }
  for (const [expectedStage, receipt] of [['seed', seedReceipt], ['verify', verifyReceipt]]) {
    if (receipt.stage !== expectedStage || receipt.contractSha256 !== contractHash
      || receipt.harnessSha256 !== sourceHash
      || receipt.beforeHealth?.build_identity !== build
      || receipt.afterHealth?.build_identity !== build
      || !Array.isArray(receipt.results) || receipt.results.length === 0
      || receipt.results.some((result) => result.passed !== true)
      || !Array.isArray(receipt.cleanup) || receipt.cleanup.some((result) => result.passed !== true)) {
      throw new Error(`${expectedStage} owner receipt is failed, stale, or from another build`);
    }
  }
  const seedChecks = seedReceipt.results.length;
  const restartChecks = verifyReceipt.results.length;
  const checkpoint = seedReceipt.results.find((result) => result.check === 'automatic checkpoint visible within 5000ms of committed fixture writes');
  const cleanupCount = verifyReceipt.cleanup.filter((result) => result.kind === 'profile' && result.passed === true).length;
  const alphaBetaIsolation = seedReceipt.results.find((result) => result.check === 'beta profile activity does not replace alpha identity')?.passed === true;
  const sourceFiles = {
    'api/cmd/profile-durability-cohort/qualification.mjs': sourceHash,
    'playwright-driver/src/session/manager.ts': createHash('sha256').update(await readFile(resolve(dirname(sourcePath), '../../../playwright-driver/src/session/manager.ts'))).digest('hex'),
  };
  const ownerRefs = [];
  for (const [receipt, path] of [[seedReceipt, seedPath], [verifyReceipt, verifyPath]]) {
    ownerRefs.push({
      path: relative(resolve(dirname(sourcePath), '../../..'), path).split(sep).join('/'),
      sha256: createHash('sha256').update(await readFile(path)).digest('hex'),
    });
  }
  const cohortReceipt = {
    contractRow: 'profile-durability',
    contractSha256: contractHash,
    sourceFiles,
    runtime: { managedBuildIdentityBeforeSeedAndAfterRestart: build },
    cohort: {
      allChecksPassed: seedChecks >= 5 && restartChecks >= 2 && checkpoint?.passed === true
        && Number.isFinite(checkpoint.observedAfterMs) && checkpoint.observedAfterMs <= 5000
        && alphaBetaIsolation && cleanupCount === 2,
      seedChecks,
      restartChecks,
      checkpointVisibleAfterMs: checkpoint?.observedAfterMs ?? null,
      alphaBetaIsolation: alphaBetaIsolation ? 'passed' : 'failed',
      alphaAndBetaAfterManagedApiDriverRestart: restartChecks >= 2 ? 'passed' : 'failed',
      syntheticProfilesDeleted: cleanupCount,
      seedOwnerReceipt: ownerRefs[0],
      restartOwnerReceipt: ownerRefs[1],
    },
  };
  const outputPath = resolve(output, '..', `${output.split(sep).filter(Boolean).at(-1)}.json`);
  await writeFile(outputPath, JSON.stringify(cohortReceipt, null, 2) + '\n');
  return { outputPath, cohort: cohortReceipt.cohort, sourceFiles: Object.keys(sourceFiles) };
}

if (stage === 'assemble') {
  const assembled = await assembleCohort();
  console.log(JSON.stringify(assembled, null, 2));
  if (!assembled.cohort.allChecksPassed) process.exitCode = 1;
} else {
let manifest;
let beforeHealth;
let afterHealth;
try {
  beforeHealth = await call('/health', undefined, 'GET');
  if (stage === 'seed') {
    await mkdir(output);
    await makeFixture();
    manifest = await seed();
  } else {
    manifest = JSON.parse(await readFile(`${output}/manifest.json`, 'utf8'));
    await makeFixture(manifest.fixturePort);
    await verify(manifest);
  }
} catch (error) {
  results.push({ check: 'harness', passed: false, error: String(error) });
} finally {
  for (const sessionId of [...liveSessions]) {
    try { await closeSession(sessionId); cleanup.push({ kind: 'session', passed: true }); }
    catch (error) { cleanup.push({ kind: 'session', passed: false, error: String(error) }); }
  }
  if (stage === 'verify' && manifest && liveSessions.size === 0) {
    for (const id of Object.values(manifest.profiles || {})) {
      try { await call(rpc + 'Delete', { id }); cleanup.push({ kind: 'profile', passed: true }); }
      catch (error) { cleanup.push({ kind: 'profile', passed: false, error: String(error) }); }
    }
  }
  if (stage === 'seed' && !keepSeededProfiles && liveSessions.size === 0) {
    for (const id of createdProfiles) {
      try { await call(rpc + 'Delete', { id }); cleanup.push({ kind: 'profile', passed: true }); }
      catch (error) { cleanup.push({ kind: 'profile', passed: false, error: String(error) }); }
    }
  }
  if (fixture) {
    fixture.closeAllConnections();
    await new Promise((resolve) => fixture.close(resolve));
  }
  try { afterHealth = await call('/health', undefined, 'GET'); }
  catch (error) { afterHealth = { error: String(error) }; }
  const receipt = {
    schemaVersion: 1, stage, observedAt: new Date().toISOString(),
    apiUrl: api, contractSha256: contractHash, harnessSha256: sourceHash,
    beforeHealth, afterHealth, results, samples, cleanup,
    manifest: stage === 'seed' && manifest ? manifest : undefined,
  };
  const receiptPath = `${output}/${stage}-receipt.json`;
  await writeFile(receiptPath, JSON.stringify(receipt, null, 2) + '\n');
  console.log(JSON.stringify({ stage, receiptPath, results, cleanup }, null, 2));
}
if (results.some((result) => result.passed === false) || cleanup.some((result) => result.passed === false)) process.exitCode = 1;
}
