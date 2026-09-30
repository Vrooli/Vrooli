/**
 * Entity Registry
 *
 * Single source of truth for every entity type's visual identity in the graph.
 * Shape encodes ENTITY TYPE; color encodes STATUS (handled in status-colors.ts).
 *
 * Adding a new entity type? Add one entry to ENTITY_REGISTRY. TypeScript
 * enforces completeness (it's Record<GraphEntityType, EntityConfig>), so
 * every downstream consumer — nodeTypes, filters, legends, layout — picks
 * it up automatically. No other file needs a hand-maintained list.
 *
 * All shapes are wider-than-tall for label readability. Clip-path polygons
 * are applied inline (no CSS classes needed).
 */

import type { ElementType } from "react";
import { ENTITY_TYPE_ICONS } from "../../../types/constants";
import type { GraphEntityType } from "../types";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export type NodeShape =
  | "stretched-hexagon"
  | "rounded-rectangle"
  | "notched-rectangle"
  | "stadium"
  | "parallelogram"
  | "trapezoid"
  | "stretched-octagon";

export interface ShapeDimensions {
  width: number;
  height: number;
}

export interface EntityConfig {
  /** Human-readable label for UI display (legend, settings, help panel). */
  label: string;
  /** Short label shown inside the node badge (e.g. "activity" not "agent-activity"). */
  badgeLabel: string;
  /** Shape identifier — must be unique per entity type. */
  shape: NodeShape;
  /** Tailwind class(es) for the shape container (border-radius, etc.).
   *  Ignored when clipPath is set, but kept for non-clipped shapes. */
  cssClass: string;
  /** clip-path polygon() value (just the coordinates, without the wrapper).
   *  null means the shape uses only CSS border-radius. */
  clipPath: string | null;
  /** Dimensions used by Dagre layout and inline node sizing. */
  dimensions: ShapeDimensions;
  /** Lucide icon component for this entity type. */
  icon: ElementType;
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

export const ENTITY_REGISTRY: Record<GraphEntityType, EntityConfig> = {
  backlog: {
    label: "Backlog",
    badgeLabel: "backlog",
    shape: "stretched-hexagon",
    cssClass: "",
    clipPath: "5% 50%, 15% 0%, 85% 0%, 95% 50%, 85% 100%, 15% 100%",
    dimensions: { width: 170, height: 76 },
    icon: ENTITY_TYPE_ICONS.backlog,
  },
  goal: {
    label: "Goals",
    badgeLabel: "goal",
    shape: "rounded-rectangle",
    cssClass: "rounded-lg",
    clipPath: null,
    dimensions: { width: 168, height: 76 },
    icon: ENTITY_TYPE_ICONS.goal,
  },
  scenario: {
    label: "Scenarios",
    badgeLabel: "scenario",
    shape: "notched-rectangle",
    cssClass: "",
    clipPath: "8% 0%, 92% 0%, 100% 12%, 100% 88%, 92% 100%, 8% 100%, 0% 88%, 0% 12%",
    dimensions: { width: 164, height: 74 },
    icon: ENTITY_TYPE_ICONS.scenario,
  },
  capture: {
    label: "Captures",
    badgeLabel: "capture",
    shape: "stadium",
    cssClass: "rounded-full",
    clipPath: null,
    dimensions: { width: 166, height: 72 },
    icon: ENTITY_TYPE_ICONS.capture,
  },
  execution: {
    label: "Execution",
    badgeLabel: "execution",
    shape: "parallelogram",
    cssClass: "",
    clipPath: "12% 0%, 100% 0%, 88% 100%, 0% 100%",
    dimensions: { width: 170, height: 74 },
    icon: ENTITY_TYPE_ICONS.execution,
  },
  "agent-run": {
    label: "Runs",
    badgeLabel: "run",
    shape: "trapezoid",
    cssClass: "",
    clipPath: "8% 0%, 92% 0%, 100% 100%, 0% 100%",
    dimensions: { width: 168, height: 76 },
    icon: ENTITY_TYPE_ICONS["agent-run"],
  },
  "agent-activity": {
    label: "Activities",
    badgeLabel: "activity",
    shape: "stretched-octagon",
    cssClass: "",
    clipPath: "10% 0%, 90% 0%, 100% 30%, 100% 70%, 90% 100%, 10% 100%, 0% 70%, 0% 30%",
    dimensions: { width: 166, height: 74 },
    icon: ENTITY_TYPE_ICONS["agent-activity"],
  },
};

// ---------------------------------------------------------------------------
// Derived constants
// ---------------------------------------------------------------------------

/** All entity types, derived from the registry. */
export const GRAPH_ENTITY_TYPES = Object.keys(ENTITY_REGISTRY) as GraphEntityType[];

/** Shape info for legends and help panels. */
export const ENTITY_SHAPE_INFO = GRAPH_ENTITY_TYPES.map((et) => ({
  entityType: et,
  shape: ENTITY_REGISTRY[et].shape,
  label: ENTITY_REGISTRY[et].label,
  cssClass: ENTITY_REGISTRY[et].cssClass,
  clipPath: ENTITY_REGISTRY[et].clipPath,
}));

// ---------------------------------------------------------------------------
// Accessor functions
// ---------------------------------------------------------------------------

/** CSS class(es) for the shape container (border-radius). Empty for clip-path-only shapes. */
export function getShapeClasses(entityType: GraphEntityType): string {
  return ENTITY_REGISTRY[entityType].cssClass;
}

/** Dimensions for Dagre layout and inline node sizing. */
export function getShapeDimensions(entityType: GraphEntityType): ShapeDimensions {
  return ENTITY_REGISTRY[entityType].dimensions;
}

/** Whether the shape uses an inline clip-path (affects shadow/ring behavior). */
export function usesClipPath(entityType: GraphEntityType): boolean {
  return ENTITY_REGISTRY[entityType].clipPath !== null;
}

/** Inline style object for clip-path, or undefined for non-clipped shapes. */
export function getClipPathStyle(entityType: GraphEntityType): React.CSSProperties | undefined {
  const cp = ENTITY_REGISTRY[entityType].clipPath;
  return cp ? { clipPath: `polygon(${cp})` } : undefined;
}

/** Lucide icon component for an entity type. */
export function getEntityIcon(entityType: GraphEntityType): ElementType {
  return ENTITY_REGISTRY[entityType].icon;
}

/** Short label for the badge inside graph nodes. */
export function getEntityBadgeLabel(entityType: GraphEntityType): string {
  return ENTITY_REGISTRY[entityType].badgeLabel;
}

/** Human-readable label for UI display. */
export function getEntityLabel(entityType: GraphEntityType): string {
  return ENTITY_REGISTRY[entityType].label;
}
