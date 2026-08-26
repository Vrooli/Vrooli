/**
 * @libraryId react-component-library:Button
 * @displayName Button
 * @description Token-bound accessible action button with variant and size styling.
 * @version 2.2.0
 * @tags ["form","interactive"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.button */
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import {
  type ControlDensity,
  type ControlShape,
  type ControlSize,
  type ControlVariant,
} from "../../../ControlBase/versions/1.0.0/ControlBase";
import { Pressable } from "../../../Pressable/versions/1.0.0/Pressable";
import {
  Icon,
  type IconName,
  type IconSize,
} from "../../../../primitives/Icon/versions/1.1.0/Icon";
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
  /** Preferred path for icons from the governed library registry. */
  iconName?: IconName;
  /** Compatibility path for custom icon content not present in the registry. */
  icon?: ReactNode;
  pending?: boolean;
  pendingLabel?: ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  function Button(
    {
      children,
      icon,
      iconName,
      pending,
      pendingLabel,
      size = "md",
      variant = "primary",
      ...props
    },
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
        size={size}
        pending={pending}
        pendingLabel={pendingLabel}
      >
        {iconName ? (
          <Icon
            name={iconName}
            size={controlIconSize(size)}
            data-testid="button-icon"
            data-control-slot="icon"
            style={{ color: "currentColor", flex: "0 0 auto" }}
          />
        ) : icon ? (
          <span
            data-testid="button-icon"
            data-control-slot="icon"
            aria-hidden="true"
            style={{ flex: "0 0 auto" }}
          >
            {icon}
          </span>
        ) : null}
        <span
          data-testid="button-label"
          data-control-slot="label"
          style={{
            minWidth: 0,
            maxWidth: "100%",
            overflow: "hidden",
            textOverflow: "ellipsis",
            whiteSpace: "nowrap",
          }}
        >
          {children}
        </span>
      </Pressable>
    );
  },
);

function controlIconSize(size: ControlSize): IconSize {
  if (size === "xs" || size === "sm") return "sm";
  if (size === "lg" || size === "xl") return "lg";
  return "md";
}
