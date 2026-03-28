/**
 * SettingsDrawer - Floating graph controls with secondary Settings/Prompts tabs.
 */

import { lazy, Suspense, useMemo, useState } from "react";
import {
  Activity,
  Eye,
  FolderTree,
  LayoutGrid,
  Lightbulb,
  Maximize2,
  MessageSquare,
  Package,
  RefreshCw,
  RotateCcw,
  Rows3,
  Target,
  Zap,
} from "lucide-react";
import { FloatingPanel } from "../../../components/ui/floating-panel";
import { cn } from "../../../lib/utils";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import type { EntityType, GraphGroupingMode } from "../stores/graph-data-store";
import type { LayoutMode } from "../stores/graph-ui-store";

const SettingsPage = lazy(() =>
  import("../../../pages/SettingsPage").then((module) => ({ default: module.SettingsPage })),
);
const PromptsPage = lazy(() =>
  import("../../../pages/PromptsPage").then((module) => ({ default: module.PromptsPage })),
);

type DrawerTab = "graph" | "settings" | "prompts";

interface SettingsDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

const ENTITY_META: Array<{ type: EntityType; label: string; icon: React.ElementType }> = [
  { type: "backlog", label: "Backlog", icon: Lightbulb },
  { type: "scenario", label: "Scenarios", icon: Package },
  { type: "execution", label: "Execution", icon: Zap },
  { type: "capture", label: "Captures", icon: MessageSquare },
  { type: "agent-run", label: "Runs", icon: Activity },
  { type: "initiative", label: "Initiatives", icon: Target },
];

function GraphControlsContent() {
  const lens = useGraphDataStore((s) => s.lens);
  const nodes = useGraphDataStore((s) => s.nodes);
  const loading = useGraphDataStore((s) => s.loading);
  const error = useGraphDataStore((s) => s.error);
  const fetchGraph = useGraphDataStore((s) => s.fetchGraph);
  const settings = useGraphDataStore((s) => s.settingsByLens[s.lens]);
  const setEntityFilter = useGraphDataStore((s) => s.setEntityFilter);
  const setStatusVisibility = useGraphDataStore((s) => s.setStatusVisibility);
  const clearStatusFilter = useGraphDataStore((s) => s.clearStatusFilter);
  const setGroupingMode = useGraphDataStore((s) => s.setGroupingMode);
  const setShowSecondaryEdges = useGraphDataStore((s) => s.setShowSecondaryEdges);
  const setAutoFitOnChange = useGraphDataStore((s) => s.setAutoFitOnChange);
  const resetLensSettings = useGraphDataStore((s) => s.resetLensSettings);

  const layoutMode = useGraphUIStore((s) => s.layoutMode);
  const layoutDirection = useGraphUIStore((s) => s.layoutDirection);
  const setLayoutForLens = useGraphUIStore((s) => s.setLayoutForLens);
  const setLayoutDirection = useGraphUIStore((s) => s.setLayoutDirection);
  const requestFitView = useGraphUIStore((s) => s.requestFitView);
  const collapseAllTopologyClusters = useGraphUIStore((s) => s.collapseAllTopologyClusters);

  const availableStatuses = useMemo(() => {
    const statuses = new Set<string>();
    for (const node of nodes) {
      const status = (node.data as Record<string, unknown> | undefined)?.status;
      if (typeof status === "string" && status.trim()) {
        statuses.add(status);
      }
    }
    return [...statuses].sort((left, right) => left.localeCompare(right));
  }, [nodes]);

  const hasCustomStatusFilters = availableStatuses.some(
    (status) => settings.statusFilters[status] === false,
  );

  const resetCurrentLens = () => {
    resetLensSettings(lens);
    if (lens === "topology") {
      collapseAllTopologyClusters();
    }
    requestFitView();
  };

  const renderToggleButton = (
    label: string,
    active: boolean,
    onClick: () => void,
    icon?: React.ElementType,
  ) => {
    const Icon = icon;
    return (
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "flex items-center justify-center gap-2 rounded-lg border px-3 py-2 text-sm font-medium transition-colors",
          active
            ? "border-cyan-500/40 bg-cyan-500/15 text-cyan-200"
            : "border-slate-700/70 bg-slate-900/60 text-slate-400 hover:border-slate-600 hover:text-slate-200",
        )}
      >
        {Icon ? <Icon className="h-4 w-4" /> : null}
        {label}
      </button>
    );
  };

  const renderPillButton = (
    label: string,
    active: boolean,
    onClick: () => void,
  ) => {
    return (
      <button
        type="button"
        onClick={onClick}
        className={cn(
          "rounded-full border px-2.5 py-1 text-xs font-medium transition-colors",
          active
            ? "border-cyan-500/40 bg-cyan-500/15 text-cyan-200"
            : "border-slate-700/70 bg-slate-900/60 text-slate-400 hover:border-slate-600 hover:text-slate-200",
        )}
      >
        {label}
      </button>
    );
  };

  return (
    <div className="space-y-6">
      <section>
        <div className="mb-3 flex items-center justify-between">
          <div>
            <h3 className="text-sm font-medium text-slate-100">Visibility</h3>
            <p className="text-xs text-slate-500">Choose which entity types appear in the current lens.</p>
          </div>
          <span className="rounded-full bg-slate-800 px-2 py-0.5 text-[11px] text-slate-400">
            {lens}
          </span>
        </div>
        <div className="grid grid-cols-2 gap-2">
          {ENTITY_META.map(({ type, label, icon }) =>
            renderToggleButton(label, settings.entityFilters[type], () => setEntityFilter(type, !settings.entityFilters[type]), icon),
          )}
        </div>
      </section>

      {availableStatuses.length > 0 && (
        <section>
          <div className="mb-3 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-medium text-slate-100">Statuses</h3>
              <p className="text-xs text-slate-500">Hide noisy states without changing the underlying graph.</p>
            </div>
            {hasCustomStatusFilters && (
              <button
                type="button"
                onClick={() => {
                  for (const status of availableStatuses) {
                    clearStatusFilter(status);
                  }
                }}
                className="text-xs font-medium text-slate-400 hover:text-slate-200"
              >
                Reset
              </button>
            )}
          </div>
          <div className="flex flex-wrap gap-2">
            {availableStatuses.map((status) =>
              renderPillButton(
                status.replace(/_/g, " "),
                settings.statusFilters[status] !== false,
                () => setStatusVisibility(status, settings.statusFilters[status] === false),
              ),
            )}
          </div>
        </section>
      )}

      {lens === "topology" && (
        <section>
          <div className="mb-3">
            <h3 className="text-sm font-medium text-slate-100">Grouping</h3>
            <p className="text-xs text-slate-500">Flat view shows every backlog item. Initiative view compresses backlog under initiative buckets.</p>
          </div>
          <div className="grid grid-cols-2 gap-2">
            {([
              { id: "none", label: "Flat Graph", icon: Rows3 },
              { id: "initiative", label: "Compact by Initiative", icon: FolderTree },
            ] as const).map((option) =>
              renderToggleButton(
                option.label,
                settings.groupingMode === option.id,
                () => setGroupingMode(option.id as GraphGroupingMode),
                option.icon,
              ),
            )}
          </div>
        </section>
      )}

      <section>
        <div className="mb-3">
          <h3 className="text-sm font-medium text-slate-100">Edges</h3>
          <p className="text-xs text-slate-500">Reduce clutter without changing the entity set.</p>
        </div>
        <div className="grid grid-cols-1 gap-2">
          {renderToggleButton(
            "Show Secondary Edges",
            settings.showSecondaryEdges,
            () => setShowSecondaryEdges(!settings.showSecondaryEdges),
            Eye,
          )}
          {renderToggleButton(
            "Auto-fit After Changes",
            settings.autoFitOnChange,
            () => setAutoFitOnChange(!settings.autoFitOnChange),
            Maximize2,
          )}
        </div>
      </section>

      <section>
        <div className="mb-3">
          <h3 className="text-sm font-medium text-slate-100">Layout</h3>
          <p className="text-xs text-slate-500">Persisted per lens so each view can keep its own structure.</p>
        </div>
        <div className="grid grid-cols-3 gap-2">
          {([
            { id: "hierarchical", label: "Hier" },
            { id: "compact", label: "Compact" },
            { id: "grouped", label: "Grouped" },
          ] as const).map((option) =>
            renderToggleButton(
              option.label,
              layoutMode === option.id,
              () => setLayoutForLens(lens, option.id as LayoutMode),
            ),
          )}
        </div>
        <div className="mt-3 grid grid-cols-2 gap-2">
          {renderToggleButton("Top to Bottom", layoutDirection === "TB", () => setLayoutDirection("TB"), LayoutGrid)}
          {renderToggleButton("Left to Right", layoutDirection === "LR", () => setLayoutDirection("LR"), LayoutGrid)}
        </div>
      </section>

      <section>
        <div className="mb-3">
          <h3 className="text-sm font-medium text-slate-100">Actions</h3>
          <p className="text-xs text-slate-500">Operate on the current graph view without leaving the canvas.</p>
        </div>
        <div className="grid grid-cols-2 gap-2">
          {renderToggleButton("Fit to View", false, requestFitView, Maximize2)}
          {renderToggleButton("Reload Graph", false, () => void fetchGraph(lens), RefreshCw)}
          {renderToggleButton("Reset Lens", false, resetCurrentLens, RotateCcw)}
        </div>
        {(loading || error) && (
          <p className={cn("text-xs", error ? "text-red-300" : "text-slate-500")}>
            {error ?? "Refreshing graph…"}
          </p>
        )}
      </section>
    </div>
  );
}

export function SettingsDrawer({ isOpen, onClose }: SettingsDrawerProps) {
  const [activeTab, setActiveTab] = useState<DrawerTab>("graph");

  return (
    <FloatingPanel
      isOpen={isOpen}
      onClose={onClose}
      title="Graph Controls"
      className="max-w-3xl"
      testId="settings-drawer"
    >
      <div className="-mx-4 mb-4 flex border-b border-slate-700/50 px-4">
        {(["graph", "settings", "prompts"] as const).map((tab) => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            className={cn(
              "border-b-2 px-4 py-2 text-sm font-medium transition-colors",
              activeTab === tab
                ? "border-cyan-400 text-cyan-300"
                : "border-transparent text-slate-400 hover:text-slate-200",
            )}
            data-testid={`settings-drawer-tab-${tab}`}
          >
            {tab === "graph" ? "Graph" : tab === "settings" ? "Settings" : "Prompts"}
          </button>
        ))}
      </div>

      <Suspense
        fallback={
          <div className="flex items-center justify-center py-12 text-sm text-slate-400">
            Loading...
          </div>
        }
      >
        <div className={activeTab === "graph" ? "block" : "hidden"}>
          <GraphControlsContent />
        </div>
        <div className={activeTab === "settings" ? "block" : "hidden"}>
          <SettingsPage />
        </div>
        <div className={activeTab === "prompts" ? "block" : "hidden"}>
          <PromptsPage />
        </div>
      </Suspense>
    </FloatingPanel>
  );
}
