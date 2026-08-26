/** @vrooliComponentSource controls.button */
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import {
  type ControlDensity,
  type ControlShape,
  type ControlSize,
  type ControlVariant,
} from "@vrooli/react-component-library/ControlBase/1.0.0";
import { Pressable } from "@vrooli/react-component-library/Pressable/1.0.0";
export const BUTTON_VARIANTS = [
  "primary",
  "secondary",
  "ghost",
  "danger",
] as const;
export const BUTTON_SIZES = ["xs", "sm", "md", "lg", "xl", "icon"] as const;
export const BUTTON_DENSITIES = ["comfortable", "compact"] as const;
export const BUTTON_SHAPES = ["square", "pill"] as const;

export type ButtonVariant = ControlVariant;
export type ButtonSize = ControlSize;

export interface ButtonProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  density?: ControlDensity;
  shape?: ControlShape;
  icon?: ReactNode;
  pending?: boolean;
  pendingLabel?: ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button(
    { children, icon, pending, pendingLabel, variant = "primary", ...props },
    ref,
  ) {
    const testId = (
      props as ButtonHTMLAttributes<HTMLButtonElement> & {
        "data-testid"?: string;
      }
    )["data-testid"];
    return (
      <Pressable
        {...props}
        ref={ref}
        data-testid={testId ?? "button-action"}
        tone={variant}
        pending={pending}
        pendingLabel={pendingLabel}
      >
        {icon && (
          <span
            data-testid="button-icon"
            data-control-slot="icon"
            role="img"
            aria-label=""
            style={{ flex: "0 0 auto" }}
          >
            {icon}
          </span>
        )}
        <span
          data-testid="button-label"
          data-control-slot="label"
          style={{ minWidth: 0, overflow: "hidden", textOverflow: "ellipsis" }}
        >
          {children}
        </span>
      </Pressable>
    );
  },
);
