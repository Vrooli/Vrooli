/**
 * @vrooliComponentSource react-component-library:ControlBase
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 27dad393-1dff-47a6-a5dc-6c0e4aa2523e
 * @vrooliComponentAppliedAt 2026-08-05T04:35:59Z
 * @vrooliComponentSourceSha256 8b785f0d4bdc836ef703c7d2dfa6fc9acda5b4cd83eaa114876b570b648ca6ec
 * @vrooliComponentDriftHash ca0052e29abbff3986cd3e1f2fc43c98912a6885ec7ddf6fd3112102342f5357
 * @vrooliComponentTokenTranslation bg-app-danger->bg-wc-error-surface,bg-app-info->bg-wc-accent,bg-app-primary->bg-wc-accent-active,bg-app-success->bg-wc-text-secondary,bg-app-surface->bg-wc-surface-raised,bg-app-surface-muted->bg-wc-surface-input,bg-app-warning->bg-wc-accent,border-app-border->border-wc-default,ring-app-primary/50->ring-wc-accent/50,text-app-foreground->text-wc-text-primary,text-app-primary-foreground->text-wc-accent-fg
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from "react";
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export type ControlVariant = "primary" | "secondary" | "ghost" | "danger" | "default" | "outline" | "destructive" | "info" | "success" | "warning" | "error" | "pipeline";
export type ControlSize = "xs" | "sm" | "md" | "lg" | "xl" | "icon" | "default";
export type ControlDensity = "comfortable" | "compact";
export type ControlShape = "square" | "pill";

export interface ControlBaseProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
  children: ReactNode;
  variant?: ControlVariant;
  size?: ControlSize;
  density?: ControlDensity;
  shape?: ControlShape;
}

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

const variantClasses: Record<ControlVariant, string> = {
  primary: "bg-wc-accent-active text-wc-accent-fg hover:brightness-95",
  secondary: "border border-wc-default bg-wc-surface-raised text-wc-text-primary hover:bg-wc-surface-input",
  ghost: "text-wc-text-primary hover:bg-wc-surface-input",
  danger: "bg-wc-error-surface text-wc-accent-fg hover:brightness-95",
  default: "bg-wc-accent-active text-wc-accent-fg hover:brightness-95",
  outline: "border border-wc-default bg-wc-surface-raised text-wc-text-primary hover:bg-wc-surface-input",
  destructive: "bg-wc-error-surface text-wc-accent-fg hover:brightness-95",
  info: "bg-wc-accent text-wc-accent-fg hover:brightness-95",
  success: "bg-wc-text-secondary text-wc-accent-fg hover:brightness-95",
  warning: "bg-wc-accent text-wc-accent-fg hover:brightness-95",
  error: "bg-wc-error-surface text-wc-accent-fg hover:brightness-95",
  pipeline: "bg-wc-accent-active text-wc-accent-fg hover:brightness-95",
};

const sizeClasses: Record<ControlSize, string> = {
  xs: "min-h-11 min-w-11 rounded-sm px-2 text-xs [&>svg]:size-3",
  sm: "min-h-11 min-w-11 rounded-md px-3 text-sm [&>svg]:size-3.5",
  md: "min-h-11 min-w-11 rounded-control px-3.5 text-sm [&>svg]:size-4",
  lg: "min-h-11 min-w-11 rounded-lg px-4 text-base [&>svg]:size-5",
  xl: "min-h-12 min-w-12 rounded-xl px-5 text-base [&>svg]:size-5",
  icon: "min-h-11 min-w-11 rounded-control p-0 text-sm [&>svg]:size-4",
  default: "min-h-11 min-w-11 rounded-control px-3.5 text-sm [&>svg]:size-4",
};

const densityClasses: Record<ControlDensity, string> = {
  comfortable: "gap-2",
  compact: "gap-1.5",
};

const shapeClasses: Record<ControlShape, string> = {
  square: "rounded-control",
  pill: "rounded-full",
};

export const ControlBase = forwardRef<HTMLButtonElement, ControlBaseProps>(function ControlBase(
  {
    children,
    className,
    density = "comfortable",
    disabled,
    shape = "square",
    size = "md",
    type = "button",
    variant = "primary",
    ...props
  },
  ref,
) {
  const testId = (props as ControlBaseProps & { "data-testid"?: string })["data-testid"];
  return (
    <button
      {...props}
      ref={ref}
      type={type}
      disabled={disabled}
      data-testid={testId ?? "control-base-root"}
      data-control-density={density}
      data-control-shape={shape}
      data-control-size={size}
      className={cn(
        "inline-flex items-center justify-center font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent/50 disabled:pointer-events-none disabled:opacity-60",
        variantClasses[variant],
        sizeClasses[size],
        densityClasses[density],
        shapeClasses[shape],
        className,
      )}
    >
      {children}
    </button>
  );
});
