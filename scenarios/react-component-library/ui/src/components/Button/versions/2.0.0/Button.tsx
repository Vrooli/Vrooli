/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption 7b949fe5-bb55-4df0-93cd-506549c841d7
 * @vrooliComponentAppliedAt 2026-08-11T00:35:31Z
 * @vrooliComponentSourceSha256 c13e1e9458971dd41ea1bc8643352b16fa9925d3140055ab259b5cd525a0898f
 * @vrooliComponentDriftHash 2bc06d26d343747eaf6dfa63cb879ca57640f7b29cfc87e25fa2b3adbee92e88
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
