import { describe, it, expect } from 'vitest';
import { buildDefaultHeaderConfig, normalizeHeaderConfig, cloneHeaderConfig } from './headerConfig';

describe('buildDefaultHeaderConfig', () => {
  it('uses the provided name and falls back to Landing', () => {
    expect(buildDefaultHeaderConfig('Acme').branding?.label).toBe('Acme');
    expect(buildDefaultHeaderConfig().branding?.label).toBe('Landing');
  });
});

describe('normalizeHeaderConfig', () => {
  it('returns a default config when none is provided', () => {
    const cfg = normalizeHeaderConfig(null, 'Hero');
    expect(cfg.branding?.label).toBe('Hero');
    expect(cfg.nav?.links).toEqual([]);
  });

  it('derives the base name from the config branding label when no name given', () => {
    const cfg = normalizeHeaderConfig({ branding: { label: 'FromConfig' } } as never);
    expect(cfg.branding?.label).toBe('FromConfig');
  });

  it('fills nav-link defaults for id/type/label/visibility and preserves behavior', () => {
    const cfg = normalizeHeaderConfig({
      branding: { mode: 'name' },
      nav: {
        links: [
          // Missing id/type/label/visibleOn -> defaults applied.
          { href: '/x' },
          // Nested menu children normalized recursively.
          { type: 'menu', label: 'Menu', children: [{ href: '/child' }] },
        ],
      },
      behavior: { sticky: false },
    } as never);

    const [first, menu] = cfg.nav?.links ?? [];
    expect(first?.type).toBe('section');
    expect(first?.label).toBe('Section');
    expect(first?.id).toContain('nav-');
    expect(first?.visibleOn?.desktop).toBe(true);
    expect(first?.visibleOn?.mobile).toBe(true);
    expect(menu?.children?.[0]?.type).toBe('section');
    expect(cfg.behavior?.sticky).toBe(false);
    expect(cfg.behavior?.hideOnScroll).toBe(false);
  });

  it('merges CTA fallbacks with incoming overrides', () => {
    const cfg = normalizeHeaderConfig({
      ctas: { primary: { mode: 'custom', label: 'Buy', href: '/buy' } },
    } as never);
    expect(cfg.ctas?.primary?.mode).toBe('custom');
    // Secondary falls back to the default downloads CTA.
    expect(cfg.ctas?.secondary?.mode).toBe('downloads');
  });
});

describe('cloneHeaderConfig', () => {
  it('produces an equivalent normalized copy', () => {
    const original = buildDefaultHeaderConfig('Copy');
    const clone = cloneHeaderConfig(original);
    expect(clone.branding?.label).toBe('Copy');
    expect(clone).not.toBe(original);
  });
});
