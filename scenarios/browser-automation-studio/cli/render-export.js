#!/usr/bin/env node

'use strict';

const fs = require('node:fs');
const fsp = require('node:fs/promises');
const path = require('node:path');
const { URL } = require('node:url');

async function fileExists(targetPath) {
  try {
    await fsp.access(targetPath, fs.constants.F_OK);
    return true;
  } catch (error) {
    return false;
  }
}

function parseArgs(argv) {
  const options = {
    input: '',
    output: '',
    baseUrl: '',
    overwrite: false,
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
        options.overwrite = true;
        break;
      default:
        if (token.startsWith('--input=')) {
          options.input = token.split('=')[1];
        } else if (token.startsWith('--output=')) {
          options.output = token.split('=')[1];
        } else if (token.startsWith('--base-url=')) {
          options.baseUrl = token.split('=')[1];
        }
        break;
    }
  }

  if (!options.input) {
    throw new Error('Missing required --input <file> argument');
  }

  if (!options.output) {
    options.output = 'bas-replay-output';
  }

  return options;
}

async function downloadAsset(asset, destinationDir, index, baseUrl) {
  if (!asset || !asset.source) {
    return null;
  }

  const idFragment = asset.id && typeof asset.id === 'string' ? asset.id : `asset-${index}`;
  const safeId = idFragment.replace(/[^a-zA-Z0-9_-]/g, '-');
  let extension = '';

  if (asset.source) {
    try {
      const resolved = new URL(asset.source, baseUrl || undefined);
      const pathname = resolved.pathname || '';
      const extname = path.extname(pathname);
      if (extname) {
        extension = extname;
      }
    } catch (error) {
      // Fallback: keep extension empty and add below
    }
  }

  if (!extension) {
    if (asset.content_type && typeof asset.content_type === 'string') {
      if (asset.content_type.includes('png')) {
        extension = '.png';
      } else if (asset.content_type.includes('jpeg')) {
        extension = '.jpg';
      } else if (asset.content_type.includes('webp')) {
        extension = '.webp';
      } else if (asset.content_type.includes('json')) {
        extension = '.json';
      }
    }
  }

  if (!extension) {
    extension = '.bin';
  }

  const filename = `${safeId}${extension}`;
  const fullPath = path.join(destinationDir, filename);

  const resolvedUrl = (() => {
    try {
      return new URL(asset.source, baseUrl || undefined).toString();
    } catch (error) {
      if (!baseUrl) {
        throw new Error(`Cannot resolve asset source ${asset.source} without --base-url`);
      }
      const joined = new URL(asset.source.replace(/^\//, ''), baseUrl.endsWith('/') ? baseUrl : `${baseUrl}/`);
      return joined.toString();
    }
  })();

  const response = await fetch(resolvedUrl);
  if (!response.ok) {
    throw new Error(`Failed to download asset ${resolvedUrl} (${response.status})`);
  }

  const buffer = Buffer.from(await response.arrayBuffer());
  await fsp.writeFile(fullPath, buffer);
  return `assets/${filename}`;
}

function normalizeBox(box, viewport) {
  if (!box || !viewport || !Number.isFinite(viewport.width) || !Number.isFinite(viewport.height)) {
    return null;
  }

  const { width, height } = viewport;
  const safeWidth = width > 0 ? width : 1;
  const safeHeight = height > 0 ? height : 1;

  return {
    left: Number.isFinite(box.x) ? box.x / safeWidth : 0,
    top: Number.isFinite(box.y) ? box.y / safeHeight : 0,
    width: Number.isFinite(box.width) ? box.width / safeWidth : 0,
    height: Number.isFinite(box.height) ? box.height / safeHeight : 0,
  };
}

function normalizePoint(point, viewport) {
  if (!point || !viewport || !Number.isFinite(viewport.width) || !Number.isFinite(viewport.height)) {
    return null;
  }

  const { width, height } = viewport;
  const safeWidth = width > 0 ? width : 1;
  const safeHeight = height > 0 ? height : 1;
  if (!Number.isFinite(point.x) || !Number.isFinite(point.y)) {
    return null;
  }

  return {
    x: point.x / safeWidth,
    y: point.y / safeHeight,
  };
}

function sanitizeTitle(title, fallback) {
  if (typeof title === 'string' && title.trim().length > 0) {
    return title.trim();
  }
  return fallback;
}

function escapeForScript(jsonString) {
  return jsonString.replace(/\u2028|\u2029/g, '')
    .replace(/</g, '\\u003c')
    .replace(/>/g, '\\u003e');
}

function buildHtmlDocument(payload) {
  const json = escapeForScript(JSON.stringify(payload));
  const theme = payload.theme || {};
  const gradient = Array.isArray(theme.backgroundGradient) && theme.backgroundGradient.length > 0
    ? `linear-gradient(135deg, ${theme.backgroundGradient.join(', ')})`
    : 'radial-gradient(circle at 20% 20%, #1e3a8a, #020617)';
  const accent = theme.accentColor || '#38bdf8';
  const surface = theme.surfaceColor || 'rgba(15, 23, 42, 0.86)';

  return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <title>Browser Automation Studio Replay</title>
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    :root {
      color-scheme: dark;
      font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    }
    body {
      margin: 0;
      min-height: 100vh;
      display: flex;
      align-items: center;
      justify-content: center;
      background: ${gradient};
      overflow: hidden;
      color: #e2e8f0;
    }
    .backdrop {
      position: absolute;
      inset: 0;
      overflow: hidden;
      z-index: 0;
    }
    .backdrop::before {
      content: '';
      position: absolute;
      width: 120vmax;
      height: 120vmax;
      background: radial-gradient(circle, rgba(56,189,248,0.18), transparent 70%);
      top: -40vmax;
      right: -20vmax;
      filter: blur(120px);
      opacity: 0.7;
    }
    .backdrop::after {
      content: '';
      position: absolute;
      width: 100vmax;
      height: 100vmax;
      background: radial-gradient(circle, rgba(59,130,246,0.18), transparent 70%);
      bottom: -30vmax;
      left: -25vmax;
      filter: blur(140px);
      opacity: 0.6;
    }
    .shell {
      position: relative;
      z-index: 1;
      width: min(1200px, 92vw);
      display: flex;
      flex-direction: column;
      gap: 20px;
      padding: 24px;
    }
    .header {
      display: flex;
      justify-content: space-between;
      flex-wrap: wrap;
      row-gap: 8px;
    }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;
      padding: 4px 12px;
      border-radius: 999px;
      background: rgba(148, 163, 184, 0.16);
      backdrop-filter: blur(6px);
    }
    .stage {
      position: relative;
      border-radius: 28px;
      background: ${surface};
      box-shadow: 0 40px 120px rgba(15, 23, 42, 0.55);
      padding: 36px;
      backdrop-filter: blur(18px);
    }
    .browser {
      position: relative;
      border-radius: 18px;
      background: #0f172a;
      overflow: hidden;
      border: 1px solid rgba(148, 163, 184, 0.14);
      box-shadow: inset 0 0 0 1px rgba(255,255,255,0.04);
    }
    .browser-bar {
      height: 44px;
      display: flex;
      align-items: center;
      padding: 0 18px;
      border-bottom: 1px solid rgba(148, 163, 184, 0.12);
      background: rgba(15, 23, 42, 0.95);
      backdrop-filter: blur(8px);
    }
    .browser-dots {
      display: flex;
      gap: 6px;
      margin-right: 12px;
    }
    .dot {
      width: 12px;
      height: 12px;
      border-radius: 999px;
    }
    .dot.red { background: #f87171; }
    .dot.amber { background: #fbbf24; }
    .dot.green { background: #34d399; }
    .address-bar {
      flex: 1;
      min-width: 0;
      background: rgba(15, 23, 42, 0.7);
      border-radius: 999px;
      padding: 7px 16px;
      font-size: 14px;
      color: rgba(226, 232, 240, 0.82);
      border: 1px solid rgba(148, 163, 184, 0.22);
    }
    .viewport {
      position: relative;
      width: 100%;
      aspect-ratio: 16 / 9;
      overflow: hidden;
      background: #020617;
    }
    #frame-image {
      width: 100%;
      height: 100%;
      object-fit: contain;
      transition: transform 1.4s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.5s ease;
      opacity: 0;
    }
    #frame-image.ready {
      opacity: 1;
    }
    .overlay-layer {
      position: absolute;
      inset: 0;
      pointer-events: none;
    }
    .highlight {
      position: absolute;
      border: 2px solid ${accent};
      border-radius: 12px;
      box-shadow: 0 0 0 6px rgba(56, 189, 248, 0.35);
      mix-blend-mode: screen;
      transition: opacity 0.3s ease;
    }
    .cursor {
      position: absolute;
      width: 36px;
      height: 36px;
      border-radius: 999px;
      border: 3px solid ${accent};
      transform: translate(-50%, -50%) scale(0.86);
      box-shadow: 0 0 30px rgba(56, 189, 248, 0.55);
      opacity: 0;
      transition: opacity 0.4s ease, transform 0.45s ease;
    }
    .cursor.visible {
      opacity: 1;
    }
    .info-panel {
      margin-top: 18px;
      display: flex;
      justify-content: space-between;
      flex-wrap: wrap;
      gap: 12px;
      font-size: 14px;
      color: rgba(226, 232, 240, 0.82);
    }
    .step-title {
      font-size: 22px;
      font-weight: 600;
      color: #f8fafc;
    }
    .status-pill {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 4px 12px;
      border-radius: 999px;
      background: rgba(59, 130, 246, 0.15);
      color: #93c5fd;
      font-size: 13px;
    }
    .status-pill.success {
      background: rgba(34, 197, 94, 0.16);
      color: #bbf7d0;
    }
    .status-pill.failure {
      background: rgba(248, 113, 113, 0.18);
      color: #fecaca;
    }
    .timeline {
      position: relative;
      width: 100%;
      height: 4px;
      background: rgba(148, 163, 184, 0.25);
      border-radius: 999px;
      overflow: hidden;
      margin-top: 16px;
    }
    .timeline-progress {
      position: absolute;
      inset: 0;
      background: linear-gradient(90deg, ${accent}, rgba(148, 163, 184, 0.45));
      transform-origin: left;
      transform: scaleX(0);
      transition: transform linear;
    }
    .meta-grid {
      display: flex;
      flex-wrap: wrap;
      gap: 14px 24px;
      margin-top: 18px;
      font-size: 13px;
    }
    .meta-label {
      text-transform: uppercase;
      letter-spacing: 0.12em;
      font-size: 11px;
      color: rgba(226, 232, 240, 0.56);
    }
    .meta-value {
      margin-top: 4px;
      color: rgba(226, 232, 240, 0.86);
    }
    @media (max-width: 900px) {
      .stage { padding: 18px; border-radius: 20px; }
      .browser { border-radius: 16px; }
      .viewport { aspect-ratio: 4 / 3; }
    }
  </style>
</head>
<body>
  <div class="backdrop"></div>
  <div class="shell">
    <div class="header">
      <div>
        <div class="badge">Automation Replay</div>
        <div style="margin-top:6px;font-size:28px;font-weight:600;">${sanitizeTitle(payload.execution?.workflowName, 'Automation Workflow')}</div>
      </div>
      <div class="badge" id="execution-summary">${payload.execution?.executionId || ''}</div>
    </div>
    <div class="stage">
      <div class="browser">
        <div class="browser-bar">
          <div class="browser-dots">
            <span class="dot red"></span>
            <span class="dot amber"></span>
            <span class="dot green"></span>
          </div>
          <div class="address-bar" id="address-bar">Loading replay...</div>
        </div>
        <div class="viewport" id="viewport">
          <img id="frame-image" alt="Replay frame" />
          <div class="overlay-layer" id="highlight-layer"></div>
          <div class="cursor" id="cursor"></div>
          <div class="timeline"><div class="timeline-progress" id="timeline-progress"></div></div>
        </div>
      </div>
      <div class="info-panel">
        <div>
          <div class="step-title" id="step-title"></div>
          <div style="margin-top:6px; color: rgba(226,232,240,0.7);" id="step-description"></div>
        </div>
        <div class="status-pill" id="step-status">PREPARING</div>
      </div>
      <div class="meta-grid" id="meta-grid"></div>
    </div>
  </div>
  <script>
    window.__BAS_REPLAY__ = ${json};
    (function initReplay() {
      const data = window.__BAS_REPLAY__;
      const frames = Array.isArray(data.frames) ? data.frames : [];
      if (!frames.length) {
        document.getElementById('step-title').textContent = 'No frames available';
        return;
      }

      const img = document.getElementById('frame-image');
      const highlightLayer = document.getElementById('highlight-layer');
      const cursor = document.getElementById('cursor');
      const titleEl = document.getElementById('step-title');
      const descEl = document.getElementById('step-description');
      const statusEl = document.getElementById('step-status');
      const addressBar = document.getElementById('address-bar');
      const metaGrid = document.getElementById('meta-grid');
      const timelineProgress = document.getElementById('timeline-progress');
      const executionSummary = document.getElementById('execution-summary');

      if (data.execution && data.execution.executionId) {
        executionSummary.textContent = data.execution.executionId;
      }

      const defaultHold = 900;
      let playhead = 0;
      let timer = null;

      function renderMeta(frame) {
        metaGrid.innerHTML = '';
        const entries = [];
        if (frame.finalUrl) {
          entries.push(['URL', frame.finalUrl]);
        }
        if (frame.durationLabel) {
          entries.push(['Duration', frame.durationLabel]);
        }
        if (frame.retryLabel) {
          entries.push(['Retries', frame.retryLabel]);
        }
        if (frame.consoleCount) {
          entries.push(['Console Logs', frame.consoleCount]);
        }
        if (frame.networkCount) {
          entries.push(['Network Events', frame.networkCount]);
        }
        if (frame.assertionMessage) {
          entries.push(['Assertion', frame.assertionMessage]);
        }

        entries.forEach(([label, value]) => {
          const wrapper = document.createElement('div');
          const labelEl = document.createElement('div');
          labelEl.className = 'meta-label';
          labelEl.textContent = label;
          const valueEl = document.createElement('div');
          valueEl.className = 'meta-value';
          valueEl.textContent = value;
          wrapper.appendChild(labelEl);
          wrapper.appendChild(valueEl);
          metaGrid.appendChild(wrapper);
        });
      }

      function applyFrame(frame, instant) {
        if (!frame) return;
        const transitionDelay = instant ? 0 : 180;
        img.classList.remove('ready');
        if (frame.screenshot) {
          setTimeout(() => {
            img.src = frame.screenshot;
          }, transitionDelay);
          img.onload = () => {
            img.classList.add('ready');
            const scale = frame.zoomFactor && frame.zoomFactor > 1 ? frame.zoomFactor : 1;
            img.style.transform = 'scale(' + scale + ')';
          };
        }

        highlightLayer.innerHTML = '';
        if (Array.isArray(frame.highlightRegions)) {
          frame.highlightRegions.forEach((region) => {
            if (!region) return;
            const node = document.createElement('div');
            node.className = 'highlight';
            node.style.left = (region.left * 100) + '%';
            node.style.top = (region.top * 100) + '%';
            node.style.width = (region.width * 100) + '%';
            node.style.height = (region.height * 100) + '%';
            if (region.color) {
              node.style.boxShadow = '0 0 0 6px ' + region.color + '44';
              node.style.borderColor = region.color;
            }
            highlightLayer.appendChild(node);
          });
        }

        if (frame.cursor && typeof frame.cursor.x === 'number' && typeof frame.cursor.y === 'number') {
          cursor.classList.add('visible');
          cursor.style.left = (frame.cursor.x * 100) + '%';
          cursor.style.top = (frame.cursor.y * 100) + '%';
          cursor.style.transform = 'translate(-50%, -50%) scale(1)';
          if (frame.cursor.pulse) {
            cursor.animate([
              { transform: 'translate(-50%, -50%) scale(0.85)', opacity: 0.9 },
              { transform: 'translate(-50%, -50%) scale(1.1)', opacity: 1 },
              { transform: 'translate(-50%, -50%) scale(0.9)', opacity: 0.9 }
            ], {
              duration: 900,
              easing: 'ease-in-out'
            });
          }
        } else {
          cursor.classList.remove('visible');
        }

        titleEl.textContent = frame.title || 'Untitled step';
        descEl.textContent = frame.subtitle || '';
        addressBar.textContent = frame.addressBar || 'automation replay';

        statusEl.classList.remove('success', 'failure');
        if (frame.status === 'success') {
          statusEl.textContent = 'SUCCESS';
          statusEl.classList.add('success');
        } else if (frame.status === 'failure') {
          statusEl.textContent = 'FAILURE';
          statusEl.classList.add('failure');
        } else {
          statusEl.textContent = String(frame.status || '').toUpperCase() || 'STEP';
        }

        renderMeta(frame);

        const duration = frame.playDuration;
        timelineProgress.style.transitionDuration = duration + 'ms';
        timelineProgress.style.transform = 'scaleX(0)';
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            timelineProgress.style.transform = 'scaleX(1)';
          });
        });
      }

      function scheduleNext() {
        if (timer) {
          clearTimeout(timer);
        }
        const frame = frames[playhead];
        applyFrame(frame, playhead === 0);
        const delay = frame.playDuration || defaultHold;
        timer = setTimeout(() => {
          playhead = (playhead + 1) % frames.length;
          scheduleNext();
        }, delay);
      }

      scheduleNext();
    })();
  </script>
</body>
</html>`;
}

async function main() {
  const options = parseArgs(process.argv.slice(2));
  const inputPath = path.resolve(process.cwd(), options.input);
  const outputDir = path.resolve(process.cwd(), options.output);
  const assetsDir = path.join(outputDir, 'assets');

  if (!(await fileExists(inputPath))) {
    throw new Error(`Input file not found: ${inputPath}`);
  }

  if (!options.overwrite && await fileExists(outputDir)) {
    throw new Error(`Output directory already exists: ${outputDir} (use --overwrite to replace)`);
  }

  const raw = await fsp.readFile(inputPath, 'utf8');
  const parsed = JSON.parse(raw);

  const exportPackage = parsed.package || parsed;
  if (!exportPackage || !Array.isArray(exportPackage.frames)) {
    throw new Error('Export package missing frames array');
  }

  await fsp.mkdir(outputDir, { recursive: true });
  await fsp.mkdir(assetsDir, { recursive: true });

  const assetPaths = new Map();
  if (Array.isArray(exportPackage.assets)) {
    let idx = 0;
    for (const asset of exportPackage.assets) {
      idx += 1;
      try {
        const localPath = await downloadAsset(asset, assetsDir, idx, options.baseUrl);
        if (localPath && asset.id) {
          assetPaths.set(asset.id, localPath);
        }
      } catch (error) {
        console.warn(`[render-export] Failed to download asset ${asset?.id || idx}: ${error.message}`);
      }
    }
  }

  const frames = exportPackage.frames.map((frame, index) => {
    const viewport = frame.viewport || { width: 1920, height: 1080 };
    const screenshotPath = frame.screenshotAssetID && assetPaths.has(frame.screenshotAssetID)
      ? assetPaths.get(frame.screenshotAssetID)
      : null;

    const highlightRegions = Array.isArray(frame.highlightRegions)
      ? frame.highlightRegions
        .map((region) => {
          if (!region) return null;
          const normalized = normalizeBox(region.boundingBox, viewport);
          if (!normalized) return null;
          return {
            ...normalized,
            color: region.color || null,
          };
        })
        .filter(Boolean)
      : [];

    const cursorPoint = frame.normalizedClickPosition
      ? { x: frame.normalizedClickPosition.x, y: frame.normalizedClickPosition.y }
      : normalizePoint(frame.clickPosition, viewport);

    const resilience = frame.resilience || {};
    const assertion = frame.assertion || null;

    let assertionMessage = '';
    if (assertion) {
      if (assertion.message) {
        assertionMessage = assertion.message;
      } else if (assertion.mode && assertion.selector) {
        assertionMessage = `${assertion.mode} on ${assertion.selector}`;
      }
    }

    const playDuration = Math.max(frame.durationMs || 1600, 800) + (frame.holdMs || 650);

    let retryLabel = '';
    if (Number.isFinite(resilience.attempt) && Number.isFinite(resilience.maxAttempts)) {
      retryLabel = `${resilience.attempt} / ${resilience.maxAttempts}`;
    }

    const durationLabel = frame.durationMs ? `${frame.durationMs} ms` : '';

    return {
      index,
      title: sanitizeTitle(frame.title, `${frame.stepType || 'step'} #${index + 1}`),
      subtitle: frame.stepType ? frame.stepType.toUpperCase() : '',
      status: frame.status ? frame.status.toLowerCase() : 'success',
      playDuration,
      screenshot: screenshotPath,
      zoomFactor: frame.zoomFactor && frame.zoomFactor > 0 ? frame.zoomFactor : 1,
      highlightRegions,
      cursor: cursorPoint ? { ...cursorPoint, pulse: frame.status !== 'failure' } : null,
      addressBar: frame.finalUrl || exportPackage.execution?.workflow_name || 'automation replay',
      finalUrl: frame.finalUrl || '',
      assertionMessage,
      retryLabel,
      durationLabel,
      consoleCount: frame.consoleLogCount || 0,
      networkCount: frame.networkEventCount || 0,
    };
  });

  const payload = {
    generatedAt: new Date().toISOString(),
    execution: {
      executionId: exportPackage.execution?.execution_id || parsed.execution_id || '',
      workflowId: exportPackage.execution?.workflow_id || '',
      workflowName: exportPackage.execution?.workflow_name || parsed.workflow_name || '',
      status: exportPackage.execution?.status || parsed.status || '',
      frameCount: frames.length,
    },
    theme: exportPackage.theme || {},
    frames,
  };

  const html = buildHtmlDocument(payload);
  await fsp.writeFile(path.join(outputDir, 'index.html'), html, 'utf8');

  await fsp.writeFile(
    path.join(outputDir, 'README.txt'),
    `Browser Automation Studio Replay\n\nGenerated: ${payload.generatedAt}\nFrames: ${frames.length}\n\nOpen index.html in a browser to view the replay.\n`,
    'utf8',
  );

  console.log(`Replay package written to ${outputDir}`);
}

main().catch((error) => {
  console.error(`[render-export] ${error.message}`);
  process.exitCode = 1;
});
