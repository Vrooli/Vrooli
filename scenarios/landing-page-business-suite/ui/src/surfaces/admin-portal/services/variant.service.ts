// DOC: docs/reference/api/variants.md - A/B testing variant API
// DOC: docs/concepts/CONCEPTS.md#ab-testing-system - A/B testing architecture
// DOC: docs/concepts/CONCEPTS.md#section-architecture - Section structure
import type {
  ContentSection,
  LandingHeaderConfig,
  LandingHeaderNavLink,
  LandingSection,
} from '../../../shared/api';
import { DOWNLOAD_ANCHOR_ID, getSectionAnchorId } from '../../../shared/lib/sections';
import { isRecord, safeParseJson } from '../../../shared/lib/utils';

/**
 * Generate a unique ID for navigation links
 */
export function generateNavLinkId(prefix: string): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    try {
      return crypto.randomUUID();
    } catch {
      // ignore
    }
  }
  return `${prefix}-${String(Date.now())}-${String(Math.round(Math.random() * 1000))}`;
}

/**
 * Create a navigation link from a content section
 */
export function createNavLinkFromSection(section: ContentSection): LandingHeaderNavLink {
  const rawSectionId: unknown = section.id;
  const anchorSection = {
    id: section.id,
    section_type: section.section_type,
    order: section.order,
    content: section.content,
  } as LandingSection;

  return {
    id: generateNavLinkId(section.section_type),
    type: 'section',
    label: section.section_type.replace(/_/g, ' '),
    section_type: section.section_type,
    section_id: typeof rawSectionId === 'number' ? rawSectionId : undefined,
    anchor: getSectionAnchorId(anchorSection),
    visible_on: { desktop: true, mobile: true },
  };
}

/**
 * Create a downloads navigation link
 */
export function createDownloadsNavLink(): LandingHeaderNavLink {
  return {
    id: generateNavLinkId('downloads'),
    type: 'downloads',
    label: 'Downloads',
    anchor: DOWNLOAD_ANCHOR_ID,
    visible_on: { desktop: true, mobile: true },
  };
}

/**
 * Create a new menu navigation item
 */
export function createMenuNavLink(): LandingHeaderNavLink {
  return {
    id: generateNavLinkId('menu'),
    type: 'menu',
    label: 'Menu',
    visible_on: { desktop: true, mobile: true },
    children: [
      {
        id: generateNavLinkId('menu-item'),
        type: 'custom',
        label: 'First link',
        href: '#',
        visible_on: { desktop: true, mobile: true },
      },
      {
        id: generateNavLinkId('menu-item'),
        type: 'custom',
        label: 'Second link',
        href: '#',
        visible_on: { desktop: true, mobile: true },
      },
    ],
  };
}

/**
 * Create a child item for a menu
 */
export function createMenuChildLink(): LandingHeaderNavLink {
  return {
    id: generateNavLinkId('child'),
    type: 'custom',
    label: 'Menu item',
    href: '#',
    visible_on: { desktop: true, mobile: true },
  };
}

/**
 * Parse a navigation target JSON string
 */
export function parseNavTarget(
  targetJson: string
): { type: string; id?: number; section_type?: string; order?: number } | null {
  const parsed = safeParseJson(targetJson);
  if (!isRecord(parsed)) {
    return null;
  }
  const typeValue = parsed.type;
  if (typeof typeValue !== 'string' || !typeValue) {
    return null;
  }
  return {
    type: typeValue,
    id: typeof parsed.id === 'number' ? parsed.id : undefined,
    section_type: typeof parsed.section_type === 'string' ? parsed.section_type : undefined,
    order: typeof parsed.order === 'number' ? parsed.order : undefined,
  };
}

/**
 * Find section by parsed nav target
 */
export function findSectionByTarget(
  sections: ContentSection[],
  target: { id?: number; section_type?: string; order?: number }
): ContentSection | undefined {
  return sections.find((section) => {
    if (typeof target.id === 'number' && section.id) {
      return section.id === target.id;
    }
    return section.section_type === target.section_type && section.order === target.order;
  });
}

/**
 * Update header config with a deep clone to avoid mutations
 */
export function updateHeaderConfig(
  config: LandingHeaderConfig,
  updater: (draft: LandingHeaderConfig) => void
): LandingHeaderConfig {
  const clone = structuredClone(config);
  updater(clone);
  return clone;
}
