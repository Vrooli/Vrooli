/** @vrooliComponentSource controls.button */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Pressable } from "@vrooli/react-component-library/Pressable/1.0.0";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md" | "icon";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
}

export function Button({
  children,
  size = "md",
  type = "button",
  variant = "primary",
  ...props
}: ButtonProps) {
  return (
    <Pressable {...props} type={type} tone={variant} size={size}>
      {children}
    </Pressable>
  );
}
