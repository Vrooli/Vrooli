/**
 * @libraryId react-component-library:Icon
 * @displayName Icon
 * @description Icon resolves named glyphs through the semantic icon registry.
 * @version 1.1.2
 * @tags ["primitive","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource primitives.icon */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { SVGProps } from "react";
import "./Icon.css";
import {
  ICON_REGISTRY,
  iconSize,
  type IconName,
} from "@vrooli/react-component-library/IconRegistry/1.0.0";

export type { IconName };

type SVGElementProps = SVGProps<SVGSVGElement>;
export type IconSize = "sm" | "md" | "lg";
export type IconTone = "default" | "muted" | "accent" | "danger";

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
export const Icon = withClassName(function Icon({
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
    <svg data-testid="primitives.icon"
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
      data-icon-size={size}
      data-icon-tone={tone}
      style={
        props.width || props.height
          ? { inlineSize: props.width ?? resolved, blockSize: props.height ?? resolved, ...style }
          : style
      }
      data-icon={name}
    >
      <path d={icon.path} />
    </svg>
  );
});
