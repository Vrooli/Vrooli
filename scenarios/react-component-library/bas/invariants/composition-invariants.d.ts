/**
 * Types for the composition invariants.
 *
 * The implementation is hand-written JavaScript rather than TypeScript because
 * it is injected verbatim into a live page through the BAS `evaluate` action.
 * Compiling it would put a build artefact between the source and the thing that
 * actually runs, so the module stays plain JS and carries its contract here.
 *
 * The `view` parameter is the window whose getComputedStyle should be used. It
 * is passed in rather than read from a global so the invariants can be exercised
 * against a stubbed implementation in jsdom, where layout is not implemented.
 */

/** One composition defect. Mirrors the Go gates.Finding shape. */
export interface CompositionFinding {
  /** Stable identifier, e.g. "composition.duplicate_label". */
  code: string;
  /** What is wrong. */
  message: string;
  /** Short description of the offending element. */
  element: string;
  /** What to do about it, and why it matters. */
  remediation: string;
  /** Repository-relative doc path giving fuller rule context. */
  docsRef: string;
}

/**
 * Minimal surface the invariants need from a window.
 *
 * The style object is described structurally rather than as CSSStyleDeclaration
 * so a test double can supply only the properties under test. The invariants
 * read named properties and call getPropertyValue for custom properties, and
 * nothing else.
 */
export interface StyleLike {
  readonly [property: string]: unknown;
  getPropertyValue?(property: string): string;
}

export interface StyleView {
  getComputedStyle(element: Element): StyleLike;
}

export function checkTypeScaleMonotonicity(root: ParentNode, view: StyleView): CompositionFinding[];
export function checkLabelUniqueness(root: ParentNode, view: StyleView): CompositionFinding[];
export function checkSpacingQuantization(
  root: ParentNode,
  view: StyleView,
  options?: { ramp?: Map<number, string> },
): CompositionFinding[];
export function checkProvidedChromeDuplication(
  root: ParentNode,
  view: StyleView,
  /** How many levels above a component root count as "nearby". Default 3. */
  options?: { scopeDepth?: number },
): CompositionFinding[];
export function readSpacingRamp(root: ParentNode, view: StyleView): Map<number, string>;
export function checkComposition(root: ParentNode, view: StyleView): CompositionFinding[];
