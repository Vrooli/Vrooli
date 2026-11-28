// The task-first customization model. Newcomers pick a job ("Recolor the board");
// the workspace then shows only the catalog that job touches, and renders the real
// relationship chain (theme → room → composition → slot → signal) instead of
// assuming the operator already holds that model in their head. Operators can drop
// to the full six-catalog surface at any time. These are pure helpers so the gate
// covers them; the orchestration lives in the (coverage-excluded) settings page.
import type { CatalogEntry, Catalogs } from "./api";

export type TaskIcon = "palette" | "layout" | "plug" | "panel" | "add-room" | "signal";

export interface SettingsTask {
  /** URL token (`/settings?task=<id>`); the path stays `/settings` so the kiosk-cycle exemption holds. */
  id: string;
  title: string;
  blurb: string;
  icon: TaskIcon;
  /** The single catalog this job edits. */
  catalog: "themes" | "rooms" | "connectors" | "signals";
  /** One-line intent shown at the top of the workspace. */
  intent: string;
  /** Seeds a brand-new entry rather than editing an existing one. */
  create?: boolean;
}

export const SETTINGS_TASKS: SettingsTask[] = [
  { id: "recolor", title: "Recolor the board", icon: "palette", catalog: "themes",
    blurb: "Change the palette, glow, and corner style. Preview it live on any room.",
    intent: "Edit a theme's paint tokens. Every room using this theme repaints." },
  { id: "rearrange", title: "Rearrange a board", icon: "layout", catalog: "rooms",
    blurb: "Swap the background scene and re-order the beats a room cycles through.",
    intent: "Choose the composition and beats that give this room its motion." },
  { id: "wire-panel", title: "Wire up a panel", icon: "panel", catalog: "rooms",
    blurb: "Bind a measurement into a slot so a scene actually draws it.",
    intent: "Bind each slot the composition declares to a shape-compatible signal." },
  { id: "add-source", title: "Add a data source", icon: "plug", catalog: "connectors", create: true,
    blurb: "Connect an upstream service so its signals can reach the board.",
    intent: "Register a connector. Disabled connectors drop cleanly off the board." },
  { id: "add-room", title: "Add a room", icon: "add-room", catalog: "rooms", create: true,
    blurb: "Create a new board view from a composition, a theme, and its signals.",
    intent: "Name a new room, then give it a composition, a theme, and bindings." },
  { id: "tune-signal", title: "Tune a signal", icon: "signal", catalog: "signals",
    blurb: "Edit a measurement's label, unit, coverage, and sample.",
    intent: "Refine how one measurement reads. Shape and source stay structural." },
];

export function taskById(id: string | null | undefined): SettingsTask | undefined {
  return id ? SETTINGS_TASKS.find((task) => task.id === id) : undefined;
}

const str = (value: unknown): string => (typeof value === "string" ? value : "");
const labelOf = (entry: CatalogEntry): string => str(entry.label) || str(entry.title) || entry.id;
const recordOf = (value: unknown): Record<string, unknown> =>
  value !== null && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {};

/** One catalog pass projects both directions of the room relationships. */
function projectRoomRelationships(catalogs: Catalogs) {
  const roomsByTheme = new Map<string, Map<string, string>>();
  const roomsBySignal = new Map<string, Map<string, string>>();
  for (const room of catalogs.rooms ?? []) {
    const label = labelOf(room);
    const themeId = str(room.theme);
    if (themeId) {
      const rooms = roomsByTheme.get(themeId) ?? new Map<string, string>();
      rooms.set(room.id, label);
      roomsByTheme.set(themeId, rooms);
    }
    const bindings = recordOf(room.bind);
    // A room can bind the same signal into multiple slots, but its back-reference
    // should list that room only once. Room IDs, rather than labels, are the key:
    // two rooms are allowed to share a display name.
    for (const signalId of new Set(Object.values(bindings).filter((value): value is string => typeof value === "string" && value.length > 0))) {
      const rooms = roomsBySignal.get(signalId) ?? new Map<string, string>();
      rooms.set(room.id, label);
      roomsBySignal.set(signalId, rooms);
    }
  }
  return {
    roomsUsingTheme: (themeId: string) => [...(roomsByTheme.get(themeId)?.values() ?? [])],
    roomsBindingSignal: (signalId: string) => [...(roomsBySignal.get(signalId)?.values() ?? [])],
  };
}

export type NodeTone = "ok" | "warn" | "muted";
export interface ChainNode {
  /** Primary text on the node chip. */
  label: string;
  /** Small caption under it — usually the layer it belongs to. */
  kind: string;
  tone: NodeTone;
}

/** Count the rooms whose bind map points any slot at `signalId`. */
export function roomsBindingSignal(signalId: string, catalogs: Catalogs): string[] {
  return projectRoomRelationships(catalogs).roomsBindingSignal(signalId);
}

/** Count the rooms painted by `themeId`. */
export function roomsUsingTheme(themeId: string, catalogs: Catalogs): string[] {
  return projectRoomRelationships(catalogs).roomsUsingTheme(themeId);
}

/**
 * The real relationship chain for one entry, resolved against the live catalogs.
 * This is what makes the surface intelligible: the model is shown, never assumed.
 */
export function describeEntry(catalog: string, entry: CatalogEntry | undefined, catalogs: Catalogs): ChainNode[] {
  if (!entry) return [];
  if (catalog === "rooms") {
    const themeId = str(entry.theme);
    const compositionId = str(entry.composition);
    const composition = (catalogs.compositions ?? []).find((item) => item.id === compositionId);
    const slots = recordOf(composition?.slots);
    const bind = recordOf(entry.bind);
    const themes = catalogs.themes ?? [];
    const themeExists = themes.some((theme) => theme.id === themeId);
    const signalsById = new Map((catalogs.signals ?? []).map((signal) => [signal.id, signal]));
    const nodes: ChainNode[] = [
      { label: !themeId ? "no theme" : themeExists ? themeId : `missing theme: ${themeId}`, kind: "theme", tone: themeExists ? "ok" : "warn" },
      { label: labelOf(entry), kind: "room", tone: "ok" },
      { label: !compositionId ? "no composition" : composition ? compositionId : `missing composition: ${compositionId}`, kind: "composition", tone: composition ? "ok" : "warn" },
    ];
    for (const slot of Object.keys(slots)) {
      const boundId = str(bind[slot]);
      const signal = signalsById.get(boundId);
      const expectedShape = str(recordOf(slots[slot]).shape);
      const compatible = !expectedShape || signal?.shape === expectedShape;
      nodes.push(boundId
        ? { label: signal ? labelOf(signal) : `missing signal: ${boundId}`, kind: `${slot} slot`, tone: !signal || !compatible ? "warn" : "ok" }
        : { label: "unbound", kind: `${slot} slot`, tone: "muted" });
    }
    return nodes;
  }
  if (catalog === "themes") {
    const rooms = projectRoomRelationships(catalogs).roomsUsingTheme(entry.id);
    return [
      { label: labelOf(entry), kind: "theme", tone: "ok" },
      { label: rooms.length ? `paints ${rooms.length} room${rooms.length === 1 ? "" : "s"}` : "paints no rooms yet", kind: rooms.join(", ") || "unused", tone: rooms.length ? "ok" : "muted" },
    ];
  }
  if (catalog === "signals") {
    const rooms = projectRoomRelationships(catalogs).roomsBindingSignal(entry.id);
    return [
      { label: labelOf(entry), kind: "signal", tone: "ok" },
      { label: str(entry.shape) || "unknown shape", kind: "shape", tone: str(entry.shape) ? "ok" : "warn" },
      { label: str(entry.coverage) || "NOW", kind: "coverage", tone: "ok" },
      { label: rooms.length ? `bound in ${rooms.length} room${rooms.length === 1 ? "" : "s"}` : "bound in no rooms", kind: rooms.join(", ") || "unbound", tone: rooms.length ? "ok" : "muted" },
    ];
  }
  if (catalog === "connectors") {
    const enabled = entry.enabled !== false;
    return [
      { label: labelOf(entry), kind: "connector", tone: "ok" },
      { label: enabled ? "enabled" : "disabled", kind: "state", tone: enabled ? "ok" : "muted" },
      { label: str(entry.transport) || "http-json", kind: "transport", tone: "ok" },
      ...(entry.pack === true ? [{ label: "connector pack", kind: "registers bespoke visuals", tone: "ok" as NodeTone }] : []),
    ];
  }
  return [{ label: labelOf(entry), kind: catalog, tone: "ok" }];
}

/** A blank draft for a create task, seeded from the current catalogs' first choices. */
export function seedEntry(catalog: string, id: string, catalogs: Catalogs): CatalogEntry {
  if (catalog === "rooms") {
    return { id, title: "New room", category: "custom", theme: catalogs.themes?.[0]?.id ?? "cosmos", composition: catalogs.compositions?.[0]?.id ?? "orbital-field", metricIds: [], beats: [], bind: {} };
  }
  if (catalog === "connectors") {
    return { id, enabled: true, transport: "http-json", baseUrlEnv: "", paths: [] };
  }
  return { id };
}

/** Slugify a human title into a stable catalog id. */
export function slugify(value: string): string {
  return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 48);
}

export const CATALOG_SINGULAR: Record<string, string> = {
  rooms: "room", themes: "theme", compositions: "composition", signals: "signal", connectors: "connector", readouts: "readout",
};
