/**
 * @libraryId react-component-library:Button
 * @version 2.0.0
 * @status released
 * @deps {"react":"^18","clsx":"^2.1.1","tailwind-merge":"^2.3.0"}
 * @category controls
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { ControlBase, type ControlDensity, type ControlShape, type ControlSize, type ControlVariant } from "../../../ControlBase/versions/1.0.0/ControlBase";

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
