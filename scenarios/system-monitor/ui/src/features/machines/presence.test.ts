import { describe, expect, it } from 'vitest';
import { describeMachine, formatPresenceAge, grantLevelFor, machinePresenceTone, sortMachinesForPicker } from './presence';
import type { Machine } from '../../types';

const machine = (overrides: Partial<Machine>): Machine => ({
  id: 'node-1',
  name: 'minimouse',
  os: 'darwin',
  arch: 'amd64',
  online: true,
  heartbeat_fresh: true,
  dispatchable: true,
  status: 'NODE_STATUS_ONLINE',
  ...overrides
});

describe('formatPresenceAge', () => {
  it('reads a week of silence as days, not hours', () => {
    // 639479s is a real swarminator heartbeat age. "177h 41m" is arithmetically
    // correct and useless: the reader is judging alarm, not measuring.
    expect(formatPresenceAge(639479)).toBe('7d');
  });

  it('keeps short ages precise', () => {
    expect(formatPresenceAge(8)).toBe('8s');
    expect(formatPresenceAge(125)).toBe('2m');
    expect(formatPresenceAge(7200)).toBe('2h');
  });
});

describe('grantLevelFor', () => {
  it('reduces concrete scopes to the level a person acts on', () => {
    expect(grantLevelFor(machine({ scopes: ['*:read'] }))).toBe('read');
    expect(grantLevelFor(machine({ scopes: ['*:read', '*:write'] }))).toBe('operate');
    expect(grantLevelFor(machine({ scopes: ['*:destructive'] }))).toBe('full');
  });

  it('does not imply read access for a node that holds no scopes', () => {
    expect(grantLevelFor(machine({ scopes: [] }))).toBe('none');
    expect(grantLevelFor(machine({}))).toBe('none');
  });

  it('treats the local machine as its own level', () => {
    expect(grantLevelFor(machine({ id: '', name: 'This machine' }))).toBe('local');
  });
});

describe('describeMachine', () => {
  it('states platform and age for a machine that is answering', () => {
    const presence = describeMachine(machine({ heartbeat_age_seconds: 8, scopes: ['*:read'] }));
    expect(presence.tone).toBe('live');
    expect(presence.meta).toBe('darwin · amd64 · 8s ago');
    expect(presence.platform).toBe('darwin/amd64');
    expect(presence.grantLabel).toBe('read only');
  });

  it('names the readiness fact that failed instead of a generic label', () => {
    const presence = describeMachine(machine({
      dispatchable: false,
      heartbeat_fresh: false,
      heartbeat_age_seconds: 639479,
      readiness: [
        { identity: 'registry_record', passed: true },
        { identity: 'heartbeat_fresh', passed: false },
        { identity: 'channel_held', passed: false }
      ]
    }));
    expect(presence.tone).toBe('unreachable');
    expect(presence.unreachableReason).toBe('not responding');
    expect(presence.meta).toBe('not responding · 7d ago');
  });

  it('distinguishes a node that beats but cannot be dispatched to', () => {
    const presence = describeMachine(machine({
      dispatchable: false,
      heartbeat_fresh: true,
      heartbeat_age_seconds: 3,
      readiness: [
        { identity: 'heartbeat_fresh', passed: true },
        { identity: 'channel_held', passed: false }
      ]
    }));
    // "not responding" would be a lie here: it is responding, and the channel
    // is the thing that is missing.
    expect(presence.unreachableReason).toBe('no open channel');
  });

  it('names the other ways a machine can fail readiness', () => {
    const failing = (identity: string) => describeMachine(machine({
      dispatchable: false,
      readiness: [{ identity, passed: false }]
    })).unreachableReason;
    expect(failing('channel_held')).toBe('no open channel');
    expect(failing('protocol_compatible')).toBe('incompatible agent version');
    expect(failing('registry_record')).toBe('not registered');
    // Undispatchable with every listed fact passing: say the honest generic
    // thing rather than inventing a cause.
    expect(describeMachine(machine({ dispatchable: false, readiness: [{ identity: 'heartbeat_fresh', passed: true }] })).unreachableReason)
      .toBe('not dispatchable');
    expect(describeMachine(machine({ dispatchable: false })).unreachableReason).toBe('not dispatchable');
  });

  it('marks a dispatchable machine with an aged heartbeat as stale, not unreachable', () => {
    const presence = describeMachine(machine({ heartbeat_fresh: false, heartbeat_age_seconds: 90 }));
    expect(presence.tone).toBe('stale');
  });

  it('says "unknown platform" rather than inventing one', () => {
    const presence = describeMachine(machine({ os: undefined, arch: undefined }));
    expect(presence.platform).toBe('unknown platform');
  });

  it('describes the local machine without a grant or an age', () => {
    const presence = describeMachine({ id: '', name: 'This machine', online: true, heartbeat_fresh: true, dispatchable: true, status: 'local' });
    expect(presence.tone).toBe('local');
    expect(presence.grantLabel).toBe('');
    expect(presence.meta).toBe('this computer');
  });

  it('never claims an age the API did not send', () => {
    const presence = describeMachine(machine({ heartbeat_age_seconds: undefined }));
    expect(presence.age).toBeUndefined();
    expect(presence.meta).toBe('darwin · amd64');
  });
});

describe('machinePresenceTone', () => {
  it('lets unreachable outrank a failing poll', () => {
    const presence = describeMachine(machine({ dispatchable: false }));
    // There is no "last reading" to freeze for a channel that never opened.
    expect(machinePresenceTone(presence, true)).toBe('unreachable');
  });

  it('narrows a live machine to stale while the poll is failing', () => {
    expect(machinePresenceTone(describeMachine(machine({})), true)).toBe('stale');
    expect(machinePresenceTone(describeMachine(machine({})), false)).toBe('live');
  });

  it('never calls this computer stale', () => {
    const local = describeMachine({ id: '', name: 'This machine', online: true, heartbeat_fresh: true, dispatchable: true, status: 'local' });
    // A failing poll against this computer is the API being down, which the
    // connection banner owns; it is not the machine going quiet.
    expect(machinePresenceTone(local, true)).toBe('local');
  });
});

describe('sortMachinesForPicker', () => {
  it('puts this computer first, then reachable machines, then the rest', () => {
    const ordered = sortMachinesForPicker([
      machine({ id: 'z', name: 'zeta' }),
      machine({ id: 'off', name: 'swarminator', dispatchable: false }),
      machine({ id: '', name: 'This machine' }),
      machine({ id: 'a', name: 'alpha' })
    ]);
    expect(ordered.map(entry => entry.name)).toEqual(['This machine', 'alpha', 'zeta', 'swarminator']);
  });
});
