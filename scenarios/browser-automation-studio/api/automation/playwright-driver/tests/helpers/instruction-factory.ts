import type { CompiledInstruction } from '../../src/types';

/**
 * Factory for creating test CompiledInstruction objects with proper defaults
 */
export function createTestInstruction(
  overrides: Partial<CompiledInstruction> = {}
): CompiledInstruction {
  return {
    index: 0,
    node_id: 'test-node',
    type: 'test',
    params: {},
    ...overrides,
  };
}
