import { beforeEach, describe, expect, it, vi } from 'vitest';

const waitlistClient = vi.hoisted(() => ({ createWaitlistEntry: vi.fn(), listWaitlistEntries: vi.fn(), deleteWaitlistEntry: vi.fn(), exportWaitlistEntries: vi.fn() }));
vi.mock('@connectrpc/connect', () => ({ createClient: vi.fn(() => waitlistClient) }));

import * as waitlist from './waitlist';

describe('waitlist API transport', () => {
  beforeEach(() => vi.clearAllMocks());

  it('uses generated Connect procedures for public signup and protected lifecycle operations', async () => {
    waitlistClient.createWaitlistEntry.mockResolvedValue({ success: true, message: 'Email added to waitlist' });
    waitlistClient.listWaitlistEntries.mockResolvedValue({ entries: [{ id: 7n, email: 'customer@example.com', source: 'campaign', createdAt: { seconds: 0n, nanos: 0 } }] });
    waitlistClient.deleteWaitlistEntry.mockResolvedValue({ deleted: true });
    waitlistClient.exportWaitlistEntries.mockResolvedValue({ csv: 'ID,Email\n7,customer@example.com\n', filename: 'waitlist.csv' });

    await expect(waitlist.submitWaitlistEmail('customer@example.com', 'campaign')).resolves.toEqual({ success: true, message: 'Email added to waitlist' });
    await expect(waitlist.getWaitlistEmails()).resolves.toMatchObject([{ id: 7, email: 'customer@example.com', source: 'campaign' }]);
    await expect(waitlist.deleteWaitlistEmail(7)).resolves.toEqual({ success: true });
    await expect(waitlist.exportWaitlistCsv()).resolves.toEqual({ csv: 'ID,Email\n7,customer@example.com\n', filename: 'waitlist.csv' });
    expect(waitlistClient.createWaitlistEntry).toHaveBeenCalledWith({ email: 'customer@example.com', source: 'campaign' });
    expect(waitlistClient.deleteWaitlistEntry).toHaveBeenCalledWith({ id: 7n });
  });

  it('preserves missing timestamps and supplies a safe export filename', async () => {
    waitlistClient.listWaitlistEntries.mockResolvedValue({
      entries: [{ id: 8n, email: 'new@example.com', source: 'landing' }],
    });
    waitlistClient.exportWaitlistEntries.mockResolvedValue({ csv: 'ID,Email\n8,new@example.com\n', filename: '' });

    await expect(waitlist.getWaitlistEmails()).resolves.toEqual([
      { id: 8, email: 'new@example.com', source: 'landing', created_at: '' },
    ]);
    await expect(waitlist.exportWaitlistCsv()).resolves.toEqual({
      csv: 'ID,Email\n8,new@example.com\n', filename: 'waitlist.csv',
    });
  });
});
