import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import test from 'node:test';
import { assemble } from './qualification.mjs';

const scenarioRoot = resolve(new URL('../../../', import.meta.url).pathname);
const build = `sha256:${'a'.repeat(64)}`;
const sources = [
  'docs/internal/REFRACTOR_CONTRACT.json',
  'playwright-driver/tests/integration/saved-workflow-fresh-context.test.ts',
  'playwright-driver/src/recording/capture/browser-scripts/recording-script.js',
  'playwright-driver/src/recording/orchestration/pipeline-manager.ts',
  'api/handlers/record_mode.go',
  'api/services/recording/service.go',
  'api/services/recording/service_test.go',
  'api/services/recording/persistence/sqlite.go',
  'playwright-driver/tests/integration/pipeline-e2e.test.ts',
  'playwright-driver/src/proto/recording.ts',
];
const managed = {
  schemaVersion: 1, contractRow: 'passive-fidelity', managedBuildIdentityBefore: build,
  managedBuildIdentityAfter: build, actions: 10000, fixtureEffects: 10000, uniqueJournalIds: 10000,
  strictlyIncreasingJournalSequence: true, appliedInputReceipts: 10000, strictlyIncreasingAppliedSequence: true,
  storageIsolation: { routedTestPool: true, testPoolRequests: 20, primaryRequestsDuringTestMode: 0, temporaryDatabase: true },
};
const crash = {
  case: 'recording-service-process-death', actionsBeforeCrash: 10000, committedBeforeAcknowledgment: true,
  childTerminatedAbruptly: true, reopenedTotal: 10001, retriedSameEventId: true,
  totalAfterRetry: 10001, expectedPrefixIntactAndOrdered: true,
};
const semantics = [
  { case: 'core-events', actionTypes: ['click', 'type', 'scroll'], assertions: ['one independent click effect', 'typed value and input selector retained', 'positive scroll delta retained', 'unique increasing sequence numbers'] },
  { case: 'navigation', actionTypes: ['click', 'navigate'], assertions: ['click triggering navigation retained', 'navigation entry identifies /page-2'] },
  { case: 'capture-after-navigation', actionTypes: ['click', 'navigate', 'click'], assertions: ['pre-navigation click retained', 'navigation completed', 'post-navigation click retained'] },
];

async function fixture(t, tag = 'case') {
  const root = await mkdtemp('/tmp/bas-passive-assembler-');
  t.after(() => rm(root, { recursive: true, force: true }));
  for (const source of sources) {
    const path = join(root, source);
    await mkdir(dirname(path), { recursive: true });
    await writeFile(path, source);
  }
  const evidence = join(root, '.vrooli/runtime/rehabilitation-evidence');
  await mkdir(evidence, { recursive: true });
  await writeFile(join(evidence, `passive-fidelity-managed-${tag}.json`), JSON.stringify(managed));
  await writeFile(join(evidence, `passive-fidelity-crash-${tag}.json`), JSON.stringify(crash));
  await writeFile(join(evidence, `passive-fidelity-semantics-${tag}.jsonl`), `${semantics.map((row) => JSON.stringify(row)).join('\n')}\n`);
  return { root, evidence };
}

test('assembles provider receipts bound to all three raw artifacts and current source digests', async (t) => {
  const { root, evidence } = await fixture(t);
  const receipts = await assemble({ root, tag: 'case', liveBuild: build });
  assert.equal(receipts.length, 3);
  const managedReceipt = JSON.parse(await readFile(join(evidence, 'passive-fidelity-managed-case-assembled.json'), 'utf8'));
  const artifactBytes = await readFile(join(root, managedReceipt.owner_artifact.path));
  assert.equal(managedReceipt.owner_artifact.sha256, createHash('sha256').update(artifactBytes).digest('hex'));
  assert.equal(managedReceipt.owner_receipt.managedBuildIdentityAfter, build);
  assert.equal(Object.keys(managedReceipt.source_sha256).length, 7);
  const semanticReceipt = JSON.parse(await readFile(join(evidence, 'passive-fidelity-semantics-case-assembled.json'), 'utf8'));
  assert.equal(semanticReceipt.tests, '3/3');
  assert.equal(semanticReceipt.cases.length, 3);
});

test('rejects stale builds, incomplete cases, missing inputs, and retained-output overwrite', async (t) => {
  await t.test('stale build', async (t) => {
    const { root } = await fixture(t, 'stale');
    await assert.rejects(assemble({ root, tag: 'stale', liveBuild: `sha256:${'b'.repeat(64)}` }), /stale/);
  });
  await t.test('incomplete browser semantics', async (t) => {
    const { root, evidence } = await fixture(t, 'bad');
    await writeFile(join(evidence, 'passive-fidelity-semantics-bad.jsonl'), `${JSON.stringify({ ...semantics[0], assertions: [] })}\n`);
    await assert.rejects(assemble({ root, tag: 'bad', liveBuild: build }), /exactly three distinct cases/);
  });
  await t.test('missing owner artifact', async (t) => {
    const { root, evidence } = await fixture(t, 'missing');
    await rm(join(evidence, 'passive-fidelity-crash-missing.json'));
    await assert.rejects(assemble({ root, tag: 'missing', liveBuild: build }), /ENOENT/);
  });
  await t.test('does not overwrite retained wrapper', async (t) => {
    const { root, evidence } = await fixture(t);
    const output = join(evidence, 'passive-fidelity-managed-case-assembled.json');
    await writeFile(output, 'retained');
    await assert.rejects(assemble({ root, tag: 'case', liveBuild: build }), /refusing to overwrite/);
  });
});
