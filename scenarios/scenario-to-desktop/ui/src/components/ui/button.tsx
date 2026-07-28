/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 1.2.0
 * @vrooliComponentAdoption 64208a21-ec2c-453b-ad93-e68ba0dd3388
 * @vrooliComponentAppliedAt 2026-07-27T22:32:21Z
 * @vrooliComponentSourceSha256 85fefc69951c8ed5b871b3f371e4483de6677cb1f2098dd1d341ad372340275e
 * @vrooliComponentDriftHash 85fefc69951c8ed5b871b3f371e4483de6677cb1f2098dd1d341ad372340275e
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ButtonHTMLAttributes, ReactNode } from "react";

// Keep the catalogue vocabulary while accepting the scenario's established
// public API. This allows a governed primitive to be introduced without a
// behaviour-changing migration across every desktop workflow.
type ButtonVariant =
  | "primary"
  | "secondary"
  | "ghost"
  | "danger"
  | "default"
  | "outline"
  | "destructive";
type ButtonSize = "sm" | "md" | "icon" | "default";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
}

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const variantClasses: Record<ButtonVariant, string> = {
  primary: "bg-app-primary text-app-primary-foreground hover:brightness-95",
  secondary:
    "border border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
  ghost: "text-app-foreground hover:bg-app-surface-muted",
  danger: "bg-app-danger text-app-primary-foreground hover:brightness-95",
  default: "bg-app-primary text-app-primary-foreground hover:brightness-95",
  outline:
    "border border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
  destructive: "bg-app-danger text-app-primary-foreground hover:brightness-95",
};

const sizeClasses: Record<ButtonSize, string> = {
  sm: "min-h-9 px-3 text-sm",
  md: "min-h-11 px-4 text-sm",
  icon: "min-h-11 min-w-11 p-0",
  default: "min-h-11 px-4 text-sm",
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
      className={cn(
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
