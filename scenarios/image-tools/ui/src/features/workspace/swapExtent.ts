import { strings } from "../../consts/strings";
import type { ExtentMeta, ExtentOptionMeta, ExtentPresetMeta } from "./capabilityCatalog.generated";
import type { FaceSwapOpParams } from "./opParams.generated";

type ExtentStrings = typeof strings.workspace.extent;
/**
 * The i18n key unions the generated catalog's extent keys resolve into. The
 * manifest carries dotted key strings (data); these unions keep `t()` checked,
 * and the catalog conformance test asserts every generated key exists.
 */
export type ExtentLabelKey =
  | ExtentStrings["stage"][keyof ExtentStrings["stage"]]["label"]
  | ExtentStrings["preset"][keyof ExtentStrings["preset"]]["label"]
  | ExtentStrings["skinTarget"][keyof ExtentStrings["skinTarget"]]["label"];
export type ExtentDescKey =
  | ExtentStrings["stage"][keyof ExtentStrings["stage"]]["desc"]
  | ExtentStrings["preset"][keyof ExtentStrings["preset"]]["desc"]
  | ExtentStrings["skinTarget"][keyof ExtentStrings["skinTarget"]]["desc"];

type ExtentItem = { readonly labelKey: string; readonly descKey: string };
export const extentLabelKey = (item: ExtentItem): ExtentLabelKey => item.labelKey as ExtentLabelKey;
export const extentDescKey = (item: ExtentItem): ExtentDescKey => item.descKey as ExtentDescKey;

/**
 * The stages a face swap carries over, chosen on the Create panel. The set of
 * stages, the presets and which stage stands in for which are the operation's
 * manifest declaration (generated into the capability catalog); nothing here
 * names a stage.
 */
export interface ExtentSelection {
  readonly stages: readonly string[];
  /** How much of the source face the swap takes, 0..1. */
  readonly faceStrength: number;
  /** The choice made for each stage option, by op parameter; absent means the manifest's default. */
  readonly options: Readonly<Record<string, string>>;
}

const inCatalogOrder = (extent: ExtentMeta, ids: Iterable<string>): string[] => {
  const wanted = new Set(ids);
  return extent.stages.filter((stage) => wanted.has(stage.id)).map((stage) => stage.id);
};

export const presetById = (extent: ExtentMeta, id: string): ExtentPresetMeta | undefined =>
  extent.presets.find((preset) => preset.id === id);

/** The default preset's stages at full strength. */
export function defaultSelection(extent: ExtentMeta): ExtentSelection {
  return applyPreset(extent, { stages: [], faceStrength: 1, options: {} }, extent.defaultPreset);
}

/** A preset sets the stages and keeps the strength. */
export function applyPreset(extent: ExtentMeta, selection: ExtentSelection, presetId: string): ExtentSelection {
  const preset = presetById(extent, presetId);
  return preset ? { ...selection, stages: inCatalogOrder(extent, preset.stages) } : selection;
}

/**
 * Turns one stage on or off. A stage that replaces another (the whole head
 * replaces the inner face) takes its place when switched on, and the replaced
 * stage takes it back when switched on again.
 */
export function toggleStage(extent: ExtentMeta, selection: ExtentSelection, stageId: string): ExtentSelection {
  const next = new Set(selection.stages);
  const stage = extent.stages.find((candidate) => candidate.id === stageId);
  if (!stage) {
    return selection;
  }
  if (next.has(stageId)) {
    next.delete(stageId);
  } else {
    next.add(stageId);
    if (stage.replaces) {
      next.delete(stage.replaces);
    }
    for (const other of extent.stages) {
      if (other.replaces === stageId) {
        next.delete(other.id);
      }
    }
  }
  // A swap always needs one stage that produces the face.
  const hasSwap = extent.stages.some((candidate) => candidate.kind === "swap" && next.has(candidate.id));
  if (!hasSwap) {
    const swap = extent.stages.find((candidate) => candidate.kind === "swap" && !candidate.replaces);
    if (swap) {
      next.add(swap.id);
    }
  }
  return { ...selection, stages: inCatalogOrder(extent, next) };
}

/** The value a stage option currently has: the choice made, else the declared default. */
export const optionValue = (option: ExtentOptionMeta, selection: ExtentSelection): string =>
  selection.options[option.param] ?? option.default;

export const setOption = (selection: ExtentSelection, option: ExtentOptionMeta, value: string): ExtentSelection => ({
  ...selection,
  options: { ...selection.options, [option.param]: value },
});

/** The preset whose stages are exactly the selection's, if any. */
export function activePreset(extent: ExtentMeta, selection: ExtentSelection): ExtentPresetMeta | undefined {
  const chosen = [...selection.stages].sort().join(",");
  return extent.presets.find((preset) => [...preset.stages].sort().join(",") === chosen);
}

/** Whether the selection runs the stage that takes the strength parameter. */
export const strengthApplies = (extent: ExtentMeta, selection: ExtentSelection): boolean =>
  extent.stages.some((stage) => stage.strengthParam && selection.stages.includes(stage.id));

/**
 * The op parameters for a selection: a named preset when it matches one (so the
 * job records the name a person picked), otherwise the stages themselves.
 */
export function extentOpParams(extent: ExtentMeta, selection: ExtentSelection): FaceSwapOpParams {
  const out: FaceSwapOpParams = {};
  const preset = activePreset(extent, selection);
  if (preset) {
    out.extent = preset.id;
  } else {
    out.match = [...selection.stages];
  }
  if (strengthApplies(extent, selection) && selection.faceStrength < 1) {
    out.face_strength = selection.faceStrength;
  }
  // Only a choice that differs from the default, on a stage that runs, is sent.
  const chosen: Record<string, string> = {};
  for (const stage of extent.stages) {
    if (stage.option && selection.stages.includes(stage.id) && optionValue(stage.option, selection) !== stage.option.default) {
      chosen[stage.option.param] = optionValue(stage.option, selection);
    }
  }
  return { ...out, ...chosen };
}
