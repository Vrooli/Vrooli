import type { ReactNode } from 'react';

/** Section identifiers for sidebar navigation */
export type SectionId = 'stream' | 'visual' | 'cursor' | 'playback' | 'branding';

/** Section group identifiers */
export type SectionGroupId = 'stream' | 'replay';

/** Configuration for a single sidebar section */
export interface SectionConfig {
  id: SectionId;
  label: string;
  icon: ReactNode;
}

/** Configuration for a section group */
export interface SectionGroup {
  id: SectionGroupId;
  label: string;
  sections: SectionConfig[];
}

/** Props passed to all section components for consistent interface */
export interface BaseSectionProps {
  /** Visual variant for section rendering */
  variant?: 'settings' | 'compact';
}
