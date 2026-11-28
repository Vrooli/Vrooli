import { mkdir, readFile, readdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';

export const TIMESTAMPS_MS = [0, 250, 1000, 2500, 5000, 10000, 14000];
export const VIEWPORT = { width: 1600, height: 900 };
const TARGET_TEST_ID = 'scene-canvas-pixels';
const SCENE_SETTLE_MS = 250;

function safeOrigin(raw) {
  const url = new URL(raw);
  if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || url.pathname !== '/' || url.search || url.hash) {
    throw new Error('COMMAND_CENTER_UI_URL must be an HTTP(S) origin');
  }
  return url.origin;
}

function screenshotBytes(screenshot) {
  if (!screenshot || screenshot.media_type !== 'image/png' || typeof screenshot.data !== 'string') {
    throw new Error('BAS target screenshot did not return PNG evidence');
  }
  const bytes = Buffer.from(screenshot.data, 'base64');
  if (bytes.length < 8 || !bytes.subarray(0, 8).equals(Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]))) {
    throw new Error('BAS target screenshot is not a valid PNG payload');
  }
  return bytes;
}

export async function verifyVisualBaseline({ brokerClient, baseURL, configRoot, baselineRoot, accept, png }) {
  if (!brokerClient || typeof brokerClient.openSession !== 'function' || typeof brokerClient.runTask !== 'function' || typeof brokerClient.closeSession !== 'function') {
    throw new Error('a BAS Agent Access client is required; local browser fallback is unavailable');
  }
  const origin = safeOrigin(baseURL);
  const rooms = (await readdir(configRoot)).filter(file => file.endsWith('.json')).map(file => file.slice(0, -5));
  const manifest = { seed: 'command-center-milestone-1', timestampsMs: TIMESTAMPS_MS, rooms, viewport: VIEWPORT };
  const differences = [];
  const opened = await brokerClient.openSession({ startUrl: origin + '/', viewportWidth: VIEWPORT.width, viewportHeight: VIEWPORT.height });
  const sessionId = opened?.sessionId;
  if (!sessionId) throw new Error('BAS did not return a session ID');
  try {
    for (const room of rooms) {
      for (const timestamp of TIMESTAMPS_MS) {
        const targetURL = `${origin}/${room}?visualDiff=1&visualDiffAt=${timestamp}&samples=hide&seed=${manifest.seed}`;
        await brokerClient.runTask(sessionId, { operation: 'navigate', url: targetURL });
        const observed = await brokerClient.runTask(sessionId, {
          operation: 'screenshot',
          test_id: TARGET_TEST_ID,
          wait_until: 'networkidle',
          settle_ms: SCENE_SETTLE_MS,
        });
        if (observed?.url !== targetURL) throw new Error(`BAS session left the expected visual room route for ${room}/${timestamp}`);
        const actual = screenshotBytes(observed?.screenshot);
        if (accept) {
          const path = join(baselineRoot, room, `${timestamp}.png`);
          await mkdir(join(baselineRoot, room), { recursive: true });
          await writeFile(path, actual);
          continue;
        }
        let expected;
        try { expected = await readFile(join(baselineRoot, room, `${timestamp}.png`)); }
        catch { differences.push(`${room}/${timestamp}: missing baseline`); continue; }
        const actualPNG = png.sync.read(actual); const expectedPNG = png.sync.read(expected);
        if (actualPNG.width !== expectedPNG.width || actualPNG.height !== expectedPNG.height || actualPNG.data.length !== expectedPNG.data.length) {
          differences.push(`${room}/${timestamp}: dimensions differ`); continue;
        }
        let differentPixels = 0;
        for (let i = 0; i < actualPNG.data.length; i += 4) {
          if (actualPNG.data[i] !== expectedPNG.data[i] || actualPNG.data[i + 1] !== expectedPNG.data[i + 1] || actualPNG.data[i + 2] !== expectedPNG.data[i + 2]) differentPixels++;
        }
        if (differentPixels) differences.push(`${room}/${timestamp}: ${differentPixels} pixels differ`);
      }
    }
    if (accept) await writeFile(join(baselineRoot, 'baseline.json'), JSON.stringify(manifest, null, 2) + '\n');
  } finally {
    await brokerClient.closeSession(sessionId);
  }
  if (differences.length) throw new Error(differences.join('\n'));
  return `${accept ? 'Captured' : 'Verified'} zero-diff baseline for ${rooms.length} rooms × ${TIMESTAMPS_MS.length} timestamps.`;
}

export { safeOrigin, screenshotBytes };
