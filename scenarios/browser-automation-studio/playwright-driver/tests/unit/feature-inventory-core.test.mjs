import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';
import { deflateSync } from 'node:zlib';
import { fileURLToPath } from 'node:url';
import { captureFeatureScreenshot, componentIdentity, featureDiffSummary, inventoryCoverage, mergeFeatureInventories, parseNamedEntrypoints, parseSafeActionLabels, parseSafeActionPaths, parseScopedSafeActionPaths, parseScopedSafeKeyChords, matchesSafeActionLabel, pngHasVisiblePixelVariation, reactComponentEvidence, reactEventPropNames, routeIdentityForState, safeInteractionKind, servedDocumentBuildIdentity, sha256Hex, stableFeatureId, validateInventory } from '../../scripts/feature-inventory-core.mjs';

const crc32 = (bytes) => {
  let crc = 0xffffffff;
  for (const byte of bytes) {
    crc ^= byte;
    for (let bit = 0; bit < 8; bit++) crc = crc & 1 ? (crc >>> 1) ^ 0xedb88320 : crc >>> 1;
  }
  return (crc ^ 0xffffffff) >>> 0;
};
const pngChunk = (type, data) => {
  const name = Buffer.from(type), length = Buffer.alloc(4), checksum = Buffer.alloc(4);
  length.writeUInt32BE(data.length); checksum.writeUInt32BE(crc32(Buffer.concat([name, data])));
  return Buffer.concat([length, name, data, checksum]);
};
const testPng = (pixels, colorType = 2) => {
  const channels = colorType === 6 ? 4 : colorType === 2 ? 3 : 1;
  const header = Buffer.alloc(13); header.writeUInt32BE(pixels.length / channels, 0); header.writeUInt32BE(1, 4); header[8] = 8; header[9] = colorType;
  return Buffer.concat([
    Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]),
    pngChunk('IHDR', header),
    pngChunk('IDAT', deflateSync(Buffer.concat([Buffer.from([0]), Buffer.from(pixels)]))),
    pngChunk('IEND', Buffer.alloc(0))
  ]);
};
const solidPng = testPng([12, 12, 12, 12, 12, 12]);
const solidColorPng = testPng([15, 23, 42, 15, 23, 42]);
const variedPng = testPng([12, 20, 32, 210, 220, 230]);

const inventory = (build, features) => ({
  schema: 'bas-feature-inventory/v1',
  identity: { scenario: 'sample', build, source_revision: `rev-${build}`, base_url: 'http://localhost:1234', captured_at: '2026-11-13T00:00:00Z', viewport: '390x844' },
  features
});
const feature = (id, kind, route = '/', identity = id) => ({ id, kind, route, identity, screenshot: `evidence/${id}.png` });

describe('served feature inventory interaction contract', () => {
  it('computes a full SHA-256 digest for served document bytes', () => {
    assert.equal(sha256Hex('abc'), 'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad');
  });

  it('detects uniform rendered PNGs while accepting varied and transparent samples', () => {
    assert.equal(pngHasVisiblePixelVariation(solidPng), false);
    assert.equal(pngHasVisiblePixelVariation(solidColorPng), false);
    assert.equal(pngHasVisiblePixelVariation(variedPng), true);
    assert.equal(pngHasVisiblePixelVariation(testPng([255, 0, 0, 0, 0, 255, 0, 0], 6)), false);
    assert.equal(pngHasVisiblePixelVariation(Buffer.from('not a png')), false);
  });

  it('uses measured served document bytes as build identity and rejects caller labels as evidence', () => {
    const document = { route: '/', state: 'default', sha256: 'a'.repeat(64), bytes: 1024 };
    assert.deepEqual(servedDocumentBuildIdentity([document]), {
      build: `sha256:${'a'.repeat(64)}`,
      build_source: document
    });
    assert.throws(() => servedDocumentBuildIdentity([]), /measured served document/);
    const captured = inventory(`sha256:${'a'.repeat(64)}`, [feature('route-home', 'route')]);
    captured.identity.build_source = document;
    captured.coverage = { servedDocuments: [document] };
    assert.deepEqual(validateInventory(captured), []);
    captured.identity.build = 'operator-label';
    assert.ok(validateInventory(captured).some((error) => error.includes('must match the served-document SHA-256')));
    captured.identity.build = `sha256:${'a'.repeat(64)}`;
    captured.coverage.servedDocuments = [];
    assert.ok(validateInventory(captured).some((error) => error.includes('must match a measured served document')));
  });

  it('keeps screenshot fallbacks clipped and prefers a failed element locator bounding box', async () => {
    const calls = [];
    const page = { screenshot: async (options) => { calls.push(options); await fs.writeFile(options.path, variedPng); } };
    const locator = {
      screenshot: async () => { throw new Error('remote element screenshot failed'); },
      boundingBox: async (options) => { assert.deepEqual(options, { timeout: 1000 }); return { x: -10, y: 12.4, width: 70.2, height: 30.1 }; }
    };
    const result = await captureFeatureScreenshot({ page, locator, path: 'crop.png' });
    assert.equal(result, 'bounded-fallback');
    assert.deepEqual(calls[0], { path: 'crop.png', clip: { x: 0, y: 12, width: 61, height: 31 } });

    calls.length = 0;
    await captureFeatureScreenshot({ page, locator: { screenshot: async () => { throw new Error('failed'); }, boundingBox: async () => null }, path: 'viewport.png' });
    assert.deepEqual(calls[0], { path: 'viewport.png', clip: { x: 0, y: 0, width: 390, height: 844 } });

    calls.length = 0;
    const blankElement = { screenshot: async ({ path }) => fs.writeFile(path, solidPng), boundingBox: async () => ({ x: 10, y: 20, width: 80, height: 30 }) };
    const fallbackMethod = await captureFeatureScreenshot({ page, locator: blankElement, path: 'blank-element.png' });
    assert.equal(fallbackMethod, 'bounded-fallback');
    assert.deepEqual(calls[0], { path: 'blank-element.png', clip: { x: 10, y: 20, width: 80, height: 30 } });
  });

  it('writes measured diffs only when every generated card is usable', async () => {
    const repo = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../..');
    const script = path.join(repo, 'scenarios/browser-automation-studio/playwright-driver/scripts/feature-inventory.mjs');
    const temp = await fs.mkdtemp(path.join(os.tmpdir(), 'bas-feature-diff-test-'));
    try {
      const png = Buffer.from('iVBORw0KGgoAAAANSUhEUg==', 'base64');
      await fs.writeFile(path.join(temp, 'before.png'), png);
      await fs.writeFile(path.join(temp, 'after.png'), png);
      const before = inventory('one', [
        { ...feature('route-home', 'route'), label: 'Home', screenshot: 'before.png' },
        { ...feature('action-save', 'action'), label: 'Save item', screenshot: 'before.png' }
      ]);
      const after = inventory('two', [{ ...feature('route-home', 'route'), label: 'Home', screenshot: 'after.png' }]);
      const beforePath = path.join(temp, 'before.json'), afterPath = path.join(temp, 'after.json'), outPath = path.join(temp, 'diff.json');
      await fs.writeFile(beforePath, JSON.stringify(before));
      await fs.writeFile(afterPath, JSON.stringify(after));
      const result = spawnSync('node', [script, 'diff', beforePath, afterPath, outPath], { encoding: 'utf8' });
      assert.equal(result.status, 0, result.stderr);
      const diff = JSON.parse(await fs.readFile(outPath, 'utf8'));
      assert.equal(diff.totals.added, 0);
      assert.equal(diff.totals.removed, 1);
      assert.equal(diff.totals.by_kind.action.removed, 1);

      before.features[1].label = '';
      await fs.writeFile(beforePath, JSON.stringify(before));
      await fs.rm(outPath);
      const rejected = spawnSync('node', [script, 'diff', beforePath, afterPath, outPath], { encoding: 'utf8' });
      assert.notEqual(rejected.status, 0);
      assert.match(rejected.stderr, /is not usable: title is empty/);
      await assert.rejects(fs.access(outPath));
    } finally {
      await fs.rm(temp, { recursive: true, force: true });
    }
  });

  it('rejects incomplete inventory identities and duplicate served features', () => {
    const invalid = inventory('one', [feature('same', 'action'), feature('same', 'component', '/other'), feature('another', 'action')]);
    invalid.features[2].identity = 'same';
    assert.ok(validateInventory(invalid).some((error) => error.includes('duplicate feature id')));
    assert.ok(validateInventory(invalid).some((error) => error.includes('duplicate feature identity')));
    const noSourceRevision = inventory('one', []);
    delete noSourceRevision.identity.source_revision;
    assert.deepEqual(validateInventory(noSourceRevision), []);
    const badCaptureMethod = inventory('one', [{ ...feature('action-a', 'action'), screenshot_capture: 'unclipped' }]);
    assert.ok(validateInventory(badCaptureMethod).some((error) => error.includes('screenshot_capture')));
    const badServedDocument = inventory('one', []);
    badServedDocument.coverage = { servedDocuments: [{ route: '/', state: 'default', sha256: 'short', bytes: 10 }] };
    assert.ok(validateInventory(badServedDocument).some((error) => error.includes('servedDocuments[0].sha256')));
  });

  it('reports added and removed totals grouped by served feature kind', () => {
    const before = inventory('one', [feature('route-a', 'route'), feature('button-a', 'action'), feature('panel-a', 'component')]);
    const after = inventory('two', [feature('route-a', 'route'), feature('button-b', 'action'), feature('panel-b', 'component'), feature('panel-c', 'component')]);
    assert.deepEqual(featureDiffSummary(before, after), {
      before: 3, after: 4, added: 3, removed: 2,
      by_kind: {
        action: { before: 1, after: 1, added: 1, removed: 1 },
        component: { before: 1, after: 2, added: 2, removed: 1 },
        route: { before: 1, after: 1, added: 0, removed: 0 }
      }
    });
  });

  it('merges same-build route crawls by stable feature identity and keeps source coverage', () => {
    const first = inventory('one', [feature('route-home', 'route'), { ...feature('panel-a', 'component'), component_evidence: 'react-type' }, { ...feature('panel-a', 'component'), component_evidence: 'react-type' }]);
    first.identity.source_revision = 'same-revision';
    first.coverage = { openedActions: [{ route: '/', state: 'settings' }], unobserved: [], components: { semantic: 1, react: 1, withEventProps: 0 } };
    first.coverage.keyboard = { candidates: [{ state: 'default', chord: 'Ctrl+Shift+H' }], observed: [{ route: '/', chord: 'Ctrl+Shift+H', state: 'keyboard:Ctrl+Shift+H' }] };
    first.coverage.loadedAssets = [{ route: '/', state: 'default', scripts: ['/assets/shared.js'] }];
    first.coverage.servedDocuments = [{ route: '/', state: 'default', sha256: 'a'.repeat(64), bytes: 1000 }];
    const second = inventory('one', [{ ...feature('panel-a', 'component'), component_evidence: 'react-type' }, { ...feature('dialog-a', 'component', '/', 'dialog-a'), component_evidence: 'semantic-region', event_props: ['onClick'] }]);
    second.identity.source_revision = 'different-unverified-label';
    second.identity.source_revision_hint = 'different-unverified-label';
    second.coverage = { openedActions: [{ route: '/', state: 'launcher' }], unobserved: [], components: { semantic: 0, react: 2, withEventProps: 1 } };
    second.coverage.keyboard = { candidates: [{ state: 'default', chord: 'Ctrl+Shift+H' }], observed: [{ route: '/', chord: 'Ctrl+Shift+H', state: 'keyboard:Ctrl+Shift+H' }] };
    second.coverage.loadedAssets = [{ route: '/', state: 'default', scripts: ['/assets/launcher.js'] }];
    second.coverage.servedDocuments = [{ route: '/', state: 'launcher', sha256: 'a'.repeat(64), bytes: 1000 }];
    const merged = mergeFeatureInventories([first, second]);
    assert.equal(merged.features.length, 3);
    assert.deepEqual(merged.coverage.openedActions.map((action) => action.state), ['settings', 'launcher']);
    assert.equal(merged.coverage.sources.length, 2);
    assert.equal(merged.coverage.sources[0].duplicate_records_removed, 1);
    assert.equal(merged.coverage.components.react, 1);
    assert.equal(merged.coverage.components.withEventProps, 1);
    assert.deepEqual(merged.coverage.keyboard, {
      candidates: [{ state: 'default', chord: 'Ctrl+Shift+H' }],
      observed: [{ route: '/', chord: 'Ctrl+Shift+H', state: 'keyboard:Ctrl+Shift+H' }]
    });
    assert.deepEqual(merged.coverage.loadedAssets, [{ route: '/', state: 'default', scripts: ['/assets/launcher.js', '/assets/shared.js'] }]);
    assert.deepEqual(merged.coverage.servedDocuments, [
      { route: '/', state: 'default', sha256: 'a'.repeat(64), bytes: 1000 },
      { route: '/', state: 'launcher', sha256: 'a'.repeat(64), bytes: 1000 }
    ]);
    const conflictingDocument = structuredClone(second);
    conflictingDocument.coverage.servedDocuments = [{ route: '/', state: 'default', sha256: 'b'.repeat(64), bytes: 1000 }];
    assert.throws(() => mergeFeatureInventories([first, conflictingDocument]), /different served documents/);
    assert.deepEqual(merged.coverage.sources.map((source) => source.loaded_assets.length), [1, 1]);
    assert.deepEqual(validateInventory(merged), []);
  });

  it('rejects merge inputs from different builds and conflicting duplicate identities', () => {
    const first = inventory('one', [feature('panel-a', 'component')]);
    const otherBuild = inventory('two', [feature('panel-b', 'component')]);
    assert.throws(() => mergeFeatureInventories([first, otherBuild]), /identity.build/);
    const conflicting = inventory('one', [feature('panel-a', 'action')]);
    assert.throws(() => mergeFeatureInventories([first, conflicting]), /conflicting semantic identity/);
  });

  it('keeps identities stable for the same served semantic feature', () => {
    assert.equal(stableFeatureId('tab-state', '/settings', 'tab|Preferences|state:Privacy'),
      stableFeatureId('tab-state', '/settings', 'tab|Preferences|state:Privacy'));
    assert.notEqual(stableFeatureId('tab-state', '/settings', 'tab|Preferences|state:Privacy'),
      stableFeatureId('tab-state', '/settings', 'tab|Preferences|state:Security'));
  });

  it('allows only explicit menu, disclosure and tab presentation semantics', () => {
    assert.equal(safeInteractionKind({ role: 'button', ariaHaspopup: 'menu', ariaExpanded: 'false' }), 'menu');
    assert.equal(safeInteractionKind({ tagName: 'button', ariaExpanded: 'false' }), 'disclosure');
    assert.equal(safeInteractionKind({ role: 'tab' }), 'tab');
    assert.equal(safeInteractionKind({ tagName: 'button', ariaHaspopup: 'dialog', ariaExpanded: 'false' }), null);
    assert.equal(safeInteractionKind({ tagName: 'button' }), null);
  });

  it('keeps unsupported gestures explicitly unobserved', () => {
    const coverage = inventoryCoverage();
    assert.deepEqual(coverage.candidates, { menu: 0, disclosure: 0, tab: 0 });
    assert.deepEqual(coverage.observed, { menu: 0, disclosure: 0, tab: 0 });
    for (const gesture of ['drag', 'hover', 'keyboard', 'scroll']) assert.ok(coverage.unattempted[gesture]);
  });

  it('prefers React type and source identity while retaining semantic fallback', () => {
    assert.equal(componentIdentity({ name: 'Panel', source: 'src/Panel.tsx:12', instance: 'key:privacy', semanticIdentity: 'region|main' }), 'react|Panel|src/Panel.tsx:12|key:privacy');
    assert.notEqual(componentIdentity({ name: 'Panel', source: 'src/Panel.tsx:12', instance: 'ordinal:0' }), componentIdentity({ name: 'Panel', source: 'src/Panel.tsx:12', instance: 'ordinal:1' }));
    assert.equal(componentIdentity({ semanticIdentity: 'region|main' }), 'region|main');
    assert.deepEqual(reactComponentEvidence({ name: 'Panel', source: 'src/Panel.tsx:12', eventProps: ['onClick', 'className', 'onPointerDown', 'onClick'] }), {
      name: 'Panel', source: 'src/Panel.tsx:12', eventProps: ['onClick', 'onPointerDown']
    });
  });

  it('reports event prop names only and never invokes handlers', () => {
    let called = false;
    const props = { onDragStart: () => { called = true; }, onKeyDown: () => { called = true; }, title: 'safe' };
    assert.deepEqual(reactEventPropNames(props), ['onDragStart', 'onKeyDown']);
    assert.equal(called, false);
  });

  it('requires exact normalized labels for safe action allowlists', () => {
    const allowlist = parseSafeActionLabels(' Settings, New session , , ');
    assert.deepEqual(allowlist, ['settings', 'new session']);
    assert.equal(matchesSafeActionLabel(' Settings ', allowlist), true);
    assert.equal(matchesSafeActionLabel('Settings with data', allowlist), false);
    assert.deepEqual(parseSafeActionPaths('Settings > Account | Settings > Devices'), [['settings', 'account'], ['settings', 'devices']]);
    assert.deepEqual(parseScopedSafeActionPaths('Settings > Account | approval::Pending > Resolved | file-preview::Dismiss'), [
      { state: null, path: ['settings', 'account'] },
      { state: 'approval', path: ['pending', 'resolved'] },
      { state: 'file-preview', path: ['dismiss'] }
    ]);
  });

  it('allows only explicitly scoped Ctrl/Cmd+Shift letter chords for keyboard paths', () => {
    assert.deepEqual(parseScopedSafeKeyChords(''), []);
    assert.deepEqual(parseScopedSafeKeyChords('archive::Ctrl+Shift+h | file-preview::Cmd+Shift+D'), [
      { state: 'archive', label: 'Ctrl+Shift+H', playwrightKey: 'Control+Shift+h' },
      { state: 'file-preview', label: 'Cmd+Shift+D', playwrightKey: 'Meta+Shift+d' }
    ]);
    assert.throws(() => parseScopedSafeKeyChords('Enter'), /safe key chord/);
    assert.throws(() => parseScopedSafeKeyChords('Alt+F4'), /safe key chord/);
    assert.throws(() => parseScopedSafeKeyChords('Control+Shift+Enter'), /safe key chord/);
    assert.throws(() => parseScopedSafeKeyChords('Control+h'), /safe key chord/);
  });

  it('accepts named same-origin synthetic entrypoints without retaining query data in route identity', () => {
    const entries = parseNamedEntrypoints([
      'approval=/?approval=synthetic-decision',
      'file-preview=/?file=%2Fsynthetic%2Fexample.txt'
    ], 'http://localhost:21233/');
    assert.equal(entries.length, 2);
    assert.equal(routeIdentityForState(entries[0].href, entries[0].name), '/ [state=approval]');
    assert.equal(routeIdentityForState(entries[1].href, entries[1].name), '/ [state=file-preview]');
    assert.equal(routeIdentityForState('http://localhost:21233/', 'default'), '/');
  });

  it('rejects unsafe, ambiguous, and cross-origin entrypoint definitions', () => {
    assert.throws(() => parseNamedEntrypoints(['approval'], 'http://localhost:21233/'), /NAME=URL/);
    assert.throws(() => parseNamedEntrypoints(['Approval=/?approval=synthetic'], 'http://localhost:21233/'), /must be lowercase/);
    assert.throws(() => parseNamedEntrypoints(['approval=/one', 'approval=/two'], 'http://localhost:21233/'), /duplicate entrypoint/);
    assert.throws(() => parseNamedEntrypoints(['away=https://example.com/'], 'http://localhost:21233/'), /served origin/);
  });
});
