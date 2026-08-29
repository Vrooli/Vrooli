import { describe, it, expect } from "vitest";
import {
  DEFAULT_TOOLBAR_PREFS,
  MIN_RECOMMENDED_TOUCH_TARGET_PX,
  TOOLBAR_CONTROLS,
  TOOLBAR_DENSITY_PX,
  TOOLBAR_PRESETS,
  controlWidth,
  layoutToolbar,
  normalizeToolbarPrefs,
  toolbarPrefsFromPreset,
  type ToolbarLayout,
  type ToolbarPrefs,
  type ToolbarPresetId,
} from "../lib/toolbarLayout";
import { MODIFIER_KEYS, SPECIAL_KEYS, modifierLabel } from "../components/toolbar/toolbarControls";

/** Phone widths the toolbar has to survive: small Android, iPhone, Pro Max. */
const WIDTHS = [320, 360, 390, 414, 430, 768];
const PRESETS: Exclude<ToolbarPresetId, "custom">[] = ["dense", "balanced", "essential"];

function seatedIds(layout: ToolbarLayout): string[] {
  return layout.rows.flatMap((row) => row.slots.map((slot) => slot.id));
}

function prefs(overrides: Partial<ToolbarPrefs> = {}): ToolbarPrefs {
  const base = toolbarPrefsFromPreset("balanced");
  return { ...base, ...overrides, enabled: { ...base.enabled, ...(overrides.enabled ?? {}) } };
}

describe("layoutToolbar — the row budget is a ceiling", () => {
  it("never uses more rows than the budget allows, at any width or density", () => {
    for (const maxRows of [1, 2, 3] as const) {
      for (const density of ["compact", "standard", "large"] as const) {
        for (const arrows of ["dpad", "inline"] as const) {
          for (const width of WIDTHS) {
            const layout = layoutToolbar(prefs({ maxRows, density, arrows }), width);
            expect(layout.rowCount).toBeLessThanOrEqual(maxRows);
            expect(layout.rows.length).toBeLessThanOrEqual(maxRows);
          }
        }
      }
    }
  });

  it("degrades the D-pad to an inline run rather than spending an unbudgeted row", () => {
    const layout = layoutToolbar(prefs({ maxRows: 1, arrows: "dpad" }), 390);
    expect(layout.arrows).toBe("inline");
    expect(layout.dpad).toBeNull();
    expect(layout.rowCount).toBe(1);
  });

  it("keeps the D-pad when the budget can afford its two rows", () => {
    const layout = layoutToolbar(prefs({ maxRows: 2, arrows: "dpad" }), 390);
    expect(layout.arrows).toBe("dpad");
    expect(layout.dpad?.width).toBeGreaterThan(0);
    expect(layout.rowCount).toBe(2);
  });

  it("reports a height that matches the rows it produced", () => {
    const layout = layoutToolbar(prefs(), 390);
    const { unit, gap, padding } = layout.metrics;
    expect(layout.keysHeightPx).toBe(layout.rowCount * unit + (layout.rowCount - 1) * gap + padding * 2);
  });
});

describe("layoutToolbar — overflow is never unreachable", () => {
  it("seats the pinned overflow host under every hostile setting", () => {
    for (const density of ["compact", "standard", "large"] as const) {
      for (const width of [280, ...WIDTHS]) {
        const layout = layoutToolbar(prefs({ maxRows: 1, density }), width);
        expect(seatedIds(layout)).toContain("more");
        expect(layout.overflow.map((s) => s.id)).not.toContain("more");
      }
    }
  });

  it("keeps the host seated even when the user hides everything else", () => {
    const hidden = Object.fromEntries(
      TOOLBAR_CONTROLS.filter((c) => !c.pinned).map((c) => [c.id, false]),
    );
    const layout = layoutToolbar(prefs({ enabled: hidden }), 390);
    expect(seatedIds(layout)).toEqual(["more"]);
    expect(layout.overflow).toHaveLength(0);
  });

  it("gives up a seat to keep the host reachable when the row is full", () => {
    // A messages view with nothing hidden does not ask for a host up front, so
    // this is the path where overflow appears with no way to open it. The
    // engine must evict the lowest-priority control rather than leave the
    // overflowed ones stranded.
    const everything = Object.fromEntries(TOOLBAR_CONTROLS.map((c) => [c.id, true]));
    const layout = layoutToolbar(prefs({ maxRows: 1, enabled: everything }), 120, { view: "messages" });

    expect(layout.overflow.length).toBeGreaterThan(0);
    expect(seatedIds(layout)).toContain("more");
    expect(layout.overflow.map((s) => s.id)).not.toContain("more");
    // The evicted control did not vanish — it is in overflow, reachable.
    expect(seatedIds(layout).length + layout.overflow.length).toBeGreaterThanOrEqual(3);
  });

  it("does not add a host to the messages view when nothing needs one", () => {
    const everything = Object.fromEntries(TOOLBAR_CONTROLS.map((c) => [c.id, true]));
    const layout = layoutToolbar(prefs({ enabled: everything }), 390, { view: "messages" });
    expect(layout.overflow).toHaveLength(0);
    expect(seatedIds(layout)).not.toContain("more");
  });

  it("adds a host to the messages view as soon as a control is hidden", () => {
    // `ai` is off by default, so it is reachable only through More.
    const layout = layoutToolbar(prefs(), 390, { view: "messages" });
    expect(seatedIds(layout)).toContain("more");
  });
});

describe("layoutToolbar — priority order is respected", () => {
  const priority = TOOLBAR_CONTROLS.map((c) => c.id);

  it("only drops a control for something smaller and lower down the list", () => {
    // A wide control genuinely cannot fit where a narrow one can, so priority
    // alone cannot be the invariant. What must never happen is a control losing
    // its seat to a lower-priority control that is no cheaper than it.
    for (const maxRows of [1, 2, 3] as const) {
      for (const density of ["compact", "standard", "large"] as const) {
        for (const arrows of ["dpad", "inline"] as const) {
          for (const width of WIDTHS) {
            const p = prefs({ maxRows, density, arrows });
            const layout = layoutToolbar(p, width);
            const m = layout.metrics;
            const base = (id: string) => {
              const spec = TOOLBAR_CONTROLS.find((c) => c.id === id)!;
              return spec.kind === "fill" ? m.unit : controlWidth(spec, m, layout.arrows);
            };
            for (const dropped of layout.overflow) {
              const droppedRank = priority.indexOf(dropped.id);
              for (const id of seatedIds(layout)) {
                const spec = TOOLBAR_CONTROLS.find((c) => c.id === id);
                if (spec?.pinned) continue; // outranks everything by construction
                if (priority.indexOf(id) < droppedRank) continue; // higher priority, fine
                expect(
                  base(id),
                  `${id} kept a seat over the higher-priority ${dropped.id}`,
                ).toBeLessThanOrEqual(base(dropped.id));
              }
            }
          }
        }
      }
    }
  });

  it("keeps the mic seated ahead of the lower-priority image button", () => {
    // 360px with standard density and a D-pad cannot hold everything.
    const layout = layoutToolbar(prefs({ density: "standard", arrows: "dpad", maxRows: 2 }), 360);
    expect(seatedIds(layout)).toContain("mic");
    expect(layout.overflow.map((s) => s.id)).toContain("image");
  });
});

describe("layoutToolbar — the fill control", () => {
  it("grows into the width its row has left", () => {
    const layout = layoutToolbar(prefs(), 430);
    const mic = layout.rows.flatMap((r) => r.slots).find((s) => s.id === "mic");
    expect(mic).toBeDefined();
    expect(mic!.width).toBeGreaterThan(TOOLBAR_DENSITY_PX.standard);
  });

  it("renders last on its row whatever order it was seated in", () => {
    const layout = layoutToolbar(prefs(), 430);
    for (const row of layout.rows) {
      const fillAt = row.slots.findIndex((s) => s.fill);
      if (fillAt >= 0) expect(fillAt).toBe(row.slots.length - 1);
    }
  });

  it("splits a row evenly when several fill controls share it", () => {
    const layout = layoutToolbar(prefs({ maxRows: 1 }), 390, { view: "messages" });
    const widths = (layout.rows[0]?.slots ?? []).map((s) => s.width);
    expect(widths.length).toBeGreaterThan(1);
    expect(Math.max(...widths) - Math.min(...widths)).toBeLessThanOrEqual(1);
  });

  it("stays at or above the button size even when the row is tight", () => {
    for (const width of WIDTHS) {
      const layout = layoutToolbar(prefs(), width);
      for (const slot of layout.rows.flatMap((r) => r.slots)) {
        expect(slot.width).toBeGreaterThanOrEqual(layout.metrics.unit);
      }
    }
  });
});

describe("layoutToolbar — the swipe strip", () => {
  it("never costs a row", () => {
    for (const width of WIDTHS) {
      const withStrip = layoutToolbar(prefs({ maxRows: 2, overflow: "strip" }), width);
      const withoutStrip = layoutToolbar(prefs({ maxRows: 2, overflow: "more" }), width);
      expect(withStrip.rowCount).toBe(withoutStrip.rowCount);
      expect(withStrip.keysHeightPx).toBe(withoutStrip.keysHeightPx);
    }
  });

  it("appears only when something actually overflowed", () => {
    const layout = layoutToolbar(prefs({ overflow: "strip" }), 430);
    expect(layout.overflow).toHaveLength(0);
    expect(layout.strip).toBeNull();
  });

  it("does not evict a control that was legitimately seated", () => {
    const withStrip = layoutToolbar(prefs({ maxRows: 2, overflow: "strip" }), 360);
    const withoutStrip = layoutToolbar(prefs({ maxRows: 2, overflow: "more" }), 360);
    expect(seatedIds(withStrip)).toEqual(seatedIds(withoutStrip));
  });

  it("is never narrower than a readable affordance", () => {
    for (const width of WIDTHS) {
      const layout = layoutToolbar(prefs({ maxRows: 1, overflow: "strip" }), width);
      if (layout.strip) expect(layout.strip.width).toBeGreaterThanOrEqual(32);
    }
  });
});

describe("layoutToolbar — views", () => {
  it("drops terminal-only controls in the messages view", () => {
    const layout = layoutToolbar(prefs(), 390, { view: "messages" });
    const ids = [...seatedIds(layout), ...layout.overflow.map((s) => s.id)];
    expect(ids).not.toContain("modifiers");
    expect(ids).not.toContain("special");
    expect(ids).not.toContain("arrows");
  });

  it("gives the messages view a single row whatever the terminal budget is", () => {
    for (const maxRows of [1, 2, 3] as const) {
      const layout = layoutToolbar(prefs({ maxRows }), 390, { view: "messages" });
      expect(layout.rowCount).toBe(1);
    }
  });

  it("splits the messages row evenly between its actions", () => {
    const everything = Object.fromEntries(TOOLBAR_CONTROLS.map((c) => [c.id, true]));
    const layout = layoutToolbar(prefs({ maxRows: 3, enabled: everything }), 390, { view: "messages" });
    const widths = (layout.rows[0]?.slots ?? []).map((s) => s.width);
    expect(widths.length).toBeGreaterThan(1);
    expect(Math.max(...widths) - Math.min(...widths)).toBeLessThanOrEqual(1);
  });

  it("claims no width for controls whose feature is unavailable", () => {
    const withMic = layoutToolbar(prefs(), 390);
    const withoutMic = layoutToolbar(prefs(), 390, { unavailable: ["mic"] });
    expect(seatedIds(withoutMic)).not.toContain("mic");
    expect(withoutMic.overflow.map((s) => s.id)).not.toContain("mic");
    expect(seatedIds(withMic)).toContain("mic");
  });
});

describe("toolbar presets", () => {
  it("fit every control on a mainstream phone", () => {
    for (const preset of PRESETS) {
      const layout = layoutToolbar(toolbarPrefsFromPreset(preset), 390);
      expect(layout.overflow, `${preset} overflowed at 390px`).toHaveLength(0);
      expect(layout.rowCount).toBeLessThanOrEqual(2);
    }
  });

  it("keeps More, snippets, and image upload on, and AI off, by default", () => {
    for (const preset of PRESETS) {
      const { enabled } = TOOLBAR_PRESETS[preset];
      expect(enabled.more).toBe(true);
      expect(enabled.snippets).toBe(true);
      expect(enabled.image).toBe(true);
      expect(enabled.ai).toBe(false);
    }
  });

  it("only the largest preset meets the recommended touch target", () => {
    expect(TOOLBAR_DENSITY_PX[TOOLBAR_PRESETS.essential.density]).toBeGreaterThanOrEqual(
      MIN_RECOMMENDED_TOUCH_TARGET_PX,
    );
    expect(TOOLBAR_DENSITY_PX[TOOLBAR_PRESETS.dense.density]).toBeLessThan(MIN_RECOMMENDED_TOUCH_TARGET_PX);
  });

  it("are shorter than the wrapping layout they replace", () => {
    // The shipped flex-wrap layout reached four visual rows at 390px.
    for (const preset of PRESETS) {
      const layout = layoutToolbar(toolbarPrefsFromPreset(preset), 390);
      expect(layout.keysHeightPx).toBeLessThan(4 * TOOLBAR_DENSITY_PX.large);
    }
  });
});

describe("normalizeToolbarPrefs", () => {
  it("falls back to the default preset for junk input", () => {
    expect(normalizeToolbarPrefs(null)).toEqual(DEFAULT_TOOLBAR_PREFS);
    expect(normalizeToolbarPrefs("expanded")).toEqual(DEFAULT_TOOLBAR_PREFS);
    expect(normalizeToolbarPrefs({ density: "enormous", maxRows: 9 })).toEqual(DEFAULT_TOOLBAR_PREFS);
  });

  it("keeps valid fields and repairs invalid ones independently", () => {
    const result = normalizeToolbarPrefs({ preset: "custom", density: "compact", maxRows: 7 });
    expect(result.preset).toBe("custom");
    expect(result.density).toBe("compact");
    expect(result.maxRows).toBe(DEFAULT_TOOLBAR_PREFS.maxRows);
  });

  it("drops control ids the registry does not know", () => {
    const result = normalizeToolbarPrefs({ enabled: { ai: true, "shortcut:ghost": true } });
    expect(result.enabled.ai).toBe(true);
    expect(result.enabled["shortcut:ghost"]).toBeUndefined();
  });

  it("enables a newly registered snippet control for older persisted preferences", () => {
    const result = normalizeToolbarPrefs({ enabled: { image: false } });
    expect(result.enabled.snippets).toBe(true);
    expect(result.enabled.image).toBe(false);
  });
});

describe("registry and renderer agree on key caps", () => {
  // `controlWidth` sizes a `keys` control from `spec.keys`, while the renderer
  // paints one cap per entry in its own list. If the two lists drift, every
  // width the engine computes for that control is wrong and nothing else
  // notices — the toolbar just starts overflowing for no visible reason.
  it("sizes the modifier control from the caps it actually renders", () => {
    const spec = TOOLBAR_CONTROLS.find((c) => c.id === "modifiers");
    expect(spec?.keys).toEqual(MODIFIER_KEYS.map(modifierLabel));
  });

  it("sizes the special-key control from the caps it actually renders", () => {
    const spec = TOOLBAR_CONTROLS.find((c) => c.id === "special");
    expect(spec?.keys).toEqual(SPECIAL_KEYS.map((k) => k.label));
  });
});
