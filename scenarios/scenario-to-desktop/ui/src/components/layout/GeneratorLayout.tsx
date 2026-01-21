/**
 * Two-column layout for the Generator page.
 * Features a collapsible sidebar with independent scrolling for both columns.
 */

import { useRef, useCallback, type ReactNode, type RefObject } from "react";
import { PipelineSidebar } from "./PipelineSidebar";
import { useSidebarStore, type SectionId } from "../../store/sidebarStore";

interface GeneratorLayoutProps {
  /** Refs to each section for scroll-to functionality */
  sectionRefs: Record<SectionId, RefObject<HTMLDivElement>>;
  /** Main content to render */
  children: ReactNode;
}

export function GeneratorLayout({ sectionRefs, children }: GeneratorLayoutProps) {
  const setActiveSection = useSidebarStore((s) => s.setActiveSection);
  const mainContentRef = useRef<HTMLDivElement>(null);

  const handleSectionClick = useCallback(
    (section: SectionId) => {
      setActiveSection(section);
      const ref = sectionRefs[section];
      if (ref?.current) {
        ref.current.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    },
    [setActiveSection, sectionRefs]
  );

  return (
    <div className="flex min-h-screen">
      {/* Sidebar - sticky so it stays visible while scrolling */}
      <div className="sticky top-0 h-screen">
        <PipelineSidebar onSectionClick={handleSectionClick} />
      </div>

      {/* Main Content - scrollable */}
      <main
        ref={mainContentRef}
        className="flex-1 overflow-y-auto transition-all duration-300"
      >
        <div className="mx-auto max-w-4xl space-y-6 p-6">
          {children}
        </div>
      </main>
    </div>
  );
}
