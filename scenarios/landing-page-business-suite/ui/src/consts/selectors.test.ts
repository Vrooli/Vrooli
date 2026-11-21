import { describe, expect, it } from 'vitest';
import { selectors, selectorsManifest } from './selectors';

type RegistryUnderTest = {
  admin: {
    login: { email: string };
    nav: { analytics: string };
    analytics: { variantRow: (params: { id: number }) => string };
    customization: { variantCard: (params: { slug: string }) => string };
  };
  publicLanding: { surface: string };
};

type DynamicSelector = (params: Record<string, string | number | undefined>) => string;

const registry = selectors as unknown as RegistryUnderTest;

describe('selector registry', () => {
  it('exposes stable literal selectors for core admin and public experiences', () => {
    expect(registry.admin.login.email).toBe('admin-login-email');
    expect(registry.admin.nav.analytics).toBe('nav-analytics');
    expect(registry.publicLanding.surface).toBe('landing-experience-surface');
  });

  it('formats typed dynamic selectors and rejects missing required parameters', () => {
    expect(registry.admin.analytics.variantRow({ id: 42 })).toBe('analytics-variant-row-42');
    expect(registry.admin.customization.variantCard({ slug: 'control' })).toBe('variant-card-control');
    expect(() => (registry.admin.analytics.variantRow as unknown as () => string)()).toThrow("missing parameter 'id'");
  });

  it('keeps every published dynamic selector stable and rejects invalid selector contracts', () => {
    const dynamic = selectors as unknown as {
      admin: {
        analytics: {
          viewDetails: DynamicSelector;
          editVariant: DynamicSelector;
          variantRow: DynamicSelector;
        };
        customization: {
          editVariant: DynamicSelector;
          previewVariant: DynamicSelector;
          archiveVariant: DynamicSelector;
          variantAnalytics: DynamicSelector;
          variantPerformance: DynamicSelector;
          variantStatus: DynamicSelector;
          deleteVariant: DynamicSelector;
          section: DynamicSelector;
          editSection: DynamicSelector;
        };
        breadcrumbSegment: DynamicSelector;
      };
    };

    expect(dynamic.admin.analytics.viewDetails({ id: 7 })).toBe('analytics-view-details-7');
    expect(dynamic.admin.analytics.editVariant({ id: 7 })).toBe('analytics-edit-7');
    expect(dynamic.admin.customization.editVariant({ slug: 'control' })).toBe('edit-variant-control');
    expect(dynamic.admin.customization.previewVariant({ slug: 'control' })).toBe('preview-variant-control');
    expect(dynamic.admin.customization.archiveVariant({ slug: 'control' })).toBe('archive-variant-control');
    expect(dynamic.admin.customization.variantAnalytics({ slug: 'control' })).toBe('variant-analytics-control');
    expect(dynamic.admin.customization.variantPerformance({ slug: 'control' })).toBe('variant-performance-control');
    expect(dynamic.admin.customization.variantStatus({ slug: 'control' })).toBe('variant-status-control');
    expect(dynamic.admin.customization.deleteVariant({ slug: 'control' })).toBe('delete-variant-control');
    expect(dynamic.admin.customization.section({ id: 4 })).toBe('section-4');
    expect(dynamic.admin.customization.editSection({ id: 4 })).toBe('edit-section-4');
    expect(dynamic.admin.breadcrumbSegment({ index: 2 })).toBe('breadcrumb-2');

    expect(() => dynamic.admin.analytics.variantRow({ id: '7' })).toThrow("must be numeric");
    expect(() => dynamic.admin.analytics.variantRow({ id: undefined })).toThrow("is undefined");
    expect(() => dynamic.admin.analytics.variantRow({ id: 7, extra: 'unexpected' })).toThrow("unknown parameter(s): extra");
  });

  it('publishes a manifest covering both literal and dynamic selector contracts', () => {
    expect(selectorsManifest.selectors['admin.login.email']).toEqual({ testId: 'admin-login-email', selector: '[data-testid="admin-login-email"]' });
    expect(selectorsManifest.dynamicSelectors['admin.analytics.variantRow']).toEqual(expect.objectContaining({ testIdPattern: 'analytics-variant-row-${id}' }));
    expect(selectorsManifest.dynamicSelectors['admin.breadcrumbSegment']).toEqual(expect.objectContaining({ testIdPattern: 'breadcrumb-${index}' }));
  });
});
