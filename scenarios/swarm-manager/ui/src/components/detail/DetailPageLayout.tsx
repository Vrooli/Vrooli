/**
 * DetailPageLayout
 *
 * Full-page overlay wrapper for entity detail pages. Renders on top of
 * the graph workspace (which stays mounted underneath). Handles:
 * - Full-screen overlay with scroll
 * - Sticky header on mobile
 * - Mobile action BottomSheet + FAB integration
 * - Consistent background and spacing
 *
 * DOC: docs/plans/navigation-header-unification-plan.md#phase-2
 */

import { type ReactNode, useState } from "react";
import { MoreVertical } from "lucide-react";
import { cn } from "../../lib/utils";
import { BottomSheet } from "../ui/bottom-sheet";
import { useIsMobile } from "../../hooks/useMediaQuery";

export interface DetailPageLayoutProps {
  /** The DetailPageHeader component. */
  header: ReactNode;
  /** Page body content. */
  children: ReactNode;
  /** Content for mobile action BottomSheet. If provided, a FAB appears on mobile. */
  mobileActions?: ReactNode;
  /** Title for the mobile actions sheet. */
  mobileActionsTitle?: string;
  className?: string;
}

export function DetailPageLayout({
  header,
  children,
  mobileActions,
  mobileActionsTitle = "Actions",
  className,
}: DetailPageLayoutProps) {
  const isMobile = useIsMobile();
  const [showActionsSheet, setShowActionsSheet] = useState(false);

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
      <div className="flex-1 px-4 py-6 md:px-6">
        {children}
      </div>

      {/* Mobile actions FAB */}
      {isMobile && mobileActions && (
        <>
          <button
            type="button"
            onClick={() => setShowActionsSheet(true)}
            className="fixed bottom-6 right-6 z-30 flex h-12 w-12 items-center justify-center rounded-full bg-cyan-600 text-white shadow-lg transition-colors hover:bg-cyan-500"
            aria-label="Open actions"
            data-testid="detail-mobile-actions-fab"
          >
            <MoreVertical className="h-5 w-5" />
          </button>

          <BottomSheet
            isOpen={showActionsSheet}
            onClose={() => setShowActionsSheet(false)}
            title={mobileActionsTitle}
            data-testid="detail-mobile-actions-sheet"
          >
            {mobileActions}
          </BottomSheet>
        </>
      )}
    </div>
  );
}
