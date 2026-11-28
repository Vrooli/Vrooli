import { describe, expect, it } from "vitest";

import { capabilityMeta } from "./capabilityCatalog";
import {
  activePreset,
  applyPreset,
  defaultSelection,
  extentOpParams,
  optionValue,
  setOption,
  strengthApplies,
  toggleStage,
} from "./swapExtent";

const extent = capabilityMeta("face_swap")?.extent;
if (!extent) {
  throw new Error("face_swap declares no extent in the generated catalog");
}

describe("swap extent selection", () => {
  it("starts on the default preset at full strength", () => {
    expect(defaultSelection(extent)).toEqual({ stages: ["face"], faceStrength: 1, options: {} });
  });

  it("applies a preset's stages in catalog order and keeps the strength", () => {
    const next = applyPreset(extent, { stages: ["face"], faceStrength: 0.4, options: {} }, "full_look");
    expect(next).toEqual({ stages: ["face", "skin", "hair_colour"], faceStrength: 0.4, options: {} });
  });

  it("toggles stages independently", () => {
    let selection = defaultSelection(extent);
    selection = toggleStage(extent, selection, "hair_colour");
    expect(selection.stages).toEqual(["face", "hair_colour"]);
    selection = toggleStage(extent, selection, "skin");
    expect(selection.stages).toEqual(["face", "skin", "hair_colour"]);
    selection = toggleStage(extent, selection, "hair_colour");
    expect(selection.stages).toEqual(["face", "skin"]);
  });

  it("offers the hair style as its own stage, ordered before the hair colour", () => {
    let selection = applyPreset(extent, defaultSelection(extent), "full_look");
    selection = toggleStage(extent, selection, "hair_style");
    expect(selection.stages).toEqual(["face", "skin", "hair_style", "hair_colour"]);
    expect(extentOpParams(extent, selection)).toEqual({ match: ["face", "skin", "hair_style", "hair_colour"] });
  });

  it("lets the whole head take the face's place and gives it back", () => {
    let selection = toggleStage(extent, defaultSelection(extent), "head");
    expect(selection.stages).toEqual(["head"]);
    selection = toggleStage(extent, selection, "head");
    expect(selection.stages).toEqual(["face"]); // never left without a swap
    selection = toggleStage(extent, toggleStage(extent, selection, "head"), "face");
    expect(selection.stages).toEqual(["face"]);
  });

  it("names the preset a selection matches, and none for a custom mix", () => {
    expect(activePreset(extent, { stages: ["face", "skin"], faceStrength: 1, options: {} })?.id).toBe("face_skin");
    expect(activePreset(extent, { stages: ["face", "hair_colour"], faceStrength: 1, options: {} })).toBeUndefined();
  });
});

describe("swap extent op params", () => {
  it("sends the preset name when the selection is one", () => {
    expect(extentOpParams(extent, { stages: ["face", "skin", "hair_colour"], faceStrength: 1, options: {} })).toEqual({ extent: "full_look" });
  });

  it("sends the stages for a custom mix", () => {
    expect(extentOpParams(extent, { stages: ["face", "hair_colour"], faceStrength: 1, options: {} })).toEqual({ match: ["face", "hair_colour"] });
  });

  it("sends the face strength only when the face stage runs and it is below full", () => {
    expect(extentOpParams(extent, { stages: ["face"], faceStrength: 0.6, options: {} })).toEqual({ extent: "face", face_strength: 0.6 });
    expect(extentOpParams(extent, { stages: ["head"], faceStrength: 0.6, options: {} })).toEqual({ extent: "whole_head" });
    expect(strengthApplies(extent, { stages: ["head"], faceStrength: 0.6, options: {} })).toBe(false);
  });
});

describe("swap extent stage options", () => {
  const skin = extent.stages.find((stage) => stage.id === "skin");
  const option = skin?.option;
  if (!option) {
    throw new Error("the skin stage declares no option in the generated catalog");
  }

  it("declares the choices and the default from the manifest", () => {
    expect(option.param).toBe("skin_target");
    expect(option.values.map((value) => value.id).sort()).toEqual(["body", "source"]);
    expect(optionValue(option, defaultSelection(extent))).toBe(option.default);
  });

  it("sends a choice only when it differs from the default and its stage runs", () => {
    const withSkin = { stages: ["face", "skin"], faceStrength: 1, options: {} };
    expect(extentOpParams(extent, setOption(withSkin, option, option.default))).toEqual({ extent: "face_skin" });
    expect(extentOpParams(extent, setOption(withSkin, option, "body"))).toEqual({ extent: "face_skin", skin_target: "body" });
    const withoutSkin = setOption({ stages: ["face"], faceStrength: 1, options: {} }, option, "body");
    expect(extentOpParams(extent, withoutSkin)).toEqual({ extent: "face" });
  });
});
