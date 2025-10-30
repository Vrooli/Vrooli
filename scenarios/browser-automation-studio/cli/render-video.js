#!/usr/bin/env node

'use strict';

const fs = require('node:fs');
const fsp = require('node:fs/promises');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { URL } = require('node:url');
const os = require('node:os');

function parseArgs(argv) {
  const options = {
    input: '',
    output: '',
    baseUrl: '',
    overwrite: false,
    fps: 30,
    format: 'mp4',
    workDir: '',
  };

  for (let i = 0; i < argv.length; i += 1) {
    const token = argv[i];
    switch (token) {
      case '--input':
      case '-i':
        options.input = argv[++i] || '';
        break;
      case '--output':
      case '-o':
        options.output = argv[++i] || '';
        break;
      case '--base-url':
        options.baseUrl = argv[++i] || '';
        break;
      case '--overwrite':
      case '-f':
        options.overwrite = true;
        break;
      case '--fps':
        options.fps = Number(argv[++i] || options.fps);
        break;
      case '--format':
        options.format = (argv[++i] || options.format).toLowerCase();
        break;
      case '--work-dir':
        options.workDir = argv[++i] || '';
        break;
      default:
        if (token.startsWith('--input=')) {
          options.input = token.split('=')[1];
        } else if (token.startsWith('--output=')) {
          options.output = token.split('=')[1];
        } else if (token.startsWith('--base-url=')) {
          options.baseUrl = token.split('=')[1];
        } else if (token.startsWith('--fps=')) {
          options.fps = Number(token.split('=')[1]);
        } else if (token.startsWith('--format=')) {
          options.format = token.split('=')[1].toLowerCase();
        }
        break;
    }
  }

  if (!options.input) {
    throw new Error('Missing required --input <file> argument');
  }

  if (!options.output) {
    const baseName = path.basename(options.input, path.extname(options.input));
    options.output = `${baseName || 'bas-replay'}.mp4`;
  }

  if (!['mp4', 'webm'].includes(options.format)) {
    throw new Error(`Unsupported format ${options.format} (expected mp4|webm)`);
  }

  if (!Number.isFinite(options.fps) || options.fps <= 0) {
    options.fps = 30;
  }

  return options;
}

async function fileExists(targetPath) {
  try {
    await fsp.access(targetPath, fs.constants.F_OK);
    return true;
  } catch (error) {
    return false;
  }
}

function ensureFfmpeg() {
  const probe = spawnSync('ffmpeg', ['-version'], { stdio: 'ignore' });
  if (probe.error) {
    throw new Error('ffmpeg is required but was not found in PATH');
  }
}

function sanitizeId(value, fallback) {
  if (typeof value === 'string' && value.trim().length) {
    return value.trim().replace(/[^a-zA-Z0-9_-]/g, '-');
  }
  return fallback;
}

function hexToFfmpegColor(hex, alpha) {
  if (typeof hex !== 'string') {
    return `0xFFFFFF${alpha}`;
  }
  const clean = hex.replace('#', '').trim();
  if (clean.length === 3) {
    const r = clean[0];
    const g = clean[1];
    const b = clean[2];
    return `0x${r}${r}${g}${g}${b}${b}${alpha}`;
  }
  if (clean.length === 6) {
    return `0x${clean}${alpha}`;
  }
  if (clean.length === 8) {
    return `0x${clean}`;
  }
  return `0xFFFFFF${alpha}`;
}

async function downloadAsset(asset, destinationDir, index, baseUrl, assetMap) {
  if (!asset || !asset.source) {
    return null;
  }

  const idFragment = sanitizeId(asset.id, `asset-${index}`);
  let extension = '';

  const detectExtension = () => {
    if (asset.source) {
      try {
        const resolved = new URL(asset.source, baseUrl || undefined);
        const extname = path.extname(resolved.pathname || '');
        if (extname) {
          return extname;
        }
      } catch (error) {
        // ignore
      }
    }
    if (asset.content_type) {
      if (asset.content_type.includes('png')) return '.png';
      if (asset.content_type.includes('jpeg')) return '.jpg';
      if (asset.content_type.includes('webp')) return '.webp';
      if (asset.content_type.includes('json')) return '.json';
    }
    return '.bin';
  };

  extension = detectExtension();
  const fileName = `${idFragment}${extension}`;
  const fullPath = path.join(destinationDir, fileName);

  if (await fileExists(fullPath)) {
    assetMap.set(asset.id, fullPath);
    return fullPath;
  }

  if (!baseUrl) {
    throw new Error(`Cannot resolve asset ${asset.source} without --base-url`);
  }

  const resolved = new URL(asset.source, baseUrl || undefined);
  const response = await fetch(resolved);
  if (!response.ok) {
    throw new Error(`Failed to download asset ${resolved} (${response.status})`);
  }
  const buffer = Buffer.from(await response.arrayBuffer());
  await fsp.writeFile(fullPath, buffer);
  assetMap.set(asset.id, fullPath);
  return fullPath;
}

function clampCoordinate(value, limit) {
  if (!Number.isFinite(value)) {
    return 0;
  }
  if (value < 0) return 0;
  if (value > limit) return limit;
  return value;
}

function buildFrameFilter(frame, viewport, options = {}) {
  const filters = [];

  const overlayBox = (box, color, opacity, border) => {
    if (!box) return;
    const x = Number(box.x || box.X || 0);
    const y = Number(box.y || box.Y || 0);
    const w = Number(box.width || box.Width || 0);
    const h = Number(box.height || box.Height || 0);
    if (!Number.isFinite(x) || !Number.isFinite(y) || !Number.isFinite(w) || !Number.isFinite(h) || w <= 0 || h <= 0) {
      return;
    }
    const alphaHex = Math.round(opacity * 255).toString(16).padStart(2, '0');
    const fillColor = hexToFfmpegColor(color, alphaHex);
    filters.push(`drawbox=x=${x}:y=${y}:w=${w}:h=${h}:color=${fillColor}:t=fill`);
    if (border) {
      const borderColor = hexToFfmpegColor(color, 'FF');
      filters.push(`drawbox=x=${x}:y=${y}:w=${w}:h=${h}:color=${borderColor}:t=${border}`);
    }
  };

  if (Array.isArray(frame.maskRegions)) {
    for (const region of frame.maskRegions) {
      overlayBox(region.boundingBox, region.color || '#0F172A', Number.isFinite(region.opacity) ? region.opacity : 0.45, null);
    }
  }

  if (Array.isArray(frame.highlightRegions)) {
    for (const region of frame.highlightRegions) {
      overlayBox(region.boundingBox, region.color || '#38BDF8', 0.18, 3);
    }
  }

  if (frame.focusedElement && frame.focusedElement.boundingBox) {
    overlayBox(frame.focusedElement.boundingBox, '#38BDF8', 0.0, 4);
  }

  if (frame.clickPosition && Number.isFinite(frame.clickPosition.x) && Number.isFinite(frame.clickPosition.y)) {
    const size = Math.max(Math.round(Math.min(viewport.width, viewport.height) * 0.02), 8);
    const x = Math.max(frame.clickPosition.x - size / 2, 0);
    const y = Math.max(frame.clickPosition.y - size / 2, 0);
    filters.push(`drawbox=x=${x}:y=${y}:w=${size}:h=${size}:color=${hexToFfmpegColor('#FFFFFF', 'AA')}:t=fill`);
    filters.push(`drawbox=x=${x}:y=${y}:w=${size}:h=${size}:color=${hexToFfmpegColor('#38BDF8', 'FF')}:t=2`);
  }

  const cursorTrail = Array.isArray(frame.cursorTrail) ? frame.cursorTrail : [];
  if (cursorTrail.length > 0 && options.drawTrail !== false) {
    const trailColor = options.trailColor || '#38BDF8';
    const opacityHex = hexToFfmpegColor(trailColor, '66');
    for (const point of cursorTrail) {
      if (!Number.isFinite(point.x) || !Number.isFinite(point.y)) continue;
      const size = 4;
      const x = Math.max(point.x - size / 2, 0);
      const y = Math.max(point.y - size / 2, 0);
      filters.push(`drawbox=x=${x}:y=${y}:w=${size}:h=${size}:color=${opacityHex}:t=fill`);
    }
  }

  const cursorPoint = options.cursorPoint;
  if (cursorPoint && Number.isFinite(cursorPoint.x) && Number.isFinite(cursorPoint.y)) {
    const baseSize = Math.max(Math.round(Math.min(viewport.width, viewport.height) * 0.02), 12);
    const clampedX = clampCoordinate(cursorPoint.x, viewport.width);
    const clampedY = clampCoordinate(cursorPoint.y, viewport.height);
    const x = Math.max(clampedX - baseSize / 2, 0);
    const y = Math.max(clampedY - baseSize / 2, 0);
    const cursorColor = options.cursorColor || '#38BDF8';
    filters.push(`drawbox=x=${x}:y=${y}:w=${baseSize}:h=${baseSize}:color=${hexToFfmpegColor('#FFFFFF', 'AA')}:t=fill`);
    filters.push(`drawbox=x=${x}:y=${y}:w=${baseSize}:h=${baseSize}:color=${hexToFfmpegColor(cursorColor, 'FF')}:t=2`);
  } else if (frame.cursor && Number.isFinite(frame.cursor.x) && Number.isFinite(frame.cursor.y)) {
    const fallbackSize = Math.max(Math.round(Math.min(viewport.width, viewport.height) * 0.018), 10);
    const x = Math.max(frame.cursor.x - fallbackSize / 2, 0);
    const y = Math.max(frame.cursor.y - fallbackSize / 2, 0);
    filters.push(`drawbox=x=${x}:y=${y}:w=${fallbackSize}:h=${fallbackSize}:color=${hexToFfmpegColor('#FFFFFF', 'AA')}:t=fill`);
    filters.push(`drawbox=x=${x}:y=${y}:w=${fallbackSize}:h=${fallbackSize}:color=${hexToFfmpegColor('#38BDF8', 'FF')}:t=2`);
  }

  if (filters.length === 0) {
    return null;
  }
  return filters.join(',');
}

function frameDurations(frame) {
  const rawDuration = Number(frame.totalDurationMs || frame.durationMs || 0);
  const motionDuration = Math.max(Number.isFinite(rawDuration) && rawDuration > 0 ? rawDuration : 800, 400);
  const holdRaw = Number(frame.holdMs);
  const holdDuration = Number.isFinite(holdRaw) && holdRaw > 0 ? holdRaw : 650;
  return {
    motion: motionDuration,
    hold: holdDuration,
    total: motionDuration + holdDuration,
  };
}

function interpolateTrail(trail, ratio) {
  if (!Array.isArray(trail) || trail.length === 0) {
    return null;
  }
  if (trail.length === 1 || ratio <= 0) {
    return trail[0];
  }
  if (ratio >= 1) {
    return trail[trail.length - 1];
  }

  const scaled = ratio * (trail.length - 1);
  const index = Math.floor(scaled);
  const segmentT = scaled - index;
  const start = trail[index];
  const end = trail[index + 1];
  if (!start || !end) {
    return trail[trail.length - 1];
  }

  const lerp = (a, b) => {
    if (!Number.isFinite(a) || !Number.isFinite(b)) {
      return Number.isFinite(a) ? a : b;
    }
    return a + (b - a) * segmentT;
  };

  return {
    x: lerp(start.x, end.x),
    y: lerp(start.y, end.y),
  };
}

function buildCursorSegments(frame, viewport) {
  const { motion, hold, total } = frameDurations(frame);
  const trail = Array.isArray(frame.cursorTrail)
    ? frame.cursorTrail
        .map((point) => ({
          x: clampCoordinate(Number(point.x), viewport.width),
          y: clampCoordinate(Number(point.y), viewport.height),
        }))
        .filter((point) => Number.isFinite(point.x) && Number.isFinite(point.y))
    : [];

  const fallbackPoint = (() => {
    if (trail.length > 0) {
      return trail[trail.length - 1];
    }
    const cursor = frame.cursor || frame.clickPosition;
    if (cursor && Number.isFinite(cursor.x) && Number.isFinite(cursor.y)) {
      return {
        x: clampCoordinate(cursor.x, viewport.width),
        y: clampCoordinate(cursor.y, viewport.height),
      };
    }
    return null;
  })();

  if (trail.length <= 1) {
    return [
      {
        cursorPoint: fallbackPoint,
        durationMs: total,
        drawTrail: trail.length > 0,
      },
    ];
  }

  const motionSegments = Math.min(45, Math.max(trail.length * 6, Math.round(motion / 90)));
  const motionDurationPerSegment = motion / motionSegments;
  const segments = [];

  for (let i = 0; i < motionSegments; i += 1) {
    const ratio = motionSegments === 1 ? 1 : i / (motionSegments - 1);
    const cursorPoint = interpolateTrail(trail, ratio);
    segments.push({
      cursorPoint,
      durationMs: Math.max(Math.round(motionDurationPerSegment), 40),
      drawTrail: true,
    });
  }

  if (hold > 0) {
    const holdSegments = Math.min(8, Math.max(1, Math.round(hold / 200)));
    const perHoldDuration = hold / holdSegments;
    const finalPoint = trail[trail.length - 1] || fallbackPoint;
    for (let i = 0; i < holdSegments; i += 1) {
      segments.push({
        cursorPoint: finalPoint,
        durationMs: Math.max(Math.round(perHoldDuration), 40),
        drawTrail: i === 0,
      });
    }
  }

  return segments;
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  ensureFfmpeg();

  const inputPath = path.resolve(process.cwd(), options.input);
  if (!(await fileExists(inputPath))) {
    throw new Error(`Input file not found: ${inputPath}`);
  }

  if (!options.overwrite && await fileExists(path.resolve(process.cwd(), options.output))) {
    throw new Error(`Output file already exists: ${options.output} (use --overwrite)`);
  }

  const raw = await fsp.readFile(inputPath, 'utf8');
  const parsed = JSON.parse(raw);
  const exportPackage = parsed.package || parsed;
  if (!exportPackage || !Array.isArray(exportPackage.frames) || exportPackage.frames.length === 0) {
    throw new Error('Export package missing frames array');
  }

  const workRoot = options.workDir
    ? path.resolve(process.cwd(), options.workDir)
    : await fsp.mkdtemp(path.join(os.tmpdir(), 'bas-video-'));
  await fsp.mkdir(workRoot, { recursive: true });

  const assetsDir = path.join(workRoot, 'assets');
  await fsp.mkdir(assetsDir, { recursive: true });
  const frameDir = path.join(workRoot, 'frames');
  await fsp.mkdir(frameDir, { recursive: true });

  const assetPaths = new Map();
  if (Array.isArray(exportPackage.assets)) {
    let idx = 0;
    for (const asset of exportPackage.assets) {
      idx += 1;
      try {
        await downloadAsset(asset, assetsDir, idx, options.baseUrl, assetPaths);
      } catch (error) {
        console.warn(`[render-video] Failed to download asset ${asset?.id || idx}: ${error.message}`);
      }
    }
  }

  const frames = exportPackage.frames.map((frame, index) => ({ frame, index }));
  const preparedFrames = [];

  for (const { frame, index } of frames) {
    const viewport = frame.viewport || { width: 1920, height: 1080 };
    let screenshotPath = null;
    if (frame.screenshotAssetID && assetPaths.has(frame.screenshotAssetID)) {
      screenshotPath = assetPaths.get(frame.screenshotAssetID);
    } else if (frame.screenshot && typeof frame.screenshot === 'string' && await fileExists(frame.screenshot)) {
      screenshotPath = frame.screenshot;
    }

    if (!screenshotPath) {
      console.warn(`[render-video] Frame ${index} missing screenshot asset; skipping`);
      continue;
    }

    const segments = buildCursorSegments(frame, viewport);
    const baseName = `frame-${String(index).padStart(4, '0')}`;

    for (let segmentIndex = 0; segmentIndex < segments.length; segmentIndex += 1) {
      const segment = segments[segmentIndex];
      const segmentName = segments.length === 1
        ? `${baseName}.png`
        : `${baseName}-${String(segmentIndex).padStart(3, '0')}.png`;
      const segmentPath = path.join(frameDir, segmentName);

      const filter = buildFrameFilter(frame, viewport, {
        cursorPoint: segment.cursorPoint,
        drawTrail: segment.drawTrail,
      });

      const args = ['-y', '-i', screenshotPath];
      if (filter) {
        args.push('-vf', filter);
      }
      args.push('-frames:v', '1', segmentPath);

      const result = spawnSync('ffmpeg', args, { stdio: 'inherit' });
      if (result.status !== 0) {
        throw new Error(`ffmpeg failed while preparing frame ${index} segment ${segmentIndex}`);
      }

      preparedFrames.push({
        path: segmentPath,
        durationMs: Math.max(segment.durationMs, 40),
      });
    }
  }

  const concatListPath = path.join(workRoot, 'frames.txt');
  const concatLines = [];
  if (preparedFrames.length === 0) {
    throw new Error('No frames were generated; aborting video render');
  }

  const escapePath = (value) => value.replace(/'/g, "'\\''");

  for (const segment of preparedFrames) {
    concatLines.push(`file '${escapePath(segment.path)}'`);
    const durationSeconds = Math.max(segment.durationMs / 1000, 0.1);
    concatLines.push(`duration ${durationSeconds.toFixed(3)}`);
  }
  const lastFramePath = preparedFrames[preparedFrames.length - 1].path;
  concatLines.push(`file '${escapePath(lastFramePath)}'`);

  await fsp.writeFile(concatListPath, concatLines.join('\n'), 'utf8');

  const outputPath = path.resolve(process.cwd(), options.output);
  const videoArgs = [
    '-y',
    '-f', 'concat',
    '-safe', '0',
    '-i', concatListPath,
    '-vsync', 'vfr',
    '-pix_fmt', options.format === 'webm' ? 'yuva420p' : 'yuv420p',
  ];

  if (options.format === 'webm') {
    videoArgs.push('-c:v', 'libvpx-vp9', '-b:v', '2M');
  } else {
    videoArgs.push('-c:v', 'libx264', '-profile:v', 'high', '-level', '4.1', '-crf', '21');
  }

  videoArgs.push(outputPath);

  const videoResult = spawnSync('ffmpeg', videoArgs, { stdio: 'inherit' });
  if (videoResult.status !== 0) {
    throw new Error('ffmpeg failed while assembling video');
  }

  console.log(`Replay video written to ${outputPath}`);

  if (!options.workDir) {
    try {
      await fsp.rm(workRoot, { recursive: true, force: true });
    } catch (cleanupError) {
      console.warn(`[render-video] Failed to clean temporary directory: ${cleanupError.message}`);
    }
  }
}

main().catch((error) => {
  console.error(`[render-video] ${error.message}`);
  process.exitCode = 1;
});
