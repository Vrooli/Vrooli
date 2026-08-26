/**
 * @libraryId react-component-library:VisualRecipes
 * @displayName Visual Recipes
 * @description Typed class recipes that keep visual variants token-backed and reviewable.
 * @version 1.0.1
 * @tags ["recipes","variants","tokens"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource foundations.visual-recipes */
import { COMPONENT_TOKENS, SEMANTIC_TOKENS } from "@vrooli/react-component-library/Tokens/1.0.0";

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
