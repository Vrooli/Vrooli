/** @vrooliComponentSource controls.icon-button */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { ButtonHTMLAttributes, ReactNode } from "react";
import {
  type ControlDensity,
  type ControlSize,
  type ControlVariant,
} from "@vrooli/react-component-library/ControlBase/1.0.0";
import { Pressable } from "@vrooli/react-component-library/Pressable/1.0.0";

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

export const IconButton = withClassName(function IconButton({
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
    <Pressable data-testid="controls.icon-button"
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
});
