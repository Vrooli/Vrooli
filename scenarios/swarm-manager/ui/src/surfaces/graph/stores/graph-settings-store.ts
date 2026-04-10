/**
 * Graph Settings Store
 *
 * Per-lens display settings: entity/status filtering, grouping mode, and
 * boolean toggles (mini-map, nav controls, etc.). Settings are persisted to
 * localStorage and restored on load.
 */

import { create } from "zustand";
import { GRAPH_ENTITY_TYPES } from "../lib/entity-shapes";
import {
  ENTITY_STATUS_REGISTRY,
  type GraphEntityType,
  type GraphGroupingMode,
  type GraphLens,
} from "../types";

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

export type EntityType = GraphEntityType;

export interface GraphLensSettings {
  entityFilters: Record<EntityType, boolean>;
  /** Per-entity-type status visibility. Outer key = entity type, inner key = status string. */
  statusFilters: Record<string, Record<string, boolean>>;
  groupingMode: GraphGroupingMode;
  showSecondaryEdges: boolean;
  autoFitOnChange: boolean;
  showMiniMap: boolean;
  /** Show on-screen pan/zoom controls for TV and accessibility. */
  showNavControls: boolean;
  /** Pulse nodes that need attention (via computeNodeAttention). Not applicable to focus lens. */
  highlightActionableNodes: boolean;
}

export interface GraphSettingsState {
  settingsByLens: Record<GraphLens, GraphLensSettings>;
  activeLens: GraphLens;
  setActiveLens: (lens: GraphLens) => void;
  toggleEntityFilter: (type: EntityType) => void;
  setEntityFilter: (type: EntityType, visible: boolean) => void;
  setStatusVisibility: (entityType: string, status: string, visible: boolean) => void;
  clearStatusFilter: (entityType: string, status: string) => void;
  setEntityStatusGroupVisibility: (entityType: string, statuses: readonly string[], visible: boolean) => void;
  setGroupingMode: (mode: GraphGroupingMode) => void;
  setShowSecondaryEdges: (visible: boolean) => void;
  setShowMiniMap: (visible: boolean) => void;
  setShowNavControls: (visible: boolean) => void;
  setAutoFitOnChange: (enabled: boolean) => void;
  setHighlightActionableNodes: (enabled: boolean) => void;
  resetLensSettings: (lens?: GraphLens) => void;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const GRAPH_SETTINGS_STORAGE_KEY = "swarm-manager.graph.settings.v5";
const LEGACY_GRAPH_SETTINGS_STORAGE_KEYS = [
  "swarm-manager.graph.settings.v4",
  "swarm-manager.graph.settings.v3",
  "swarm-manager.graph.settings.v2",
];

// ---------------------------------------------------------------------------
// Entity filter defaults
// ---------------------------------------------------------------------------

const DEFAULT_ENTITY_FILTERS: Record<EntityType, boolean> = Object.fromEntries(
  GRAPH_ENTITY_TYPES.map((et) => [et, true]),
) as Record<EntityType, boolean>;

function cloneEntityFilters(): Record<EntityType, boolean> {
  return { ...DEFAULT_ENTITY_FILTERS };
}

const ENTITY_TYPE_SET: ReadonlySet<string> = new Set(GRAPH_ENTITY_TYPES);

function isEntityType(value: unknown): value is EntityType {
  return typeof value === "string" && ENTITY_TYPE_SET.has(value);
}

// ---------------------------------------------------------------------------
// Lens settings helpers
// ---------------------------------------------------------------------------

export function createDefaultLensSettings(lens: GraphLens): GraphLensSettings {
  return {
    entityFilters: cloneEntityFilters(),
    statusFilters: {},
    groupingMode: lens === "topology" ? "initiative" : "none",
    showSecondaryEdges: lens !== "focus",
    autoFitOnChange: true,
    showMiniMap: false,
    showNavControls: false,
    highlightActionableNodes: lens !== "focus",
  };
}

export function createDefaultSettingsByLens(): Record<GraphLens, GraphLensSettings> {
  return {
    focus: createDefaultLensSettings("focus"),
    topology: createDefaultLensSettings("topology"),
    operations: createDefaultLensSettings("operations"),
  };
}

export function cloneLensSettings(settings: GraphLensSettings): GraphLensSettings {
  return {
    entityFilters: { ...settings.entityFilters },
    statusFilters: { ...settings.statusFilters },
    groupingMode: settings.groupingMode,
    showSecondaryEdges: settings.showSecondaryEdges,
    autoFitOnChange: settings.autoFitOnChange,
    showMiniMap: settings.showMiniMap,
    showNavControls: settings.showNavControls,
    highlightActionableNodes: settings.highlightActionableNodes,
  };
}

export function cloneSettingsByLens(
  settingsByLens: Record<GraphLens, GraphLensSettings>,
): Record<GraphLens, GraphLensSettings> {
  return {
    focus: cloneLensSettings(settingsByLens.focus),
    topology: cloneLensSettings(settingsByLens.topology),
    operations: cloneLensSettings(settingsByLens.operations),
  };
}

// ---------------------------------------------------------------------------
// Persistence
// ---------------------------------------------------------------------------

export function loadPersistedSettings(): Record<GraphLens, GraphLensSettings> {
  const defaults = createDefaultSettingsByLens();

  if (typeof window === "undefined") {
    return defaults;
  }

  try {
    let raw = window.localStorage.getItem(GRAPH_SETTINGS_STORAGE_KEY);
    let migratingLegacySettings = false;
    if (!raw) {
      for (const key of LEGACY_GRAPH_SETTINGS_STORAGE_KEYS) {
        raw = window.localStorage.getItem(key);
        if (raw) {
          migratingLegacySettings = true;
          break;
        }
      }
    }
    if (!raw) return defaults;

    const parsed = JSON.parse(raw) as Record<string, unknown>;
    const next = createDefaultSettingsByLens();

    for (const lens of ["focus", "topology", "operations"] as const) {
      const lensValue = parsed[lens];
      if (typeof lensValue !== "object" || lensValue === null) {
        continue;
      }

      const record = lensValue as Record<string, unknown>;
      const entityFilters = cloneEntityFilters();
      if (typeof record.entityFilters === "object" && record.entityFilters !== null) {
        const filters = record.entityFilters as Record<string, unknown>;
        for (const [key, value] of Object.entries(filters)) {
          if (isEntityType(key) && typeof value === "boolean") {
            entityFilters[key] = value;
          }
        }
      }

      const statusFilters: Record<string, Record<string, boolean>> = {};
      if (typeof record.statusFilters === "object" && record.statusFilters !== null) {
        const rawStatuses = record.statusFilters as Record<string, unknown>;
        const firstValue = Object.values(rawStatuses)[0];

        if (typeof firstValue === "boolean") {
          // v3 flat format -- broadcast each status to all entity types that include it
          for (const [status, value] of Object.entries(rawStatuses)) {
            if (typeof value !== "boolean") continue;
            for (const [et, knownStatuses] of Object.entries(ENTITY_STATUS_REGISTRY)) {
              if (knownStatuses && (knownStatuses as readonly string[]).includes(status)) {
                statusFilters[et] = statusFilters[et] || {};
                statusFilters[et][status] = value;
              }
            }
          }
        } else {
          // v4 grouped format -- parse directly
          for (const [entityType, group] of Object.entries(rawStatuses)) {
            if (typeof group === "object" && group !== null) {
              const parsed: Record<string, boolean> = {};
              for (const [status, value] of Object.entries(group as Record<string, unknown>)) {
                if (typeof value === "boolean") parsed[status] = value;
              }
              if (Object.keys(parsed).length > 0) statusFilters[entityType] = parsed;
            }
          }
        }
      }

      next[lens] = {
        entityFilters,
        statusFilters,
        groupingMode:
          !migratingLegacySettings &&
          (record.groupingMode === "initiative" || record.groupingMode === "none")
            ? record.groupingMode
            : defaults[lens].groupingMode,
        showSecondaryEdges:
          typeof record.showSecondaryEdges === "boolean"
            ? record.showSecondaryEdges
            : defaults[lens].showSecondaryEdges,
        autoFitOnChange:
          typeof record.autoFitOnChange === "boolean"
            ? record.autoFitOnChange
            : defaults[lens].autoFitOnChange,
        showMiniMap:
          typeof record.showMiniMap === "boolean"
            ? record.showMiniMap
            : defaults[lens].showMiniMap,
        showNavControls:
          typeof record.showNavControls === "boolean"
            ? record.showNavControls
            : defaults[lens].showNavControls,
        highlightActionableNodes:
          typeof record.highlightActionableNodes === "boolean"
            ? record.highlightActionableNodes
            : defaults[lens].highlightActionableNodes,
      };
    }

    return next;
  } catch {
    return defaults;
  }
}

export function savePersistedSettings(settingsByLens: Record<GraphLens, GraphLensSettings>): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(GRAPH_SETTINGS_STORAGE_KEY, JSON.stringify(settingsByLens));
  } catch {
    // Ignore persistence failures and continue in-memory.
  }
}

// ---------------------------------------------------------------------------
// Internal helper
// ---------------------------------------------------------------------------

function updateLensSettings(
  state: GraphSettingsState,
  updater: (settings: GraphLensSettings) => GraphLensSettings,
): Pick<GraphSettingsState, "settingsByLens"> {
  const lens = state.activeLens;
  const nextSettings = {
    ...state.settingsByLens,
    [lens]: updater(state.settingsByLens[lens]),
  };
  savePersistedSettings(nextSettings);
  return { settingsByLens: nextSettings };
}

// ---------------------------------------------------------------------------
// Store
// ---------------------------------------------------------------------------

export function createGraphSettingsInitialState() {
  return {
    settingsByLens:
      typeof window !== "undefined" ? loadPersistedSettings() : createDefaultSettingsByLens(),
    activeLens: "topology" as GraphLens,
  };
}

export const graphSettingsInitialState = createGraphSettingsInitialState();

export const useGraphSettingsStore = create<GraphSettingsState>((set) => ({
  ...graphSettingsInitialState,

  setActiveLens: (lens) => set({ activeLens: lens }),

  toggleEntityFilter: (type) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        entityFilters: {
          ...settings.entityFilters,
          [type]: !settings.entityFilters[type],
        },
      })),
    ),

  setEntityFilter: (type, visible) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        entityFilters: {
          ...settings.entityFilters,
          [type]: visible,
        },
      })),
    ),

  setStatusVisibility: (entityType, status, visible) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        statusFilters: {
          ...settings.statusFilters,
          [entityType]: {
            ...settings.statusFilters[entityType],
            [status]: visible,
          },
        },
      })),
    ),

  clearStatusFilter: (entityType, status) =>
    set((state) =>
      updateLensSettings(state, (settings) => {
        const entityGroup = { ...settings.statusFilters[entityType] };
        delete entityGroup[status];
        const nextFilters = { ...settings.statusFilters };
        if (Object.keys(entityGroup).length === 0) {
          delete nextFilters[entityType];
        } else {
          nextFilters[entityType] = entityGroup;
        }
        return { ...settings, statusFilters: nextFilters };
      }),
    ),

  setEntityStatusGroupVisibility: (entityType, statuses, visible) =>
    set((state) =>
      updateLensSettings(state, (settings) => {
        const entityGroup = { ...settings.statusFilters[entityType] };
        for (const status of statuses) {
          entityGroup[status] = visible;
        }
        return {
          ...settings,
          statusFilters: {
            ...settings.statusFilters,
            [entityType]: entityGroup,
          },
        };
      }),
    ),

  setGroupingMode: (mode) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        groupingMode: mode,
      })),
    ),

  setShowSecondaryEdges: (visible) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        showSecondaryEdges: visible,
      })),
    ),

  setShowMiniMap: (visible) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        showMiniMap: visible,
      })),
    ),

  setShowNavControls: (visible) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        showNavControls: visible,
      })),
    ),

  setAutoFitOnChange: (enabled) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        autoFitOnChange: enabled,
      })),
    ),

  setHighlightActionableNodes: (enabled) =>
    set((state) =>
      updateLensSettings(state, (settings) => ({
        ...settings,
        highlightActionableNodes: enabled,
      })),
    ),

  resetLensSettings: (lensArg) =>
    set((state) => {
      const lens = lensArg ?? state.activeLens;
      const nextSettings = {
        ...state.settingsByLens,
        [lens]: createDefaultLensSettings(lens),
      };
      savePersistedSettings(nextSettings);
      return { settingsByLens: nextSettings };
    }),
}));

export function cloneGraphSettingsInitialState(): Pick<GraphSettingsState, "settingsByLens" | "activeLens"> {
  const initialState = createGraphSettingsInitialState();
  return {
    ...initialState,
    settingsByLens: cloneSettingsByLens(initialState.settingsByLens),
  };
}
