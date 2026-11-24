#!/usr/bin/env node
/**
 * Served feature inventory for redesign preservation. Uses BAS's pinned
 * rebrowser-playwright runtime and stores one screenshot crop per feature.
 */
import fs from 'node:fs/promises';
import path from 'node:path';
import { chromium } from 'rebrowser-playwright';
import { sha256Hex, servedDocumentBuildIdentity, stableFeatureId, stableHash, inventoryCoverage, safeInteractionKind, parseSafeActionLabels, parseScopedSafeActionPaths, parseScopedSafeKeyChords, parseNamedEntrypoints, routeIdentityForState, matchesSafeActionLabel, validateInventory, featureDiffSummary, deduplicateFeatureRecords, mergeFeatureInventories, captureFeatureScreenshot } from './feature-inventory-core.mjs';
import { SYNTHETIC_PROFILES, createSyntheticRouteHandler } from './synthetic-network.mjs';

const [command, ...args] = process.argv.slice(2);
const hash = stableHash;
const read = async (file) => JSON.parse(await fs.readFile(file, 'utf8'));
const write = async (file, value) => fs.writeFile(file, `${JSON.stringify(value, null, 2)}\n`);

if (command === 'merge') {
  const [outDir, ...inputPaths] = args;
  if (!outDir || !inputPaths.length) throw new Error('usage: feature-inventory.mjs merge OUTDIR INPUT.json [INPUT.json ...]');
  const absoluteOutDir = path.resolve(outDir);
  const existing = await fs.readdir(absoluteOutDir).catch((error) => error.code === 'ENOENT' ? [] : Promise.reject(error));
  if (existing.length) throw new Error(`merge output directory must be empty: ${absoluteOutDir}`);
  const inventories = await Promise.all(inputPaths.map(read));
  const merged = mergeFeatureInventories(inventories);
  const mergedById = new Map(merged.features.map((feature) => [feature.id, feature]));
  merged.coverage.sources.forEach((source, index) => { source.file = path.resolve(inputPaths[index]); });
  await fs.mkdir(path.join(absoluteOutDir, 'evidence'), { recursive: true });
  const copiedById = new Map();
  let evidenceVariants = 0;
  for (const [index, inventory] of inventories.entries()) {
    const sourceDir = path.dirname(path.resolve(inputPaths[index]));
    for (const feature of inventory.features) {
      const sourcePath = path.resolve(sourceDir, feature.screenshot);
      if (!sourcePath.startsWith(`${sourceDir}${path.sep}`)) throw new Error(`screenshot path escapes source inventory: ${feature.id}`);
      const bytes = await fs.readFile(sourcePath);
      if (bytes.length < 8 || !bytes.subarray(0, 8).equals(Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]))) throw new Error(`expected PNG evidence for ${feature.id}: ${sourcePath}`);
      const key = hash(bytes);
      const existingEvidence = copiedById.get(feature.id);
      if (!existingEvidence) {
        const target = `evidence/${feature.id}.png`;
        await fs.copyFile(sourcePath, path.join(absoluteOutDir, target));
        copiedById.set(feature.id, [{ key, target }]);
        const mergedFeature = mergedById.get(feature.id);
        mergedFeature.screenshot = target;
      } else if (!existingEvidence.some((item) => item.key === key)) {
        const target = `evidence/${index}-${feature.id}.png`;
        await fs.copyFile(sourcePath, path.join(absoluteOutDir, target));
        existingEvidence.push({ key, target }); evidenceVariants++;
        const mergedFeature = mergedById.get(feature.id);
        mergedFeature.evidence_variants ||= [];
        mergedFeature.evidence_variants.push(target);
      }
    }
  }
  const errors = validateInventory(merged, 'merged inventory');
  if (errors.length) throw new Error(`merged inventory is invalid:\n${errors.map((error) => `- ${error}`).join('\n')}`);
  await write(path.join(absoluteOutDir, 'inventory.json'), merged);
  process.stdout.write(`${JSON.stringify({ status: 'ok', sources: inputPaths.length, features: merged.features.length, evidence_variants: evidenceVariants, inventory: path.join(absoluteOutDir, 'inventory.json') })}\n`);
  process.exit(0);
}

if (command === 'diff') {
  const [beforePath, afterPath, outPath] = args;
  if (!beforePath || !afterPath || !outPath) throw new Error('usage: feature-inventory.mjs diff BEFORE.json AFTER.json OUT.json');
  const before = await read(beforePath), after = await read(afterPath);
  const identityErrors = [...validateInventory(before, 'before'), ...validateInventory(after, 'after')];
  if (identityErrors.length) throw new Error(`invalid feature inventory:\n${identityErrors.map((error) => `- ${error}`).join('\n')}`);
  if (before.identity.scenario !== after.identity.scenario) throw new Error('before and after inventories must identify the same scenario');
  if (before.identity.build === after.identity.build && before.identity.source_revision === after.identity.source_revision) throw new Error('before and after inventories must identify distinct builds or source revisions');
  const afterIds = new Set(after.features.map((f) => f.id));
  const disappeared = before.features.filter((f) => !afterIds.has(f.id));
  const afterRoutes = new Map(after.features.filter((f) => f.kind === 'route').map((f) => [f.route, f]));
  const embed = async (inventoryPath, feature) => {
    const evidencePath = path.resolve(path.dirname(inventoryPath), feature.screenshot);
    const bytes = await fs.readFile(evidencePath);
    const mime = feature.screenshot.toLowerCase().endsWith('.png') ? 'png' : null;
    if (!mime || !bytes.subarray(0, 8).equals(Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]))) {
      throw new Error(`expected PNG evidence for ${feature.id}: ${evidencePath}`);
    }
    if (bytes.length > 1_048_576) throw new Error(`decision-card evidence exceeds 1 MiB: ${evidencePath} (${bytes.length} bytes)`);
    return `data:image/${mime};base64,${bytes.toString('base64')}`;
  };
  const cards = await Promise.all(disappeared.map(async (feature) => {
    const routeAfter = afterRoutes.get(feature.route);
    const evidence = [{ alt: `Before the change: ${feature.label} on ${feature.route}.`, image: await embed(beforePath, feature) }];
    if (routeAfter) evidence.push({ alt: `After the change: current screen on ${feature.route}.`, image: await embed(afterPath, routeAfter) });
    return {
      schema: 'research-feature-decision/v1',
      id: `FEATURE_LOSS_${feature.id.replace(/[^A-Za-z0-9_-]/g, '_')}`.toUpperCase(),
      feature_id: feature.id,
      title: feature.label,
      location: { screen: feature.route, taps: [`Open ${feature.route}`, `Review ${feature.kind}: ${feature.label}`] },
      summary: `This ${feature.kind} was visible before the change and is missing from the served screen now.`,
      evidence,
      options: [
        { key: 'keep', label: 'Keep this feature', effect: 'Restore or retain this feature in the changed build.' },
        { key: 'drop', label: 'Drop this feature', effect: 'Delete or remove the code that provided this feature.' }
      ],
      recommendation: 'keep', status: 'pending', answer: null, terms: []
    };
  }));
  // The cards are built above from validated inventories; reject one a reader could not act on.
  for (const card of cards) {
    const problems = [];
    if (!card.title?.trim()) problems.push('title is empty');
    if (!card.location?.screen) problems.push('screen is missing');
    if (!card.evidence.length || card.evidence.some((item) => !item.alt?.trim() || !item.image)) problems.push('evidence needs an alt text and an image');
    if (!card.options.some((option) => option.key === card.recommendation)) problems.push('recommendation is not one of the options');
    if (problems.length) throw new Error(`generated decision card ${card.id} is not usable: ${problems.join('; ')}`);
  }
  await write(outPath, {
    schema: 'bas-feature-diff/v1', before: before.identity, after: after.identity,
    totals: featureDiffSummary(before, after), cards
  });
  process.exit(0);
}

if (command !== 'crawl') throw new Error('usage: feature-inventory.mjs crawl URL OUTDIR --scenario NAME --build ID [--max-routes N]');
const [startUrl, outDir, ...options] = args;
if (!startUrl || !outDir) throw new Error('crawl requires URL and output directory');
const opts = { entrypoint: [] };
for (let index = 0; index < options.length; index += 2) {
  const flag = options[index];
  const value = options[index + 1];
  if (!flag?.startsWith('--') || !value || value.startsWith('--')) throw new Error(`expected --option VALUE near ${flag ?? '<end>'}`);
  const name = flag.slice(2);
  if (name === 'entrypoint') opts.entrypoint.push(value);
  else opts[name] = value;
}
const origin = new URL(startUrl).origin;
if (new URL(startUrl).search) throw new Error('start URL must not contain a query; use --entrypoint NAME=URL to name and keep query data private');
const maxRoutes = Math.max(1, Math.min(200, Number(opts['max-routes'] ?? 50)));
const syntheticProfile = opts['synthetic-profile'] ? SYNTHETIC_PROFILES[opts['synthetic-profile']] : null;
if (opts['synthetic-profile'] && !syntheticProfile) throw new Error(`unknown synthetic profile: ${opts['synthetic-profile']}`);
const namedEntrypoints = parseNamedEntrypoints(opts.entrypoint, startUrl);
if (namedEntrypoints.length && !syntheticProfile) throw new Error('named entrypoints require a synthetic network profile');
const safeActionPaths = [
  ...parseScopedSafeActionPaths(opts['safe-action-paths'] ?? ''),
  ...parseSafeActionLabels(opts['safe-actions'] ?? '').map((label) => ({ state: null, path: [label] })),
];
const safeKeyChords = parseScopedSafeKeyChords(opts['safe-key-chords'] ?? '');
if ((safeActionPaths.length || safeKeyChords.length) && !syntheticProfile) throw new Error('safe action options require a synthetic network profile');
await fs.mkdir(path.join(outDir, 'evidence'), { recursive: true });
const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 1 });
const networkEvidence = [];
const runtimeErrors = [];
page.on('console', (message) => {
  if (message.type() === 'error' && runtimeErrors.length < 20) runtimeErrors.push(message.text().slice(0, 2000));
});
page.on('pageerror', (error) => {
  if (runtimeErrors.length < 20) runtimeErrors.push(`${error.name}: ${error.message}`.slice(0, 2000));
});
await page.addInitScript(() => {
  const diagnostics = [];
  Object.defineProperty(window, '__basRuntimeErrors', { value: diagnostics, configurable: false });
  const originalError = console.error.bind(console);
  console.error = (...args) => {
    const safe = args.map((value) => value instanceof Error
      ? `${value.name}: ${value.message}\n${value.stack || ''}`
      : typeof value === 'string' ? value : '[non-text diagnostic]');
    diagnostics.push(safe.join(' ').slice(0, 2000));
    originalError(...args);
  };
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
      const event = new CloseEvent('close', { code: 1000, reason: 'synthetic crawl', wasClean: true });
      this.dispatchEvent(event); this.onclose?.(event);
    }
  }
  window.WebSocket = ClosedSyntheticWebSocket;
});
if (syntheticProfile) {
  await page.route('**/*', createSyntheticRouteHandler({ origin, profile: syntheticProfile, onRecord: (row) => networkEvidence.push({ ...row, url: new URL(row.url).pathname, method: row.method }) }));
}
const queue = [{ href: startUrl, state: 'default' }, ...namedEntrypoints.map((entrypoint) => ({ ...entrypoint, state: entrypoint.name }))];
const seen = new Set(), features = [];
const coverage = inventoryCoverage();
coverage.keyboard.candidates = safeKeyChords.map(({ state, label }) => ({ state, chord: label }));
if (safeKeyChords.length) coverage.unattempted.keyboard = 'Only explicitly allowlisted Ctrl/Cmd+Shift+letter chords were attempted; all other key sequences remain unobserved.';
const runtime = `${opts.scenario ?? 'unknown'}@build-hint:${opts.build ?? 'unverified'}`;
const recordLoadedScripts = async (route, state) => {
  const scripts = await page.evaluate(() => [...new Set(performance.getEntriesByType('resource').flatMap((entry) => {
    try {
      const asset = new URL(entry.name);
      return asset.origin === location.origin && /\.m?js$/i.test(asset.pathname) ? [asset.pathname] : [];
    } catch { return []; }
  }))].sort());
  const existing = coverage.loadedAssets.find((item) => item.route === route && item.state === state);
  if (existing) existing.scripts = [...new Set([...existing.scripts, ...scripts])].sort();
  else coverage.loadedAssets.push({ route, state, scripts });
};
const surfaceSignature = async () => page.evaluate(() => {
  const body = document.body;
  const visible = (el) => {
    const rect = el.getBoundingClientRect(), style = getComputedStyle(el);
    return rect.width > 0 && rect.height > 0 && style.display !== 'none' && style.visibility !== 'hidden';
  };
  const text = (body.innerText || '').replace(/\s+/g, ' ').slice(0, 20000);
  const anchors = [...body.querySelectorAll('main,header,nav,aside,dialog,section,[role],[data-testid],button,a[href],input,select,textarea,h1,h2,h3')]
    .filter(visible).map((el) => {
      const rect = el.getBoundingClientRect();
      return [el.tagName, el.getAttribute('role') || '', el.getAttribute('data-testid') || '', el.getAttribute('aria-label') || '', el.getAttribute('aria-expanded') || '', Math.round(rect.x), Math.round(rect.y), Math.round(rect.width), Math.round(rect.height)].join(':');
    }).join('|');
  return `${text}\n${anchors}`;
});
const waitForStableSurface = async ({ before = null, timeout = 5000, stableMs = 350 } = {}) => {
  const start = Date.now();
  let previous = null, stableSince = 0, observedChange = before === null;
  while (Date.now() - start < timeout) {
    const current = await surfaceSignature();
    if (current !== previous) { previous = current; stableSince = Date.now(); }
    if (before !== null && current !== before) observedChange = true;
    if (observedChange && Date.now() - stableSince >= stableMs) return current;
    await page.waitForTimeout(100);
  }
  throw new Error(before === null ? 'served page did not reach a stable visible surface' : 'action did not produce a stable visible state');
};
const scan = () => page.evaluate(() => {
  const visible = (el) => {
    const r = el.getBoundingClientRect();
    if (r.width <= 0 || r.height <= 0) return false;
    let effectiveOpacity = 1;
    for (let current = el; current && current.nodeType === 1; current = current.parentElement) {
      const style = getComputedStyle(current);
      if (style.display === 'none' || style.visibility === 'hidden' || style.visibility === 'collapse') return false;
      effectiveOpacity *= Number.parseFloat(style.opacity || '1');
      if (current.getAttribute('aria-hidden') === 'true' || current.hasAttribute('inert')) return false;
    }
    return effectiveOpacity > 0;
  };
  const clean = (v) => String(v || '').trim().replace(/\s+/g, ' ').slice(0, 160);
  const sourceOf = (el) => { const k = Object.keys(el).find(x => x.startsWith('__reactFiber$')); if (k) { let f = el[k]; for (let i=0; f && i<8; i++, f=f.return) if (f._debugSource?.fileName) return f._debugSource.fileName; } return null; };
  const eventPropsOf = (el) => { const k = Object.keys(el).find(x => x.startsWith('__reactFiber$')); const names = []; if (k) { for (let f = el[k], i=0; f && i<12; i++, f=f.return) names.push(...Object.keys(f.memoizedProps || {}).filter(prop => /^on[A-Z]/.test(prop))); } return [...new Set(names)].sort(); };
  const links = [...document.querySelectorAll('a[href]')].filter(visible).map(a => a.href);
  const occurrences = new Map();
  let domId = 0;
  const tag = (el, kind, label, context) => {
    const semantic = [kind, el.getAttribute('data-testid'), el.id, el.getAttribute('role'), el.getAttribute('aria-label'), label, context].filter(Boolean).join('|');
    const nth = occurrences.get(semantic) || 0; occurrences.set(semantic, nth + 1);
    const id = `bas-${domId++}`;
    el.setAttribute('data-bas-inventory-id', id); return { id, identity: `${semantic}|${nth}` };
  };
  const actions = [...document.querySelectorAll('button,[role="button"],[role="radio"],[role="tab"],a[href],[role="menuitem"],input[type="button"],input[type="submit"]')].filter(visible).map(el => {
    const label = clean(el.getAttribute('aria-label') || el.innerText || el.getAttribute('title') || el.value || el.tagName.toLowerCase());
    const context = el.parentElement?.getAttribute('aria-label') || el.parentElement?.getAttribute('data-testid') || el.parentElement?.getAttribute('role') || '';
    const kind = el.getAttribute('role') === 'menuitem' ? 'menu-item' : 'action';
    const explicitRole = (el.getAttribute('role') || '').toLowerCase();
    const role = ['button', 'radio', 'tab', 'link', 'menuitem'].includes(explicitRole)
      ? explicitRole
      : el.tagName.toLowerCase() === 'a' ? 'link'
        : ['button', 'input'].includes(el.tagName.toLowerCase()) ? 'button' : null;
    return { ...tag(el, kind, label, context), kind, label, context, role, source: sourceOf(el), eventProps: eventPropsOf(el) };
  });
  const components = [...document.querySelectorAll('main,header,nav,aside,dialog,[role="dialog"],[role="region"],section,form,[data-testid]')].filter(el => visible(el) && !el.matches('button,a,input,[role="button"],[role="tab"],summary')).map(el => {
    const label = clean(el.getAttribute('aria-label') || el.getAttribute('data-testid') || el.querySelector('h1,h2,h3')?.textContent || el.tagName.toLowerCase());
    const context = el.parentElement?.getAttribute('aria-label') || el.parentElement?.getAttribute('data-testid') || el.parentElement?.getAttribute('role') || '';
    const semanticIdentity = tag(el, 'component', label, context).identity;
    const rect = el.getBoundingClientRect();
    return { id: el.getAttribute('data-bas-inventory-id'), identity: semanticIdentity, componentEvidence: 'semantic-region', kind: 'component', label, context, source: sourceOf(el), eventProps: [], rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height } };
  });
  // A host element's fiber.return chain exposes rendered component boundaries
  // in React builds. Keep one representative visible host per component fiber;
  // never inspect handler values or execute application callbacks.
  const reactInstances = new Map();
  const seenFibers = new WeakSet();
  const reactOccurrences = new Map();
  for (const el of [...document.querySelectorAll('body *')].filter(visible)) {
    const fiberKey = Object.keys(el).find(key => key.startsWith('__reactFiber$'));
    if (!fiberKey) continue;
    let fiber = el[fiberKey];
    for (let depth = 0; fiber && depth < 30; depth++, fiber = fiber.return) {
      const type = fiber.elementType || fiber.type;
      if (!type || (typeof type !== 'function' && typeof type !== 'object')) continue;
      if (seenFibers.has(fiber)) continue;
      seenFibers.add(fiber);
      const name = type.displayName || type.name || type.render?.displayName || type.render?.name || 'Anonymous';
      const source = fiber._debugSource?.fileName ? `${fiber._debugSource.fileName}:${fiber._debugSource.lineNumber || 0}` : null;
      const base = `${name}|${source || ''}|${String(fiber.key ?? '')}`;
      const occurrence = reactOccurrences.get(base) || 0;
      reactOccurrences.set(base, occurrence + 1);
      const key = `${base}|${occurrence}`;
      const instance = fiber.key !== null && fiber.key !== undefined ? `key:${String(fiber.key)}` : `ordinal:${occurrence}`;
      const eventProps = Object.keys(fiber.memoizedProps || {}).filter(prop => /^on[A-Z]/.test(prop)).sort();
      const semanticIdentity = `component|${name}|${source || 'source-unavailable'}|${instance}|${el.tagName.toLowerCase()}`;
      const tagged = tag(el, 'component', name, 'react-component');
      const rect = el.getBoundingClientRect();
      const identity = source ? `react|${name || 'anonymous'}|${source}|${instance}` : semanticIdentity;
      reactInstances.set(key, { id: tagged.id, identity, componentEvidence: source ? 'react-type-source' : 'react-type', kind: 'component', label: name, context: 'React component', source, eventProps, rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height } });
    }
  }
  components.push(...reactInstances.values());
  const interactionCandidates = [...document.querySelectorAll('button,[role="button"],[role="tab"],summary')].filter(visible).map(el => {
    const role = el.getAttribute('role') || '';
    const isDetails = el.tagName.toLowerCase() === 'summary' && !!el.closest('details');
    const label = clean(el.getAttribute('aria-label') || el.innerText || el.textContent || role || 'control');
    const context = el.parentElement?.getAttribute('aria-label') || el.parentElement?.getAttribute('data-testid') || el.parentElement?.getAttribute('role') || '';
    return { ...tag(el, 'interaction-candidate', label, context), tagName: el.tagName, role, ariaHaspopup: el.getAttribute('aria-haspopup') || '', ariaExpanded: el.getAttribute('aria-expanded'), detailsSummary: isDetails, label, context, source: sourceOf(el) };
  });
  return { links, items: [...actions, ...components], interactionCandidates };
});
try {
  while (queue.length && seen.size < maxRoutes) {
    const entrypoint = queue.shift();
    const url = new URL(entrypoint.href, origin); url.hash = '';
    const visitKey = `${url.href}\0${entrypoint.state}`;
    if (url.origin !== origin || seen.has(visitKey)) continue;
    seen.add(visitKey);
    if (syntheticProfile) {
      await page.goto(`${origin}/`, { waitUntil: 'domcontentloaded', timeout: 30000 });
      await page.evaluate(() => { localStorage.clear(); sessionStorage.clear(); });
    }
    const route = routeIdentityForState(url.href, entrypoint.state);
    const documentResponse = await page.goto(url.href, { waitUntil: 'domcontentloaded', timeout: 30000 });
    if (!documentResponse || documentResponse.status() < 200 || documentResponse.status() >= 400) {
      throw new Error(`served document response was not successful for ${route}`);
    }
    const documentBody = await documentResponse.body();
    coverage.servedDocuments.push({ route, state: entrypoint.state, sha256: sha256Hex(documentBody), bytes: documentBody.length });
    await waitForStableSurface();
    const pageDiagnostics = await page.evaluate(() => window.__basRuntimeErrors || []);
    for (const message of pageDiagnostics) if (runtimeErrors.length < 20 && !runtimeErrors.includes(message)) runtimeErrors.push(message);
    const routeId = `route-${hash(route)}`, routeShot = `evidence/${routeId}.png`;
    features.push({ id: routeId, identity: route, kind: 'route', label: await page.title() || route, route, source: null, runtime, screenshot: routeShot });
    const discovered = await scan();
    await recordLoadedScripts(route, entrypoint.state);
    coverage.components.semantic += discovered.items.filter(item => item.kind === 'component' && item.componentEvidence === 'semantic-region').length;
    coverage.components.react += discovered.items.filter(item => item.kind === 'component' && item.componentEvidence.startsWith('react-')).length;
    coverage.components.withEventProps += discovered.items.filter(item => item.kind === 'component' && item.eventProps?.length).length;
    discovered.interactions = discovered.interactionCandidates.map((candidate) => ({ ...candidate, type: safeInteractionKind(candidate) })).filter((candidate) => candidate.type);
    for (const interaction of discovered.interactions) coverage.candidates[interaction.type]++;
    for (const link of discovered.links) {
      const target = new URL(link);
      if (target.origin === origin && !seen.has(`${target.href}\0${entrypoint.state}`)) queue.push({ href: link, state: entrypoint.state });
    }
    for (const item of discovered.items) {
      const id = stableFeatureId(item.kind, route, `${item.identity}|${item.source ?? ''}`), shot = `evidence/${id}.png`;
      const screenshotCapture = await captureFeatureScreenshot({ page, locator: item.rect ? null : page.locator(`[data-bas-inventory-id="${item.id}"]`), rect: item.rect, path: path.join(outDir, shot), viewport: page.viewportSize() || undefined });
      features.push({ id, identity: item.identity, kind: item.kind, label: item.label, route, source: item.source, runtime, screenshot: shot, screenshot_capture: screenshotCapture, ...(item.kind === 'component' ? { component_evidence: item.componentEvidence, event_props: item.eventProps || [] } : item.eventProps?.length ? { event_props: item.eventProps } : {}) });
    }
    // Capture the route after its individual feature crops. Screenshotting the
    // first frame immediately after navigation can precede app hydration even
    // when the DOM scan that follows sees the finished controls.
    // The target viewport is the evidence frame. BAS's remote browser backend
    // returns a blank texture for an unclipped screenshot; an explicit
    // viewport clip produces the same 390x844 frame as the component crops.
    await page.screenshot({ path: path.join(outDir, routeShot), clip: { x: 0, y: 0, width: 390, height: 844 } });
    // Activate only ARIA menu buttons, native disclosures, and ARIA tabs. No
    // generic buttons, links, submits, destructive actions, or menu items are clicked.
    for (const interaction of discovered.interactions) {
      try {
        await page.goto(url.href, { waitUntil: 'domcontentloaded', timeout: 30000 }); await page.waitForTimeout(150);
        await scan();
        const control = page.locator(`[data-bas-inventory-id="${interaction.id}"]`);
        if (await control.count() !== 1) throw new Error('control identity was not unique after reload');
        const beforeState = await surfaceSignature();
        await control.click({ timeout: 1500 });
        await waitForStableSurface({ before: beforeState });
        const state = await page.evaluate(({ controlId, type }) => {
          const visible = (node) => { if (!node) return false; const r = node.getBoundingClientRect(), s = getComputedStyle(node); return r.width > 0 && r.height > 0 && s.visibility !== 'hidden' && s.display !== 'none'; };
          const el = document.querySelector(`[data-bas-inventory-id="${controlId}"]`);
          const controls = el?.getAttribute('aria-controls');
          const labelled = el?.id && document.querySelector(`[aria-labelledby="${el.id.replace(/"/g, '\\"')}"]`);
          const localPanel = (controls && document.getElementById(controls)) || labelled || el?.closest('details') || el?.parentElement?.querySelector('[role="menu"],[role="tabpanel"],[role="listbox"]');
          const role = type === 'tab' ? '[role="tabpanel"]' : type === 'menu' ? '[role="menu"],[role="listbox"]' : '[role="region"]';
          const panel = visible(localPanel) ? localPanel : [...document.querySelectorAll(role)].find(visible);
          return { expanded: el?.getAttribute('aria-expanded'), selected: el?.getAttribute('aria-selected'), panelId: panel?.id || '', panelLabel: panel?.getAttribute('aria-label') || panel?.querySelector('h1,h2,h3')?.textContent || '', panelText: (panel?.innerText || panel?.textContent || '').trim().replace(/\s+/g, ' ').slice(0, 240) };
        }, { controlId: interaction.id, type: interaction.type });
        const stateName = state.panelLabel || state.panelText || interaction.label;
        await recordLoadedScripts(route, `interaction:${stateName}`);
        const identity = `${interaction.identity}|state:${state.panelLabel || interaction.label}`;
        const kind = `${interaction.type}-state`, id = stableFeatureId(kind, route, identity), shot = `evidence/${id}.png`;
        const target = state.panelId ? page.locator(`[id="${state.panelId.replace(/"/g, '\\"')}"]`).first() : control;
        const screenshotCapture = await captureFeatureScreenshot({ page, locator: target, path: path.join(outDir, shot), viewport: page.viewportSize() || undefined });
        features.push({ id, identity, kind, label: `${interaction.label}: ${stateName}`, route, source: interaction.source, runtime, state: { control: interaction.label, expanded: state.expanded, selected: state.selected, content: state.panelText }, screenshot: shot, screenshot_capture: screenshotCapture });
        coverage.observed[interaction.type]++;
        const openState = await scan();
        for (const link of openState.links) {
          const target = new URL(link);
          if (target.origin === origin && !seen.has(`${target.href}\0${entrypoint.state}`) && !queue.some((queued) => queued.href === link && queued.state === entrypoint.state)) queue.push({ href: link, state: entrypoint.state });
        }
        openState.interactions = openState.interactionCandidates.map((candidate) => ({ ...candidate, type: safeInteractionKind(candidate) })).filter((candidate) => candidate.type);
        const initialInteractions = new Set(discovered.interactions.map((candidate) => candidate.identity));
        for (const nested of openState.interactions) {
          if (!initialInteractions.has(nested.identity)) coverage.unobserved.push({ route, kind: nested.type, label: nested.label, reason: `Found inside ${interaction.type} state "${stateName}"; nested interaction depth is not explored.` });
        }
        for (const item of openState.items) {
          if (item.kind !== 'menu-item' && !item.id) continue;
          const itemId = stableFeatureId(item.kind, route, `${item.identity}|state:${identity}|${item.source ?? ''}`), itemShot = `evidence/${itemId}.png`;
          if (features.some(feature => feature.id === itemId)) continue;
          const screenshotCapture = await captureFeatureScreenshot({ page, locator: page.locator(`[data-bas-inventory-id="${item.id}"]`), path: path.join(outDir, itemShot), viewport: page.viewportSize() || undefined });
          features.push({ id: itemId, identity: `${item.identity}|state:${identity}`, kind: item.kind, label: item.label, route, source: item.source, runtime, state: { control: interaction.label, name: stateName }, screenshot: itemShot, screenshot_capture: screenshotCapture });
        }
      } catch (error) {
        coverage.unobserved.push({ route, kind: interaction.type, label: interaction.label, reason: error.message });
      }
    }
    // Explicitly allowlisted action labels may open non-destructive local UI
    // surfaces (for example, Settings). This is disabled unless every API call
    // is synthetic and the operator supplied the exact labels on the command.
    for (const actionSpec of safeActionPaths) {
      if (actionSpec.state && actionSpec.state !== entrypoint.state) continue;
      const actionPath = actionSpec.path;
      try {
        // Clear the ephemeral profile before loading the exact entry state.
        // Some apps consume deep-link queries and strip them during bootstrap;
        // clearing after navigation and reloading would erase that named state.
        await page.goto(`${origin}/`, { waitUntil: 'domcontentloaded', timeout: 30000 });
        // Each explicitly allowlisted path starts from a clean, synthetic browser
        // profile. This prevents one selected tab from changing which actions
        // appear in later paths; the Playwright context is ephemeral and holds no
        // operator browser storage.
        await page.evaluate(() => { localStorage.clear(); sessionStorage.clear(); });
        await page.goto(url.href, { waitUntil: 'domcontentloaded', timeout: 30000 }); await page.waitForTimeout(250);
        for (let step = 0; step < actionPath.length; step++) {
          const requestedLabel = actionPath[step];
          // Hydration and modal transitions can take longer than the navigation
          // event even when the page is already usable. Wait for the exact,
          // explicitly allowlisted control before scanning its current state.
          const exactName = new RegExp(`^${requestedLabel.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`, 'i');
          await page.getByRole('button', { name: exactName }).first().waitFor({ state: 'visible', timeout: 3000 }).catch(() => {});
          const current = await scan();
          const action = current.items.find((item) => item.kind === 'action' && matchesSafeActionLabel(item.label, [requestedLabel]));
          if (!action) {
            coverage.unobserved.push({ route, kind: 'safe-action', label: requestedLabel, reason: `This exact allowlisted action was not visible at step ${step + 1} of ${actionPath.join(' > ')}.` });
            break;
          }
          const controlRole = action.role || 'button';
          const control = page.getByRole(controlRole, { name: action.label, exact: true });
          if (await control.count() !== 1) throw new Error(`Expected one ${controlRole} named "${action.label}"; found ${await control.count()}.`);
          const beforeState = await surfaceSignature();
          await control.click({ timeout: 1500 });
          await waitForStableSurface({ before: beforeState });
          const stateName = actionPath.slice(0, step + 1).join(' > ');
          await recordLoadedScripts(route, stateName);
          if (coverage.openedActions.some((opened) => opened.route === route && opened.state === stateName)) continue;
          const stateId = stableFeatureId('action-state', route, stateName), stateShot = `evidence/${stateId}.png`;
          await page.screenshot({ path: path.join(outDir, stateShot), clip: { x: 0, y: 0, width: 390, height: 844 } });
          features.push({ id: stateId, identity: stateName, kind: 'action-state', label: `After ${stateName}`, route, source: action.source, runtime, state: { openedBy: actionPath.slice(0, step + 1) }, screenshot: stateShot });
          const state = await scan();
          for (const item of state.items) {
            const identity = `${item.identity}|state:${stateName}|${item.source ?? ''}`;
            const id = stableFeatureId(item.kind, route, identity), shot = `evidence/${id}.png`;
            const screenshotCapture = await captureFeatureScreenshot({ page, locator: item.rect ? null : page.locator(`[data-bas-inventory-id="${item.id}"]`), rect: item.rect, path: path.join(outDir, shot), viewport: page.viewportSize() || undefined });
            features.push({ id, identity, kind: item.kind, label: item.label, route, source: item.source, runtime, state: { openedBy: actionPath.slice(0, step + 1) }, screenshot: shot, screenshot_capture: screenshotCapture, ...(item.kind === 'component' ? { component_evidence: item.componentEvidence, event_props: item.eventProps || [] } : item.eventProps?.length ? { event_props: item.eventProps } : {}) });
          }
          coverage.openedActions.push({ route, label: action.label, state: stateName });
        }
      } catch (error) {
        coverage.unobserved.push({ route, kind: 'safe-action', label: actionPath.join(' > '), reason: error.message });
      }
    }
    for (const chord of safeKeyChords) {
      if (chord.state && chord.state !== entrypoint.state) continue;
      try {
        await page.goto(`${origin}/`, { waitUntil: 'domcontentloaded', timeout: 30000 });
        await page.evaluate(() => { localStorage.clear(); sessionStorage.clear(); });
        await page.goto(url.href, { waitUntil: 'domcontentloaded', timeout: 30000 });
        // A key handler may belong to a lazily loaded page module. Wait for
        // the served surface to stop changing before sending the chord.
        await waitForStableSurface();
        const beforeState = await surfaceSignature();
        await page.keyboard.press(chord.playwrightKey);
        // The state may itself load a lazy chunk or animate into view. Do not
        // mark it observed until visible text/structure/geometry settles.
        await waitForStableSurface({ before: beforeState });
        const stateName = `keyboard:${chord.label}`;
        await recordLoadedScripts(route, stateName);
        const stateId = stableFeatureId('action-state', route, stateName), stateShot = `evidence/${stateId}.png`;
        await page.screenshot({ path: path.join(outDir, stateShot), clip: { x: 0, y: 0, width: 390, height: 844 } });
        features.push({ id: stateId, identity: stateName, kind: 'action-state', label: `After ${chord.label}`, route, source: null, runtime, state: { openedBy: [chord.label] }, screenshot: stateShot });
        const state = await scan();
        for (const item of state.items) {
          const identity = `${item.identity}|state:${stateName}|${item.source ?? ''}`;
          const id = stableFeatureId(item.kind, route, identity), shot = `evidence/${id}.png`;
          const screenshotCapture = await captureFeatureScreenshot({ page, locator: item.rect ? null : page.locator(`[data-bas-inventory-id="${item.id}"]`), rect: item.rect, path: path.join(outDir, shot), viewport: page.viewportSize() || undefined });
          features.push({ id, identity, kind: item.kind, label: item.label, route, source: item.source, runtime, state: { openedBy: [chord.label] }, screenshot: shot, screenshot_capture: screenshotCapture, ...(item.kind === 'component' ? { component_evidence: item.componentEvidence, event_props: item.eventProps || [] } : item.eventProps?.length ? { event_props: item.eventProps } : {}) });
        }
        coverage.openedActions.push({ route, label: chord.label, state: stateName });
        coverage.keyboard.observed.push({ route, chord: chord.label, state: stateName });
      } catch (error) {
        coverage.unobserved.push({ route, kind: 'safe-key-chord', label: chord.label, reason: error.message });
      }
    }
  }
} finally { await browser.close(); }
coverage.routeLimitReached = queue.length > 0 && seen.size >= maxRoutes;
coverage.interactionScope = 'Interaction-only states are observed only when a control declares native details, aria-expanded, aria-haspopup=menu/listbox/true, or role=tab semantics.';
coverage.openedActions = [...new Map(coverage.openedActions.map((action) => [`${action.route}\0${action.state}`, action])).values()];
coverage.unobserved = [...new Map(coverage.unobserved.map((item) => [`${item.route}\0${item.kind}\0${item.label}\0${item.reason}`, item])).values()];
const buildSource = coverage.servedDocuments[0];
if (!buildSource) throw new Error('crawl did not record a served document; cannot identify its build');
const servedBuild = servedDocumentBuildIdentity(coverage.servedDocuments);
const inventory = { schema: 'bas-feature-inventory/v1', identity: { scenario: opts.scenario ?? null, ...servedBuild, build_hint: opts.build ?? null, source_revision: null, source_revision_hint: opts.revision ?? null, base_url: origin, captured_at: new Date().toISOString(), viewport: '390x844', synthetic_profile: opts['synthetic-profile'] ?? null }, coverage: { ...coverage, network: syntheticProfile ? { mode: 'synthetic', requests: networkEvidence, websocket: 'replaced by inert closed client before app bootstrap; no payload sent or inspected' } : { mode: 'live-owner-authorized' } }, runtime_errors: runtimeErrors, features: deduplicateFeatureRecords(features, 'crawl features') };
const inventoryErrors = validateInventory(inventory, 'crawl inventory');
if (inventoryErrors.length) throw new Error(`crawl produced an invalid inventory:\n${inventoryErrors.map((error) => `- ${error}`).join('\n')}`);
await write(path.join(outDir, 'inventory.json'), inventory);
console.log(JSON.stringify({ status: 'ok', routes: seen.size, features: features.length, inventory: path.join(outDir, 'inventory.json') }));
