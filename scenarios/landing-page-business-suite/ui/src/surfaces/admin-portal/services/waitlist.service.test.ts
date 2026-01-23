import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { WaitlistEmail, SiteBranding } from '../../../shared/api';
import {
  calculateStats,
  fetchWaitlistEmails,
  deleteWaitlistEmail,
  fetchBranding,
  toggleComingSoonMode,
  getExportUrl,
  exportToCsv,
} from './waitlist.service';
import * as waitlistApi from '../../../shared/api/waitlist';
import * as brandingApi from '../../../shared/api/branding';
import { createWindowOpenMock } from '../../../shared/test-utils/api-mocks';

vi.mock('../../../shared/api/waitlist', () => ({
  getWaitlistEmails: vi.fn(),
  deleteWaitlistEmail: vi.fn(),
  getWaitlistExportUrl: vi.fn(() => 'https://api.example.com/waitlist/export'),
}));

vi.mock('../../../shared/api/branding', () => ({
  getBranding: vi.fn(),
  updateBranding: vi.fn(),
}));

const createMockEmail = (overrides: Partial<WaitlistEmail> = {}): WaitlistEmail => ({
  id: 1,
  email: 'test@example.com',
  source: 'coming_soon',
  created_at: '2024-01-15T10:30:00Z',
  ...overrides,
});

const createMockBranding = (overrides: Partial<SiteBranding> = {}): SiteBranding => ({
  id: 1,
  site_name: 'Test Site',
  coming_soon_enabled: false,
  ...overrides,
} as SiteBranding);

describe('waitlist.service', () => {
  describe('calculateStats', () => {
    it('calculates correct stats for empty list', () => {
      const stats = calculateStats([]);
      expect(stats.totalSignups).toBe(0);
      expect(stats.comingSoonCount).toBe(0);
    });

    it('calculates correct stats for mixed sources', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'coming_soon' }),
        createMockEmail({ id: 2, source: 'coming_soon' }),
        createMockEmail({ id: 3, source: 'newsletter' }),
        createMockEmail({ id: 4, source: 'referral' }),
      ];

      const stats = calculateStats(emails);
      expect(stats.totalSignups).toBe(4);
      expect(stats.comingSoonCount).toBe(2);
    });

    it('calculates correct stats when all from coming_soon', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'coming_soon' }),
        createMockEmail({ id: 2, source: 'coming_soon' }),
      ];

      const stats = calculateStats(emails);
      expect(stats.totalSignups).toBe(2);
      expect(stats.comingSoonCount).toBe(2);
    });

    it('calculates correct stats when none from coming_soon', () => {
      const emails: WaitlistEmail[] = [
        createMockEmail({ id: 1, source: 'newsletter' }),
        createMockEmail({ id: 2, source: 'referral' }),
      ];

      const stats = calculateStats(emails);
      expect(stats.totalSignups).toBe(2);
      expect(stats.comingSoonCount).toBe(0);
    });
  });

  describe('fetchWaitlistEmails', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('returns emails from API', async () => {
      const mockEmails = [createMockEmail({ id: 1 }), createMockEmail({ id: 2 })];
      vi.mocked(waitlistApi.getWaitlistEmails).mockResolvedValue(mockEmails);

      const result = await fetchWaitlistEmails();

      expect(result).toEqual(mockEmails);
      expect(waitlistApi.getWaitlistEmails).toHaveBeenCalledTimes(1);
    });

    it('returns empty array when API returns null', async () => {
      vi.mocked(waitlistApi.getWaitlistEmails).mockResolvedValue(null as unknown as WaitlistEmail[]);

      const result = await fetchWaitlistEmails();

      expect(result).toEqual([]);
    });

    it('propagates API errors', async () => {
      vi.mocked(waitlistApi.getWaitlistEmails).mockRejectedValue(new Error('Network error'));

      await expect(fetchWaitlistEmails()).rejects.toThrow('Network error');
    });
  });

  describe('deleteWaitlistEmail', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('deletes email via API', async () => {
      vi.mocked(waitlistApi.deleteWaitlistEmail).mockResolvedValue(undefined);

      await deleteWaitlistEmail(1);

      expect(waitlistApi.deleteWaitlistEmail).toHaveBeenCalledWith(1);
    });

    it('propagates API errors', async () => {
      vi.mocked(waitlistApi.deleteWaitlistEmail).mockRejectedValue(new Error('Not found'));

      await expect(deleteWaitlistEmail(999)).rejects.toThrow('Not found');
    });
  });

  describe('fetchBranding', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('returns branding from API', async () => {
      const mockBranding = createMockBranding({ coming_soon_enabled: true });
      vi.mocked(brandingApi.getBranding).mockResolvedValue(mockBranding);

      const result = await fetchBranding();

      expect(result).toEqual(mockBranding);
      expect(brandingApi.getBranding).toHaveBeenCalledTimes(1);
    });

    it('propagates API errors', async () => {
      vi.mocked(brandingApi.getBranding).mockRejectedValue(new Error('Server error'));

      await expect(fetchBranding()).rejects.toThrow('Server error');
    });
  });

  describe('toggleComingSoonMode', () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it('toggles from false to true', async () => {
      vi.mocked(brandingApi.updateBranding).mockResolvedValue(createMockBranding({ coming_soon_enabled: true }));

      const result = await toggleComingSoonMode(false);

      expect(result).toBe(true);
      expect(brandingApi.updateBranding).toHaveBeenCalledWith({ coming_soon_enabled: true });
    });

    it('toggles from true to false', async () => {
      vi.mocked(brandingApi.updateBranding).mockResolvedValue(createMockBranding({ coming_soon_enabled: false }));

      const result = await toggleComingSoonMode(true);

      expect(result).toBe(false);
      expect(brandingApi.updateBranding).toHaveBeenCalledWith({ coming_soon_enabled: false });
    });

    it('propagates API errors', async () => {
      vi.mocked(brandingApi.updateBranding).mockRejectedValue(new Error('Update failed'));

      await expect(toggleComingSoonMode(false)).rejects.toThrow('Update failed');
    });
  });

  describe('getExportUrl', () => {
    it('returns export URL from API', () => {
      const result = getExportUrl();
      expect(result).toBe('https://api.example.com/waitlist/export');
    });
  });

  describe('exportToCsv', () => {
    let windowMock: ReturnType<typeof createWindowOpenMock>;

    beforeEach(() => {
      windowMock = createWindowOpenMock();
    });

    afterEach(() => {
      windowMock.restore();
    });

    it('opens export URL in new window', () => {
      exportToCsv();
      expect(windowMock.mock).toHaveBeenCalledWith(
        'https://api.example.com/waitlist/export',
        '_blank'
      );
    });
  });
});
