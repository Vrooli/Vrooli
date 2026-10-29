import * as React from "react";
import { ChevronLeft } from "lucide-react";
import { cn } from "../../../lib/utils";
import { Button } from "@vrooli/react-component-library/Button/2";
import { DataState } from "../../feedback/DataState";

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
  /** Load failure; shown instead of the empty state, or above stale rows. */
  error?: unknown;
  /** Retry the failed load. */
  onRetry?: () => void;
  /** Rendered only after a successful load that returned nothing. */
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
  loading = false,
  error,
  onRetry,
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
      <div
        className={cn(
          "shrink-0 border-b border-border",
          collapsed ? "px-2 py-3 flex items-center justify-center" : "px-4 py-3"
        )}
      >
        <div className={cn("flex items-center", collapsed ? "justify-center" : "justify-between")}>
          <div className="flex items-center gap-2">
            {onToggleCollapse && (
              <Button
                variant="ghost"
                size="icon"
                className="h-6 w-6"
                onClick={onToggleCollapse}
                aria-label={collapsed ? "Expand panel" : "Collapse panel"}
                title={collapsed ? `Expand ${title}` : `Collapse ${title}`}
              >
                <ChevronLeft
                  className={cn(
                    "h-4 w-4 transition-transform",
                    collapsed && "rotate-180"
                  )}
                />
              </Button>
            )}
            {!collapsed && (
              <span className="font-semibold text-sm">
                {title}{error && count === 0 ? "" : ` (${count})`}
              </span>
            )}
          </div>
          {!collapsed && headerActions}
        </div>
        {!collapsed && toolbar && <div className="mt-3">{toolbar}</div>}
      </div>

      {/* Content */}
      {!collapsed && (
        <div className="flex-1 min-h-0 overflow-y-auto">
          <DataState
            loading={loading}
            error={error}
            isEmpty={!hasItems || count === 0}
            subject={title}
            onRetry={onRetry}
            empty={empty ?? <p className="py-12 text-center text-sm text-muted-foreground">No items found</p>}
          >
            <div>{children}</div>
          </DataState>
        </div>
      )}
    </div>
  );
}
