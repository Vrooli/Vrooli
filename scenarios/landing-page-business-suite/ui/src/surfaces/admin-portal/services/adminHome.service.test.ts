import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { AnalyticsSummary, DownloadApp, DownloadAsset, SiteBranding, Variant, VariantStats } from '../../../shared/api';
import {
  STALE_VARIANT_DAYS,
  DAY_MS,
  HEALTH_SNAPSHOT_DAYS,
  calculateDaysSinceUpdate,
  formatVariantUpdatedLabel,
  getAttentionPriority,
  describeWeightStatus,
  buildHealthSnapshot,
  computeBrandingHealth,
  computeDownloadsHealth,
  type WeightStatus,
  type VariantAttention,
} from './adminHome.service';

// Helper to create test branding
const createBranding = (overrides: Partial<SiteBranding>): SiteBranding => ({
  id: 1,
  site_name: 'Test Site',
  theme_primary_color: '#000',
  ...overrides,
} as SiteBranding);

// Helper to create test download apps
const createDownloadApp = (overrides: Partial<DownloadApp>): DownloadApp => ({
  bundle_key: 'test-bundle',
  app_key: 'test-app',
  name: 'Test App',
  platforms: [],
  ...overrides,
} as DownloadApp);

// Helper to create test download assets
const createDownloadAsset = (overrides: Partial<DownloadAsset>): DownloadAsset => ({
  bundle_key: 'test-bundle',
  app_key: 'test-app',
  platform: 'windows',
  artifact_url: '',
  release_version: '',
  requires_entitlement: false,
  ...overrides,
} as DownloadAsset);

// Helper to create test variant stats
const createVariantStats = (overrides: Partial<VariantStats>): VariantStats => ({
  variant_id: 1,
  variant_slug: 'test',
  variant_name: 'Test',
  views: 100,
  conversions: 10,
  conversion_rate: 10,
  cta_clicks: 5,
  downloads: 3,
  ...overrides,
} as VariantStats);

describe('adminHome.service', () => {
  describe('constants', () => {
    it('exports expected constant values', () => {
      expect(STALE_VARIANT_DAYS).toBe(10);
      expect(DAY_MS).toBe(24 * 60 * 60 * 1000);
      expect(HEALTH_SNAPSHOT_DAYS).toBe(7);
    });
  });

  describe('calculateDaysSinceUpdate', () => {
    let realDate: typeof Date;

    beforeEach(() => {
      realDate = global.Date;
      const mockDate = new Date('2025-01-15T12:00:00Z');
      vi.setSystemTime(mockDate);
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns null for null/undefined input', () => {
      expect(calculateDaysSinceUpdate(null)).toBeNull();
      expect(calculateDaysSinceUpdate(undefined)).toBeNull();
    });

    it('returns null for invalid date string', () => {
      expect(calculateDaysSinceUpdate('invalid-date')).toBeNull();
    });

    it('returns 0 for today', () => {
      expect(calculateDaysSinceUpdate('2025-01-15T00:00:00Z')).toBe(0);
    });

    it('returns 1 for yesterday', () => {
      expect(calculateDaysSinceUpdate('2025-01-14T00:00:00Z')).toBe(1);
    });

    it('returns correct days for older dates', () => {
      expect(calculateDaysSinceUpdate('2025-01-10T00:00:00Z')).toBe(5);
      expect(calculateDaysSinceUpdate('2025-01-01T00:00:00Z')).toBe(14);
    });

    it('returns 0 for future dates', () => {
      expect(calculateDaysSinceUpdate('2025-01-20T00:00:00Z')).toBe(0);
    });
  });

  describe('formatVariantUpdatedLabel', () => {
    let realDate: typeof Date;

    beforeEach(() => {
      realDate = global.Date;
      const mockDate = new Date('2025-01-15T12:00:00Z');
      vi.setSystemTime(mockDate);
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('returns "Never customized" for null/undefined', () => {
      expect(formatVariantUpdatedLabel(null)).toBe('Never customized');
      expect(formatVariantUpdatedLabel(undefined)).toBe('Never customized');
    });

    it('returns unavailable message for invalid date', () => {
      expect(formatVariantUpdatedLabel('invalid')).toBe('Last updated date unavailable');
    });

    it('returns "Updated today" for today', () => {
      expect(formatVariantUpdatedLabel('2025-01-15T08:00:00Z')).toBe('Updated today');
    });

    it('returns "Updated yesterday" for yesterday', () => {
      expect(formatVariantUpdatedLabel('2025-01-14T08:00:00Z')).toBe('Updated yesterday');
    });

    it('returns "Updated X days ago" for older dates', () => {
      expect(formatVariantUpdatedLabel('2025-01-10T08:00:00Z')).toBe('Updated 5 days ago');
      expect(formatVariantUpdatedLabel('2025-01-01T08:00:00Z')).toBe('Updated 14 days ago');
    });
  });

  describe('getAttentionPriority', () => {
    it('returns 3 for lowest conversion', () => {
      const entry: VariantAttention = {
        slug: 'test',
        name: 'Test',
        reasons: ['Lowest conversion'],
        updatedLabel: 'Updated today',
      };
      expect(getAttentionPriority(entry)).toBe(3);
    });

    it('returns 3 for lowest conversion (case insensitive)', () => {
      const entry: VariantAttention = {
        slug: 'test',
        name: 'Test',
        reasons: ['LOWEST CONVERSION'],
        updatedLabel: 'Updated today',
      };
      expect(getAttentionPriority(entry)).toBe(3);
    });

    it('returns 2 for stale variants', () => {
      const entry: VariantAttention = {
        slug: 'test',
        name: 'Test',
        reasons: ['Stale · 15d'],
        updatedLabel: 'Updated 15 days ago',
      };
      expect(getAttentionPriority(entry)).toBe(2);
    });

    it('returns 1 for never customized', () => {
      const entry: VariantAttention = {
        slug: 'test',
        name: 'Test',
        reasons: ['Never customized'],
        updatedLabel: 'Never customized',
      };
      expect(getAttentionPriority(entry)).toBe(1);
    });

    it('returns 0 for other reasons', () => {
      const entry: VariantAttention = {
        slug: 'test',
        name: 'Test',
        reasons: ['Some other reason'],
        updatedLabel: 'Updated today',
      };
      expect(getAttentionPriority(entry)).toBe(0);
    });

    it('returns highest priority when multiple reasons exist', () => {
      const entry: VariantAttention = {
        slug: 'test',
        name: 'Test',
        reasons: ['Stale · 15d', 'Lowest conversion'],
        updatedLabel: 'Updated today',
      };
      expect(getAttentionPriority(entry)).toBe(3);
    });
  });

  describe('describeWeightStatus', () => {
    it('returns no active variants message when count is 0', () => {
      expect(describeWeightStatus('empty', 0, 0)).toBe(
        'No active variants are routing traffic. Create one to render the public landing.'
      );
    });

    it('returns balanced message when status is balanced', () => {
      expect(describeWeightStatus('balanced', 100, 3)).toBe(
        'Traffic is fully allocated across variants.'
      );
    });

    it('returns under-allocated message when status is under', () => {
      expect(describeWeightStatus('under', 70, 3)).toBe(
        '30% of visitors are idle because weights total less than 100%.'
      );
    });

    it('returns over-allocated message when status is over', () => {
      expect(describeWeightStatus('over', 150, 3)).toBe(
        'Weights exceed 100% by 50%. Adjust them to match your intent.'
      );
    });

    it('returns default message for empty status with active variants', () => {
      expect(describeWeightStatus('empty', 0, 2)).toBe(
        'Assign weights to control where visitors land.'
      );
    });
  });

  describe('buildHealthSnapshot', () => {
    beforeEach(() => {
      const mockDate = new Date('2025-01-15T12:00:00Z');
      vi.setSystemTime(mockDate);
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const createVariant = (overrides: Partial<Variant> = {}): Variant => ({
      id: 1,
      slug: 'test-variant',
      name: 'Test Variant',
      status: 'active',
      weight: 50,
      updated_at: '2025-01-14T00:00:00Z',
      ...overrides,
    });

    it('counts active and archived variants correctly', () => {
      const variants = [
        createVariant({ slug: 'v1', status: 'active' }),
        createVariant({ slug: 'v2', status: 'active' }),
        createVariant({ slug: 'v3', status: 'archived' }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.activeCount).toBe(2);
      expect(result.archivedCount).toBe(1);
    });

    it('calculates total weight from active variants', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: 40, status: 'active' }),
        createVariant({ slug: 'v2', weight: 60, status: 'active' }),
        createVariant({ slug: 'v3', weight: 30, status: 'archived' }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.totalWeight).toBe(100);
    });

    it('returns empty weight status when no active variants', () => {
      const variants = [createVariant({ status: 'archived' })];

      const result = buildHealthSnapshot(variants, null);

      expect(result.weightStatus).toBe('empty');
    });

    it('returns balanced weight status when weights total 100', () => {
      const variants = [
        createVariant({ weight: 60 }),
        createVariant({ slug: 'v2', weight: 40 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.weightStatus).toBe('balanced');
    });

    it('returns under weight status when weights total less than 100', () => {
      const variants = [createVariant({ weight: 50 })];

      const result = buildHealthSnapshot(variants, null);

      expect(result.weightStatus).toBe('under');
    });

    it('returns over weight status when weights exceed 100', () => {
      const variants = [
        createVariant({ weight: 80 }),
        createVariant({ slug: 'v2', weight: 50 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.weightStatus).toBe('over');
    });

    it('flags never-customized variants', () => {
      const variants = [createVariant({ updated_at: undefined })];

      const result = buildHealthSnapshot(variants, null);

      expect(result.attentionCount).toBe(1);
      expect(result.highlightedAttention?.reasons).toContain('Never customized');
    });

    it('flags stale variants', () => {
      const staleDate = new Date('2025-01-01T00:00:00Z').toISOString();
      const variants = [createVariant({ updated_at: staleDate })];

      const result = buildHealthSnapshot(variants, null);

      expect(result.attentionCount).toBe(1);
      expect(result.highlightedAttention?.reasons[0]).toMatch(/^Stale/);
    });

    it('does not flag recently updated variants', () => {
      const recentDate = new Date('2025-01-14T00:00:00Z').toISOString();
      const variants = [createVariant({ updated_at: recentDate })];

      const result = buildHealthSnapshot(variants, null);

      expect(result.attentionCount).toBe(0);
      expect(result.highlightedAttention).toBeUndefined();
    });

    it('flags lowest conversion variant from analytics', () => {
      const variants = [
        createVariant({ slug: 'v1' }),
        createVariant({ slug: 'v2' }),
      ];
      const analytics: AnalyticsSummary = {
        total_visitors: 1000,
        variant_stats: [
          createVariantStats({ variant_slug: 'v1', views: 500, conversions: 30, conversion_rate: 6 }),
          createVariantStats({ variant_slug: 'v2', views: 500, conversions: 20, conversion_rate: 4 }),
        ],
      };

      const result = buildHealthSnapshot(variants, analytics);

      const attentionEntry = result.highlightedAttention;
      expect(attentionEntry?.slug).toBe('v2');
      expect(attentionEntry?.reasons).toContain('Lowest conversion');
      expect(attentionEntry?.conversionRate).toBe(4);
    });

    it('prioritizes lowest conversion over stale', () => {
      const staleDate = new Date('2025-01-01T00:00:00Z').toISOString();
      const variants = [
        createVariant({ slug: 'stale', updated_at: staleDate }),
        createVariant({ slug: 'low-conv', updated_at: '2025-01-14T00:00:00Z' }),
      ];
      const analytics: AnalyticsSummary = {
        total_visitors: 1000,
        variant_stats: [
          createVariantStats({ variant_slug: 'stale', views: 500, conversions: 30, conversion_rate: 6 }),
          createVariantStats({ variant_slug: 'low-conv', views: 500, conversions: 10, conversion_rate: 2 }),
        ],
      };

      const result = buildHealthSnapshot(variants, analytics);

      expect(result.highlightedAttention?.slug).toBe('low-conv');
      expect(result.highlightedAttention?.reasons).toContain('Lowest conversion');
    });

    it('ignores archived variants in analytics check', () => {
      const variants = [
        createVariant({ slug: 'active', status: 'active' }),
        createVariant({ slug: 'archived', status: 'archived' }),
      ];
      const analytics: AnalyticsSummary = {
        total_visitors: 1000,
        variant_stats: [
          createVariantStats({ variant_slug: 'active', views: 500, conversions: 30, conversion_rate: 6 }),
          createVariantStats({ variant_slug: 'archived', views: 500, conversions: 10, conversion_rate: 2 }),
        ],
      };

      const result = buildHealthSnapshot(variants, analytics);

      // Should not flag the archived variant despite lower conversion
      const attentionForArchived = result.highlightedAttention?.slug === 'archived';
      expect(attentionForArchived).toBe(false);
    });

    it('handles empty variants array', () => {
      const result = buildHealthSnapshot([], null);

      expect(result.activeCount).toBe(0);
      expect(result.archivedCount).toBe(0);
      expect(result.attentionCount).toBe(0);
      expect(result.totalWeight).toBe(0);
      expect(result.weightStatus).toBe('empty');
    });
  });

  describe('computeBrandingHealth', () => {
    it('returns all false for empty branding', () => {
      const branding = createBranding({
        site_name: undefined,
      });

      const result = computeBrandingHealth(branding);

      expect(result.hasIdentity).toBe(false);
      expect(result.hasFavicon).toBe(false);
      expect(result.hasSeo).toBe(false);
      expect(result.hasOgImage).toBe(false);
      expect(result.configuredCount).toBe(0);
      expect(result.totalChecks).toBe(4);
    });

    it('detects identity when site_name and logo_url are set', () => {
      const branding = createBranding({
        site_name: 'My Site',
        logo_url: 'https://example.com/logo.png',
      });

      const result = computeBrandingHealth(branding);

      expect(result.hasIdentity).toBe(true);
    });

    it('requires both site_name and logo_url for identity', () => {
      const brandingNoLogo = createBranding({
        site_name: 'My Site',
      });

      expect(computeBrandingHealth(brandingNoLogo).hasIdentity).toBe(false);
    });

    it('detects favicon when set', () => {
      const branding = createBranding({
        favicon_url: 'https://example.com/favicon.ico',
      });

      const result = computeBrandingHealth(branding);

      expect(result.hasFavicon).toBe(true);
    });

    it('detects SEO when title and description are set', () => {
      const branding = createBranding({
        default_title: 'My Site - Great Products',
        default_description: 'We offer great products',
      });

      const result = computeBrandingHealth(branding);

      expect(result.hasSeo).toBe(true);
    });

    it('requires both title and description for SEO', () => {
      const brandingNoDesc = createBranding({
        default_title: 'My Site',
      });

      expect(computeBrandingHealth(brandingNoDesc).hasSeo).toBe(false);
    });

    it('detects OG image when set', () => {
      const branding = createBranding({
        default_og_image_url: 'https://example.com/og.png',
      });

      const result = computeBrandingHealth(branding);

      expect(result.hasOgImage).toBe(true);
    });

    it('counts configured checks correctly', () => {
      const branding = createBranding({
        site_name: 'My Site',
        logo_url: 'https://example.com/logo.png',
        favicon_url: 'https://example.com/favicon.ico',
        default_title: 'My Site - Great Products',
        default_description: 'We offer great products',
        default_og_image_url: 'https://example.com/og.png',
      });

      const result = computeBrandingHealth(branding);

      expect(result.hasIdentity).toBe(true);
      expect(result.hasFavicon).toBe(true);
      expect(result.hasSeo).toBe(true);
      expect(result.hasOgImage).toBe(true);
      expect(result.configuredCount).toBe(4);
    });
  });

  describe('computeDownloadsHealth', () => {
    it('returns zero counts for empty apps array', () => {
      const result = computeDownloadsHealth([]);

      expect(result.appCount).toBe(0);
      expect(result.platformsConfigured).toBe(0);
      expect(result.storefrontsConfigured).toBe(0);
      expect(result.hasApps).toBe(false);
    });

    it('counts apps correctly', () => {
      const apps = [
        createDownloadApp({ app_key: 'app1', name: 'App 1' }),
        createDownloadApp({ app_key: 'app2', name: 'App 2' }),
      ];

      const result = computeDownloadsHealth(apps);

      expect(result.appCount).toBe(2);
      expect(result.hasApps).toBe(true);
    });

    it('counts configured platforms', () => {
      const apps = [
        createDownloadApp({
          app_key: 'app1',
          name: 'App 1',
          platforms: [
            createDownloadAsset({ platform: 'windows', artifact_url: 'https://example.com/app.exe', release_version: '1.0.0' }),
            createDownloadAsset({ platform: 'mac', artifact_url: 'https://example.com/app.dmg', release_version: '1.0.0' }),
            createDownloadAsset({ platform: 'linux', artifact_url: '', release_version: '' }), // Not configured
          ],
        }),
      ];

      const result = computeDownloadsHealth(apps);

      expect(result.platformsConfigured).toBe(2);
    });

    it('requires both artifact_url and release_version for configured platform', () => {
      const apps = [
        createDownloadApp({
          app_key: 'app1',
          name: 'App 1',
          platforms: [
            createDownloadAsset({ platform: 'windows', artifact_url: 'https://example.com/app.exe', release_version: '' }),
            createDownloadAsset({ platform: 'mac', artifact_url: '', release_version: '1.0.0' }),
          ],
        }),
      ];

      const result = computeDownloadsHealth(apps);

      expect(result.platformsConfigured).toBe(0);
    });

    it('counts configured storefronts', () => {
      const apps = [
        createDownloadApp({
          app_key: 'app1',
          name: 'App 1',
          storefronts: [
            { store: 'app_store', label: 'App Store', url: 'https://apps.apple.com/123' },
            { store: 'play_store', label: 'Google Play', url: '' }, // Not configured
          ],
        }),
      ];

      const result = computeDownloadsHealth(apps);

      expect(result.storefrontsConfigured).toBe(1);
    });

    it('handles apps without platforms or storefronts', () => {
      const apps = [
        createDownloadApp({ app_key: 'app1', name: 'App 1' }),
      ];

      const result = computeDownloadsHealth(apps);

      expect(result.appCount).toBe(1);
      expect(result.platformsConfigured).toBe(0);
      expect(result.storefrontsConfigured).toBe(0);
      expect(result.hasApps).toBe(true);
    });

    it('aggregates counts across multiple apps', () => {
      const apps = [
        createDownloadApp({
          app_key: 'app1',
          name: 'App 1',
          platforms: [
            createDownloadAsset({ platform: 'windows', artifact_url: 'https://example.com/app1.exe', release_version: '1.0.0' }),
          ],
          storefronts: [
            { store: 'app_store', label: 'App Store', url: 'https://apps.apple.com/1' },
          ],
        }),
        createDownloadApp({
          app_key: 'app2',
          name: 'App 2',
          platforms: [
            createDownloadAsset({ platform: 'mac', artifact_url: 'https://example.com/app2.dmg', release_version: '2.0.0' }),
            createDownloadAsset({ platform: 'linux', artifact_url: 'https://example.com/app2.deb', release_version: '2.0.0' }),
          ],
          storefronts: [
            { store: 'play_store', label: 'Google Play', url: 'https://play.google.com/2' },
          ],
        }),
      ];

      const result = computeDownloadsHealth(apps);

      expect(result.appCount).toBe(2);
      expect(result.platformsConfigured).toBe(3);
      expect(result.storefrontsConfigured).toBe(2);
    });
  });

  describe('edge cases - weight handling', () => {
    beforeEach(() => {
      const mockDate = new Date('2025-01-15T12:00:00Z');
      vi.setSystemTime(mockDate);
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const createVariant = (overrides: Partial<Variant> = {}): Variant => ({
      id: 1,
      slug: 'test-variant',
      name: 'Test Variant',
      status: 'active',
      weight: 50,
      updated_at: '2025-01-14T00:00:00Z',
      ...overrides,
    });

    it('handles weight of 0%', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: 0 }),
        createVariant({ slug: 'v2', weight: 100 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.totalWeight).toBe(100);
      expect(result.weightStatus).toBe('balanced');
    });

    it('handles all weights at 0%', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: 0 }),
        createVariant({ slug: 'v2', weight: 0 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.totalWeight).toBe(0);
      expect(result.weightStatus).toBe('under');
    });

    it('handles negative weight values', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: -10 }),
        createVariant({ slug: 'v2', weight: 110 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.totalWeight).toBe(100);
      expect(result.weightStatus).toBe('balanced');
    });

    it('handles extremely high weight values', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: 500 }),
        createVariant({ slug: 'v2', weight: 500 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.totalWeight).toBe(1000);
      expect(result.weightStatus).toBe('over');
    });

    it('handles undefined weight', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: undefined as unknown as number }),
        createVariant({ slug: 'v2', weight: 100 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      // undefined should be treated as 0
      expect(result.totalWeight).toBe(100);
    });

    it('handles null weight', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: null as unknown as number }),
        createVariant({ slug: 'v2', weight: 100 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.totalWeight).toBe(100);
    });

    it('handles decimal weight values', () => {
      const variants = [
        createVariant({ slug: 'v1', weight: 33.33 }),
        createVariant({ slug: 'v2', weight: 33.33 }),
        createVariant({ slug: 'v3', weight: 33.34 }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.totalWeight).toBe(100);
      expect(result.weightStatus).toBe('balanced');
    });
  });

  describe('edge cases - variant attention with multiple reasons', () => {
    beforeEach(() => {
      const mockDate = new Date('2025-01-15T12:00:00Z');
      vi.setSystemTime(mockDate);
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    const createVariant = (overrides: Partial<Variant> = {}): Variant => ({
      id: 1,
      slug: 'test-variant',
      name: 'Test Variant',
      status: 'active',
      weight: 50,
      updated_at: '2025-01-14T00:00:00Z',
      ...overrides,
    });

    it('accumulates multiple attention reasons on same variant', () => {
      // Create a variant that is both stale AND has lowest conversion
      const staleDate = new Date('2025-01-01T00:00:00Z').toISOString();
      const variants = [
        createVariant({ slug: 'problem-variant', updated_at: staleDate }),
        createVariant({ slug: 'good-variant', updated_at: '2025-01-14T00:00:00Z' }),
      ];
      const analytics: AnalyticsSummary = {
        total_visitors: 1000,
        variant_stats: [
          createVariantStats({ variant_slug: 'problem-variant', views: 500, conversions: 5, conversion_rate: 1 }),
          createVariantStats({ variant_slug: 'good-variant', views: 500, conversions: 50, conversion_rate: 10 }),
        ],
      };

      const result = buildHealthSnapshot(variants, analytics);

      expect(result.highlightedAttention?.slug).toBe('problem-variant');
      expect(result.highlightedAttention?.reasons.length).toBeGreaterThanOrEqual(2);
      expect(result.highlightedAttention?.reasons).toContain('Lowest conversion');
      expect(result.highlightedAttention?.reasons.some(r => r.startsWith('Stale'))).toBe(true);
    });

    it('counts attention correctly when multiple variants need attention', () => {
      const staleDate = new Date('2025-01-01T00:00:00Z').toISOString();
      const variants = [
        createVariant({ slug: 'stale1', updated_at: staleDate }),
        createVariant({ slug: 'stale2', updated_at: staleDate }),
        createVariant({ slug: 'never-customized', updated_at: undefined }),
      ];

      const result = buildHealthSnapshot(variants, null);

      expect(result.attentionCount).toBe(3);
    });

    it('handles single variant with very low conversion rate', () => {
      const variants = [createVariant({ slug: 'only' })];
      const analytics: AnalyticsSummary = {
        total_visitors: 1000,
        variant_stats: [
          createVariantStats({ variant_slug: 'only', views: 1000, conversions: 0, conversion_rate: 0 }),
        ],
      };

      const result = buildHealthSnapshot(variants, analytics);

      // Single variant IS flagged for lowest conversion - this surfaces the issue to the user
      expect(result.attentionCount).toBe(1);
      expect(result.highlightedAttention?.reasons).toContain('Lowest conversion');
    });
  });

  describe('edge cases - unicode and special characters', () => {
    it('handles unicode characters in variant names', () => {
      const entry: VariantAttention = {
        slug: 'test-日本語',
        name: '日本語テスト 🚀',
        reasons: ['Lowest conversion'],
        updatedLabel: 'Updated today',
      };
      expect(getAttentionPriority(entry)).toBe(3);
    });

    it('handles empty string reasons array', () => {
      const entry: VariantAttention = {
        slug: 'test',
        name: 'Test',
        reasons: [],
        updatedLabel: 'Updated today',
      };
      expect(getAttentionPriority(entry)).toBe(0);
    });

    it('handles branding with unicode site name', () => {
      const branding = createBranding({
        site_name: '日本語サイト 🌟',
        logo_url: 'https://example.com/logo.png',
      });

      const result = computeBrandingHealth(branding);

      expect(result.hasIdentity).toBe(true);
    });

    it('handles branding with whitespace-only values', () => {
      const branding = createBranding({
        site_name: '   ',
        logo_url: '   ',
      });

      const result = computeBrandingHealth(branding);

      // Whitespace-only strings are truthy but should probably be treated as empty
      // This tests current behavior - site_name with spaces is considered "set"
      expect(result.hasIdentity).toBe(true);
    });
  });
});
