/** @vrooliComponentSource react-component-library:MorphingIcon */
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { useReducedMotion } from "../../../../hooks/useReducedMotion/versions/1.0.0/useReducedMotion";
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

export function MorphingIcon({
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
    if (
      reducedMotion ||
      strategy !== "morph" ||
      duration === 0 ||
      Math.sign(duration) === -1
    ) {
      current.current = target;
      setGeometry(target);
      return undefined;
    }
    const started = performance.now();
    const tick = (now: number) => {
      const raw = Math.min(1, (now - started) / duration);
      const eased = 1 - Math.pow(1 - raw, 3);
      const next = interpolateGeometry(source, target, eased);
      current.current = next;
      setGeometry(next);
      if (raw >= 1) {
        current.current = target;
        setGeometry(target);
      } else frame.current = requestAnimationFrame(tick);
    };
    frame.current = requestAnimationFrame(tick);
    return () => {
      if (frame.current !== undefined) cancelAnimationFrame(frame.current);
    };
  }, [icon, strategy, duration, reducedMotion]);
  return (
    <span
      className={className}
      style={{
        display: "inline-grid",
        placeItems: "center",
        inlineSize: sizeValue(size),
        blockSize: sizeValue(size),
        verticalAlign: "middle",
        ...style,
      }}
      aria-hidden={!label}
      aria-label={label}
      role={label ? "img" : undefined}
      data-rcl-morphing-icon
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
          <path
            key={index}
            d={geometryPath(subpath)}
            opacity={subpath.opacity}
          />
        ))}
      </svg>
    </span>
  );
}
