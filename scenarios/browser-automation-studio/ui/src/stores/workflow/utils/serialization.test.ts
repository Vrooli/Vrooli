import { describe, it, expect, vi } from 'vitest';
import type { Node, Edge } from 'reactflow';
import {
  PREVIEW_DATA_KEYS,
  TRANSIENT_NODE_KEYS,
  TRANSIENT_EDGE_KEYS,
  buildFlowDefinition,
  stripPreviewDataFromNodes,
  sanitizeNodesForPersistence,
  sanitizeEdgesForPersistence,
} from './serialization';

// Mock actionBuilder
vi.mock('../../../utils/actionBuilder', () => ({
  buildActionDefinition: vi.fn((type, data) => ({ type, ...data })),
}));

describe('serialization utilities', () => {
  describe('constants', () => {
    it('PREVIEW_DATA_KEYS contains expected keys', () => {
      expect(PREVIEW_DATA_KEYS).toContain('previewScreenshot');
      expect(PREVIEW_DATA_KEYS).toContain('previewScreenshotCapturedAt');
      expect(PREVIEW_DATA_KEYS).toContain('previewScreenshotSourceUrl');
    });

    it('TRANSIENT_NODE_KEYS contains reactflow internal keys', () => {
      expect(TRANSIENT_NODE_KEYS.has('selected')).toBe(true);
      expect(TRANSIENT_NODE_KEYS.has('dragging')).toBe(true);
      expect(TRANSIENT_NODE_KEYS.has('width')).toBe(true);
      expect(TRANSIENT_NODE_KEYS.has('height')).toBe(true);
      expect(TRANSIENT_NODE_KEYS.has('__rf')).toBe(true);
    });

    it('TRANSIENT_EDGE_KEYS contains reactflow internal keys', () => {
      expect(TRANSIENT_EDGE_KEYS.has('selected')).toBe(true);
      expect(TRANSIENT_EDGE_KEYS.has('sourceX')).toBe(true);
      expect(TRANSIENT_EDGE_KEYS.has('targetY')).toBe(true);
      expect(TRANSIENT_EDGE_KEYS.has('__rf')).toBe(true);
    });
  });

  describe('stripPreviewDataFromNodes', () => {
    it('returns empty array for null/undefined/empty input', () => {
      expect(stripPreviewDataFromNodes(null)).toEqual([]);
      expect(stripPreviewDataFromNodes(undefined)).toEqual([]);
      expect(stripPreviewDataFromNodes([])).toEqual([]);
    });

    it('removes preview data keys from node data', () => {
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'navigate',
          position: { x: 0, y: 0 },
          data: {
            url: 'https://example.com',
            previewScreenshot: 'base64data...',
            previewScreenshotCapturedAt: '2025-01-01T00:00:00Z',
            previewScreenshotSourceUrl: 'https://example.com',
          },
        },
      ];

      const result = stripPreviewDataFromNodes(nodes);

      expect(result[0].data).toEqual({ url: 'https://example.com' });
      expect(result[0].data).not.toHaveProperty('previewScreenshot');
      expect(result[0].data).not.toHaveProperty('previewScreenshotCapturedAt');
      expect(result[0].data).not.toHaveProperty('previewScreenshotSourceUrl');
    });

    it('preserves nodes without preview data', () => {
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'click',
          position: { x: 100, y: 100 },
          data: { selector: '#button', label: 'Click Button' },
        },
      ];

      const result = stripPreviewDataFromNodes(nodes);

      expect(result[0]).toBe(nodes[0]); // Same reference - no modification needed
    });

    it('handles nodes with missing or invalid data', () => {
      const nodes: Node[] = [
        { id: 'node-1', type: 'test', position: { x: 0, y: 0 }, data: null as any },
        { id: 'node-2', type: 'test', position: { x: 0, y: 0 }, data: undefined as any },
      ];

      const result = stripPreviewDataFromNodes(nodes);

      expect(result).toHaveLength(2);
    });
  });

  describe('sanitizeNodesForPersistence', () => {
    it('returns empty array for null/undefined/empty input', () => {
      expect(sanitizeNodesForPersistence(null)).toEqual([]);
      expect(sanitizeNodesForPersistence(undefined)).toEqual([]);
      expect(sanitizeNodesForPersistence([])).toEqual([]);
    });

    it('removes transient reactflow properties', () => {
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'navigate',
          position: { x: 100, y: 200 },
          data: { url: 'https://example.com' },
          selected: true,
          dragging: false,
          width: 200,
          height: 100,
        } as Node,
      ];

      const result = sanitizeNodesForPersistence(nodes);

      expect(result[0]).not.toHaveProperty('selected');
      expect(result[0]).not.toHaveProperty('dragging');
      expect(result[0]).not.toHaveProperty('width');
      expect(result[0]).not.toHaveProperty('height');
    });

    it('preserves essential node properties', () => {
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'navigate',
          position: { x: 100, y: 200 },
          data: { url: 'https://example.com' },
        },
      ];

      const result = sanitizeNodesForPersistence(nodes);

      expect(result[0]).toHaveProperty('id', 'node-1');
      expect(result[0]).toHaveProperty('type', 'navigate');
      expect(result[0]).toHaveProperty('position', { x: 100, y: 200 });
      expect(result[0]).toHaveProperty('data', { url: 'https://example.com' });
    });

    it('deep clones data to prevent mutations', () => {
      const originalData = { nested: { value: 'test' } };
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'test',
          position: { x: 0, y: 0 },
          data: originalData,
        },
      ];

      const result = sanitizeNodesForPersistence(nodes);

      expect(result[0]).toHaveProperty('data');
      expect((result[0] as any).data).not.toBe(originalData);
      expect((result[0] as any).data).toEqual(originalData);
    });

    it('normalizes position values', () => {
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'test',
          position: { x: '100' as any, y: undefined as any },
          data: {},
        },
      ];

      const result = sanitizeNodesForPersistence(nodes);

      expect((result[0] as any).position).toEqual({ x: 100, y: 0 });
    });

    it('adds action field for V2 compatibility', () => {
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'navigate',
          position: { x: 0, y: 0 },
          data: { url: 'https://example.com' },
        },
      ];

      const result = sanitizeNodesForPersistence(nodes);

      expect(result[0]).toHaveProperty('action');
    });

    it('removes function properties', () => {
      const nodes: Node[] = [
        {
          id: 'node-1',
          type: 'test',
          position: { x: 0, y: 0 },
          data: {},
          onNodeClick: () => {},
        } as any,
      ];

      const result = sanitizeNodesForPersistence(nodes);

      expect(result[0]).not.toHaveProperty('onNodeClick');
    });
  });

  describe('sanitizeEdgesForPersistence', () => {
    it('returns empty array for null/undefined/empty input', () => {
      expect(sanitizeEdgesForPersistence(null)).toEqual([]);
      expect(sanitizeEdgesForPersistence(undefined)).toEqual([]);
      expect(sanitizeEdgesForPersistence([])).toEqual([]);
    });

    it('removes transient reactflow properties', () => {
      const edges: Edge[] = [
        {
          id: 'edge-1',
          source: 'node-1',
          target: 'node-2',
          selected: true,
          sourceX: 100,
          sourceY: 200,
          targetX: 300,
          targetY: 400,
        } as Edge,
      ];

      const result = sanitizeEdgesForPersistence(edges);

      expect(result[0]).not.toHaveProperty('selected');
      expect(result[0]).not.toHaveProperty('sourceX');
      expect(result[0]).not.toHaveProperty('sourceY');
      expect(result[0]).not.toHaveProperty('targetX');
      expect(result[0]).not.toHaveProperty('targetY');
    });

    it('preserves essential edge properties', () => {
      const edges: Edge[] = [
        {
          id: 'edge-1',
          source: 'node-1',
          target: 'node-2',
          type: 'smoothstep',
          label: 'Connection',
        },
      ];

      const result = sanitizeEdgesForPersistence(edges);

      expect(result[0]).toHaveProperty('id', 'edge-1');
      expect(result[0]).toHaveProperty('source', 'node-1');
      expect(result[0]).toHaveProperty('target', 'node-2');
      expect(result[0]).toHaveProperty('type', 'smoothstep');
      expect(result[0]).toHaveProperty('label', 'Connection');
    });

    it('deep clones complex properties', () => {
      const edges: Edge[] = [
        {
          id: 'edge-1',
          source: 'node-1',
          target: 'node-2',
          data: { condition: 'success' },
          style: { stroke: '#ff0000' },
        },
      ];

      const result = sanitizeEdgesForPersistence(edges);

      expect((result[0] as any).data).toEqual({ condition: 'success' });
      expect((result[0] as any).style).toEqual({ stroke: '#ff0000' });
    });
  });

  describe('buildFlowDefinition', () => {
    it('creates flow definition with nodes and edges', () => {
      const nodes = [{ id: 'node-1', type: 'test', position: { x: 0, y: 0 } }];
      const edges = [{ id: 'edge-1', source: 'node-1', target: 'node-2' }];

      const result = buildFlowDefinition({}, nodes, edges);

      expect(result.nodes).toEqual(nodes);
      expect(result.edges).toEqual(edges);
    });

    it('preserves non-structural properties from base definition', () => {
      const base = {
        version: '2.0',
        metadata: { name: 'Test' },
        customField: 'value',
      };

      const result = buildFlowDefinition(base, [], []);

      expect(result.version).toBe('2.0');
      expect(result.metadata).toEqual({ name: 'Test' });
      expect(result.customField).toBe('value');
    });

    it('strips nodes, edges, and settings from base', () => {
      const base = {
        nodes: [{ id: 'old' }],
        edges: [{ id: 'old' }],
        settings: { old: 'setting' },
        otherField: 'kept',
      };

      const result = buildFlowDefinition(base, [{ id: 'new' }], []);

      expect(result.nodes).toEqual([{ id: 'new' }]);
      expect(result.edges).toEqual([]);
      expect(result.otherField).toBe('kept');
    });

    it('adds viewport to settings when provided', () => {
      const viewport = { width: 1920, height: 1080, preset: 'desktop' as const };

      const result = buildFlowDefinition({}, [], [], viewport);

      expect(result.settings).toEqual({ executionViewport: viewport });
      expect(result.settings_typed).toEqual({ viewport_width: 1920, viewport_height: 1080 });
    });

    it('removes executionViewport from settings when not provided', () => {
      const base = {
        settings: { executionViewport: { width: 800, height: 600 }, otherSetting: true },
      };

      const result = buildFlowDefinition(base, [], []);

      expect((result.settings as any).executionViewport).toBeUndefined();
      expect((result.settings as any).otherSetting).toBe(true);
    });

    it('derives metadata_typed from metadata', () => {
      const base = {
        metadata: {
          name: 'Test Workflow',
          description: 'A test',
          labels: { env: 'test' },
          version: '1.0',
        },
      };

      const result = buildFlowDefinition(base, [], []);

      expect(result.metadata_typed).toEqual({
        name: 'Test Workflow',
        description: 'A test',
        labels: { env: 'test' },
        version: '1.0',
      });
    });

    it('handles null/undefined input gracefully', () => {
      const result = buildFlowDefinition(null, [], []);

      expect(result.nodes).toEqual([]);
      expect(result.edges).toEqual([]);
    });
  });
});
