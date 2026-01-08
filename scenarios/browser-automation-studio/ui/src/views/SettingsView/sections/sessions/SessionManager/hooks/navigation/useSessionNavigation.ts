import { useState, useCallback } from 'react';
import type { SectionId } from '../../SessionSidebar';

export interface UseSessionNavigationProps {
  /** Initial section to display when dialog opens */
  initialSection?: SectionId;
}

export interface UseSessionNavigationReturn {
  /** Currently active section */
  activeSection: SectionId;
  /** Set the active section */
  setActiveSection: (section: SectionId) => void;
}

/**
 * Manages navigation state for the session settings dialog.
 *
 * Responsibilities:
 * - Track which section is currently active
 */
export function useSessionNavigation({
  initialSection = 'presets',
}: UseSessionNavigationProps = {}): UseSessionNavigationReturn {
  const [activeSection, setActiveSectionState] = useState<SectionId>(initialSection);

  const setActiveSection = useCallback((section: SectionId) => {
    setActiveSectionState(section);
  }, []);

  return {
    activeSection,
    setActiveSection,
  };
}
