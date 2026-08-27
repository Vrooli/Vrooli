/**
 * @libraryId react-component-library:MotionPrimitive
 * @displayName MotionPrimitive
 * @description A tokenized animated element with reduced-motion policy and direct motion-value binding without frame-by-frame React renders.
 * @version 1.0.1
 * @tags ["primitive","motion","reduced-motion","token-bound","no-layout-animation"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource motion.motion-primitive */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import {
  createElement,
  forwardRef,
  useEffect,
  useRef,
  type CSSProperties,
  type ElementType,
  type HTMLAttributes,
  type ReactNode,
  type Ref,
} from "react";
import { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1.0.0";

export type MotionVariant =
  | "none"
  | "fade"
  | "scale"
  | "slide-up"
  | "slide-down"
  | "slide-inline"
  | "blur";
export type MotionDuration = "instant" | "quick" | "moderate" | "deliberate";
export type MotionScalar = string | number;

export interface MotionValue<T extends MotionScalar = MotionScalar> {
  get: () => T;
  set: (next: T) => void;
  subscribe: (listener: (next: T) => void) => () => void;
}

export function createMotionValue<T extends MotionScalar>(initial: T): MotionValue<T> {
  let current = initial;
  const listeners = new Set<(next: T) => void>();
  return {
    get: () => current,
    set: (next) => {
      if (Object.is(current, next)) return;
      current = next;
      listeners.forEach((listener) => listener(next));
    },
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}

export function useMotionValue<T extends MotionScalar>(initial: T): MotionValue<T> {
  const valueRef = useRef<MotionValue<T> | null>(null);
  if (!valueRef.current) valueRef.current = createMotionValue(initial);
  return valueRef.current;
}

export interface MotionPrimitiveProps extends HTMLAttributes<HTMLElement> {
  children?: ReactNode;
  as?: ElementType;
  variant?: MotionVariant;
  active?: boolean;
  duration?: MotionDuration | number;
  reducedMotion?: "respect" | "always" | "never";
  motionValues?: Record<string, MotionValue>;
}

const motionStyles = `
[data-rcl-motion] { --rcl-motion-duration: var(--dur-moderate, 280ms); --rcl-motion-ease: var(--ease-standard, cubic-bezier(.2, .8, .2, 1)); transition-property: opacity, transform, filter; transition-duration: var(--rcl-motion-duration); transition-timing-function: var(--rcl-motion-ease); will-change: opacity, transform, filter; }
[data-rcl-motion][data-motion-variant="fade"][data-motion-active="false"] { opacity: 0; }
[data-rcl-motion][data-motion-variant="scale"][data-motion-active="false"] { opacity: 0; transform: scale(.96); }
[data-rcl-motion][data-motion-variant="slide-up"][data-motion-active="false"] { opacity: 0; transform: translateY(var(--space-sm, 12px)); }
[data-rcl-motion][data-motion-variant="slide-down"][data-motion-active="false"] { opacity: 0; transform: translateY(calc(var(--space-sm, 12px) * -1)); }
[data-rcl-motion][data-motion-variant="slide-inline"][data-motion-active="false"] { opacity: 0; transform: translateX(var(--space-sm, 12px)); }
[data-rcl-motion][data-motion-variant="blur"][data-motion-active="false"] { opacity: 0; filter: blur(var(--space-2xs, 8px)); }
[data-rcl-motion][data-motion-reduced="true"] { transition: none; animation: none; opacity: 1; transform: none; filter: none; }

`;

function resolveDuration(duration: MotionPrimitiveProps["duration"]) {
  if (typeof duration === "number" && Number.isFinite(duration)) {
    return `${Math.min(Math.max(duration, 0), 2000)}ms`;
  }
  return duration ? `var(--dur-${duration})` : "var(--dur-moderate, 280ms)";
}

function applyMotionValue(element: HTMLElement, property: string, value: MotionScalar) {
  element.style.setProperty(property, typeof value === "number" ? `${value}px` : value);
}

function assignRef<T>(ref: Ref<T> | undefined, value: T | null) {
  if (typeof ref === "function") ref(value);
  else if (ref) Object.assign(ref, { current: value });
}

export const MotionPrimitive = forwardRef<HTMLElement, MotionPrimitiveProps>(
  function MotionPrimitive(
    {
      children,
      as: Component = "div",
      variant = "fade",
      active = true,
      duration,
      reducedMotion = "respect",
      motionValues,
      style,
      ...props
    },
    forwardedRef,
  ) {
    const reducedPreference = useReducedMotion();
    const elementRef = useRef<HTMLElement | null>(null);
    const reduced =
      reducedMotion === "always" || (reducedMotion === "respect" && reducedPreference);
    const setRef = (element: HTMLElement | null) => {
      elementRef.current = element;
      assignRef(forwardedRef, element);
    };

    useEffect(() => {
      const element = elementRef.current;
      if (!element || !motionValues) return;
      const cleanups = Object.entries(motionValues).map(([property, value]) => {
        applyMotionValue(element, property, value.get());
        return value.subscribe((next) => applyMotionValue(element, property, next));
      });
      return () => cleanups.forEach((cleanup) => cleanup());
    }, [motionValues]);

    const customProperties = {
      "--rcl-motion-duration": resolveDuration(duration),
      ...style,
    } as CSSProperties;
    return createElement(
      Component,
      {
        ...props,
        ref: setRef,
        style: customProperties,
        "data-testid": "motion.motion-primitive",
        "data-rcl-motion": true,
        "data-motion-variant": variant,
        "data-motion-active": active || reduced ? "true" : "false",
        "data-motion-reduced": reduced ? "true" : "false",
        "data-motion-no-layout-animation": true,
      },
      <StyleSheet name="motionprimitive-1-0-1-1" css={motionStyles} />,
      children,
    );
  },
);
