/** @vrooliComponentSource controls.icon-button */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import {
  type ControlDensity,
  type ControlSize,
  type ControlVariant,
} from "../../../ControlBase/versions/1.0.0/ControlBase";
import { Pressable } from "../../../Pressable/versions/1.0.0/Pressable";

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
