import { action, closeLease, env, openLease, registerLeaseCleanup, releaseLease, runAction, fixtureAfter, fixtureState, withJourneyResources } from './support';

describe('[REQ:BAS-RH-J19] released session configuration and ownership', () => {
  it('given a released session, when compatible and incompatible owners arrive, then identity, configuration and lease ownership stay aligned', () => withJourneyResources(async (resources) => {
    const open = async (...args: Parameters<typeof openLease>) => {
      const lease = await openLease(...args);
      registerLeaseCleanup(resources, lease);
      return lease;
    };
    const labels = { journey: 'J19-profile-pool' };
    const base = { labels, sessionProfileVersion: 'profile-A:1', locale: 'en-US' };
    const before = await fixtureState();
      const first = await open(base);
      await runAction(first, action('navigate', { url: `${env.fixture}/` }));
      await runAction(first, action('click', { selector: '#counter' }));
      const initial = await fixtureAfter('effect', before.effects.length);
      const signedInIdentity = initial.effects.at(-1)?.context;
      expect(signedInIdentity).toContain('fixture_context=');

      await releaseLease(first);
      const continuation = await open({ ...base, reuseMode: 'reuse' });
      expect(continuation.sessionId).toBe(first.sessionId);
      expect(continuation.leaseId).not.toBe(first.leaseId);
      await runAction(continuation, action('click', { selector: '#counter' }));
      const continued = await fixtureAfter('effect', initial.effects.length);
      expect(continued.effects.at(-1)?.context).toBe(signedInIdentity);

      // A released pool entry cannot cross the admitted profile, locale,
      // viewport or storage identity. Each mismatch receives a new context.
      await releaseLease(continuation);
      const mismatches = [
        { ...base, reuseMode: 'reuse' as const, sessionProfileVersion: 'profile-B:1' },
        { ...base, reuseMode: 'reuse' as const, locale: 'fr-FR' },
        { ...base, reuseMode: 'reuse' as const, viewport: { width: 800, height: 600 } },
        { ...base, reuseMode: 'reuse' as const, storageState: { cookies: [], origins: [] } },
      ];
      for (const requested of mismatches) {
        const isolated = await open(requested);
        expect(isolated.sessionId).not.toBe(first.sessionId);
        await runAction(isolated, action('navigate', { url: `${env.fixture}/` }));
        const effectCount = (await fixtureState()).effects.length;
        await runAction(isolated, action('click', { selector: '#counter' }));
        const observed = await fixtureAfter('effect', effectCount);
        expect(observed.effects.at(-1)?.context).not.toBe(signedInIdentity);
        await closeLease(isolated);
      }

      // A live owner is not borrowable. Fresh starts also cannot inherit its identity.
      const liveLabels = { journey: 'J19-unreleased-owner' };
      const active = await open({ ...base, labels: liveLabels, reuseMode: 'fresh' });
      const blockedReuse = await open({ ...base, labels: liveLabels, reuseMode: 'reuse' });
      expect(blockedReuse.sessionId).not.toBe(active.sessionId);

      // Clean mode may continue the compatible context but clears its prior site identity.
      await releaseLease(continuation);
      const clean = await open({ ...base, reuseMode: 'clean' });
      expect(clean.sessionId).toBe(first.sessionId);
      await runAction(clean, action('navigate', { url: `${env.fixture}/` }));
      const cleanEffectCount = (await fixtureState()).effects.length;
      await runAction(clean, action('click', { selector: '#counter' }));
      const cleaned = await fixtureAfter('effect', cleanEffectCount);
      expect(cleaned.effects.at(-1)?.context).not.toBe(signedInIdentity);
  }));
});
