import { action, closeLease, env, openLease, registerLeaseCleanup, releaseLease, runAction, withJourneyResources } from './support';

describe('[REQ:BAS-RH-J15] session capacity and reuse', () => {
  it('given two admitted sessions at once, when one lease is released and reacquired by label, then capacity is freed and browser identity is reused', () => withJourneyResources(async (resources) => {
      const open = async (...args: Parameters<typeof openLease>) => {
        const lease = await openLease(...args);
        registerLeaseCleanup(resources, lease);
        return lease;
      };
      const first = await openLease();
      registerLeaseCleanup(resources, first);
      const second = await open();
      expect(second.sessionId).not.toBe(first.sessionId);
      await runAction(first, action('navigate', { url: `${env.fixture}/forms` }));
      await closeLease(first);
      const reusable = await open({ labels: { journey: 'J15-reusable' } });
      await releaseLease(reusable);
      const reused = await open({ reuseMode: 'reuse', labels: { journey: 'J15-reusable' } });
      expect(reused.sessionId).toBe(reusable.sessionId);
      expect(reused.leaseId).not.toBe(first.leaseId);
  }));
});
