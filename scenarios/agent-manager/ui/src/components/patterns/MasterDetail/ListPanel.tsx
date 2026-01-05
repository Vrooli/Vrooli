import * as React from "react";
import { ChevronLeft } from "lucide-react";
import { cn } from "../../../lib/utils";
import { Button } from "../../ui/button";

interface ListPanelProps {
  /** Panel title */
  title: string;
  /** Item count to display next to title */
  count: number;
  /** Optional toolbar (search, filters, etc.) */
  toolbar?: React.ReactNode;
  /** Optional header actions (refresh button, etc.) */
  headerActions?: React.ReactNode;
  /** Whether content is loading */
  loading?: boolean;
  /** Custom empty state component */
  empty?: React.ReactNode;
  /** List item children */
  children: React.ReactNode;
  /** Whether panel is collapsed */
  collapsed?: boolean;
  /** Toggle collapse callback */
  onToggleCollapse?: () => void;
  /** Additional CSS classes */
  className?: string;
}

export function ListPanel({
  title,
  count,
  toolbar,
  headerActions,
  loading,
  empty,
  children,
  collapsed = false,
  onToggleCollapse,
  className,
}: ListPanelProps) {
  const hasItems = React.Children.count(children) > 0;

  return (
    <div
      className={cn(
        "h-full flex flex-col bg-card overflow-hidden",
        className
      )}
    >
      {/* Header */}
      <div className="shrink-0 px-4 py-3 border-b">
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
                <ChevronLeft
                  className={cn(
                    "h-4 w-4 transition-transform",
                    collapsed && "rotate-180"
                  )}
                />
              </Button>
            )}
            <span className="font-semibold text-sm">
              {title} ({count})
            </span>
          </div>
          {!collapsed && headerActions}
        </div>
        {!collapsed && toolbar && <div className="mt-3">{toolbar}</div>}
      </div>

      {/* Content */}
      {!collapsed && (
        <div className="flex-1 min-h-0 overflow-y-auto">
          {loading ? (
            <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
              <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
              <p className="mt-3 text-sm">Loading...</p>
            </div>
          ) : !hasItems || count === 0 ? (
            empty || (
              <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                <p className="text-sm">No items found</p>
              </div>
            )
          ) : (
            <div>{children}</div>
          )}
        </div>
      )}
    </div>
  );
}
