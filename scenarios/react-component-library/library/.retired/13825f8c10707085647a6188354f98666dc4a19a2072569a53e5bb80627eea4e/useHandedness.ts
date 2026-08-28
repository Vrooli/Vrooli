/**
 * @libraryId react-component-library:useHandedness
 * @displayName useHandedness
 * @description Carries the reach-side preference that decides which edge panels anchor to, independently of the document writing direction.
 * @version 1.0.0
 * @tags ["runtime","accessibility","gesture"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource hooks.use-handedness */
import { createContext, createElement, useContext, useMemo, type ReactNode } from "react";

import {
  resolveAnchorEdge,
  resolveGestureDirection,
  gestureSign,
  type AnchorEdge,
  type DirectionSign,
  type GestureIntent,
  type PhysicalEdge,
} from "@vrooli/react-component-library/GestureDirection/1.0.0";
import { useDirection } from "@vrooli/react-component-library/useDirection/2.0.0";

export type { AnchorEdge, GestureIntent, PhysicalEdge };

/**
 * Which side of the screen the user prefers to reach.
 *
 * This is an ergonomic setting, not a locale one. It is a separate concept from
 * writing direction on purpose: someone reading English with their phone in
 * their left hand wants the drawer moved without their text being mirrored, and
 * expressing that through `dir` would mirror both.
 */
export const DEFAULT_HANDEDNESS: AnchorEdge = "inline-start";

const HandednessContext = createContext<AnchorEdge>(DEFAULT_HANDEDNESS);

export interface HandednessProviderProps {
  /** The edge anchored surfaces should sit against. */
  value?: AnchorEdge;
  children?: ReactNode;
}

/**
 * Publishes the reach-side preference to every anchored surface below it.
 *
 * The library deliberately does not persist this. Where a preference is stored,
 * and whether it syncs across a user's devices, is an application decision; the
 * library's job is to make one answer reach every consumer consistently.
 */
export function HandednessProvider({ value, children }: HandednessProviderProps) {
  return createElement(
    HandednessContext.Provider,
    { value: value ?? DEFAULT_HANDEDNESS },
    children,
  );
}

/**
 * The current reach-side preference, defaulting to the inline start edge when
 * no provider is mounted so an app that never opts in behaves exactly as it did
 * before this hook existed.
 */
export function useHandedness(): AnchorEdge {
  return useContext(HandednessContext);
}

export interface ResolvedGestureDirection {
  /** The physical side an anchored surface sits against. */
  anchorEdge: PhysicalEdge;
  /** The physical direction the gesture travels. */
  direction: PhysicalEdge;
  /** `-1` for leftward travel, `1` for rightward. */
  sign: DirectionSign;
}

/**
 * Resolves a semantic gesture intent against both inputs at once.
 *
 * Components should reach for this rather than reading `dir` themselves. A
 * component that resolves direction privately — as SidebarShell did, with its
 * own `getComputedStyle(panel).direction` check — can silently disagree with the
 * stylesheet beside it, which is exactly how a drawer ends up gesturing one way
 * and animating the other.
 */
export function useGestureDirection(intent: GestureIntent): ResolvedGestureDirection {
  const writingDirection = useDirection();
  const handedness = useHandedness();
  return useMemo(
    () => ({
      anchorEdge: resolveAnchorEdge(writingDirection, handedness),
      direction: resolveGestureDirection(writingDirection, handedness, intent),
      sign: gestureSign(writingDirection, handedness, intent),
    }),
    [writingDirection, handedness, intent],
  );
}
