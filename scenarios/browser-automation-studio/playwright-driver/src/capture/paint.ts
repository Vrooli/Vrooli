/** L0 paint readiness and blank-pixel detection.
 *
 * A single-page app fires `load` before its framework commits the first
 * render. A screenshot at that moment is a flat background colour even though
 * every request succeeded (bug knw-1794710213815910554: 7 of 8 Image Tools
 * captures were blank because `#root` was still empty). Captures therefore wait
 * for visible content and a quiet DOM, and an image that is still one flat
 * colour is never reported as present pixels. */
import { inflateSync } from 'node:zlib';

export type PaintReadiness = {
  /** True when the document showed visible text or media before the deadline. */
  content: boolean;
  /** True when the DOM stopped changing for the quiet window before the deadline. */
  settled: boolean;
  waitedMs: number;
};

/** Closure-free so Playwright can serialize it into the page. */
export async function waitForPaintReadinessInPage(options: {
  timeoutMs: number;
  quietMs: number;
}): Promise<PaintReadiness> {
  const started = performance.now();
  const deadline = started + options.timeoutMs;
  const hasContent = (): boolean => {
    const body = document.body;
    if (!body) return false;
    if ((body.innerText || '').trim().length > 0) return true;
    for (const element of Array.from(body.querySelectorAll('img,svg,canvas,video,picture'))) {
      const rect = element.getBoundingClientRect();
      if (rect.width > 0 && rect.height > 0) return true;
    }
    return false;
  };
  const frame = (): Promise<void> =>
    new Promise((resolve) => {
      // rAF does not fire in a hidden page; the timer keeps the wait bounded.
      const timer = setTimeout(resolve, 50);
      requestAnimationFrame(() => {
        clearTimeout(timer);
        resolve();
      });
    });
  while (!hasContent() && performance.now() < deadline) await frame();
  const content = hasContent();
  let lastMutation = performance.now();
  const observer = new MutationObserver(() => {
    lastMutation = performance.now();
  });
  observer.observe(document.documentElement, {
    subtree: true,
    childList: true,
    attributes: true,
    characterData: true,
  });
  while (performance.now() - lastMutation < options.quietMs && performance.now() < deadline)
    await frame();
  observer.disconnect();
  const settled = performance.now() - lastMutation >= options.quietMs;
  await frame();
  await frame();
  try {
    await document.fonts.ready;
  } catch {
    /* font loading state is optional */
  }
  return { content, settled, waitedMs: Math.round(performance.now() - started) };
}

export type PixelStats = {
  width: number;
  height: number;
  /** Share of sampled pixels within a small distance of the most common colour. */
  dominantShare: number;
};

/** A capture is blank when almost every pixel is the page background. A page
 * with one short line of text on a phone still has well over 0.2% non-background
 * pixels, so 99.8% is a conservative "nothing painted" threshold. */
export const BLANK_DOMINANT_SHARE = 0.998;

export function isBlankPixels(stats: PixelStats): boolean {
  return stats.dominantShare >= BLANK_DOMINANT_SHARE;
}

/** Decodes an 8-bit, non-interlaced PNG (what Chromium screenshots emit) and
 * measures how uniform it is. Throws on any other PNG shape. */
export function measurePng(png: Buffer): PixelStats {
  const signature = '89504e470d0a1a0a';
  if (png.length < 33 || png.subarray(0, 8).toString('hex') !== signature)
    throw new Error('not a PNG image');
  let offset = 8;
  let width = 0;
  let height = 0;
  let colourType = -1;
  const idat: Buffer[] = [];
  while (offset + 8 <= png.length) {
    const length = png.readUInt32BE(offset);
    const type = png.toString('latin1', offset + 4, offset + 8);
    const data = png.subarray(offset + 8, offset + 8 + length);
    if (type === 'IHDR') {
      width = data.readUInt32BE(0);
      height = data.readUInt32BE(4);
      const bitDepth = data[8];
      colourType = data[9] ?? -1;
      if (bitDepth !== 8 || data[12] !== 0)
        throw new Error('only 8-bit non-interlaced PNG images are measured');
    } else if (type === 'IDAT') idat.push(data);
    else if (type === 'IEND') break;
    offset += 12 + length;
  }
  const channels = colourType === 6 ? 4 : colourType === 2 ? 3 : colourType === 0 ? 1 : 0;
  if (!channels || !width || !height) throw new Error('unsupported PNG colour type');
  const raw = inflateSync(Buffer.concat(idat));
  const stride = width * channels;
  const rows = Buffer.alloc(stride * height);
  let previous = Buffer.alloc(stride);
  for (let y = 0; y < height; y++) {
    const filter = raw[y * (stride + 1)];
    const line = raw.subarray(y * (stride + 1) + 1, (y + 1) * (stride + 1));
    const out = rows.subarray(y * stride, (y + 1) * stride);
    for (let x = 0; x < stride; x++) {
      const left = x >= channels ? (out[x - channels] ?? 0) : 0;
      const up = previous[x] ?? 0;
      const upLeft = x >= channels ? (previous[x - channels] ?? 0) : 0;
      const value = line[x] ?? 0;
      let predicted = 0;
      if (filter === 1) predicted = left;
      else if (filter === 2) predicted = up;
      else if (filter === 3) predicted = (left + up) >> 1;
      else if (filter === 4) {
        const p = left + up - upLeft;
        const pa = Math.abs(p - left);
        const pb = Math.abs(p - up);
        const pc = Math.abs(p - upLeft);
        predicted = pa <= pb && pa <= pc ? left : pb <= pc ? up : upLeft;
      }
      out[x] = (value + predicted) & 0xff;
    }
    previous = out;
  }
  // Sample on a grid of at most ~40k pixels; quantize to 4 bits per channel.
  const step = Math.max(1, Math.floor(Math.sqrt((width * height) / 40_000)));
  const counts = new Map<number, number>();
  let sampled = 0;
  for (let y = 0; y < height; y += step)
    for (let x = 0; x < width; x += step) {
      const i = y * stride + x * channels;
      const r = rows[i] ?? 0;
      const g = channels >= 3 ? (rows[i + 1] ?? 0) : r;
      const b = channels >= 3 ? (rows[i + 2] ?? 0) : r;
      const key = ((r >> 4) << 8) | ((g >> 4) << 4) | (b >> 4);
      counts.set(key, (counts.get(key) ?? 0) + 1);
      sampled++;
    }
  let dominant = 0;
  for (const count of counts.values()) dominant = Math.max(dominant, count);
  return { width, height, dominantShare: sampled ? dominant / sampled : 1 };
}
