/**
 * @vrooliComponentSource react-component-library:IconButton
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption 59f07a82-a0b1-4dd2-987d-4d497a059ba5
 * @vrooliComponentAppliedAt 2026-08-11T00:24:46Z
 * @vrooliComponentSourceSha256 f73de8840bf838fe9eca29ab657587bd667443b2658ba4296632916b5d72d5a2
 * @vrooliComponentDriftHash 03f7660b3628cb98830a46305fb34366af8d98c31c1d8e17336ceeb747e513cb
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { type ControlDensity, type ControlSize, type ControlVariant } from "./ControlBase";
import { Pressable } from "./Pressable";

export interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
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
      title={disableTooltip ? undefined : (title ?? ariaLabel)}
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
