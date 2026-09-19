/**
 * Two-column layout for the Generator page.
 *
 * Desktop: collapsible sidebar + scrollable main content (side-by-side).
 * Mobile (<768 px): sidebar becomes a slide-out drawer triggered by a
 * floating button; main content fills the viewport.
 *
 * Seam: layout ↔ viewport — the useIsMobile hook controls which layout
 * variant renders, so tests can assert both paths independently.
 *
 * DOC: docs/internal/SEAMS.md#generator-layout-seam
 */

import { useRef, useCallback, type ReactNode, type RefObject } from "react";
import { X } from "lucide-react";
import { PipelineSidebar } from "./PipelineSidebar";
import { MobilePipelineSummary } from "./MobilePipelineSummary";
import { useSidebarStore, type SectionId } from "../../store/sidebarStore";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { cn } from "../../lib/utils";

interface GeneratorLayoutProps {
  /** Refs to each section for scroll-to functionality */
  sectionRefs: Record<SectionId, RefObject<HTMLDivElement>>;
  /** Main content to render */
  children: ReactNode;
}

export function GeneratorLayout({
  sectionRefs,
  children,
}: GeneratorLayoutProps) {
  const setActiveSection = useSidebarStore((s) => s.setActiveSection);
  const mobileDrawerOpen = useSidebarStore((s) => s.mobileDrawerOpen);
  const setMobileDrawerOpen = useSidebarStore((s) => s.setMobileDrawerOpen);
  const mainContentRef = useRef<HTMLDivElement>(null);
  const isMobile = useIsMobile();

  const handleSectionClick = useCallback(
    (section: SectionId) => {
      setActiveSection(section);
      // Close drawer on mobile after navigation
      if (isMobile) setMobileDrawerOpen(false);
      const ref = sectionRefs[section];
      const container = mainContentRef.current;
      if (ref.current && container) {
        container.scrollTo({
          top: ref.current.offsetTop - container.offsetTop,
          behavior: "smooth",
        });
      }
    },
    [setActiveSection, sectionRefs, isMobile, setMobileDrawerOpen],
  );

  /* ── Mobile layout ─────────────────────────────────────────────── */
  if (isMobile) {
    return (
      <div className="relative min-h-full">
        {/* Compact pipeline summary bar */}
        <MobilePipelineSummary
          onOpenDrawer={() => {
            setMobileDrawerOpen(true);
          }}
        />

        {/* Drawer overlay */}
        {mobileDrawerOpen && (
          <div
            className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm"
            onClick={() => {
              setMobileDrawerOpen(false);
            }}
            aria-hidden="true"
          />
        )}

        {/* Slide-out drawer */}
        <aside
          className={cn(
            "fixed inset-y-0 left-0 z-50 w-72 transform transition-transform duration-300 ease-in-out bg-slate-950 shadow-2xl",
            mobileDrawerOpen ? "translate-x-0" : "-translate-x-full",
          )}
        >
          <div className="flex items-center justify-end p-2 border-b border-white/10">
            <button
              type="button"
              onClick={() => {
                setMobileDrawerOpen(false);
              }}
              className="rounded-lg p-2 text-slate-400 hover:text-slate-200 hover:bg-white/5"
              aria-label="Close pipeline sidebar"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
          <div className="overflow-y-auto h-[calc(100%-3rem)]">
            <PipelineSidebar onSectionClick={handleSectionClick} />
          </div>
        </aside>

        {/* Main content — full width on mobile */}
        <main ref={mainContentRef} className="overflow-y-auto">
          <div className="mx-auto max-w-4xl space-y-3 p-2">{children}</div>
        </main>
      </div>
    );
  }

  /* ── Desktop layout ────────────────────────────────────────────── */
  return (
    <div className="flex min-h-full">
      {/* Sidebar - sticky so it stays visible while scrolling */}
      <div className="sticky top-0 h-full">
        <PipelineSidebar onSectionClick={handleSectionClick} />
      </div>

      {/* Main Content - scrollable */}
      <main
        ref={mainContentRef}
        className="flex-1 overflow-y-auto transition-all duration-300"
      >
        <div className="mx-auto max-w-4xl space-y-6 p-6">{children}</div>
      </main>
    </div>
  );
}
