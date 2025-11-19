import type { Edge, Node } from 'reactflow';
import { describe, expect, it } from 'vitest';
import { autoLayoutNodes } from './workflowNormalizers';

describe('autoLayoutNodes', () => {
    const createNode = (id: string, x = 0, y = 0): Node => ({
        id,
        type: 'default',
        position: { x, y },
        data: {},
    });

    const createEdge = (source: string, target: string): Edge => ({
        id: `e-${source}-${target}`,
        source,
        target,
    });

    it('returns empty array for empty input', () => {
        expect(autoLayoutNodes([], [])).toEqual([]);
    });

    it('preserves single node position if not default/zero', () => {
        const nodes = [createNode('1', 500, 500)];
        const result = autoLayoutNodes(nodes, []);
        expect(result[0].position).toEqual({ x: 500, y: 500 });
    });

    it('layouts single node at (0, 100) if position is (0,0)', () => {
        const nodes = [createNode('1', 0, 0)];
        const result = autoLayoutNodes(nodes, []);
        expect(result[0].position).toEqual({ x: 0, y: 100 });
    });

    it('layouts linear chain A -> B -> C', () => {
        const nodes = [
            createNode('A', 0, 0),
            createNode('B', 0, 0),
            createNode('C', 0, 0),
        ];
        const edges = [
            createEdge('A', 'B'),
            createEdge('B', 'C'),
        ];

        const result = autoLayoutNodes(nodes, edges);
        const posA = result.find(n => n.id === 'A')!.position;
        const posB = result.find(n => n.id === 'B')!.position;
        const posC = result.find(n => n.id === 'C')!.position;

        // Check x levels
        expect(posA.x).toBe(0);
        expect(posB.x).toBe(300);
        expect(posC.x).toBe(600);

        // Check y positions (should be same row as they are single in level)
        expect(posA.y).toBe(100);
        expect(posB.y).toBe(100);
        expect(posC.y).toBe(100);
    });

    it('layouts branching A -> B, A -> C', () => {
        const nodes = [
            createNode('A', 0, 0),
            createNode('B', 0, 0),
            createNode('C', 0, 0),
        ];
        const edges = [
            createEdge('A', 'B'),
            createEdge('A', 'C'),
        ];

        const result = autoLayoutNodes(nodes, edges);
        const posA = result.find(n => n.id === 'A')!.position;
        const posB = result.find(n => n.id === 'B')!.position;
        const posC = result.find(n => n.id === 'C')!.position;

        expect(posA.x).toBe(0);
        expect(posB.x).toBe(300);
        expect(posC.x).toBe(300);

        // B and C should have different Y positions
        expect(posB.y).not.toBe(posC.y);
    });

    it('handles cycles A -> B -> A gracefully', () => {
        const nodes = [
            createNode('A', 0, 0),
            createNode('B', 0, 0),
        ];
        const edges = [
            createEdge('A', 'B'),
            createEdge('B', 'A'),
        ];

        const result = autoLayoutNodes(nodes, edges);
        // Should not crash and assign some positions
        expect(result).toHaveLength(2);
        expect(result[0].position.x).not.toBe(result[1].position.x); // Should be different levels
    });

    it('layouts disconnected components', () => {
        const nodes = [
            createNode('A', 0, 0),
            createNode('B', 0, 0),
        ];
        const edges: Edge[] = [];

        const result = autoLayoutNodes(nodes, edges);
        const posA = result.find(n => n.id === 'A')!.position;
        const posB = result.find(n => n.id === 'B')!.position;

        expect(posA.x).toBe(0);
        expect(posB.x).toBe(0);
        expect(posA.y).not.toBe(posB.y);
    });

    it('detects default diagonal layout and re-layouts', () => {
        // Simulate normalizeNodes default output
        const nodes = [
            createNode('A', 100, 100),
            createNode('B', 300, 220), // 100 + 1*200, 100 + 1*120
        ];
        const edges = [createEdge('A', 'B')];

        const result = autoLayoutNodes(nodes, edges);
        const posA = result.find(n => n.id === 'A')!.position;
        const posB = result.find(n => n.id === 'B')!.position;

        // Should be laid out horizontally (0, 300) instead of diagonal
        expect(posA.x).toBe(0);
        expect(posB.x).toBe(300);
        expect(posA.y).toBe(100);
        expect(posB.y).toBe(100);
    });
});
