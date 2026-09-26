import { describe, expect, it } from 'vitest';
import { buildDefaultHeaderConfig, cloneHeaderConfig, normalizeHeaderConfig } from './headerConfig';

describe('headerConfig', () => {
  it('builds a safe complete header when no persisted configuration exists', () => {
    expect(buildDefaultHeaderConfig()).toMatchObject({
      branding: { mode: 'logo_and_name', label: 'Landing', mobile_preference: 'auto' },
      ctas: { primary: { mode: 'inherit_hero' }, secondary: { mode: 'downloads' } },
      behavior: { sticky: true, hide_on_scroll: false },
    });
    expect(normalizeHeaderConfig(null, 'Acme')).toMatchObject({ branding: { label: 'Acme' } });
  });

  it('normalizes partial migrated data without allowing an invisible navigation item', () => {
    const normalized = normalizeHeaderConfig({
      branding: { mode: 'name', label: '', mobile_preference: 'logo' },
      nav: { links: [
        { type: 'section', label: '', section_type: 'faq', visible_on: { desktop: false, mobile: false } },
        { id: 'parent', type: 'menu', label: 'Resources', children: [{ type: 'custom', href: '/docs', visible_on: { desktop: false } }] },
      ] },
      ctas: { primary: { mode: 'custom', label: 'Contact', href: '/contact' } },
      behavior: { sticky: false },
    } as never, 'Fallback name');

    expect(normalized.branding).toMatchObject({ mode: 'name', label: '', mobile_preference: 'logo' });
    expect(normalized.nav.links[0]).toMatchObject({ id: 'nav-section-0', label: 'Section', visible_on: { desktop: true, mobile: true } });
    expect(normalized.nav.links[1]).toMatchObject({ id: 'parent', label: 'Resources' });
    expect(normalized.nav.links[1]?.children?.[0]).toMatchObject({ id: 'nav-custom-0', label: 'Section', visible_on: { desktop: false, mobile: true } });
    expect(normalized.ctas.primary).toMatchObject({ mode: 'custom', label: 'Contact', href: '/contact', variant: 'solid' });
    expect(normalized.ctas.secondary).toMatchObject({ mode: 'downloads', variant: 'ghost' });
    expect(normalized.behavior).toEqual({ sticky: false, hide_on_scroll: false });
  });

  it('deep-clones header state before editors mutate a draft', () => {
    const source = buildDefaultHeaderConfig('Source');
    const clone = cloneHeaderConfig(source);
    clone.branding.label = 'Draft';
    clone.nav.links.push({ id: 'custom', type: 'custom', label: 'Docs', href: '/docs', visible_on: { desktop: true, mobile: true } });

    expect(source.branding.label).toBe('Source');
    expect(source.nav.links).toEqual([]);
  });
});
