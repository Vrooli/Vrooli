/**
 * @libraryId react-component-library:ControlBase
 * @displayName Control Base
 * @version 1.0.0
 * @status released
 * @deps {"react":"^18","clsx":"^2.1.1","tailwind-merge":"^2.3.0"}
 * @category controls
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
  xs: "min-h-11 min-w-11 rounded-control px-space-xs text-xs [&>svg]:size-3",
  sm: "min-h-11 min-w-11 rounded-md px-3 text-sm [&>svg]:size-3.5",
  md: "min-h-11 min-w-11 rounded-control px-space-sm text-sm [&>svg]:size-4",
  lg: "min-h-11 min-w-11 rounded-control px-space-md text-base [&>svg]:size-5",
  xl: "min-h-12 min-w-12 rounded-control px-space-lg text-base [&>svg]:size-5",
  icon: "min-h-11 min-w-11 rounded-control p-0 text-sm [&>svg]:size-4",
  default: "min-h-11 min-w-11 rounded-control px-space-sm text-sm [&>svg]:size-4",
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
