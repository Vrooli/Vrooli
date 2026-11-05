import test from 'node:test';
import assert from 'node:assert/strict';
import { execFile as execFileCallback } from 'node:child_process';
import { mkdtemp, rm, stat, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { promisify } from 'node:util';

import { allocateOutputDir, captureUrl, presentationExpectation, validatePresentationBinding, captureJourney, captureRouteSelectors, captureRouting, installCaptureRouting, correlateCheckpoints, validateRecordingDimensions, captureProductionEvidence, decodeVideoFrames, rgbStats, validateCaptureEvidence, validatePageDiagnostics, validateVideoFrames, resolveBrowserExecutable, resolveHeadless } from './capture-public-evidence.mjs';

test('resolveBrowserExecutable prefers explicit existing browser and falls back safely', () => {
  assert.equal(resolveBrowserExecutable('/usr/bin/google-chrome'), '/usr/bin/google-chrome');
  assert.ok(resolveBrowserExecutable('/definitely/missing-browser'));
});

test('resolveHeadless fails safe when no display server is available', () => {
  assert.equal(resolveHeadless(false), true);
});

const execFile = promisify(execFileCallback);

function solidRGB(width, height, [red, green, blue]) {
  return Buffer.alloc(width * height * 3, Buffer.from([red, green, blue]));
}

function darkPageRGB(width = 10, height = 10) {
  const pixels = solidRGB(width, height, [8, 8, 8]);
  pixels.writeUInt8(220, 0);
  pixels.writeUInt8(220, 1);
  pixels.writeUInt8(220, 2);
  return pixels;
}

function visiblePage(index = 0) {
  const pixels = darkPageRGB();
  pixels.writeUInt8(40 + index * 10, 3);
  pixels.writeUInt8(90 + index * 10, 4);
  pixels.writeUInt8(160 + index * 10, 5);
  return pixels;
}

const identity = {
  route: '/',
  variant: 'control',
  revision: 'rev-20260915-test',
  viewport: { width: 1440, height: 900, dpr: 1 },
  ready: true,
  preview: false,
  fallback: false,
  fontFailures: [],
  landmarks: ['hero', 'product', 'catalog', 'closing', 'app-detail'],
  overlays: [],
  assets: [{ src: '/hero.png', complete: true, naturalWidth: 1440, naturalHeight: 720 }],
};

test('navigation cannot escape the configured HTTP origin or file fixture directory', () => {
  assert.equal(new URL(captureUrl('https://example.test', '/apps/aquila', 'fixture')).pathname, '/apps/aquila');
  for (const route of ['https://other.test/', '//other.test/', 'http://example.test/', 'https://example.test:4430/', 'https://user:secret@example.test/']) assert.throws(() => captureUrl('https://example.test', route, 'fixture'), /configured origin/);
  assert.equal(new URL(captureUrl('file:///tmp/fixture/', 'detail.html', 'fixture')).pathname, '/tmp/fixture/detail.html');
  for (const route of ['../outside.html', 'file:///etc/passwd', 'file://other/tmp/fixture/index.html', '%2e%2e/outside.html']) assert.throws(() => captureUrl('file:///tmp/fixture/', route, 'fixture'));
});

test('recorded bootstrap, expected page digest and mounted block order must agree', () => {
  const payload = { appKey: 'web-console', page: { id: 'aquila', blocks: [{ id: 'hero', kind: 'product-hero' }, { id: 'voice', kind: 'voice-story' }] },
    diagnostics: { resolvedRoute: '/', resolvedVariant: identity.variant, resolvedRevision: identity.revision, blockDigest: `sha256:${'a'.repeat(64)}` } };
  const publication = presentationExpectation(payload);
  const expected = { ...identity, presentation: publication };
  const actual = { ...publication, renderedBlocks: publication.blocks };
  assert.equal(validatePresentationBinding(actual, expected).ok, true);
  for (const changed of [undefined, { ...actual, revision: 'stale' }, { ...actual, digest: 'bad' }, { ...actual, pageID: 'other-app' }, { ...actual, appKey: 'other-app' },
    { ...actual, renderedBlocks: [...actual.blocks].reverse() }, { ...actual, renderedBlocks: actual.blocks.slice(1) }, { ...actual, preview: true }, { ...actual, payloadHash: 'other-content' }]) {
    assert.equal(validatePageDiagnostics({ ...identity, presentation: changed }, expected).ok, false);
  }
});

test('isolated routing adds only the canonical marker and never claims public cutover', () => {
  assert.deepEqual(captureRouting({ baseUrl: 'https://example.test' }), { evidenceScope: 'public-route' });
  for (const baseUrl of ['http://127.0.0.1:1234', 'http://localhost:1234', 'http://[::1]:1234']) {
    assert.deepEqual(captureRouting({ testMode: true, baseUrl }), {
      evidenceScope: 'isolated-integration-fixture', extraHTTPHeaders: { 'x-vrooli-test-mode': '1' },
    });
  }
  for (const baseUrl of ['https://example.test', 'http://localhost.evil.test', 'file:///tmp/fixture.html', 'http://user:secret@localhost']) {
    assert.throws(() => captureRouting({ testMode: true, baseUrl }), /loopback/);
  }
  assert.throws(() => captureRouting({ testMode: '1', baseUrl: 'http://localhost' }), /boolean/);
});

test('captures every configured section in its configured order, never path-shaped labels', () => {
  assert.deepEqual(captureJourney({ hero: '#hero', voice: '#voice', artifacts: '#files', sessions: '#sessions', closing: '#closing', detail: '#detail' }), ['hero', 'voice', 'artifacts', 'sessions', 'closing']);
  assert.deepEqual(captureJourney({ hero: '#hero', closing: '#closing', product: '#product' }), ['hero', 'closing', 'product']);
  for (const checkpoints of [null, [], {}, { hero: '' }, { hero: '#hero', '../outside': '#escape' }, { hero: '#hero', 'app-detail': '#reserved' }]) assert.throws(() => captureJourney(checkpoints));
});

test('test-mode request headers apply only to the exact captured origin, never external fonts', async () => {
  let predicate, handler;
  await installCaptureRouting({ route: async (match, callback) => { predicate = match; handler = callback; } },
    captureRouting({ testMode: true, baseUrl: 'http://localhost:1234' }), 'http://localhost:1234');
  assert.equal(predicate(new URL('http://localhost:1234/assets/font.ttf')), true);
  for (const url of ['https://fonts.gstatic.com/font.woff2', 'http://localhost:9999', 'http://localhost.evil.test:1234']) assert.equal(predicate(new URL(url)), false);
  let forwarded;
  await handler({ request: () => ({ headers: () => ({ accept: 'application/json' }) }), continue: async options => { forwarded = options.headers; } });
  assert.deepEqual(forwarded, { accept: 'application/json', 'x-vrooli-test-mode': '1' });
  await installCaptureRouting({ route: () => assert.fail('ordinary capture must not intercept requests') }, { evidenceScope: 'public-route' }, 'https://example.test');
});

test('primes only landmarks belonging to the current route', () => {
  const checkpoints = { hero: '#hero', voice: '#voice', detail: '#detail' };
  assert.deepEqual(captureRouteSelectors(checkpoints), { hero: '#hero', voice: '#voice' });
  assert.deepEqual(captureRouteSelectors(checkpoints, 'detail'), { 'app-detail': '#detail' });
  assert.deepEqual(captureRouteSelectors({ hero: '#hero' }, 'detail'), {});
  assert.throws(() => captureRouteSelectors(checkpoints, 'unknown'));
});

test('decoder rejects unbounded sampling and process-timeout settings before launching media tools', async () => {
  for (const options of [{ processTimeoutMs: 0 }, { processTimeoutMs: Infinity }, { processTimeoutMs: 60001 }, { maxWidth: NaN }, { maxWidth: 999999 }]) {
    const result = await decodeVideoFrames('/must-not-be-opened.mp4', 4, options);
    assert.equal(result.frames.length, 0);
    assert.match(result.decoderError, /timeout|configuration/);
  }
  for (const fps of [0, -1, Infinity, NaN, 61]) {
    assert.match((await decodeVideoFrames('/must-not-be-opened.mp4', fps)).decoderError, /configuration/);
  }
});

test('regression: old YAVG check accepts legal-range black, RGB validation rejects it', () => {
  // signalstats reports legal-range black near YAVG=16, so the old `YAVG > 2`
  // guard accepted this frame even though its decoded RGB pixels are uniform.
  assert.equal(16 > 2, true);
  const result = validateVideoFrames({
    fps: 4,
    frames: [{ pixels: solidRGB(10, 10, [0, 0, 0]) }, { pixels: solidRGB(10, 10, [0, 0, 0]) }, { pixels: solidRGB(10, 10, [0, 0, 0]) }, { pixels: solidRGB(10, 10, [0, 0, 0]) }],
  });
  assert.equal(result.ok, false);
  assert.ok(result.reasons.some((reason) => reason.code === 'sustained-blank'));
});

test('regression: old pixel variation check accepts a first-run/wrong-page overlay', () => {
  // The old script only required first-frame YAVG > 2. This wrong-window frame
  // is therefore bright/nonuniform enough to pass its pixel gate.
  assert.ok(rgbStats(visiblePage(0)).mean > 2);
  const wrongWindow = validateCaptureEvidence({
    frames: [{ pixels: visiblePage(0) }, { pixels: visiblePage(1) }, { pixels: visiblePage(2) }],
    page: { ...identity, landmarks: ['hero'], overlays: [{ kind: 'chrome-first-run', visible: true, text: 'Make Chrome your default browser' }] },
    expected: identity,
  });
  assert.equal(wrongWindow.ok, false);
  assert.ok(wrongWindow.reasons.some((reason) => reason.code === 'blocking-overlay'));
});

test('rejects requested control when only the URL query says control', () => {
  const renderedFallback = validatePageDiagnostics({
    ...identity,
    variant: '',
    url: 'https://fixture.test/?variant_slug=control',
  }, identity);
  assert.equal(renderedFallback.ok, false);
  assert.ok(renderedFallback.reasons.some((reason) => reason.code === 'variant-mismatch'));
});

test('requires explicit public/non-fallback state and rejects failed fonts independently of ready', () => {
  for (const [field, code] of [['preview', 'preview-state'], ['fallback', 'fallback-state']]) {
    for (const value of [true, undefined, 'false']) {
      const result = validatePageDiagnostics({ ...identity, [field]: value }, identity);
      assert.equal(result.ok, false);
      assert.ok(result.reasons.some(reason => reason.code === code));
    }
  }
  const result = validatePageDiagnostics({ ...identity, fontFailures: ['Configured font'] }, identity);
  assert.equal(result.ok, false);
  assert.ok(result.reasons.some(reason => reason.code === 'font-failure'));
});

test('rejects a uniform white recording as well as uniform black', () => {
  const result = validateVideoFrames({
    fps: 4,
    frames: [{ pixels: solidRGB(10, 10, [255, 255, 255]) }, { pixels: solidRGB(10, 10, [255, 255, 255]) }, { pixels: solidRGB(10, 10, [255, 255, 255]) }, { pixels: solidRGB(10, 10, [255, 255, 255]) }],
  });
  assert.equal(result.ok, false);
  assert.ok(result.reasons.some((reason) => reason.code === 'sustained-blank'));
});

test('rejects a sustained late-black interval instead of checking only the opening', () => {
  const frames = [
    { pixels: visiblePage(0) }, { pixels: visiblePage(1) }, { pixels: visiblePage(2) },
    { pixels: solidRGB(10, 10, [0, 0, 0]) }, { pixels: solidRGB(10, 10, [0, 0, 0]) }, { pixels: solidRGB(10, 10, [0, 0, 0]) },
  ];
  const result = validateVideoFrames({ fps: 4, frames, lateBlankSeconds: 0.5 });
  assert.equal(result.ok, false);
  assert.ok(result.reasons.some((reason) => reason.code === 'sustained-blank' && reason.startFrame === 3));
});

test('rejects a frozen recording when the planned journey required a page change', () => {
  const same = visiblePage(0);
  const result = validateVideoFrames({
    fps: 4,
    frames: [{ pixels: same }, { pixels: Buffer.from(same) }, { pixels: Buffer.from(same) }],
    plannedChanges: [{ label: 'scroll-to-product', beforeFrame: 0, afterFrame: 2, requiresMotion: true, minMotion: 0.01 }],
  });
  assert.equal(result.ok, false);
  assert.ok(result.reasons.some((reason) => reason.code === 'frozen-change'));
});

test('allows a valid dark static page when landmarks and pixels are present', () => {
  const result = validateCaptureEvidence({
    frames: [{ pixels: darkPageRGB() }, { pixels: darkPageRGB() }, { pixels: darkPageRGB() }],
    page: identity,
    expected: identity,
    plannedChanges: [],
  });
  assert.equal(result.ok, true, JSON.stringify(result.reasons));
});

test('fails closed for route/variant/revision/viewport mismatch, overlays, assets, and decoder errors', () => {
  const page = validatePageDiagnostics({
    ...identity,
    route: '/wrong',
    variant: 'other',
    revision: 'old-revision',
    viewport: { width: 390, height: 844, dpr: 2 },
    overlays: [{ kind: 'permission', visible: true, text: 'Allow microphone access' }],
    assets: [{ src: '/missing.png', complete: false, naturalWidth: 0, naturalHeight: 0 }],
  }, identity);
  assert.equal(page.ok, false);
  for (const code of ['route-mismatch', 'variant-mismatch', 'revision-mismatch', 'viewport-mismatch', 'blocking-overlay', 'asset-failure']) {
    assert.ok(page.reasons.some((reason) => reason.code === code), `missing ${code}`);
  }

  const decoder = validateVideoFrames({ fps: 4, frames: [], decoderError: 'ffmpeg exited 1' });
  assert.equal(decoder.ok, false);
  assert.ok(decoder.reasons.some((reason) => reason.code === 'decoder-failure'));
});

test('checkpoint screenshots must actually appear in the video in journey order', () => {
  const first = solidRGB(10, 10, [40, 90, 130]);
  const second = solidRGB(10, 10, [190, 140, 20]);
  const frames = [{ pixels: first }, { pixels: second }];
  assert.equal(correlateCheckpoints(frames, [{ label: 'hero', pixels: first }, { label: 'product', pixels: second }]).ok, true);
  assert.equal(correlateCheckpoints(frames, [{ label: 'hero', pixels: second }, { label: 'product', pixels: first }]).ok, false);
  assert.equal(correlateCheckpoints([{ pixels: first }, { pixels: first }], [{ label: 'product', pixels: second }]).ok, false);
  assert.equal(correlateCheckpoints([{ pixels: first }], [{ label: 'root', pixels: first }, { label: 'detail', pixels: first }]).ok, false, 'one decoded frame cannot prove two ordered checkpoints');
  assert.equal(correlateCheckpoints([{ pixels: first }, { pixels: first }], [{ label: 'root', pixels: first }, { label: 'detail', pixels: first }]).ok, true, 'distinct frames of valid static content remain supported');
  assert.equal(validateRecordingDimensions({ width: 800, height: 600 }, { width: 1440, height: 900 }).ok, false);
});

test('concurrent captures never reuse an existing or another capture output directory', async () => {
  const root = await mkdtemp(join(tmpdir(), 'lpbs-output-allocation-'));
  try {
    const paths = await Promise.all(Array.from({ length: 4 }, () => allocateOutputDir(join(root, 'evidence'))));
    assert.equal(new Set(paths).size, 4);
    assert.notEqual(await allocateOutputDir(paths[0]), paths[0]);
  } finally { await rm(root, { recursive: true, force: true }); }
});

test('captures headed and headless page-bound desktop/mobile journeys and validates the real video', { timeout: 90000 }, async () => {
  const fixtureDir = await mkdtemp(join(tmpdir(), 'lpbs-capture-fixture-'));
  const outputDir = join(fixtureDir, 'evidence');
  const fixtureImage = 'data:image/svg+xml,%3Csvg%20xmlns%3D%22http%3A%2F%2Fwww.w3.org%2F2000%2Fsvg%22%20width%3D%2264%22%20height%3D%2264%22%3E%3Crect%20width%3D%2264%22%20height%3D%2264%22%20fill%3D%22%23e8795f%22%2F%3E%3C%2Fsvg%3E';
  const fixture = (heading, detail = false, renderedVariant = true) => `<!doctype html>
<html data-experience-state="ready" data-presentation-preview="false" data-presentation-fallback="false"${renderedVariant ? ' data-variant-slug="fixture"' : ''} data-presentation-revision="fixture-rev-1">
  <head><meta charset="utf-8"><title>LPBS capture fixture</title><style>
    *{box-sizing:border-box}html,body{margin:0;background:#070b14;color:#f8fafc;font:20px system-ui,sans-serif}
    [data-capture-landmark]{min-height:700px;padding:80px;background:linear-gradient(135deg,#070b14,#132b40)}
    [data-capture-landmark="product"]{background:linear-gradient(135deg,#102b38,#254d4b)}
    [data-capture-landmark="catalog"]{background:linear-gradient(135deg,#2b203c,#4b3152)}
    [data-capture-landmark="closing"]{background:linear-gradient(135deg,#3a241d,#71402f)}
    [data-capture-landmark="app-detail"]{background:linear-gradient(135deg,#1c3a2f,#356b50)}
    h1{font-size:48px;max-width:720px}img{width:64px;height:64px}
  </style><script>const unusedFont = new FontFace('UnrelatedAdmin', 'url(data:font/woff2;base64,broken)');document.fonts.add(unusedFont);unusedFont.load().catch(()=>{});</script></head>
  <body><img loading="lazy" style="display:none" src="missing-unused-lazy-image.png" alt="Unvisited hidden fixture image"><div class="presentation-page"><main>
    <section data-capture-landmark="hero"><img src="${fixtureImage}" alt="Fixture art"><h1>${heading}</h1><p>Same-page capture fixture; this is not production LPBS evidence.</p></section>
    <section data-capture-landmark="product"><h2>Product view</h2><p>Product checkpoint content.</p></section>
    <section data-capture-landmark="catalog"><h2>Bundle catalog</h2><p>Catalog checkpoint content.</p></section>
    <section data-capture-landmark="closing"><h2>Closing download state</h2><p>Download checkpoint content.</p></section>
    ${detail ? '<section data-capture-landmark="app-detail"><h2>App detail route</h2><p>App-detail checkpoint content.</p></section>' : ''}
  </main></div><script>
    const blocks = [...document.querySelectorAll('main > section')].map(element => {
      element.id = element.dataset.captureLandmark; element.dataset.block = element.id;
      return {id: element.id, kind: element.id};
    });
    const bootstrap = document.createElement('script'); bootstrap.id = 'lpbs-presentation-bootstrap'; bootstrap.type = 'application/json';
    bootstrap.textContent = JSON.stringify({config:{presentation:{appKey:'fixture',page:{id:'fixture-page',blocks},diagnostics:{resolvedRoute:location.pathname,resolvedVariant:'fixture',resolvedRevision:'fixture-rev-1',blockDigest:'sha256:${'a'.repeat(64)}'}}}});
    document.head.append(bootstrap);
  </script></body>
</html>`;
  await writeFile(join(fixtureDir, 'index.html'), fixture('Controlled page-bound capture'));
  await writeFile(join(fixtureDir, 'detail.html'), fixture('Controlled app detail capture', true));
  await writeFile(join(fixtureDir, 'bad-variant.html'), fixture('Missing rendered variant', false, false));
  await writeFile(join(fixtureDir, 'unavailable.html'), fixture('Unavailable page').replace('data-experience-state="ready"', 'data-experience-state="unavailable"'));
  await writeFile(join(fixtureDir, 'failed-font.html'), fixture('Required font failed').replace('</style>', 'body{font-family:UnrelatedAdmin,system-ui}</style>'));

  const baseUrl = pathToFileURL(`${fixtureDir}/`).href;
  const checkpoints = {
    hero: '[data-capture-landmark="hero"]',
    product: '[data-capture-landmark="product"]',
    artifacts: '[data-capture-landmark="catalog"]',
    closing: '[data-capture-landmark="closing"]',
    detail: '[data-capture-landmark="app-detail"]',
  };
  const checkpointSelectors = {
    hero: checkpoints.hero,
    product: checkpoints.product,
    artifacts: checkpoints.artifacts,
    closing: checkpoints.closing,
    'app-detail': checkpoints.detail,
  };
  const previousPlaywrightModule = process.env.MOCKUP_PLAYWRIGHT_MODULE;
  process.env.MOCKUP_PLAYWRIGHT_MODULE = '/home/matthalloran8/Vrooli/node_modules/.pnpm/playwright@1.62.1/node_modules/playwright/index.mjs';
  try {
    assert.equal(await stat(process.env.MOCKUP_PLAYWRIGHT_MODULE).then(() => true), true);
    const receipt = await captureProductionEvidence({
      baseUrl,
      outputDir,
      variant: 'fixture',
      route: 'index.html',
      detailRoute: 'detail.html',
      revision: 'fixture-rev-1',
      width: 1440,
      height: 900,
      dpr: 1,
      headless: false,
      timeoutMs: 15000,
      settleMs: 350,
      recordMs: 1600,
      sampleFps: 4,
      checkpoints,
      checkpointSelectors,
    });
    assert.equal(receipt.status, 'passed', JSON.stringify(receipt.errors));
    assert.equal(await stat(receipt.evidence.desktopScreenshot).then(() => true), true);
    assert.equal(await stat(receipt.evidence.mobileScreenshot).then(() => true), true);
    assert.equal(await stat(receipt.evidence.video).then(() => true), true);
    assert.equal(await stat(receipt.evidence.mobileVideo).then(() => true), true);
    assert.equal(await stat(receipt.evidence.contactSheet).then(value => value.size > 0), true);
    assert.equal(await stat(join(receipt.outputDir, 'checkpoint-artifacts.png')).then(() => true), true);
    assert.equal(await stat(join(receipt.outputDir, 'mobile-artifacts.png')).then(() => true), true);
    const validation = receipt.diagnostics.at(-1).validation;
    assert.equal(validation.video.frameCount > 0, true);
    const decodedVideo = receipt.diagnostics.at(-1).video;
    assert.deepEqual(decodedVideo.dimensions, { width: 1440, height: 900 });
    assert.equal(decodedVideo.sampledDimensions.width <= 320, true);
    const mobileDiagnostic = receipt.diagnostics.find((entry) => entry.phase === 'mobile');
    assert.equal(mobileDiagnostic.validation.ok, true);
    const mobileVideo = receipt.diagnostics.find(entry => entry.phase === 'mobile-video');
    assert.equal(mobileVideo.validation.ok, true);
    assert.equal(mobileVideo.correlation.ok, true);
    assert.equal(mobileVideo.dimensions.ok, true);
    assert.equal(receipt.diagnostics.at(-1).correlation.ok, true);
    assert.equal(receipt.diagnostics.find(entry => entry.phase === 'mobile-journey').checkpoints.length, 5);

    // This is a real decode of a generated near-black MP4, not a spoofed
    // stats object. It must fail the same validator used by the capture path.
    const nearBlack = join(fixtureDir, 'near-black.mp4');
    await execFile('ffmpeg', ['-y', '-v', 'error', '-f', 'lavfi', '-i', 'color=c=#050505:s=64x64:r=4:d=1', '-c:v', 'libx264', '-pix_fmt', 'yuv420p', nearBlack]);
    const decoded = await decodeVideoFrames(nearBlack, 4);
    assert.equal(decoded.decoderError, undefined, decoded.decoderError);
    const negative = validateVideoFrames({ frames: decoded.frames, fps: 4 });
    assert.equal(negative.ok, false);
    assert.ok(negative.reasons.some((reason) => reason.code === 'sustained-blank'));

    // This bad fixture has the requested variant only in the URL query. A
    // real browser run must reject it because the rendered page did not expose
    // the variant identity.
    const badReceipt = await captureProductionEvidence({
      baseUrl,
      outputDir: join(fixtureDir, 'bad-evidence'),
      variant: 'fixture',
      route: 'bad-variant.html',
      detailRoute: 'detail.html',
      revision: 'fixture-rev-1',
      width: 900,
      height: 700,
      dpr: 1,
      timeoutMs: 15000,
      settleMs: 150,
      recordMs: 500,
      sampleFps: 4,
      checkpoints,
      checkpointSelectors,
    });
    assert.equal(badReceipt.status, 'failed');
    assert.ok(badReceipt.errors.some((reason) => reason.code === 'variant-mismatch'));

    const unavailable = await captureProductionEvidence({ baseUrl, outputDir: join(fixtureDir, 'unavailable-evidence'),
      variant: 'fixture', route: 'unavailable.html', detailRoute: 'detail.html', revision: 'fixture-rev-1',
      width: 900, height: 700, dpr: 1, timeoutMs: 15000, settleMs: 150, recordMs: 500, sampleFps: 4,
      checkpoints, checkpointSelectors });
    assert.equal(unavailable.status, 'failed');
    assert.ok(unavailable.errors.some(reason => reason.code === 'page-preflight-failure'));
    assert.equal(await stat(unavailable.evidence.preflightFailure).then(() => true), true);

    const failedFont = await captureProductionEvidence({ baseUrl, outputDir: join(fixtureDir, 'failed-font-evidence'),
      variant: 'fixture', route: 'failed-font.html', detailRoute: 'detail.html', revision: 'fixture-rev-1',
      width: 900, height: 700, dpr: 1, timeoutMs: 15000, settleMs: 150, recordMs: 500, sampleFps: 4,
      checkpoints, checkpointSelectors });
    assert.equal(failedFont.status, 'failed');
    assert.ok(failedFont.errors.some(reason => /Required presentation font/.test(reason.message)), JSON.stringify(failedFont.errors));

    const deadlineStarted = Date.now();
    const deadline = await captureProductionEvidence({ baseUrl, outputDir: join(fixtureDir, 'deadline-evidence'),
      variant: 'fixture', route: 'index.html', detailRoute: 'detail.html', revision: 'fixture-rev-1',
      width: 900, height: 700, dpr: 1, timeoutMs: 10000, settleMs: 150, recordMs: 2000, maxCaptureMs: 500, sampleFps: 4,
      checkpoints, checkpointSelectors });
    assert.equal(deadline.status, 'failed');
    assert.ok(deadline.errors.some(reason => reason.code === 'capture-deadline'), JSON.stringify(deadline.errors));
    assert.ok(Date.now() - deadlineStarted < 6000, 'outer deadline must stop and clean up the browser journey');
  } finally {
    if (previousPlaywrightModule === undefined) delete process.env.MOCKUP_PLAYWRIGHT_MODULE;
    else process.env.MOCKUP_PLAYWRIGHT_MODULE = previousPlaywrightModule;
    await rm(fixtureDir, { recursive: true, force: true });
  }
});
