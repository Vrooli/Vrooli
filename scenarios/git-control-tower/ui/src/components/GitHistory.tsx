import { ChevronDown, ChevronRight, GitCommit, Loader2, SlidersHorizontal, Eye, EyeOff } from "lucide-react";
import { useCallback, useLayoutEffect, useMemo, useRef } from "react";
import { Card, CardHeader, CardTitle, CardContent } from "./ui/card";
import { ScrollArea } from "./ui/scroll-area";
import { Button } from "./ui/button";
import type { RepoHistoryEntry } from "../lib/api";
import type { GroupingRule } from "./FileList";
import { HistoryFiltersModal } from "./HistoryFiltersModal";

interface GitHistoryProps {
  lines?: string[];
  entries?: RepoHistoryEntry[];
  isLoading: boolean;
  error?: Error | null;
  collapsed?: boolean;
  height?: number;
  fillHeight?: boolean;
  onToggleCollapse?: () => void;
  onLoadMore?: () => void;
  isFetching?: boolean;
  hasMore?: boolean;
  searchQuery?: string;
  onSearchQueryChange?: (value: string) => void;
  scopeFilter?: string | null;
  onScopeFilterChange?: (value: string | null) => void;
  groupingEnabled?: boolean;
  groupingRules?: GroupingRule[];
  workingSetPaths?: string[];
  workingSetOnly?: boolean;
  onWorkingSetOnlyChange?: (value: boolean) => void;
  filtersOpen?: boolean;
  onOpenFilters?: () => void;
  onCloseFilters?: () => void;
  // History mode props
  selectedCommitHash?: string | null;
  onSelectCommit?: (entry: RepoHistoryEntry | null) => void;
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

type NormalizedGroupingRule = GroupingRule & {
  normalizedPrefixes: string[];
  label: string;
  mode: "prefix" | "segment";
};

function parseHistoryLine(line: string): HistoryEntry | null {
  const hashMatch = line.match(/[0-9a-f]{7,}/);
  if (!hashMatch || hashMatch.index === undefined) {
    return null;
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

function normalizePrefix(prefix: string) {
  const trimmed = prefix.trim().replace(/^\/+/, "");
  if (!trimmed) return "";
  return trimmed.endsWith("/") ? trimmed : `${trimmed}/`;
}

export function GitHistory({
  lines = [],
  entries = [],
  isLoading,
  error,
  collapsed = false,
  onToggleCollapse,
  height = 200,
  fillHeight = false,
  onLoadMore,
  isFetching = false,
  hasMore = false,
  searchQuery = "",
  onSearchQueryChange,
  scopeFilter = null,
  onScopeFilterChange,
  groupingEnabled = false,
  groupingRules = [],
  workingSetPaths = [],
  workingSetOnly = false,
  onWorkingSetOnlyChange,
  filtersOpen = false,
  onOpenFilters,
  onCloseFilters,
  selectedCommitHash,
  onSelectCommit
}: GitHistoryProps) {
  const handleToggleCollapse = onToggleCollapse ?? (() => {});
  const handleSearchChange = onSearchQueryChange ?? (() => {});
  const handleScopeChange = onScopeFilterChange ?? (() => {});
  const handleWorkingSetChange = onWorkingSetOnlyChange ?? (() => {});
  const handleOpenFilters = onOpenFilters ?? (() => {});
  const handleCloseFilters = onCloseFilters ?? (() => {});

  // Handle clicking on a commit entry to enter history mode
  const handleCommitClick = useCallback(
    (entry: HistoryEntry) => {
      if (!onSelectCommit || !entry.hash) return;

      // Find the corresponding RepoHistoryEntry with full details
      const details = entries.find(
        (e) => e.hash === entry.hash || entry.hash?.startsWith(e.hash) || e.hash.startsWith(entry.hash ?? "")
      );

      if (!details) {
        // Can't view commit without entry details
        return;
      }

      // Toggle: if already selected, deselect
      if (selectedCommitHash === entry.hash || selectedCommitHash === details.hash) {
        onSelectCommit(null);
      } else {
        onSelectCommit(details);
      }
    },
    [entries, onSelectCommit, selectedCommitHash]
  );
  const scrollRef = useRef<HTMLDivElement | null>(null);
  const isLoadingMoreRef = useRef(false);
  const prevLinesLengthRef = useRef(0);
  const historyEntries = useMemo(() => {
    let remoteSeen = false;
    let firstRemoteIndex = -1;
    const parsed = lines
      .map((line) => parseHistoryLine(line))
      .filter((entry): entry is HistoryEntry => Boolean(entry));
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
  const totalLines = historyEntries.length;
  const hasLines = totalLines > 0;

  // Ref to store scroll position when loading more
  const scrollPositionRef = useRef<number | null>(null);

  const handleScroll = useCallback(() => {
    if (!scrollRef.current || !onLoadMore || !hasMore) return;
    if (isLoadingMoreRef.current || isLoading || isFetching) return;

    const { scrollTop, clientHeight, scrollHeight } = scrollRef.current;
    if (scrollTop + clientHeight >= scrollHeight - 80) {
      isLoadingMoreRef.current = true;
      // Save scroll position RIGHT when we trigger loading, not on every render
      scrollPositionRef.current = scrollTop;
      onLoadMore();
    }
  }, [hasMore, isFetching, isLoading, onLoadMore]);

  // Restore scroll position after new data is rendered, then reset flags
  useLayoutEffect(() => {
    if (
      isLoadingMoreRef.current &&
      scrollPositionRef.current !== null &&
      scrollRef.current &&
      lines.length > prevLinesLengthRef.current
    ) {
      scrollRef.current.scrollTop = scrollPositionRef.current;
      // Reset flags AFTER restoring scroll, inside the effect
      isLoadingMoreRef.current = false;
      scrollPositionRef.current = null;
    }
    prevLinesLengthRef.current = lines.length;
  }, [lines.length]);

  const normalizedRules = useMemo<NormalizedGroupingRule[]>(
    () =>
      groupingRules
        .map((rule) => {
          const rawPrefixes = Array.isArray(rule.prefixes)
            ? rule.prefixes
            : typeof rule.prefix === "string"
              ? [rule.prefix]
              : [];
          const normalizedPrefixes = rawPrefixes
            .map((prefix) => normalizePrefix(prefix))
            .filter((prefix) => prefix);
          if (normalizedPrefixes.length === 0) return null;
          const fallbackLabel = rawPrefixes.find((prefix) => prefix.trim()) ?? "";
          return {
            ...rule,
            mode: (rule.mode === "segment" ? "segment" : "prefix") as "prefix" | "segment",
            normalizedPrefixes,
            label: rule.label.trim() || fallbackLabel.trim()
          };
        })
        .filter((rule): rule is NormalizedGroupingRule => Boolean(rule)),
    [groupingRules]
  );

  const resolveScopeForPath = useCallback(
    (path: string) => {
      for (const rule of normalizedRules) {
        for (const normalizedPrefix of rule.normalizedPrefixes) {
          if (!path.startsWith(normalizedPrefix)) continue;
          if (rule.mode === "segment") {
            const rest = path.slice(normalizedPrefix.length);
            const segment = rest.split("/")[0];
            const label = segment || rule.label;
            const id = segment ? `${rule.id}:${segment}` : rule.id;
            return { id, label };
          }
          return { id: rule.id, label: rule.label };
        }
      }
      return null;
    },
    [normalizedRules]
  );

  const detailsAvailable = entries.length > 0;
  const scopeOptions = useMemo(() => {
    if (!groupingEnabled || normalizedRules.length === 0) return [];
    const scopeMap = new Map<string, { id: string; label: string; count: number }>();
    const order: string[] = [];
    const ensureScope = (id: string, label: string) => {
      if (!scopeMap.has(id)) {
        scopeMap.set(id, { id, label, count: 0 });
        order.push(id);
      }
      return scopeMap.get(id)!;
    };

    if (detailsAvailable) {
      entries.forEach((entry) => {
        const entryScopes = new Map<string, string>();
        entry.files.forEach((file) => {
          const scope = resolveScopeForPath(file);
          if (scope) {
            entryScopes.set(scope.id, scope.label);
          } else {
            entryScopes.set("other", "Other");
          }
        });
        entryScopes.forEach((label, id) => {
          ensureScope(id, label).count += 1;
        });
      });
    } else {
      workingSetPaths.forEach((path) => {
        const scope = resolveScopeForPath(path);
        if (scope) {
          ensureScope(scope.id, scope.label).count += 1;
        } else {
          ensureScope("other", "Other").count += 1;
        }
      });
    }

    return order
      .map((id) => {
        const scope = scopeMap.get(id)!;
        return {
          ...scope,
          display: detailsAvailable ? `${scope.label} (${scope.count})` : scope.label
        };
      })
      .filter((scope) => scope.count > 0);
  }, [detailsAvailable, entries, groupingEnabled, normalizedRules, resolveScopeForPath, workingSetPaths]);

  const workingSetSet = useMemo(() => new Set(workingSetPaths), [workingSetPaths]);
  const normalizedSearch = searchQuery.trim().toLowerCase();
  const hasActiveFilters = Boolean(normalizedSearch || scopeFilter || workingSetOnly);
  const filterNeedsDetails = Boolean(scopeFilter || workingSetOnly);
  const resolveDetails = useCallback(
    (hash?: string) => {
      if (!hash) return undefined;
      const exact = entries.find((entry) => entry.hash === hash);
      if (exact) return exact;
      let match: RepoHistoryEntry | undefined;
      for (const entry of entries) {
        if (!entry.hash.startsWith(hash)) continue;
        if (match) return undefined;
        match = entry;
      }
      return match;
    },
    [entries]
  );
  const visibleEntries = useMemo(() => {
    return historyEntries.filter((entry) => {
      const details = resolveDetails(entry.hash);
      if (normalizedSearch) {
        const haystack = details
          ? `${details.subject} ${details.author ?? ""}`
          : entry.message;
        if (!haystack.toLowerCase().includes(normalizedSearch)) return false;
      }
      if (filterNeedsDetails && !detailsAvailable) {
        return true;
      }
      if (workingSetOnly && details) {
        if (!details.files.some((file) => workingSetSet.has(file))) return false;
      }
      if (scopeFilter && details) {
        const matchesScope = details.files.some((file) => {
          const scope = resolveScopeForPath(file);
          return scope?.id === scopeFilter;
        });
        if (!matchesScope) return false;
      }
      return true;
    });
  }, [
    detailsAvailable,
    filterNeedsDetails,
    historyEntries,
    normalizedSearch,
    resolveDetails,
    resolveScopeForPath,
    scopeFilter,
    workingSetOnly,
    workingSetSet
  ]);

  const countLabel = hasActiveFilters
    ? `${visibleEntries.length}/${totalLines}`
    : `${totalLines}`;
  const hasVisibleEntries = visibleEntries.length > 0;
  const showScopeFilters = scopeOptions.length > 0;
  const detailsPending = hasActiveFilters && filterNeedsDetails && !detailsAvailable && hasLines;

  return (
    <>
      <Card
        className="flex flex-col min-w-0"
        style={{ height: collapsed ? "auto" : fillHeight ? "100%" : height }}
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
          <div className="flex items-center gap-2 ml-auto">
            <Button
              variant="outline"
              size="sm"
              className={`h-7 px-2 text-[10px] ${
                hasActiveFilters ? "bg-white/10 text-white" : ""
              }`}
              onClick={handleOpenFilters}
              type="button"
            >
              <SlidersHorizontal className="h-3 w-3 mr-1" />
              Filters
            </Button>
            <span className="text-xs text-slate-600">{countLabel}</span>
          </div>
        </CardHeader>

        {!collapsed && (
          <CardContent className="flex-1 min-w-0 p-0 overflow-hidden flex flex-col">
            <ScrollArea
              className="min-h-0 flex-1 min-w-0 px-2 pt-2"
              ref={scrollRef}
              onScroll={handleScroll}
            >
              <div style={{ paddingBottom: 48 }}>
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
              {!isLoading && detailsPending && (
                <div className="rounded-md border border-slate-800/70 bg-slate-950/50 px-3 py-2 text-xs text-slate-500 mb-2">
                  Loading detailed history for filters...
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
              {!isLoading && !error && hasLines && !hasVisibleEntries && (
                <div className="flex flex-col items-center justify-center py-6 text-center">
                  <p className="text-sm text-slate-500">No matching commits</p>
                  <p className="text-xs text-slate-600 mt-1">
                    Try clearing filters or widening the search
                  </p>
                </div>
              )}
              {!isLoading && !error && hasLines && hasVisibleEntries && (
                <div className="relative space-y-1 font-mono text-xs text-slate-200">
                  {visibleEntries.map((entry, index) => {
                    const isUnpushed = entry.hash && !entry.isPushed;
                    const isSelected = entry.hash && (
                      selectedCommitHash === entry.hash ||
                      (selectedCommitHash && entry.hash.startsWith(selectedCommitHash)) ||
                      (selectedCommitHash && selectedCommitHash.startsWith(entry.hash))
                    );
                    const nodeTone = isUnpushed ? "bg-amber-400" : "bg-emerald-400";
                    const lineTone = isUnpushed ? "bg-amber-500/30" : "bg-emerald-500/30";
                    const chipTone = isUnpushed
                      ? "border-amber-500/40 text-amber-200 bg-amber-500/10"
                      : "border-emerald-500/40 text-emerald-200 bg-emerald-500/10";
                    const showHeadBadge = Boolean(
                      entry.headBranch && entry.isHead && entry.remoteBranches.length === 0
                    );
                    const showRemoteBadge = entry.isFirstRemote && entry.remoteBranches.length > 0;
                    const canSelect = Boolean(onSelectCommit && entry.hash && entries.length > 0);

                    return (
                      <div
                        key={`${entry.raw}-${index}`}
                        className={`group relative rounded-lg border px-2 py-2 text-slate-200 transition-colors ${
                          isSelected
                            ? "border-amber-500/60 bg-amber-950/40 ring-1 ring-amber-500/30"
                            : "border-slate-800/70 bg-slate-950/30 hover:bg-slate-900/60"
                        } ${canSelect ? "cursor-pointer" : ""}`}
                        onClick={canSelect ? () => handleCommitClick(entry) : undefined}
                        role={canSelect ? "button" : undefined}
                        tabIndex={canSelect ? 0 : undefined}
                        onKeyDown={
                          canSelect
                            ? (e) => {
                                if (e.key === "Enter" || e.key === " ") {
                                  e.preventDefault();
                                  handleCommitClick(entry);
                                }
                              }
                            : undefined
                        }
                      >
                        <div className="flex items-start gap-3">
                          <div className="relative w-5 flex-shrink-0">
                            <span
                              className={`absolute left-2 top-0 h-full w-px ${lineTone}`}
                              aria-hidden="true"
                            />
                            <span
                              className={`absolute left-1.5 top-1.5 h-2.5 w-2.5 rounded-full ${nodeTone} shadow-[0_0_0_2px_rgba(10,12,20,0.9)]`}
                            />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="flex flex-wrap items-center gap-2 text-[11px] text-slate-400">
                              {entry.hash && (
                                <span className="rounded border border-slate-800/80 bg-slate-950/70 px-1.5 py-0.5 text-[10px] text-slate-300">
                                  {entry.hash}
                                </span>
                              )}
                              {isSelected && (
                                <span className="rounded border border-amber-500/40 bg-amber-500/20 px-1.5 py-0.5 text-[10px] text-amber-200 flex items-center gap-1">
                                  <Eye className="h-2.5 w-2.5" />
                                  viewing
                                </span>
                              )}
                              {showHeadBadge && entry.headBranch && (
                                <span
                                  className={`rounded border px-1.5 py-0.5 text-[10px] ${chipTone}`}
                                >
                                  {entry.headBranch}
                                </span>
                              )}
                              {showRemoteBadge && entry.remoteBranches.length > 0 && (
                                <span
                                  className={`rounded border px-1.5 py-0.5 text-[10px] ${chipTone}`}
                                >
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
              {hasMore && (
                <div className="py-3 text-center text-xs text-slate-500">
                  {isFetching ? "Loading more history..." : "Scroll to load more"}
                </div>
              )}
              </div>
            </ScrollArea>
          </CardContent>
        )}
      </Card>
      <HistoryFiltersModal
        isOpen={filtersOpen}
        searchQuery={searchQuery}
        onSearchQueryChange={handleSearchChange}
        scopeFilter={scopeFilter}
        onScopeFilterChange={handleScopeChange}
        scopeOptions={showScopeFilters ? scopeOptions : []}
        workingSetOnly={workingSetOnly}
        onWorkingSetOnlyChange={handleWorkingSetChange}
        onClearFilters={() => {
          handleSearchChange("");
          handleScopeChange(null);
          handleWorkingSetChange(false);
        }}
        onClose={handleCloseFilters}
      />
    </>
  );
}
