import { describe, expect, it } from 'vitest';
import { normalizeNodes } from './normalizers';

describe('normalizeNodes', () => {
  it('derives the React Flow renderer key from the typed action', () => {
    const [node] = normalizeNodes([{
      id: 'navigate',
      action: { type: 'ACTION_TYPE_NAVIGATE', navigate: { url: 'https://example.com' } },
    }]);

    expect(node.type).toBe('navigate');
    expect(node.action?.type).toBe('ACTION_TYPE_NAVIGATE');
  });

  it('does not synthesize an action from legacy node fields', () => {
    const [node] = normalizeNodes([{
      id: 'legacy',
      type: 'navigate',
      data: { url: 'https://example.com' },
    }]);

    expect(node.type).toBe('navigate');
    expect(node.action).toBeUndefined();
  });
});
