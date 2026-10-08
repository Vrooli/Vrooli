import type { SessionSpec } from '../../src/types';
import { action, closeLease, driver, env, fixtureAfter, fixtureState, openLease, runAction, type Lease } from './support';

type StorageState = NonNullable<SessionSpec['storage_state']>;
const identity = (value: string): StorageState => ({
  cookies: [{ name: 'bas_j25_identity', value, domain: '127.0.0.1', path: '/', expires: -1, httpOnly: false, secure: false, sameSite: 'Lax' }],
  origins: [],
});

async function observeFingerprint(lease: Lease) {
  const before = (await fixtureState()).fingerprints?.length ?? 0;
  await runAction(lease, action('navigate', { url: `${env.fixture}/fingerprint` }));
  const state = await fixtureAfter('fingerprint', before);
  const observation = state.fingerprints?.at(-1);
  expect(observation).toBeDefined();
  expect(typeof observation?.webdriver).toBe('boolean');
  expect(typeof observation?.language).toBe('string');
  expect(Array.isArray(observation?.languages)).toBe(true);
  expect(typeof observation?.timezone).toBe('string');
  expect(Number.isInteger(observation?.hardwareConcurrency)).toBe(true);
  return observation!;
}

describe('[REQ:BAS-RH-J25] local detectability and persistent sign-in continuity', () => {
  it('reports fixture-owned signals, preserves profile isolation and leaves real sign-in to an authorized operator', async () => {
    const leases: Lease[] = [];
    try {
      const interactive = await openLease({ storageState: identity('test-profile-alpha'), labels: { mode: 'recording', journey: 'J25' } });
      leases.push(interactive);
      const first = await observeFingerprint(interactive);
      expect(first.webdriver).toBe(false);
      const saved = (await driver<{ storage_state: StorageState }>(`/session/${interactive.sessionId}/storage-state`)).storage_state;
      await closeLease(interactive);

      const restored = await openLease({ storageState: saved, labels: { mode: 'recording', journey: 'J25-restored' } });
      leases.push(restored);
      await runAction(restored, action('navigate', { url: `${env.fixture}/` }));
      const restoredState = (await driver<{ storage_state: StorageState }>(`/session/${restored.sessionId}/storage-state`)).storage_state;
      expect(restoredState.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j25_identity', value: 'test-profile-alpha' }));
      await closeLease(restored);

      const isolated = await openLease({ storageState: identity('test-profile-beta'), labels: { mode: 'recording', journey: 'J25-isolated' } });
      leases.push(isolated);
      await runAction(isolated, action('navigate', { url: `${env.fixture}/` }));
      const isolatedState = (await driver<{ storage_state: StorageState }>(`/session/${isolated.sessionId}/storage-state`)).storage_state;
      expect(isolatedState.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j25_identity', value: 'test-profile-beta' }));
      expect(isolatedState.cookies).not.toContainEqual(expect.objectContaining({ value: 'test-profile-alpha' }));
      await closeLease(isolated);

      const execution = await openLease({ labels: { mode: 'execution', journey: 'J25-execution' } });
      leases.push(execution);
      const replaySignals = await observeFingerprint(execution);
      expect(typeof replaySignals.webdriver).toBe('boolean');
    } finally {
      for (const lease of leases.reverse()) await closeLease(lease).catch(() => undefined);
    }
  });
});
