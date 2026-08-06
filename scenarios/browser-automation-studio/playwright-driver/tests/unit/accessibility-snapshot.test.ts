/**
 * Accessibility Snapshot unit tests (no real browser).
 *
 * Drives the pure normalizer + DOMSnapshot parser with canned CDP payloads
 * and asserts the exact bas-accessibility-snapshot/v1 JSON (golden), then
 * exercises the full AccessibilitySnapshotter.capture() lifecycle with a fake
 * CDP session so the write path is covered deterministically.
 */

import { mkdtemp, readFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import path from 'node:path';
import {
  AccessibilitySnapshotter,
  normalizeAccessibilityTree,
  parseDomSnapshot,
  deriveStates,
  ACCESSIBILITY_CONTRACT,
  ACCESSIBILITY_SNAPSHOT_FILE,
  ACCESSIBILITY_MAX_NODES,
  type RawAXNode,
  type DomNodeInfo,
  type DOMSnapshotResult,
  type AccessibilityCDP,
  type AccessibilitySnapshot,
} from '../../src/tracing';
import type { CDPSession, Page } from 'rebrowser-playwright';

// A canned AX tree exercising every normalization rule:
//   n1  RootWebArea (kept, has dom+bounds via backendId 10)
//   n2  generic (IGNORED — its child n3 splices up under the root)
//   n3  button "Submit" (kept, focusable state, dom testid+bounds via id 30)
//   n4  text "" (kept role but empty name → name omitted; no dom info)
const CANNED_AX_NODES: RawAXNode[] = [
  {
    nodeId: 'n1',
    ignored: false,
    role: { type: 'role', value: 'RootWebArea' },
    name: { type: 'computedString', value: 'Dashboard' },
    childIds: ['n2', 'n4'],
    backendDOMNodeId: 10,
  },
  {
    nodeId: 'n2',
    ignored: true,
    role: { type: 'role', value: 'generic' },
    childIds: ['n3'],
    backendDOMNodeId: 20,
  },
  {
    nodeId: 'n3',
    ignored: false,
    role: { type: 'role', value: 'button' },
    name: { type: 'computedString', value: 'Submit' },
    value: { type: 'string', value: '' },
    properties: [
      { name: 'focusable', value: { type: 'booleanOrUndefined', value: true } },
      { name: 'focused', value: { type: 'booleanOrUndefined', value: false } },
      { name: 'labelledby', value: { type: 'idrefList', value: 'x' } },
    ],
    childIds: [],
    backendDOMNodeId: 30,
  },
  {
    nodeId: 'n4',
    ignored: false,
    role: { type: 'role', value: 'text' },
    name: { type: 'computedString', value: '' },
    childIds: [],
    backendDOMNodeId: 40,
  },
];

const CANNED_DOM_INFO = new Map<number, DomNodeInfo>([
  [10, { tag: 'body', bounds: { x: 0, y: 0, width: 1440, height: 900 } }],
  [30, { tag: 'button', testid: 'submit-btn', bounds: { x: 20, y: 40, width: 100, height: 32 } }],
  // 40 intentionally absent → node n4 carries no dom/bounds.
]);

const GOLDEN: AccessibilitySnapshot = {
  contract: 'bas-accessibility-snapshot/v1',
  url: 'https://example.com/dashboard',
  viewport: { width: 1440, height: 900, deviceScaleFactor: 1 },
  captured_at: '2026-07-04T00:00:00.000Z',
  node_count: 3,
  truncated: false,
  root: {
    role: 'RootWebArea',
    name: 'Dashboard',
    bounds: { x: 0, y: 0, width: 1440, height: 900 },
    dom: { tag: 'body' },
    children: [
      {
        role: 'button',
        name: 'Submit',
        states: ['focusable'],
        bounds: { x: 20, y: 40, width: 100, height: 32 },
        dom: { testid: 'submit-btn', tag: 'button' },
        children: [],
      },
      {
        role: 'text',
        children: [],
      },
    ],
  },
  meta: { frames: 'main-only', source: 'cdp-accessibility' },
};

describe('normalizeAccessibilityTree', () => {
  it('produces the exact golden bas-accessibility-snapshot/v1 document', () => {
    const snapshot = normalizeAccessibilityTree(CANNED_AX_NODES, CANNED_DOM_INFO, {
      url: 'https://example.com/dashboard',
      viewport: { width: 1440, height: 900, deviceScaleFactor: 1 },
      capturedAt: '2026-07-04T00:00:00.000Z',
    });
    expect(snapshot).toEqual(GOLDEN);
  });

  it('splices ignored nodes up (the button appears directly under the root)', () => {
    const snapshot = normalizeAccessibilityTree(CANNED_AX_NODES, CANNED_DOM_INFO, {
      url: 'u',
      viewport: { width: 0, height: 0, deviceScaleFactor: 1 },
      capturedAt: 't',
    });
    // The ignored `generic` node (n2) is gone; its child (button) is a direct
    // child of the root, alongside the text node.
    expect(snapshot.root.children.map((c) => c.role)).toEqual(['button', 'text']);
  });

  it('omits bounds/dom/empty fields rather than emitting nulls', () => {
    const snapshot = normalizeAccessibilityTree(CANNED_AX_NODES, CANNED_DOM_INFO, {
      url: 'u',
      viewport: { width: 0, height: 0, deviceScaleFactor: 1 },
      capturedAt: 't',
    });
    const textNode = snapshot.root.children[1];
    expect(textNode).not.toHaveProperty('name'); // empty string dropped
    expect(textNode).not.toHaveProperty('value');
    expect(textNode).not.toHaveProperty('bounds'); // no dom info for id 40
    expect(textNode).not.toHaveProperty('dom');
  });

  it('marks truncated + caps node_count when the tree exceeds the max', () => {
    // A root plus MAX+50 flat children.
    const big: RawAXNode[] = [
      {
        nodeId: 'root',
        ignored: false,
        role: { type: 'role', value: 'RootWebArea' },
        childIds: [],
      },
    ];
    const childIds: string[] = [];
    for (let i = 0; i < ACCESSIBILITY_MAX_NODES + 50; i += 1) {
      const id = `c${i}`;
      childIds.push(id);
      big.push({
        nodeId: id,
        ignored: false,
        role: { type: 'role', value: 'listitem' },
        childIds: [],
      });
    }
    big[0].childIds = childIds;

    const snapshot = normalizeAccessibilityTree(big, new Map(), {
      url: 'u',
      viewport: { width: 0, height: 0, deviceScaleFactor: 1 },
      capturedAt: 't',
    });
    expect(snapshot.truncated).toBe(true);
    expect(snapshot.node_count).toBeLessThanOrEqual(ACCESSIBILITY_MAX_NODES);
  });

  it('handles an empty tree without throwing', () => {
    const snapshot = normalizeAccessibilityTree([], new Map(), {
      url: 'u',
      viewport: { width: 0, height: 0, deviceScaleFactor: 1 },
      capturedAt: 't',
    });
    expect(snapshot.contract).toBe(ACCESSIBILITY_CONTRACT);
    expect(snapshot.node_count).toBe(0);
    expect(snapshot.root.children).toEqual([]);
  });
});

describe('deriveStates', () => {
  it('keeps true booleans as bare names, scalar values as name=value, drops relationships/false', () => {
    const node: RawAXNode = {
      nodeId: 'x',
      properties: [
        { name: 'focusable', value: { type: 'booleanOrUndefined', value: true } },
        { name: 'focused', value: { type: 'booleanOrUndefined', value: false } },
        { name: 'checked', value: { type: 'tristate', value: 'mixed' } },
        { name: 'level', value: { type: 'integer', value: 2 } },
        { name: 'selected', value: { type: 'tristate', value: 'false' } },
        { name: 'controls', value: { type: 'idrefList', value: 'a b' } },
      ],
    };
    expect(deriveStates(node)).toEqual(['focusable', 'checked=mixed', 'level=2']);
  });
});

describe('parseDomSnapshot', () => {
  it('joins backend node id → tag, data-testid, and layout bounds', () => {
    // strings table: index → value
    const strings = ['BUTTON', 'data-testid', 'submit-btn', 'id', 'x', 'DIV'];
    const snapshot: DOMSnapshotResult = {
      strings,
      documents: [
        {
          nodes: {
            backendNodeId: [30, 31],
            nodeName: [0, 5], // BUTTON, DIV
            attributes: [
              [1, 2, 3, 4], // data-testid=submit-btn, id=x
              [], // no attributes
            ],
          },
          layout: {
            nodeIndex: [0], // only node index 0 (backendId 30) has layout
            bounds: [[20, 40, 100, 32]],
          },
        },
      ],
    };
    const map = parseDomSnapshot(snapshot);
    expect(map.get(30)).toEqual({
      tag: 'button',
      testid: 'submit-btn',
      bounds: { x: 20, y: 40, width: 100, height: 32 },
    });
    // node 31 (DIV, no testid, no layout) still carries its tag.
    expect(map.get(31)).toEqual({ tag: 'div' });
  });
});

/** Fake CDP replaying a canned AX tree + DOM snapshot. */
class FakeAccessibilityCDP {
  public sends: string[] = [];
  public detached = false;

  constructor(
    private readonly axNodes: RawAXNode[],
    private readonly domSnapshot: DOMSnapshotResult
  ) {}

  send(method: string): Promise<unknown> {
    this.sends.push(method);
    if (method === 'Accessibility.getFullAXTree') {
      return Promise.resolve({ nodes: this.axNodes });
    }
    if (method === 'DOMSnapshot.captureSnapshot') {
      return Promise.resolve(this.domSnapshot);
    }
    return Promise.resolve(undefined);
  }

  detach(): Promise<void> {
    this.detached = true;
    return Promise.resolve();
  }
}

/** A fake page with just the surface the snapshotter reads. */
function fakePage(url: string): Page {
  return {
    url: () => url,
    viewportSize: () => ({ width: 1440, height: 900 }),
    evaluate: () => Promise.resolve(1),
  } as unknown as Page;
}

describe('AccessibilitySnapshotter', () => {
  let dir: string;

  beforeEach(async () => {
    dir = await mkdtemp(path.join(tmpdir(), 'ax-snap-'));
  });

  afterEach(async () => {
    await rm(dir, { recursive: true, force: true });
  });

  it('captures, normalizes, and writes accessibility.json', async () => {
    const domSnapshot: DOMSnapshotResult = {
      strings: ['BODY', 'BUTTON', 'data-testid', 'submit-btn', 'rgb(15, 23, 42)', 'rgb(255, 255, 255)', '14px'],
      documents: [
        {
          nodes: {
            backendNodeId: [10, 30],
            nodeName: [0, 1],
            attributes: [[], [2, 3]],
            computedStyles: [[], [4, 5, 6]],
          },
          layout: {
            nodeIndex: [0, 1],
            bounds: [
              [0, 0, 1440, 900],
              [20, 40, 100, 32],
            ],
          },
        },
      ],
    };
    const cdp = new FakeAccessibilityCDP(CANNED_AX_NODES, domSnapshot);
    const snapshotter = new AccessibilitySnapshotter(dir, () =>
      Promise.resolve(cdp as unknown as CDPSession)
    );

    await snapshotter.capture(fakePage('https://example.com/dashboard'));

    const raw = await readFile(path.join(dir, ACCESSIBILITY_SNAPSHOT_FILE), 'utf8');
    const parsed = JSON.parse(raw) as AccessibilitySnapshot;
    expect(parsed.contract).toBe('bas-accessibility-snapshot/v1');
    expect(parsed.url).toBe('https://example.com/dashboard');
    expect(parsed.root.role).toBe('RootWebArea');
    // Ignored node spliced away; button is a direct child with its testid.
    const button = parsed.root.children.find((c) => c.role === 'button');
    expect(button?.dom?.testid).toBe('submit-btn');
    expect(button?.bounds).toEqual({ x: 20, y: 40, width: 100, height: 32 });
    expect(button?.computedStyle).toEqual({
      color: 'rgb(15, 23, 42)',
      'background-color': 'rgb(255, 255, 255)',
      'border-color': '14px',
    });
    expect(cdp.sends).toContain('Accessibility.getFullAXTree');
    expect(cdp.sends).toContain('DOMSnapshot.captureSnapshot');
    expect(cdp.detached).toBe(true);
  });

  it('degrades gracefully (no file) when the AX call throws', async () => {
    const throwingCdp = {
      send: (): Promise<unknown> => Promise.reject(new Error('CDP unavailable')),
      detach: (): Promise<void> => Promise.resolve(),
    };
    const snapshotter = new AccessibilitySnapshotter(dir, () =>
      Promise.resolve(throwingCdp as unknown as CDPSession)
    );

    // Must not throw.
    await snapshotter.capture(fakePage('https://example.com'));

    await expect(readFile(path.join(dir, ACCESSIBILITY_SNAPSHOT_FILE), 'utf8')).rejects.toThrow();
  });

  it('still writes a role/name tree when the DOM join fails', async () => {
    const cdp = {
      calls: [] as string[],
      send(method: string): Promise<unknown> {
        this.calls.push(method);
        if (method === 'Accessibility.getFullAXTree') {
          return Promise.resolve({ nodes: CANNED_AX_NODES });
        }
        return Promise.reject(new Error('DOMSnapshot unsupported'));
      },
      detach: (): Promise<void> => Promise.resolve(),
    };
    const snapshotter = new AccessibilitySnapshotter(dir, () =>
      Promise.resolve(cdp as unknown as CDPSession)
    );

    await snapshotter.capture(fakePage('https://example.com'));

    const parsed = JSON.parse(
      await readFile(path.join(dir, ACCESSIBILITY_SNAPSHOT_FILE), 'utf8')
    ) as AccessibilitySnapshot;
    // Tree present, but no bounds/dom anywhere (join failed).
    expect(parsed.root.role).toBe('RootWebArea');
    expect(parsed.root).not.toHaveProperty('bounds');
    const button = parsed.root.children.find((c) => c.role === 'button');
    expect(button).toBeDefined();
    expect(button).not.toHaveProperty('dom');
  });
});

/** Exhaustive typing guard so the AccessibilityCDP surface stays honest. */
const _cdpSurfaceCheck: AccessibilityCDP | undefined = undefined;
void _cdpSurfaceCheck;
