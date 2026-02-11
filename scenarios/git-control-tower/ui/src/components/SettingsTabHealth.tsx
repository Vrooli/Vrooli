import { useState, useMemo } from "react";
import { AlertTriangle, ArrowRight, Check, ChevronDown, ChevronRight, Info, X } from "lucide-react";
import { Button } from "./ui/button";
import { useGitignoreHealth, useGitignoreMove, useGroupingRules } from "../lib/hooks";
import type { GitignoreSuggestion } from "../lib/api";

interface SettingsTabHealthProps {
  isMobile: boolean;
  repoId?: string | null;
}

export function SettingsTabHealth({ isMobile, repoId }: SettingsTabHealthProps) {
  const healthQuery = useGitignoreHealth(repoId);
  const moveMutation = useGitignoreMove(repoId);
  const groupingRulesQuery = useGroupingRules(repoId);
  const [dismissedPatterns, setDismissedPatterns] = useState<Set<string>>(() => {
    try {
      const stored = localStorage.getItem(`gct.gitignore.dismissals`);
      return stored ? new Set(JSON.parse(stored)) : new Set();
    } catch {
      return new Set();
    }
  });
  const [showCrossGroup, setShowCrossGroup] = useState(false);
  const [movingLine, setMovingLine] = useState<number | null>(null);

  const hasGroupingRules = (groupingRulesQuery.data?.rules?.length ?? 0) > 0;

  const suggestions = healthQuery.data?.suggestions ?? [];

  const movable = useMemo(
    () => suggestions.filter(s => s.type === "single_group" && !dismissedPatterns.has(s.pattern)),
    [suggestions, dismissedPatterns]
  );
  const crossGroup = useMemo(
    () => suggestions.filter(s => s.type === "cross_group"),
    [suggestions]
  );
  const dismissedCount = useMemo(
    () => suggestions.filter(s => s.type === "single_group" && dismissedPatterns.has(s.pattern)).length,
    [suggestions, dismissedPatterns]
  );

  const handleDismiss = (pattern: string) => {
    const next = new Set(dismissedPatterns);
    next.add(pattern);
    setDismissedPatterns(next);
    try {
      localStorage.setItem(`gct.gitignore.dismissals`, JSON.stringify([...next]));
    } catch { /* ignore */ }
  };

  const handleResetDismissals = () => {
    setDismissedPatterns(new Set());
    try {
      localStorage.removeItem(`gct.gitignore.dismissals`);
    } catch { /* ignore */ }
  };

  const handleMove = (suggestion: GitignoreSuggestion) => {
    setMovingLine(suggestion.line);
    moveMutation.mutate(
      {
        line: suggestion.line,
        pattern: suggestion.pattern,
        group_dir: suggestion.group_dir,
        target_pattern: suggestion.target_pattern,
      },
      {
        onSettled: () => setMovingLine(null),
      }
    );
  };

  const textSm = isMobile ? "text-sm" : "text-xs";
  const textXs = isMobile ? "text-xs" : "text-[11px]";
  const gap = isMobile ? "gap-3" : "gap-2";
  const py = isMobile ? "py-3" : "py-2";
  const px = isMobile ? "px-4" : "px-3";

  if (!hasGroupingRules) {
    return (
      <div className="space-y-4">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>.gitignore Health</h3>
        <div className={`flex items-start gap-2 rounded-lg border border-slate-800 bg-slate-900/40 ${px} ${py}`}>
          <Info className="h-4 w-4 text-slate-400 mt-0.5 shrink-0" />
          <p className={`${textXs} text-slate-400`}>
            Configure grouping rules to enable .gitignore health analysis. Grouping rules define how files are organized into groups, which is needed to detect misplaced .gitignore entries.
          </p>
        </div>
      </div>
    );
  }

  if (healthQuery.isLoading) {
    return (
      <div className="space-y-4">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>.gitignore Health</h3>
        <p className={`${textXs} text-slate-500`}>Analyzing...</p>
      </div>
    );
  }

  if (healthQuery.isError) {
    return (
      <div className="space-y-4">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>.gitignore Health</h3>
        <p className={`${textXs} text-red-400`}>Failed to analyze .gitignore: {healthQuery.error.message}</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className={`font-semibold text-slate-200 ${textSm}`}>.gitignore Health</h3>
        <span className={`${textXs} text-slate-500`}>
          {healthQuery.data?.root_entry_count ?? 0} entries in root
        </span>
      </div>

      {movable.length === 0 && crossGroup.length === 0 && (
        <div className={`flex items-center gap-2 rounded-lg border border-green-900/50 bg-green-950/20 ${px} ${py}`}>
          <Check className="h-4 w-4 text-green-400 shrink-0" />
          <p className={`${textXs} text-green-300`}>Root .gitignore looks clean. No group-specific entries detected.</p>
        </div>
      )}

      {movable.length > 0 && (
        <div className="space-y-2">
          <p className={`${textXs} text-slate-400`}>
            {movable.length} {movable.length === 1 ? "entry" : "entries"} could be moved to group-level .gitignore files:
          </p>
          {movable.map((s) => (
            <div
              key={`${s.line}-${s.pattern}`}
              className={`flex items-center justify-between rounded-lg border border-slate-800 bg-slate-900/40 ${px} ${py} ${gap}`}
            >
              <div className="flex-1 min-w-0">
                <p className={`${textSm} text-slate-200 font-mono truncate`}>{s.pattern}</p>
                <div className={`flex items-center gap-1 ${textXs} text-slate-500 mt-0.5`}>
                  <span>Line {s.line}</span>
                  <ArrowRight className="h-3 w-3" />
                  <span className="text-slate-400">{s.group_dir}.gitignore</span>
                  {!s.has_gitignore && (
                    <span className="text-amber-500 ml-1">(new file)</span>
                  )}
                </div>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <Button
                  variant="outline"
                  size="sm"
                  className="h-7 px-2 text-xs"
                  onClick={() => handleMove(s)}
                  disabled={movingLine === s.line || moveMutation.isPending}
                >
                  {movingLine === s.line ? "Moving..." : "Move"}
                </Button>
                <button
                  type="button"
                  className="h-7 w-7 inline-flex items-center justify-center rounded text-slate-500 hover:text-slate-300 hover:bg-slate-800"
                  onClick={() => handleDismiss(s.pattern)}
                  title="Dismiss"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {crossGroup.length > 0 && (
        <div className="space-y-2">
          <button
            type="button"
            className={`flex items-center gap-1 ${textXs} text-slate-500 hover:text-slate-300`}
            onClick={() => setShowCrossGroup(!showCrossGroup)}
          >
            {showCrossGroup ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
            {crossGroup.length} cross-group {crossGroup.length === 1 ? "pattern" : "patterns"} (informational)
          </button>
          {showCrossGroup && crossGroup.map((s) => (
            <div
              key={`${s.line}-${s.pattern}`}
              className={`flex items-center rounded-lg border border-slate-800/50 bg-slate-900/20 ${px} ${py} ${gap}`}
            >
              <AlertTriangle className="h-3.5 w-3.5 text-slate-600 shrink-0" />
              <div className="flex-1 min-w-0">
                <p className={`${textXs} text-slate-500 font-mono truncate`}>{s.pattern}</p>
                <p className={`${textXs} text-slate-600 mt-0.5`}>
                  Line {s.line} -- spans multiple {s.group_label} groups
                </p>
              </div>
            </div>
          ))}
        </div>
      )}

      {dismissedCount > 0 && (
        <div className={`flex items-center justify-between ${textXs} text-slate-600`}>
          <span>{dismissedCount} dismissed</span>
          <button
            type="button"
            className="text-slate-500 hover:text-slate-300 underline"
            onClick={handleResetDismissals}
          >
            Reset dismissals
          </button>
        </div>
      )}
    </div>
  );
}
