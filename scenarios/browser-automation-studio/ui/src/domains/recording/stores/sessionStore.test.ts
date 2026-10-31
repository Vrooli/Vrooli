import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useSessionStore } from './sessionStore';

vi.mock('@/config', () => ({ getApiBase: () => 'http://api.test' }));
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}
const response = (ok: boolean) => ({ ok, status: ok ? 200 : 404 } as Response);

describe('recording validation belongs to its current session attempt', () => {
  beforeEach(() => useSessionStore.getState().clearSession());
  afterEach(() => { vi.unstubAllGlobals(); useSessionStore.getState().clearSession(); });
  it.each([true, false])('keeps validated B when older A later completes with valid=%s', async valid => {
    const a = deferred<Response>();
    vi.stubGlobal('fetch', vi.fn().mockReturnValueOnce(a.promise).mockResolvedValueOnce(response(true)));
    const pending = useSessionStore.getState().validateSession('a');
    expect(await useSessionStore.getState().validateSession('b')).toBe(true);
    expect(useSessionStore.getState().sessionId).toBe('b');
    a.resolve(response(valid)); await pending;
    expect(useSessionStore.getState().sessionId).toBe('b');
    expect(useSessionStore.getState().isValidated).toBe(true);
    expect(useSessionStore.getState().isValidating).toBe(false);
  });
  it('does not resurrect validation after clearSession', async () => {
    const pending = deferred<Response>(); vi.stubGlobal('fetch', vi.fn().mockReturnValue(pending.promise));
    const validation = useSessionStore.getState().validateSession('a');
    useSessionStore.getState().clearSession();
    pending.resolve(response(true)); await validation;
    expect(useSessionStore.getState().sessionId).toBeNull();
    expect(useSessionStore.getState().isValidated).toBe(false);
  });
  it('does not overwrite an explicitly adopted replacement session', async () => {
    const pending = deferred<Response>(); vi.stubGlobal('fetch', vi.fn().mockReturnValue(pending.promise));
    const validation = useSessionStore.getState().validateSession('a');
    useSessionStore.getState().setSession({ sessionId: 'b', profileId: 'profile-b' });
    pending.resolve(response(true)); await validation;
    expect(useSessionStore.getState().sessionId).toBe('b');
    expect(useSessionStore.getState().profileId).toBe('profile-b');
  });
  it('lets a creation retire only its own shared loading flag', () => {
    const a = {}; const b = {};
    useSessionStore.getState().setIsCreating(true, a);
    useSessionStore.getState().setIsCreating(true, b);
    useSessionStore.getState().setIsCreating(false, a);
    expect(useSessionStore.getState().isCreating).toBe(true);
    useSessionStore.getState().setIsCreating(false, b);
    expect(useSessionStore.getState().isCreating).toBe(false);
  });
  it('clears the retired validation indicator immediately when a new URL identity is selected', async () => {
    const pending = deferred<Response>(); vi.stubGlobal('fetch', vi.fn().mockReturnValue(pending.promise));
    const validation = useSessionStore.getState().validateSession('a');
    expect(useSessionStore.getState().isValidating).toBe(true);
    useSessionStore.getState().setSessionId('b');
    expect(useSessionStore.getState().isValidating).toBe(false);
    pending.resolve(response(true)); await validation;
    expect(useSessionStore.getState().sessionId).toBe('b');
    expect(useSessionStore.getState().isValidated).toBe(false);
  });
  it('clears pending B validation when returning to already-validated A', async () => {
    useSessionStore.getState().setSession({ sessionId: 'a' });
    const pending = deferred<Response>(); vi.stubGlobal('fetch', vi.fn().mockReturnValue(pending.promise));
    const validationB = useSessionStore.getState().validateSession('b');
    expect(useSessionStore.getState().isValidating).toBe(true);
    expect(await useSessionStore.getState().validateSession('a')).toBe(true);
    expect(useSessionStore.getState().isValidating).toBe(false);
    pending.resolve(response(true)); expect(await validationB).toBe(false);
    expect(useSessionStore.getState().sessionId).toBe('a');
    expect(useSessionStore.getState().isValidated).toBe(true);
  });
});
