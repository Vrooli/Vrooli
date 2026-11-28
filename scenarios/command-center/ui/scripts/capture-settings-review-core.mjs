import { mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';

const SETTINGS_URL = 'http://127.0.0.1:24993/settings';
const VIEWPORTS = [390, 768];
const VIEWPORT_HEIGHT = 900;

function isSettingsRoute(raw) {
  try {
    const url = new URL(raw);
    return url.origin === 'http://127.0.0.1:24993' &&
      (url.pathname === '/settings' || url.pathname === '/settings/') &&
      !url.username && !url.password && !url.search && !url.hash;
  } catch {
    return false;
  }
}

function imageBytes(screenshot) {
  if (!screenshot || typeof screenshot.data !== 'string' || !screenshot.data) {
    throw new Error('BAS observation omitted screenshot bytes');
  }
  if (screenshot.media_type !== 'image/png') {
    throw new Error(`BAS returned ${screenshot.media_type || 'an unknown format'}; PNG evidence is required`);
  }
  const bytes = Buffer.from(screenshot.data, 'base64');
  if (bytes.length < 8 || !bytes.subarray(0, 8).equals(Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]))) {
    throw new Error('BAS screenshot is not a valid PNG payload');
  }
  return bytes;
}

export async function captureSettingsReview({ brokerClient, outputDir }) {
  if (!brokerClient || typeof brokerClient.openSettingsSession !== 'function' || typeof brokerClient.runTask !== 'function' || typeof brokerClient.closeSession !== 'function') {
    throw new Error('a BAS Agent Access client is required; local browser fallback is unavailable');
  }
  await mkdir(outputDir, { recursive: true });
  const captures = [];

  for (const viewportWidth of VIEWPORTS) {
    const opened = await brokerClient.openSettingsSession({
      viewportWidth,
      viewportHeight: VIEWPORT_HEIGHT,
    });
    const sessionId = opened?.sessionId;
    if (!sessionId) throw new Error('BAS did not return a session ID');
    try {
      const observed = await brokerClient.runTask(sessionId, 'screenshot');
      if (!isSettingsRoute(observed?.url)) {
        throw new Error('BAS session left the fixed Command Center settings route; observation rejected');
      }
      const screenshot = observed?.screenshot;
      if (screenshot?.width !== viewportWidth || screenshot?.height <= 0) {
        throw new Error(`BAS screenshot dimensions do not match the ${viewportWidth}px review viewport`);
      }
      const bytes = imageBytes(screenshot);
      const screenshotPath = join(outputDir, `settings-${viewportWidth}.png`);
      await writeFile(screenshotPath, bytes);

      const treeResponse = await brokerClient.runTask(sessionId, 'read');
      const tree = treeResponse?.tree;
      const layout = tree?.layout;
      if (!isSettingsRoute(tree?.url) && tree?.url !== 'http://127.0.0.1:24993') {
        throw new Error('BAS accessibility read left the fixed Command Center origin');
      }
      if (!layout || layout.viewportWidthPx !== viewportWidth || !Number.isSafeInteger(layout.documentWidthPx) || layout.documentWidthPx <= 0) {
        throw new Error('BAS accessibility read omitted trusted horizontal layout metrics');
      }
      if (layout.documentWidthPx > layout.viewportWidthPx) {
        throw new Error(`horizontal overflow at ${viewportWidth}px (${layout.documentWidthPx}px document width)`);
      }
      captures.push({ width: viewportWidth, path: screenshotPath });
    } finally {
      await brokerClient.closeSession(sessionId);
    }
  }

  return captures;
}

export { SETTINGS_URL, VIEWPORTS, VIEWPORT_HEIGHT };
