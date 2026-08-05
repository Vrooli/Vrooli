/**
 * @vrooliComponentSource react-component-library:Button
 * @vrooliComponentVersion 1.3.0
 * @vrooliComponentAdoption 27dad393-1dff-47a6-a5dc-6c0e4aa2523e
 * @vrooliComponentAppliedAt 2026-08-04T07:57:05Z
 * @vrooliComponentSourceSha256 18ecca1808ebdfee3fe83585ad08d5a27b5d2a91d0218374222e9068216f0fd8
 * @vrooliComponentDriftHash 84ef2bd52d1edfdbab2a64b65910f2434bc3f89664b197016f7fed34f952544e
 * @vrooliComponentTokenTranslation bg-app-danger->bg-wc-error-surface,bg-app-primary->bg-wc-accent-active,bg-app-surface->bg-wc-surface-raised,bg-app-surface-muted->bg-wc-surface-input,border-app-border->border-wc-default,ring-app-primary/50->ring-wc-accent/50,text-app-foreground->text-wc-text-primary,text-app-primary-foreground->text-wc-accent-fg
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ButtonHTMLAttributes, ReactNode } from "react";

export type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";
export type ButtonSize = "xs" | "sm" | "md" | "lg" | "xl" | "icon";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  children: ReactNode;
  variant?: ButtonVariant;
  size?: ButtonSize;
  icon?: ReactNode;
}

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const variantClasses: Record<ButtonVariant, string> = {
  primary: "bg-wc-accent-active text-wc-accent-fg hover:brightness-95",
  secondary: "border border-wc-default bg-wc-surface-raised text-wc-text-primary hover:bg-wc-surface-input",
  ghost: "text-wc-text-primary hover:bg-wc-surface-input",
  danger: "bg-wc-error-surface text-wc-accent-fg hover:brightness-95",
};

const sizeClasses: Record<ButtonSize, string> = {
  xs: "min-h-8 min-w-8 rounded-sm px-2 text-xs",
  sm: "min-h-9 min-w-9 rounded-md px-3 text-sm",
  md: "min-h-10 min-w-10 rounded-control px-3.5 text-sm",
  lg: "min-h-11 min-w-11 rounded-lg px-4 text-base",
  xl: "min-h-12 min-w-12 rounded-xl px-5 text-base",
  icon: "min-h-10 min-w-10 rounded-control p-0",
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
      data-control-size={size}
      className={cn(
        "inline-flex items-center justify-center gap-2 rounded-control font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent/50 disabled:pointer-events-none disabled:opacity-60",
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
