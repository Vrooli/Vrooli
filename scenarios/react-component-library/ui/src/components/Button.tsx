/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption aad5c156-33c6-44a8-9e4a-a3bdeeae5a60
 * @vrooliComponentAppliedAt 2026-08-06T03:44:59Z
 * @vrooliComponentSourceSha256 c65b1798221a3822e624904b8e5b92fc5393ac9fe1352de014982ea8c4b68b20
 * @vrooliComponentDriftHash d72146ff3e16ebc0028671ae14fa4d78ea666421e8108dd9719206954c8bd256
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { ControlBase, type ControlDensity, type ControlShape, type ControlSize, type ControlVariant } from "./ControlBase";
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
}

export function Button({ children, icon, ...props }: ButtonProps) {
  const testId = (props as ButtonHTMLAttributes<HTMLButtonElement> & { "data-testid"?: string })["data-testid"];
  return (
    <ControlBase {...props} data-testid={testId ?? "button-action"}>
      {icon && <span data-testid="button-icon" data-control-slot="icon" role="img" className="shrink-0">{icon}</span>}
      <span data-testid="button-label" data-control-slot="label" className="min-w-0">{children}</span>
    </ControlBase>
  );
}
