import { useState, useMemo, useCallback } from "react";
import { Search, Bot, X } from "lucide-react";
import { useRequirements } from "../../hooks/useRequirements";
import { useRequirementsImprove } from "../../hooks/useRequirementsImprove";
import { SyncStatusBanner } from "./SyncStatusBanner";
import { CoverageStats } from "./CoverageStats";
import { RequirementsHelpSection } from "./RequirementsHelpSection";
import { RequirementsTree } from "./RequirementsTree";
import { ActionTypeSelector } from "./ActionTypeSelector";
import { RequirementsImproveStatusCard } from "../cards/RequirementsImproveStatusCard";
import { Button } from "../ui/button";
import { MessagePopover } from "../ui/MessagePopover";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type {
  ImproveActionType,
  RequirementImproveInfo,
  RequirementItem,
  ValidationItem
} from "../../lib/api";

interface RequirementsPanelProps {
  scenarioName: string;
}

type FilterKey = "all" | "passed" | "failed" | "not_run";

const FILTERS: Array<{ key: FilterKey; label: string }> = [
  { key: "all", label: "All" },
  { key: "passed", label: "Passed" },
  { key: "failed", label: "Failed" },
  { key: "not_run", label: "Not Run" }
];

export function RequirementsPanel({ scenarioName }: RequirementsPanelProps) {
  const [filter, setFilter] = useState<FilterKey>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [isSelectionMode, setIsSelectionMode] = useState(false);
  const [selectedRequirements, setSelectedRequirements] = useState<
    Map<string, RequirementImproveInfo>
  >(new Map());
  const [actionType, setActionType] = useState<ImproveActionType>("write_tests");
  const [improveMessage, setImproveMessage] = useState("");
  const [dismissedImproveId, setDismissedImproveId] = useState<string | null>(
    null
  );

  const {
    isLoading,
    isError,
    coverage,
    syncStatus,
    modules,
    sync,
    isSyncing,
    lastSyncSuccess
  } = useRequirements(scenarioName);

  const {
    activeImprove,
    isActive,
    spawn,
    stop,
    isSpawning,
    isStopping
  } = useRequirementsImprove(scenarioName);

  // Get improvable requirements (failed or not_run) from all modules
  const improvableRequirements = useMemo(() => {
    const result: Array<RequirementItem & { modulePath: string }> = [];
    for (const mod of modules) {
      for (const req of mod.requirements ?? []) {
        if (req.liveStatus === "failed" || req.liveStatus === "not_run") {
          result.push({ ...req, modulePath: mod.filePath });
        }
      }
    }
    return result;
  }, [modules]);

  const hasImprovableRequirements = improvableRequirements.length > 0;

  // Show improve status card if active and not dismissed
  const showImproveStatus =
    activeImprove && activeImprove.id !== dismissedImproveId;

  const enterSelectionMode = useCallback(() => {
    // Start with no selection
    setSelectedRequirements(new Map());
    setIsSelectionMode(true);
  }, []);

  const selectAll = useCallback(() => {
    const allSelection = new Map<string, RequirementImproveInfo>();
    for (const req of improvableRequirements) {
      allSelection.set(req.id, {
        id: req.id,
        title: req.title,
        description: req.description,
        status: req.status,
        liveStatus: req.liveStatus,
        criticality: req.criticality,
        modulePath: req.modulePath,
        validations: req.validations?.map((v: ValidationItem) => ({
          type: v.type,
          ref: v.ref,
          phase: v.phase,
          status: v.status,
          liveStatus: v.liveStatus
        }))
      });
    }
    setSelectedRequirements(allSelection);
  }, [improvableRequirements]);

  const deselectAll = useCallback(() => {
    setSelectedRequirements(new Map());
  }, []);

  const exitSelectionMode = useCallback(() => {
    setSelectedRequirements(new Map());
    setIsSelectionMode(false);
    setImproveMessage("");
  }, []);

  const toggleRequirementSelection = useCallback(
    (req: RequirementItem & { modulePath: string }) => {
      setSelectedRequirements((prev) => {
        const next = new Map(prev);
        if (next.has(req.id)) {
          next.delete(req.id);
        } else {
          next.set(req.id, {
            id: req.id,
            title: req.title,
            description: req.description,
            status: req.status,
            liveStatus: req.liveStatus,
            criticality: req.criticality,
            modulePath: req.modulePath,
            validations: req.validations?.map((v: ValidationItem) => ({
              type: v.type,
              ref: v.ref,
              phase: v.phase,
              status: v.status,
              liveStatus: v.liveStatus
            }))
          });
        }
        return next;
      });
    },
    []
  );

  const handleStartImprove = useCallback(async () => {
    if (selectedRequirements.size === 0) return;

    try {
      await spawn(
        Array.from(selectedRequirements.values()),
        actionType,
        improveMessage || undefined
      );
      exitSelectionMode();
      setDismissedImproveId(null);
    } catch (err) {
      console.error("Failed to spawn improve agent:", err);
    }
  }, [selectedRequirements, actionType, improveMessage, spawn, exitSelectionMode]);

  const handleStopImprove = useCallback(() => {
    if (activeImprove) {
      stop(activeImprove.id);
    }
  }, [activeImprove, stop]);

  const handleDismissImprove = useCallback(() => {
    if (activeImprove) {
      setDismissedImproveId(activeImprove.id);
    }
  }, [activeImprove]);

  if (isLoading) {
    return (
      <div className="space-y-4">
        <div className="h-12 animate-pulse rounded-xl bg-white/5" />
        <div className="grid grid-cols-4 gap-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-20 animate-pulse rounded-xl bg-white/5" />
          ))}
        </div>
        <div className="h-64 animate-pulse rounded-xl bg-white/5" />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-6 text-center">
        <p className="text-red-300">Failed to load requirements data</p>
        <p className="mt-2 text-sm text-slate-400">
          Make sure the scenario has been set up correctly and tests have been
          run.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4" data-testid={selectors.requirements.panel}>
      {/* Improve Status Card */}
      {showImproveStatus && (
        <RequirementsImproveStatusCard
          improve={activeImprove}
          onStop={handleStopImprove}
          onDismiss={handleDismissImprove}
          isStopping={isStopping}
        />
      )}

      {/* Sync Status Banner */}
      <SyncStatusBanner
        syncStatus={syncStatus}
        onSync={() => sync({})}
        isSyncing={isSyncing}
        lastSyncSuccess={lastSyncSuccess}
      />

      {/* Coverage Stats */}
      <CoverageStats coverage={coverage} />

      {/* Help Section */}
      <RequirementsHelpSection />

      {/* Filter, Search, and AI Actions Bar */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex rounded-lg border border-white/10 bg-black/30 p-1">
          {FILTERS.map((f) => (
            <button
              key={f.key}
              type="button"
              onClick={() => setFilter(f.key)}
              disabled={isSelectionMode}
              className={cn(
                "rounded-md px-3 py-1 text-sm transition",
                filter === f.key
                  ? "bg-white/10 text-white"
                  : "text-slate-400 hover:text-white",
                isSelectionMode && "cursor-not-allowed opacity-50"
              )}
              data-testid={
                selectors.requirements[
                  `filter${f.key.charAt(0).toUpperCase() + f.key.slice(1).replace("_", "")}` as keyof typeof selectors.requirements
                ]
              }
            >
              {f.label}
            </button>
          ))}
        </div>

        <div className="relative min-w-[200px] flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search requirements..."
            disabled={isSelectionMode}
            className={cn(
              "w-full rounded-lg border border-white/10 bg-black/30 py-2 pl-9 pr-3 text-sm placeholder-slate-500 focus:border-white/20 focus:outline-none",
              isSelectionMode && "cursor-not-allowed opacity-50"
            )}
            data-testid={selectors.requirements.searchInput}
          />
        </div>

        {/* AI Actions */}
        {!isSelectionMode ? (
          <Button
            variant="outline"
            size="sm"
            onClick={enterSelectionMode}
            disabled={!hasImprovableRequirements || isActive}
            className={cn(
              hasImprovableRequirements && !isActive
                ? "text-violet-400 hover:border-violet-400/50 hover:text-violet-300"
                : ""
            )}
            data-testid={selectors.requirements.improveButton}
          >
            <Bot className="mr-2 h-4 w-4" />
            Improve with AI
          </Button>
        ) : (
          <div className="flex items-center gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={exitSelectionMode}
              className="text-slate-400 hover:text-slate-300"
              data-testid={selectors.requirements.cancelImproveButton}
            >
              <X className="mr-1 h-4 w-4" />
              Cancel
            </Button>
            <Button
              variant="ghost"
              size="sm"
              onClick={
                selectedRequirements.size === improvableRequirements.length
                  ? deselectAll
                  : selectAll
              }
              className="text-slate-400 hover:text-slate-300"
              data-testid={selectors.requirements.selectAllButton}
            >
              {selectedRequirements.size === improvableRequirements.length
                ? "Deselect All"
                : "Select All"}
            </Button>
            <MessagePopover
              message={improveMessage}
              onChange={setImproveMessage}
              disabled={isSpawning}
              placeholder="Add context for the improve agent..."
              data-testid={selectors.requirements.improveMessagePopover}
            />
            <Button
              variant="outline"
              size="sm"
              onClick={handleStartImprove}
              disabled={selectedRequirements.size === 0 || isSpawning}
              className="text-violet-400 hover:border-violet-400/50 hover:text-violet-300"
              data-testid={selectors.requirements.startImproveButton}
            >
              <Bot className="mr-2 h-4 w-4" />
              Start Improve ({selectedRequirements.size})
            </Button>
          </div>
        )}
      </div>

      {/* Action Type Selector (only in selection mode) */}
      {isSelectionMode && (
        <ActionTypeSelector
          value={actionType}
          onChange={setActionType}
          disabled={isSpawning}
        />
      )}

      {/* Requirements Tree */}
      <RequirementsTree
        modules={modules}
        filter={filter}
        searchQuery={searchQuery}
        selectionMode={isSelectionMode}
        selectedRequirements={selectedRequirements}
        onToggleSelection={toggleRequirementSelection}
      />
    </div>
  );
}
