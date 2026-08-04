/**
 * @libraryId react-component-library:IconButton
 * @displayName IconButton
 * @description Accessible icon-only action control backed by ControlBase.
 * @version 2.0.0
 * @status released
 * @deps {"react":"^18","clsx":"^2.1.1","tailwind-merge":"^2.3.0"}
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { ControlBase, type ControlDensity, type ControlSize, type ControlVariant } from "../../../ControlBase/versions/1.0.0/ControlBase";

export interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
  "aria-label": string;
  children: ReactNode;
  size?: ControlSize;
  density?: ControlDensity;
  variant?: ControlVariant;
}

export function IconButton({ "aria-label": ariaLabel, children, density = "comfortable", size = "icon", title, type = "button", variant = "ghost", ...props }: IconButtonProps) {
  return (
    <ControlBase
      {...props}
      aria-label={ariaLabel}
      type={type}
      title={title ?? ariaLabel}
      variant={variant}
      size={size}
      density={density}
      shape="square"
    >
      {children}
    </ControlBase>
  );
}
