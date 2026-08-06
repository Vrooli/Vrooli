/**
 * @vrooliComponentSource react-component-library:ControlBase
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption aad5c156-33c6-44a8-9e4a-a3bdeeae5a60
 * @vrooliComponentAppliedAt 2026-08-06T14:06:21Z
 * @vrooliComponentSourceSha256 4ea9bf9a593d227d743f290a1cc68b02800ba5f7d336a70dc9a767b7dd8747f0
 * @vrooliComponentDriftHash 0f8abba44e39114981426cf204ed456afe1aab3454fdc275864dbaf080874d88
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { forwardRef, type ButtonHTMLAttributes, type CSSProperties, type ReactNode } from "react";

export type ControlVariant =
  | "primary"
  | "secondary"
  | "ghost"
  | "danger"
  | "default"
  | "outline"
  | "destructive"
  | "info"
  | "success"
  | "warning"
  | "error"
  | "pipeline";
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
  primary: {
    background: "var(--color-primary)",
    borderColor: "var(--color-primary)",
    color: "var(--color-primary-foreground)",
  },
  secondary: {
    background: "var(--color-surface)",
    borderColor: "var(--color-border)",
    color: "var(--color-foreground)",
  },
  ghost: {
    background: "transparent",
    borderColor: "transparent",
    color: "var(--color-foreground)",
  },
  danger: {
    background: "var(--color-danger)",
    borderColor: "var(--color-danger)",
    color: "var(--color-primary-foreground)",
  },
  default: {
    background: "var(--color-primary)",
    borderColor: "var(--color-primary)",
    color: "var(--color-primary-foreground)",
  },
  outline: {
    background: "var(--color-surface)",
    borderColor: "var(--color-border)",
    color: "var(--color-foreground)",
  },
  destructive: {
    background: "var(--color-danger)",
    borderColor: "var(--color-danger)",
    color: "var(--color-primary-foreground)",
  },
  info: {
    background: "var(--color-info)",
    borderColor: "var(--color-info)",
    color: "var(--color-primary-foreground)",
  },
  success: {
    background: "var(--color-success)",
    borderColor: "var(--color-success)",
    color: "var(--color-primary-foreground)",
  },
  warning: {
    background: "var(--color-warning)",
    borderColor: "var(--color-warning)",
    color: "var(--color-primary-foreground)",
  },
  error: {
    background: "var(--color-danger)",
    borderColor: "var(--color-danger)",
    color: "var(--color-primary-foreground)",
  },
  pipeline: {
    background: "var(--color-primary)",
    borderColor: "var(--color-primary)",
    color: "var(--color-primary-foreground)",
  },
};

const sizeStyles: Record<ControlSize, CSSProperties> = {
  xs: {
    paddingInline: "var(--space-xs)",
    fontSize: "var(--text-label-size)",
    lineHeight: "var(--text-label-line)",
  },
  sm: {
    paddingInline: "var(--space-sm)",
    fontSize: "var(--text-body-sm-size)",
    lineHeight: "var(--text-body-sm-line)",
  },
  md: {
    paddingInline: "var(--space-sm)",
    fontSize: "var(--text-body-size)",
    lineHeight: "var(--text-body-line)",
  },
  lg: {
    paddingInline: "var(--space-md)",
    fontSize: "var(--text-body-size)",
    lineHeight: "var(--text-body-line)",
  },
  xl: {
    paddingInline: "var(--space-lg)",
    fontSize: "var(--text-body-size)",
    lineHeight: "var(--text-body-line)",
  },
  icon: {
    paddingInline: "var(--space-xs)",
    minWidth: "var(--tap-target-min)",
    fontSize: "var(--text-body-size)",
  },
  default: {
    paddingInline: "var(--space-sm)",
    fontSize: "var(--text-body-size)",
    lineHeight: "var(--text-body-line)",
  },
};

const styleSheet = `
[data-rcl-control] {
  appearance: none;
  min-height: var(--tap-target-min);
  min-width: var(--tap-target-min);
  border-width: var(--border-hairline);
  border-style: solid;
  border-radius: var(--radius-control);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-family: inherit;
  font-weight: 650;
  cursor: pointer;
  user-select: none;
  transition: transform var(--dur-quick) var(--ease-standard), filter var(--dur-quick) var(--ease-standard), box-shadow var(--dur-quick) var(--ease-standard);
}
[data-rcl-control]:hover:not(:disabled) {
  filter: brightness(1.06) saturate(1.04);
  transform: translateY(calc(var(--space-3xs) * -0.25));
  box-shadow: var(--elev-raised);
}
[data-rcl-control]:active:not(:disabled) {
  filter: brightness(0.98);
  transform: translateY(0) scale(0.985);
  transition-duration: var(--dur-instant);
}
[data-rcl-control]:focus-visible {
  outline: var(--border-strong) solid var(--color-focus);
  outline-offset: var(--space-3xs);
}
[data-rcl-control]:disabled {
  cursor: not-allowed;
  opacity: var(--opacity-disabled);
}
@media (prefers-reduced-motion: reduce) {
  [data-rcl-control] { transition: none; }
  [data-rcl-control]:hover:not(:disabled), [data-rcl-control]:active:not(:disabled) { transform: none; }
}
`;

function ControlStyles() {
  return <style data-rcl-control-styles dangerouslySetInnerHTML={{ __html: styleSheet }} />;
}

export const ControlBase = forwardRef<HTMLButtonElement, ControlBaseProps>(function ControlBase(
  {
    children,
    className,
    density = "comfortable",
    disabled,
    shape = "square",
    size = "md",
    style,
    type = "button",
    variant = "primary",
    ...props
  },
  ref,
) {
  const testId = (props as ControlBaseProps & { "data-testid"?: string })["data-testid"];
  return (
    <>
      <ControlStyles />
      <button
        {...props}
        ref={ref}
        type={type}
        disabled={disabled}
        data-testid={testId ?? "control-base-root"}
        data-rcl-control="true"
        data-control-density={density}
        data-control-shape={shape}
        data-control-size={size}
        data-control-variant={variant}
        className={className}
        style={{
          ...variantStyles[variant],
          ...sizeStyles[size],
          gap: density === "compact" ? "var(--space-3xs)" : "var(--space-2xs)",
          borderRadius: shape === "pill" ? "var(--radius-pill)" : "var(--radius-control)",
          ...style,
        }}
      >
        {children}
      </button>
    </>
  );
});
