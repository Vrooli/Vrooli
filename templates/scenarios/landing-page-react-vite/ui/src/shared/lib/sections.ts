import type { LandingSection } from '../api';

export const DOWNLOAD_ANCHOR_ID = 'downloads-section';

export function getSectionKey(section: LandingSection) {
  return `${section.sectionType}-${section.order}`;
}

export function getSectionAnchorId(section: LandingSection) {
  if (section.sectionType === 'downloads') {
    return DOWNLOAD_ANCHOR_ID;
  }
  const base = section.sectionType.replace(/_/g, '-');
  return `${base}-${section.order}`;
}
