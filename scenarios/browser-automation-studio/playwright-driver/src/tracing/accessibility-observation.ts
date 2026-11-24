import type { CDPSession, Page } from 'rebrowser-playwright';
import { randomBytes } from 'node:crypto';
import { createCDPSession, detachCDPSession } from '../session/cdp-session';
import {
  normalizeAccessibilityTree,
  parseDomSnapshot,
  type AccessibilityCDP,
  type AccessibilitySnapshot,
  type DomNodeInfo,
  type NormalizedAXNode,
} from './accessibility-snapshot';

export const ACCESSIBILITY_TREE_OBSERVATION_CONTRACT = 'bas-accessibility-tree-observation/v1';
export const ACCESSIBILITY_TREE_MAX_BYTES = 512 * 1024;
const MAX_RAW_NODES = 5_000;
const MAX_NORMALIZED_NODES = 2_500;
const MAX_TEXT_CHARS = 128 * 1024;
const MAX_FIELD_CHARS = 256;

export interface AccessibilityTreeObservation {
  contract: typeof ACCESSIBILITY_TREE_OBSERVATION_CONTRACT;
  url: string;
  viewport: AccessibilitySnapshot['viewport'];
  /** Minimal, static layout evidence for viewport-fit checks. This contains no
   * DOM text, selectors, styles, or caller-supplied expression. */
  layout: { viewportWidthPx: number; documentWidthPx: number; viewportHeightPx: number; documentHeightPx: number } | null;
  captured_at: string;
  node_count: number;
  truncated: boolean;
  root: NormalizedAXNode;
  /** Private owner-only selector table; stripped before the HTTP response. */
  agentReferenceTargets?: Map<string, { selector: string; role?: string; name?: string }>;
}

/** Build an on-demand tree from the current page. This returns no CDP session,
 * node identifiers, input values, DOM attributes, URLs beyond the origin, or
 * computed styles. The result is bounded before it leaves the driver. */
export async function observeAccessibilityTree(
  page: Page,
  cdpFactory: (page: Page) => Promise<CDPSession> = createCDPSession
): Promise<AccessibilityTreeObservation> {
  let cdp: CDPSession | undefined;
  try {
    cdp = await cdpFactory(page);
    const result = await (cdp as unknown as AccessibilityCDP).send('Accessibility.getFullAXTree');
    const allNodes = Array.isArray(result?.nodes) ? result.nodes : [];
    const rawNodes = allNodes.slice(0, MAX_RAW_NODES);
    const viewport = page.viewportSize() ?? { width: 0, height: 0 };
    let deviceScaleFactor = 1;
    try {
      deviceScaleFactor = await page.evaluate(() => window.devicePixelRatio);
    } catch {
      /* default */
    }
    let layout: AccessibilityTreeObservation['layout'] = null;
    try {
      const observedLayout = await page.evaluate(() => ({
        viewportWidthPx: window.innerWidth,
        documentWidthPx: document.documentElement.scrollWidth,
        viewportHeightPx: window.innerHeight,
        documentHeightPx: document.documentElement.scrollHeight,
      }));
      if (
        observedLayout &&
        Number.isSafeInteger(observedLayout.viewportWidthPx) && observedLayout.viewportWidthPx > 0 &&
        Number.isSafeInteger(observedLayout.documentWidthPx) && observedLayout.documentWidthPx > 0 &&
        Number.isSafeInteger(observedLayout.viewportHeightPx) && observedLayout.viewportHeightPx > 0 &&
        Number.isSafeInteger(observedLayout.documentHeightPx) && observedLayout.documentHeightPx > 0
      ) layout = observedLayout;
    } catch {
      /* Layout evidence is optional for general AX reads; consumers requiring
       * it fail closed when it is unavailable. */
    }
    const url = safeOrigin(page.url());
    let domInfo = new Map<number, DomNodeInfo>();
    try {
      const snapshot = await (cdp as unknown as AccessibilityCDP).send('DOMSnapshot.captureSnapshot', { computedStyles: [] });
      domInfo = parseDomSnapshot(snapshot as never);
    } catch {
      // The AX tree remains useful for reading when DOM metadata is unavailable.
    }
    const normalized = normalizeAccessibilityTree(rawNodes, domInfo, {
      url,
      viewport: { width: viewport.width, height: viewport.height, deviceScaleFactor },
      capturedAt: new Date().toISOString(),
    });
    const referenceTargets = await addOpaqueReferences(page, normalized.root, domInfo);
    const budget = { nodes: 0, text: 0 };
    const root = boundNode(normalized.root, budget);
    const output: AccessibilityTreeObservation = {
      contract: ACCESSIBILITY_TREE_OBSERVATION_CONTRACT,
      url,
      viewport: normalized.viewport,
      layout,
      captured_at: normalized.captured_at,
      node_count: budget.nodes,
      truncated:
        normalized.truncated ||
        allNodes.length > rawNodes.length ||
        budget.nodes < normalized.node_count,
      root,
    };
    Object.defineProperty(output, 'agentReferenceTargets', { value: referenceTargets, enumerable: false });
    // Structural overhead is bounded by MAX_NORMALIZED_NODES and text by
    // MAX_TEXT_CHARS. Keep a hard serialized-size assertion as a final guard.
    if (Buffer.byteLength(JSON.stringify(output), 'utf8') > ACCESSIBILITY_TREE_MAX_BYTES) {
      throw new Error('accessibility tree exceeded the response size limit');
    }
    return output;
  } finally {
    if (cdp) await detachCDPSession(cdp);
  }
}

function safeOrigin(raw: string): string {
  try {
    const url = new URL(raw);
    if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password) return '';
    return url.origin;
  } catch {
    return '';
  }
}

function boundNode(
  source: NormalizedAXNode,
  budget: { nodes: number; text: number }
): NormalizedAXNode {
  if (budget.nodes >= MAX_NORMALIZED_NODES) return { children: [] };
  budget.nodes += 1;
  const result: NormalizedAXNode = { children: [] };
  if (source.ref) result.ref = source.ref;
  if (source.backendDOMNodeId != null) {
    Object.defineProperty(result, 'backendDOMNodeId', { value: source.backendDOMNodeId, enumerable: false });
  }
  const text = (value: string | undefined): string | undefined => {
    if (!value || budget.text >= MAX_TEXT_CHARS) return undefined;
    const remaining = MAX_TEXT_CHARS - budget.text;
    const clipped = redactText(value).slice(0, Math.min(MAX_FIELD_CHARS, remaining));
    budget.text += clipped.length;
    return clipped || undefined;
  };
  if (source.role) result.role = source.role.slice(0, 80);
  result.name = text(source.name);
  result.description = text(source.description);
  // Values, DOM metadata, bounds and style are intentionally excluded from
  // this first live contract. They are not needed for role/name navigation.
  const safeStates = (source.states ?? []).filter(
    (state) => !/(value|text|url|href|token|secret|password|email|phone)/i.test(state)
  );
  const states = safeStates
    .slice(0, 8)
    .map((state) => text(state))
    .filter((state): state is string => !!state);
  if (states.length) result.states = states;
  for (const child of source.children) {
    if (budget.nodes >= MAX_NORMALIZED_NODES || budget.text >= MAX_TEXT_CHARS) break;
    result.children.push(boundNode(child, budget));
  }
  return result;
}

const actionableRoles = new Set([
  'button', 'link', 'textbox', 'searchbox', 'combobox', 'checkbox', 'radio', 'switch',
  'tab', 'menuitem', 'option', 'slider', 'spinbutton', 'treeitem',
]);

async function addOpaqueReferences(
  page: Page,
  root: NormalizedAXNode,
  domInfo: Map<number, DomNodeInfo>
): Promise<Map<string, { selector: string; role?: string; name?: string }>> {
  const targets = new Map<string, { selector: string; role?: string; name?: string }>();
  const visit = async (node: NormalizedAXNode): Promise<void> => {
    const backendNodeId = node.backendDOMNodeId;
    if (backendNodeId != null && node.role && actionableRoles.has(node.role.toLowerCase())) {
      const info = domInfo.get(backendNodeId);
      const selector = await uniqueSelector(page, info);
      if (selector) {
        const ref = `e${BigInt(`0x${randomBytes(16).toString('hex')}`).toString()}`;
        node.ref = ref;
        targets.set(ref, { selector, role: node.role, name: node.name });
      }
    }
    for (const child of node.children) await visit(child);
  };
  await visit(root);
  return targets;
}

async function uniqueSelector(page: Page, info: DomNodeInfo | undefined): Promise<string> {
  if (!info) return '';
  const attributes = info.attributes ?? {};
  const candidates: string[] = [];
  const add = (name: string, value: string | undefined) => {
    if (!value || value.length > 160 || /[\u0000-\u001f]/.test(value)) return;
    const tag = info.tag && /^[a-z][a-z0-9-]*$/i.test(info.tag) ? info.tag.toLowerCase() : '';
    candidates.push(`${tag}[${name}="${escapeCssString(value)}"]`);
  };
  add('id', attributes.id);
  add('data-testid', attributes['data-testid']);
  add('aria-label', attributes['aria-label']);
  add('name', attributes.name);
  for (const [name, value] of Object.entries(attributes)) {
    if (name.startsWith('data-') && name !== 'data-testid') add(name, value);
  }
  for (const candidate of candidates) {
    try {
      if (await page.locator(candidate).count() === 1) return candidate;
    } catch {
      // An invalid or unsupported selector is not a usable action reference.
    }
  }
  return '';
}

function escapeCssString(value: string): string {
  return value.replace(/\\/g, '\\\\').replace(/"/g, '\\"').replace(/[\n\r\f]/g, ' ');
}

function redactText(raw: string): string {
  const assignment =
    /\b(password|passwd|passcode|secret|token|otp|one[-_ ]?time[-_ ]?code|verification[-_ ]?code|cvv|cvc|api[-_ ]?key|access[-_ ]?token|refresh[-_ ]?token)\b\s*([:=])\s*([^\s,;]+)/gi;
  return raw
    .replace(assignment, (_match, key: string, separator: string) => `${key}${separator}[REDACTED]`)
    .replace(/\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi, '[REDACTED]')
    .replace(/\b\+?\d[\d ().-]{7,}\d\b/g, '[REDACTED]');
}
