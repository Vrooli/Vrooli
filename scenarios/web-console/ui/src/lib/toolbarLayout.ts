// DOC: docs/reference/configuration.md#mobile-toolbar-layout
// DOC: docs/internal/SEAMS.md#axis-2-toolbar-keys-key-combos-p0-007
/**
 * Mobile toolbar layout engine.
 *
 * ── VOLATILE: This file is an extension point. ──
 * The toolbar's arrangement is data, not markup. Adding a control means adding
 * one entry to TOOLBAR_CONTROLS; it must not mean adding a branch to a
 * component. `layoutToolbar` is pure and DOM-free so the live toolbar and the
 * settings preview can share it — a preview that re-implements the arrangement
 * is a preview that eventually lies.
 *
 * The engine replaces flex-wrap with a *row budget*. Wrapping has no ceiling,
 * so a toolbar that outgrows its width silently eats the screen. Here the user
 * picks the ceiling and controls that do not fit move to overflow instead of
 * opening another row.
 *
 * [REQ:P0-007a] Floating Toolbar Component
 */

/** Controls the engine knows about. Open-ended so pinned shortcuts
 *  (`shortcut:<id>`) can join later without a schema change. */
export type KnownToolbarControlId =
  | "more"
  | "modifiers"
  | "special"
  | "arrows"
  | "mic"
  | "image"
  | "ai";
export type ToolbarControlId = KnownToolbarControlId | (string & {});

export type ToolbarDensity = "compact" | "standard" | "large";
export type ToolbarArrowStyle = "dpad" | "inline";
/** Where controls that miss the budget go. They are in More either way; the
 *  strip is an extra shortcut on the last row. */
export type ToolbarOverflowStyle = "strip" | "more";
export type ToolbarRowBudget = 1 | 2 | 3;
export type ToolbarPresetId = "dense" | "balanced" | "essential" | "custom";
export type ToolbarView = "terminal" | "messages";

export interface ToolbarPrefs {
  preset: ToolbarPresetId;
  density: ToolbarDensity;
  arrows: ToolbarArrowStyle;
  maxRows: ToolbarRowBudget;
  overflow: ToolbarOverflowStyle;
  /** Per-control visibility. Pinned controls ignore this. */
  enabled: Record<string, boolean>;
}

/**
 * How a control occupies space.
 *   icon   — one square button
 *   keys   — a run of text key caps sized to their labels
 *   arrows — four arrows, as a two-row D-pad or a one-row run
 *   fill   — takes the width left over on its row (splitting it when several
 *            fill controls share one row)
 */
export type ToolbarControlKind = "icon" | "keys" | "arrows" | "fill";

export interface ToolbarControlSpec {
  id: ToolbarControlId;
  kind: ToolbarControlKind;
  /** Key cap labels for `keys` controls. Width is derived from these. */
  keys?: readonly string[];
  /**
   * Pinned controls are seated before everything else and can never overflow.
   * `more` is pinned because it is the surface overflowed controls land in:
   * stranding it would strand everything it holds.
   */
  pinned?: boolean;
  /** Hosts the controls that are hidden or did not fit. */
  overflowHost?: boolean;
  /** Dropped in the messages view, where escape sequences mean nothing. */
  terminalOnly?: boolean;
}

/**
 * The registry, in priority order: when the budget runs short, controls lower
 * in this list give up their seat first. The order is user-visible — the
 * settings checklist renders it top to bottom.
 */
export const TOOLBAR_CONTROLS: readonly ToolbarControlSpec[] = [
  { id: "more", kind: "icon", pinned: true, overflowHost: true },
  { id: "modifiers", kind: "keys", keys: ["Ctrl", "Alt", "Shift"], terminalOnly: true },
  { id: "special", kind: "keys", keys: ["Esc", "Tab", "Enter"], terminalOnly: true },
  { id: "arrows", kind: "arrows", terminalOnly: true },
  { id: "mic", kind: "fill" },
  { id: "image", kind: "icon" },
  { id: "ai", kind: "icon" },
];

export function toolbarControlSpec(id: ToolbarControlId): ToolbarControlSpec | undefined {
  return TOOLBAR_CONTROLS.find((c) => c.id === id);
}

/** Button edge length per density, in px. */
export const TOOLBAR_DENSITY_PX: Record<ToolbarDensity, number> = {
  compact: 32,
  standard: 40,
  large: 44,
};

/** WCAG 2.5.5 target size (enhanced). Below this we warn; we never block —
 *  trading target size for more controls per row is a legitimate choice. */
export const MIN_RECOMMENDED_TOUCH_TARGET_PX = 44;

/** A strip narrower than this reads as a rendering glitch, not an affordance. */
const MIN_STRIP_WIDTH_PX = 32;

/**
 * Deliberately generous: the caps render with this as their minimum width, so
 * over-estimating costs a few px of slack (absorbed by the fill control) while
 * under-estimating would clip a label. Do not tune it down without measuring
 * the widest label ("Shift") in every supported locale.
 */
const CAP_WIDTH_PER_CHAR_EM = 0.62;

export interface ToolbarMetrics {
  /** Button edge length. */
  unit: number;
  gap: number;
  padding: number;
  fontPx: number;
  capPaddingX: number;
}

export function toolbarMetrics(density: ToolbarDensity): ToolbarMetrics {
  const unit = TOOLBAR_DENSITY_PX[density];
  return {
    unit,
    gap: density === "compact" ? 3 : 4,
    padding: 6,
    fontPx: density === "compact" ? 11 : density === "standard" ? 12.5 : 13.5,
    capPaddingX: density === "compact" ? 6 : 9,
  };
}

export function capWidth(label: string, m: ToolbarMetrics): number {
  return Math.max(m.unit, Math.ceil(label.length * m.fontPx * CAP_WIDTH_PER_CHAR_EM) + m.capPaddingX * 2);
}

export function controlWidth(
  spec: ToolbarControlSpec,
  m: ToolbarMetrics,
  arrows: ToolbarArrowStyle,
): number {
  switch (spec.kind) {
    case "keys": {
      const keys = spec.keys ?? [];
      if (keys.length === 0) return m.unit;
      return keys.reduce((sum, k) => sum + capWidth(k, m), 0) + m.gap * (keys.length - 1);
    }
    case "arrows":
      // A D-pad is three wide and two tall; the inline run is four wide.
      return arrows === "dpad" ? m.unit * 3 + m.gap * 2 : m.unit * 4 + m.gap * 3;
    case "icon":
    case "fill":
    default:
      return m.unit;
  }
}

export interface ToolbarSlot {
  id: ToolbarControlId;
  spec: ToolbarControlSpec;
  /** Resolved width in px. Fill controls carry their grown width. */
  width: number;
  fill: boolean;
}

export interface ToolbarRow {
  slots: ToolbarSlot[];
  /** Width still unclaimed on this row, after fill controls have grown. */
  freeWidth: number;
}

export interface ToolbarLayout {
  metrics: ToolbarMetrics;
  rows: ToolbarRow[];
  /** Present when the arrows render as a D-pad spanning the first two rows. */
  dpad: { width: number } | null;
  /** Controls that did not fit. Always reachable through the overflow host. */
  overflow: ToolbarSlot[];
  /** The inline swipe strip, when one was requested and had room. */
  strip: { rowIndex: number; width: number } | null;
  /** Rows actually used — never more than `prefs.maxRows`. */
  rowCount: number;
  /** Rendered height of the key area, padding included. */
  keysHeightPx: number;
  /** Arrow style after budget resolution (a D-pad cannot fit a 1-row budget). */
  arrows: ToolbarArrowStyle;
}

export interface LayoutToolbarOptions {
  view?: ToolbarView;
  /** Controls whose feature is unavailable this render (no mic permission,
   *  no upload handler). Filtered before layout so they claim no width. */
  unavailable?: readonly ToolbarControlId[];
}

/**
 * Arrange the enabled controls into at most `prefs.maxRows` rows.
 *
 * Invariants (each covered by a test in `__tests__/toolbarLayout.test.ts`):
 *   1. `rowCount <= prefs.maxRows`, always.
 *   2. The pinned overflow host is seated whenever anything overflows, so
 *      overflowed controls are never unreachable.
 *   3. A control never overflows while a lower-priority control keeps a seat.
 */
export function layoutToolbar(
  prefs: ToolbarPrefs,
  availableWidth: number,
  options: LayoutToolbarOptions = {},
): ToolbarLayout {
  const view = options.view ?? "terminal";
  const unavailable = new Set(options.unavailable ?? []);
  const m = toolbarMetrics(prefs.density);
  // The messages view has no key controls to stack, so a second row would only
  // add height around a handful of actions. It gets one row whatever the
  // terminal's budget is, and its controls share that row evenly.
  const maxRows = view === "messages" ? 1 : prefs.maxRows;

  // A D-pad needs two rows. Under a one-row budget it degrades to the inline
  // run rather than quietly spending a row the user did not authorise.
  const arrows: ToolbarArrowStyle = prefs.arrows === "dpad" && maxRows >= 2 ? "dpad" : "inline";

  const candidates = TOOLBAR_CONTROLS.filter((spec) => {
    if (unavailable.has(spec.id)) return false;
    if (view === "messages" && spec.terminalOnly) return false;
    return true;
  });

  const isEnabled = (spec: ToolbarControlSpec) => spec.pinned || prefs.enabled[spec.id] !== false;

  // In the messages view every control shares the row equally — that view has
  // few enough controls that spreading reads better than left-packing.
  const kindOf = (spec: ToolbarControlSpec): ToolbarControlKind =>
    view === "messages" ? "fill" : spec.kind;

  // Seat the overflow host when it has something to hold. In the terminal view
  // it always does (the key combos); elsewhere only when a control is hidden.
  // Anything this misses is caught by the reachability invariant below.
  const anyHidden = candidates.some((spec) => !spec.pinned && !isEnabled(spec));
  const wantsOverflowHost = view === "terminal" || anyHidden;

  const inner = Math.max(0, availableWidth - m.padding * 2);
  const rows: ToolbarRow[] = [];
  for (let i = 0; i < maxRows; i += 1) rows.push({ slots: [], freeWidth: inner });

  let dpad: { width: number } | null = null;
  const arrowsSpec = candidates.find((c) => c.id === "arrows");
  const bandTop = rows[0];
  const bandBottom = rows[1];
  if (arrows === "dpad" && arrowsSpec && isEnabled(arrowsSpec) && bandTop && bandBottom) {
    const width = controlWidth(arrowsSpec, m, "dpad");
    dpad = { width };
    // The D-pad claims a column on the trailing edge of the two-row band.
    bandTop.freeWidth -= width + m.gap;
    bandBottom.freeWidth -= width + m.gap;
  }

  const firstRow = rows[0];
  const overflow: ToolbarSlot[] = [];
  const fillSeats: { slot: ToolbarSlot; rowIndex: number }[] = [];

  const seat = (row: ToolbarRow, slot: ToolbarSlot, rowIndex: number) => {
    row.slots.push(slot);
    row.freeWidth -= slot.width + m.gap;
    if (slot.fill) fillSeats.push({ slot, rowIndex });
  };

  /**
   * One walk in priority order, pinned controls first.
   *
   * A fill control reserves a seat at its own rank rather than being placed
   * last. Placing it last is the tempting simplification and it is wrong: the
   * mic would overflow while a lower-priority icon kept a seat. It reserves its
   * minimum in the roomiest row, then grows into what that row has left.
   */
  for (const pinnedPass of [true, false]) {
    for (const spec of candidates) {
      if (!!spec.pinned !== pinnedPass) continue;
      if (spec.pinned && !wantsOverflowHost) continue;
      if (!spec.pinned && !isEnabled(spec)) continue;
      // The D-pad occupies its reserved column, not a seat in the flow.
      if (spec.id === "arrows" && dpad) continue;

      const fill = kindOf(spec) === "fill";
      const width = fill ? m.unit : controlWidth(spec, m, arrows);

      if (fill) {
        let best: { row: ToolbarRow; index: number } | null = null;
        rows.forEach((row, index) => {
          if (row.freeWidth < width) return;
          if (!best || row.freeWidth > best.row.freeWidth) best = { row, index };
        });
        const chosen = best as { row: ToolbarRow; index: number } | null;
        if (!chosen) {
          overflow.push({ id: spec.id, spec, width, fill: true });
          continue;
        }
        seat(chosen.row, { id: spec.id, spec, width, fill: true }, chosen.index);
        continue;
      }

      const rowIndex = rows.findIndex((row) => row.freeWidth >= width);
      const target = rowIndex >= 0 ? rows[rowIndex] : undefined;
      if (target) {
        seat(target, { id: spec.id, spec, width, fill: false }, rowIndex);
      } else if (spec.pinned && firstRow) {
        // A pinned control is never dropped. It takes the front of row one and
        // the row runs tight rather than losing the escape hatch.
        firstRow.slots.unshift({ id: spec.id, spec, width, fill: false });
        firstRow.freeWidth = Math.max(0, firstRow.freeWidth - width - m.gap);
      } else {
        overflow.push({ id: spec.id, spec, width, fill: false });
      }
    }
  }

  let rowCount = 0;
  rows.forEach((row, index) => {
    if (row.slots.length > 0) rowCount = index + 1;
  });
  if (dpad) rowCount = Math.max(rowCount, 2);

  // Reachability invariant: if something overflowed and no host is on the
  // surface, seat one — evicting the lowest-priority seated control if that is
  // what it takes. Overflow that cannot be opened is worse than a lost seat.
  const hostSpec = candidates.find((c) => c.overflowHost);
  const hostSeated = rows.some((row) => row.slots.some((s) => s.spec.overflowHost));
  if (overflow.length > 0 && hostSpec && !hostSeated) {
    const width = controlWidth(hostSpec, m, arrows);
    let index = rows.findIndex((row, i) => i < Math.max(rowCount, 1) && row.freeWidth >= width);
    if (index < 0) index = Math.max(0, rowCount - 1);
    const host = rows[index];
    if (host) {
      if (host.freeWidth < width) {
        const evicted = host.slots.pop();
        if (evicted) {
          host.freeWidth += evicted.width + m.gap;
          if (evicted.fill) {
            const at = fillSeats.findIndex((f) => f.slot === evicted);
            if (at >= 0) fillSeats.splice(at, 1);
          }
          overflow.unshift(evicted);
        }
      }
      if (host.freeWidth >= width) {
        host.slots.unshift({ id: hostSpec.id, spec: hostSpec, width, fill: false });
        host.freeWidth -= width + m.gap;
        rowCount = Math.max(rowCount, index + 1);
      }
    }
  }

  /**
   * The strip claims its width BEFORE fill controls grow, because both want the
   * same leftover. Settling it afterwards means evicting a control that was
   * legitimately seated.
   */
  let strip: { rowIndex: number; width: number } | null = null;
  if (overflow.length > 0 && prefs.overflow === "strip" && rowCount > 0) {
    const rowIndex = rowCount - 1;
    const row = rows[rowIndex];
    if (row && row.freeWidth >= MIN_STRIP_WIDTH_PX) {
      strip = { rowIndex, width: row.freeWidth };
      row.freeWidth = 0;
    }
  }

  // Grow the reserved seats into what their row has left, split evenly when
  // several share a row.
  rows.forEach((row, index) => {
    const seats = fillSeats.filter((f) => f.rowIndex === index);
    if (seats.length === 0) return;
    const share = Math.floor(row.freeWidth / seats.length);
    for (const s of seats) s.slot.width += share;
    row.freeWidth -= share * seats.length;
  });

  // A fill control belongs at the end of its row whatever order it was seated
  // in. `sort` is stable, so the rest keep their priority order.
  for (const row of rows) row.slots.sort((a, b) => Number(a.fill) - Number(b.fill));

  const used = rows.slice(0, rowCount);
  const keysHeightPx =
    rowCount === 0
      ? 0
      : rowCount * m.unit + (rowCount - 1) * m.gap + m.padding * 2;

  return { metrics: m, rows: used, dpad, overflow, strip, rowCount, keysHeightPx, arrows };
}

/**
 * Presets are starting points, not modes: each one is a complete `ToolbarPrefs`
 * value, and changing any single control switches the preset to "custom"
 * without changing anything else. They span the real trade-off — more controls
 * versus larger targets — rather than being a size ramp.
 *
 * Defaults keep More and image upload on. More because it is the escape hatch;
 * image upload because screenshots are how coding agents are usually driven.
 * AI suggest starts off: it costs a full button and earns it for few users.
 */
export const TOOLBAR_PRESETS: Record<Exclude<ToolbarPresetId, "custom">, Omit<ToolbarPrefs, "preset">> = {
  dense: {
    density: "compact",
    arrows: "inline",
    maxRows: 2,
    overflow: "strip",
    enabled: { more: true, modifiers: true, special: true, arrows: true, mic: true, image: true, ai: false },
  },
  balanced: {
    density: "standard",
    arrows: "dpad",
    maxRows: 2,
    overflow: "strip",
    enabled: { more: true, modifiers: true, special: true, arrows: true, mic: true, image: true, ai: false },
  },
  essential: {
    density: "large",
    arrows: "dpad",
    maxRows: 2,
    overflow: "more",
    enabled: { more: true, modifiers: true, special: false, arrows: true, mic: true, image: true, ai: false },
  },
};

export const DEFAULT_TOOLBAR_PREFS: ToolbarPrefs = toolbarPrefsFromPreset("balanced");

export function toolbarPrefsFromPreset(preset: Exclude<ToolbarPresetId, "custom">): ToolbarPrefs {
  const base = TOOLBAR_PRESETS[preset];
  return { preset, ...base, enabled: { ...base.enabled } };
}

/** Normalise a persisted or partial value into a complete, safe prefs object. */
export function normalizeToolbarPrefs(value: unknown): ToolbarPrefs {
  const fallback = toolbarPrefsFromPreset("balanced");
  if (!value || typeof value !== "object") return fallback;
  const raw = value as Partial<ToolbarPrefs>;
  const density: ToolbarDensity =
    raw.density === "compact" || raw.density === "standard" || raw.density === "large"
      ? raw.density
      : fallback.density;
  const maxRows: ToolbarRowBudget =
    raw.maxRows === 1 || raw.maxRows === 2 || raw.maxRows === 3 ? raw.maxRows : fallback.maxRows;
  return {
    preset:
      raw.preset === "dense" || raw.preset === "balanced" || raw.preset === "essential" || raw.preset === "custom"
        ? raw.preset
        : fallback.preset,
    density,
    arrows: raw.arrows === "dpad" || raw.arrows === "inline" ? raw.arrows : fallback.arrows,
    maxRows,
    overflow: raw.overflow === "strip" || raw.overflow === "more" ? raw.overflow : fallback.overflow,
    // Unknown ids are dropped rather than trusted: `enabled` is persisted user
    // data and the registry is the authority on what a control is.
    enabled: TOOLBAR_CONTROLS.reduce<Record<string, boolean>>((acc, spec) => {
      const stored = raw.enabled?.[spec.id];
      acc[spec.id] = typeof stored === "boolean" ? stored : fallback.enabled[spec.id] ?? true;
      return acc;
    }, {}),
  };
}
