/**
 * @vrooliComponentSource react-component-library:ControlBase
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 1facf313-4a56-4ab5-8731-0d468b0a929b
 * @vrooliComponentAppliedAt 2026-08-05T04:35:59Z
 * @vrooliComponentSourceSha256 8b785f0d4bdc836ef703c7d2dfa6fc9acda5b4cd83eaa114876b570b648ca6ec
 * @vrooliComponentDriftHash d4e19a8fe5f75837a07a033ccfb288554748ab9cdeb570e1ff768ada7c37c274
 * @vrooliComponentTokenTranslation bg-app-danger->bg-slate-600,bg-app-info->bg-slate-400,bg-app-primary->bg-slate-300,bg-app-success->bg-slate-100,bg-app-surface->bg-slate-900,bg-app-surface-muted->bg-slate-800,bg-app-warning->bg-slate-500,border-app-border->border-slate-700,ring-app-primary/50->ring-slate-300/50,text-app-foreground->text-slate-50,text-app-primary-foreground->text-slate-950
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
  primary: "bg-slate-300 text-slate-950 hover:brightness-95",
  secondary: "border border-slate-700 bg-slate-900 text-slate-50 hover:bg-slate-800",
  ghost: "text-slate-50 hover:bg-slate-800",
  danger: "bg-slate-600 text-slate-950 hover:brightness-95",
  default: "bg-slate-300 text-slate-950 hover:brightness-95",
  outline: "border border-slate-700 bg-slate-900 text-slate-50 hover:bg-slate-800",
  destructive: "bg-slate-600 text-slate-950 hover:brightness-95",
  info: "bg-slate-400 text-slate-950 hover:brightness-95",
  success: "bg-slate-100 text-slate-950 hover:brightness-95",
  warning: "bg-slate-500 text-slate-950 hover:brightness-95",
  error: "bg-slate-600 text-slate-950 hover:brightness-95",
  pipeline: "bg-slate-300 text-slate-950 hover:brightness-95",
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
        "inline-flex items-center justify-center font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-300/50 disabled:pointer-events-none disabled:opacity-60",
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
