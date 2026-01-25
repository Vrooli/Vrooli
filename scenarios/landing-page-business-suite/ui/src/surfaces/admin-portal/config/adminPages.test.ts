import { describe, it, expect } from 'vitest';
import { getAdminPageByPath, getAdminPageDocLink } from './adminPages';

describe('adminPages config', () => {
  it('matches dynamic routes to the correct page', () => {
    const page = getAdminPageByPath('/admin/customization/variants/control/sections/42');

    expect(page?.id).toBe('section-editor');
  });

  it('builds doc links for pages with documentation', () => {
    const docLink = getAdminPageDocLink('/admin/customization/variants/control');

    expect(docLink).not.toBeNull();
    expect(docLink?.url).toBe('/admin/docs?doc=guides%2FADMIN_GUIDE.md#variant-editor');
  });

  it('returns null when a page has no documentation', () => {
    const docLink = getAdminPageDocLink('/admin/unknown');

    expect(docLink).toBeNull();
  });
});
