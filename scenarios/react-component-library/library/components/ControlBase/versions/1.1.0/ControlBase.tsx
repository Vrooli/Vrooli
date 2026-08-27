/**
 * @libraryId react-component-library:ControlBase
 * @displayName Control Base
 * @description Shared semantic control primitive for consistent sizing, density, focus, hover, and disabled behavior.
 * @version 1.1.0
 * @tags ["control","primitive","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:ControlBase */
import { forwardRef, useEffect, type ButtonHTMLAttributes, type CSSProperties, type ReactNode } from "react";
import { motionTransition } from "@vrooli/react-component-library/VisualRecipes/1.0.0";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";

export type ControlVariant =
  | "primary" | "secondary" | "ghost" | "danger" | "default" | "outline"
  | "destructive" | "info" | "success" | "warning" | "error" | "pipeline";
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

const variantStyles: Record<ControlVariant, CSSProperties> = {
  primary: { background: "var(--color-primary)", borderColor: "var(--color-primary)", color: "var(--color-primary-foreground)" },
  secondary: { background: "var(--color-surface)", borderColor: "var(--color-border)", color: "var(--color-foreground)" },
  ghost: { background: "transparent", borderColor: "transparent", color: "var(--color-foreground)" },
  danger: { background: "var(--color-danger)", borderColor: "var(--color-danger)", color: "var(--color-primary-foreground)" },
  default: { background: "var(--color-primary)", borderColor: "var(--color-primary)", color: "var(--color-primary-foreground)" },
  outline: { background: "var(--color-surface)", borderColor: "var(--color-border)", color: "var(--color-foreground)" },
  destructive: { background: "var(--color-danger)", borderColor: "var(--color-danger)", color: "var(--color-primary-foreground)" },
  info: { background: "var(--color-info)", borderColor: "var(--color-info)", color: "var(--color-primary-foreground)" },
  success: { background: "var(--color-success)", borderColor: "var(--color-success)", color: "var(--color-primary-foreground)" },
  warning: { background: "var(--color-warning)", borderColor: "var(--color-warning)", color: "var(--color-primary-foreground)" },
  error: { background: "var(--color-danger)", borderColor: "var(--color-danger)", color: "var(--color-primary-foreground)" },
  pipeline: { background: "var(--color-primary)", borderColor: "var(--color-primary)", color: "var(--color-primary-foreground)" },
};

const sizeStyles: Record<ControlSize, CSSProperties> = {
  xs: { minBlockSize: "var(--control-size-xs)", paddingInline: "var(--space-2xs)", fontSize: "var(--text-label-size)", lineHeight: "var(--text-label-line)" },
  sm: { minBlockSize: "var(--control-size-sm)", paddingInline: "var(--space-xs)", fontSize: "var(--text-body-sm-size)", lineHeight: "var(--text-body-sm-line)" },
  md: { minBlockSize: "var(--control-size-md)", paddingInline: "var(--space-sm)", fontSize: "var(--text-body-size)", lineHeight: "var(--text-body-line)" },
  lg: { minBlockSize: "var(--control-size-lg)", paddingInline: "var(--space-md)", fontSize: "var(--text-body-size)", lineHeight: "var(--text-body-line)" },
  xl: { minBlockSize: "var(--control-size-xl)", paddingInline: "var(--space-lg)", fontSize: "var(--text-body-size)", lineHeight: "var(--text-body-line)" },
  icon: { minBlockSize: "var(--control-size-icon)", minInlineSize: "var(--control-size-icon)", paddingInline: "var(--space-2xs)", fontSize: "var(--text-body-size)" },
  default: { minBlockSize: "var(--control-size-md)", paddingInline: "var(--space-sm)", fontSize: "var(--text-body-size)", lineHeight: "var(--text-body-line)" },
};

const styleSheet = `
[data-rcl-control] { font-weight: 650; letter-spacing: var(--text-label-tracking, 0); cursor: pointer; user-select: none; white-space: nowrap; -webkit-tap-highlight-color: transparent; transition: ${motionTransition(["transform", "filter", "box-shadow", "background-color", "border-color", "color"])}; }
[data-rcl-control]:hover:not(:disabled) { filter: brightness(1.06) saturate(1.04); transform: translateY(calc(var(--space-3xs) * -0.25)); box-shadow: var(--elev-raised); }
[data-rcl-control]:active:not(:disabled) { filter: brightness(0.98); transform: translateY(0) scale(0.985); transition-duration: var(--dur-instant); }
[data-rcl-control][data-rcl-pending="true"] { cursor: progress; } [data-rcl-control]:disabled { cursor: not-allowed; opacity: max(var(--opacity-disabled), .68); }
`;
const warnedSizes = new Set<string>();

function developmentBuild() {
  const meta = import.meta as ImportMeta & { env: { DEV?: boolean } };
  return meta.env.DEV || (typeof window !== "undefined" && /^(localhost|127\.0\.0\.1)$/.test(window.location.hostname));
}

export const ControlBase = forwardRef<HTMLButtonElement, ControlBaseProps>(function ControlBase(
  { children, className, density = "comfortable", disabled, shape = "square", size = "md", style, type = "button", variant = "primary", ...props }, ref,
) {
  useLibraryStyleSheet("control-base", styleSheet);
  const testId = (props as ControlBaseProps & { "data-testid"?: string })["data-testid"];
  useEffect(() => {
    const value = { xs: 32, sm: 36, md: 40, lg: 44, xl: 48, icon: 40, default: 40 }[size];
    const key = `${(props as { "data-rcl-component"?: string })["data-rcl-component"] ?? "ControlBase"}:${size}`;
    if (developmentBuild() && value < 44 && !warnedSizes.has(key)) {
      warnedSizes.add(key);
      console.warn(`[react-component-library] ${key} resolves to ${value}px, below the 44px tap-target guidance.`);
    }
  }, [props, size]);
  const belowTapTarget = size === "xs" || size === "sm" || size === "md" || size === "icon" || size === "default";
  return <button {...props} ref={ref} type={type} disabled={disabled} data-testid={testId ?? "control-base-root"} data-rcl-control="true" data-control-density={density} data-control-shape={shape} data-control-size={size} data-control-variant={variant} data-control-below-tap-target={belowTapTarget ? "true" : undefined} className={className} style={{ ...variantStyles[variant], ...sizeStyles[size], gap: density === "compact" ? "var(--space-3xs)" : "var(--space-2xs)", borderRadius: shape === "pill" ? "var(--radius-pill)" : "var(--radius-control)", ...style }}>{children}</button>;
});
