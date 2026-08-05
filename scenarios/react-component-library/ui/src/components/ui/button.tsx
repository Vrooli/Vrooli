/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption 68204016-c809-4c4f-803c-99957ebea46e
 * @vrooliComponentAppliedAt 2026-08-04T23:16:42Z
 * @vrooliComponentSourceSha256 52d943f4a19675301da337a0385ff3d11fa6a3c420cfcb663a2647354394244d
 * @vrooliComponentDriftHash c7819b557b325b34f25c426a4be4dcc6ec0aa06f1ddf93d5e5a68ca6de52cd0b
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
  const testId = (props as ButtonHTMLAttributes<HTMLButtonElement> & { "data-testid"?: string })["data-testid"];
  return (
    <ControlBase {...props} data-testid={testId ?? "button-action"}>
      {icon && <span data-testid="button-icon" data-control-slot="icon" role="img" className="shrink-0">{icon}</span>}
      <span data-testid="button-label" data-control-slot="label" className="min-w-0">{children}</span>
    </ControlBase>
  );
}
