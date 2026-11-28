import crypto from 'node:crypto';
import { readFile } from 'node:fs/promises';
import { inflateSync } from 'node:zlib';

const hash = (value) => crypto.createHash('sha256').update(value).digest('hex').slice(0, 20);

export const stableHash = hash;
export const sha256Hex = (value) => crypto.createHash('sha256').update(value).digest('hex');
const pngSignature = Buffer.from([137, 80, 78, 71, 13, 10, 26, 10]);
const paeth = (a, b, c) => {
  const p = a + b - c, pa = Math.abs(p - a), pb = Math.abs(p - b), pc = Math.abs(p - c);
  return pa <= pb && pa <= pc ? a : pb <= pc ? b : c;
};

// Return false only for a supported PNG whose visible pixels are all the same
// color. Unsupported encodings are left to higher-level visual review.
export const pngHasVisiblePixelVariation = (png) => {
  if (!Buffer.isBuffer(png) || png.length < 33 || !png.subarray(0, 8).equals(pngSignature)) return false;
  let width = 0, height = 0, bitDepth = 0, colorType = 0;
  const imageData = [];
  for (let offset = 8; offset + 12 <= png.length;) {
    const length = png.readUInt32BE(offset), type = png.toString('ascii', offset + 4, offset + 8);
    const start = offset + 8, end = start + length;
    if (end + 4 > png.length) return false;
    if (type === 'IHDR') {
      width = png.readUInt32BE(start); height = png.readUInt32BE(start + 4);
      bitDepth = png[start + 8]; colorType = png[start + 9];
    } else if (type === 'IDAT') imageData.push(png.subarray(start, end));
    else if (type === 'IEND') break;
    offset = end + 4;
  }
  const channels = ({ 0: 1, 2: 3, 4: 2, 6: 4 })[colorType];
  if (!width || !height || bitDepth !== 8 || !channels || imageData.length === 0) return true;
  let decoded;
  try { decoded = inflateSync(Buffer.concat(imageData)); } catch { return false; }
  const rowBytes = width * channels;
  if (decoded.length < height * (rowBytes + 1)) return false;
  let offset = 0, previous = new Uint8Array(rowBytes), firstVisible = null;
  const colorChannels = colorType === 2 || colorType === 6 ? 3 : 1;
  const alphaIndex = colorType === 6 ? 3 : colorType === 4 ? 1 : -1;
  for (let y = 0; y < height; y++) {
    const filter = decoded[offset++], row = new Uint8Array(rowBytes);
    for (let x = 0; x < rowBytes; x++) {
      const raw = decoded[offset++], left = x >= channels ? row[x - channels] : 0, up = previous[x], upperLeft = x >= channels ? previous[x - channels] : 0;
      if (filter === 0) row[x] = raw;
      else if (filter === 1) row[x] = (raw + left) & 255;
      else if (filter === 2) row[x] = (raw + up) & 255;
      else if (filter === 3) row[x] = (raw + Math.floor((left + up) / 2)) & 255;
      else if (filter === 4) row[x] = (raw + paeth(left, up, upperLeft)) & 255;
      else return true;
    }
    for (let x = 0; x < rowBytes; x += channels) {
      if (alphaIndex >= 0 && row[x + alphaIndex] === 0) continue;
      const color = Array.from(row.subarray(x, x + colorChannels));
      if (alphaIndex >= 0) color.push(row[x + alphaIndex]);
      if (!firstVisible) firstVisible = color;
      else if (color.some((value, index) => value !== firstVisible[index])) return true;
    }
    previous = row;
  }
  return false;
};

export const servedDocumentBuildIdentity = (servedDocuments) => {
  const document = Array.isArray(servedDocuments) ? servedDocuments[0] : null;
  if (!document || typeof document.route !== 'string' || !document.route.trim() || typeof document.state !== 'string' || !document.state.trim() || !/^[0-9a-f]{64}$/.test(document.sha256 || '') || !Number.isInteger(document.bytes) || document.bytes <= 0) {
    throw new Error('a measured served document is required to identify the build');
  }
  return {
    build: `sha256:${document.sha256}`,
    build_source: { route: document.route, state: document.state, sha256: document.sha256, bytes: document.bytes }
  };
};
export const stableFeatureId = (kind, route, identity) => `${kind}-${hash(`${route}\0${identity}`)}`;

// A clipped fallback is required because some remote browser backends return
// a blank texture for an unclipped screenshot. Preserve element crops when
// possible; if the locator shot fails, use its live bounding box, then the
// configured viewport as an honest, reviewable fallback.
export const captureFeatureScreenshot = async ({ page, locator, rect, path, timeout = 1000, viewport = { width: 390, height: 844 } }) => {
  const clipToViewport = (box) => {
    if (!box || !Number.isFinite(box.x) || !Number.isFinite(box.y) || !Number.isFinite(box.width) || !Number.isFinite(box.height)) return null;
    const x = Math.max(0, Math.floor(box.x)), y = Math.max(0, Math.floor(box.y));
    const right = Math.min(viewport.width, Math.ceil(box.x + box.width));
    const bottom = Math.min(viewport.height, Math.ceil(box.y + box.height));
    return right > x && bottom > y ? { x, y, width: right - x, height: bottom - y } : null;
  };
  if (locator) {
    try {
      await locator.screenshot({ path, timeout });
      if (pngHasVisiblePixelVariation(await readFile(path))) return 'element';
      rect ||= await locator.boundingBox({ timeout }).catch(() => null);
    } catch {
      rect ||= await locator.boundingBox({ timeout }).catch(() => null);
    }
  }
  const clip = clipToViewport(rect);
  if (clip) {
    try {
      await page.screenshot({ path, clip });
      if (pngHasVisiblePixelVariation(await readFile(path))) return locator ? 'bounded-fallback' : 'rect';
    } catch { /* Fall back to a clipped viewport below. */ }
  }
  await page.screenshot({ path, clip: { x: 0, y: 0, width: viewport.width, height: viewport.height } });
  if (!pngHasVisiblePixelVariation(await readFile(path))) throw new Error('screenshot capture returned a uniform image at element, rectangle, and viewport bounds');
  return 'viewport-fallback';
};

const requiredIdentity = ['scenario', 'build', 'base_url', 'captured_at', 'viewport'];

export const validateInventory = (inventory, label = 'inventory') => {
  const errors = [];
  if (!inventory || typeof inventory !== 'object' || Array.isArray(inventory)) return [`${label} must be an object`];
  if (inventory.schema !== 'bas-feature-inventory/v1') errors.push(`${label}.schema must be bas-feature-inventory/v1`);
  const identity = inventory.identity;
  if (!identity || typeof identity !== 'object' || Array.isArray(identity)) errors.push(`${label}.identity must be an object`);
  else for (const field of requiredIdentity) {
    if (typeof identity[field] !== 'string' || !identity[field].trim()) errors.push(`${label}.identity.${field} must be a non-empty string`);
  }
  const buildSource = identity?.build_source;
  if (buildSource !== undefined) {
    if (!buildSource || typeof buildSource !== 'object' || Array.isArray(buildSource)) errors.push(`${label}.identity.build_source must be an object`);
    else {
      if (typeof buildSource.route !== 'string' || !buildSource.route.trim()) errors.push(`${label}.identity.build_source.route must be a non-empty string`);
      if (typeof buildSource.state !== 'string' || !buildSource.state.trim()) errors.push(`${label}.identity.build_source.state must be a non-empty string`);
      if (typeof buildSource.sha256 !== 'string' || !/^[0-9a-f]{64}$/.test(buildSource.sha256)) errors.push(`${label}.identity.build_source.sha256 must be a full lowercase SHA-256 digest`);
      if (!Number.isInteger(buildSource.bytes) || buildSource.bytes <= 0) errors.push(`${label}.identity.build_source.bytes must be a positive integer`);
      if (typeof buildSource.sha256 === 'string' && identity.build !== `sha256:${buildSource.sha256}`) errors.push(`${label}.identity.build must match the served-document SHA-256`);
    }
  }
  if (!Array.isArray(inventory.features)) errors.push(`${label}.features must be an array`);
  else {
    const ids = new Set();
    const identities = new Set();
    for (const [index, feature] of inventory.features.entries()) {
      const at = `${label}.features[${index}]`;
      if (!feature || typeof feature !== 'object' || Array.isArray(feature)) { errors.push(`${at} must be an object`); continue; }
      for (const field of ['id', 'identity', 'kind', 'route', 'screenshot']) {
        if (typeof feature[field] !== 'string' || !feature[field].trim()) errors.push(`${at}.${field} must be a non-empty string`);
      }
      if (feature.screenshot_capture !== undefined && !['element', 'rect', 'bounded-fallback', 'viewport-fallback'].includes(feature.screenshot_capture)) {
        errors.push(`${at}.screenshot_capture must name a supported capture method`);
      }
      if (typeof feature.id === 'string' && feature.id.trim()) {
        if (ids.has(feature.id)) errors.push(`${label} has duplicate feature id: ${feature.id}`);
        ids.add(feature.id);
      }
      if (['identity', 'kind', 'route'].every((field) => typeof feature[field] === 'string' && feature[field].trim())) {
        const key = `${feature.kind}\0${feature.route}\0${feature.identity}`;
        if (identities.has(key)) errors.push(`${label} has duplicate feature identity: ${feature.kind} at ${feature.route}`);
        identities.add(key);
      }
    }
  }
  const servedDocuments = inventory.coverage?.servedDocuments;
  if (servedDocuments !== undefined && !Array.isArray(servedDocuments)) errors.push(`${label}.coverage.servedDocuments must be an array`);
  else for (const [index, document] of (servedDocuments || []).entries()) {
    const at = `${label}.coverage.servedDocuments[${index}]`;
    if (!document || typeof document !== 'object' || Array.isArray(document)) { errors.push(`${at} must be an object`); continue; }
    if (typeof document.route !== 'string' || !document.route.trim()) errors.push(`${at}.route must be a non-empty string`);
    if (typeof document.state !== 'string' || !document.state.trim()) errors.push(`${at}.state must be a non-empty string`);
    if (typeof document.sha256 !== 'string' || !/^[0-9a-f]{64}$/.test(document.sha256)) errors.push(`${at}.sha256 must be a full lowercase SHA-256 digest`);
    if (!Number.isInteger(document.bytes) || document.bytes <= 0) errors.push(`${at}.bytes must be a positive integer`);
  }
  if (buildSource && Array.isArray(servedDocuments) && !servedDocuments.some((document) => document.route === buildSource.route && document.state === buildSource.state && document.sha256 === buildSource.sha256 && document.bytes === buildSource.bytes)) {
    errors.push(`${label}.identity.build_source must match a measured served document`);
  }
  return errors;
};

export const featureDiffSummary = (before, after) => {
  const counts = (features) => features.reduce((result, feature) => {
    result[feature.kind] = (result[feature.kind] || 0) + 1;
    return result;
  }, {});
  const beforeIds = new Set(before.features.map((feature) => feature.id));
  const afterIds = new Set(after.features.map((feature) => feature.id));
  const added = after.features.filter((feature) => !beforeIds.has(feature.id));
  const removed = before.features.filter((feature) => !afterIds.has(feature.id));
  const kinds = [...new Set([...Object.keys(counts(before.features)), ...Object.keys(counts(after.features))])].sort();
  const beforeKinds = counts(before.features), afterKinds = counts(after.features);
  return {
    before: before.features.length, after: after.features.length,
    added: added.length, removed: removed.length,
    by_kind: Object.fromEntries(kinds.map((kind) => [kind, {
      before: beforeKinds[kind] || 0, after: afterKinds[kind] || 0,
      added: added.filter((feature) => feature.kind === kind).length,
      removed: removed.filter((feature) => feature.kind === kind).length
    }]))
  };
};

// Combine repeated crawls of the same served build into one diff-ready
// baseline. A feature id is stable across captures; duplicate observations are
// deduplicated only when their semantic identities agree.
export const deduplicateFeatureRecords = (features, label = 'features') => {
  if (!Array.isArray(features)) throw new Error(`${label} must be an array`);
  const byId = new Map();
  for (const feature of features) {
    const existing = byId.get(feature.id);
    if (!existing) { byId.set(feature.id, { ...feature }); continue; }
    if (['identity', 'kind', 'route', 'label'].some((field) => existing[field] !== feature[field]) || (existing.source ?? null) !== (feature.source ?? null)) {
      throw new Error(`conflicting semantic identity for duplicate feature ${feature.id} in ${label}`);
    }
    existing.event_props = [...new Set([...(existing.event_props || []), ...(feature.event_props || [])])].sort();
    if (feature.component_evidence === 'react-type-source') existing.component_evidence = feature.component_evidence;
    const sameState = JSON.stringify(existing.state ?? null) === JSON.stringify(feature.state ?? null);
    if (!sameState) {
      existing.state_variants ||= [existing.state ?? null];
      if (!existing.state_variants.some((state) => JSON.stringify(state) === JSON.stringify(feature.state ?? null))) existing.state_variants.push(feature.state ?? null);
    }
  }
  return [...byId.values()];
};

export const mergeFeatureInventories = (inventories) => {
  if (!Array.isArray(inventories) || inventories.length === 0) throw new Error('merge requires at least one inventory');
  const normalized = inventories.map((inventory, index) => ({
    ...inventory,
    features: deduplicateFeatureRecords(inventory.features, `input[${index}].features`)
  }));
  for (const [index, inventory] of normalized.entries()) {
    const errors = validateInventory(inventory, `input[${index}]`);
    if (errors.length) throw new Error(`invalid merge input:\n${errors.map((error) => `- ${error}`).join('\n')}`);
  }
  // source_revision is an optional caller hint, not a build identity. Captures
  // are merge-compatible when their measured served app shell and crawl
  // conditions match, even if an operator supplied different revision labels.
  const identityFields = ['scenario', 'build', 'base_url', 'viewport', 'synthetic_profile'];
  const expected = normalized[0].identity;
  for (const [index, inventory] of normalized.entries()) for (const field of identityFields) {
    if (inventory.identity[field] !== expected[field]) throw new Error(`merge inputs must share identity.${field}; input[${index}] differs`);
  }
  const features = deduplicateFeatureRecords(normalized.flatMap((inventory) => inventory.features), 'merged features');
  const openedActions = new Map();
  const keyboardCandidates = new Map();
  const keyboardObserved = new Map();
  const loadedAssets = new Map();
  const servedDocuments = new Map();
  const unobserved = new Map();
  const sources = normalized.map((inventory, index) => {
    const coverage = inventory.coverage || {};
    for (const action of coverage.openedActions || []) openedActions.set(`${action.route}\0${action.state}`, action);
    for (const item of coverage.keyboard?.candidates || []) keyboardCandidates.set(`${item.state || ''}\0${item.chord}`, item);
    for (const item of coverage.keyboard?.observed || []) keyboardObserved.set(`${item.route}\0${item.state}`, item);
    for (const item of coverage.loadedAssets || []) {
      const key = `${item.route}\0${item.state}`;
      const existing = loadedAssets.get(key) || { route: item.route, state: item.state, scripts: [] };
      existing.scripts = [...new Set([...existing.scripts, ...(item.scripts || [])])].sort();
      loadedAssets.set(key, existing);
    }
    for (const item of coverage.servedDocuments || []) {
      const key = `${item.route}\0${item.state}`;
      const existing = servedDocuments.get(key);
      if (existing && existing.sha256 !== item.sha256) throw new Error(`same-build inventories have different served documents at ${item.route} [${item.state}]`);
      servedDocuments.set(key, item);
    }
    for (const item of coverage.unobserved || []) unobserved.set(`${item.route}\0${item.kind}\0${item.label}\0${item.reason}`, item);
    return {
      index,
      captured_at: inventory.identity.captured_at,
      route_states: [...new Set(inventory.features.filter((feature) => feature.kind === 'route').map((feature) => feature.route))],
      feature_count: inventory.features.length,
      duplicate_records_removed: inventories[index].features.length - inventory.features.length,
      opened_actions: coverage.openedActions || [],
      keyboard: coverage.keyboard || { candidates: [], observed: [] },
      loaded_assets: coverage.loadedAssets || [],
      served_documents: coverage.servedDocuments || [],
      build_hint: inventory.identity.build_hint ?? null,
      source_revision_hint: inventory.identity.source_revision_hint ?? null,
      unobserved_count: (coverage.unobserved || []).length
    };
  });
  const coverage = {
    candidates: Object.fromEntries(['menu', 'disclosure', 'tab'].map((kind) => [kind, Math.max(...normalized.map((i) => i.coverage?.candidates?.[kind] || 0))])),
    observed: Object.fromEntries(['menu', 'disclosure', 'tab'].map((kind) => [kind, Math.max(...normalized.map((i) => i.coverage?.observed?.[kind] || 0))])),
    openedActions: [...openedActions.values()],
    keyboard: { candidates: [...keyboardCandidates.values()], observed: [...keyboardObserved.values()] },
    loadedAssets: [...loadedAssets.values()],
    servedDocuments: [...servedDocuments.values()],
    unobserved: [...unobserved.values()],
    components: {
      semantic: features.filter((feature) => feature.kind === 'component' && feature.component_evidence === 'semantic-region').length,
      react: features.filter((feature) => feature.kind === 'component' && String(feature.component_evidence || '').startsWith('react-')).length,
      withEventProps: features.filter((feature) => feature.kind === 'component' && (feature.event_props || []).length > 0).length
    },
    componentEvidence: 'Unique merged component feature identities from same-build captures; source capture counts and coverage details are retained in sources.',
    routeLimitReached: normalized.some((inventory) => inventory.coverage?.routeLimitReached),
    interactionScope: 'Coverage is the union of the listed same-build captures, not a proof that unsupported interactions or hidden states were explored.',
    unattempted: normalized.reduce((result, inventory) => ({ ...result, ...(inventory.coverage?.unattempted || {}) }), {}),
    sources
  };
  const capturedAt = normalized.map((inventory) => inventory.identity.captured_at).sort()[0];
  return {
    schema: 'bas-feature-inventory/v1',
    identity: { ...expected, captured_at: capturedAt },
    coverage,
    runtime_errors: [...new Set(normalized.flatMap((inventory) => inventory.runtime_errors || []))],
    features
  };
};

// Names and source positions are evidence, not executable React data. Never
// invoke handlers or inspect props beyond their public event-prop names.
export const reactComponentEvidence = ({ name, source = null, eventProps = [] } = {}) => ({
  name: String(name || '').trim(),
  source: source ? String(source).trim() : null,
  eventProps: [...new Set(eventProps.map(String).filter((key) => /^on[A-Z]/.test(key)))].sort()
});

export const componentIdentity = ({ name, source = null, instance = 'single', semanticIdentity = '' } = {}) =>
  source ? `react|${name || 'anonymous'}|${source}|${instance}` : semanticIdentity;

export const reactEventPropNames = (props = {}) => Object.keys(props || {}).filter((key) => /^on[A-Z]/.test(key)).sort();

export const safeInteractionKind = ({ tagName = '', role = '', ariaHaspopup = '', ariaExpanded = null, detailsSummary = false } = {}) => {
  const normalizedRole = String(role).toLowerCase();
  const popup = String(ariaHaspopup).toLowerCase();
  if (normalizedRole === 'tab') return 'tab';
  if (detailsSummary) return 'disclosure';
  if (ariaExpanded === 'false' && ['menu', 'listbox', 'true'].includes(popup)) return 'menu';
  if (ariaExpanded === 'false' && popup !== 'dialog' && (normalizedRole === 'button' || String(tagName).toLowerCase() === 'button')) return 'disclosure';
  return null;
};

export const parseSafeActionLabels = (value = '') => String(value).split(',').map((label) => label.trim().toLocaleLowerCase()).filter(Boolean);

export const matchesSafeActionLabel = (label, allowlist = []) => allowlist.includes(String(label || '').trim().toLocaleLowerCase());

export const parseSafeActionPaths = (value = '') => String(value).split('|').map((path) => path.split('>').map((label) => label.trim()).filter(Boolean)).filter((path) => path.length).map((path) => path.map((label) => label.toLocaleLowerCase()));

// Prefix a path with `<entrypoint>::` to run it only against that named state
// (including `default` for the base pathname). Unscoped legacy paths continue
// to run against every discovered route/state.
export const parseScopedSafeActionPaths = (value = '') => String(value).split('|').map((segment) => {
  const text = segment.trim();
  const scope = /^([a-z0-9][a-z0-9-]{0,39})::\s*(.*)$/.exec(text);
  const pathText = scope ? scope[2] : text;
  const path = parseSafeActionPaths(pathText)[0] || [];
  return path.length ? { state: scope?.[1] ?? null, path } : null;
}).filter(Boolean);

// The synthetic crawler accepts only explicit Ctrl/Cmd+Shift+letter chords.
// This bounded shape supports navigation shortcuts without exposing arbitrary
// typing, Enter/submit, destructive keys, or unscoped keyboard sequences.
export const parseScopedSafeKeyChords = (value = '') => String(value).trim() ? String(value).split('|').map((segment) => {
  const text = segment.trim();
  const scope = /^([a-z0-9][a-z0-9-]{0,39})::\s*(.*)$/.exec(text);
  const chordText = scope ? scope[2] : text;
  const parts = chordText.split('+').map((part) => part.trim()).filter(Boolean);
  if (parts.length !== 3) throw new Error(`safe key chord must be Ctrl/Cmd+Shift+<letter>: ${text}`);
  const aliases = { ctrl: 'Control', control: 'Control', cmd: 'Meta', command: 'Meta', meta: 'Meta', shift: 'Shift' };
  const modifiers = parts.slice(0, -1).map((part) => aliases[part.toLowerCase()]);
  const key = parts.at(-1);
  if (modifiers.some((modifier) => !modifier) || new Set(modifiers).size !== 2 || !modifiers.includes('Shift') || !modifiers.some((modifier) => ['Control', 'Meta'].includes(modifier)) || !/^[a-z]$/i.test(key)) {
    throw new Error(`safe key chord must be Ctrl/Cmd+Shift+<letter>: ${text}`);
  }
  const primary = modifiers.includes('Control') ? 'Ctrl' : 'Cmd';
  const lowerKey = key.toLowerCase();
  return { state: scope?.[1] ?? null, label: `${primary}+Shift+${key.toUpperCase()}`, playwrightKey: `${modifiers.join('+')}+${lowerKey}` };
}).filter(Boolean) : [];

export const parseNamedEntrypoints = (values = [], baseUrl) => {
  const origin = new URL(baseUrl).origin;
  const names = new Set();
  return values.map((value) => {
    const separator = String(value).indexOf('=');
    if (separator < 1) throw new Error(`entrypoint must be NAME=URL: ${value}`);
    const name = String(value).slice(0, separator).trim();
    if (name !== name.toLowerCase()) throw new Error(`entrypoint name must be lowercase: ${name}`);
    if (!/^[a-z0-9][a-z0-9-]{0,39}$/.test(name)) throw new Error(`entrypoint name must be a short lowercase slug: ${name}`);
    if (names.has(name)) throw new Error(`duplicate entrypoint name: ${name}`);
    names.add(name);
    const href = new URL(String(value).slice(separator + 1).trim(), baseUrl);
    if (href.origin !== origin) throw new Error(`entrypoint must stay on the served origin: ${name}`);
    if (href.username || href.password) throw new Error(`entrypoint must not contain credentials: ${name}`);
    href.hash = '';
    return { name, href: href.href };
  });
};

export const routeIdentityForState = (url, state = 'default') => {
  const pathname = new URL(url).pathname || '/';
  return state === 'default' ? pathname : `${pathname} [state=${state}]`;
};

export const inventoryCoverage = () => ({
  candidates: { menu: 0, disclosure: 0, tab: 0 },
  observed: { menu: 0, disclosure: 0, tab: 0 },
  openedActions: [],
  keyboard: { candidates: [], observed: [] },
  loadedAssets: [],
  servedDocuments: [],
  components: { semantic: 0, react: 0, withEventProps: 0 },
  componentEvidence: 'Visible component instances exposed through React fiber/type/source metadata, plus semantic and test-id regions. Native/custom handlers may be inaccessible.',
  unobserved: [],
  unattempted: {
    drag: 'Not synthesized: can move or reorder user data.',
    hover: 'Not synthesized: may reveal transient state with side effects.',
    keyboard: 'Not synthesized: requires an explicit key map.',
    scroll: 'Only normal page rendering was captured; nested scroll regions were not explored.'
  }
});
