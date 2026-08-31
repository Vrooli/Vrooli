/**
 * @libraryId react-component-library:IconButton
 * @displayName IconButton
 * @description Icon-only action with a real hover surface, circular by default, animating whenever its icon changes.
 * @version 2.0.3
 * @tags ["button","icon","accessibility","motion"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.icon-button */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import {
  type ControlDensity,
  type ControlSize,
  type ControlVariant,
} from "@vrooli/react-component-library/ControlBase/1";
import { Pressable } from "@vrooli/react-component-library/Pressable/1";

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
