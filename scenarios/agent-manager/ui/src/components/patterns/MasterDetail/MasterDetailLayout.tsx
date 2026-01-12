import * as React from "react";
import { cn } from "../../../lib/utils";
import { useViewportSize } from "../../../hooks/useViewportSize";
import { useResizablePanel } from "../../../hooks/useResizablePanel";
import { useCollapsiblePanel } from "../../../hooks/useCollapsiblePanel";
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

  /** Optional content to display at top (error banners, alerts, etc.) */
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
const COLLAPSED_LIST_WIDTH = 48;

export function MasterDetailLayout({
  listPanel,
  detailPanel,
  selectedId,
  onDeselect,
  detailTitle,
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

  const { isCollapsed, toggle: toggleCollapse } = useCollapsiblePanel({
    storageKey: `${storageKey}.list`,
    defaultCollapsed: false,
  });

  // Clone listPanel to inject collapse props
  const listPanelWithCollapse = React.isValidElement(listPanel)
    ? React.cloneElement(listPanel as React.ReactElement<{ collapsed?: boolean; onToggleCollapse?: () => void }>, {
        collapsed: isCollapsed,
        onToggleCollapse: toggleCollapse,
      })
    : listPanel;

  // Use collapsed width when collapsed, otherwise resizable width
  const effectiveListWidth = isCollapsed ? COLLAPSED_LIST_WIDTH : listWidth;

  // Desktop layout
  if (isDesktop) {
    return (
      <div className={cn("h-full flex flex-col overflow-hidden", className)}>
        {/* Header content (error banners, etc.) - only when present */}
        {headerContent && (
          <div className="shrink-0 px-4 py-2 sm:px-6 lg:px-10">
            {headerContent}
          </div>
        )}

        {/* Panel area - fills entire height */}
        <div
          ref={containerRef}
          className="flex-1 min-h-0 flex bg-card overflow-hidden"
        >
          {/* List panel */}
          <div
            style={{ width: effectiveListWidth }}
            className={cn(
              "shrink-0 min-h-0 overflow-hidden border-r border-border",
              "transition-[width] duration-200 ease-in-out"
            )}
          >
            {listPanelWithCollapse}
          </div>

          {/* Resize divider - hidden when collapsed */}
          {!isCollapsed && (
            <ResizableDivider
              onMouseDown={handleResizeStart}
              isResizing={isResizing}
            />
          )}

          {/* Detail panel */}
          <div className="flex-1 min-w-0 min-h-0 overflow-hidden">
            {detailPanel}
          </div>
        </div>
      </div>
    );
  }

  // Mobile layout - list with modal for details
  return (
    <div className={cn("h-full flex flex-col overflow-hidden", className)}>
      {/* Header content (error banners, etc.) - only when present */}
      {headerContent && (
        <div className="shrink-0 px-4 py-2">
          {headerContent}
        </div>
      )}

      {/* List panel - fills entire height */}
      <div className="flex-1 min-h-0 overflow-hidden bg-card">
        {listPanel}
      </div>

      {/* Mobile detail modal */}
      <DetailModal open={!!selectedId} onClose={onDeselect} title={detailTitle}>
        {detailPanel}
      </DetailModal>
    </div>
  );
}
