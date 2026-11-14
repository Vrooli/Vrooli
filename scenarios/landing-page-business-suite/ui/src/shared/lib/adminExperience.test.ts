import { afterEach, describe, expect, it, vi } from 'vitest';
import { getAdminExperienceSnapshot, rememberAnalyticsFilters, rememberVariantSession } from './adminExperience';

describe('adminExperience', () => {
  afterEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('returns a safe base snapshot for missing, malformed, and obsolete local data', () => {
    expect(getAdminExperienceSnapshot()).toEqual({ version: 1 });
    localStorage.setItem('landing_admin_experience', '{not json');
    expect(vi.spyOn(console, 'warn').mockImplementation(() => undefined));
    expect(getAdminExperienceSnapshot()).toEqual({ version: 1 });
    localStorage.setItem('landing_admin_experience', JSON.stringify({ version: 0, lastVariant: { slug: 'old' } }));
    expect(getAdminExperienceSnapshot()).toEqual({ version: 1 });
  });

  it('persists the last variant and analytics context together for operator continuity', () => {
    const variant = rememberVariantSession({ slug: 'launch', name: 'Launch', surface: 'section', sectionId: 3, sectionType: 'pricing' });
    expect(variant.lastVariant).toMatchObject({ slug: 'launch', name: 'Launch', surface: 'section', sectionId: 3, sectionType: 'pricing' });
    expect(variant.lastVariant?.lastVisitedAt).toMatch(/^\d{4}-\d{2}-\d{2}T/);

    const analytics = rememberAnalyticsFilters({ variantSlug: null, timeRangeDays: 30 });
    expect(analytics.lastVariant?.slug).toBe('launch');
    expect(analytics.lastAnalytics).toMatchObject({ variantSlug: null, timeRangeDays: 30 });
    expect(getAdminExperienceSnapshot()).toMatchObject({ version: 1, lastVariant: { slug: 'launch' }, lastAnalytics: { timeRangeDays: 30 } });
  });

  it('does not throw when browser storage declines a persistence attempt', () => {
    const warning = vi.spyOn(console, 'warn').mockImplementation(() => undefined);
    vi.spyOn(Storage.prototype, 'setItem').mockImplementation(() => { throw new Error('quota exceeded'); });

    expect(() => rememberAnalyticsFilters({ variantSlug: 'launch', variantName: 'Launch', timeRangeDays: 7 })).not.toThrow();
    expect(warning).toHaveBeenCalledWith('Failed to persist admin experience snapshot:', expect.any(Error));
  });
});
