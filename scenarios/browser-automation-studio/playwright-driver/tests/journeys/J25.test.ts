import type { SessionSpec } from '../../src/types';
import { action, closeLease, driver, env, fixtureAfter, fixtureState, openLease, registerLeaseCleanup, runAction, withJourneyResources, type Lease } from './support';

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
  it('reports fixture-owned signals, preserves profile isolation and leaves real sign-in to an authorized operator', () => withJourneyResources(async (resources) => {
      const open = async (...args: Parameters<typeof openLease>) => {
        const lease = await openLease(...args);
        registerLeaseCleanup(resources, lease);
        return lease;
      };
      const interactive = await open({ storageState: identity('test-profile-alpha'), labels: { mode: 'recording', journey: 'J25' } });
      const first = await observeFingerprint(interactive);
      expect(first.webdriver).toBe(false);
      const saved = (await driver<{ storage_state: StorageState }>(`/session/${interactive.sessionId}/storage-state`)).storage_state;
      await closeLease(interactive);

      const restored = await open({ storageState: saved, labels: { mode: 'recording', journey: 'J25-restored' } });
      await runAction(restored, action('navigate', { url: `${env.fixture}/` }));
      const restoredState = (await driver<{ storage_state: StorageState }>(`/session/${restored.sessionId}/storage-state`)).storage_state;
      expect(restoredState.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j25_identity', value: 'test-profile-alpha' }));
      await closeLease(restored);

      const isolated = await open({ storageState: identity('test-profile-beta'), labels: { mode: 'recording', journey: 'J25-isolated' } });
      await runAction(isolated, action('navigate', { url: `${env.fixture}/` }));
      const isolatedState = (await driver<{ storage_state: StorageState }>(`/session/${isolated.sessionId}/storage-state`)).storage_state;
      expect(isolatedState.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j25_identity', value: 'test-profile-beta' }));
      expect(isolatedState.cookies).not.toContainEqual(expect.objectContaining({ value: 'test-profile-alpha' }));
      await closeLease(isolated);

      const execution = await open({ labels: { mode: 'execution', journey: 'J25-execution' } });
      const replaySignals = await observeFingerprint(execution);
      expect(typeof replaySignals.webdriver).toBe('boolean');
  }));
});
