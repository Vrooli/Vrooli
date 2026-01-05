import * as React from "react";
import { cn } from "../../../lib/utils";
import { useViewportSize } from "../../../hooks/useViewportSize";
import { useResizablePanel } from "../../../hooks/useResizablePanel";
import { DetailModal } from "./DetailModal";
import { ResizableDivider } from "./ResizableDivider";

interface MasterDetailLayoutProps {
  /** The list panel component */
  listPanel: React.ReactNode;
  /** The detail panel component */
  detailPanel: React.ReactNode;
  /** Currently selected item ID (used for mobile modal) */
  selectedId: string | null;
  /** Callback to clear selection (used for mobile modal close) */
  onDeselect: () => void;
  /** Title for the detail modal on mobile */
  detailTitle: string;

  /** Page title displayed in header */
  pageTitle: string;
  /** Page subtitle displayed below title */
  pageSubtitle?: string;
  /** Optional header actions (buttons, etc.) to display next to title */
  headerActions?: React.ReactNode;
  /** Optional content to display below the header (banners, alerts, etc.) */
  headerContent?: React.ReactNode;

  /** Storage key for localStorage persistence (e.g., "runs", "profiles") */
  storageKey: string;
  /** Default list panel width as percentage (default: 40) */
  defaultListWidthPercent?: number;
  /** Minimum list panel width in pixels (default: 280) */
  minListWidth?: number;
  /** Minimum detail panel width in pixels (default: 320) */
  minDetailWidth?: number;

  /** Additional CSS classes */
  className?: string;
}

const DEFAULT_LIST_WIDTH_PERCENT = 40;
const DEFAULT_MIN_LIST_WIDTH = 280;
const DEFAULT_MIN_DETAIL_WIDTH = 320;

export function MasterDetailLayout({
  listPanel,
  detailPanel,
  selectedId,
  onDeselect,
  detailTitle,
  pageTitle,
  pageSubtitle,
  headerActions,
  headerContent,
  storageKey,
  defaultListWidthPercent = DEFAULT_LIST_WIDTH_PERCENT,
  minListWidth = DEFAULT_MIN_LIST_WIDTH,
  minDetailWidth = DEFAULT_MIN_DETAIL_WIDTH,
  className,
}: MasterDetailLayoutProps) {
  const { isDesktop } = useViewportSize();

  // Calculate default width based on a reference container width
  // This will be adjusted when the container is measured
  const defaultWidth = Math.round((800 * defaultListWidthPercent) / 100);

  const { width: listWidth, isResizing, handleResizeStart, containerRef } = useResizablePanel({
    storageKey: `${storageKey}.list`,
    defaultWidth,
    minWidth: minListWidth,
    minOtherWidth: minDetailWidth,
  });

  // Desktop layout
  if (isDesktop) {
    return (
      <div className={cn("h-full flex flex-col overflow-hidden", className)}>
        {/* Page header - fixed height */}
        <div className="shrink-0 px-4 py-4 sm:px-6 lg:px-10 space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold">{pageTitle}</h2>
              {pageSubtitle && (
                <p className="text-sm text-muted-foreground">{pageSubtitle}</p>
              )}
            </div>
            {headerActions}
          </div>
          {headerContent}
        </div>

        {/* Panel area - fills remaining height */}
        <div className="flex-1 min-h-0 overflow-hidden px-4 sm:px-6 lg:px-10 pb-4">
          <div
            ref={containerRef}
            className="h-full flex rounded-lg border bg-card overflow-hidden"
          >
            {/* List panel */}
            <div
              style={{ width: listWidth }}
              className="shrink-0 min-h-0 overflow-hidden border-r"
            >
              {listPanel}
            </div>

            {/* Resize divider */}
            <ResizableDivider
              onMouseDown={handleResizeStart}
              isResizing={isResizing}
            />

            {/* Detail panel */}
            <div className="flex-1 min-w-0 min-h-0 overflow-hidden">
              {detailPanel}
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Mobile layout - list with modal for details
  return (
    <div className={cn("h-full flex flex-col overflow-hidden", className)}>
      {/* Page header - fixed height */}
      <div className="shrink-0 px-4 py-4 space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold">{pageTitle}</h2>
            {pageSubtitle && (
              <p className="text-sm text-muted-foreground">{pageSubtitle}</p>
            )}
          </div>
          {headerActions}
        </div>
        {headerContent}
      </div>

      {/* List panel - fills remaining height */}
      <div className="flex-1 min-h-0 overflow-hidden px-4 pb-4">
        <div className="h-full rounded-lg border bg-card overflow-hidden">
          {listPanel}
        </div>
      </div>

      {/* Mobile detail modal */}
      <DetailModal open={!!selectedId} onClose={onDeselect} title={detailTitle}>
        {detailPanel}
      </DetailModal>
    </div>
  );
}
