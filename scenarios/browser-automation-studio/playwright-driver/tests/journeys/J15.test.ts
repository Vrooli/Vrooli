import { action, closeLease, env, openLease, releaseLease, runAction, type Lease } from './support';

describe('[REQ:BAS-RH-J15] session capacity and reuse', () => {
  it('given two admitted sessions at once, when one lease is released and reacquired by label, then capacity is freed and browser identity is reused', async () => {
    const leases: Lease[] = [];
    try {
      const first = await openLease();
      leases.push(first);
      const second = await openLease();
      leases.push(second);
      expect(second.sessionId).not.toBe(first.sessionId);
      await runAction(first, action('navigate', { url: `${env.fixture}/forms` }));
      await closeLease(first);
      const reusable = await openLease({ labels: { journey: 'J15-reusable' } });
      leases.push(reusable);
      await releaseLease(reusable);
      const reused = await openLease({ reuseMode: 'reuse', labels: { journey: 'J15-reusable' } });
      leases.push(reused);
      expect(reused.sessionId).toBe(reusable.sessionId);
      expect(reused.leaseId).not.toBe(first.leaseId);
    } finally {
      for (const lease of leases.reverse()) await closeLease(lease).catch(() => undefined);
    }
  });
});
