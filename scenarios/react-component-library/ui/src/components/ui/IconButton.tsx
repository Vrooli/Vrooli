/**
 * @vrooliComponentSource react-component-library:IconButton
 * @vrooliComponentVersion 2.0.0
 * @vrooliComponentAdoption 8a3af1ec-0e94-4b6a-ae8d-2bb53f995f27
 * @vrooliComponentAppliedAt 2026-08-04T15:20:42Z
 * @vrooliComponentSourceSha256 2a284f3267c55d4a771747f9d6d164d4c868deae20a2b70027cbc89f563601ac
 * @vrooliComponentDriftHash d298f93ba6d37b7a647636fb527f6c9ff2746c962fa8e80a97b1bbebcedfff15
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { ControlBase, type ControlDensity, type ControlSize, type ControlVariant } from "./ControlBase";

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
