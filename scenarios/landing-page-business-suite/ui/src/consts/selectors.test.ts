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

  it('publishes a manifest covering both literal and dynamic selector contracts', () => {
    expect(selectorsManifest.selectors['admin.login.email']).toEqual({ testId: 'admin-login-email', selector: '[data-testid="admin-login-email"]' });
    expect(selectorsManifest.dynamicSelectors['admin.analytics.variantRow']).toEqual(expect.objectContaining({ testIdPattern: 'analytics-variant-row-${id}' }));
  });
});
