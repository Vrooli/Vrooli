import { describe, expect, it } from 'vitest';
import { newGatewayRootOperationID } from './ai';

describe('newGatewayRootOperationID', () => {
  it('returns a fresh 128-bit hexadecimal id compatible with AI Gateway', () => {
    const first = newGatewayRootOperationID();
    const second = newGatewayRootOperationID();

    expect(first).toMatch(/^[a-f0-9]{32}$/);
    expect(second).toMatch(/^[a-f0-9]{32}$/);
    expect(second).not.toBe(first);
  });
});
