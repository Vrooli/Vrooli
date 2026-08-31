/**
 * @libraryId react-component-library:Presence
 * @displayName Presence
 * @version 1.0.2
 * @tags ["primitive","motion","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:Presence */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import {
  createElement,
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type ElementType,
  type ReactNode,
} from "react";
import { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1";

export type PresencePhase = "entering" | "entered" | "exiting" | "exited";
export type PresenceDuration = "instant" | "quick" | "moderate" | "deliberate";

export interface PresenceProps {
  /** Whether the content should be present. Toggling this value is interruptible. */
  present: boolean;
  children: ReactNode;
  /** The semantic wrapper used to keep the animation boundary explicit. */
  as?: ElementType;
  /** Skip the initial enter animation while preserving later transitions. */
  initial?: boolean;
  /** Keep the boundary mounted and hidden after exit instead of removing it. */
  keepMounted?: boolean;
  /** A shared motion token or an explicit duration for advanced integrations. */
  duration?: PresenceDuration | number;
  /** Exit is intentionally shorter than entrance by default. */
  exitDuration?: PresenceDuration | number;
  className?: string;
  style?: CSSProperties;
  id?: string;
  "aria-label"?: string;
  onExitComplete?: () => void;
}

const durationTokens: Record<PresenceDuration, { css: string; ms: number }> = {
  instant: { css: "var(--dur-instant, 120ms)", ms: 120 },
  quick: { css: "var(--dur-quick, 180ms)", ms: 180 },
  moderate: { css: "var(--dur-moderate, 280ms)", ms: 280 },
  deliberate: { css: "var(--dur-deliberate, 400ms)", ms: 400 },
};

function resolveDuration(duration: PresenceProps["duration"]) {
  if (typeof duration === "number" && Number.isFinite(duration)) {
    const ms = Math.min(Math.max(duration, 0), 2000);
    return { css: `${ms}ms`, ms };
  }
  const token = typeof duration === "string" ? duration : "moderate";
  return durationTokens[token];
}

const styles = `
  [data-rcl-presence] {
    --rcl-presence-duration: var(--dur-moderate, 280ms);
    transform-origin: center;
    will-change: opacity, transform;
  }
  [data-rcl-presence][data-presence-phase="entering"] {
    animation: rcl-presence-enter var(--rcl-presence-duration) var(--ease-enter, cubic-bezier(0, 0, 0, 1)) both;
  }
  [data-rcl-presence][data-presence-phase="exiting"] {
    animation: rcl-presence-exit var(--rcl-presence-exit-duration, var(--rcl-presence-duration)) var(--ease-exit, cubic-bezier(.3, 0, 1, 1)) both;
    pointer-events: none;
  }
  [data-rcl-presence][data-presence-hidden="true"] {
    display: none;
  }
  @keyframes rcl-presence-enter {
    from { opacity: 0; transform: translateY(var(--space-2xs, 8px)) scale(.985); }
    to { opacity: 1; transform: translateY(0) scale(1); }
  }
  @keyframes rcl-presence-exit {
    from { opacity: 1; transform: translateY(0) scale(1); }
    to { opacity: 0; transform: translateY(calc(var(--space-2xs, 8px) * -1)) scale(.985); }
  }

`;

export const Presence = withClassName(function Presence({
  present,
  children,
  as: Component = "div",
  initial = true,
  keepMounted = false,
  duration = "moderate",
  exitDuration,
  className,
  style,
  id,
  "aria-label": ariaLabel,
  onExitComplete,
}: PresenceProps) {
  const reducedMotion = useReducedMotion();
  const firstRender = useRef(true);
  const mountedRef = useRef(present);
  const [mounted, setMounted] = useState(present);
  const [phase, setPhase] = useState<PresencePhase>(() => {
    if (!present) return "exited";
    return initial ? "entering" : "entered";
  });
  const resolvedDuration = resolveDuration(duration);
  const resolvedExitDuration = resolveDuration(
    exitDuration ??
      (duration === "deliberate" ? "moderate" : duration === "moderate" ? "quick" : duration),
  );

  useEffect(() => {
    let frame: number | undefined;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const shouldAnimate = !firstRender.current || initial;
    firstRender.current = false;

    if (present) {
      mountedRef.current = true;
      setMounted(true);
      if (reducedMotion || !shouldAnimate) {
        setPhase("entered");
        return undefined;
      }
      setPhase("entering");
      const advance = () => setPhase("entered");
      if (typeof window !== "undefined" && typeof window.requestAnimationFrame === "function") {
        frame = window.requestAnimationFrame(advance);
      } else {
        timer = setTimeout(advance, 0);
      }
      return () => {
        if (frame !== undefined && typeof window !== "undefined")
          window.cancelAnimationFrame(frame);
        if (timer !== undefined) clearTimeout(timer);
      };
    }

    if (!mountedRef.current) {
      setPhase("exited");
      return undefined;
    }
    if (reducedMotion || resolvedDuration.ms === 0) {
      mountedRef.current = false;
      setMounted(false);
      setPhase("exited");
      onExitComplete?.();
      return undefined;
    }

    setPhase("exiting");
    timer = setTimeout(() => {
      mountedRef.current = false;
      setMounted(false);
      setPhase("exited");
      onExitComplete?.();
    }, resolvedExitDuration.ms);
    return () => clearTimeout(timer);
  }, [
    initial,
    onExitComplete,
    present,
    reducedMotion,
    resolvedDuration.ms,
    resolvedExitDuration.ms,
  ]);

  if (!mounted && !keepMounted) return null;

  const customProperties = {
    "--rcl-presence-duration": resolvedDuration.css,
    "--rcl-presence-exit-duration": resolvedExitDuration.css,
    ...style,
  } as CSSProperties;
  return createElement(
    Component,
    {
      id,
      className,
      style: customProperties,
      "aria-label": ariaLabel,
      "data-testid": "motion.presence",
      "data-rcl-presence": true,
      "data-presence-phase": phase,
      "data-presence-hidden": !mounted || undefined,
    },
    createElement(StyleSheet, { name: "presence", css: styles }),
    children,
  );
});
