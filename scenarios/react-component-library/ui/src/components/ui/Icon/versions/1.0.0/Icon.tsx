/**
 * @vrooliComponentSource primitives.icon
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption 3300c045-811e-474a-8de2-3994ae86b436
 * @vrooliComponentAppliedAt 2026-08-18T01:12:35Z
 * @vrooliComponentSourceSha256 b6d27671e51cf4c9f345910559849601e7790059b77a0ae3320c22cb882a0a32
 * @vrooliComponentDriftHash b6b30f771ba055fde0ccc98cb9ed6cd0bd549e23540e905a6e6bfcf6f9b048ec
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

/**
 * 1.1.0 — size through CSS rather than SVG presentation attributes.
 *
 * 1.0.0 applied the token to `width=`/`height=`, but those are SVG geometry
 * attributes and their grammar is `<length>`; `var()` is not a length, so the
 * browser rejected the whole attribute:
 *
 *   Error: <svg> attribute width: Expected length, "var(--icon-size-md, 1.25rem)"
 *
 * A rejected geometry attribute means the element has no author-specified
 * size, so it falls back to the replaced-element default (300x150) and renders
 * enormous. `iconSize()` has always returned a `var()` expression, so every
 * icon rendered through this primitive was affected — the defect is invisible
 * in any test that does not measure the laid-out box.
 *
 * The CSS `inline-size`/`block-size` properties do accept `var()`, and being
 * logical they also mirror correctly under `dir="rtl"`. Explicit width/height
 * props still pass through as attributes for callers that supply real lengths,
 * and are mirrored into CSS so the two cannot disagree.
 */
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
  const resolved = iconSize(size);
  return (
    <svg
      {...props}
      aria-hidden={!label}
      aria-label={label}
      role={label ? "img" : undefined}
      viewBox={icon.viewBox}
      fill="none"
      stroke="currentColor"
      strokeWidth={props.strokeWidth ?? 2}
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{
        inlineSize: props.width ?? resolved,
        blockSize: props.height ?? resolved,
        color: toneColors[tone],
        ...style,
      }}
      data-icon={name}
    >
      <path d={icon.path} />
    </svg>
  );
}
