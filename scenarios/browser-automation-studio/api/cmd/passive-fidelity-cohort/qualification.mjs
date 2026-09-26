#!/usr/bin/env node
import { createHash } from 'node:crypto';
import { mkdir, readFile, rename, writeFile } from 'node:fs/promises';
import { dirname, relative, resolve, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const managedSources = [
  'docs/internal/REFRACTOR_CONTRACT.json',
  'playwright-driver/tests/integration/saved-workflow-fresh-context.test.ts',
  'playwright-driver/src/recording/capture/browser-scripts/recording-script.js',
  'playwright-driver/src/recording/orchestration/pipeline-manager.ts',
  'api/handlers/record_mode.go',
  'api/services/recording/service.go',
  'api/services/recording/persistence/sqlite.go',
];
const crashSources = [
  'docs/internal/REFRACTOR_CONTRACT.json',
  'api/services/recording/service.go',
  'api/services/recording/service_test.go',
  'api/services/recording/persistence/sqlite.go',
];
const semanticsSources = [
  'docs/internal/REFRACTOR_CONTRACT.json',
  'playwright-driver/tests/integration/pipeline-e2e.test.ts',
  'playwright-driver/src/recording/capture/browser-scripts/recording-script.js',
  'playwright-driver/src/recording/orchestration/pipeline-manager.ts',
  'playwright-driver/src/proto/recording.ts',
];
const semanticTests = [
  '[CRITICAL] should capture all core event types in single session',
  '[CRITICAL] should capture navigation events',
  '[CRITICAL] should continue capturing events after navigation',
];
const requiredAssertions = new Map([
  ['core-events', ['one independent click effect', 'typed value and input selector retained', 'positive scroll delta retained', 'unique increasing sequence numbers']],
  ['navigation', ['click triggering navigation retained', 'navigation entry identifies /page-2']],
  ['capture-after-navigation', ['pre-navigation click retained', 'navigation completed', 'post-navigation click retained']],
]);

const hash = (data) => createHash('sha256').update(data).digest('hex');
const scenarioRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../../../');

async function readJSON(path) {
  return JSON.parse(await readFile(path, 'utf8'));
}

async function sourceHashes(root, paths) {
  const entries = await Promise.all(paths.map(async (path) => [path, hash(await readFile(resolve(root, path)))]));
  return Object.fromEntries(entries);
}

async function artifactRef(root, path) {
  return { path: relative(root, path).split(sep).join('/'), sha256: hash(await readFile(path)) };
}

function readJSONLines(text, label) {
  const lines = text.split(/\r?\n/).filter((line) => line.trim());
  if (lines.length === 0) throw new Error(`${label} contains no observations`);
  return lines.map((line, index) => {
    try {
      return JSON.parse(line);
    } catch (error) {
      throw new Error(`${label} line ${index + 1} is invalid JSON: ${error.message}`);
    }
  });
}

function validateManaged(owner, liveBuild) {
  const isolation = owner.storageIsolation;
  if (owner.schemaVersion !== 1 || owner.contractRow !== 'passive-fidelity'
      || owner.managedBuildIdentityBefore !== liveBuild || owner.managedBuildIdentityAfter !== liveBuild) {
    throw new Error('managed receipt is stale or has the wrong contract/schema');
  }
  if (owner.actions < 10000 || owner.fixtureEffects !== owner.actions || owner.uniqueJournalIds !== owner.actions
      || owner.strictlyIncreasingJournalSequence !== true || owner.appliedInputReceipts !== owner.actions
      || owner.strictlyIncreasingAppliedSequence !== true) {
    throw new Error('managed receipt does not prove 10,000 ordered effects, journal IDs, and applied-input receipts');
  }
  if (isolation?.routedTestPool !== true || isolation.testPoolRequests < 1
      || isolation.primaryRequestsDuringTestMode !== 0 || isolation.temporaryDatabase !== true) {
    throw new Error('managed receipt does not prove isolated temporary storage');
  }
}

function validateCrash(owner) {
  if (owner.case !== 'recording-service-process-death' || owner.actionsBeforeCrash < 10000
      || owner.committedBeforeAcknowledgment !== true || owner.childTerminatedAbruptly !== true
      || owner.reopenedTotal !== owner.actionsBeforeCrash + 1 || owner.retriedSameEventId !== true
      || owner.totalAfterRetry !== owner.reopenedTotal || owner.expectedPrefixIntactAndOrdered !== true) {
    throw new Error('crash receipt does not prove durable same-ID reconnect without duplication');
  }
}

function validateSemantics(cases) {
  const byName = new Map(cases.map((entry) => [entry.case, entry]));
  if (byName.size !== requiredAssertions.size) throw new Error('browser-semantics receipt must contain exactly three distinct cases');
  for (const [name, assertions] of requiredAssertions) {
    const entry = byName.get(name);
    if (!entry || !Array.isArray(entry.actionTypes) || !Array.isArray(entry.assertions)
        || assertions.some((expected) => !entry.assertions.includes(expected))) {
      throw new Error(`browser-semantics receipt is missing required assertions for ${name}`);
    }
  }
}

async function writeJSONAtomic(path, value) {
  await mkdir(dirname(path), { recursive: true });
  const temporary = `${path}.${process.pid}.tmp`;
  await writeFile(temporary, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o600, flag: 'wx' });
  await rename(temporary, path);
}

export async function assemble({ root = scenarioRoot, tag, liveBuild, managedPath, crashPath, semanticsPath }) {
  if (!/^[a-z0-9][a-z0-9-]*$/i.test(tag ?? '')) throw new Error('tag must be a short alphanumeric identifier');
  if (!/^sha256:[a-f0-9]{64}$/.test(liveBuild ?? '')) throw new Error('live build must be sha256:<64 lowercase hex>');
  const evidenceDir = resolve(root, '.vrooli/runtime/rehabilitation-evidence');
  const input = {
    managed: resolve(root, managedPath ?? `.vrooli/runtime/rehabilitation-evidence/passive-fidelity-managed-${tag}.json`),
    crash: resolve(root, crashPath ?? `.vrooli/runtime/rehabilitation-evidence/passive-fidelity-crash-${tag}.json`),
    semantics: resolve(root, semanticsPath ?? `.vrooli/runtime/rehabilitation-evidence/passive-fidelity-semantics-${tag}.jsonl`),
  };
  const [managedRaw, crashRaw, semanticsText] = await Promise.all([
    readJSON(input.managed), readJSON(input.crash), readFile(input.semantics, 'utf8'),
  ]);
  const cases = readJSONLines(semanticsText, 'browser-semantics receipt');
  validateManaged(managedRaw, liveBuild);
  validateCrash(crashRaw);
  validateSemantics(cases);

  const [managedHashes, crashHashes, semanticsHashes] = await Promise.all([
    sourceHashes(root, managedSources), sourceHashes(root, crashSources), sourceHashes(root, semanticsSources),
  ]);
  const [managedArtifact, crashArtifact, semanticsArtifact] = await Promise.all([
    artifactRef(root, input.managed), artifactRef(root, input.crash), artifactRef(root, input.semantics),
  ]);
  const outputs = [
    [resolve(evidenceDir, `passive-fidelity-managed-${tag}-assembled.json`), {
      contractRow: 'passive-fidelity', result: 'passed', source_sha256: managedHashes,
      owner_artifact: managedArtifact, owner_receipt: managedRaw,
    }],
    [resolve(evidenceDir, `passive-fidelity-process-crash-${tag}-assembled.json`), {
      contractRow: 'passive-fidelity', result: 'passed', tests: '1/1',
      ownerTest: 'TestJournalSameIDRetryRecoversAcrossServiceProcessDeath',
      source_sha256: crashHashes, owner_artifact: crashArtifact, owner: crashRaw,
    }],
    [resolve(evidenceDir, `passive-fidelity-semantics-${tag}-assembled.json`), {
      contractRow: 'passive-fidelity', result: 'passed', tests: '3/3', ownerTests: semanticTests,
      source_sha256: semanticsHashes, owner_artifact: semanticsArtifact, cases,
    }],
  ];
  for (const [path] of outputs) {
    try {
      await readFile(path);
      throw new Error(`refusing to overwrite retained receipt: ${path}`);
    } catch (error) {
      if (error.code !== 'ENOENT') throw error;
    }
  }
  for (const [path, value] of outputs) await writeJSONAtomic(path, value);
  return outputs.map(([path, value]) => ({ path: relative(root, path).split(sep).join('/'), sha256: hash(JSON.stringify(value, null, 2) + '\n') }));
}

async function main(argv) {
  const [command, tag, liveBuild] = argv;
  if (command !== 'assemble' || !tag || !liveBuild) {
    throw new Error('usage: qualification.mjs assemble <tag> <live-build-identity> [managed.json crash.json semantics.jsonl]');
  }
  const receipts = await assemble({ tag, liveBuild, managedPath: argv[3], crashPath: argv[4], semanticsPath: argv[5] });
  process.stdout.write(`${JSON.stringify({ result: 'passed', receipts }, null, 2)}\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main(process.argv.slice(2)).catch((error) => {
    process.stderr.write(`${error.message}\n`);
    process.exitCode = 1;
  });
}
