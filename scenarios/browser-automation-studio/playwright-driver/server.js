#!/usr/bin/env node
// Lightweight Playwright driver for the Go AutomationEngine. It exposes an
// HTTP API the Go adapter calls so desktop/Electron bundles can run workflows
// without Browserless. This server is intentionally dependency-light: it uses
// built-in http plus Playwright.

const http = require('http');
const path = require('path');
const os = require('os');
const { chromium } = require('playwright'); // Ensure playwright is installed in the bundle environment.

const PORT = process.env.PLAYWRIGHT_DRIVER_PORT || 39400;

/** @type {Map<string, {browser: any, context: any, page: any}>} */
const sessions = new Map();

const json = (res, status, payload) => {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(payload));
};

const parseBody = (req) =>
  new Promise((resolve, reject) => {
    let data = '';
    req.on('data', (chunk) => {
      data += chunk;
      if (data.length > 5 * 1024 * 1024) {
        reject(new Error('request body too large'));
      }
    });
    req.on('end', () => {
      if (!data) return resolve({});
      try {
        resolve(JSON.parse(data));
      } catch (err) {
        reject(err);
      }
    });
  });

const makeSessionId = () => `sess_${Math.random().toString(36).slice(2, 10)}`;

async function startSession(req, res) {
  const body = await parseBody(req);
  const viewport = body.viewport || {};
  const tracing = !!body.tracing;
  const video = !!body.video;
  const harPath = body.har_path;
  const tracePath = body.trace_path || path.join(os.tmpdir(), `bas-trace-${Date.now()}.zip`);

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: {
      width: viewport.width || 1280,
      height: viewport.height || 720,
    },
    baseURL: body.base_url || undefined,
    recordVideo: video ? { dir: body.video_dir || os.tmpdir() } : undefined,
    recordHar: harPath ? { path: harPath } : undefined,
  });
  if (tracing) {
    await context.tracing.start({ screenshots: true, snapshots: true, path: tracePath });
  }
  const page = await context.newPage();

  const sessionId = makeSessionId();
  sessions.set(sessionId, { browser, context, page, tracing, video, harPath, tracePath });

  json(res, 200, { session_id: sessionId });
}

function health(_req, res) {
  json(res, 200, {
    status: 'ok',
    timestamp: new Date().toISOString(),
    sessions: sessions.size,
  });
}

async function resetSession(req, res, sessionId) {
  const session = sessions.get(sessionId);
  if (!session) return json(res, 404, { error: 'session not found' });

  await session.page.close().catch(() => {});
  await session.context.close().catch(() => {});
  session.context = await session.browser.newContext();
  session.page = await session.context.newPage();
  json(res, 200, { status: 'ok' });
}

async function closeSession(req, res, sessionId) {
  const session = sessions.get(sessionId);
  if (!session) return json(res, 404, { error: 'session not found' });
  if (session.tracing && session.context) {
    await session.context.tracing.stop().catch(() => {});
  }
  await session.browser.close().catch(() => {});
  sessions.delete(sessionId);
  json(res, 200, { status: 'closed' });
}

async function runInstruction(req, res, sessionId) {
  const session = sessions.get(sessionId);
  if (!session) return json(res, 404, { error: 'session not found' });
  const page = session.page;
  const body = await parseBody(req);
  const instr = body.instruction || {};

  const startedAt = new Date();
  const consoleLogs = [];
  const network = [];
  let extractedData = undefined;
  let assertion = undefined;

  const onConsole = (msg) => {
    consoleLogs.push({
      type: msg.type(),
      text: msg.text(),
      timestamp: new Date(),
      location: msg.location
        ? `${msg.location().url || ''}:${msg.location().lineNumber || 0}`
        : '',
    });
  };
  const onRequest = (req) => {
    network.push({
      type: 'request',
      url: req.url(),
      method: req.method(),
      timestamp: new Date(),
    });
  };
  const onResponse = async (resp) => {
    network.push({
      type: 'response',
      url: resp.url(),
      status: resp.status(),
      ok: resp.ok(),
      timestamp: new Date(),
    });
  };
  const onRequestFailed = (req) => {
    network.push({
      type: 'failure',
      url: req.url(),
      failure: req.failure() ? req.failure().errorText : '',
      timestamp: new Date(),
    });
  };

  page.on('console', onConsole);
  page.on('request', onRequest);
  page.on('response', onResponse);
  page.on('requestfailed', onRequestFailed);

  let success = true;
  let failure = null;

  try {
    const res = await applyInstruction(page, instr);
    if (res && res.extractedData) {
      extractedData = res.extractedData;
    }
    if (res && res.assertion) {
      assertion = res.assertion;
      if (!assertion.success) {
        success = false;
        failure = {
          kind: 'engine',
          code: 'assertion_failed',
          message: assertion.message || 'assertion failed',
          fatal: false,
          retryable: false,
          occurred_at: new Date(),
          source: 'engine',
        };
      }
    }
  } catch (err) {
    success = false;
    failure = {
      kind: 'engine',
      code: 'playwright_error',
      message: err.message || 'playwright error',
      fatal: false,
      retryable: false,
      occurred_at: new Date(),
      source: 'engine',
    };
  } finally {
    page.off('console', onConsole);
    page.off('request', onRequest);
    page.off('response', onResponse);
    page.off('requestfailed', onRequestFailed);
  }

  const completedAt = new Date();

  let screenshotBase64 = '';
  let screenshotWidth = 0;
  let screenshotHeight = 0;
  let videoPath = '';
  let tracePath = '';
  let harPath = session.harPath || '';
  try {
    const shot = await page.screenshot({ fullPage: true, type: 'png' });
    screenshotBase64 = shot.toString('base64');
    const vp = page.viewportSize();
    screenshotWidth = vp?.width || 0;
    screenshotHeight = vp?.height || 0;
  } catch (err) {
    // Best-effort; failures are non-fatal.
  }

  if (session.video && page.video) {
    try {
      const video = await page.video();
      if (video) {
        videoPath = await video.path();
      }
    } catch (err) {
      // ignore
    }
  }

  if (session.tracing && session.context?.tracing) {
    try {
      tracePath = session.tracePath || '';
    } catch (err) {
      // ignore
    }
  }

  let domHTML = '';
  let domPreview = '';
  try {
    domHTML = await page.content();
    domPreview = domHTML.slice(0, 512);
  } catch (err) {
    // ignore
  }

  json(res, 200, {
    schema_version: 'automation-step-outcome-v1',
    payload_version: '1',
    step_index: instr.index || 0,
    step_type: instr.type || '',
    node_id: instr.node_id || '',
    success,
    started_at: startedAt.toISOString(),
    completed_at: completedAt.toISOString(),
    duration_ms: completedAt.getTime() - startedAt.getTime(),
    final_url: page.url(),
    console_logs: consoleLogs,
    network,
    extracted_data: extractedData,
    assertion,
    failure,
    screenshot_base64: screenshotBase64,
    screenshot_media_type: screenshotBase64 ? 'image/png' : '',
    screenshot_width: screenshotWidth,
    screenshot_height: screenshotHeight,
    video_path: videoPath,
    trace_path: tracePath,
    har_path: harPath,
    dom_html: domHTML,
    dom_preview: domPreview,
  });
}

async function applyInstruction(page, instr) {
  const params = instr.params || {};
  switch ((instr.type || '').toLowerCase()) {
    case 'navigate': {
      const target = params.url || params.target || params.href;
      if (!target) throw new Error('navigate missing url');
      await page.goto(target, {
        timeout: params.timeoutMs || 45000,
        waitUntil: params.waitUntil || 'networkidle',
      });
      break;
    }
    case 'click': {
      const selector = params.selector;
      if (!selector) throw new Error('click missing selector');
      await page.click(selector, { timeout: params.timeoutMs || 15000 });
      break;
    }
    case 'hover': {
      const selector = params.selector;
      if (!selector) throw new Error('hover missing selector');
      await page.hover(selector, { timeout: params.timeoutMs || 15000 });
      break;
    }
    case 'type': {
      const selector = params.selector;
      const text = params.text || params.value || '';
      if (!selector) throw new Error('type missing selector');
      await page.fill(selector, text, { timeout: params.timeoutMs || 20000 });
      break;
    }
    case 'uploadfile': {
      const selector = params.selector;
      const path = params.filePath || params.file_path || params.path;
      if (!selector) throw new Error('uploadFile missing selector');
      if (!path) throw new Error('uploadFile missing file path');
      await page.setInputFiles(selector, path, { timeout: params.timeoutMs || 20000 });
      break;
    }
    case 'scroll': {
      const x = Number(params.x || 0);
      const y = Number(params.y || 0);
      await page.evaluate(
        ([sx, sy]) => window.scrollTo(sx, sy),
        [x, y]
      );
      break;
    }
    case 'wait': {
      if (params.selector) {
        await page.waitForSelector(params.selector, {
          timeout: params.timeoutMs || 20000,
        });
      } else {
        await page.waitForTimeout(params.timeoutMs || params.ms || 1000);
      }
      break;
    }
    case 'evaluate': {
      const script = params.script || params.expression;
      if (!script) throw new Error('evaluate missing script');
      const result = await page.evaluate(script, params.args || {});
      return { extractedData: { result } };
    }
    case 'extract': {
      const selector = params.selector;
      if (!selector) throw new Error('extract missing selector');
      const text = await page.textContent(selector, { timeout: params.timeoutMs || 15000 });
      return { extractedData: { [selector]: text || '' } };
    }
    case 'assert': {
      const selector = params.selector;
      if (!selector) throw new Error('assert missing selector');
      const mode = (params.mode || params.kind || (params.contains === false ? 'equals' : 'contains')).toLowerCase();
      const timeout = params.timeoutMs || 15000;

      const exists = async () => {
        const el = await page.$(selector);
        return !!el;
      };

      const textContent = async () => (await page.textContent(selector, { timeout })) || '';

      let pass = false;
      let actual = '';
      let message = '';

      switch (mode) {
        case 'exists':
          pass = await exists();
          actual = pass ? 'exists' : 'missing';
          message = pass ? '' : 'expected element to exist';
          break;
        case 'notexists':
        case 'not_exists':
        case 'absent':
          pass = !(await exists());
          actual = pass ? 'absent' : 'present';
          message = pass ? '' : 'expected element to be absent';
          break;
        case 'visible':
          pass = await page.isVisible(selector, { timeout });
          actual = pass ? 'visible' : 'not visible';
          message = pass ? '' : 'expected element to be visible';
          break;
        case 'hidden':
        case 'notvisible':
          pass = !(await page.isVisible(selector, { timeout }));
          actual = pass ? 'hidden' : 'visible';
          message = pass ? '' : 'expected element to be hidden';
          break;
        case 'attribute': {
          const name = params.attribute || params.attr;
          if (!name) throw new Error('assert attribute missing name');
          const val = await page.getAttribute(selector, name, { timeout });
          actual = val || '';
          const expected = params.expected || params.value || '';
          pass = actual === expected;
          message = pass ? '' : `expected attribute ${name} "${actual}" to equal "${expected}"`;
          return {
            assertion: {
              mode: 'attribute',
              selector,
              expected,
              actual,
              success: pass,
              message,
            },
          };
        }
        case 'equals':
        case 'contains':
    case 'matches':
    default: {
      const expected = params.expected || params.text || '';
      actual = await textContent();
      if (mode === 'matches') {
        const re = new RegExp(expected);
        pass = re.test(actual);
        message = pass ? '' : `expected "${actual}" to match /${expected}/`;
      } else {
        const contains = mode === 'contains';
        pass = contains ? actual.includes(expected) : actual === expected;
        message = pass ? '' : `expected "${actual}" to ${contains ? 'contain' : 'equal'} "${expected}"`;
      }
      break;
    }
  }

  return {
    assertion: {
      mode,
      selector,
      expected: params.expected || params.text,
      actual,
      success: pass,
      message,
    },
  };
}
    case 'download': {
      const timeout = params.timeoutMs || 30000;
      const selector = params.selector;
      const targetURL = params.url;
      let download = null;

      if (selector) {
        [download] = await Promise.all([
          page.waitForEvent('download', { timeout }),
          page.click(selector, { timeout }),
        ]);
      } else if (targetURL) {
        [download] = await Promise.all([
          page.waitForEvent('download', { timeout }),
          page.goto(targetURL, { timeout }),
        ]);
      } else {
        throw new Error('download requires selector or url');
      }

      const savePath = await download.path().catch(() => null);
      const suggested = download.suggestedFilename();
      const dlUrl = download.url();
      return {
        extractedData: {
          download: {
            url: dlUrl,
            suggestedFilename: suggested,
            path: savePath || '',
          },
        },
      };
    }
    case 'screenshot': {
      await page.screenshot({ fullPage: true });
      break;
    }
    default:
      throw new Error(`unsupported step type: ${instr.type}`);
  }
}

const server = http.createServer(async (req, res) => {
  try {
    if (req.method === 'POST' && req.url === '/session/start') {
      return await startSession(req, res);
    }
    if (req.method === 'GET' && req.url === '/health') {
      return health(req, res);
    }
    const runMatch = req.url.match(/^\/session\/([^/]+)\/run$/);
    const resetMatch = req.url.match(/^\/session\/([^/]+)\/reset$/);
    const closeMatch = req.url.match(/^\/session\/([^/]+)\/close$/);

    if (req.method === 'POST' && runMatch) {
      return await runInstruction(req, res, runMatch[1]);
    }
    if (req.method === 'POST' && resetMatch) {
      return await resetSession(req, res, resetMatch[1]);
    }
    if (req.method === 'POST' && closeMatch) {
      return await closeSession(req, res, closeMatch[1]);
    }
    json(res, 404, { error: 'not found' });
  } catch (err) {
    json(res, 500, { error: err.message || 'internal error' });
  }
});

server.listen(PORT, () => {
  // eslint-disable-next-line no-console
  console.log(`Playwright driver listening on ${PORT}`);
});
