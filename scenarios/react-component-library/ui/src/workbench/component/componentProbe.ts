import { contrastRatioFromCss, effectiveBackgroundFromStack } from "./instruments";

/**
 * What the preview harness can see that the workbench page cannot.
 *
 * The specimen renders inside the preview iframe, so measuring the page finds
 * the page's own buttons and text. The harness reports the component's control
 * sizes and one readable type sample instead; the workbench turns those into
 * the component-fit checks with the same instrument rules it uses elsewhere.
 */
export interface ComponentProbeTypography {
  fontFamily: string;
  fontSize: string;
  color: string;
  /** Background colours from the text node out to the component stage. */
  backgroundStack: string[];
  label: string;
}

export interface ComponentProbe {
  measured: boolean;
  targets: { total: number; small: number };
  typography: ComponentProbeTypography | null;
}

export interface ComponentProbeContrast {
  /** False when there is no measured probe to read. */
  on: boolean;
  ratio: number | null;
  label: string;
  background: string;
}

/** The component's own text contrast, using the same instrument maths. */
export function componentProbeContrast(probe: ComponentProbe | null): ComponentProbeContrast {
  if (!probe?.measured) {
    return { on: false, ratio: null, label: "", background: "" };
  }
  const type = probe.typography;
  if (!type) {
    // An icon-only control has no text to compare. The page's own text is not
    // evidence about the component.
    return { on: true, ratio: null, label: "", background: "" };
  }
  const background = effectiveBackgroundFromStack(type.backgroundStack) ?? "";
  return {
    on: true,
    ratio: background ? contrastRatioFromCss(type.color, background) : null,
    label: type.label,
    background,
  };
}

function record(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined;
}

function finite(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function text(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value : undefined;
}

/** Messages cross an iframe boundary, so every field is checked. */
export function componentProbeFromMessage(value: unknown): ComponentProbe | null {
  const root = record(value);
  if (!root) return null;
  const targets = record(root.targets);
  const total = finite(targets?.total);
  const small = finite(targets?.small);
  if (typeof root.measured !== "boolean" || total === undefined || small === undefined) {
    return null;
  }
  const type = record(root.typography);
  const stack = Array.isArray(type?.backgroundStack)
    ? type.backgroundStack.filter((entry): entry is string => typeof entry === "string")
    : [];
  const typography =
    type && text(type.fontFamily) && text(type.fontSize) && text(type.color)
      ? {
          fontFamily: type.fontFamily as string,
          fontSize: type.fontSize as string,
          color: type.color as string,
          backgroundStack: stack,
          label: text(type.label) ?? "",
        }
      : null;
  return {
    measured: root.measured,
    targets: { total: Math.max(0, total), small: Math.max(0, small) },
    typography,
  };
}
