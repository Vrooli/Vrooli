import type { DesignFrame } from "../designs";
import { walk, type TemplateNode, type TemplateNodeType } from "../templates/spec";

/** Minimum interactive height the studio checks against, in CSS pixels. */
export const TOUCH_TARGET_MIN = 44;

const STATEFUL_TYPES = new Set<TemplateNodeType>(["stat", "meter", "slot", "button", "input"]);

export interface StateAudit {
  /** Layers that present data or accept input. */
  total: number;
  /** Layers that declare any empty/loading/error state. */
  covered: number;
  missing: string[];
}

function declaresStates(current: TemplateNode): boolean {
  const states = current.props.states;
  if (typeof states === "string" && states.trim().length > 0) return true;
  return typeof current.props.state === "string" && current.props.state.trim().length > 0;
}

function labelOf(current: TemplateNode): string {
  for (const key of ["label", "text", "name", "hint"]) {
    const value = current.props[key];
    if (typeof value === "string" && value.trim()) return value.slice(0, 32);
  }
  return current.type;
}

/**
 * A cheap declared-state audit: metric, slot and input layers should say how
 * they behave when empty, loading or failed. The studio reports the gap rather
 * than inventing states the composition does not have.
 */
export function stateAudit(spec: TemplateNode): StateAudit {
  let total = 0;
  let covered = 0;
  const missing: string[] = [];
  for (const child of spec.children ?? []) {
    walk(child, (current) => {
      if (!STATEFUL_TYPES.has(current.type)) return;
      total += 1;
      if (declaresStates(current)) covered += 1;
      else missing.push(labelOf(current));
    });
  }
  return { total, covered, missing };
}

export interface ButtonEstimate {
  nodeId: string;
  label: string;
  height: number;
}

const BUTTON_HEIGHT: Record<string, number> = {
  xs: 28,
  sm: 32,
  md: 40,
  lg: 48,
  xl: 56,
};

export function declaredButtonHeight(size: unknown): number {
  return typeof size === "string" ? (BUTTON_HEIGHT[size] ?? 40) : 40;
}

/**
 * Declared button heights from the spec. The runtime audit measures the real
 * DOM and prefers that; this is the deterministic fallback used in tests and
 * anywhere layout cannot be measured.
 */
export function buttonEstimates(spec: TemplateNode): ButtonEstimate[] {
  const buttons: ButtonEstimate[] = [];
  for (const child of spec.children ?? []) {
    walk(child, (current) => {
      if (current.type !== "button") return;
      buttons.push({
        nodeId: current.id,
        label: labelOf(current),
        height: declaredButtonHeight(current.props.size),
      });
    });
  }
  return buttons;
}

export function underTargetButtons(spec: TemplateNode): ButtonEstimate[] {
  return buttonEstimates(spec).filter((button) => button.height < TOUCH_TARGET_MIN);
}

export interface FrameAudit {
  /** Pixels of horizontal overflow inside the rendered page, 0 when measured clean. */
  overflow: number;
  /** Buttons measured in the DOM; `null` when layout is not available yet. */
  buttons: { total: number; under: number } | null;
  /**
   * Painted area of leaf layers, in logical px². Count-based coverage says a
   * page can be 90% RCL nodes and 30% of its pixels; this is the second number.
   */
  painted: { rcl: number; total: number } | null;
  measured: boolean;
}

export interface FrameAuditMap {
  [frameId: string]: FrameAudit;
}

/** Boot-time shape so the panel never reads undefined before the first measure. */
export function emptyAudit(frames: DesignFrame[]): FrameAuditMap {
  const map: FrameAuditMap = {};
  for (const frame of frames) {
    map[frame.id] = { overflow: 0, buttons: null, painted: null, measured: false };
  }
  return map;
}

export interface AreaShare {
  rcl: number;
  total: number;
  /** rcl / total, clamped to 0..1. */
  ratio: number;
}

/**
 * Aggregate painted area across measured frames. Returns `null` until at least
 * one frame has layout, so the studio shows nothing rather than a fake 0%.
 */
export function paintedShare(audits: FrameAuditMap): AreaShare | null {
  let rcl = 0;
  let total = 0;
  for (const audit of Object.values(audits)) {
    if (!audit.painted) continue;
    rcl += audit.painted.rcl;
    total += audit.painted.total;
  }
  if (total <= 0) return null;
  return { rcl, total, ratio: Math.min(1, Math.max(0, rcl / total)) };
}

export interface CheckVerdict {
  verdict: "pass" | "warn" | "fail" | "unmeasured";
}

export function layoutVerdicts(audit: FrameAudit | undefined, missingStates: number): {
  overflow: CheckVerdict["verdict"];
  targets: CheckVerdict["verdict"];
  states: CheckVerdict["verdict"];
} {
  const overflow = !audit?.measured ? "unmeasured" : audit.overflow > 0 ? "fail" : "pass";
  const targets = !audit?.buttons
    ? "unmeasured"
    : audit.buttons.under > 0
      ? "warn"
      : "pass";
  const states = missingStates > 0 ? "warn" : "pass";
  return { overflow, targets, states };
}

/**
 * A component slot is not a primary action, and a `component=` query param is
 * not captured evidence. Both keep their honest verdict until the design
 * itself supplies the thing being claimed.
 */
export function primaryActionVerdict(primaryActions: number): CheckVerdict["verdict"] {
  return primaryActions > 0 ? "pass" : "warn";
}

export function captureEvidenceVerdict(hasCapture: boolean): CheckVerdict["verdict"] {
  return hasCapture ? "pass" : "unmeasured";
}
