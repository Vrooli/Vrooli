import { X } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";

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
  const isMobile = useIsMobile();
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

  // Shared form content
  const formContent = (
    <>
      <div className={`space-y-2 ${isMobile ? "space-y-3" : ""}`}>
        <label className={`uppercase tracking-wide text-slate-500 ${isMobile ? "text-xs" : "text-[11px]"}`}>
          Search
        </label>
        <input
          className={`w-full rounded-md border border-slate-800/80 bg-slate-950/60 text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-slate-500/60 ${
            isMobile ? "rounded-lg px-4 py-3 text-sm touch-target" : "px-2.5 py-2 text-xs"
          }`}
          placeholder="Commit message or author"
          value={searchQuery}
          onChange={(event) => onSearchQueryChange(event.target.value)}
          type="search"
        />
      </div>

      <div className={`space-y-2 ${isMobile ? "space-y-3" : ""}`}>
        <label className={`uppercase tracking-wide text-slate-500 ${isMobile ? "text-xs" : "text-[11px]"}`}>
          Scope
        </label>
        {scopeOptions.length === 0 ? (
          <div className={`rounded-lg border border-slate-800/70 bg-slate-900/40 text-slate-500 ${
            isMobile ? "px-4 py-4 text-sm" : "px-3 py-3 text-xs"
          }`}>
            No scopes available yet. Add changes or enable grouping to see scopes.
          </div>
        ) : (
          <div className="relative">
            <input
              className={`w-full rounded-md border border-slate-800/80 bg-slate-950/60 text-slate-200 placeholder:text-slate-600 focus:outline-none focus:ring-1 focus:ring-slate-500/60 ${
                isMobile ? "rounded-lg px-4 py-3 text-sm pr-10 touch-target" : "px-2.5 py-2 text-xs"
              }`}
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
                className={`absolute top-1/2 -translate-y-1/2 rounded text-slate-500 hover:text-slate-200 ${
                  isMobile ? "right-3 p-1 touch-target" : "right-2 p-0.5"
                }`}
                onClick={() => {
                  setScopeInput("");
                  onScopeFilterChange(null);
                }}
                type="button"
                aria-label="Clear scope filter"
              >
                <X className={isMobile ? "h-4 w-4" : "h-3 w-3"} />
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

      <div className={`flex items-center justify-between gap-2 rounded-lg border border-slate-800/70 bg-slate-900/40 ${
        isMobile ? "px-4 py-4 rounded-xl" : "px-3 py-2"
      }`}>
        <div>
          <div className={`font-semibold text-slate-200 ${isMobile ? "text-sm" : "text-xs"}`}>
            Working set only
          </div>
          <div className={`text-slate-500 ${isMobile ? "text-xs mt-1" : "text-[11px]"}`}>
            Show commits that touch staged or unstaged files.
          </div>
        </div>
        <button
          type="button"
          onClick={() => onWorkingSetOnlyChange(!workingSetOnly)}
          className={`rounded-full border ${
            isMobile ? "h-10 px-5 text-sm touch-target" : "h-7 px-3 text-xs"
          } ${
            workingSetOnly
              ? "border-emerald-400/40 text-emerald-200 bg-emerald-900/20"
              : "border-slate-700 text-slate-300 hover:bg-slate-800/50"
          }`}
        >
          {workingSetOnly ? "On" : "Off"}
        </button>
      </div>
    </>
  );

  // Mobile: full-screen modal
  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="History filters"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <div>
            <h2 className="text-base font-semibold text-slate-100">History Filters</h2>
            <p className="text-xs text-slate-500 mt-1">
              Narrow commits by search, scope, or working set.
            </p>
          </div>
          <button
            type="button"
            className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-6 space-y-6">
          {formContent}
        </div>

        {/* Footer */}
        <div className="flex items-center gap-3 border-t border-slate-800 px-4 py-4 pb-safe">
          <Button
            variant="outline"
            size="sm"
            onClick={onClearFilters}
            className="flex-1 h-12 text-sm touch-target"
          >
            Clear
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={onClose}
            className="flex-1 h-12 text-sm touch-target"
          >
            Done
          </Button>
        </div>
      </div>
    );
  }

  // Desktop: centered modal
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
          {formContent}
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
