import { ChevronDown, ChevronRight, GitCommit, Loader2 } from "lucide-react";
import { useMemo } from "react";
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

type HistoryEntry = {
  raw: string;
  graph: string;
  hash?: string;
  message: string;
  decorations: string[];
  headBranch?: string;
  remoteBranches: string[];
  isHead: boolean;
  isRemote: boolean;
  isFirstRemote: boolean;
  isPushed: boolean;
};

function parseHistoryLine(line: string) {
  const hashMatch = line.match(/[0-9a-f]{7,}/);
  if (!hashMatch || hashMatch.index === undefined) {
    return {
      raw: line,
      graph: line,
      hash: undefined,
      message: line.trim(),
      decorations: [] as string[],
      headBranch: undefined,
      remoteBranches: [] as string[],
      isHead: false,
      isRemote: false
    };
  }

  const hashIndex = hashMatch.index;
  const hash = hashMatch[0];
  const graph = line.slice(0, hashIndex).trimEnd();
  let rest = line.slice(hashIndex + hash.length).trim();
  let decorationText = "";
  if (rest.startsWith("(")) {
    const closeIndex = rest.indexOf(")");
    if (closeIndex >= 0) {
      decorationText = rest.slice(1, closeIndex);
      rest = rest.slice(closeIndex + 1).trim();
    }
  }

  const decorations = decorationText
    ? decorationText.split(",").map((token) => token.trim())
    : [];
  const headToken = decorations.find((token) => token.startsWith("HEAD -> "));
  const headBranch = headToken ? headToken.replace("HEAD -> ", "").trim() : undefined;
  const remoteBranches = decorations.filter(
    (token) => token.startsWith("origin/") || token.startsWith("upstream/")
  );

  return {
    raw: line,
    graph,
    hash,
    message: rest,
    decorations,
    headBranch,
    remoteBranches,
    isHead: Boolean(headToken || decorations.includes("HEAD")),
    isRemote: remoteBranches.length > 0
  };
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
  const entries = useMemo(() => {
    let remoteSeen = false;
    let firstRemoteIndex = -1;
    const parsed = lines.map((line) => parseHistoryLine(line));
    parsed.forEach((entry, index) => {
      if (entry.isRemote && firstRemoteIndex < 0) {
        firstRemoteIndex = index;
      }
    });

    return parsed.map((entry, index) => {
      if (entry.isRemote && !remoteSeen) {
        remoteSeen = true;
      }
      const isPushed = remoteSeen;
      return {
        ...entry,
        isPushed,
        isFirstRemote: index === firstRemoteIndex
      } as HistoryEntry;
    });
  }, [lines]);

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
              <div className="space-y-1 font-mono text-xs text-slate-200">
                {entries.map((entry, index) => {
                  const isUnpushed = entry.hash && !entry.isPushed;
                  const nodeTone = isUnpushed ? "bg-amber-400" : "bg-emerald-400";
                  const lineTone = isUnpushed ? "bg-amber-500/40" : "bg-emerald-500/40";
                  const chipTone = isUnpushed
                    ? "border-amber-500/40 text-amber-200 bg-amber-500/10"
                    : "border-emerald-500/40 text-emerald-200 bg-emerald-500/10";
                  const showHeadBadge = Boolean(
                    entry.headBranch && entry.isHead && entry.remoteBranches.length === 0
                  );
                  const showRemoteBadge = entry.isFirstRemote && entry.remoteBranches.length > 0;

                  return (
                    <div
                      key={`${entry.raw}-${index}`}
                      className="relative rounded-md border border-slate-800/60 bg-slate-900/40 px-2 py-1.5 text-slate-200 hover:bg-slate-800/60 transition-colors"
                    >
                      <div className="flex items-start gap-2">
                        <div className="relative mt-1 flex flex-col items-center">
                          <span className={`h-2.5 w-2.5 rounded-full ${nodeTone}`} />
                          <span className={`mt-1 w-px flex-1 ${lineTone}`} />
                        </div>
                        <div className="min-w-0 flex-1">
                          <div className="flex flex-wrap items-center gap-2 text-[11px] text-slate-400">
                            {entry.graph && (
                              <span className="text-slate-600">{entry.graph}</span>
                            )}
                            {entry.hash && (
                              <span className="rounded border border-slate-800/70 bg-slate-950/60 px-1.5 py-0.5 text-[10px] text-slate-300">
                                {entry.hash}
                              </span>
                            )}
                            {showHeadBadge && entry.headBranch && (
                              <span className={`rounded border px-1.5 py-0.5 text-[10px] ${chipTone}`}>
                                {entry.headBranch}
                              </span>
                            )}
                            {showRemoteBadge && entry.remoteBranches.length > 0 && (
                              <span className={`rounded border px-1.5 py-0.5 text-[10px] ${chipTone}`}>
                                {entry.remoteBranches[0]}
                              </span>
                            )}
                          </div>
                          <div className="mt-1 text-xs text-slate-200 whitespace-pre-wrap break-words">
                            {entry.message}
                          </div>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </ScrollArea>
        </CardContent>
      )}
    </Card>
  );
}
