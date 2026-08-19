/**
 * @vrooliComponentSource controls.button
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption 40e6ea2c-1740-46d9-9809-bb8402ee5287
 * @vrooliComponentAppliedAt 2026-08-18T01:12:42Z
 * @vrooliComponentSourceSha256 4fa4e66fa4d6c195614e06020fe013660a77043c759177acc011401309539105
 * @vrooliComponentDriftHash bd828f44dca7498c8172e88b548f8a83ccc69ae551c8a536d90b750501315514
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import {
  type ControlDensity,
  type ControlShape,
  type ControlSize,
  type ControlVariant,
} from "./ControlBase";
import { Pressable } from "./Pressable";
export const BUTTON_VARIANTS = ["primary", "secondary", "ghost", "danger"] as const;
export const BUTTON_SIZES = ["xs", "sm", "md", "lg", "xl", "icon"] as const;
export const BUTTON_DENSITIES = ["comfortable", "compact"] as const;
export const BUTTON_SHAPES = ["square", "pill"] as const;

export type ButtonVariant = ControlVariant;
export type ButtonSize = ControlSize;

export interface ButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  density?: ControlDensity;
  shape?: ControlShape;
  icon?: ReactNode;
  pending?: boolean;
  pendingLabel?: ReactNode;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
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
});
