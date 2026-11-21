import type { DesignFrame } from "../designs";
import { walk, type TemplateNode, type TemplateNodeType } from "../templates/spec";

/**
 * Which TemplateNode types resolve to a published RCL primitive.
 *
 * This mirrors the imports in TemplateRenderer: a layer earns the badge only
 * when the same composition renders the shared component, not a lookalike.
 * Layout nodes (page, stack, row, grid, surface) and custom leaves (text,
 * swatch, spacer, divider, slot) stay custom on purpose.
 */
export const RCL_PRIMITIVES: Partial<Record<TemplateNodeType, string>> = {
  avatar: "Avatar",
  button: "Button",
  card: "Card",
  chip: "Chip",
  input: "Input",
  meter: "BoundedMeter",
  stat: "Stat",
};

export function rclPrimitive(type: TemplateNodeType): string | null {
  return RCL_PRIMITIVES[type] ?? null;
}

export interface Coverage {
  /** Layers whose type maps to a published RCL primitive. */
  rcl: number;
  /** Layers that are custom or layout-only. */
  custom: number;
  total: number;
  /** rcl / total, clamped to 0..1; 0 for an empty composition. */
  ratio: number;
}

const EMPTY: Coverage = { rcl: 0, custom: 0, total: 0, ratio: 0 };

/**
 * Count-based coverage for one frame.
 *
 * The page root is the frame itself, not a layer, so it is excluded: a frame
 * with five RCL primitives reads 100%, not 83%.
 */
export function frameCoverage(spec: TemplateNode): Coverage {
  let rcl = 0;
  let total = 0;
  for (const child of spec.children ?? []) {
    walk(child, (current) => {
      total += 1;
      if (rclPrimitive(current.type)) rcl += 1;
    });
  }
  if (total === 0) return EMPTY;
  return { rcl, custom: total - rcl, total, ratio: rcl / total };
}

/** Coverage across every frame of a design; the count that drives the status bar. */
export function designCoverage(frames: DesignFrame[]): Coverage {
  let rcl = 0;
  let total = 0;
  for (const frame of frames) {
    const coverage = frameCoverage(frame.spec);
    rcl += coverage.rcl;
    total += coverage.total;
  }
  if (total === 0) return EMPTY;
  return { rcl, custom: total - rcl, total, ratio: rcl / total };
}

export interface PrimitiveUse {
  type: TemplateNodeType;
  name: string;
  /** Uses inside the reference frame. */
  count: number;
  /** How many frames of the design contain at least one use. */
  frames: number;
}

/**
 * RCL primitives used by one frame, with the design-wide frame count beside
 * them. The left pane's System lens shows the active frame's tree, so `count`
 * is read there; `frames` answers "used by N frames".
 */
export function primitiveUsage(frames: DesignFrame[], referenceFrameId: string): PrimitiveUse[] {
  const reference = frames.find((frame) => frame.id === referenceFrameId) ?? frames[0];
  const counts = new Map<TemplateNodeType, number>();
  if (reference) {
    for (const child of reference.spec.children ?? []) {
      walk(child, (current) => {
        if (!rclPrimitive(current.type)) return;
        counts.set(current.type, (counts.get(current.type) ?? 0) + 1);
      });
    }
  }
  const result: PrimitiveUse[] = [];
  for (const [type, count] of counts) {
    const name = rclPrimitive(type);
    if (!name) continue;
    result.push({
      type,
      name,
      count,
      frames: frames.filter((frame) => containsType(frame.spec, type)).length,
    });
  }
  return result.sort((left, right) => right.count - left.count || left.name.localeCompare(right.name));
}

export interface PatternUse {
  type: TemplateNodeType;
  count: number;
}

const PATTERN_TYPES = new Set<TemplateNodeType>(["stack", "row", "grid", "card", "surface", "slot"]);

/** Grouping layers in one frame: the patterns an implementer would name. */
export function patternUsage(spec: TemplateNode): PatternUse[] {
  const counts = new Map<TemplateNodeType, number>();
  for (const child of spec.children ?? []) {
    walk(child, (current) => {
      if (!PATTERN_TYPES.has(current.type)) return;
      if (current.type !== "slot" && !(current.children && current.children.length > 0)) return;
      counts.set(current.type, (counts.get(current.type) ?? 0) + 1);
    });
  }
  return [...counts]
    .map(([type, count]) => ({ type, count }))
    .sort((left, right) => right.count - left.count || left.type.localeCompare(right.type));
}

export function layerCount(spec: TemplateNode): number {
  return frameCoverage(spec).total;
}

function containsType(spec: TemplateNode, type: TemplateNodeType): boolean {
  let found = false;
  for (const child of spec.children ?? []) {
    walk(child, (current) => {
      if (current.type === type) found = true;
    });
  }
  return found;
}
