/**
 * @vrooliComponentSource react-component-library:IconButton
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption fca0af9a-3a97-46e6-b43a-b8c6504d9361
 * @vrooliComponentAppliedAt 2026-08-09T14:56:08Z
 * @vrooliComponentSourceSha256 ccccf184e76a94bddee516dc40c34879f87b0b33750aa2ebe0855bb23845bd6b
 * @vrooliComponentDriftHash 04628579373a27c186c2e55d0e4de04c44f8f62aa39e5cc31aae489c491955f8
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import {
  type ControlDensity,
  type ControlSize,
  type ControlVariant,
} from "./ControlBase";
import { Pressable } from "./Pressable";

export interface IconButtonProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
  "aria-label": string;
  children: ReactNode;
  size?: ControlSize;
  density?: ControlDensity;
  variant?: ControlVariant;
  pending?: boolean;
  disableTooltip?: boolean;
}

export function IconButton({
  "aria-label": ariaLabel,
  children,
  density = "comfortable",
  size = "icon",
  title,
  pending,
  disableTooltip = false,
  type = "button",
  variant = "ghost",
  ...props
}: IconButtonProps) {
  return (
    <Pressable
      {...props}
      aria-label={ariaLabel}
      type={type}
      title={disableTooltip ? undefined : title ?? ariaLabel}
      size={size}
      density={density}
      shape="square"
      pending={pending}
      tone={variant}
    >
      {children}
    </Pressable>
  );
}
