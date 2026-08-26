/**
 * @libraryId react-component-library:MorphingIcon
 * @displayName MorphingIcon
 * @description An icon transition that chooses true path morphing when geometry is compatible and a polished fallback when it is not.
 * @version 2.0.9
 * @tags ["primitive","motion","icon","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource react-component-library:MorphingIcon */
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1.0.0";
import "./MorphingIcon.css";
import {
  geometryPath,
  interpolateGeometry,
  normalizeIcon,
  type MorphingIconName,
} from "./geometry";

export type MorphingIconStrategy = "morph" | "crossfade" | "transform";
export interface MorphingIconProps {
  icon: MorphingIconName;
  from?: MorphingIconName;
  strategy?: MorphingIconStrategy;
  duration?: number;
  size?: "sm" | "md" | "lg" | number;
  label?: string;
  className?: string;
  style?: CSSProperties;
}

const sizeValue = (size: MorphingIconProps["size"]) =>
  typeof size === "number"
    ? `${Math.max(12, Math.min(size, 64))}px`
    : size === "sm"
      ? "var(--icon-size-sm, 1rem)"
      : size === "lg"
        ? "var(--icon-size-lg, 1.5rem)"
        : "var(--icon-size-md, 1.25rem)";

function springProgress(elapsed: number, duration: number) {
  const seconds = elapsed / 1000;
  if (elapsed >= duration * 1.8) return 1;
  const damping = 0.8;
  const frequency = 12 / (duration / 1000);
  const dampedFrequency = frequency * Math.sqrt(1 - damping * damping);
  const envelope = Math.exp(-damping * frequency * seconds);
  const displacement =
    envelope *
    (Math.cos(dampedFrequency * seconds) +
      ((damping * frequency) / dampedFrequency) * Math.sin(dampedFrequency * seconds));
  return Math.max(0, Math.min(1.04, 1 - displacement));
}

export const MorphingIcon = withClassName(function MorphingIcon({
  icon,
  from,
  strategy = "morph",
  duration = 420,
  size = "md",
  label,
  className,
  style,
}: MorphingIconProps) {
  const reducedMotion = useReducedMotion();
  const [geometry, setGeometry] = useState(() => normalizeIcon(from ?? icon));
  const current = useRef(geometry);
  const frame = useRef(undefined as number | undefined);

  useEffect(() => {
    const target = normalizeIcon(icon);
    const source = current.current;
    if (reducedMotion || strategy !== "morph" || duration === 0 || Math.sign(duration) === -1) {
      current.current = target;
      setGeometry(target);
      return undefined;
    }
    const started = performance.now();
    const tick = (now: number) => {
      const progress = springProgress(now - started, duration);
      const next = interpolateGeometry(source, target, progress);
      current.current = next;
      setGeometry(next);
      if (progress >= 1) {
        current.current = target;
        setGeometry(target);
      } else {
        frame.current = requestAnimationFrame(tick);
      }
    };
    frame.current = requestAnimationFrame(tick);
    return () => {
      if (frame.current !== undefined) cancelAnimationFrame(frame.current);
    };
  }, [icon, strategy, duration, reducedMotion]);

  const computedStyle = { inlineSize: sizeValue(size), blockSize: sizeValue(size), ...style };

  return (
    <span data-testid="motion.morphing-icon"
      className={className}
      data-rcl-morphing-icon
      data-size={size}
      style={computedStyle}
      aria-hidden={!label}
      aria-label={label}
      role={label ? "img" : undefined}
      data-rcl-transition-mode={strategy}
    >
      <svg
        viewBox={geometry.viewBox}
        width="100%"
        height="100%"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
        focusable="false"
      >
        {geometry.subpaths.map((subpath, index) => (
          <path key={index} d={geometryPath(subpath)} opacity={subpath.opacity} />
        ))}
      </svg>
    </span>
  );
});
