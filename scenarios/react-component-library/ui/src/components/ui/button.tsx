/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption 68204016-c809-4c4f-803c-99957ebea46e
 * @vrooliComponentAppliedAt 2026-08-04T15:20:42Z
 * @vrooliComponentSourceSha256 15c63d90a49a5c51ce9e0777509245711592c5cd1931894df3bca346cd8cb05c
 * @vrooliComponentDriftHash 5a216bb568bab0a5828b17f6ea152527c279337c679bb5917809b9d9d3ae30ee
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { ControlBase, type ControlDensity, type ControlShape, type ControlSize, type ControlVariant } from "./ControlBase";

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
  return (
    <ControlBase {...props}>
      {icon && <span data-control-slot="icon" className="shrink-0">{icon}</span>}
      <span data-control-slot="label" className="min-w-0">{children}</span>
    </ControlBase>
  );
}
