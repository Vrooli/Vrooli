import { ChevronDown, ChevronRight, GitCommit, Loader2 } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";

interface GitHistoryProps {
  lines?: string[];
  isLoading: boolean;
  error?: Error | null;
  collapsed?: boolean;
  onToggleCollapse?: () => void;
  height?: number;
}

export function GitHistory({
  lines = [],
  isLoading,
  error,
  collapsed = false,
  onToggleCollapse,
  height = 200
}: GitHistoryProps) {
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const hasLines = lines.length > 0;

  return (
    <Card
      className="flex flex-col min-w-0"
      style={{ height: collapsed ? "auto" : height }}
      data-testid="git-history-panel"
    >
      <CardHeader className="flex-row items-center justify-between space-y-0 py-3 gap-2 min-w-0">
        <CardTitle className="flex items-center gap-2 min-w-0">
          <button
            className="p-1 rounded hover:bg-slate-800/70 transition-colors"
            onClick={handleToggleCollapse}
            aria-label={collapsed ? "Expand history" : "Collapse history"}
            type="button"
          >
            {collapsed ? (
              <ChevronRight className="h-3 w-3 text-slate-400" />
            ) : (
              <ChevronDown className="h-3 w-3 text-slate-400" />
            )}
          </button>
          <GitCommit className="h-3.5 w-3.5 text-slate-400" />
          <span className="truncate">History</span>
        </CardTitle>
        <span className="text-xs text-slate-600 ml-auto">{lines.length}</span>
      </CardHeader>

      {!collapsed && (
        <CardContent className="flex-1 min-w-0 p-0 overflow-hidden">
          <ScrollArea className="h-full min-w-0 px-2 py-2">
            {isLoading && (
              <div className="flex items-center justify-center py-6 text-slate-500 text-sm">
                <Loader2 className="h-4 w-4 animate-spin mr-2 text-slate-500" />
                Loading history...
              </div>
            )}
            {!isLoading && error && (
              <div className="flex flex-col items-center justify-center py-6 text-center">
                <p className="text-sm text-red-400">History unavailable</p>
                <p className="text-xs text-red-500/80 mt-1">{error.message}</p>
              </div>
            )}
            {!isLoading && !error && !hasLines && (
              <div className="flex flex-col items-center justify-center py-6 text-center">
                <p className="text-sm text-slate-500">No commits yet</p>
                <p className="text-xs text-slate-600 mt-1">
                  Start by creating your first commit
                </p>
              </div>
            )}
            {!isLoading && !error && hasLines && (
              <div className="space-y-1 font-mono text-xs text-slate-200 whitespace-pre">
                {lines.map((line, index) => (
                  <div
                    key={`${line}-${index}`}
                    className="rounded-md border border-slate-800/60 bg-slate-900/40 px-2 py-1 text-slate-200 hover:bg-slate-800/60 transition-colors"
                  >
                    {line}
                  </div>
                ))}
              </div>
            )}
          </ScrollArea>
        </CardContent>
      )}
    </Card>
  );
}
