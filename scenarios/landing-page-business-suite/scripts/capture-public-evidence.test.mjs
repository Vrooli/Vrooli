import test from 'node:test';
import assert from 'node:assert/strict';
import { execFile as execFileCallback } from 'node:child_process';
import { mkdtemp, rm, stat, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { promisify } from 'node:util';

import { allocateOutputDir, correlateCheckpoints, validateRecordingDimensions, captureProductionEvidence, decodeVideoFrames, rgbStats, validateCaptureEvidence, validatePageDiagnostics, validateVideoFrames } from './capture-public-evidence.mjs';

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
  landmarks: ['hero', 'product', 'catalog', 'closing', 'app-detail'],
  overlays: [],
  assets: [{ src: '/hero.png', complete: true, naturalWidth: 1440, naturalHeight: 720 }],
};

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
<html data-experience-state="ready"${renderedVariant ? ' data-variant-slug="fixture"' : ''} data-presentation-revision="fixture-rev-1">
  <head><meta charset="utf-8"><title>LPBS capture fixture</title><style>
    *{box-sizing:border-box}html,body{margin:0;background:#070b14;color:#f8fafc;font:20px system-ui,sans-serif}
    [data-capture-landmark]{min-height:700px;padding:80px;background:linear-gradient(135deg,#070b14,#132b40)}
    [data-capture-landmark="product"]{background:linear-gradient(135deg,#102b38,#254d4b)}
    [data-capture-landmark="catalog"]{background:linear-gradient(135deg,#2b203c,#4b3152)}
    [data-capture-landmark="closing"]{background:linear-gradient(135deg,#3a241d,#71402f)}
    [data-capture-landmark="app-detail"]{background:linear-gradient(135deg,#1c3a2f,#356b50)}
    h1{font-size:48px;max-width:720px}img{width:64px;height:64px}
  </style></head>
  <body>
    <section data-capture-landmark="hero"><img src="${fixtureImage}" alt="Fixture art"><h1>${heading}</h1><p>Same-page capture fixture; this is not production LPBS evidence.</p></section>
    <section data-capture-landmark="product"><h2>Product view</h2><p>Product checkpoint content.</p></section>
    <section data-capture-landmark="catalog"><h2>Bundle catalog</h2><p>Catalog checkpoint content.</p></section>
    <section data-capture-landmark="closing"><h2>Closing download state</h2><p>Download checkpoint content.</p></section>
    <section data-capture-landmark="app-detail"><h2>${detail ? 'App detail route' : 'App detail target'}</h2><p>App-detail checkpoint content.</p></section>
  </body>
</html>`;
  await writeFile(join(fixtureDir, 'index.html'), fixture('Controlled page-bound capture'));
  await writeFile(join(fixtureDir, 'detail.html'), fixture('Controlled app detail capture', true));
  await writeFile(join(fixtureDir, 'bad-variant.html'), fixture('Missing rendered variant', false, false));

  const baseUrl = pathToFileURL(`${fixtureDir}/`).href;
  const checkpoints = {
    hero: '[data-capture-landmark="hero"]',
    product: '[data-capture-landmark="product"]',
    catalog: '[data-capture-landmark="catalog"]',
    closing: '[data-capture-landmark="closing"]',
    detail: '[data-capture-landmark="app-detail"]',
  };
  const checkpointSelectors = {
    hero: checkpoints.hero,
    product: checkpoints.product,
    catalog: checkpoints.catalog,
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
    const validation = receipt.diagnostics.at(-1).validation;
    assert.equal(validation.video.frameCount > 0, true);
    const decodedVideo = receipt.diagnostics.at(-1).video;
    assert.deepEqual(decodedVideo.dimensions, { width: 1440, height: 900 });
    assert.equal(decodedVideo.sampledDimensions.width <= 320, true);
    const mobileDiagnostic = receipt.diagnostics.find((entry) => entry.phase === 'mobile');
    assert.equal(mobileDiagnostic.validation.ok, true);
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
  } finally {
    if (previousPlaywrightModule === undefined) delete process.env.MOCKUP_PLAYWRIGHT_MODULE;
    else process.env.MOCKUP_PLAYWRIGHT_MODULE = previousPlaywrightModule;
    await rm(fixtureDir, { recursive: true, force: true });
  }
});
