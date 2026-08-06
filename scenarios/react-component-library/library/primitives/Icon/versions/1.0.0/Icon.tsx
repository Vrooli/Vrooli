/** @vrooliComponentSource primitives.icon */
import type { SVGProps } from "react";
import {
  ICON_REGISTRY,
  type IconName,
} from "../../../../foundations/IconRegistry/versions/1.0.0/IconRegistry";

type SVGElementProps = SVGProps<SVGSVGElement>;

export function Icon({
  name,
  label,
  ...props
}: { name: IconName; label?: string } & SVGElementProps) {
  const icon = ICON_REGISTRY[name];
  return (
    <svg
      aria-hidden={!label}
      aria-label={label}
      role={label ? "img" : undefined}
      viewBox={icon.viewBox}
      fill="none"
      stroke="currentColor"
      strokeLinecap="round"
      strokeLinejoin="round"
      data-icon={name}
      {...props}
    >
      <path d={icon.path} />
    </svg>
  );
}
