import type { LandingSection } from '../api';

export const DOWNLOAD_ANCHOR_ID = 'downloads-section';

export function getSectionKey(section: LandingSection) {
  return section.id ?? `${section.section_type}-${section.order}`;
}

export function getSectionAnchorId(section: LandingSection) {
  if (section.section_type === 'downloads') {
    return DOWNLOAD_ANCHOR_ID;
  }
  const base = section.section_type.replace(/_/g, '-');
  const suffix = section.id ?? section.order;
  return `${base}-${suffix}`;
}
