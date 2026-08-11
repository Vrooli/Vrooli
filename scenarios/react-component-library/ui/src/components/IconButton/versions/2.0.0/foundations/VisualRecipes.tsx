/**
 * @vrooliComponentSource react-component-library:VisualRecipes
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 59f07a82-a0b1-4dd2-987d-4d497a059ba5
 * @vrooliComponentAppliedAt 2026-08-11T00:35:31Z
 * @vrooliComponentSourceSha256 002adc8d4ef7854cbd4fb08eb2156e2b0deb04eff5a91ece409fa3431a255fdd
 * @vrooliComponentDriftHash b65df4be4778bcc33701c60ca4ca6105d2db5a7ff26f8f7e9aa5542610c192df
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { COMPONENT_TOKENS, SEMANTIC_TOKENS } from "./Tokens";

export const recipe = (...classes: Array<string | false | null | undefined>) =>
  classes.filter(Boolean).join(" ");

export type MotionTransitionPhase = "interaction" | "enter" | "exit" | "spring";

export const MOTION_TRANSITIONS: Record<MotionTransitionPhase, string> = {
  interaction: "var(--dur-quick) var(--ease-standard)",
  enter: "var(--dur-moderate) var(--ease-enter)",
  exit: "var(--dur-quick) var(--ease-exit)",
  spring: "var(--dur-moderate) var(--spring-subtle)",
};

/** Build a token-backed transition list without allowing ad hoc timing values. */
export const motionTransition = (
  properties: string | readonly string[],
  phase: MotionTransitionPhase = "interaction",
) => {
  const names = Array.isArray(properties) ? properties : [properties];
  return names.map((property) => `${property} ${MOTION_TRANSITIONS[phase]}`).join(", ");
};
export const CONTROL_VARIANTS = {
  primary: recipe(
    "bg-[var(--app-primary)]",
    "text-[var(--app-primary-foreground)]",
    "border-transparent",
  ),
  secondary: recipe(
    "bg-[var(--app-surface)]",
    "text-[var(--app-foreground)]",
    "border-[var(--app-border)]",
  ),
  ghost: recipe("bg-transparent", "text-[var(--app-foreground)]", "border-transparent"),
  danger: recipe(
    "bg-[var(--app-danger)]",
    "text-[var(--app-primary-foreground)]",
    "border-transparent",
  ),
} as const;
export const CONTROL_SIZES = {
  sm: recipe(
    "min-h-[var(--control-height-sm)]",
    "px-[var(--space-sm)]",
    "text-[var(--text-label)]",
  ),
  md: recipe(
    `min-h-[${COMPONENT_TOKENS.controlHeight}]`,
    `px-[${COMPONENT_TOKENS.controlPadding}]`,
    `text-[${SEMANTIC_TOKENS.foreground}]`,
  ),
  lg: recipe(
    "min-h-[var(--control-height-lg)]",
    "px-[var(--space-md)]",
    "text-[var(--text-label)]",
  ),
} as const;
export const SURFACE_ELEVATIONS = {
  flat: "shadow-[var(--elev-flat)]",
  raised: "shadow-[var(--elev-raised)]",
  floating: "shadow-[var(--elev-floating)]",
  overlay: "shadow-[var(--elev-overlay)]",
} as const;
export type ControlVariant = keyof typeof CONTROL_VARIANTS;
export type ControlSize = keyof typeof CONTROL_SIZES;
