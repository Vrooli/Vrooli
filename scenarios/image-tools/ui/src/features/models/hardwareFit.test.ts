/**
 * Unit tests for `hardwareFitChips` — the pure derivation that turns a model's
 * Hardware message into the operator-facing fit chips. Asserts on stable key
 * paths + tones, not translated copy, so locale edits never break it.
 */
import { create } from "@bufbuild/protobuf";
import { describe, expect, it } from "vitest";
import { HardwareSchema } from "@vrooli/proto-types/image-tools/v1/models/models_pb";

import { strings } from "../../consts/strings";
import { hardwareFitChips } from "./hardwareFit";

describe("hardwareFitChips", () => {
  it("returns no chips when hardware is undefined", () => {
    expect(hardwareFitChips(undefined)).toEqual([]);
  });

  it("leads with a positive CPU chip when the model runs on CPU", () => {
    const chips = hardwareFitChips(
      create(HardwareSchema, { cpuCapable: true, gpuRequired: false }),
    );
    expect(chips[0]).toEqual({ key: strings.models.hardware.cpuOk, tone: "positive" });
  });

  it("leads with a caution GPU chip when a GPU is required", () => {
    const chips = hardwareFitChips(
      create(HardwareSchema, { gpuRequired: true }),
    );
    expect(chips[0]).toEqual({ key: strings.models.hardware.gpuRequired, tone: "caution" });
  });

  it("adds VRAM/RAM floors only when the seed declares a non-zero minimum", () => {
    const none = hardwareFitChips(create(HardwareSchema, { gpuRequired: true, minVramGb: 0, minRamGb: 0 }));
    expect(none.map((c) => c.key)).not.toContain(strings.models.hardware.vram);
    expect(none.map((c) => c.key)).not.toContain(strings.models.hardware.ram);

    const both = hardwareFitChips(
      create(HardwareSchema, { gpuRequired: true, minVramGb: 8, minRamGb: 16 }),
    );
    expect(both).toContainEqual({
      key: strings.models.hardware.vram,
      values: { gb: 8 },
      tone: "neutral",
    });
    expect(both).toContainEqual({
      key: strings.models.hardware.ram,
      values: { gb: 16 },
      tone: "neutral",
    });
  });

  it("appends the seed speed note as a neutral trailing chip when present", () => {
    const withNote = hardwareFitChips(
      create(HardwareSchema, { cpuCapable: true, speedNote: "~30s per image on CPU" }),
    );
    expect(withNote).toContainEqual({
      key: strings.models.hardware.speedNote,
      values: { note: "~30s per image on CPU" },
      tone: "neutral",
    });

    const noNote = hardwareFitChips(create(HardwareSchema, { cpuCapable: true, speedNote: "  " }));
    expect(noNote.map((c) => c.key)).not.toContain(strings.models.hardware.speedNote);
  });
});
