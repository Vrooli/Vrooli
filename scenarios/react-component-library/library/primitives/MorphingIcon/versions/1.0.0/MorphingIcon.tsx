/** @vrooliComponentSource react-component-library:MorphingIcon */
import { useEffect, useRef, useState, type CSSProperties } from "react";
import {
  Presence,
  type PresenceDuration,
} from "../../../Presence/versions/1.0.0/Presence";
import type { IconName } from "../../../../foundations/IconRegistry/versions/1.0.0/IconRegistry";
import { useReducedMotion } from "../../../../hooks/useReducedMotion/versions/1.0.0/useReducedMotion";
import { normalizeIconGeometry, type NormalizedIconGeometry } from "./geometry";
import {
  chooseTransitionStrategy,
  type MorphingIconMode,
  type MorphingIconStrategy,
} from "./strategy";
import { animatePathMorph } from "./execution";

export interface MorphingIconProps {
  /** The semantic icon the control currently represents. */
  icon: IconName;
  /** Optional starting icon for a deterministic first transition. */
  from?: IconName;
  strategy?: MorphingIconStrategy;
  duration?: PresenceDuration | number;
  size?: "sm" | "md" | "lg" | number;
  label?: string;
  className?: string;
  style?: CSSProperties;
}

const durationTokens: Record<PresenceDuration, number> = {
  instant: 120,
  quick: 180,
  moderate: 280,
  deliberate: 400,
};

const styles = `
  [data-rcl-morphing-icon] {
    --rcl-morphing-icon-size: var(--icon-size-md, 1.25rem);
    --rcl-morphing-icon-duration: var(--dur-moderate, 280ms);
    display: inline-grid;
    inline-size: var(--rcl-morphing-icon-size);
    block-size: var(--rcl-morphing-icon-size);
    flex: 0 0 auto;
    place-items: center;
    position: relative;
    vertical-align: middle;
  }
  [data-rcl-morphing-icon] > svg,
  [data-rcl-morphing-icon] > [data-rcl-presence] > svg {
    display: block;
    inline-size: 100%;
    block-size: 100%;
    overflow: visible;
  }
  [data-rcl-morphing-icon][data-transition-mode="crossfade"] > [data-rcl-presence] {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    pointer-events: none;
  }
  [data-rcl-morphing-icon][data-transition-mode="transform"] > svg {
    animation: rcl-morphing-icon-transform var(--rcl-morphing-icon-duration) var(--ease-standard, cubic-bezier(.2, .8, .2, 1)) both;
  }
  @keyframes rcl-morphing-icon-transform {
    0% { opacity: .35; transform: rotate(-12deg) scale(.78); }
    58% { opacity: 1; transform: rotate(3deg) scale(1.06); }
    100% { opacity: 1; transform: rotate(0) scale(1); }
  }
  @media (prefers-reduced-motion: reduce) {
    [data-rcl-morphing-icon] > svg,
    [data-rcl-morphing-icon] > [data-rcl-presence] > svg { animation: none !important; }
  }
`;

function durationMs(value: MorphingIconProps["duration"]) {
  if (typeof value === "number") {
    return Number.isFinite(value)
      ? Math.max(0, Math.min(value, 2000))
      : durationTokens.moderate;
  }
  if (typeof value === "string") return durationTokens[value];
  return durationTokens.moderate;
}

function geometry(name: IconName): NormalizedIconGeometry {
  return normalizeIconGeometry(name);
}

function sizeValue(size: MorphingIconProps["size"]) {
  if (typeof size === "number") return `${Math.max(12, Math.min(size, 64))}px`;
  if (size === "sm") return "var(--icon-size-sm, 1rem)";
  if (size === "lg") return "var(--icon-size-lg, 1.5rem)";
  return "var(--icon-size-md, 1.25rem)";
}

export function MorphingIcon({
  icon,
  from,
  strategy = "auto",
  duration = "moderate",
  size = "md",
  label,
  className,
  style,
}: MorphingIconProps) {
  const reducedMotion = useReducedMotion();
  const previousIcon = useRef<IconName>(from ?? icon);
  const pathRef = useRef<SVGPathElement>(null);
  const [displayIcon, setDisplayIcon] = useState<IconName>(from ?? icon);
  const [mode, setMode] = useState<MorphingIconMode | null>(null);
  const [transitionKey, setTransitionKey] = useState(0);
  const [crossfadeSource, setCrossfadeSource] =
    useState<NormalizedIconGeometry | null>(null);
  const [crossfadePresent, setCrossfadePresent] = useState(false);
  const milliseconds = durationMs(duration);

  useEffect(() => {
    const sourceName = previousIcon.current;
    const sourceGeometry = geometry(sourceName);
    const targetGeometry = geometry(icon);
    const plan = chooseTransitionStrategy(
      strategy,
      sourceGeometry,
      targetGeometry,
    );
    previousIcon.current = icon;
    if (sourceName === icon) {
      setDisplayIcon(icon);
      setMode(null);
      return undefined;
    }
    setMode(plan.mode);
    setTransitionKey((value) => value + 1);
    if (plan.mode === "crossfade") {
      setCrossfadeSource(sourceGeometry);
      setCrossfadePresent(true);
    }
    if (plan.mode === "morph") {
      return animatePathMorph(
        pathRef.current,
        sourceGeometry,
        targetGeometry,
        milliseconds,
        reducedMotion,
        () => {
          setDisplayIcon(icon);
          setMode(null);
        },
      );
    }
    setDisplayIcon(icon);
    if (reducedMotion || milliseconds === 0) {
      setCrossfadePresent(false);
      setMode(null);
      return undefined;
    }
    const timer = window.setTimeout(() => {
      if (plan.mode === "crossfade") {
        setCrossfadePresent(false);
      } else {
        setMode(null);
      }
    }, milliseconds);
    return () => window.clearTimeout(timer);
  }, [icon, milliseconds, reducedMotion, strategy]);

  const displayGeometry = geometry(displayIcon);
  const customStyle = {
    "--rcl-morphing-icon-size": sizeValue(size),
    "--rcl-morphing-icon-duration": `var(--dur-${typeof duration === "string" ? duration : "moderate"}, ${milliseconds}ms)`,
    ...style,
  } as CSSProperties;

  return (
    <>
      <style data-rcl-morphing-icon-styles>{styles}</style>
      <span
        className={className}
        style={customStyle}
        data-rcl-morphing-icon
        data-transition-mode={mode ?? undefined}
        data-transition-key={transitionKey}
        aria-hidden={!label}
        aria-label={label}
        role={label ? "img" : undefined}
      >
        <svg
          aria-hidden="true"
          viewBox={displayGeometry.definition.viewBox}
          fill="none"
          stroke="currentColor"
          strokeLinecap="round"
          strokeLinejoin="round"
          focusable="false"
        >
          <path ref={pathRef} d={displayGeometry.d} />
        </svg>
        {mode === "crossfade" && crossfadeSource ? (
          <Presence
            as="span"
            present={crossfadePresent}
            keepMounted={false}
            initial={false}
            duration={duration}
            onExitComplete={() => {
              setCrossfadeSource(null);
              setMode(null);
            }}
            style={{ position: "absolute", inset: 0 }}
          >
            <svg
              aria-hidden="true"
              viewBox={crossfadeSource.definition.viewBox}
              fill="none"
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
              focusable="false"
            >
              <path d={crossfadeSource.d} />
            </svg>
          </Presence>
        ) : null}
      </span>
    </>
  );
}
