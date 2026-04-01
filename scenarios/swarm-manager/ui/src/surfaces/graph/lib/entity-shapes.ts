/**
 * Entity Shapes
 *
 * Maps each entity type to a distinct geometric shape for instant
 * visual differentiation in the graph view. Shape encodes ENTITY TYPE.
 */

import type { GraphEntityType } from "../types";

export type NodeShape = "diamond" | "rectangle" | "hexagon" | "circle" | "pentagon" | "octagon" | "pill";

export interface ShapeDimensions {
  width: number;
  height: number;
}

export const ENTITY_SHAPE_MAP: Record<GraphEntityType, NodeShape> = {
  backlog: "diamond",
  scenario: "rectangle",
  execution: "hexagon",
  initiative: "circle",
  capture: "pentagon",
  "agent-run": "octagon",
  "agent-activity": "pill",
};

/**
 * CSS classes that produce the shape. For shapes using clip-path,
 * the classes are defined in styles.css.
 *
 * Diamond: rotate the container 45deg, counter-rotate inner content.
 * Circle: border-radius 50%.
 * Hexagon/Pentagon/Octagon: clip-path polygons.
 * Pill: rounded-full on a rectangular aspect ratio.
 * Rectangle: standard rounded-lg.
 */
const SHAPE_CLASSES: Record<NodeShape, string> = {
  diamond: "rotate-45",
  rectangle: "rounded-lg",
  hexagon: "clip-hexagon",
  circle: "rounded-full",
  pentagon: "clip-pentagon",
  octagon: "clip-octagon",
  pill: "rounded-full",
};

/**
 * Default dimensions for each shape, used by Dagre layout.
 * Shapes that need equal width/height (diamond, circle, octagon) are square.
 * Rectangular shapes are wider than tall.
 */
const SHAPE_DIMENSIONS: Record<NodeShape, ShapeDimensions> = {
  diamond: { width: 130, height: 130 },
  rectangle: { width: 160, height: 72 },
  hexagon: { width: 160, height: 100 },
  circle: { width: 130, height: 130 },
  pentagon: { width: 150, height: 110 },
  octagon: { width: 150, height: 110 },
  pill: { width: 160, height: 60 },
};

/** Get the CSS class(es) that produce the shape for an entity type. */
export function getShapeClasses(entityType: GraphEntityType): string {
  const shape = ENTITY_SHAPE_MAP[entityType] ?? "rectangle";
  return SHAPE_CLASSES[shape];
}

/** Whether the shape requires inner content to be counter-rotated. */
export function needsContentCounterRotation(entityType: GraphEntityType): boolean {
  return ENTITY_SHAPE_MAP[entityType] === "diamond";
}

/** Get the default dimensions for an entity type's shape. */
export function getShapeDimensions(entityType: GraphEntityType): ShapeDimensions {
  const shape = ENTITY_SHAPE_MAP[entityType] ?? "rectangle";
  return SHAPE_DIMENSIONS[shape];
}

/** Whether the shape uses clip-path (affects shadow/ring behavior). */
export function usesClipPath(entityType: GraphEntityType): boolean {
  const shape = ENTITY_SHAPE_MAP[entityType];
  return shape === "hexagon" || shape === "pentagon" || shape === "octagon";
}

/** All entity shapes with labels, for use in legends/help panels. */
export const ENTITY_SHAPE_INFO: { entityType: GraphEntityType; shape: NodeShape; label: string; shapeClass: string }[] = [
  { entityType: "backlog", shape: "diamond", label: "Backlog", shapeClass: SHAPE_CLASSES.diamond },
  { entityType: "scenario", shape: "rectangle", label: "Scenario", shapeClass: SHAPE_CLASSES.rectangle },
  { entityType: "execution", shape: "hexagon", label: "Execution", shapeClass: SHAPE_CLASSES.hexagon },
  { entityType: "initiative", shape: "circle", label: "Initiative", shapeClass: SHAPE_CLASSES.circle },
  { entityType: "capture", shape: "pentagon", label: "Capture", shapeClass: SHAPE_CLASSES.pentagon },
  { entityType: "agent-run", shape: "octagon", label: "Agent Run", shapeClass: SHAPE_CLASSES.octagon },
  { entityType: "agent-activity", shape: "pill", label: "Activity", shapeClass: SHAPE_CLASSES.pill },
];
