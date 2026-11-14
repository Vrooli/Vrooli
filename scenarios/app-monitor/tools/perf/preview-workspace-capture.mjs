#!/usr/bin/env node

import fs from 'node:fs/promises';
import net from 'node:net';
import path from 'node:path';
import { spawn } from 'node:child_process';
import { createRequire } from 'node:module';

const require = createRequire(new URL('../../ui/package.json', import.meta.url));
const WebSocket = require('ws');

const DEFAULT_URL = 'http://localhost:20000';
const DEFAULT_PANES = 8;
const DEFAULT_OUT = '/tmp/app-monitor/perf/preview-workspace-stable.json';
const DEFAULT_CHROME = process.env.CHROME_BIN || 'google-chrome';

const delay = (ms) => new Promise((resolve) => {
  setTimeout(resolve, ms);
});

const parseArgs = (argv) => {
  const args = {
    url: DEFAULT_URL,
    panes: DEFAULT_PANES,
    out: DEFAULT_OUT,
    trace: null,
    chrome: DEFAULT_CHROME,
  };

  for (let index = 0; index < argv.length; index += 1) {
    const arg = argv[index];
    const [key, inlineValue] = arg.startsWith('--') ? arg.split('=', 2) : [arg, undefined];
    const value = inlineValue ?? argv[index + 1];
    if (inlineValue === undefined && key.startsWith('--')) {
      index += 1;
    }

    if (key === '--url' && value) {
      args.url = value;
    } else if (key === '--panes' && value) {
      args.panes = Math.max(1, Math.min(12, Number.parseInt(value, 10) || DEFAULT_PANES));
    } else if (key === '--out' && value) {
      args.out = value;
    } else if (key === '--trace' && value) {
      args.trace = value;
    } else if (key === '--chrome' && value) {
      args.chrome = value;
    }
  }

  return args;
};

const findFreePort = async () => new Promise((resolve, reject) => {
  const server = net.createServer();
  server.once('error', reject);
  server.listen(0, '127.0.0.1', () => {
    const address = server.address();
    server.close(() => {
      resolve(typeof address === 'object' && address ? address.port : 0);
    });
  });
});

class CdpClient {
  constructor(wsUrl) {
    this.nextId = 1;
    this.pending = new Map();
    this.listeners = new Map();
    this.socket = new WebSocket(wsUrl);
  }

  async open() {
    await new Promise((resolve, reject) => {
      this.socket.once('open', resolve);
      this.socket.once('error', reject);
    });
    this.socket.on('message', (message) => {
      const payload = JSON.parse(String(message));
      if (payload.id && this.pending.has(payload.id)) {
        const { resolve, reject } = this.pending.get(payload.id);
        this.pending.delete(payload.id);
        if (payload.error) {
          reject(new Error(payload.error.message));
        } else {
          resolve(payload.result ?? {});
        }
        return;
      }
      const callbacks = this.listeners.get(payload.method);
      callbacks?.forEach((callback) => callback(payload.params ?? {}));
    });
  }

  send(method, params = {}) {
    const id = this.nextId;
    this.nextId += 1;
    this.socket.send(JSON.stringify({ id, method, params }));
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve, reject });
    });
  }

  on(method, callback) {
    const callbacks = this.listeners.get(method) ?? [];
    callbacks.push(callback);
    this.listeners.set(method, callbacks);
  }

  close() {
    this.socket.close();
  }
}

const launchChrome = async ({ chrome, port }) => {
  const userDataDir = `/tmp/app-monitor/perf/chrome-${Date.now()}-${Math.random().toString(16).slice(2)}`;
  await fs.mkdir(userDataDir, { recursive: true });
  const child = spawn(chrome, [
    '--headless=new',
    '--no-first-run',
    '--disable-gpu',
    '--disable-dev-shm-usage',
    `--remote-debugging-port=${port}`,
    `--user-data-dir=${userDataDir}`,
    'about:blank',
  ], { stdio: ['ignore', 'ignore', 'pipe'] });

  child.stderr.on('data', () => undefined);
  await delay(800);

  return {
    async stop() {
      if (!child.killed) {
        child.kill('SIGTERM');
      }
      await new Promise((resolve) => {
        child.once('exit', resolve);
        setTimeout(resolve, 1500);
      });
      await fs.rm(userDataDir, { recursive: true, force: true });
    },
  };
};

const createPageTarget = async (port, url) => {
  const response = await fetch(`http://127.0.0.1:${port}/json/new?${encodeURIComponent(url)}`, { method: 'PUT' });
  if (!response.ok) {
    throw new Error(`Failed to create Chrome target: ${response.status} ${response.statusText}`);
  }
  const target = await response.json();
  return target.webSocketDebuggerUrl;
};

const evaluate = async (client, expression, awaitPromise = false) => {
  const result = await client.send('Runtime.evaluate', {
    expression,
    awaitPromise,
    returnByValue: true,
  });
  if (result.exceptionDetails) {
    throw new Error(result.exceptionDetails.text || 'Runtime evaluation failed');
  }
  return result.result?.value;
};

const evaluateJson = async (client, expression) => {
  const serialized = await evaluate(client, `JSON.stringify(${expression})`);
  return serialized ? JSON.parse(serialized) : null;
};

const waitForExpression = async (client, expression, timeoutMs = 15000) => {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    const matched = await evaluate(client, `Boolean(${expression})`);
    if (matched) {
      return;
    }
    await delay(100);
  }
  throw new Error(`Timed out waiting for ${expression}`);
};

const seedWorkspace = async (client, paneCount) => {
  await evaluate(client, `(() => {
    const now = Date.now();
    const panes = Array.from({ length: ${paneCount} }, (_, index) => ({
      id: 'perf-pane-' + (index + 1),
      appId: index % 2 === 0 ? 'scenario-1' : 'scenario-2',
      createdAt: now + index,
    }));
    const paneViewState = Object.fromEntries(panes.map((pane) => [
      pane.id,
      {
        previewUrl: 'http://localhost:3000/apps/' + pane.appId + '/proxy/',
        previewUrlInput: 'http://localhost:3000/apps/' + pane.appId + '/proxy/',
        hasCustomPreviewUrl: false,
        history: [],
        historyIndex: -1,
        initialPreviewUrl: 'http://localhost:3000/apps/' + pane.appId + '/proxy/',
        isLogsVisible: false,
        isFullView: false,
      },
    ]));
    localStorage.setItem('app-monitor:preview-workspace-v1', JSON.stringify({
      state: {
        interactionMode: 'browse',
        workspaceZoom: 1,
        isWorkspaceMinimapVisible: true,
        panes,
        paneViewState,
        focusedPaneId: panes[0]?.id ?? null,
        pinnedPaneId: null,
        pinnedColumn: null,
        columnFractions: [1, 1],
        rowFractions: [1, 1, 1, 1, 1, 1],
      },
      version: 1,
    }));
  })()`);
};

const collectCounters = async (client) => evaluateJson(client, `(() => ({
  iframes: document.querySelectorAll('.preview-pane iframe').length,
  panes: document.querySelectorAll('.preview-pane').length,
  tabCards: document.querySelectorAll('.tab-card').length,
  profilerMarks: performance.getEntriesByType('measure')
    .filter((entry) => entry.name.includes('React') || entry.name.includes('Profiler'))
    .length,
  storageBytes: new Blob([localStorage.getItem('app-monitor:preview-workspace-v1') ?? '']).size,
}))()`);

const readTraceStream = async (client, stream) => {
  let content = '';
  for (;;) {
    const chunk = await client.send('IO.read', { handle: stream });
    content += chunk.data ?? '';
    if (chunk.eof) {
      break;
    }
  }
  await client.send('IO.close', { handle: stream });
  return content;
};

const run = async () => {
  const args = parseArgs(process.argv.slice(2));
  await fs.mkdir(path.dirname(args.out), { recursive: true });
  if (args.trace) {
    await fs.mkdir(path.dirname(args.trace), { recursive: true });
  }

  const port = await findFreePort();
  const chrome = await launchChrome({ chrome: args.chrome, port });
  const errors = [];

  try {
    const wsUrl = await createPageTarget(port, args.url);
    const client = new CdpClient(wsUrl);
    await client.open();
    client.on('Runtime.exceptionThrown', (event) => {
      errors.push(event.exceptionDetails?.text ?? 'Runtime exception');
    });
    client.on('Log.entryAdded', (event) => {
      if (event.entry?.level === 'error') {
        errors.push(event.entry.text);
      }
    });

    await client.send('Page.enable');
    await client.send('Runtime.enable');
    await client.send('Log.enable');
    if (args.trace) {
      await client.send('Tracing.start', {
        categories: 'devtools.timeline,blink.user_timing,v8.execute,disabled-by-default-v8.cpu_profiler',
        transferMode: 'ReturnAsStream',
      });
    }

    await waitForExpression(client, 'document.readyState !== "loading"');
    await seedWorkspace(client, args.panes);
    const start = performance.now();
    await client.send('Page.reload', { ignoreCache: true });
    await waitForExpression(client, 'document.querySelectorAll(".preview-pane").length > 0');
    await delay(1000);
    const afterMount = await collectCounters(client);

    await client.send('Input.dispatchMouseEvent', {
      type: 'mouseWheel',
      x: 900,
      y: 800,
      deltaX: 0,
      deltaY: 1600,
    });
    await delay(1000);
    const afterScroll = await collectCounters(client);

    await client.send('Input.dispatchKeyEvent', {
      type: 'rawKeyDown',
      key: 'k',
      code: 'KeyK',
      windowsVirtualKeyCode: 75,
      modifiers: process.platform === 'darwin' ? 4 : 2,
    });
    await client.send('Input.dispatchKeyEvent', {
      type: 'keyUp',
      key: 'k',
      code: 'KeyK',
      windowsVirtualKeyCode: 75,
      modifiers: process.platform === 'darwin' ? 4 : 2,
    });
    await delay(500);
    await evaluate(client, `(() => {
      const input = Array.from(document.querySelectorAll('input')).find((node) => (
        /search/i.test(node.placeholder || '') || /search/i.test(node.getAttribute('aria-label') || '')
      ));
      if (!input) return false;
      input.value = 'scenario';
      input.dispatchEvent(new Event('input', { bubbles: true }));
      return true;
    })()`);
    await delay(500);
    const afterPicker = await collectCounters(client);

    let traceBytes = 0;
    if (args.trace) {
      const tracingComplete = new Promise((resolve) => {
        client.on('Tracing.tracingComplete', resolve);
      });
      await client.send('Tracing.end');
      const complete = await tracingComplete;
      const traceContent = await readTraceStream(client, complete.stream);
      traceBytes = Buffer.byteLength(traceContent);
      await fs.writeFile(args.trace, traceContent, 'utf8');
    }

    const result = {
      capturedAt: new Date().toISOString(),
      url: args.url,
      panesRequested: args.panes,
      elapsedMs: Math.round(performance.now() - start),
      tracePath: args.trace,
      traceBytes,
      counters: {
        afterMount,
        afterScroll,
        afterPicker,
      },
      errors,
    };

    await fs.writeFile(args.out, `${JSON.stringify(result, null, 2)}\n`, 'utf8');
    console.log(JSON.stringify(result, null, 2));
    client.close();
  } finally {
    await chrome.stop();
  }
};

run().catch((error) => {
  console.error(`[preview-workspace-capture] ${error.stack || error.message}`);
  process.exitCode = 1;
});
