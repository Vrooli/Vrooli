import {
  observeAccessibilityTree,
  ACCESSIBILITY_TREE_MAX_BYTES,
} from '../../../src/tracing/accessibility-observation';
import type { Page } from 'rebrowser-playwright';

function fakePage() {
  return {
    url: () => 'https://allowed.example/path?session=private#fragment',
    viewportSize: () => ({ width: 1280, height: 720 }),
    evaluate: async (operation: unknown) => String(operation).includes('document.documentElement.scrollWidth')
      ? { viewportWidthPx: 390, documentWidthPx: 420, viewportHeightPx: 800, documentHeightPx: 860 }
      : 1,
  } as unknown as Page;
}

describe('observeAccessibilityTree', () => {
  test('returns a bounded, normalized current tree without values or node handles', async () => {
    const childIds = Array.from({ length: 3_100 }, (_, index) => `node-${index}`);
    const nodes = [
      {
        nodeId: 'root',
        role: { type: 'role', value: 'RootWebArea' },
        name: { type: 'string', value: 'alex@example.test' },
        value: { type: 'string', value: 'should never leave the driver' },
        childIds,
      },
      ...childIds.map((nodeId) => ({
        nodeId,
        role: { type: 'role', value: 'button' },
        name: { type: 'string', value: `Button ${nodeId}` },
      })),
    ];
    const cdp = {
      send: async (method: string) => (method === 'Accessibility.getFullAXTree' ? { nodes } : {}),
      detach: async () => undefined,
    };
    const result = await observeAccessibilityTree(fakePage(), async () => cdp as never);
    const encoded = JSON.stringify(result);
    expect(result.contract).toBe('bas-accessibility-tree-observation/v1');
    expect(result.url).toBe('https://allowed.example');
    expect(result.truncated).toBe(true);
    expect(result.node_count).toBeLessThanOrEqual(2_500);
    expect(Buffer.byteLength(encoded)).toBeLessThanOrEqual(ACCESSIBILITY_TREE_MAX_BYTES);
    expect(encoded).toContain('[REDACTED]');
    expect(encoded).not.toContain('should never leave the driver');
    expect(encoded).not.toContain('session=private');
    expect(encoded).not.toContain('backendDOMNodeId');
    expect(result.root.name).toBe('[REDACTED]');
    expect(result.layout).toEqual({ viewportWidthPx: 390, documentWidthPx: 420, viewportHeightPx: 800, documentHeightPx: 860 });
  });

  test('uses only the fixed read-only width metric and fails closed when it is unavailable', async () => {
    let metricSource = '';
    const page = {
      ...fakePage(),
      evaluate: async (operation: unknown) => {
        metricSource = String(operation);
        return undefined;
      },
    } as unknown as Page;
    const cdp = { send: async () => ({ nodes: [] }), detach: async () => undefined };
    const result = await observeAccessibilityTree(page, async () => cdp as never);
    expect(result.layout).toBeNull();
    expect(metricSource).toContain('window.innerWidth');
    expect(metricSource).toContain('document.documentElement.scrollWidth');
    expect(metricSource).toContain('window.innerHeight');
    expect(metricSource).toContain('document.documentElement.scrollHeight');
    expect(metricSource).not.toContain('eval');
    expect(metricSource).not.toContain('selector');
  });

  test('emits only session-opaque refs and keeps the selector target private', async () => {
    const page = {
      ...fakePage(),
      locator: jest.fn(() => ({ count: async () => 1 })),
    } as unknown as Page;
    const nodes = [
      { nodeId: 'root', role: { type: 'role', value: 'RootWebArea' }, childIds: ['button'] },
      { nodeId: 'button', backendDOMNodeId: 42, role: { type: 'role', value: 'button' }, name: { type: 'string', value: 'Continue' } },
    ];
    const cdp = {
      send: async (method: string) => method === 'Accessibility.getFullAXTree'
        ? { nodes }
        : { strings: ['button', 'data-testid', 'continue'], documents: [{ nodes: { backendNodeId: [42], nodeName: [0], attributes: [[1, 2]] } }] },
      detach: async () => undefined,
    };
    const result = await observeAccessibilityTree(page, async () => cdp as never);
    const encoded = JSON.stringify(result);
    const button = result.root.children[0];
    expect(button.ref).toMatch(/^e[0-9]+$/);
    expect(encoded).toContain(button.ref);
    expect(encoded).not.toContain('data-testid');
    expect(encoded).not.toContain('continue');
    expect(result.agentReferenceTargets?.get(button.ref!)).toMatchObject({ selector: 'button[data-testid="continue"]' });
    expect(Object.keys(result)).not.toContain('agentReferenceTargets');
  });
});
