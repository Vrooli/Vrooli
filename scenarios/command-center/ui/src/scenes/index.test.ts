import { afterEach, describe, expect, it } from "vitest";
import { compositionSlotManifests, compositions, configureEnabledPacks, createScene } from "./index";

type RoomPreset = { id: string; theme?: string; composition?: string; bind?: Record<string, string>; beats?: Array<{ composition?: string }> };
type ThemeEntry = { id: string };
const roomPresets = Object.values(import.meta.glob<RoomPreset>("../../../config/rooms/*.json", { eager: true, import: "default" }));
const themeIDs = new Set(Object.values(import.meta.glob<ThemeEntry>("../../../config/themes/*.json", { eager: true, import: "default" })).map((entry) => entry.id));

describe("composition pack registry", () => {
  afterEach(() => configureEnabledPacks(new Set(["vrooli"])));
  it("removes Vrooli-specific compositions when the pack is disabled", () => {
    configureEnabledPacks(new Set());
    expect(compositions["meridian-arc"]).toBeUndefined();
    expect(compositions["funnel-cascade"]).toBeUndefined();
    expect(createScene("meridian-arc")).toBeDefined();
  });
  it("restores bespoke compositions when enabled", () => {
    configureEnabledPacks(new Set(["vrooli"]));
    expect(compositions["meridian-arc"]).toBeDefined();
    expect(compositions["funnel-cascade"]).toBeDefined();
  });

  it("keeps a scene factory for every declared composition and a usable fallback", () => {
    configureEnabledPacks(new Set(["vrooli"]));
    for (const id of Object.keys(compositionSlotManifests)) {
      const scene = createScene(id);
      expect(typeof scene.draw, `${id} should create a drawable scene`).toBe("function");
    }
    expect(typeof createScene("unknown-composition").draw).toBe("function");
  });

  it("resolves every shipped room and beat to a registered composition and declared slots", () => {
    configureEnabledPacks(new Set(["vrooli"]));
    expect(roomPresets.length).toBeGreaterThan(0);
    for (const room of roomPresets) {
      expect(room.composition, `${room.id} must define its default composition`).toBeTruthy();
      expect(themeIDs.has(room.theme ?? ""), `${room.id} references an unknown theme ${room.theme}`).toBe(true);
      const compositionIDs = new Set([room.composition, ...(room.beats ?? []).map((beat) => beat.composition)].filter((id): id is string => Boolean(id)));
      for (const id of compositionIDs) {
        expect(compositions[id], `${room.id} references unregistered composition ${id}`).toBeDefined();
        expect(compositionSlotManifests[id], `${room.id} references composition ${id} without a slot manifest`).toBeDefined();
      }
      const slots = compositionSlotManifests[room.composition ?? ""] ?? {};
      for (const slotName of Object.keys(room.bind ?? {})) {
        expect(slots[slotName], `${room.id} binds undeclared slot ${slotName}`).toBeDefined();
      }
    }
  });
});
