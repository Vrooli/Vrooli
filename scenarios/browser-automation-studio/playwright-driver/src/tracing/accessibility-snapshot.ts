/**
 * Accessibility Snapshot
 *
 * Captures a normalized snapshot of the Chromium accessibility tree for a
 * session and writes it as JSON to a target directory. This is the
 * driver-side mechanism behind BAS's CAPTURE_TYPE_ACCESSIBILITY artifact.
 *
 * DESIGN CHOICES:
 * - Raw CDP, not the deprecated `page.accessibility.snapshot()`. The
 *   `Accessibility.getFullAXTree` domain exposes the full role/name/state
 *   set plus `backendDOMNodeId`, which we join against DOM geometry +
 *   attributes so downstream consumers can locate and target each node.
 * - Geometry + `data-testid` come from a single `DOMSnapshot.captureSnapshot`
 *   call (one round-trip for the whole document) rather than per-node
 *   `DOM.getBoxModel`/`DOM.describeNode` chatter.
 * - Captured at a SETTLED point (after wait_for + any interaction), mirroring
 *   where the final screenshot fires — the snapshotter is invoked at session
 *   close on the final page, exactly like the PerformanceTracer reads its
 *   web-vitals global there.
 * - NEVER throws on the capture path: a failure degrades to "no file written"
 *   (the artifact is absent, the capture still succeeds). The Go producer
 *   surfaces the absence as an unavailable artifact.
 *
 * The output is normalized to the frozen contract `bas-accessibility-snapshot/v1`.
 * That shape is a seam another scenario builds against — the field names below
 * must not be renamed.
 */

import type { CDPSession, Page } from 'rebrowser-playwright';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { logger, scopedLog, LogContext } from '../utils';
import { createCDPSession, detachCDPSession } from '../session/cdp-session';

/** Canonical artifact filename the Go side reads back. */
export const ACCESSIBILITY_SNAPSHOT_FILE = 'accessibility.json';

/** Frozen contract identifier stamped into every snapshot. */
export const ACCESSIBILITY_CONTRACT = 'bas-accessibility-snapshot/v1';

/**
 * Safety cap on the number of normalized nodes. A pathological page cannot
 * balloon the artifact past this; when exceeded the tree is cut and
 * `truncated` is set true. Chosen well above any real UI's node count.
 */
export const ACCESSIBILITY_MAX_NODES = 20000;

/** CSS properties included in the computed-style evidence contract. */
export const COMPUTED_STYLE_PROPERTIES = [
  'color',
  'background-color',
  'border-color',
  'font-size',
  'line-height',
  'font-weight',
  'letter-spacing',
  'margin',
  'padding',
  'width',
  'height',
  'gap',
  'border-radius',
  'box-shadow',
  'opacity',
  'transition-duration',
  'transition-timing-function',
] as const;

// --- CDP payload shapes (minimal, only the fields we read) ------------------

/** An AXValue as returned by CDP Accessibility.getFullAXTree. */
export interface AXValue {
  type: string;
  value?: unknown;
}

/** An AXProperty (state/relationship) on an AX node. */
export interface AXProperty {
  name: string;
  value?: AXValue;
}

/** A raw AX node from Accessibility.getFullAXTree. */
export interface RawAXNode {
  nodeId: string;
  ignored?: boolean;
  role?: AXValue;
  name?: AXValue;
  description?: AXValue;
  value?: AXValue;
  properties?: AXProperty[];
  childIds?: string[];
  backendDOMNodeId?: number;
}

/** The `nodes` wrapper Accessibility.getFullAXTree returns. */
export interface AXTreeResult {
  nodes: RawAXNode[];
}

/** Geometry + DOM attributes joined per backend DOM node id. */
export interface DomNodeInfo {
  tag?: string;
  testid?: string;
  /** Stable DOM attributes used by downstream experience contracts. */
  attributes?: Record<string, string>;
  bounds?: { x: number; y: number; width: number; height: number };
  computedStyle?: Record<string, string>;
}

// --- Minimal CDP command surface (declared for the test seam) ---------------

/**
 * The CDP commands the snapshotter issues. Declaring the surface lets the
 * unit test drive capture() with a fake CDP session (no real browser) and
 * documents exactly which CDP calls are made.
 */
export interface AccessibilityCDP {
  send(
    method: 'Accessibility.getFullAXTree',
    params?: Record<string, unknown>
  ): Promise<AXTreeResult>;
  send(
    method: 'DOMSnapshot.captureSnapshot',
    params: Record<string, unknown>
  ): Promise<DOMSnapshotResult>;
  send(method: string, params?: Record<string, unknown>): Promise<unknown>;
}

/** The subset of DOMSnapshot.captureSnapshot output we parse. */
export interface DOMSnapshotResult {
  documents?: Array<{
    nodes?: {
      backendNodeId?: number[];
      nodeName?: number[];
      attributes?: number[][];
      computedStyles?: number[][];
    };
    layout?: {
      nodeIndex?: number[];
      bounds?: number[][];
    };
  }>;
  strings?: string[];
}

// --- Normalized output (the frozen contract) --------------------------------

/** A normalized node in the bas-accessibility-snapshot/v1 tree. */
export interface NormalizedAXNode {
  role?: string;
  name?: string;
  description?: string;
  value?: string;
  states?: string[];
  bounds?: { x: number; y: number; width: number; height: number };
  dom?: { testid?: string; tag?: string; attributes?: Record<string, string> };
  computedStyle?: Record<string, string>;
  children: NormalizedAXNode[];
}

/** The full bas-accessibility-snapshot/v1 document. */
export interface AccessibilitySnapshot {
  contract: string;
  url: string;
  viewport: { width: number; height: number; deviceScaleFactor: number };
  captured_at: string;
  node_count: number;
  truncated: boolean;
  root: NormalizedAXNode;
  meta: { frames: string; source: string };
}

/** Metadata threaded into the normalizer (all page-derived). */
export interface SnapshotMeta {
  url: string;
  viewport: { width: number; height: number; deviceScaleFactor: number };
  capturedAt: string;
}

// --- Pure normalization (the golden-test target) ----------------------------

/**
 * State property value types we surface as `states`. Relationship-valued
 * properties (idref/idrefList/nodeList/node) are relationships, not states,
 * and are excluded so `states` stays a flat, human-readable token list.
 */
const RELATIONSHIP_VALUE_TYPES = new Set(['idref', 'idrefList', 'nodeList', 'node']);

/** Extract the scalar string form of an AXValue, or '' when absent/empty. */
function axString(v?: AXValue): string {
  if (!v || v.value == null) {
    return '';
  }
  if (typeof v.value === 'string') {
    return v.value;
  }
  return String(v.value);
}

/**
 * Derive the `states` token list from an AX node's properties. Boolean
 * properties contribute their bare name when true (and are dropped when
 * false/undefined); scalar-valued properties contribute `name=value`.
 * Relationship properties are skipped.
 */
export function deriveStates(node: RawAXNode): string[] {
  const states: string[] = [];
  for (const prop of node.properties ?? []) {
    const value = prop.value;
    if (!value || RELATIONSHIP_VALUE_TYPES.has(value.type)) {
      continue;
    }
    if (value.type === 'boolean' || value.type === 'booleanOrUndefined') {
      if (value.value === true) {
        states.push(prop.name);
      }
      continue;
    }
    // Scalar (tristate/token/string/integer/number/…). Drop empties and the
    // tristate "false" default so the list stays signal, not noise.
    const scalar = axString(value);
    if (scalar === '' || scalar === 'false') {
      continue;
    }
    states.push(`${prop.name}=${scalar}`);
  }
  return states;
}

/**
 * Normalize a raw AX tree to the bas-accessibility-snapshot/v1 contract.
 *
 * Pure: no CDP, no I/O. Given the flat AX node list, a backend-node-id → DOM
 * info map, and page-derived meta, it returns the full snapshot document.
 *
 * Rules:
 *   - Ignored AX nodes are pruned; their children are spliced up into the
 *     parent's child list (so the tree carries only meaningful nodes).
 *   - `bounds`, `dom`, and empty-string scalar fields are omitted rather than
 *     emitted as null/"".
 *   - Node count is capped at ACCESSIBILITY_MAX_NODES; overflow cuts the tree
 *     and sets `truncated`.
 */
export function normalizeAccessibilityTree(
  rawNodes: RawAXNode[],
  domInfo: Map<number, DomNodeInfo>,
  meta: SnapshotMeta
): AccessibilitySnapshot {
  const byId = new Map<string, RawAXNode>();
  for (const n of rawNodes) {
    byId.set(n.nodeId, n);
  }

  const counter = { count: 0, truncated: false };

  const buildDom = (
    node: RawAXNode
  ): { testid?: string; tag?: string; attributes?: Record<string, string> } | undefined => {
    if (node.backendDOMNodeId == null) {
      return undefined;
    }
    const info = domInfo.get(node.backendDOMNodeId);
    if (!info) {
      return undefined;
    }
    const dom: { testid?: string; tag?: string; attributes?: Record<string, string> } = {};
    if (info.testid) {
      dom.testid = info.testid;
    }
    if (info.tag) {
      dom.tag = info.tag;
    }
    if (info.attributes && Object.keys(info.attributes).length > 0) {
      dom.attributes = info.attributes;
    }
    return dom.testid || dom.tag ? dom : undefined;
  };

  const bounds = (node: RawAXNode): DomNodeInfo['bounds'] | undefined => {
    if (node.backendDOMNodeId == null) {
      return undefined;
    }
    return domInfo.get(node.backendDOMNodeId)?.bounds;
  };

  // Returns the normalized node(s) that should take this node's place in the
  // parent's child list: [self] for a kept node, or its (already-spliced)
  // children for an ignored one.
  const normalize = (nodeId: string, seen: Set<string>): NormalizedAXNode[] => {
    const node = byId.get(nodeId);
    if (!node || seen.has(nodeId)) {
      return [];
    }
    seen.add(nodeId);

    const childrenOf = (): NormalizedAXNode[] => {
      const kids: NormalizedAXNode[] = [];
      for (const childId of node.childIds ?? []) {
        kids.push(...normalize(childId, seen));
      }
      return kids;
    };

    // Prune ignored nodes: splice their children up into the parent. Ignored
    // nodes are not counted and do not consume the cap.
    if (node.ignored) {
      return childrenOf();
    }

    // Count self before recursing so the cap holds exactly: once the budget is
    // spent, this (and every later) kept node is dropped and `truncated` set.
    if (counter.count >= ACCESSIBILITY_MAX_NODES) {
      counter.truncated = true;
      return [];
    }
    counter.count += 1;

    const kids = childrenOf();
    const out: NormalizedAXNode = { children: kids };
    const role = axString(node.role);
    const name = axString(node.name);
    const description = axString(node.description);
    const value = axString(node.value);
    if (role) {
      out.role = role;
    }
    if (name) {
      out.name = name;
    }
    if (description) {
      out.description = description;
    }
    if (value) {
      out.value = value;
    }
    const states = deriveStates(node);
    if (states.length > 0) {
      out.states = states;
    }
    const b = bounds(node);
    if (b) {
      out.bounds = b;
    }
    const dom = buildDom(node);
    if (dom) {
      out.dom = dom;
    }
    if (node.backendDOMNodeId != null) {
      const computedStyle = domInfo.get(node.backendDOMNodeId)?.computedStyle;
      if (computedStyle && Object.keys(computedStyle).length > 0) {
        out.computedStyle = computedStyle;
      }
    }
    return [out];
  };

  const rootId = findRootId(rawNodes);
  let root: NormalizedAXNode = { children: [] };
  if (rootId != null) {
    const normalized = normalize(rootId, new Set());
    if (normalized.length === 1 && normalized[0]) {
      root = normalized[0];
    } else if (normalized.length > 1) {
      // The root was ignored and spliced away; wrap its survivors so the
      // contract always has a single root. Count the synthetic root.
      counter.count += 1;
      root = { role: 'RootWebArea', children: normalized };
    }
  }

  return {
    contract: ACCESSIBILITY_CONTRACT,
    url: meta.url,
    viewport: meta.viewport,
    captured_at: meta.capturedAt,
    node_count: counter.count,
    truncated: counter.truncated,
    root,
    meta: { frames: 'main-only', source: 'cdp-accessibility' },
  };
}

/**
 * Find the AX tree root: the first node not referenced as any node's child.
 * getFullAXTree lists the RootWebArea first, but computing it from the child
 * references is robust to ordering.
 */
function findRootId(rawNodes: RawAXNode[]): string | undefined {
  if (rawNodes.length === 0) {
    return undefined;
  }
  const children = new Set<string>();
  for (const n of rawNodes) {
    for (const c of n.childIds ?? []) {
      children.add(c);
    }
  }
  for (const n of rawNodes) {
    if (!children.has(n.nodeId)) {
      return n.nodeId;
    }
  }
  return rawNodes[0]?.nodeId;
}

/**
 * Parse a DOMSnapshot.captureSnapshot payload into a backend-node-id → DOM
 * info map (tag, stable DOM attributes, layout bounds). Layout bounds are the
 * document-relative CSS-pixel box DOMSnapshot reports.
 */
export function parseDomSnapshot(snapshot: DOMSnapshotResult): Map<number, DomNodeInfo> {
  const out = new Map<number, DomNodeInfo>();
  const strings = snapshot.strings ?? [];
  const str = (idx: number | undefined): string | undefined =>
    idx == null || idx < 0 ? undefined : strings[idx];

  for (const doc of snapshot.documents ?? []) {
    const nodes = doc.nodes ?? {};
    const backendIds = nodes.backendNodeId ?? [];
    const nodeNames = nodes.nodeName ?? [];
    const attributes = nodes.attributes ?? [];

    // node index → bounds, from the layout arrays.
    const layoutBounds = new Map<number, number[]>();
    const layout = doc.layout ?? {};
    const nodeIndex = layout.nodeIndex ?? [];
    const boundsList = layout.bounds ?? [];
    for (let j = 0; j < nodeIndex.length; j += 1) {
      const ni = nodeIndex[j];
      const b = boundsList[j];
      if (ni != null && b && b.length >= 4) {
        layoutBounds.set(ni, b);
      }
    }

    for (let i = 0; i < backendIds.length; i += 1) {
      const backendId = backendIds[i];
      if (backendId == null) {
        continue;
      }
      const info: DomNodeInfo = {};

      const rawTag = str(nodeNames[i]);
      if (rawTag && !rawTag.startsWith('#')) {
        info.tag = rawTag.toLowerCase();
      }

      const attrs = attributes[i] ?? [];
      const stableAttributes: Record<string, string> = {};
      for (let a = 0; a + 1 < attrs.length; a += 2) {
        const name = str(attrs[a]);
        const value = str(attrs[a + 1]);
        if (!name || value == null) {
          continue;
        }
        if (name === 'data-testid') {
          info.testid = value;
        }
        if (
          name.startsWith('data-') ||
          name === 'id' ||
          name === 'name' ||
          name === 'type' ||
          name.startsWith('aria-')
        ) {
          stableAttributes[name] = value;
        }
      }
      if (Object.keys(stableAttributes).length > 0) {
        info.attributes = stableAttributes;
      }

      const b = layoutBounds.get(i);
      if (b && b.length >= 4) {
        info.bounds = { x: b[0] ?? 0, y: b[1] ?? 0, width: b[2] ?? 0, height: b[3] ?? 0 };
      }

      const styleValues = nodes.computedStyles?.[i] ?? [];
      if (styleValues.length > 0) {
        const computedStyle: Record<string, string> = {};
        for (
          let propertyIndex = 0;
          propertyIndex < COMPUTED_STYLE_PROPERTIES.length;
          propertyIndex += 1
        ) {
          const value = str(styleValues[propertyIndex]);
          const property = COMPUTED_STYLE_PROPERTIES[propertyIndex];
          if (value != null && property != null) {
            computedStyle[property] = value;
          }
        }
        if (Object.keys(computedStyle).length > 0) {
          info.computedStyle = computedStyle;
        }
      }

      if (info.tag || info.testid || info.attributes || info.bounds || info.computedStyle) {
        out.set(backendId, info);
      }
    }
  }
  return out;
}

/**
 * Chromium versions used by the managed driver have intermittently returned
 * geometry and attributes from DOMSnapshot.captureSnapshot while omitting its
 * optional computedStyles arrays. The page is still live at this point, so
 * collect styles for declared test-id surfaces as a deterministic fallback.
 * This also preserves pseudo-class state (for example :hover) after the
 * interaction profile has been applied.
 */
async function computedStylesByTestID(page: Page): Promise<Map<string, Record<string, string>>> {
  const values = await page.evaluate(
    (properties: readonly string[]) => {
      const result: Array<{ testid: string; style: Record<string, string> }> = [];
      for (const element of document.querySelectorAll<HTMLElement>('[data-testid]')) {
        const testid = element.getAttribute('data-testid');
        if (!testid) continue;
        const computed = getComputedStyle(element);
        const style: Record<string, string> = {};
        for (const property of properties) {
          const value = computed.getPropertyValue(property).trim();
          if (value) style[property] = value;
        }
        if (Object.keys(style).length > 0) result.push({ testid, style });
      }
      return result;
    },
    [...COMPUTED_STYLE_PROPERTIES]
  );
  return new Map(values.map(({ testid, style }) => [testid, style]));
}

function mergeComputedStyles(
  domInfo: Map<number, DomNodeInfo>,
  stylesByTestID: Map<string, Record<string, string>>
): void {
  if (stylesByTestID.size === 0) return;
  for (const info of domInfo.values()) {
    if (!info.testid || info.computedStyle) continue;
    const style = stylesByTestID.get(info.testid);
    if (style) info.computedStyle = style;
  }
}

/**
 * AccessibilitySnapshotter captures the AX tree for a page at a settled point
 * and writes the normalized snapshot to `<outDir>/accessibility.json`.
 *
 * Usage (session close, on the final settled page):
 *   const snapshotter = new AccessibilitySnapshotter(accessibilityDir);
 *   await snapshotter.capture(session.page);
 */
export class AccessibilitySnapshotter {
  private readonly outDir: string;

  /** Test seam: override CDP creation (defaults to a real per-page session). */
  constructor(
    outDir: string,
    private readonly cdpFactory: (page: Page) => Promise<CDPSession> = createCDPSession
  ) {
    this.outDir = outDir;
  }

  /**
   * Capture and write the snapshot. Best-effort and never throws: any failure
   * (no browser, CDP error, write error) leaves no file, which the Go producer
   * surfaces as an unavailable artifact — the capture itself still succeeds.
   */
  async capture(page: Page): Promise<void> {
    let cdp: CDPSession | undefined;
    try {
      cdp = await this.cdpFactory(page);
      const axcdp = cdp as unknown as AccessibilityCDP;

      const axResult = await axcdp.send('Accessibility.getFullAXTree');
      const rawNodes = axResult?.nodes ?? [];

      // Geometry + attributes are a best-effort join: a DOMSnapshot failure
      // still yields a role/name/state tree, just without bounds/dom.
      let domInfo = new Map<number, DomNodeInfo>();
      try {
        const snap = await axcdp.send('DOMSnapshot.captureSnapshot', {
          computedStyles: [...COMPUTED_STYLE_PROPERTIES],
        });
        domInfo = parseDomSnapshot(snap);
        if (![...domInfo.values()].some((info) => info.computedStyle)) {
          mergeComputedStyles(domInfo, await computedStylesByTestID(page));
        }
      } catch (error) {
        logger.warn(scopedLog(LogContext.TELEMETRY, 'accessibility DOM join failed'), {
          error: error instanceof Error ? error.message : String(error),
          hint: 'snapshot will omit bounds/dom for its nodes',
        });
      }

      const snapshot = normalizeAccessibilityTree(rawNodes, domInfo, {
        url: safePageUrl(page),
        viewport: await safeViewport(page),
        capturedAt: new Date().toISOString(),
      });

      await mkdir(this.outDir, { recursive: true });
      await writeFile(
        path.join(this.outDir, ACCESSIBILITY_SNAPSHOT_FILE),
        JSON.stringify(snapshot, null, 2),
        'utf8'
      );
      logger.info(scopedLog(LogContext.TELEMETRY, 'accessibility snapshot written'), {
        nodes: snapshot.node_count,
        truncated: snapshot.truncated,
        file: ACCESSIBILITY_SNAPSHOT_FILE,
      });
    } catch (error) {
      logger.warn(scopedLog(LogContext.TELEMETRY, 'accessibility snapshot failed'), {
        error: error instanceof Error ? error.message : String(error),
        hint: 'capture continues without an accessibility artifact',
      });
    } finally {
      if (cdp) {
        await detachCDPSession(cdp);
      }
    }
  }
}

/** Read page.url() defensively (a torn-down page can throw). */
function safePageUrl(page: Page): string {
  try {
    return page.url();
  } catch {
    return '';
  }
}

/** Read the viewport + devicePixelRatio defensively. */
async function safeViewport(
  page: Page
): Promise<{ width: number; height: number; deviceScaleFactor: number }> {
  let width = 0;
  let height = 0;
  let deviceScaleFactor = 1;
  try {
    const vp = page.viewportSize();
    if (vp) {
      width = vp.width;
      height = vp.height;
    }
  } catch {
    // fall through to defaults
  }
  try {
    deviceScaleFactor = await page.evaluate(() => window.devicePixelRatio);
  } catch {
    deviceScaleFactor = 1;
  }
  return { width, height, deviceScaleFactor };
}
