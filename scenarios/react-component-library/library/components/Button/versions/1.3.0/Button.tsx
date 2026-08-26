/** @vrooliComponentSource controls.button */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Pressable } from "../../../Pressable/versions/1.0.0/Pressable";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
export type ButtonSize = "xs" | "sm" | "md" | "icon";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: ReactNode;
}

export function Button({
  children,
  icon,
  size = "md",
  type = "button",
  variant = "primary",
  ...props
}: ButtonProps) {
  return (
    <Pressable {...props} type={type} tone={variant} size={size}>
      {icon && <span style={{ flex: "0 0 auto" }}>{icon}</span>}
      <span
        style={{ minWidth: 0, overflow: "hidden", textOverflow: "ellipsis" }}
      >
        {children}
      </span>
    </Pressable>
  );
}
