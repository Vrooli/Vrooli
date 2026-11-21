import { describe, expect, it } from 'vitest';
import { parseMachineList } from './api';

describe('parseMachineList', () => {
  it('preserves machine capabilities and optional readiness details', () => {
    expect(parseMachineList([{
      id: 'node-a',
      name: 'Node A',
      os: 'linux',
      arch: 'amd64',
      online: true,
      heartbeat_fresh: true,
      heartbeat_age_seconds: 2,
      dispatchable: true,
      status: 'ready',
      grant: 'read',
      scopes: ['system:read'],
      readiness: [{ identity: 'api', passed: true, detail: 'healthy', recovery_action: 'retry' }],
    }])).toEqual([{
      id: 'node-a', name: 'Node A', os: 'linux', arch: 'amd64', online: true,
      heartbeat_fresh: true, heartbeat_age_seconds: 2, dispatchable: true,
      status: 'ready', grant: 'read', scopes: ['system:read'],
      readiness: [{ identity: 'api', passed: true, detail: 'healthy', recovery_action: 'retry' }],
    }]);
  });

  it('rejects malformed machine inventory instead of exposing unvalidated fields', () => {
    expect(() => parseMachineList({ machines: [] })).toThrow('machines response must be an array');
    expect(() => parseMachineList([{ id: 'node-a' }])).toThrow('machine.name must be a string');
  });
});
