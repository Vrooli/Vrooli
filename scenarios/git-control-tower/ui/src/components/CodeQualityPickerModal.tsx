import { useCallback, useEffect, useMemo, useState } from "react";
import { CheckSquare, Square, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";
import { useTidinessIssues } from "../lib/hooks";
import { codeQualityContextItems } from "../lib/agentContext";
import type { AgentContextItem, TidinessIssue } from "../lib/api";

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

export const SEVERITY_LEVELS = ["high", "medium", "low"] as const;
export type SeverityLevel = (typeof SEVERITY_LEVELS)[number];

export const CATEGORIES = [
  "length",
  "complexity",
  "duplication",
  "technical_debt",
  "coupling",
  "type_safety",
] as const;
export type IssueCategory = (typeof CATEGORIES)[number];

export const LIMIT_PRESETS = [10, 25, 50] as const;

const SEVERITY_COLORS: Record<SeverityLevel, { dot: string; active: string; inactive: string }> = {
  high: {
    dot: "bg-red-500",
    active: "border-red-700 bg-red-950 text-red-300",
    inactive: "border-slate-700 text-slate-400 hover:text-slate-200",
  },
  medium: {
    dot: "bg-amber-500",
    active: "border-amber-700 bg-amber-950 text-amber-300",
    inactive: "border-slate-700 text-slate-400 hover:text-slate-200",
  },
  low: {
    dot: "bg-blue-500",
    active: "border-blue-700 bg-blue-950 text-blue-300",
    inactive: "border-slate-700 text-slate-400 hover:text-slate-200",
  },
};

/** Human-friendly display name for issue categories. */
const CATEGORY_LABELS: Record<string, string> = {
  length: "Length",
  complexity: "Complexity",
  duplication: "Duplication",
  technical_debt: "Tech Debt",
  coupling: "Coupling",
  type_safety: "Type Safety",
};

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

interface CodeQualityPickerModalProps {
  isOpen: boolean;
  onClose: () => void;
  scenarioSlug: string;
  repoId?: string | null;
  onAttachItems: (items: AgentContextItem[]) => void;
}

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export function CodeQualityPickerModal({
  isOpen,
  onClose,
  scenarioSlug,
  repoId,
  onAttachItems,
}: CodeQualityPickerModalProps) {
  const isMobile = useIsMobile();

  // ---- filter state ----
  const [selectedSeverities, setSelectedSeverities] = useState<Set<SeverityLevel>>(
    () => new Set(SEVERITY_LEVELS),
  );
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null);
  const [limit, setLimit] = useState(25);

  // ---- data fetching ----
  const { data: issues, isLoading } = useTidinessIssues(scenarioSlug, {
    category: selectedCategory ?? undefined,
    limit,
    enabled: isOpen,
    repoId,
  });

  // Client-side severity filter (backend only supports single-value exact match).
  const filteredIssues = useMemo(() => {
    if (!issues) return [];
    if (selectedSeverities.size === SEVERITY_LEVELS.length) return issues;
    return issues.filter((i) => selectedSeverities.has(i.severity as SeverityLevel));
  }, [issues, selectedSeverities]);

  // ---- selection state ----
  const [checkedIds, setCheckedIds] = useState<Set<number>>(() => new Set());

  // Auto-select all visible issues when filters change.
  useEffect(() => {
    setCheckedIds(new Set(filteredIssues.map((i) => i.id)));
  }, [filteredIssues]);

  // ---- handlers ----
  const toggleSeverity = useCallback((level: SeverityLevel) => {
    setSelectedSeverities((prev) => {
      const next = new Set(prev);
      if (next.has(level)) {
        // Don't allow deselecting the last severity.
        if (next.size > 1) next.delete(level);
      } else {
        next.add(level);
      }
      return next;
    });
  }, []);

  const toggleCategory = useCallback((cat: string) => {
    setSelectedCategory((prev) => (prev === cat ? null : cat));
  }, []);

  const toggleIssue = useCallback((id: number) => {
    setCheckedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const selectAll = useCallback(() => {
    setCheckedIds(new Set(filteredIssues.map((i) => i.id)));
  }, [filteredIssues]);

  const clearSelection = useCallback(() => {
    setCheckedIds(new Set());
  }, []);

  const handleAttach = useCallback(() => {
    const selected = filteredIssues.filter((i) => checkedIds.has(i.id));
    if (selected.length === 0) return;
    onAttachItems(codeQualityContextItems(selected, scenarioSlug));
    onClose();
  }, [filteredIssues, checkedIds, scenarioSlug, onAttachItems, onClose]);

  const clearFilters = useCallback(() => {
    setSelectedSeverities(new Set(SEVERITY_LEVELS));
    setSelectedCategory(null);
    setLimit(25);
  }, []);

  if (!isOpen) return null;

  const selectedCount = checkedIds.size;

  // ----- shared filter chips -----

  const severityChips = SEVERITY_LEVELS.map((level) => {
    const active = selectedSeverities.has(level);
    const colors = SEVERITY_COLORS[level];
    return (
      <button
        key={level}
        type="button"
        onClick={() => toggleSeverity(level)}
        className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs whitespace-nowrap transition-colors ${
          active ? colors.active : colors.inactive
        }`}
        aria-pressed={active}
      >
        <span className={`h-1.5 w-1.5 rounded-full ${colors.dot}`} />
        {level.charAt(0).toUpperCase() + level.slice(1)}
      </button>
    );
  });

  const categoryChips = (
    <>
      <button
        type="button"
        onClick={() => setSelectedCategory(null)}
        className={`rounded-full border px-2.5 py-1 text-xs whitespace-nowrap transition-colors ${
          selectedCategory === null
            ? "border-slate-500 bg-slate-800 text-slate-200"
            : "border-slate-700 text-slate-400 hover:text-slate-200"
        }`}
      >
        All
      </button>
      {CATEGORIES.map((cat) => (
        <button
          key={cat}
          type="button"
          onClick={() => toggleCategory(cat)}
          className={`rounded-full border px-2.5 py-1 text-xs whitespace-nowrap transition-colors ${
            selectedCategory === cat
              ? "border-slate-500 bg-slate-800 text-slate-200"
              : "border-slate-700 text-slate-400 hover:text-slate-200"
          }`}
        >
          {CATEGORY_LABELS[cat] ?? cat}
        </button>
      ))}
    </>
  );

  const limitChips = LIMIT_PRESETS.map((preset) => (
    <button
      key={preset}
      type="button"
      onClick={() => setLimit(preset)}
      className={`rounded-full border px-2.5 py-1 text-xs whitespace-nowrap transition-colors ${
        limit === preset
          ? "border-slate-500 bg-slate-800 text-slate-200"
          : "border-slate-700 text-slate-400 hover:text-slate-200"
      }`}
    >
      {preset}
    </button>
  ));

  // Desktop: labeled sections with wrapping
  const desktopFiltersContent = (
    <div className="space-y-3">
      <div className="space-y-1.5">
        <label className="uppercase tracking-wide text-slate-500 text-[11px]">Severity</label>
        <div className="flex flex-wrap gap-1.5">{severityChips}</div>
      </div>
      <div className="space-y-1.5">
        <label className="uppercase tracking-wide text-slate-500 text-[11px]">Category</label>
        <div className="flex flex-wrap gap-1.5">{categoryChips}</div>
      </div>
      <div className="space-y-1.5">
        <label className="uppercase tracking-wide text-slate-500 text-[11px]">Show</label>
        <div className="flex gap-1.5">{limitChips}</div>
      </div>
    </div>
  );

  // Mobile: compact horizontal scrollable rows, no labels — just inline groups
  const mobileFiltersContent = (
    <div className="space-y-2">
      <div className="flex gap-1.5 overflow-x-auto pb-1">
        {severityChips}
        <span className="border-l border-slate-700 mx-0.5 shrink-0" />
        {limitChips}
      </div>
      <div className="flex gap-1.5 overflow-x-auto pb-1">
        {categoryChips}
      </div>
    </div>
  );

  const issueListContent = (
    <div className="flex-1 min-h-0 flex flex-col">
      {/* Bulk actions + count */}
      <div className={`flex items-center justify-between ${isMobile ? "px-4 py-2" : "px-4 py-1.5"}`}>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={selectAll}
            className="text-[11px] text-blue-400 hover:text-blue-300"
          >
            Select all
          </button>
          <span className="text-slate-700">|</span>
          <button
            type="button"
            onClick={clearSelection}
            className="text-[11px] text-slate-500 hover:text-slate-300"
          >
            Clear
          </button>
        </div>
        <span className="text-[11px] text-slate-500">
          {filteredIssues.length} issue{filteredIssues.length !== 1 ? "s" : ""}
        </span>
      </div>

      {/* Scrollable issue list */}
      <div className={`flex-1 overflow-y-auto border-t border-slate-800/50 ${isMobile ? "px-2" : "px-2"}`}>
        {isLoading && (
          <div className="space-y-2 py-3 px-2">
            <div className="h-8 animate-pulse bg-slate-800 rounded" />
            <div className="h-8 animate-pulse bg-slate-800 rounded" />
            <div className="h-8 animate-pulse bg-slate-800 rounded" />
          </div>
        )}
        {!isLoading && filteredIssues.length === 0 && (
          <p className={`text-center py-8 text-slate-500 ${isMobile ? "text-sm" : "text-xs"}`}>
            No issues match the current filters
          </p>
        )}
        {!isLoading &&
          filteredIssues.map((issue) => (
            <IssueRow
              key={issue.id}
              issue={issue}
              checked={checkedIds.has(issue.id)}
              onToggle={toggleIssue}
              isMobile={isMobile}
            />
          ))}
      </div>
    </div>
  );

  // ----- Mobile layout -----
  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Code quality issue picker"
      >
        {/* Header — compact with inline close */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-2.5 pt-safe">
          <h2 className="text-sm font-semibold text-slate-100">Code Quality Issues</h2>
          <button
            type="button"
            className="h-9 w-9 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Filters — compact horizontal layout */}
        <div className="px-3 py-2 border-b border-slate-800/50">
          {mobileFiltersContent}
        </div>

        {/* Issue list — takes remaining space */}
        {issueListContent}

        {/* Footer — compact */}
        <div className="flex items-center gap-3 border-t border-slate-800 px-4 py-3 pb-safe">
          <Button
            variant="outline"
            size="sm"
            onClick={clearFilters}
            className="h-10 px-4 text-sm touch-target"
          >
            Clear
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={handleAttach}
            disabled={selectedCount === 0}
            className="flex-1 h-10 text-sm touch-target"
          >
            Attach {selectedCount} selected
          </Button>
        </div>
      </div>
    );
  }

  // ----- Desktop layout -----
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Code quality issue picker"
    >
      <div className="w-full max-w-2xl rounded-xl border border-slate-800 bg-slate-950 shadow-xl flex flex-col max-h-[80vh]">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-100">Code Quality Issues</h2>
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Filters */}
        <div className="px-4 py-3 border-b border-slate-800/50">
          {desktopFiltersContent}
        </div>

        {/* Issue list */}
        {issueListContent}

        {/* Footer */}
        <div className="flex items-center justify-between border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={clearFilters} className="h-8 px-3">
            Clear filters
          </Button>
          <Button
            variant="default"
            size="sm"
            onClick={handleAttach}
            disabled={selectedCount === 0}
            className="h-8 px-3"
          >
            Attach {selectedCount} selected
          </Button>
        </div>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Issue row sub-component
// ---------------------------------------------------------------------------

function IssueRow({
  issue,
  checked,
  onToggle,
  isMobile,
}: {
  issue: TidinessIssue;
  checked: boolean;
  onToggle: (id: number) => void;
  isMobile: boolean;
}) {
  const severityColor =
    issue.severity === "critical" || issue.severity === "high"
      ? "bg-red-500"
      : issue.severity === "medium"
        ? "bg-amber-500"
        : "bg-blue-500";

  return (
    <button
      type="button"
      onClick={() => onToggle(issue.id)}
      className={`w-full flex items-start gap-2 rounded text-left hover:bg-slate-800/60 transition-colors ${
        isMobile ? "px-3 py-3 touch-target" : "px-2 py-1.5"
      }`}
    >
      {checked ? (
        <CheckSquare className="h-3.5 w-3.5 text-blue-400 shrink-0 mt-0.5" />
      ) : (
        <Square className="h-3.5 w-3.5 text-slate-600 shrink-0 mt-0.5" />
      )}
      <span className={`${severityColor} h-1.5 w-1.5 rounded-full shrink-0 mt-1.5`} />
      <div className="min-w-0 flex-1">
        <div className={`truncate ${isMobile ? "text-sm" : "text-xs"} text-slate-300`}>
          <span className="text-slate-500">{CATEGORY_LABELS[issue.category] ?? issue.category}:</span>{" "}
          <code className="text-slate-200">{issue.file_path}</code>
          {issue.line_number != null && (
            <span className="text-slate-600">:{issue.line_number}</span>
          )}
        </div>
        <div className={`truncate text-slate-500 ${isMobile ? "text-xs mt-0.5" : "text-[11px]"}`}>
          {issue.title}
        </div>
      </div>
    </button>
  );
}
