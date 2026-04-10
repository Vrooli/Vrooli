import * as React from "react";
import { ChevronRight } from "lucide-react";
import { cn } from "../../../lib/utils";
import { Button } from "../../ui/button";

interface DetailPanelProps {
  /** Panel title */
  title: string;
  /** Custom empty state component (shown when hasSelection is false) */
  empty?: React.ReactNode;
  /** Optional header actions (edit button, etc.) */
  headerActions?: React.ReactNode;
  /** Panel content */
  children: React.ReactNode;
  /** Whether an item is selected */
  hasSelection?: boolean;
  /** Whether panel is collapsed */
  collapsed?: boolean;
  /** Toggle collapse callback */
  onToggleCollapse?: () => void;
  /** Hide the built-in header (for custom headers in children) */
  hideHeader?: boolean;
  /** Additional CSS classes */
  className?: string;
}

export function DetailPanel({
  title,
  empty,
  headerActions,
  children,
  hasSelection = true,
  collapsed = false,
  onToggleCollapse,
  hideHeader = false,
  className,
}: DetailPanelProps) {
  return (
    <div
      className={cn(
        "h-full flex flex-col bg-card overflow-hidden",
        className
      )}
    >
      {/* Header */}
      {!hideHeader && (
        <div className="shrink-0 px-4 py-3 border-b border-border">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              {onToggleCollapse && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={onToggleCollapse}
                  aria-label={collapsed ? "Expand panel" : "Collapse panel"}
                >
                  <ChevronRight
                    className={cn(
                      "h-4 w-4 transition-transform",
                      collapsed && "rotate-180"
                    )}
                  />
                </Button>
              )}
              <span className="font-semibold text-sm">{title}</span>
            </div>
            {!collapsed && hasSelection && headerActions}
          </div>
        </div>
      )}

      {/* Content */}
      {!collapsed && (
        <div className="flex-1 min-h-0 overflow-y-auto">
          {!hasSelection ? (
            empty || (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <p className="text-sm">Select an item to view details</p>
              </div>
            )
          ) : (
            hideHeader ? (
              <div className="h-full">{children}</div>
            ) : (
              <div className="p-4">{children}</div>
            )
          )}
        </div>
      )}
    </div>
  );
}
