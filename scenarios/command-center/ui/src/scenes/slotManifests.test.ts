import { describe, expect, it } from "vitest";
import type { SlotManifest } from "./engine";
import { compositionSlotManifests } from "./index";

type SlotSidecar = { composition: string; slots: SlotManifest };
const sidecars = Object.values(import.meta.glob<SlotSidecar>("../../../config/compositions/*.slots.json", { eager: true, import: "default" }));
const sidecarsByID = new Map(sidecars.map((sidecar) => [sidecar.composition, sidecar]));

describe("composition slot sidecars", () => {
  it("match every module definition without orphan or duplicate ids", () => {
    expect(sidecarsByID.size, "each catalog sidecar must have a distinct composition id").toBe(sidecars.length);
    expect(sidecars.length, "every module definition needs exactly one catalog sidecar").toBe(Object.keys(compositionSlotManifests).length);

    for (const [composition, slots] of Object.entries(compositionSlotManifests)) {
      const sidecar = sidecarsByID.get(composition);
      expect(sidecar, `missing .slots.json sidecar for ${composition}`).toBeDefined();
      expect(sidecar?.composition).toBe(composition);
      expect(sidecar?.slots).toEqual(slots);
    }

    for (const sidecar of sidecars) {
      expect(compositionSlotManifests[sidecar.composition], `sidecar has no registered module: ${sidecar.composition}`).toBeDefined();
    }
  });
});
