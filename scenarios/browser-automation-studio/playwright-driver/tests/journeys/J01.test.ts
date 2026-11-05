import type { SessionSpec } from '../../src/types';
import { action, closeLease, driver, env, runAction, withLeases, type Lease } from './support';

type StorageState = NonNullable<SessionSpec['storage_state']>;
const identityCookie = (value: string) => ({
  name: 'bas_j01_identity', value, domain: '127.0.0.1', path: '/', expires: -1, httpOnly: false, secure: false, sameSite: 'Lax' as const,
});
const signedIn = (value: string): StorageState => ({ cookies: [identityCookie(value)], origins: [] });

async function visitForms(lease: Lease): Promise<StorageState> {
  const outcome = await runAction(lease, action('navigate', { url: `${env.fixture}/forms` }));
  expect(outcome.finalUrl).toBe(`${env.fixture}/forms`);
  return (await driver<{ storage_state: StorageState }>(`/session/${lease.sessionId}/storage-state`)).storage_state;
}

describe('[REQ:BAS-RH-J01] local profile close and reopen', () => {
  it('given a signed-in profile, when it closes and reopens, then only its own saved state returns', () => withLeases(async (open) => {
    const first = await open({ storageState: signedIn('alpha') });
    const saved = await visitForms(first);
    await closeLease(first);
    expect(saved.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j01_identity', value: 'alpha' }));

    const restored = await visitForms(await open({ storageState: saved }));
    expect(restored.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j01_identity', value: 'alpha' }));

    const other = await visitForms(await open({ storageState: signedIn('beta') }));
    expect(other.cookies).toContainEqual(expect.objectContaining({ name: 'bas_j01_identity', value: 'beta' }));
    expect(other.cookies).not.toContainEqual(expect.objectContaining({ value: 'alpha' }));
  }));
});
