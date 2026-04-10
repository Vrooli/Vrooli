import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { createPortal } from "react-dom";
import { Search, X, Loader2, CheckCircle2 } from "lucide-react";
import { useScenarios } from "../lib/hooks";
import { useIsMobile } from "../hooks";


const RECENTS_KEY = "gct-recent-scenarios";
const MAX_RECENTS = 10;

type SortOption = "recent" | "name" | "status";
type FilterOption = "all" | "running" | "stopped";

interface ScenarioPickerModalProps {
  isOpen: boolean;
  onClose: () => void;
  currentScenario: string;
  onSelect: (slug: string) => void;
}

function readRecents(): string[] {
  try {
    const raw = localStorage.getItem(RECENTS_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed as string[]) : [];
  } catch {
    return [];
  }
}

function pushRecent(slug: string) {
  try {
    const recents = readRecents().filter((s) => s !== slug);
    recents.unshift(slug);
    localStorage.setItem(RECENTS_KEY, JSON.stringify(recents.slice(0, MAX_RECENTS)));
  } catch {
    // Ignore storage errors
  }
}

function StatusDot({ status, healthStatus }: { status: string; healthStatus: string | null }) {
  if (status === "running") {
    const healthy = healthStatus === "healthy";
    return (
      <span
        className={`inline-block h-2 w-2 rounded-full ${healthy ? "bg-emerald-400" : "bg-yellow-400"}`}
        title={healthy ? "Running (healthy)" : `Running (${healthStatus ?? "unknown"})`}
      />
    );
  }
  return <span className="inline-block h-2 w-2 rounded-full bg-slate-600" title="Stopped" />;
}

export function ScenarioPickerModal({ isOpen, onClose, currentScenario, onSelect }: ScenarioPickerModalProps) {
  const isMobile = useIsMobile();
  const scenariosQuery = useScenarios(isOpen);
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<SortOption>("recent");
  const [filter, setFilter] = useState<FilterOption>("all");
  const [focusedIndex, setFocusedIndex] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);

  // Focus search on open
  useEffect(() => {
    if (isOpen) {
      setSearch("");
      setFocusedIndex(0);
      // Delay focus to allow portal to mount
      const t = setTimeout(() => searchRef.current?.focus(), 50);
      return () => clearTimeout(t);
    }
  }, [isOpen]);

  const scenarios = useMemo(() => scenariosQuery.data ?? [], [scenariosQuery.data]);
  const recents = useMemo(() => readRecents(), [isOpen]); // eslint-disable-line react-hooks/exhaustive-deps

  const filtered = useMemo(() => {
    let list = scenarios;

    // Filter by status
    if (filter !== "all") {
      list = list.filter((s) => s.status === filter);
    }

    // Filter by search
    if (search.trim()) {
      const q = search.toLowerCase();
      list = list.filter(
        (s) =>
          s.name.toLowerCase().includes(q) ||
          s.display_name.toLowerCase().includes(q) ||
          s.description.toLowerCase().includes(q)
      );
    }

    // Sort
    const sorted = [...list];
    switch (sortBy) {
      case "name":
        sorted.sort((a, b) => a.display_name.localeCompare(b.display_name));
        break;
      case "status":
        sorted.sort((a, b) => {
          if (a.status === b.status) return a.display_name.localeCompare(b.display_name);
          return a.status === "running" ? -1 : 1;
        });
        break;
      case "recent": {
        const recentIndex = new Map(recents.map((slug, i) => [slug, i]));
        sorted.sort((a, b) => {
          const ai = recentIndex.get(a.name) ?? Infinity;
          const bi = recentIndex.get(b.name) ?? Infinity;
          if (ai !== bi) return ai - bi;
          return a.display_name.localeCompare(b.display_name);
        });
        break;
      }
    }

    return sorted;
  }, [scenarios, search, sortBy, filter, recents]);

  // Keep focused index in bounds
  useEffect(() => {
    if (focusedIndex >= filtered.length) {
      setFocusedIndex(Math.max(0, filtered.length - 1));
    }
  }, [filtered.length, focusedIndex]);

  const handleSelect = useCallback(
    (slug: string) => {
      pushRecent(slug);
      onSelect(slug);
      onClose();
    },
    [onSelect, onClose]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Escape") {
        e.stopPropagation();
        onClose();
        return;
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setFocusedIndex((i) => Math.min(i + 1, filtered.length - 1));
      } else if (e.key === "ArrowUp") {
        e.preventDefault();
        setFocusedIndex((i) => Math.max(i - 1, 0));
      } else if (e.key === "Enter" && filtered.length > 0) {
        e.preventDefault();
        const target = filtered[focusedIndex];
        if (target) handleSelect(target.name);
      }
    },
    [filtered, focusedIndex, handleSelect, onClose]
  );

  // Scroll focused card into view
  useEffect(() => {
    if (!gridRef.current) return;
    const cards = gridRef.current.querySelectorAll("[data-scenario-card]");
    const card = cards[focusedIndex];
    if (card) {
      card.scrollIntoView({ block: "nearest" });
    }
  }, [focusedIndex]);

  if (!isOpen) return null;

  const filterButtons: { value: FilterOption; label: string }[] = [
    { value: "all", label: "All" },
    { value: "running", label: "Running" },
    { value: "stopped", label: "Stopped" },
  ];

  const sortOptions: { value: SortOption; label: string }[] = [
    { value: "recent", label: "Recent" },
    { value: "name", label: "Name (A-Z)" },
    { value: "status", label: "Status" },
  ];

  const content = (
    <div
      className={
        isMobile
          ? "fixed inset-0 z-[60] flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
          : "fixed inset-0 z-[60] flex items-center justify-center bg-slate-950/80 px-4"
      }
      role="dialog"
      aria-modal="true"
      aria-label="Select Scenario"
      onKeyDown={handleKeyDown}
    >
      <div
        className={
          isMobile
            ? "flex flex-col h-full"
            : "w-full max-w-2xl max-h-[80vh] flex flex-col rounded-xl border border-slate-800 bg-slate-950 shadow-xl"
        }
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h3 className="font-semibold text-slate-100 text-sm">Switch Scenario</h3>
          <button
            type="button"
            className={`inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 ${isMobile ? "h-11 w-11" : "h-8 w-8"}`}
            onClick={onClose}
            aria-label="Close"
          >
            <X className={isMobile ? "h-5 w-5" : "h-4 w-4"} />
          </button>
        </div>

        {/* Search + Controls */}
        <div className="px-4 py-3 space-y-2 border-b border-slate-800/50">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
            <input
              ref={searchRef}
              type="text"
              placeholder="Search scenarios..."
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setFocusedIndex(0);
              }}
              className="w-full rounded-lg border border-slate-700 bg-slate-900 pl-10 pr-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>

          <div className="flex items-center justify-between gap-2">
            {/* Filter buttons */}
            <div className="flex gap-1">
              {filterButtons.map((fb) => (
                <button
                  key={fb.value}
                  type="button"
                  className={`px-2.5 py-1 rounded text-xs font-medium transition-colors ${
                    filter === fb.value
                      ? "bg-blue-600/20 text-blue-400 border border-blue-500/30"
                      : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60 border border-transparent"
                  }`}
                  onClick={() => setFilter(fb.value)}
                >
                  {fb.label}
                </button>
              ))}
            </div>

            {/* Sort */}
            <select
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as SortOption)}
              className="rounded border border-slate-700 bg-slate-900 px-2 py-1 text-xs text-slate-300 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              {sortOptions.map((so) => (
                <option key={so.value} value={so.value}>
                  {so.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {/* Results */}
        <div ref={gridRef} className="flex-1 overflow-y-auto px-4 py-3">
          {scenariosQuery.isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-5 w-5 animate-spin text-slate-400" />
              <span className="ml-2 text-sm text-slate-400">Loading scenarios...</span>
            </div>
          ) : scenariosQuery.isError ? (
            <div className="text-center py-12 text-sm text-red-400">
              Failed to load scenarios. Try again later.
            </div>
          ) : filtered.length === 0 ? (
            <div className="text-center py-12 text-sm text-slate-500">
              {search ? "No scenarios match your search." : "No scenarios found."}
            </div>
          ) : (
            <div className={isMobile ? "space-y-2" : "grid grid-cols-2 gap-2"}>
              {filtered.map((scenario, idx) => {
                const isCurrent = scenario.name === currentScenario;
                const isFocused = idx === focusedIndex;
                return (
                  <button
                    key={scenario.name}
                    type="button"
                    data-scenario-card
                    className={`w-full text-left rounded-lg border p-3 transition-colors cursor-pointer ${
                      isCurrent
                        ? "border-blue-500/50 bg-blue-950/30"
                        : isFocused
                          ? "border-slate-600 bg-slate-800/40"
                          : "border-slate-800 hover:border-slate-600 hover:bg-slate-800/30"
                    }`}
                    onClick={() => handleSelect(scenario.name)}
                    onMouseEnter={() => setFocusedIndex(idx)}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <StatusDot status={scenario.status} healthStatus={scenario.health_status} />
                          <span className="font-medium text-sm text-slate-100 truncate">
                            {scenario.display_name}
                          </span>
                          {isCurrent && (
                            <CheckCircle2 className="h-3.5 w-3.5 text-blue-400 shrink-0" />
                          )}
                        </div>
                        <p className="text-[11px] text-slate-500 mt-0.5 truncate">{scenario.name}</p>
                      </div>
                      {scenario.status === "running" && scenario.runtime !== "N/A" && (
                        <span className="text-[10px] text-slate-500 shrink-0">{scenario.runtime}</span>
                      )}
                    </div>
                    {scenario.description && (
                      <p className="text-xs text-slate-400 mt-1.5 line-clamp-2">{scenario.description}</p>
                    )}
                  </button>
                );
              })}
            </div>
          )}
        </div>

        {/* Footer with count */}
        <div className="border-t border-slate-800 px-4 py-2 text-xs text-slate-500">
          {filtered.length} of {scenarios.length} scenarios
        </div>
      </div>
    </div>
  );

  return createPortal(content, document.body);
}
