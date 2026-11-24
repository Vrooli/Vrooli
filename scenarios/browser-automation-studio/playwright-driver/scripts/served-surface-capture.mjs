#!/usr/bin/env node
/**
 * Private, synthetic-network screenshots for an exact served UI build.
 * Unlike feature-inventory, this captures full viewport frames at a caller
 * selected desktop or phone size. All data APIs are intercepted; live mode is
 * deliberately not supported by this command.
 */
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import { chromium } from 'rebrowser-playwright';
import { getSyntheticProfile, createSyntheticRouteHandler } from './synthetic-network.mjs';

const sha256 = (bytes) => createHash('sha256').update(bytes).digest('hex');
const blockedAction = /\b(delete|install|launch|send|submit|save|approve|revoke|remove|export|copy|run|retry|apply|close group)\b/i;

export function parseCaptureManifest(raw) {
  const value = typeof raw === 'string' ? JSON.parse(raw) : raw;
  if (!value || !Array.isArray(value.states) || value.states.length === 0) {
    throw new Error('capture manifest requires a non-empty states array');
  }
  const ids = new Set();
  for (const state of value.states) {
    if (!/^[a-z0-9][a-z0-9-]{0,47}$/.test(state.id ?? '')) throw new Error(`invalid state id: ${state.id}`);
    if (ids.has(state.id)) throw new Error(`duplicate state id: ${state.id}`);
    ids.add(state.id);
    if (!Array.isArray(state.actions ?? [])) throw new Error(`state ${state.id} actions must be an array`);
    for (const action of state.actions ?? []) {
      if (!['click', 'right-click-text', 'right-click-testid', 'click-testid'].includes(action.kind)) throw new Error(`unsupported action kind: ${action.kind}`);
      if (typeof action.name !== 'string' || action.name.trim() === '') throw new Error(`state ${state.id} has an unnamed action`);
      if (blockedAction.test(action.name)) throw new Error(`effectful action is not allowed in a synthetic capture: ${action.name}`);
      const safeTestId = ['nav-settings', 'workspace-sidebar-toggle', 'group-ctx-manage-groups', 'group-ctx-toggle-collapse', 'toolbar-ai', 'msg-export-enter', 'msg-export-select-all', 'msg-export-continue', 'messages-search-field'].includes || /^manage-groups-recolor-synthetic-group-\d{4}$/.test(action.testId ?? '')(action.testId) || /^sidebar-session-synthetic-session-\d{4}$/.test(action.testId ?? '') || /^msg-(?:open-reader|actions-more)-synthetic-event-\d{4}$/.test(action.testId ?? '') || (action.kind === 'right-click-testid' && /^msg-card-synthetic-event-\d{4}$/.test(action.testId ?? ''));
      if (action.kind.endsWith('testid') && !safeTestId) {
        throw new Error(`test id is not on the safe view-only allowlist: ${action.testId}`);
      }
      if (action.kind === 'click' && !['button', 'tab', 'radio', 'link', 'menuitem', 'textbox'].includes(action.role)) {
        throw new Error(`click action requires a supported semantic role: ${action.name}`);
      }
    }
  }
  return value;
}

export function parseViewport(value) {
  const match = /^(\d{3,4})x(\d{3,4})$/.exec(value ?? '');
  if (!match) throw new Error('viewport must use WIDTHxHEIGHT, for example 1440x900 or 390x844');
  const width = Number(match[1]);
  const height = Number(match[2]);
  if (width < 320 || width > 2560 || height < 568 || height > 2000) throw new Error('viewport is outside the supported review range');
  return { width, height };
}

function parseOptions(args) {
  const options = {};
  for (let i = 0; i < args.length; i += 2) {
    const key = args[i];
    const value = args[i + 1];
    if (!key?.startsWith('--') || !value || value.startsWith('--')) throw new Error(`expected --option VALUE near ${key ?? '<end>'}`);
    options[key.slice(2)] = value;
  }
  return options;
}

async function main() {
  const [url, outDir, ...args] = process.argv.slice(2);
  if (!url || !outDir) throw new Error('usage: served-surface-capture.mjs URL OUTDIR --scenario ID --build HINT --revision HINT --synthetic-profile PROFILE --viewport WxH --manifest FILE');
  const opts = parseOptions(args);
  for (const required of ['scenario', 'build', 'revision', 'synthetic-profile', 'viewport', 'manifest']) {
    if (!opts[required]) throw new Error(`missing required --${required}`);
  }
  const base = new URL(url);
  if (!['http:', 'https:'].includes(base.protocol) || base.username || base.password || base.search || base.hash) {
    throw new Error('URL must be an HTTP(S) origin/path without credentials, query, or fragment');
  }
  const profile = getSyntheticProfile(opts['synthetic-profile']);
  if (!profile) throw new Error(`unknown synthetic profile: ${opts['synthetic-profile']}`);
  const viewport = parseViewport(opts.viewport);
  const manifest = parseCaptureManifest(await fs.readFile(path.resolve(opts.manifest), 'utf8'));
  const destination = path.resolve(outDir);
  await fs.mkdir(destination, { recursive: true });
  if ((await fs.readdir(destination)).length) throw new Error(`output directory must be empty: ${destination}`);

  const browser = await chromium.launch({ headless: true });
  const captures = [];
  const requests = [];
  try {
    for (const state of manifest.states) {
      const page = await browser.newPage({ viewport, deviceScaleFactor: 1, colorScheme: 'dark' });
      const runtimeErrors = [];
      let servedHTML = '';
      page.on('pageerror', (error) => { if (runtimeErrors.length < 20) runtimeErrors.push(`${error.name}: ${error.message}`.slice(0, 500)); });
      page.on('console', (message) => {
        if (message.type() === 'error' && runtimeErrors.length < 20) runtimeErrors.push(message.text().slice(0, 500));
      });
      page.on('requestfailed', (request) => {
        if (runtimeErrors.length < 20 && ['document', 'script', 'stylesheet'].includes(request.resourceType())) {
          runtimeErrors.push(`request failed: ${new URL(request.url()).pathname}`);
        }
      });
      await page.addInitScript(() => {
        class ClosedSyntheticWebSocket extends EventTarget {
          static CONNECTING = 0; static OPEN = 1; static CLOSING = 2; static CLOSED = 3;
          CONNECTING = 0; OPEN = 1; CLOSING = 2; CLOSED = 3;
          readyState = 0; bufferedAmount = 0; extensions = ''; protocol = ''; binaryType = 'blob';
          onopen = null; onmessage = null; onerror = null; onclose = null;
          constructor(url) {
            super(); this.url = String(url);
            queueMicrotask(() => this.finishClose());
          }
          send() { throw new DOMException('Synthetic WebSocket is closed', 'InvalidStateError'); }
          close() { this.finishClose(); }
          finishClose() {
            if (this.readyState === this.CLOSED) return;
            this.readyState = this.CLOSED;
            const event = new CloseEvent('close', { code: 1000, reason: 'synthetic capture', wasClean: true });
            this.dispatchEvent(event); this.onclose?.(event);
          }
        }
        Object.defineProperty(window, 'WebSocket', { value: ClosedSyntheticWebSocket, configurable: false });
      });
      await page.route('**/*', createSyntheticRouteHandler({
        origin: base.origin,
        profile,
        onRecord: (row) => requests.push({ state: state.id, method: row.method, pathname: new URL(row.url).pathname, outcome: row.outcome }),
      }));
      const response = await page.goto(base.href, { waitUntil: 'domcontentloaded' });
      servedHTML = response ? await response.text() : '';
      if (!response?.ok()) throw new Error(`${state.id}: top-level document returned ${response?.status() ?? 'no response'}`);
      await page.waitForTimeout(450);
      for (const action of state.actions ?? []) {
        const target = action.kind === 'right-click-text'
          ? page.getByText(action.name, { exact: true })
          : action.kind.endsWith('testid')
            ? page.getByTestId(action.testId)
            : page.getByRole(action.role, { name: action.name, exact: true });
        const matches = await target.count();
        if (matches !== 1) throw new Error(`${state.id}: expected one ${action.kind} target named ${JSON.stringify(action.name)}, found ${matches}`);
        if (!(await target.isVisible())) throw new Error(`${state.id}: action target is not visible: ${action.name}`);
        await target.click({ button: action.kind.startsWith('right-click') ? 'right' : 'left' });
        await page.waitForTimeout(300);
      }
      const png = await page.screenshot({ type: 'png', animations: 'disabled' });
      const file = `${state.id}.png`;
      await fs.writeFile(path.join(destination, file), png);
      captures.push({
        id: state.id,
        actions: state.actions ?? [],
        html_sha256: sha256(Buffer.from(servedHTML)),
        png_sha256: sha256(png),
        png_bytes: png.length,
        runtime_errors: runtimeErrors,
      });
      await page.close();
    }
  } finally {
    await browser.close();
  }
  const receipt = {
    schema: 'bas-served-surface-capture/v1',
    scenario: opts.scenario,
    build_hint: opts.build,
    source_revision_hint: opts.revision,
    synthetic_profile: opts['synthetic-profile'],
    origin: base.origin,
    viewport: opts.viewport,
    color_scheme: 'dark',
    captured_at: new Date().toISOString(),
    captures,
    synthetic_requests: requests,
  };
  await fs.writeFile(path.join(destination, 'capture.json'), `${JSON.stringify(receipt, null, 2)}\n`);
  process.stdout.write(`${JSON.stringify({ status: 'ok', states: captures.length, errors: captures.reduce((n, c) => n + c.runtime_errors.length, 0), output: destination })}\n`);
}

if (process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1])) {
  main().catch((error) => { process.stderr.write(`${error.message}\n`); process.exitCode = 1; });
}
