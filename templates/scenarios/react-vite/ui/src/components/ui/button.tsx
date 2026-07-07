/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption template:react-vite:button
 * @vrooliComponentAppliedAt 2026-07-07T00:00:00Z
 * @vrooliComponentSourceSha256 b6bbd7fb599bb343de5ea50d9ddbddebba1c6d905f795ebd1531ade11b0dda2c
 * @vrooliComponentDriftHash b6bbd7fb599bb343de5ea50d9ddbddebba1c6d905f795ebd1531ade11b0dda2c
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
type ButtonSize = "sm" | "md" | "icon";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
}

const joinClasses = (...classes: Array<string | undefined | false>) =>
  classes.filter(Boolean).join(" ");

const variantClasses: Record<ButtonVariant, string> = {
  primary: "bg-app-primary text-app-primary-foreground hover:brightness-95",
  secondary: "border border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
  ghost: "text-app-foreground hover:bg-app-surface-muted",
  danger: "bg-app-danger text-app-primary-foreground hover:brightness-95",
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: "min-h-9 px-3 text-sm",
  md: "min-h-11 px-4 text-sm",
  icon: "min-h-11 min-w-11 p-0",
};

export function Button({
  children,
  className,
  size = "md",
  type = "button",
  variant = "primary",
  ...props
}: ButtonProps) {
  return (
    <button
      type={type}
      className={joinClasses(
        "inline-flex items-center justify-center gap-2 rounded-control font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:pointer-events-none disabled:opacity-60",
        variantClasses[variant],
        sizeClasses[size],
        className,
      )}
      {...props}
    >
      {children}
    </button>
  );
}
