import { useState, useEffect } from "react";
import { Search, Filter, Copy, Save, Download, X, AlertTriangle } from "lucide-react";
import { Button } from "../../components/ui/button";
import type { FilterMode } from "./types";

interface ManifestToolbarProps {
  searchQuery: string;
  filter: FilterMode;
  hasPendingChanges: boolean;
  isSaving: boolean;
  currentTier: string;
  availableTiers: string[];
  isCopying?: boolean;
  copyError?: string | null;
  onSearchChange: (query: string) => void;
  onFilterChange: (filter: FilterMode) => void;
  onSaveAll: () => void;
  onExport: () => void;
  onClose: () => void;
  onCopyFromTier?: (sourceTier: string) => Promise<void>;
}

const FILTER_OPTIONS: { value: FilterMode; label: string }[] = [
  { value: "all", label: "All secrets" },
  { value: "blocking", label: "Blocking only" },
  { value: "overridden", label: "Overridden only" },
  { value: "excluded", label: "Excluded only" }
];

export function ManifestToolbar({
  searchQuery,
  filter,
  hasPendingChanges,
  isSaving,
  currentTier,
  availableTiers,
  isCopying,
  copyError,
  onSearchChange,
  onFilterChange,
  onSaveAll,
  onExport,
  onClose,
  onCopyFromTier
}: ManifestToolbarProps) {
  const [showCopyDropdown, setShowCopyDropdown] = useState(false);
  const [localCopyError, setLocalCopyError] = useState<string | null>(null);
  const otherTiers = availableTiers.filter((t) => t !== currentTier);

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
      // Show local error if parent doesn't provide one
      setLocalCopyError(err instanceof Error ? err.message : "Copy failed");
    }
  };

  const displayError = copyError || localCopyError;

  return (
    <div className="flex items-center gap-3 border-b border-white/10 bg-black/30 px-4 py-3">
      <div className="relative flex-1 max-w-xs">
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search secrets..."
          className="w-full rounded-xl border border-white/10 bg-white/5 px-3 py-1.5 pr-9 text-sm text-white placeholder:text-white/40"
        />
        <Search className="pointer-events-none absolute right-3 top-2 h-4 w-4 text-white/40" />
      </div>

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

      <div className="ml-auto flex items-center gap-2">
        {onCopyFromTier && otherTiers.length > 0 && (
          <div className="relative">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowCopyDropdown(!showCopyDropdown)}
              disabled={isCopying}
              className="gap-1.5"
            >
              <Copy className="h-3.5 w-3.5" />
              {isCopying ? "Copying..." : "Copy from tier"}
            </Button>
            {showCopyDropdown && (
              <div className="absolute right-0 top-full mt-1 z-50 min-w-[180px] rounded-xl border border-white/10 bg-slate-800 py-1 shadow-lg">
                <p className="px-3 py-1.5 text-[10px] uppercase tracking-wider text-white/50">
                  Copy overrides from:
                </p>
                {otherTiers.map((tier) => (
                  <button
                    key={tier}
                    onClick={() => handleCopyFromTier(tier)}
                    className="w-full px-3 py-1.5 text-left text-xs text-white/80 hover:bg-white/10"
                  >
                    {tier}
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
        <Button variant="outline" size="sm" onClick={onExport} className="gap-1.5">
          <Download className="h-3.5 w-3.5" />
          Export
        </Button>
        <Button variant="ghost" size="sm" onClick={onClose} className="text-white/60 hover:text-white">
          <X className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
