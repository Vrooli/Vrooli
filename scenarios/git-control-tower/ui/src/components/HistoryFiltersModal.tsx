import { X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Button } from "./ui/button";

export interface HistoryScopeOption {
  id: string;
  label: string;
  count: number;
  display: string;
}

interface HistoryFiltersModalProps {
  isOpen: boolean;
  searchQuery: string;
  onSearchQueryChange: (value: string) => void;
  scopeFilter: string | null;
  onScopeFilterChange: (value: string | null) => void;
  scopeOptions: HistoryScopeOption[];
  workingSetOnly: boolean;
  onWorkingSetOnlyChange: (value: boolean) => void;
  onClearFilters: () => void;
  onClose: () => void;
}

export function HistoryFiltersModal({
  isOpen,
  searchQuery,
  onSearchQueryChange,
  scopeFilter,
  onScopeFilterChange,
  scopeOptions,
  workingSetOnly,
  onWorkingSetOnlyChange,
  onClearFilters,
  onClose
}: HistoryFiltersModalProps) {
  const [scopeInput, setScopeInput] = useState("");
  const scopeLabelById = useMemo(() => {
    const map = new Map<string, string>();
    scopeOptions.forEach((scope) => {
      map.set(scope.id, scope.label);
    });
    return map;
  }, [scopeOptions]);
  const scopeIdByValue = useMemo(() => {
    const map = new Map<string, string>();
    scopeOptions.forEach((scope) => {
      map.set(scope.label.toLowerCase(), scope.id);
      map.set(scope.display.toLowerCase(), scope.id);
    });
    return map;
  }, [scopeOptions]);

  useEffect(() => {
    if (!scopeFilter) {
      setScopeInput("");
      return;
    }
    const label = scopeLabelById.get(scopeFilter);
    setScopeInput(label ?? "");
  }, [scopeFilter, scopeLabelById]);

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="History filters"
    >
      <div className="w-full max-w-xl rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <div>
            <h2 className="text-sm font-semibold text-slate-100">History filters</h2>
            <p className="text-[11px] text-slate-500 mt-1">
              Narrow commits by search, scope, or working set.
            </p>
          </div>
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
            onClick={onClose}
            aria-label="Close filters"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="px-4 py-4 space-y-4">
          <div className="space-y-2">
            <label className="text-[11px] uppercase tracking-wide text-slate-500">
              Search
            </label>
            <input
              className="w-full rounded-md border border-slate-800/80 bg-slate-950/60 px-2.5 py-2 text-xs text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-slate-500/60"
              placeholder="Commit message or author"
              value={searchQuery}
              onChange={(event) => onSearchQueryChange(event.target.value)}
              type="search"
            />
          </div>

          <div className="space-y-2">
            <label className="text-[11px] uppercase tracking-wide text-slate-500">
              Scope
            </label>
            {scopeOptions.length === 0 ? (
              <div className="rounded-lg border border-slate-800/70 bg-slate-900/40 px-3 py-3 text-xs text-slate-500">
                No scopes available yet. Add changes or enable grouping to see scopes.
              </div>
            ) : (
              <div className="relative">
                <input
                  className="w-full rounded-md border border-slate-800/80 bg-slate-950/60 px-2.5 py-2 text-xs text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-slate-500/60"
                  placeholder="Filter by scope"
                  value={scopeInput}
                  onChange={(event) => {
                    const next = event.target.value;
                    setScopeInput(next);
                    const match = scopeIdByValue.get(next.trim().toLowerCase());
                    onScopeFilterChange(match ?? null);
                  }}
                  list="history-scope-options"
                  type="search"
                />
                {scopeInput && (
                  <button
                    className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-slate-500 hover:text-slate-200"
                    onClick={() => {
                      setScopeInput("");
                      onScopeFilterChange(null);
                    }}
                    type="button"
                    aria-label="Clear scope filter"
                  >
                    <X className="h-3 w-3" />
                  </button>
                )}
                <datalist id="history-scope-options">
                  {scopeOptions.map((scope) => (
                    <option key={scope.id} value={scope.display} />
                  ))}
                </datalist>
              </div>
            )}
          </div>

          <div className="flex items-center justify-between gap-2 rounded-lg border border-slate-800/70 bg-slate-900/40 px-3 py-2">
            <div>
              <div className="text-xs font-semibold text-slate-200">Working set only</div>
              <div className="text-[11px] text-slate-500">
                Show commits that touch staged or unstaged files.
              </div>
            </div>
            <button
              type="button"
              onClick={() => onWorkingSetOnlyChange(!workingSetOnly)}
              className={`h-7 px-3 rounded-full border text-xs ${
                workingSetOnly
                  ? "border-emerald-400/40 text-emerald-200 bg-emerald-900/20"
                  : "border-slate-700 text-slate-300 hover:bg-slate-800/50"
              }`}
            >
              {workingSetOnly ? "On" : "Off"}
            </button>
          </div>
        </div>

        <div className="flex items-center justify-between border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onClearFilters} className="h-8 px-3">
            Clear filters
          </Button>
          <Button variant="outline" size="sm" onClick={onClose} className="h-8 px-3">
            Done
          </Button>
        </div>
      </div>
    </div>
  );
}
