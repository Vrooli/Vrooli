/** @vrooliComponentSource primitives.icon */
import type { SVGProps } from "react";
import {
  ICON_REGISTRY,
  iconSize,
  type IconName,
} from "../../../../foundations/IconRegistry/versions/1.0.0/IconRegistry";
import { SEMANTIC_TOKENS } from "../../../../foundations/Tokens/versions/1.0.0/Tokens";

export type { IconName };

type SVGElementProps = SVGProps<SVGSVGElement>;
export type IconSize = "sm" | "md" | "lg";
export type IconTone = "default" | "muted" | "accent" | "danger";

const toneColors: Record<IconTone, string> = {
  default: SEMANTIC_TOKENS.foreground,
  muted: SEMANTIC_TOKENS.muted,
  accent: SEMANTIC_TOKENS.primary,
  danger: SEMANTIC_TOKENS.danger,
};

export function Icon({
  name,
  label,
  size = "md",
  tone = "default",
  style,
  ...props
}: {
  name: IconName;
  label?: string;
  size?: IconSize;
  tone?: IconTone;
} & SVGElementProps) {
  const icon = ICON_REGISTRY[name];
  return (
    <svg
      {...props}
      aria-hidden={!label}
      aria-label={label}
      role={label ? "img" : undefined}
      viewBox={icon.viewBox}
      width={props.width ?? iconSize(size)}
      height={props.height ?? iconSize(size)}
      fill="none"
      stroke="currentColor"
      strokeWidth={props.strokeWidth ?? 2}
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{ color: toneColors[tone], ...style }}
      data-icon={name}
    >
      <path d={icon.path} />
    </svg>
  );
}
