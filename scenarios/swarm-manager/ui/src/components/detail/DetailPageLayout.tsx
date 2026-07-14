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
}

export function DetailPageLayout({
  header,
  children,
  className,
  bodyClassName,
}: DetailPageLayoutProps) {
  return (
    <div
      className={cn("flex min-h-full flex-col bg-slate-950 text-slate-50", className)}
      data-testid="detail-page-layout"
    >
      {/* Sticky header on mobile, static on desktop */}
      <div className="sticky top-0 z-30 bg-slate-950/95 backdrop-blur">
        {header}
      </div>

      {/* Page body */}
      <div className={cn("flex-1 px-2 py-3 md:px-6 md:py-6", bodyClassName)}>
        {children}
      </div>
    </div>
  );
}
