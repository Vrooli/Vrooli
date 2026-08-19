/**
 * @vrooliComponentSource react-component-library:ControlBase
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 0ee254b3-a523-4ebf-b17f-8a763bcef3b5
 * @vrooliComponentAppliedAt 2026-08-05T04:36:00Z
 * @vrooliComponentSourceSha256 8b785f0d4bdc836ef703c7d2dfa6fc9acda5b4cd83eaa114876b570b648ca6ec
 * @vrooliComponentDriftHash e856525a26082615e813872dcaad962e0f55b88b2688c8b66ce30e68c334f5cc
 * @vrooliComponentTokenTranslation bg-app-danger->bg-app-danger,bg-app-info->bg-app-info,bg-app-primary->bg-app-primary,bg-app-success->bg-app-success,bg-app-surface->bg-app-surface,bg-app-surface-muted->bg-app-surface-muted,bg-app-warning->bg-app-warning,border-app-border->border-app-border,ring-app-primary/50->ring-app-primary/50,text-app-foreground->text-app-foreground,text-app-primary-foreground->text-app-primary-foreground
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
  primary: "bg-app-primary text-app-primary-foreground hover:brightness-95",
  secondary: "border border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
  ghost: "text-app-foreground hover:bg-app-surface-muted",
  danger: "bg-app-danger text-app-primary-foreground hover:brightness-95",
  default: "bg-app-primary text-app-primary-foreground hover:brightness-95",
  outline: "border border-app-border bg-app-surface text-app-foreground hover:bg-app-surface-muted",
  destructive: "bg-app-danger text-app-primary-foreground hover:brightness-95",
  info: "bg-app-info text-app-primary-foreground hover:brightness-95",
  success: "bg-app-success text-app-primary-foreground hover:brightness-95",
  warning: "bg-app-warning text-app-primary-foreground hover:brightness-95",
  error: "bg-app-danger text-app-primary-foreground hover:brightness-95",
  pipeline: "bg-app-primary text-app-primary-foreground hover:brightness-95",
};

const sizeClasses: Record<ControlSize, string> = {
  xs: "min-h-12 min-w-12 rounded-sm px-2 text-xs [&>svg]:size-3",
  sm: "min-h-12 min-w-12 rounded-md px-3 text-sm [&>svg]:size-3.5",
  md: "min-h-12 min-w-12 rounded-control px-3.5 text-sm [&>svg]:size-4",
  lg: "min-h-12 min-w-12 rounded-lg px-4 text-base [&>svg]:size-5",
  xl: "min-h-12 min-w-12 rounded-xl px-5 text-base [&>svg]:size-5",
  icon: "min-h-12 min-w-12 rounded-control p-0 text-sm [&>svg]:size-4",
  default: "min-h-12 min-w-12 rounded-control px-3.5 text-sm [&>svg]:size-4",
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
        "inline-flex items-center justify-center font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50 disabled:pointer-events-none disabled:opacity-60",
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
