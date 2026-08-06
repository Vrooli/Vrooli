import { canMorph, type NormalizedIconGeometry } from "./geometry";

export type MorphingIconStrategy = "auto" | "morph" | "transform" | "crossfade";
export type MorphingIconMode = "morph" | "transform" | "crossfade";

export interface TransitionPlan {
  mode: MorphingIconMode;
  reason: "compatible-geometry" | "requested" | "incompatible-geometry";
}

export function chooseTransitionStrategy(
  requested: MorphingIconStrategy,
  from: NormalizedIconGeometry,
  to: NormalizedIconGeometry,
): TransitionPlan {
  if (requested === "morph") {
    return canMorph(from, to)
      ? { mode: "morph", reason: "requested" }
      : { mode: "crossfade", reason: "incompatible-geometry" };
  }
  if (requested === "transform" || requested === "crossfade") {
    return { mode: requested, reason: "requested" };
  }
  return canMorph(from, to)
    ? { mode: "morph", reason: "compatible-geometry" }
    : { mode: "transform", reason: "incompatible-geometry" };
}
