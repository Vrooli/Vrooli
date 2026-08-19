/**
 * @vrooliComponentSource foundations.visual-recipes
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption b3a33386-5e25-423e-9ee8-022f86679c1f
 * @vrooliComponentAppliedAt 2026-08-18T01:12:45Z
 * @vrooliComponentSourceSha256 c6dc5df5c70cba2137dc0af74be35acee4fdae44f914e43775e78a7bda9ffd57
 * @vrooliComponentDriftHash 8c162e977ab1d31c6dc217d0059f8162c67a942a213ca752209012d87059591b
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
    "bg-[var(--color-primary)]",
    "text-[var(--color-primary-foreground)]",
    "border-transparent",
  ),
  secondary: recipe(
    "bg-[var(--color-surface)]",
    "text-[var(--color-foreground)]",
    "border-[var(--color-border)]",
  ),
  ghost: recipe("bg-transparent", "text-[var(--color-foreground)]", "border-transparent"),
  danger: recipe(
    "bg-[var(--color-danger)]",
    "text-[var(--color-primary-foreground)]",
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
