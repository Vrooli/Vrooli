import { describe, it, expect } from 'vitest';
import type { Node } from 'reactflow';
import { collectWorkflowVariables } from './useWorkflowVariables';

describe('collectWorkflowVariables', () => {
  it('returns empty array when no nodes exist', () => {
    expect(collectWorkflowVariables(undefined)).toEqual([]);
    expect(collectWorkflowVariables([], 'node-1')).toEqual([]);
  });

  it('collects, trims, dedupes, and sorts variable names with metadata', () => {
    const nodes: Partial<Node>[] = [
      {
        id: 'a',
        type: 'setVariable',
        data: { name: ' token ', storeAs: 'tokenAlias ' },
      },
      {
        id: 'b',
        type: 'useVariable',
        data: { storeAs: 'copy' },
      },
      {
        id: 'c',
        type: 'evaluate',
        data: { storeResult: 'pageTitle' },
      },
      {
        id: 'd',
        type: 'extract',
        data: { storeIn: 'pageTitle' },
      },
    ];

    const variables = collectWorkflowVariables(nodes as Node[]);
    expect(variables.map((entry) => entry.name)).toEqual(['copy', 'pageTitle', 'token', 'tokenAlias']);
    const tokenMeta = variables.find((entry) => entry.name === 'token');
    expect(tokenMeta?.sourceNodeId).toBe('a');
    expect(tokenMeta?.sourceType).toBe('setVariable');
    expect(tokenMeta?.sourceLabel).toContain('Set Variable');
  });

  it('omits variables contributed by the excluded node', () => {
    const nodes: Partial<Node>[] = [
      { id: 'keep', type: 'evaluate', data: { storeResult: 'kept' } },
      { id: 'skip', type: 'setVariable', data: { name: 'secret' } },
    ];

    const variables = collectWorkflowVariables(nodes as Node[], 'skip');
    expect(variables.map((entry) => entry.name)).toEqual(['kept']);
  });
});
