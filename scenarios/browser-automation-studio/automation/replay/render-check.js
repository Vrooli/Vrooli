#!/usr/bin/env node

/**
 * Render Check Automation
 *
 * Runs a short Browser Automation Studio workflow, exports the replay package,
 * renders the HTML bundle via `browser-automation-studio execution render`,
 * and validates that the output contains the expected assets and metadata.
 */

'use strict';

const { execFile } = require('node:child_process');
const { promisify } = require('node:util');
const fs = require('node:fs');
const fsp = require('node:fs/promises');
const path = require('node:path');
const os = require('node:os');

const execFileAsync = promisify(execFile);
const SCENARIO = 'browser-automation-studio';
const WORKFLOW_NAME = `render-check-${Date.now()}`;
const SHOULD_STOP = process.env.BAS_AUTOMATION_STOP_SCENARIO !== '0';

async function run(cmd, args = [], options = {}) {
  const result = await execFileAsync(cmd, args, { encoding: 'utf8', ...options });
  return result.stdout.trim();
}

async function runWithOutput(cmd, args = [], options = {}) {
  return execFileAsync(cmd, args, { encoding: 'utf8', ...options });
}

async function ensureCommand(name) {
  try {
    await run('which', [name]);
  } catch (error) {
    throw new Error(`Required command '${name}' is not available on PATH`);
  }
}

async function waitForHealth(baseUrl, timeoutMs = 120000) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    try {
      const response = await fetch(`${baseUrl}/health`);
      if (response.ok) {
        return;
      }
    } catch (error) {
      // retry until timeout
    }
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }
  throw new Error('API did not become healthy within timeout');
}

function buildWorkflowDefinition() {
  const markup = `<!doctype html><html><head><meta charset="utf-8"><title>BAS Render Check</title>
    <style>
      body { font-family: Inter, sans-serif; margin: 0; min-height: 100vh; display: flex; justify-content: center; align-items: center; background: radial-gradient(circle at 25% 20%, #0f172a, #020617 70%); color: #f8fafc; }
      .card { max-width: 720px; background: rgba(17, 24, 39, 0.92); border-radius: 24px; padding: 48px; box-shadow: 0 32px 64px rgba(2, 8, 23, 0.55); text-align: center; }
      #hero { margin-bottom: 32px; }
      #hero h1 { font-size: 2.5rem; margin-bottom: 12px; text-transform: uppercase; letter-spacing: 0.12em; }
      #hero p { font-size: 1.1rem; color: rgba(148, 163, 184, 0.9); }
      a.cta { display: inline-flex; margin-top: 24px; padding: 14px 28px; border-radius: 999px; font-weight: 600; text-transform: uppercase; background: #38bdf8; color: #0f172a; text-decoration: none; }
    </style>
  </head><body>
    <div class="card">
      <div id="hero">
        <h1>Browser Automation Studio</h1>
        <p>Render check run for marketing-ready replay validation.</p>
      </div>
      <a class="cta" href="https://example.com">View demo</a>
    </div>
  </body></html>`;

  const dataUrl = `data:text/html;base64,${Buffer.from(markup).toString('base64')}`;

  return {
    nodes: [
      {
        id: 'navigate-demo',
        type: 'navigate',
        position: { x: -260, y: -60 },
        data: {
          label: 'Navigate demo page',
          url: dataUrl,
          waitUntil: 'networkidle2',
          timeoutMs: 20000,
        },
      },
      {
        id: 'highlight-shot',
        type: 'screenshot',
        position: { x: 20, y: -60 },
        data: {
          label: 'Highlight hero',
          name: 'render-check-frame',
          focusSelector: '#hero',
          highlightSelectors: ['#hero h1', '#hero p'],
          highlightColor: '#38bdf8',
          highlightPadding: 16,
          maskSelectors: [],
          maskOpacity: 0.5,
          background: '#0f172a',
          zoomFactor: 1.12,
        },
      },
    ],
    edges: [
      { id: 'edge-navigate-screenshot', source: 'navigate-demo', target: 'highlight-shot' },
    ],
  };
}


async function main() {
  await ensureCommand('vrooli');
  await ensureCommand('browser-automation-studio');
  await ensureCommand('curl');
  await ensureCommand('jq');
  await ensureCommand('node');

  // Start fresh scenario state
  await run('vrooli', ['scenario', 'stop', SCENARIO]).catch(() => {});
  await run('vrooli', ['scenario', 'start', SCENARIO, '--clean-stale']);

  const portOutputRaw = await run('vrooli', ['scenario', 'port', SCENARIO, 'API_PORT']);
  const portOutput = portOutputRaw.trim();
  const apiPort = portOutput.includes('=') ? portOutput.split('=')[1].trim() : portOutput;
  if (!apiPort) {
    throw new Error(`Unable to determine API_PORT from output: ${portOutput}`);
  }

  const baseUrl = `http://localhost:${apiPort}`;
  const apiUrl = `${baseUrl}/api/v1`;
  await waitForHealth(baseUrl);

  const tmpDir = await fsp.mkdtemp(path.join(os.tmpdir(), 'bas-render-'));
  const exportPath = path.join(tmpDir, 'export.json');
  const renderDir = path.join(tmpDir, 'rendered');

  let workflowId = '';
  try {
    // Create workflow
    const workflowDef = buildWorkflowDefinition();
    const createResponse = await fetch(`${apiUrl}/workflows/create`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: WORKFLOW_NAME,
        folder_path: '/automation',
        flow_definition: workflowDef,
      }),
    });

    if (!createResponse.ok) {
      const text = await createResponse.text();
      throw new Error(`Failed to create workflow: ${createResponse.status} ${text}`);
    }

    const created = await createResponse.json();
    workflowId = created.workflow_id || created.id;
    if (!workflowId) {
      throw new Error('Workflow create response missing workflow_id');
    }

    // Execute workflow and wait for completion
    const executeResponse = await fetch(`${apiUrl}/workflows/${workflowId}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ parameters: {}, wait_for_completion: false }),
    });

    if (!executeResponse.ok) {
      const text = await executeResponse.text();
      throw new Error(`Failed to execute workflow: ${executeResponse.status} ${text}`);
    }

    const executionPayload = await executeResponse.json();
    const executionId = executionPayload.execution_id;
    if (!executionId) {
      throw new Error('Execution response missing execution_id');
    }

    console.log(`Execution ID: ${executionId}`);

    await waitForExecutionCompletion(apiUrl, executionId);

    // Export replay JSON
    const env = { ...process.env, API_PORT: apiPort };
    await run('browser-automation-studio', ['execution', 'export', executionId, '--output', exportPath], { env });

    const exportContent = await fsp.readFile(exportPath, 'utf8');
    const parsed = JSON.parse(exportContent);
    const pkg = parsed.package || parsed;
    if (!pkg || !Array.isArray(pkg.frames) || pkg.frames.length === 0) {
      console.error('Export payload (truncated):');
      console.error(exportContent.slice(0, 4000));
      throw new Error('Export package missing frames');
    }

    // Render HTML replay
    try {
      await run('browser-automation-studio', ['execution', 'render', executionId, '--output', renderDir, '--overwrite'], { env });
    } catch (error) {
      if (error.stdout || error.stderr) {
        console.error('execution render stdout:', error.stdout);
        console.error('execution render stderr:', error.stderr);
      }
      throw error;
    }

    console.log(`Rendered bundle directory: ${renderDir}`);

    const indexHtmlPath = path.join(renderDir, 'index.html');
    const readmePath = path.join(renderDir, 'README.txt');
    const assetsDir = path.join(renderDir, 'assets');

    await assertFileExists(indexHtmlPath, 'Rendered replay index.html missing');
    await assertFileExists(assetsDir, 'Assets directory missing');

    const assetFiles = await fsp.readdir(assetsDir);
    console.log(`Replay summary: frames=${pkg.frames.length}, assets=${assetFiles.length}`);
    if (assetFiles.length === 0) {
      console.warn('⚠️  No downloadable assets present; relying on inline frame metadata.');
    }

    const html = await fsp.readFile(indexHtmlPath, 'utf8');
    const requires = ['Automation Replay', 'step-title', 'frame-image', 'timeline-progress'];
    for (const snippet of requires) {
      if (!html.includes(snippet)) {
        throw new Error(`Rendered HTML missing expected snippet: ${snippet}`);
      }
    }

    if (pkg.summary && pkg.summary.frame_count && pkg.summary.frame_count !== pkg.frames.length) {
      throw new Error(`Frame count mismatch (summary=${pkg.summary.frame_count}, frames=${pkg.frames.length})`);
    }

    if (pkg.frames.length < 1) {
      throw new Error('No frames in export package');
    }

    await assertFileExists(readmePath, 'Rendered bundle missing README.txt');

    console.log('🎬 Render check successful');
    console.log(`Replay bundle located at: ${renderDir}`);
    console.log(`Assets found: ${assetFiles.length}, frames exported: ${pkg.frames.length}`);
  } finally {
    if (workflowId) {
      await fetch(`${apiUrl}/workflows/${workflowId}`, { method: 'DELETE' }).catch(() => {});
    }
    if (SHOULD_STOP) {
      await run('vrooli', ['scenario', 'stop', SCENARIO]).catch(() => {});
    }
  }
}

async function assertFileExists(targetPath, message) {
  try {
    await fsp.access(targetPath, fs.constants.F_OK);
  } catch (error) {
    throw new Error(message);
  }
}

async function waitForExecutionCompletion(apiUrl, executionId, timeoutMs = 180000) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const response = await fetch(`${apiUrl}/executions/${executionId}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch execution status (${response.status})`);
    }
    const payload = await response.json();
    const status = payload.status;
    if (status === 'completed') {
      return;
    }
    if (status === 'failed') {
      const error = payload.error || 'execution failed';
      throw new Error(`Execution ${executionId} failed: ${error}`);
    }
    await new Promise((resolve) => setTimeout(resolve, 2000));
  }
  throw new Error(`Execution ${executionId} did not complete within timeout`);
}

main().catch((error) => {
  console.error(`❌ render-check failed: ${error.message}`);
  process.exitCode = 1;
});
