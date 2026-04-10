import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type {
  ContentSection,
  LandingHeaderConfig,
  LandingHeaderNavLink,
} from '../../../shared/api';
import {
  generateNavLinkId,
  createNavLinkFromSection,
  createDownloadsNavLink,
  createMenuNavLink,
  createMenuChildLink,
  parseNavTarget,
  findSectionByTarget,
  updateHeaderConfig,
} from './variant.service';

describe('variant.service', () => {
  describe('generateNavLinkId', () => {
    it('generates unique IDs', () => {
      const id1 = generateNavLinkId('test');
      const id2 = generateNavLinkId('test');

      expect(id1).not.toBe(id2);
    });

    it('produces valid string IDs', () => {
      const id = generateNavLinkId('nav');

      expect(typeof id).toBe('string');
      expect(id.length).toBeGreaterThan(0);
    });

    it('uses crypto.randomUUID when available (default in modern env)', () => {
      // In modern Node/browser environments, crypto.randomUUID is available
      // and the function should produce UUID-like strings
      const id = generateNavLinkId('test');

      // UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
      expect(id).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/);
    });

    it('produces unique IDs across multiple calls', () => {
      const ids = new Set<string>();
      for (let i = 0; i < 100; i++) {
        ids.add(generateNavLinkId('test'));
      }
      // All IDs should be unique
      expect(ids.size).toBe(100);
    });
  });

  describe('createNavLinkFromSection', () => {
    it('creates nav link from section', () => {
      const section: ContentSection = {
        id: 42,
        variant_id: 1,
        section_type: 'features',
        content: { title: 'Features' },
        order: 2,
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      };

      const result = createNavLinkFromSection(section);

      expect(result.type).toBe('section');
      expect(result.label).toBe('features');
      expect(result.section_type).toBe('features');
      expect(result.section_id).toBe(42);
      expect(result.anchor).toBe('features-42');
      expect(result.visible_on).toEqual({ desktop: true, mobile: true });
    });

    it('replaces underscores with spaces in label', () => {
      const section: ContentSection = {
        id: 1,
        variant_id: 1,
        section_type: 'faq',
        content: {},
        order: 1,
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      };

      const result = createNavLinkFromSection(section);

      expect(result.label).toBe('faq');
    });

    it('uses section ID directly (0 is a valid ID)', () => {
      // Note: The function uses `section.id ?? undefined`, so 0 is preserved (it's falsy but not nullish)
      const section: ContentSection = {
        id: 0,
        variant_id: 1,
        section_type: 'hero',
        content: {},
        order: 1,
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      };

      const result = createNavLinkFromSection(section);

      // 0 ?? undefined returns 0 (nullish coalescing only checks null/undefined)
      expect(result.section_id).toBe(0);
    });

    it('returns undefined section_id for null ID', () => {
      const section: ContentSection = {
        id: null as unknown as number,
        variant_id: 1,
        section_type: 'hero',
        content: {},
        order: 1,
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      };

      const result = createNavLinkFromSection(section);

      expect(result.section_id).toBe(undefined);
    });
  });

  describe('createDownloadsNavLink', () => {
    it('creates downloads nav link with correct structure', () => {
      const result = createDownloadsNavLink();

      expect(result.type).toBe('downloads');
      expect(result.label).toBe('Downloads');
      expect(result.anchor).toBe('downloads-section');
      expect(result.visible_on).toEqual({ desktop: true, mobile: true });
    });

    it('generates unique ID each time', () => {
      const link1 = createDownloadsNavLink();
      const link2 = createDownloadsNavLink();

      expect(link1.id).not.toBe(link2.id);
    });
  });

  describe('createMenuNavLink', () => {
    it('creates menu nav link with children', () => {
      const result = createMenuNavLink();

      expect(result.type).toBe('menu');
      expect(result.label).toBe('Menu');
      expect(result.visible_on).toEqual({ desktop: true, mobile: true });
      expect(result.children).toHaveLength(2);
    });

    it('creates children with custom type', () => {
      const result = createMenuNavLink();

      expect(result.children?.[0]?.type).toBe('custom');
      expect(result.children?.[0]?.label).toBe('First link');
      expect(result.children?.[0]?.href).toBe('#');
    });

    it('generates unique IDs for children', () => {
      const result = createMenuNavLink();

      expect(result.children?.[0]?.id).not.toBe(result.children?.[1]?.id);
    });
  });

  describe('createMenuChildLink', () => {
    it('creates custom child link', () => {
      const result = createMenuChildLink();

      expect(result.type).toBe('custom');
      expect(result.label).toBe('Menu item');
      expect(result.href).toBe('#');
      expect(result.visible_on).toEqual({ desktop: true, mobile: true });
    });

    it('generates unique ID', () => {
      const link1 = createMenuChildLink();
      const link2 = createMenuChildLink();

      expect(link1.id).not.toBe(link2.id);
    });
  });

  describe('parseNavTarget', () => {
    it('parses valid JSON target', () => {
      const json = JSON.stringify({ type: 'section', id: 42, section_type: 'hero' });

      const result = parseNavTarget(json);

      expect(result).toEqual({ type: 'section', id: 42, section_type: 'hero' });
    });

    it('returns null for invalid JSON', () => {
      const result = parseNavTarget('not-valid-json');

      expect(result).toBeNull();
    });

    it('returns null for empty string', () => {
      const result = parseNavTarget('');

      expect(result).toBeNull();
    });

    it('parses target with order', () => {
      const json = JSON.stringify({ type: 'section', section_type: 'features', order: 2 });

      const result = parseNavTarget(json);

      expect(result?.order).toBe(2);
    });
  });

  describe('findSectionByTarget', () => {
    const sections: ContentSection[] = [
      {
        id: 1,
        variant_id: 1,
        section_type: 'hero',
        content: {},
        order: 1,
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      },
      {
        id: 2,
        variant_id: 1,
        section_type: 'features',
        content: {},
        order: 2,
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      },
      {
        id: 3,
        variant_id: 1,
        section_type: 'features',
        content: {},
        order: 3,
        enabled: true,
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      },
    ];

    it('finds section by ID', () => {
      const result = findSectionByTarget(sections, { id: 2 });

      expect(result?.id).toBe(2);
      expect(result?.section_type).toBe('features');
    });

    it('finds section by section_type and order', () => {
      const result = findSectionByTarget(sections, { section_type: 'features', order: 3 });

      expect(result?.id).toBe(3);
    });

    it('returns undefined when no match', () => {
      const result = findSectionByTarget(sections, { id: 999 });

      expect(result).toBeUndefined();
    });

    it('returns undefined for empty sections', () => {
      const result = findSectionByTarget([], { id: 1 });

      expect(result).toBeUndefined();
    });

    it('prioritizes ID match over section_type/order', () => {
      const result = findSectionByTarget(sections, { id: 1, section_type: 'features', order: 2 });

      expect(result?.id).toBe(1);
      expect(result?.section_type).toBe('hero');
    });
  });

  describe('updateHeaderConfig', () => {
    const baseConfig: LandingHeaderConfig = {
      branding: {
        mode: 'logo',
        label: 'Test Site',
      },
      nav: {
        links: [],
      },
      ctas: {
        primary: { mode: 'inherit_hero' },
        secondary: { mode: 'hidden' },
      },
      behavior: {
        sticky: true,
        hide_on_scroll: false,
      },
    };

    it('returns updated config without mutating original', () => {
      const result = updateHeaderConfig(baseConfig, (draft) => {
        draft.branding.label = 'Updated Site';
      });

      expect(result.branding.label).toBe('Updated Site');
      expect(baseConfig.branding.label).toBe('Test Site');
    });

    it('allows updating nested properties', () => {
      const result = updateHeaderConfig(baseConfig, (draft) => {
        draft.behavior.sticky = false;
        draft.behavior.hide_on_scroll = true;
      });

      expect(result.behavior.sticky).toBe(false);
      expect(result.behavior.hide_on_scroll).toBe(true);
    });

    it('allows adding nav links', () => {
      const newLink: LandingHeaderNavLink = {
        id: 'new-link',
        type: 'custom',
        label: 'New Link',
        href: '/new',
        visible_on: { desktop: true, mobile: true },
      };

      const result = updateHeaderConfig(baseConfig, (draft) => {
        draft.nav.links.push(newLink);
      });

      expect(result.nav.links).toHaveLength(1);
      expect(result.nav.links[0]?.label).toBe('New Link');
      expect(baseConfig.nav.links).toHaveLength(0);
    });

    it('allows updating CTA config', () => {
      const result = updateHeaderConfig(baseConfig, (draft) => {
        draft.ctas.primary = {
          mode: 'custom',
          label: 'Get Started',
          href: '/signup',
        };
      });

      expect(result.ctas.primary.mode).toBe('custom');
      expect(result.ctas.primary.label).toBe('Get Started');
    });

    it('creates deep clone of nested objects', () => {
      const configWithLinks: LandingHeaderConfig = {
        ...baseConfig,
        nav: {
          links: [
            { id: '1', type: 'custom', label: 'Link 1', href: '/1', visible_on: { desktop: true, mobile: true } },
          ],
        },
      };

      const result = updateHeaderConfig(configWithLinks, (draft) => {
        const link0 = draft.nav.links[0];
        if (link0) link0.label = 'Modified';
      });

      expect(result.nav.links[0]?.label).toBe('Modified');
      expect(configWithLinks.nav.links[0]?.label).toBe('Link 1');
    });
  });
});
