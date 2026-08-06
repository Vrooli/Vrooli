/** @vrooliComponentSource controls.button */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Pressable } from "../../../Pressable/versions/1.0.0/Pressable";

type ButtonVariant = "primary" | "secondary";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
}

export function Button({
  children,
  variant = "primary",
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <Pressable {...props} type={type} tone={variant} size="md">
      {children}
    </Pressable>
  );
}
