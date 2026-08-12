/**
 * @vrooliComponentSource react-component-library:ControlBase
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption b11fce87-57ac-4860-b70f-75c25987167f
 * @vrooliComponentAppliedAt 2026-08-12T02:09:05Z
 * @vrooliComponentSourceSha256 16389ca051fbbe63df470e938c61bf07028e2ccb4ab6a893f341ef697551c3d4
 * @vrooliComponentDriftHash eca7809528d46f910d8ff6b9bd6cf386c43af8bce7a1a47a6e65bbb8e07e574e
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import {
  forwardRef,
  type ButtonHTMLAttributes,
  type CSSProperties,
  type ReactNode,
} from "react";
import { motionTransition } from "./foundations/VisualRecipes";

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

export interface ControlBaseProps
  extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, "color"> {
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
  box-sizing: border-box;
      isolation: isolate;
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
  letter-spacing: var(--text-label-tracking, 0);
  cursor: pointer;
  user-select: none;
  white-space: nowrap;
  -webkit-tap-highlight-color: transparent;
  transition: ${motionTransition(["transform", "filter", "box-shadow", "background-color", "border-color", "color"])};
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
[data-rcl-control][data-rcl-pending="true"] {
  cursor: progress;
}
[data-rcl-control]:focus-visible {
  outline: var(--border-strong) solid var(--color-focus);
  outline-offset: var(--space-3xs);
}
[data-rcl-control]:disabled {
  cursor: not-allowed;
  opacity: max(var(--opacity-disabled), .68);
}
@media (prefers-reduced-motion: reduce) {
  [data-rcl-control] { transition: none; }
  [data-rcl-control]:hover:not(:disabled), [data-rcl-control]:active:not(:disabled) { transform: none; }
}
`;

function ControlStyles() {
  return (
    <style
      data-rcl-control-styles
      dangerouslySetInnerHTML={{ __html: styleSheet }}
    />
  );
}

export const ControlBase = forwardRef<HTMLButtonElement, ControlBaseProps>(
  function ControlBase(
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
    const testId = (props as ControlBaseProps & { "data-testid"?: string })[
      "data-testid"
    ];
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
            gap:
              density === "compact" ? "var(--space-3xs)" : "var(--space-2xs)",
            borderRadius:
              shape === "pill" ? "var(--radius-pill)" : "var(--radius-control)",
            ...style,
          }}
        >
          {children}
        </button>
      </>
    );
  },
);
