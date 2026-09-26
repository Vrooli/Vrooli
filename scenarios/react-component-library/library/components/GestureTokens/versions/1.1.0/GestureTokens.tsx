/**
 * @libraryId react-component-library:GestureTokens
 * @displayName GestureTokens
 * @description
 * @version 1.1.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource foundations.gesture-tokens */

export interface GestureFeel {
  /** Movement required before a pointer commits to an axis. */
  axisSlop: number;
  /** Tail velocity that qualifies a release as a flick, in pixels/ms. */
  flickVelocity: number;
  /** Portion of overtravel rendered beyond the final stage. */
  resistance: number;
  /** Travel required to dismiss a surface. */
  dismissThreshold: number;
  /** Delay before fine-pointer hover opens a child surface. */
  hoverOpenDelay: number;
  /** Delay before an abandoned hover closes. */
  hoverCloseDelay: number;
  /** Fuse for a pointer parked in the safe polygon. */
  safePolygonFuse: number;
  /** Delay before a stationary pointer becomes a long press. */
  longPressDelay: number;
  /** Movement that cancels a long press, in pixels. */
  longPressMoveTolerance: number;
}

/**
 * One shared interaction vocabulary. These values preserve the established
 * library feel while making future tuning explicit and reviewable.
 */
export const GestureTokens: Readonly<GestureFeel> = Object.freeze({
  axisSlop: 8,
  flickVelocity: 0.5,
  resistance: 0.32,
  dismissThreshold: 96,
  hoverOpenDelay: 280,
  hoverCloseDelay: 100,
  safePolygonFuse: 300,
  longPressDelay: 450,
  longPressMoveTolerance: 10,
});

export function resolveGestureFeel(overrides: Partial<GestureFeel> = {}): GestureFeel {
  return { ...GestureTokens, ...overrides };
}

export default GestureTokens;
