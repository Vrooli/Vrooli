import { describe, it, expect } from "vitest";
import { migrateToolbarPrefs } from "./useWorkspaceStore";
import { DEFAULT_TOOLBAR_PREFS, TOOLBAR_CONTROLS, layoutToolbar } from "../lib/toolbarLayout";

/**
 * The toolbar settings changed shape in v23. A migration that quietly resets a
 * preference is worse than one that fails loudly, so these cases pin the
 * mapping rather than the implementation.
 */
describe("migrateToolbarPrefs — v22 → v23", () => {
  it("maps the old expanded layout onto the preset that looks like it", () => {
    const result = migrateToolbarPrefs({ toolbarLayout: "expanded" }, 22);
    expect(result.preset).toBe("balanced");
    expect(result.arrows).toBe("dpad");
    expect(result.maxRows).toBe(2);
  });

  it("maps the old compact layout onto small buttons in one run", () => {
    const result = migrateToolbarPrefs({ toolbarLayout: "compact" }, 22);
    expect(result.preset).toBe("dense");
    expect(result.density).toBe("compact");
    expect(result.arrows).toBe("inline");
  });

  it("treats a missing legacy value as the default rather than resetting to nothing", () => {
    const result = migrateToolbarPrefs({}, 22);
    expect(result).toEqual(DEFAULT_TOOLBAR_PREFS);
  });

  it("produces prefs that lay out cleanly on a mainstream phone", () => {
    for (const legacy of ["compact", "expanded"]) {
      const layout = layoutToolbar(migrateToolbarPrefs({ toolbarLayout: legacy }, 22), 390);
      expect(layout.overflow, `${legacy} overflowed after migration`).toHaveLength(0);
    }
  });
});

describe("migrateToolbarPrefs — already on v23", () => {
  it("keeps prefs the user already chose", () => {
    const stored = {
      preset: "custom" as const,
      density: "large" as const,
      arrows: "inline" as const,
      maxRows: 3 as const,
      overflow: "more" as const,
      enabled: { more: true, modifiers: true, special: false, arrows: true, mic: true, image: true, ai: true },
    };
    expect(migrateToolbarPrefs({ toolbarPrefs: stored }, 23)).toEqual(stored);
  });

  it("repairs values an older or hand-edited build could have written", () => {
    const result = migrateToolbarPrefs(
      { toolbarPrefs: { density: "enormous", maxRows: 9, overflow: "sideways", enabled: { "shortcut:ghost": true } } },
      23,
    );
    expect(result.density).toBe(DEFAULT_TOOLBAR_PREFS.density);
    expect(result.maxRows).toBe(DEFAULT_TOOLBAR_PREFS.maxRows);
    expect(result.overflow).toBe(DEFAULT_TOOLBAR_PREFS.overflow);
    expect(result.enabled["shortcut:ghost"]).toBeUndefined();
  });

  it("never drops the pinned overflow host, whatever was persisted", () => {
    const pinned = TOOLBAR_CONTROLS.filter((c) => c.pinned).map((c) => c.id);
    expect(pinned.length).toBeGreaterThan(0);
    const result = migrateToolbarPrefs({ toolbarPrefs: { enabled: { more: false } } }, 23);
    const layout = layoutToolbar(result, 390);
    const seated = layout.rows.flatMap((r) => r.slots.map((s) => s.id));
    for (const id of pinned) expect(seated).toContain(id);
  });
});
