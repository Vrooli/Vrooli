import { describe, it, expect, vi } from 'vitest';
import type { Node, Edge } from 'reactflow';
import { stableSerialize, computeWorkflowFingerprint } from './fingerprint';
import type { Workflow } from '../types';

// Mock dependencies
vi.mock('./serialization', () => ({
  sanitizeNodesForPersistence: vi.fn((nodes) => nodes?.map((n: Node) => ({ id: n.id, type: n.type })) ?? []),
  sanitizeEdgesForPersistence: vi.fn((edges) => edges?.map((e: Edge) => ({ id: e.id, source: e.source, target: e.target })) ?? []),
}));

vi.mock('./viewport', () => ({
  sanitizeViewportSettings: vi.fn((v) => v),
}));

describe('fingerprint utilities', () => {
  describe('stableSerialize', () => {
    it('serializes primitives', () => {
      expect(stableSerialize('string')).toBe('"string"');
      expect(stableSerialize(123)).toBe('123');
      expect(stableSerialize(true)).toBe('true');
      expect(stableSerialize(null)).toBe('null');
    });

    it('serializes arrays', () => {
      expect(stableSerialize([1, 2, 3])).toBe('[1,2,3]');
      expect(stableSerialize(['a', 'b'])).toBe('["a","b"]');
    });

    it('serializes objects with sorted keys', () => {
      const obj = { z: 1, a: 2, m: 3 };
      const result = stableSerialize(obj);
      // Keys should be sorted alphabetically
      expect(result).toBe('{"a":2,"m":3,"z":1}');
    });

    it('produces identical output regardless of key order', () => {
      const obj1 = { a: 1, b: 2, c: 3 };
      const obj2 = { c: 3, a: 1, b: 2 };
      const obj3 = { b: 2, c: 3, a: 1 };

      const result1 = stableSerialize(obj1);
      const result2 = stableSerialize(obj2);
      const result3 = stableSerialize(obj3);

      expect(result1).toBe(result2);
      expect(result2).toBe(result3);
    });

    it('handles nested objects with sorted keys', () => {
      const obj = {
        z: { b: 2, a: 1 },
        a: { y: 3, x: 4 },
      };
      const result = stableSerialize(obj);
      expect(result).toBe('{"a":{"x":4,"y":3},"z":{"a":1,"b":2}}');
    });

    it('handles arrays of objects', () => {
      const arr = [
        { z: 1, a: 2 },
        { b: 3, a: 4 },
      ];
      const result = stableSerialize(arr);
      expect(result).toBe('[{"a":2,"z":1},{"a":4,"b":3}]');
    });

    it('filters out undefined values', () => {
      const obj = { a: 1, b: undefined, c: 3 };
      const result = stableSerialize(obj);
      expect(result).toBe('{"a":1,"c":3}');
    });

    it('filters out function values', () => {
      const obj = { a: 1, b: () => {}, c: 3 };
      const result = stableSerialize(obj);
      expect(result).toBe('{"a":1,"c":3}');
    });

    it('handles circular references gracefully', () => {
      const obj: any = { a: 1 };
      obj.self = obj;

      // Should not throw
      expect(() => stableSerialize(obj)).not.toThrow();

      // Circular reference should be replaced with null
      const result = stableSerialize(obj);
      expect(result).toBe('{"a":1,"self":null}');
    });

    it('handles deeply nested structures', () => {
      const obj = {
        level1: {
          level2: {
            level3: {
              value: 'deep',
            },
          },
        },
      };
      const result = stableSerialize(obj);
      expect(result).toBe('{"level1":{"level2":{"level3":{"value":"deep"}}}}');
    });
  });

  describe('computeWorkflowFingerprint', () => {
    const createWorkflow = (overrides: Partial<Workflow> = {}): Workflow => ({
      id: 'workflow-1',
      name: 'Test Workflow',
      description: 'A test workflow',
      folderPath: '/workflows',
      nodes: [],
      edges: [],
      tags: [],
      version: 1,
      createdAt: new Date('2025-01-01'),
      updatedAt: new Date('2025-01-01'),
      ...overrides,
    });

    it('returns empty string for null workflow', () => {
      expect(computeWorkflowFingerprint(null, [], [])).toBe('');
    });

    it('generates fingerprint for workflow', () => {
      const workflow = createWorkflow();
      const fingerprint = computeWorkflowFingerprint(workflow, [], []);

      expect(fingerprint).toBeTruthy();
      expect(typeof fingerprint).toBe('string');
    });

    it('generates same fingerprint for identical workflows', () => {
      const workflow1 = createWorkflow({ name: 'Same' });
      const workflow2 = createWorkflow({ name: 'Same' });

      const fp1 = computeWorkflowFingerprint(workflow1, [], []);
      const fp2 = computeWorkflowFingerprint(workflow2, [], []);

      expect(fp1).toBe(fp2);
    });

    it('generates different fingerprints for different names', () => {
      const workflow1 = createWorkflow({ name: 'Workflow A' });
      const workflow2 = createWorkflow({ name: 'Workflow B' });

      const fp1 = computeWorkflowFingerprint(workflow1, [], []);
      const fp2 = computeWorkflowFingerprint(workflow2, [], []);

      expect(fp1).not.toBe(fp2);
    });

    it('generates different fingerprints for different descriptions', () => {
      const workflow1 = createWorkflow({ description: 'Description A' });
      const workflow2 = createWorkflow({ description: 'Description B' });

      const fp1 = computeWorkflowFingerprint(workflow1, [], []);
      const fp2 = computeWorkflowFingerprint(workflow2, [], []);

      expect(fp1).not.toBe(fp2);
    });

    it('generates different fingerprints for different nodes', () => {
      const workflow = createWorkflow();
      const nodes1: Node[] = [{ id: 'node-1', type: 'navigate', position: { x: 0, y: 0 }, data: {} }];
      const nodes2: Node[] = [{ id: 'node-2', type: 'click', position: { x: 0, y: 0 }, data: {} }];

      const fp1 = computeWorkflowFingerprint(workflow, nodes1, []);
      const fp2 = computeWorkflowFingerprint(workflow, nodes2, []);

      expect(fp1).not.toBe(fp2);
    });

    it('generates different fingerprints for different edges', () => {
      const workflow = createWorkflow();
      const edges1: Edge[] = [{ id: 'edge-1', source: 'a', target: 'b' }];
      const edges2: Edge[] = [{ id: 'edge-2', source: 'b', target: 'c' }];

      const fp1 = computeWorkflowFingerprint(workflow, [], edges1);
      const fp2 = computeWorkflowFingerprint(workflow, [], edges2);

      expect(fp1).not.toBe(fp2);
    });

    it('generates same fingerprint regardless of tag order', () => {
      const workflow1 = createWorkflow({ tags: ['alpha', 'beta', 'gamma'] });
      const workflow2 = createWorkflow({ tags: ['gamma', 'alpha', 'beta'] });

      const fp1 = computeWorkflowFingerprint(workflow1, [], []);
      const fp2 = computeWorkflowFingerprint(workflow2, [], []);

      expect(fp1).toBe(fp2);
    });

    it('includes executionViewport in fingerprint', () => {
      const workflow1 = createWorkflow({ executionViewport: { width: 1920, height: 1080 } });
      const workflow2 = createWorkflow({ executionViewport: { width: 800, height: 600 } });

      const fp1 = computeWorkflowFingerprint(workflow1, [], []);
      const fp2 = computeWorkflowFingerprint(workflow2, [], []);

      expect(fp1).not.toBe(fp2);
    });

    it('includes flowDefinition in fingerprint', () => {
      const workflow1 = createWorkflow({ flowDefinition: { version: '1.0' } });
      const workflow2 = createWorkflow({ flowDefinition: { version: '2.0' } });

      const fp1 = computeWorkflowFingerprint(workflow1, [], []);
      const fp2 = computeWorkflowFingerprint(workflow2, [], []);

      expect(fp1).not.toBe(fp2);
    });

    it('handles missing optional fields gracefully', () => {
      const workflow = createWorkflow({
        name: '',
        description: undefined,
        folderPath: undefined as any,
        tags: undefined,
        executionViewport: undefined,
        flowDefinition: undefined,
      });

      expect(() => computeWorkflowFingerprint(workflow, [], [])).not.toThrow();
    });
  });
});
