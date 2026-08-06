/** @vrooliComponentSource controls.icon-button */
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Pressable } from "../../../Pressable/versions/1.0.0/Pressable";

export interface IconButtonProps
  extends ButtonHTMLAttributes<HTMLButtonElement> {
  "aria-label": string;
  children: ReactNode;
}

export function IconButton({
  children,
  title,
  type = "button",
  ...props
}: IconButtonProps) {
  return (
    <Pressable
      {...props}
      type={type}
      title={title ?? props["aria-label"]}
      tone="ghost"
      size="icon"
    >
      {children}
    </Pressable>
  );
}
