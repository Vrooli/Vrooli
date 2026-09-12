/**
 * DetailPageLayout
 *
 * Full-page overlay wrapper for entity detail pages. Renders on top of
 * the graph workspace (which stays mounted underneath). Handles:
 * - Full-screen overlay with scroll
 * - Sticky header on mobile
 * - Consistent background and spacing
 *
 * Actions live in the DetailPageHeader (primary button + overflow menu,
 * which becomes a bottom sheet on mobile) — there is no separate mobile FAB.
 */

import { type ReactNode } from "react";
import { cn } from "../../lib/utils";

export interface DetailPageLayoutProps {
  /** The DetailPageHeader component. */
  header: ReactNode;
  /** Page body content. */
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
  /**
   * Edge-to-edge body: drops the body gutters and pins the page to the
   * available height so the child owns its own scrolling. Tabs that render a
   * self-contained workspace (Files) use this so their header sits flush under
   * the tab bar instead of floating inside the page gutters.
   */
  fullBleed?: boolean;
}

export function DetailPageLayout({
  header,
  children,
  className,
  bodyClassName,
  fullBleed = false,
}: DetailPageLayoutProps) {
  return (
    <div
      className={cn(
        "flex flex-col bg-slate-950 text-slate-50",
        fullBleed ? "h-full overflow-hidden" : "min-h-full",
        className,
      )}
      data-testid="detail-page-layout"
      data-full-bleed={fullBleed ? "true" : undefined}
    >
      {/* Sticky header on mobile, static on desktop */}
      <div className="sticky top-0 z-30 shrink-0 bg-slate-950/95 backdrop-blur">
        {header}
      </div>

      {/* Page body */}
      <div
        className={cn(
          "flex-1",
          fullBleed ? "min-h-0 overflow-hidden" : "px-2 py-3 md:px-6 md:py-6",
          bodyClassName,
        )}
      >
        {children}
      </div>
    </div>
  );
}
