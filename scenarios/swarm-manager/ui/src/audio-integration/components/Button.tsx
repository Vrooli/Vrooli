/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 1.3.0
 * @vrooliComponentAdoption 1facf313-4a56-4ab5-8731-0d468b0a929b
 * @vrooliComponentAppliedAt 2026-08-04T07:57:06Z
 * @vrooliComponentSourceSha256 18ecca1808ebdfee3fe83585ad08d5a27b5d2a91d0218374222e9068216f0fd8
 * @vrooliComponentDriftHash bafe49d18f2631c87a820a522234a3a1774f7e17ae84b7f248fee8011a939e47
 * @vrooliComponentTokenTranslation bg-app-danger->bg-slate-600,bg-app-primary->bg-slate-300,bg-app-surface->bg-slate-900,bg-app-surface-muted->bg-slate-800,border-app-border->border-slate-700,ring-app-primary/50->ring-slate-300/50,text-app-foreground->text-slate-50,text-app-primary-foreground->text-slate-950
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ButtonHTMLAttributes, ReactNode } from "react";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
export type ButtonSize = "xs" | "sm" | "md" | "icon";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: ReactNode;
}

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const variantClasses: Record<ButtonVariant, string> = {
  primary: "bg-slate-300 text-slate-950 hover:brightness-95",
  secondary: "border border-slate-700 bg-slate-900 text-slate-50 hover:bg-slate-800",
  ghost: "text-slate-50 hover:bg-slate-800",
  danger: "bg-slate-600 text-slate-950 hover:brightness-95",
};

const sizeClasses: Record<ButtonSize, string> = {
  xs: "min-h-8 px-2 text-xs",
  sm: "min-h-9 px-3 text-sm",
  md: "min-h-11 px-4 text-sm",
  icon: "min-h-11 min-w-11 p-0",
};

export function Button({
  children,
  className,
  icon,
  size = "md",
  type = "button",
  variant = "primary",
  ...props
}: ButtonProps) {
  return (
    <button
      type={type}
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-control font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-300/50 disabled:pointer-events-none disabled:opacity-60",
        variantClasses[variant],
        sizeClasses[size],
        className,
      )}
      {...props}
    >
      {icon}
      {children}
    </button>
  );
}
