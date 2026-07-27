import { beforeEach, describe, expect, it, vi } from 'vitest';
import * as waitlist from './waitlist';
import { apiCall } from './common';

vi.mock('./common', () => ({ API_BASE: 'http://api.example', apiCall: vi.fn() }));
const mockApiCall = vi.mocked(apiCall);

describe('waitlist API transport', () => {
  beforeEach(() => { vi.clearAllMocks(); mockApiCall.mockResolvedValue({} as never); });

  it('submits public signups, manages admin records, and exposes the configured export URL', async () => {
    await waitlist.submitWaitlistEmail('customer@example.com');
    await waitlist.submitWaitlistEmail('partner@example.com', 'campaign');
    await waitlist.getWaitlistEmails();
    await waitlist.deleteWaitlistEmail(7);
    expect(waitlist.getWaitlistExportUrl()).toBe('http://api.example/admin/waitlist/export');
    expect(mockApiCall).toHaveBeenCalledWith('/waitlist', expect.objectContaining({ method: 'POST', body: JSON.stringify({ email: 'customer@example.com', source: 'coming_soon' }) }));
    expect(mockApiCall).toHaveBeenCalledWith('/waitlist', expect.objectContaining({ method: 'POST', body: JSON.stringify({ email: 'partner@example.com', source: 'campaign' }) }));
    expect(mockApiCall).toHaveBeenCalledWith('/admin/waitlist');
    expect(mockApiCall).toHaveBeenCalledWith('/admin/waitlist/7', { method: 'DELETE' });
  });
});
