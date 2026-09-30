import { describe, it, expect } from 'vitest';
import { selectors, selectorsManifest } from './selectors';

describe('selectors registry', () => {
  it('exposes literal selectors as plain testId strings', () => {
    expect((selectors as any).admin.login.email).toBe('admin-login-email');
    expect((selectors as any).admin.nav.logout).toBe('nav-logout');
  });

  it('exposes dynamic selectors as functions that interpolate params', () => {
    const fn = (selectors as any).admin.customization.variantCard;
    expect(typeof fn).toBe('function');
    expect(fn({ slug: 'hero' })).toBe('variant-card-hero');
    expect((selectors as any).admin.customization.section({ id: 12 })).toBe('section-12');
  });

  it('throws a descriptive error when a required param is missing', () => {
    expect(() => (selectors as any).admin.customization.variantCard({} as never)).toThrow(/slug/);
  });

  it('throws when a numeric param is given a non-numeric value', () => {
    expect(() => (selectors as any).admin.customization.section({ id: 'x' } as never)).toThrow(/numeric/);
  });

  it('rejects unknown extra params', () => {
    expect(() =>
      (selectors as any).admin.customization.variantCard({ slug: 'hero', extra: 'nope' } as never),
    ).toThrow(/unknown parameter/);
  });
});

describe('selectorsManifest', () => {
  it('flattens literal selectors into dotted keys with a CSS selector', () => {
    expect(selectorsManifest.selectors['admin.login.email']).toEqual({
      testId: 'admin-login-email',
      selector: '[data-testid="admin-login-email"]',
    });
  });

  it('describes dynamic selectors with their params and pattern', () => {
    const entry = selectorsManifest.dynamicSelectors['admin.customization.variantCard']!;
    expect(entry.testIdPattern).toBe('variant-card-${slug}');
    expect(entry.selectorPattern).toContain('data-testid');
    expect(entry.params).toEqual([{ name: 'slug', type: 'string', values: undefined }]);
  });
});
