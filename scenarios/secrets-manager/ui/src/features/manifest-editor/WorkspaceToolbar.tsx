import { useState, useEffect } from "react";
import { Search, Filter, Copy, Save, Download, AlertTriangle, Layers, FileCode, HelpCircle } from "lucide-react";
import { Button } from "../../components/ui/button";
import type { FilterMode } from "./types";

interface WorkspaceToolbarProps {
  scenario: string;
  tier: string;
  availableTiers: Array<{ value: string; label: string }>;
  availableScenarios: Array<{ name: string; display_name?: string }>;
  searchQuery: string;
  filter: FilterMode;
  hasPendingChanges: boolean;
  isSaving: boolean;
  isCopying?: boolean;
  copyError?: string | null;
  onScenarioChange: (scenario: string) => void;
  onTierChange: (tier: string) => void;
  onSearchChange: (query: string) => void;
  onFilterChange: (filter: FilterMode) => void;
  onSaveAll: () => void;
  onExport: () => void;
  onCopyFromTier?: (sourceTier: string) => Promise<void>;
}

const FILTER_OPTIONS: { value: FilterMode; label: string; description: string }[] = [
  { value: "all", label: "All secrets", description: "Show all secrets in the manifest" },
  { value: "blocking", label: "Blocking only", description: "Show secrets without a valid strategy" },
  { value: "overridden", label: "Overridden only", description: "Show secrets with custom overrides" },
  { value: "excluded", label: "Excluded only", description: "Show secrets excluded from export" }
];

export function WorkspaceToolbar({
  scenario,
  tier,
  availableTiers,
  availableScenarios,
  searchQuery,
  filter,
  hasPendingChanges,
  isSaving,
  isCopying,
  copyError,
  onScenarioChange,
  onTierChange,
  onSearchChange,
  onFilterChange,
  onSaveAll,
  onExport,
  onCopyFromTier
}: WorkspaceToolbarProps) {
  const [showCopyDropdown, setShowCopyDropdown] = useState(false);
  const [localCopyError, setLocalCopyError] = useState<string | null>(null);
  const otherTiers = availableTiers.filter((t) => t.value !== tier);

  // Auto-dismiss error after 5 seconds
  useEffect(() => {
    const displayError = copyError || localCopyError;
    if (displayError) {
      const timer = setTimeout(() => {
        setLocalCopyError(null);
      }, 5000);
      return () => clearTimeout(timer);
    }
  }, [copyError, localCopyError]);

  const handleCopyFromTier = async (sourceTier: string) => {
    setShowCopyDropdown(false);
    setLocalCopyError(null);
    try {
      await onCopyFromTier?.(sourceTier);
    } catch (err) {
      setLocalCopyError(err instanceof Error ? err.message : "Copy failed");
    }
  };

  const displayError = copyError || localCopyError;

  return (
    <div className="flex flex-wrap items-center gap-3 border-b border-white/10 bg-black/20 px-4 py-3">
      {/* Scenario selector */}
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-1.5 text-xs text-white/50">
          <FileCode className="h-3.5 w-3.5" />
          <span>Scenario</span>
          <Tooltip content="Select which scenario to generate a deployment manifest for" />
        </div>
        {availableScenarios.length > 0 ? (
          <select
            value={scenario}
            onChange={(e) => onScenarioChange(e.target.value)}
            className="rounded-xl border border-white/10 bg-slate-800 px-3 py-1.5 text-sm text-white [&_option]:bg-slate-800 [&_option]:text-white"
          >
            <option value="">Select scenario...</option>
            {availableScenarios.map((s) => (
              <option key={s.name} value={s.name}>
                {s.display_name || s.name}
              </option>
            ))}
          </select>
        ) : (
          <input
            type="text"
            value={scenario}
            onChange={(e) => onScenarioChange(e.target.value)}
            placeholder="Enter scenario name..."
            className="w-40 rounded-xl border border-white/10 bg-slate-800 px-3 py-1.5 text-sm text-white placeholder:text-white/40"
          />
        )}
      </div>

      {/* Tier selector */}
      <div className="flex items-center gap-2">
        <div className="flex items-center gap-1.5 text-xs text-white/50">
          <Layers className="h-3.5 w-3.5" />
          <span>Tier</span>
          <Tooltip content="Deployment tier determines which handling strategies are used. Each tier has different secret requirements." />
        </div>
        <select
          value={tier}
          onChange={(e) => onTierChange(e.target.value)}
          className="rounded-xl border border-white/10 bg-slate-800 px-3 py-1.5 text-sm text-white [&_option]:bg-slate-800 [&_option]:text-white"
        >
          {availableTiers.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>

      <div className="h-6 w-px bg-white/10" />

      {/* Search */}
      <div className="relative flex-1 max-w-[200px]">
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search secrets..."
          className="w-full rounded-xl border border-white/10 bg-white/5 px-3 py-1.5 pr-9 text-sm text-white placeholder:text-white/40"
        />
        <Search className="pointer-events-none absolute right-3 top-2 h-4 w-4 text-white/40" />
      </div>

      {/* Filter */}
      <div className="flex items-center gap-1 rounded-xl border border-white/10 bg-white/5 px-1 py-1">
        <Filter className="h-4 w-4 text-white/40 ml-2" />
        <select
          value={filter}
          onChange={(e) => onFilterChange(e.target.value as FilterMode)}
          className="bg-transparent text-xs text-white/80 focus:outline-none [&_option]:bg-slate-800 [&_option]:text-white"
        >
          {FILTER_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
      </div>

      {/* Actions */}
      <div className="ml-auto flex items-center gap-2">
        {onCopyFromTier && otherTiers.length > 0 && (
          <div className="relative">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowCopyDropdown(!showCopyDropdown)}
              disabled={isCopying || !scenario}
              className="gap-1.5"
            >
              <Copy className="h-3.5 w-3.5" />
              {isCopying ? "Copying..." : "Copy from tier"}
            </Button>
            {showCopyDropdown && (
              <div className="absolute right-0 top-full mt-1 z-50 min-w-[200px] rounded-xl border border-white/10 bg-slate-800 py-1 shadow-lg">
                <p className="px-3 py-1.5 text-[10px] uppercase tracking-wider text-white/50">
                  Copy overrides from:
                </p>
                {otherTiers.map((t) => (
                  <button
                    key={t.value}
                    onClick={() => handleCopyFromTier(t.value)}
                    className="w-full px-3 py-1.5 text-left text-xs text-white/80 hover:bg-white/10"
                  >
                    {t.label}
                  </button>
                ))}
              </div>
            )}
            {displayError && (
              <div className="absolute right-0 top-full mt-1 z-50 flex items-center gap-1.5 rounded-lg border border-red-500/30 bg-red-500/10 px-2 py-1 text-xs text-red-200">
                <AlertTriangle className="h-3 w-3" />
                {displayError}
              </div>
            )}
          </div>
        )}
        {hasPendingChanges && (
          <Button variant="secondary" size="sm" onClick={onSaveAll} disabled={isSaving} className="gap-1.5">
            <Save className="h-3.5 w-3.5" />
            {isSaving ? "Saving..." : "Save All"}
          </Button>
        )}
        <Button variant="outline" size="sm" onClick={onExport} disabled={!scenario} className="gap-1.5">
          <Download className="h-3.5 w-3.5" />
          Export
        </Button>
      </div>
    </div>
  );
}

// Simple inline tooltip
function Tooltip({ content }: { content: string }) {
  return (
    <div className="group relative inline-flex">
      <HelpCircle className="h-3 w-3 text-white/30 cursor-help" />
      <div className="pointer-events-none absolute left-full top-1/2 z-50 ml-2 w-48 -translate-y-1/2 rounded-lg border border-white/10 bg-slate-900 px-2 py-1.5 text-[10px] text-white/70 opacity-0 shadow-lg transition-opacity group-hover:opacity-100">
        {content}
      </div>
    </div>
  );
}
