import { createRequire } from 'node:module';
import { execFile as execFileCallback } from 'node:child_process';
import { mkdir, mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { basename, dirname, join, resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { promisify } from 'node:util';

// LP-PRES-011/012: this module owns trustworthy page-bound production evidence
// and rejection receipts. The parent presentation/API work owns the missing
// route and revision producers; this module does not invent their values.

const execFile = promisify(execFileCallback);
const require = createRequire(import.meta.url);

export const DEFAULT_LATE_BLANK_SECONDS = 0.75;
export const DEFAULT_SAMPLE_FPS = 4;
export const DEFAULT_DECODE_WIDTH = 320;

function reason(code, message, details = {}) {
  return { code, message, ...details };
}

export function rgbStats(pixels) {
  if (!pixels || pixels.length === 0 || pixels.length % 3 !== 0) {
    throw new Error('RGB frame must contain a non-empty multiple of three bytes');
  }
  let sum = 0;
  let sum2 = 0;
  let min = 255;
  let max = 0;
  let black = 0;
  let white = 0;
  const count = pixels.length / 3;
  for (let index = 0; index < pixels.length; index += 3) {
    const luminance = 0.2126 * pixels[index] + 0.7152 * pixels[index + 1] + 0.0722 * pixels[index + 2];
    sum += luminance;
    sum2 += luminance * luminance;
    min = Math.min(min, luminance);
    max = Math.max(max, luminance);
    if (luminance < 12) black += 1;
    if (luminance > 245) white += 1;
  }
  const mean = sum / count;
  const stddev = Math.sqrt(Math.max(0, sum2 / count - mean * mean));
  const range = max - min;
  const blackFraction = black / count;
  const whiteFraction = white / count;
  const uniform = (stddev < 2 && range < 12) || blackFraction >= 0.995 || whiteFraction >= 0.995;
  return {
    mean: Number(mean.toFixed(3)),
    stddev: Number(stddev.toFixed(3)),
    range: Number(range.toFixed(3)),
    blackFraction: Number(blackFraction.toFixed(4)),
    whiteFraction: Number(whiteFraction.toFixed(4)),
    uniform,
  };
}

function frameStats(frame) {
  return frame.stats ?? (frame.pixels ? rgbStats(frame.pixels) : null);
}

function pixelDifference(left, right) {
  if (!left || !right || left.length !== right.length || left.length === 0) return null;
  let total = 0;
  for (let index = 0; index < left.length; index += 1) total += Math.abs(left[index] - right[index]);
  return total / (left.length * 255);
}

function checkpointMotion(frames, beforeFrame, afterFrame) {
  const radius = 2;
  const beforeStart = Math.max(0, beforeFrame - radius);
  const beforeEnd = Math.min(frames.length - 1, beforeFrame + radius);
  const afterStart = Math.max(beforeEnd + 1, afterFrame - radius);
  const afterEnd = Math.min(frames.length - 1, afterFrame + radius);
  let maximum = null;
  let pair;
  for (let before = beforeStart; before <= beforeEnd; before += 1) {
    for (let after = afterStart; after <= afterEnd; after += 1) {
      const difference = pixelDifference(frames[before]?.pixels, frames[after]?.pixels);
      if (difference !== null && (maximum === null || difference > maximum)) {
        maximum = difference;
        pair = { beforeFrame: before, afterFrame: after };
      }
    }
  }
  return { motion: maximum, pair };
}

export function validateVideoFrames({
  frames,
  fps = DEFAULT_SAMPLE_FPS,
  lateBlankSeconds = DEFAULT_LATE_BLANK_SECONDS,
  plannedChanges = [],
  decoderError,
} = {}) {
  const reasons = [];
  if (decoderError) reasons.push(reason('decoder-failure', 'FFmpeg could not decode the recording', { error: String(decoderError) }));
  if (!Array.isArray(frames) || frames.length === 0) {
    reasons.push(reason('decoder-failure', 'The recording produced no decoded RGB frames'));
    return { ok: false, reasons, frameCount: 0 };
  }

  const stats = frames.map(frameStats);
  if (stats.some((value) => !value)) reasons.push(reason('decoder-failure', 'A decoded frame has no RGB pixels or stats'));

  let runStart = null;
  const blankRuns = [];
  stats.forEach((value, index) => {
    if (value?.uniform && runStart === null) runStart = index;
    const isLast = index === stats.length - 1;
    if ((!value?.uniform || isLast) && runStart !== null) {
      const end = value?.uniform && isLast ? index + 1 : index;
      blankRuns.push({ startFrame: runStart, endFrame: end - 1, seconds: (end - runStart) / fps });
      runStart = null;
    }
  });
  for (const run of blankRuns) {
    if (run.seconds >= lateBlankSeconds) {
      reasons.push(reason('sustained-blank', 'Recording contains a sustained uniform blank interval', run));
    }
  }

  for (const change of plannedChanges) {
    const before = frames[change.beforeFrame];
    const after = frames[change.afterFrame];
    const windowed = checkpointMotion(frames, change.beforeFrame, change.afterFrame);
    const motion = change.motion ?? windowed.motion ?? pixelDifference(before?.pixels, after?.pixels);
    if (change.requiresMotion !== false && (motion === null || motion < (change.minMotion ?? 0.01))) {
      reasons.push(reason('frozen-change', 'A planned page change produced no measurable frame change', {
        label: change.label,
        beforeFrame: change.beforeFrame,
        afterFrame: change.afterFrame,
        motion,
        comparedPair: windowed.pair,
      }));
    }
  }

  return {
    ok: reasons.length === 0,
    reasons,
    frameCount: frames.length,
    fps,
    blankRuns,
    stats,
  };
}

function visibleOverlay(overlay) {
  return overlay && overlay.visible !== false;
}

export function validatePageDiagnostics(actual = {}, expected = {}) {
  const reasons = [];
  if (actual.route !== expected.route) reasons.push(reason('route-mismatch', 'Captured route does not match the requested route', { expected: expected.route, actual: actual.route }));
  if (actual.variant !== expected.variant) reasons.push(reason('variant-mismatch', 'Captured variant does not match the requested variant', { expected: expected.variant, actual: actual.variant }));
  if (!actual.revision) reasons.push(reason('revision-missing', 'Captured page did not expose a presentation revision')); 
  else if (expected.revision && actual.revision !== expected.revision) reasons.push(reason('revision-mismatch', 'Captured revision does not match the requested revision', { expected: expected.revision, actual: actual.revision }));
  const expectedViewport = expected.viewport ?? {};
  const viewport = actual.viewport ?? {};
  for (const key of ['width', 'height', 'dpr']) {
    if (expectedViewport[key] !== undefined && viewport[key] !== expectedViewport[key]) {
      reasons.push(reason('viewport-mismatch', 'Captured viewport does not match the requested viewport', { field: key, expected: expectedViewport[key], actual: viewport[key] }));
      break;
    }
  }
  if (actual.ready !== true) reasons.push(reason('page-not-ready', 'Captured page did not report ready state'));
  const expectedLandmarks = expected.landmarks ?? [];
  const actualLandmarks = new Set(actual.landmarks ?? []);
  for (const landmark of expectedLandmarks) {
    if (!actualLandmarks.has(landmark)) reasons.push(reason('missing-landmark', `Required capture landmark is missing: ${landmark}`, { landmark }));
  }
  for (const overlay of (actual.overlays ?? []).filter(visibleOverlay)) {
    reasons.push(reason('blocking-overlay', 'A visible overlay may obscure the captured page', { kind: overlay.kind, text: overlay.text }));
  }
  for (const asset of actual.assets ?? []) {
    if (asset.complete !== true || !(asset.naturalWidth > 0) || !(asset.naturalHeight > 0)) {
      reasons.push(reason('asset-failure', 'A required page image did not decode', { src: asset.src, complete: asset.complete, naturalWidth: asset.naturalWidth, naturalHeight: asset.naturalHeight }));
    }
  }
  return { ok: reasons.length === 0, reasons, actual, expected };
}

export function validateCaptureEvidence({ frames, page, expected, plannedChanges = [], decoderError, fps } = {}) {
  const pageResult = validatePageDiagnostics(page, expected);
  const videoResult = validateVideoFrames({ frames, plannedChanges, decoderError, fps });
  return {
    ok: pageResult.ok && videoResult.ok,
    reasons: [...pageResult.reasons, ...videoResult.reasons],
    page: pageResult,
    video: videoResult,
  };
}

// Page/video clocks do not have a shared origin. Match real screenshot pixels
// to decoded frames in journey order instead of guessing a launch-time offset.
export function correlateCheckpoints(frames, checkpoints, maxDifference = 0.015) {
  const matches = [];
  const reasons = [];
  let startFrame = 0;
  for (const checkpoint of checkpoints) {
    let match = null;
    for (let index = startFrame; index < frames.length; index += 1) {
      const difference = pixelDifference(checkpoint.pixels, frames[index].pixels);
      if (difference !== null && difference <= maxDifference) {
        match = { label: checkpoint.label, frame: index, difference };
        startFrame = index;
        break;
      }
    }
    if (!match) reasons.push(reason('checkpoint-not-in-video', 'Checkpoint screenshot was not found in the decoded recording in journey order', { label: checkpoint.label }));
    matches.push(match);
  }
  return { ok: reasons.length === 0, matches, reasons };
}

export function validateRecordingDimensions(dimensions, viewport) {
  if (dimensions?.width !== viewport.width || dimensions?.height !== viewport.height) {
    return { ok: false, reasons: [reason('video-viewport-mismatch', 'Decoded video dimensions do not match the requested viewport', { actual: dimensions, expected: viewport })] };
  }
  return { ok: true, reasons: [] };
}

function parseJSONEnv(name, fallback) {
  if (!process.env[name]) return fallback;
  try {
    return JSON.parse(process.env[name]);
  } catch (error) {
    throw new Error(`${name} must be valid JSON: ${error.message}`);
  }
}

async function commandAvailable(command) {
  try {
    await execFile('which', [command]);
    return true;
  } catch {
    return false;
  }
}

async function requireCommands(commands) {
  for (const command of commands) {
    if (!(await commandAvailable(command))) throw new Error(`required command is missing: ${command}`);
  }
}

async function loadPlaywright() {
  const moduleName = process.env.MOCKUP_PLAYWRIGHT_MODULE || process.env.LPBS_PLAYWRIGHT_MODULE || 'playwright';
  const specifier = moduleName.startsWith('/') ? pathToFileURL(moduleName).href : moduleName;
  try {
    const loaded = await import(specifier);
    return { moduleName, playwright: loaded.default?.chromium ? loaded.default : loaded };
  } catch (importError) {
    try {
      return { moduleName, playwright: require(moduleName) };
    } catch (requireError) {
      throw new Error(`Playwright module is unavailable (${moduleName}); import failed: ${importError.message}; require failed: ${requireError.message}`);
    }
  }
}

function expectedConfig(options, routeKind = 'root') {
  const width = Number(options.width);
  const height = Number(options.height);
  const route = routeKind === 'detail' ? options.detailRoute : options.route;
  const defaultLandmarks = Object.keys(options.checkpointSelectors ?? {})
    .filter((name) => routeKind === 'detail' ? name === 'app-detail' : name !== 'app-detail');
  return {
    route: new URL(route, options.baseUrl).pathname,
    variant: options.variant,
    revision: options.revision,
    viewport: { width, height, dpr: Number(options.dpr ?? 1) },
    landmarks: options.routeLandmarks?.[routeKind] ?? defaultLandmarks,
  };
}

async function pageDiagnostics(page, checkpointSelectors) {
  return page.evaluate((selectors) => {
    const visible = (element) => {
      if (!element) return false;
      const style = getComputedStyle(element);
      const box = element.getBoundingClientRect();
      return style.display !== 'none' && style.visibility !== 'hidden' && box.width > 0 && box.height > 0;
    };
    const intersectsViewport = (element) => {
      if (!visible(element)) return false;
      const box = element.getBoundingClientRect();
      return box.right > 0 && box.bottom > 0 && box.left < window.innerWidth && box.top < window.innerHeight;
    };
    const revision = document.documentElement.dataset.presentationRevision
      || document.body.dataset.presentationRevision
      || document.querySelector('meta[name="presentation-revision"]')?.content
      || window.__LPBS_PRESENTATION_REVISION
      || '';
    const variant = document.documentElement.dataset.variantSlug
      || document.body.dataset.variantSlug
      || '';
    const overlays = [...document.querySelectorAll('dialog,[role="dialog"],[data-capture-overlay]')]
      .filter(visible)
      .map((element) => ({
        kind: element.getAttribute('data-overlay-kind') || element.getAttribute('role') || 'dialog',
        visible: true,
        text: (element.textContent || '').trim().slice(0, 240),
      }));
    const assets = [...document.images].map((image) => ({
      src: image.currentSrc || image.src,
      complete: image.complete,
      naturalWidth: image.naturalWidth,
      naturalHeight: image.naturalHeight,
    }));
    const landmarks = Object.entries(selectors)
      .filter(([, selector]) => [...document.querySelectorAll(selector)].some(intersectsViewport))
      .map(([name]) => name);
    return {
      url: window.location.href,
      route: window.location.pathname,
      variant,
      revision: String(revision),
      viewport: { width: window.innerWidth, height: window.innerHeight, dpr: window.devicePixelRatio },
      ready: document.documentElement.dataset.experienceState === 'ready'
        || document.documentElement.dataset.captureReady === 'true'
        || document.body.dataset.captureReady === 'true',
      landmarks,
      overlays,
      assets,
      title: document.title,
    };
  }, checkpointSelectors);
}

async function waitForPage(page, timeoutMs) {
  await page.waitForLoadState('networkidle', { timeout: timeoutMs });
  await page.evaluate(async (imageTimeoutMs) => {
    const bounded = (promise) => Promise.race([
      promise,
      new Promise((resolve) => setTimeout(resolve, imageTimeoutMs)),
    ]);
    if (document.fonts?.ready) await bounded(document.fonts.ready);
    const images = [...document.images];
    await Promise.all(images.map(async (image) => {
      if (!image.complete) await bounded(new Promise((resolve) => { image.addEventListener('load', resolve, { once: true }); image.addEventListener('error', resolve, { once: true }); }));
      if (image.complete && image.naturalWidth > 0 && image.decode) await bounded(image.decode().catch(() => {}));
    }));
  }, timeoutMs);
}

async function primeCaptureTargets(page, selectors, timeoutMs) {
  for (const selector of Object.values(selectors ?? {})) {
    if (!selector) continue;
    try {
      await page.locator(selector).first().scrollIntoViewIfNeeded({ timeout: timeoutMs });
    } catch {
      // pageDiagnostics reports the missing/intersection failure with the
      // stable landmark name; priming must not hide that receipt.
    }
  }
  await page.evaluate(() => window.scrollTo(0, 0));
}

async function screenshotCheckpoint(page, outputPath, timeoutMs) {
  await page.screenshot({ path: outputPath, fullPage: false, timeout: timeoutMs });
}

function diagnosticStderr(stderr) {
  const text = String(stderr || '').trim();
  return /(?:error|invalid|corrupt|decode|failed|unable)/i.test(text) ? text : '';
}

export async function decodeVideoFrames(videoPath, fps, { maxWidth = DEFAULT_DECODE_WIDTH } = {}) {
  let probe;
  try {
    probe = await execFile('ffprobe', ['-v', 'error', '-select_streams', 'v:0', '-show_entries', 'stream=width,height', '-of', 'json', videoPath], { encoding: 'utf8' });
  } catch (error) {
    return { frames: [], decoderError: `ffprobe failed: ${error.message}` };
  }
  let dimensions;
  try {
    const stream = JSON.parse(probe.stdout).streams?.[0];
    dimensions = { width: Number(stream.width), height: Number(stream.height) };
    if (!(dimensions.width > 0) || !(dimensions.height > 0)) throw new Error('video dimensions are invalid');
  } catch (error) {
    return { frames: [], decoderError: `invalid ffprobe output: ${error.message}` };
  }
  try {
    const sampleWidth = Math.min(dimensions.width, maxWidth);
    const sampleHeight = Math.max(2, Math.round((dimensions.height * sampleWidth / dimensions.width) / 2) * 2);
    const decoded = await execFile('ffmpeg', ['-v', 'error', '-xerror', '-i', videoPath, '-vf', `fps=${String(fps)},scale=${sampleWidth}:${sampleHeight}:flags=area`, '-f', 'rawvideo', '-pix_fmt', 'rgb24', 'pipe:1'], { encoding: 'buffer', maxBuffer: 64 * 1024 * 1024 });
    const stderrError = diagnosticStderr(decoded.stderr);
    if (stderrError) return { frames: [], dimensions, sampledDimensions: { width: sampleWidth, height: sampleHeight }, decoderError: `ffmpeg reported a decoder error: ${stderrError}` };
    const frameSize = sampleWidth * sampleHeight * 3;
    if (decoded.stdout.length < frameSize || decoded.stdout.length % frameSize !== 0) throw new Error(`decoded byte count ${decoded.stdout.length} is not a complete frame sequence`);
    const frames = [];
    for (let offset = 0; offset < decoded.stdout.length; offset += frameSize) frames.push({ pixels: decoded.stdout.subarray(offset, offset + frameSize) });
    return { frames, dimensions, sampledDimensions: { width: sampleWidth, height: sampleHeight } };
  } catch (error) {
    return { frames: [], decoderError: `ffmpeg decode failed: ${error.message}` };
  }
}

function captureUrl(baseUrl, route, variant) {
  const url = new URL(route, baseUrl);
  url.searchParams.set('variant_slug', variant);
  return url.toString();
}

async function writeReceipt(outputDir, receipt) {
  await mkdir(outputDir, { recursive: true });
  const receiptPath = join(outputDir, 'capture-receipt.json');
  await writeFile(receiptPath, `${JSON.stringify(receipt, null, 2)}\n`, 'utf8');
  return receiptPath;
}

export async function allocateOutputDir(requestedOutputDir) {
  const requested = resolve(requestedOutputDir);
  await mkdir(dirname(requested), { recursive: true });
  try {
    // Exclusive creation: even a pre-existing empty directory belongs to its
    // earlier caller. Concurrent captures never share an output namespace.
    await mkdir(requested);
    return requested;
  } catch (error) {
    if (error.code !== 'EEXIST') throw error;
    return mkdtemp(join(dirname(requested), `${basename(requested)}-`));
  }
}

async function decodeScreenshotPixels(path, dimensions) {
  const { width, height } = dimensions;
  const decoded = await execFile('ffmpeg', ['-v', 'error', '-xerror', '-i', path, '-frames:v', '1', '-vf', `scale=${width}:${height}:flags=area`, '-f', 'rawvideo', '-pix_fmt', 'rgb24', 'pipe:1'], { encoding: 'buffer', maxBuffer: 4 * 1024 * 1024, timeout: 30000 });
  if (decoded.stdout.length !== width * height * 3 || diagnosticStderr(decoded.stderr)) throw new Error('Checkpoint screenshot did not decode to the expected RGB dimensions');
  return decoded.stdout;
}

export async function captureProductionEvidence(options) {
  const startedAt = new Date().toISOString();
  const requestedOutputDir = resolve(options.outputDir);
  const outputDir = await allocateOutputDir(requestedOutputDir);
  const receipt = {
    schemaVersion: 1,
    status: 'failed',
    startedAt,
    requested: options,
    outputDir,
    requestedOutputDir,
    diagnostics: [],
    errors: [],
    evidence: {},
  };
  let context;
  let mobileProfile;
  let mobileContext;
  let tempProfile;
  try {
    await requireCommands(['ffmpeg', 'ffprobe']);
    if (!options.revision) throw new Error('expected presentation revision is required (--revision or LPBS_EXPECTED_REVISION)');
    if (!options.checkpoints?.hero) throw new Error('a hero checkpoint is required');
    const { moduleName, playwright } = await loadPlaywright();
    receipt.playwrightModule = moduleName;
    tempProfile = await mkdtemp(join(tmpdir(), 'lpbs-capture-profile-'));
    const videoDir = join(outputDir, 'video');
    await mkdir(videoDir, { recursive: true });
    const viewport = { width: Number(options.width), height: Number(options.height) };
    const captureStarted = Date.now();
    context = await playwright.chromium.launchPersistentContext(tempProfile, {
      headless: options.headless ?? true,
      executablePath: options.executablePath || '/usr/bin/google-chrome',
      viewport,
      deviceScaleFactor: Number(options.dpr ?? 1),
      reducedMotion: 'reduce',
      recordVideo: { dir: videoDir, size: viewport },
      args: ['--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--no-first-run', '--no-default-browser-check', '--test-type'],
    });
    const page = context.pages()[0] || await context.newPage();
    const url = captureUrl(options.baseUrl, options.route, options.variant);
    const checkpointSelectors = options.checkpointSelectors;
    const expected = expectedConfig(options, 'root');
    const checkpoints = [];
    const checkpoint = async (label, selector, requiresMotion = false) => {
      if (selector) {
        try { await page.locator(selector).scrollIntoViewIfNeeded({ timeout: options.timeoutMs }); }
        catch (error) { receipt.errors.push({ code: 'checkpoint-selector', label, selector, error: error.message }); }
      }
      await page.waitForTimeout(options.settleMs);
      const elapsedSeconds = (Date.now() - captureStarted) / 1000;
      const diagnostic = await pageDiagnostics(page, checkpointSelectors);
      const checkpointPath = join(outputDir, `checkpoint-${label}.png`);
      await screenshotCheckpoint(page, checkpointPath, options.timeoutMs);
      checkpoints.push({ label, elapsedSeconds, selector, diagnostic, screenshot: basename(checkpointPath), requiresMotion });
    };

    await page.goto(url, { waitUntil: 'domcontentloaded', timeout: options.timeoutMs });
    await primeCaptureTargets(page, checkpointSelectors, options.timeoutMs);
    await waitForPage(page, options.timeoutMs);
    receipt.diagnostics.push({ phase: 'initial', diagnostic: await pageDiagnostics(page, checkpointSelectors) });
    const rootJourney = ['hero', 'product', 'catalog', 'closing'].filter(label => options.checkpoints[label]);
    for (const label of rootJourney) await checkpoint(label, options.checkpoints[label], label !== 'hero');
    if (options.detailRoute) {
      await page.goto(captureUrl(options.baseUrl, options.detailRoute, options.variant), { waitUntil: 'domcontentloaded', timeout: options.timeoutMs });
      await primeCaptureTargets(page, { 'app-detail': options.checkpoints.detail }, options.timeoutMs);
      await waitForPage(page, options.timeoutMs);
      // Route navigation is proven by the exact route diagnostic below. The
      // scroll journey owns the motion requirement; detail pages may be
      // visually static after navigation and must still be recordable.
      await checkpoint('app-detail', options.checkpoints.detail, false);
    } else {
      receipt.errors.push({ code: 'detail-route-missing', message: 'No app-detail route was supplied; no detail navigation was claimed.' });
    }
    receipt.evidence.desktopScreenshot = join(outputDir, 'public-landing-desktop.png');
    await page.screenshot({ path: receipt.evidence.desktopScreenshot, fullPage: false, timeout: options.timeoutMs });
    const videoHandle = page.video();
    await page.waitForTimeout(options.recordMs);
    await context.close();
    context = undefined;
    const webmPath = await videoHandle.path();
    receipt.evidence.webm = webmPath;
    const mp4Path = join(outputDir, 'public-landing-desktop.mp4');
    try {
      const encoded = await execFile('ffmpeg', ['-n', '-v', 'error', '-xerror', '-i', webmPath, '-c:v', 'libx264', '-preset', 'fast', '-crf', '22', '-pix_fmt', 'yuv420p', '-movflags', '+faststart', mp4Path], { encoding: 'utf8', maxBuffer: 4 * 1024 * 1024 });
      const encodeError = diagnosticStderr(encoded.stderr);
      if (encodeError) throw new Error(`ffmpeg reported an encode error: ${encodeError}`);
      receipt.evidence.video = mp4Path;
    } catch (error) {
      receipt.errors.push({ code: 'video-encode-failure', message: error.message });
    }
    const mobileViewport = { width: 390, height: 844 };
    mobileProfile = await mkdtemp(join(tmpdir(), 'lpbs-capture-mobile-'));
    mobileContext = await playwright.chromium.launchPersistentContext(mobileProfile, {
      headless: options.headless ?? true,
      executablePath: options.executablePath || '/usr/bin/google-chrome',
      viewport: mobileViewport,
      deviceScaleFactor: Number(options.dpr ?? 1),
      reducedMotion: 'reduce',
      args: ['--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--no-first-run', '--no-default-browser-check', '--test-type'],
    });
    const mobilePage = mobileContext.pages()[0] || await mobileContext.newPage();
    await mobilePage.goto(url, { waitUntil: 'domcontentloaded', timeout: options.timeoutMs });
    await primeCaptureTargets(mobilePage, checkpointSelectors, options.timeoutMs);
    await waitForPage(mobilePage, options.timeoutMs);
    const mobileDiagnostic = await pageDiagnostics(mobilePage, checkpointSelectors);
    const mobileExpected = { ...expectedConfig({ ...options, width: mobileViewport.width, height: mobileViewport.height }, 'root'), landmarks: ['hero'] };
    const mobileValidation = validatePageDiagnostics(mobileDiagnostic, mobileExpected);
    receipt.diagnostics.push({ phase: 'mobile', diagnostic: mobileDiagnostic, validation: mobileValidation });
    receipt.evidence.mobileScreenshot = join(outputDir, 'public-landing-mobile.png');
    await mobilePage.screenshot({ path: receipt.evidence.mobileScreenshot, fullPage: false, timeout: options.timeoutMs });
    const mobileCheckpoints = [];
    for (const label of rootJourney) {
      await mobilePage.locator(options.checkpoints[label]).scrollIntoViewIfNeeded({ timeout: options.timeoutMs });
      await mobilePage.waitForTimeout(options.settleMs);
      const diagnostic = await pageDiagnostics(mobilePage, checkpointSelectors);
      const expected = { ...mobileExpected, landmarks: [label] };
      const validation = validatePageDiagnostics(diagnostic, expected);
      const screenshot = join(outputDir, `mobile-${label}.png`);
      await screenshotCheckpoint(mobilePage, screenshot, options.timeoutMs);
      mobileCheckpoints.push({ label, diagnostic, validation, screenshot: basename(screenshot) });
    }
    if (options.detailRoute) {
      await mobilePage.goto(captureUrl(options.baseUrl, options.detailRoute, options.variant), { waitUntil: 'domcontentloaded', timeout: options.timeoutMs });
      await primeCaptureTargets(mobilePage, { 'app-detail': options.checkpoints.detail }, options.timeoutMs);
      await waitForPage(mobilePage, options.timeoutMs);
      await mobilePage.locator(options.checkpoints.detail).scrollIntoViewIfNeeded({ timeout: options.timeoutMs });
      const diagnostic = await pageDiagnostics(mobilePage, checkpointSelectors);
      const validation = validatePageDiagnostics(diagnostic, expectedConfig({ ...options, ...mobileViewport }, 'detail'));
      const screenshot = join(outputDir, 'mobile-app-detail.png');
      await screenshotCheckpoint(mobilePage, screenshot, options.timeoutMs);
      mobileCheckpoints.push({ label: 'app-detail', diagnostic, validation, screenshot: basename(screenshot) });
    }
    receipt.diagnostics.push({ phase: 'mobile-journey', checkpoints: mobileCheckpoints });
    receipt.errors.push(...mobileCheckpoints.flatMap(item => item.validation.reasons));
    await mobileContext.close();
    mobileContext = undefined;
    const videoResult = receipt.evidence.video ? await decodeVideoFrames(receipt.evidence.video, options.sampleFps, { maxWidth: options.decodeWidth }) : { frames: [], decoderError: 'encoded video is unavailable' };
    const checkpointPixels = [];
    if (videoResult.sampledDimensions) {
      for (const item of checkpoints) checkpointPixels.push({ label: item.label, pixels: await decodeScreenshotPixels(join(outputDir, item.screenshot), videoResult.sampledDimensions) });
    }
    const correlation = correlateCheckpoints(videoResult.frames, checkpointPixels);
    const recordingDimensions = validateRecordingDimensions(videoResult.dimensions, viewport);
    receipt.errors.push(...correlation.reasons, ...recordingDimensions.reasons);
    const plannedFrameChanges = checkpoints.flatMap((item, index) => {
      if (!item.requiresMotion || index === 0 || !correlation.matches[index - 1] || !correlation.matches[index]) return [];
      return [{ label: `${checkpoints[index - 1].label}-to-${item.label}`, beforeFrame: correlation.matches[index - 1].frame, afterFrame: correlation.matches[index].frame, requiresMotion: true }];
    });
    const checkpointValidations = checkpoints.map((item) => {
      const detail = item.label === 'app-detail' && options.detailRoute;
      const routeExpected = detail ? expectedConfig(options, 'detail') : expected;
      return validatePageDiagnostics(item.diagnostic, { ...routeExpected, landmarks: [item.label] });
    });
    const rootLandmarks = [...new Set(checkpoints
      .filter((item) => item.label !== 'app-detail')
      .flatMap((item) => item.diagnostic.landmarks ?? []))];
    const routeActual = { ...checkpoints[0]?.diagnostic, landmarks: rootLandmarks };
    const routeValidation = validatePageDiagnostics(routeActual, expected);
    const pageResult = {
      ok: routeValidation.ok && checkpointValidations.every((item) => item.ok) && mobileValidation.ok,
      reasons: [...routeValidation.reasons, ...checkpointValidations.flatMap((item) => item.reasons), ...mobileValidation.reasons],
      actual: routeActual,
      expected,
      route: routeValidation,
      checkpoints: checkpointValidations,
      mobile: mobileValidation,
    };
    const validation = validateCaptureEvidence({ frames: videoResult.frames, page: pageResult.actual, expected, plannedChanges: plannedFrameChanges, decoderError: videoResult.decoderError, fps: options.sampleFps });
    validation.reasons = [...pageResult.reasons, ...validation.reasons];
    validation.ok = pageResult.ok && validation.ok;
    receipt.diagnostics.push({ checkpoints, correlation, recordingDimensions, video: { dimensions: videoResult.dimensions, sampledDimensions: videoResult.sampledDimensions }, validation });
    receipt.errors.push(...validation.reasons);
    receipt.status = receipt.errors.length === 0 ? 'passed' : 'failed';
  } catch (error) {
    receipt.errors.push({ code: 'capture-failure', message: error.message });
  } finally {
    if (context) await context.close().catch(() => {});
    if (mobileContext) await mobileContext.close().catch(() => {});
    if (tempProfile) await rm(tempProfile, { recursive: true, force: true }).catch(() => {});
    if (mobileProfile) await rm(mobileProfile, { recursive: true, force: true }).catch(() => {});
  }
  receipt.finishedAt = new Date().toISOString();
  receipt.receiptPath = await writeReceipt(outputDir, receipt);
  return receipt;
}

function cliOptions(argv) {
  const options = {
    baseUrl: process.env.LPBS_CAPTURE_BASE_URL || 'http://127.0.0.1:23224',
    outputDir: process.env.LPBS_CAPTURE_OUTPUT_DIR || '.vrooli/artifacts/lpbs-evidence',
    variant: process.env.LPBS_CAPTURE_VARIANT || 'control',
    route: process.env.LPBS_EXPECTED_ROUTE || '/',
    detailRoute: process.env.LPBS_DETAIL_ROUTE || '',
    revision: process.env.LPBS_EXPECTED_REVISION || '',
    width: Number(process.env.LPBS_CAPTURE_WIDTH || 1440),
    height: Number(process.env.LPBS_CAPTURE_HEIGHT || 1000),
    dpr: Number(process.env.LPBS_CAPTURE_DPR || 1),
    timeoutMs: Number(process.env.LPBS_CAPTURE_TIMEOUT_MS || 30000),
    settleMs: Number(process.env.LPBS_CAPTURE_SETTLE_MS || 900),
    recordMs: Number(process.env.LPBS_CAPTURE_RECORD_MS || 1200),
    sampleFps: Number(process.env.LPBS_CAPTURE_SAMPLE_FPS || DEFAULT_SAMPLE_FPS),
    decodeWidth: Number(process.env.LPBS_CAPTURE_DECODE_WIDTH || DEFAULT_DECODE_WIDTH),
    routeLandmarks: parseJSONEnv('LPBS_CAPTURE_ROUTE_LANDMARKS_JSON', {
      root: ['hero', 'product', 'catalog', 'closing'],
      detail: ['app-detail'],
    }),
    checkpoints: parseJSONEnv('LPBS_CAPTURE_CHECKPOINTS_JSON', {
      hero: '[data-capture-landmark="hero"]',
      product: '[data-capture-landmark="product"]',
      catalog: '[data-capture-landmark="catalog"]',
      closing: '[data-capture-landmark="closing"]',
      detail: '[data-capture-landmark="app-detail"]',
    }),
    checkpointSelectors: parseJSONEnv('LPBS_CAPTURE_LANDMARK_SELECTORS_JSON', {
      hero: '[data-capture-landmark="hero"]',
      product: '[data-capture-landmark="product"]',
      catalog: '[data-capture-landmark="catalog"]',
      closing: '[data-capture-landmark="closing"]',
      'app-detail': '[data-capture-landmark="app-detail"]',
    }),
  };
  const valueFlags = new Map([
    ['--base-url', 'baseUrl'], ['--output-dir', 'outputDir'], ['--variant', 'variant'], ['--route', 'route'],
    ['--detail-route', 'detailRoute'], ['--revision', 'revision'], ['--width', 'width'], ['--height', 'height'],
    ['--dpr', 'dpr'], ['--record-ms', 'recordMs'], ['--sample-fps', 'sampleFps'], ['--decode-width', 'decodeWidth'],
  ]);
  for (let index = 0; index < argv.length; index += 1) {
    const key = argv[index];
    if (valueFlags.has(key)) {
      const target = valueFlags.get(key);
      options[target] = ['width', 'height', 'dpr', 'recordMs', 'sampleFps', 'decodeWidth'].includes(target) ? Number(argv[++index]) : argv[++index];
    }
  }
  return options;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const options = cliOptions(process.argv.slice(2));
  const result = await captureProductionEvidence(options);
  console.log(JSON.stringify({ status: result.status, receiptPath: result.receiptPath, errors: result.errors }, null, 2));
  process.exitCode = result.status === 'passed' ? 0 : 1;
}
