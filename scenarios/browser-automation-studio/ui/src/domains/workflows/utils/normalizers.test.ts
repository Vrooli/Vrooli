import { describe, expect, it } from 'vitest';
import { workflowDefinitionToCanvas } from '@/stores/workflow/utils/codec';

describe('workflowDefinitionToCanvas', () => {
  it('derives the React Flow renderer key from the typed action', () => {
    const [node] = workflowDefinitionToCanvas({ nodes: [{
      id: 'navigate',
      action: { type: 'ACTION_TYPE_NAVIGATE', navigate: { url: 'https://example.com' } },
    }] }).nodes;

    expect(node.type).toBe('navigate');
    expect(node.action?.type).toBe('ACTION_TYPE_NAVIGATE');
  });

  it('does not synthesize an action from legacy node fields', () => {
    const [node] = workflowDefinitionToCanvas({ nodes: [{
      id: 'legacy',
      type: 'navigate',
      data: { url: 'https://example.com' },
    }] }).nodes;

    expect(node.type).toBe('navigate');
    expect(node.action).toBeUndefined();
  });
});
