/**
 * SettingsDrawer - Floating graph controls with secondary Settings/Prompts tabs.
 */

import { lazy, Suspense, useMemo, useState } from "react";
import {
  ChevronRight,
  Eye,
  FolderTree,
  Gamepad2,
  LayoutGrid,
  Map as MapIcon,
  Maximize2,
  RefreshCw,
  RotateCcw,
  Rows3,
} from "lucide-react";
import { FloatingPanel } from "../../../components/ui/floating-panel";
import { cn } from "../../../lib/utils";
import { useGraphDataStore } from "../stores/graph-data-store";
import { useGraphSettingsStore } from "../stores/graph-settings-store";
import { useGraphUIStore } from "../stores/graph-ui-store";
import { ENTITY_REGISTRY, GRAPH_ENTITY_TYPES } from "../lib/entity-shapes";
import { ENTITY_STATUS_REGISTRY, getGraphNodeEntityType, getGraphNodeStatus } from "../types";
import type { GraphEntityType, GraphGroupingMode } from "../types";
import type { EntityType } from "../stores/graph-settings-store";
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

/** Derived from ENTITY_REGISTRY — no hand-maintained list needed. */
const ENTITY_META: Array<{ type: EntityType; label: string; icon: React.ElementType }> =
  GRAPH_ENTITY_TYPES.map((et) => ({
    type: et,
    label: ENTITY_REGISTRY[et].label,
    icon: ENTITY_REGISTRY[et].icon,
  }));

function GraphControlsContent() {
  const lens = useGraphDataStore((s) => s.lens);
  const nodes = useGraphDataStore((s) => s.nodes);
  const loading = useGraphDataStore((s) => s.loading);
  const error = useGraphDataStore((s) => s.error);
  const fetchGraph = useGraphDataStore((s) => s.fetchGraph);
  const settings = useGraphSettingsStore((s) => s.settingsByLens[s.activeLens]);
  const setEntityFilter = useGraphSettingsStore((s) => s.setEntityFilter);
  const setStatusVisibility = useGraphSettingsStore((s) => s.setStatusVisibility);
  const clearStatusFilter = useGraphSettingsStore((s) => s.clearStatusFilter);
  const setEntityStatusGroupVisibility = useGraphSettingsStore((s) => s.setEntityStatusGroupVisibility);
  const setGroupingMode = useGraphSettingsStore((s) => s.setGroupingMode);
  const setShowSecondaryEdges = useGraphSettingsStore((s) => s.setShowSecondaryEdges);
  const setShowMiniMap = useGraphSettingsStore((s) => s.setShowMiniMap);
  const setShowNavControls = useGraphSettingsStore((s) => s.setShowNavControls);
  const setAutoFitOnChange = useGraphSettingsStore((s) => s.setAutoFitOnChange);
  const setHighlightActionableNodes = useGraphSettingsStore((s) => s.setHighlightActionableNodes);
  const resetLensSettings = useGraphSettingsStore((s) => s.resetLensSettings);

  const layoutMode = useGraphUIStore((s) => s.layoutMode);
  const layoutDirection = useGraphUIStore((s) => s.layoutDirection);
  const setLayoutForLens = useGraphUIStore((s) => s.setLayoutForLens);
  const setLayoutDirection = useGraphUIStore((s) => s.setLayoutDirection);
  const requestFitView = useGraphUIStore((s) => s.requestFitView);
  const collapseAllTopologyClusters = useGraphUIStore((s) => s.collapseAllTopologyClusters);

  interface StatusGroup {
    entityType: GraphEntityType;
    label: string;
    statuses: string[];
  }

  const statusGroups = useMemo<StatusGroup[]>(() => {
    const groups: StatusGroup[] = [];
    const entityLabels: Record<string, string> = {};
    for (const meta of ENTITY_META) {
      entityLabels[meta.type] = meta.label;
    }

    // Entity types with known registries
    for (const meta of ENTITY_META) {
      const knownStatuses = ENTITY_STATUS_REGISTRY[meta.type];
      if (knownStatuses) {
        groups.push({
          entityType: meta.type,
          label: meta.label,
          statuses: [...knownStatuses],
        });
      }
    }

    // Discover statuses for entity types not in registry (e.g. initiative)
    const discoveredByEntity = new Map<GraphEntityType, Set<string>>();
    for (const node of nodes) {
      const entityType = getGraphNodeEntityType(node);
      if (ENTITY_STATUS_REGISTRY[entityType]) continue;
      const status = getGraphNodeStatus(node);
      if (typeof status === "string" && status.trim()) {
        let set = discoveredByEntity.get(entityType);
        if (!set) {
          set = new Set();
          discoveredByEntity.set(entityType, set);
        }
        set.add(status);
      }
    }
    for (const [entityType, statuses] of discoveredByEntity) {
      groups.push({
        entityType,
        label: entityLabels[entityType] ?? entityType,
        statuses: [...statuses].sort(),
      });
    }

    return groups;
  }, [nodes]);

  const nodeCountsByEntity = useMemo(() => {
    const counts: Partial<Record<GraphEntityType, number>> = {};
    for (const node of nodes) {
      const entityType = getGraphNodeEntityType(node);
      counts[entityType] = (counts[entityType] ?? 0) + 1;
    }
    return counts;
  }, [nodes]);

  const hasCustomStatusFilters = Object.values(settings.statusFilters).some(
    (group) => Object.values(group).some((v) => !v),
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

      {statusGroups.length > 0 && (
        <section>
          <div className="mb-3 flex items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-medium text-slate-100">Statuses</h3>
              <p className="text-xs text-slate-500">Hide noisy states per entity type without changing the underlying graph.</p>
            </div>
            {hasCustomStatusFilters && (
              <button
                type="button"
                onClick={() => {
                  for (const group of statusGroups) {
                    for (const status of group.statuses) {
                      clearStatusFilter(group.entityType, status);
                    }
                  }
                }}
                className="text-xs font-medium text-slate-400 hover:text-slate-200"
              >
                Reset
              </button>
            )}
          </div>
          <div className="space-y-2">
            {statusGroups.map((group) => {
              const entityGroup = settings.statusFilters[group.entityType] ?? {};
              const allVisible = group.statuses.every((s) => entityGroup[s] !== false);
              const entityHidden = !settings.entityFilters[group.entityType];
              return (
                <StatusGroupAccordion
                  key={group.entityType}
                  group={group}
                  entityGroup={entityGroup}
                  allVisible={allVisible}
                  defaultOpen={!entityHidden}
                  nodeCount={nodeCountsByEntity[group.entityType] ?? 0}
                  onToggleAll={() => {
                    if (allVisible) {
                      setEntityStatusGroupVisibility(group.entityType, group.statuses, false);
                    } else {
                      // Clear all filters for this group to restore default (show all)
                      for (const status of group.statuses) {
                        clearStatusFilter(group.entityType, status);
                      }
                    }
                  }}
                  onToggleStatus={(status) => {
                    const current = entityGroup[status];
                    if (current === false) {
                      clearStatusFilter(group.entityType, status);
                    } else {
                      setStatusVisibility(group.entityType, status, false);
                    }
                  }}
                />
              );
            })}
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
          {renderToggleButton(
            "Show Mini Map",
            settings.showMiniMap,
            () => setShowMiniMap(!settings.showMiniMap),
            MapIcon,
          )}
          {renderToggleButton(
            "Show Nav Controls",
            settings.showNavControls,
            () => setShowNavControls(!settings.showNavControls),
            Gamepad2,
          )}
          {lens !== "focus" && renderToggleButton(
            "Highlight Actionable Nodes",
            settings.highlightActionableNodes,
            () => setHighlightActionableNodes(!settings.highlightActionableNodes),
            Eye,
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

function StatusGroupAccordion({
  group,
  entityGroup,
  allVisible,
  defaultOpen,
  nodeCount,
  onToggleAll,
  onToggleStatus,
}: {
  group: { entityType: string; label: string; statuses: string[] };
  entityGroup: Record<string, boolean>;
  allVisible: boolean;
  defaultOpen: boolean;
  nodeCount: number;
  onToggleAll: () => void;
  onToggleStatus: (status: string) => void;
}) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div className="rounded-lg border border-slate-700/50 bg-slate-900/40">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between px-3 py-2 text-left"
      >
        <span className="flex items-center gap-2">
          <ChevronRight
            className={cn(
              "h-3.5 w-3.5 text-slate-400 transition-transform",
              open && "rotate-90",
            )}
          />
          <span className="text-xs font-medium text-slate-200">{group.label}</span>
          <span className="text-[10px] text-slate-500">({nodeCount})</span>
        </span>
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onToggleAll();
          }}
          className="text-[10px] font-medium text-slate-500 hover:text-slate-300"
        >
          {allVisible ? "Hide all" : "Show all"}
        </button>
      </button>
      {open && (
        <div className="flex flex-wrap gap-1.5 px-3 pb-2.5">
          {group.statuses.map((status) => (
            <button
              key={status}
              type="button"
              onClick={() => onToggleStatus(status)}
              className={cn(
                "rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors",
                entityGroup[status] !== false
                  ? "border-cyan-500/40 bg-cyan-500/15 text-cyan-200"
                  : "border-slate-700/70 bg-slate-900/60 text-slate-400 hover:border-slate-600 hover:text-slate-200",
              )}
            >
              {status.replace(/_/g, " ")}
            </button>
          ))}
        </div>
      )}
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
