/**
 * @vrooliComponentSource react-component-library:Icon
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 3300c045-811e-474a-8de2-3994ae86b436
 * @vrooliComponentAppliedAt 2026-08-12T12:59:50Z
 * @vrooliComponentSourceSha256 35659cb3622fb3e5d05927b3b01d98c96240df28eb05c644a2b2e3a0f9221c1a
 * @vrooliComponentDriftHash 1f5a2e6aa98a0a1bd6f308a0edfc9a4738c5837892c514f59a4f33aff5b1b2ad
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { SVGProps } from "react";
import { ICON_REGISTRY, iconSize, type IconName } from "./foundations/IconRegistry";
import { SEMANTIC_TOKENS } from "./foundations/Tokens";

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
