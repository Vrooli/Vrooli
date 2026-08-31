/**
 * @libraryId react-component-library:GestureDirection
 * @displayName Gesture Direction
 * @version 1.0.0
 * @tags ["foundations","gesture","accessibility","internationalization"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource foundations.gesture-direction */

/**
 * Which way the document's text runs. A locale property.
 */
export type WritingDirection = "ltr" | "rtl";

/**
 * Which logical edge a panel is anchored to. An ergonomic property, chosen by
 * the person using the app rather than by their language.
 *
 * These two are deliberately separate types even though both ultimately answer
 * "left or right?". Deriving one from the other is the mistake this module
 * exists to prevent: a left-handed reader of English still reads left-to-right,
 * so moving their drawer must not mirror their text, and mirroring their text
 * must not be the only way to move their drawer.
 */
export type AnchorEdge = "inline-start" | "inline-end";

/** A resolved side of the screen. */
export type PhysicalEdge = "left" | "right";

/**
 * What a gesture is trying to do, named by outcome rather than by direction so
 * that no consumer has to know which way that is on the current screen.
 */
export type GestureIntent = "dismiss" | "reveal";

/** Movement along the horizontal axis, as a multiplier for a pixel delta. */
export type DirectionSign = -1 | 1;

const MIRRORED: Record<PhysicalEdge, PhysicalEdge> = { left: "right", right: "left" };

/** The physical side opposite the one given. */
export function oppositeEdge(edge: PhysicalEdge): PhysicalEdge {
  return MIRRORED[edge];
}

/**
 * Resolves where an anchored panel physically sits.
 *
 * The truth table is a single XOR, but writing it out is worth the lines: each
 * row is a real configuration someone runs, and the pair that lands on the same
 * side by opposite routes (`rtl` + `inline-start`, and `ltr` + `inline-end`) is
 * exactly the pair that a single-input implementation collapses by accident.
 */
export function resolveAnchorEdge(
  direction: WritingDirection,
  anchor: AnchorEdge,
): PhysicalEdge {
  const startIsLeft = direction === "ltr";
  const anchoredToStart = anchor === "inline-start";
  return anchoredToStart === startIsLeft ? "left" : "right";
}

/**
 * Resolves a semantic intent to the physical direction a finger must travel.
 *
 * A panel is dismissed by pushing it toward the edge it is anchored to, and
 * content behind it is revealed by pulling away from that edge. Keeping both on
 * one axis with opposite signs is what lets a swipeable row live inside a
 * swipe-to-close drawer without any arbitration between them: the sign of the
 * delta already says which gesture the user meant.
 */
export function resolveGestureDirection(
  direction: WritingDirection,
  anchor: AnchorEdge,
  intent: GestureIntent,
): PhysicalEdge {
  const anchored = resolveAnchorEdge(direction, anchor);
  return intent === "dismiss" ? anchored : oppositeEdge(anchored);
}

/**
 * The multiplier that turns travel-toward-`edge` into a signed CSS translation.
 * Screen coordinates grow rightward, so leftward travel is negative.
 */
export function edgeSign(edge: PhysicalEdge): DirectionSign {
  return edge === "left" ? -1 : 1;
}

/**
 * Convenience for the common case: the signed axis a gesture runs along.
 *
 * Multiply a raw pointer delta by this to get distance traveled *in the
 * intended direction*, where negative means the user is moving the wrong way.
 */
export function gestureSign(
  direction: WritingDirection,
  anchor: AnchorEdge,
  intent: GestureIntent,
): DirectionSign {
  return edgeSign(resolveGestureDirection(direction, anchor, intent));
}

/**
 * Normalizes an arbitrary string to a writing direction.
 *
 * `document.documentElement.dir` is frequently `""` (unset) or `"auto"`, and
 * both mean "not RTL" for the purpose of resolving a gesture.
 */
export function normalizeWritingDirection(value: string | null | undefined): WritingDirection {
  return value === "rtl" ? "rtl" : "ltr";
}
