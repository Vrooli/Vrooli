/** @vrooliComponentSource primitives.progress */
import type { CSSProperties, HTMLAttributes } from "react";
import { useReducedMotion } from "../../../../hooks/useReducedMotion/versions/1.0.0/useReducedMotion";

export type ProgressShape = "linear" | "circular";
export type ProgressMode =
  | "determinate"
  | "indeterminate"
  | "buffered"
  | "segmented";

export interface ProgressProps
  extends Omit<
    HTMLAttributes<HTMLDivElement>,
    "aria-valuemax" | "aria-valuemin" | "aria-valuenow"
  > {
  value?: number;
  max?: number;
  bufferedValue?: number;
  segments?: number;
  shape?: ProgressShape;
  mode?: ProgressMode;
  label?: string;
  showValue?: boolean;
  tone?: "accent" | "success" | "warning" | "danger";
  size?: "sm" | "md" | "lg";
}

const styles = `
  [data-rcl-progress] {
    --rcl-progress-track: var(--color-border-subtle, color-mix(in srgb, var(--color-foreground, #0f172a) 12%, transparent));
    --rcl-progress-fill: var(--color-primary, #2563eb);
    --rcl-progress-buffer: color-mix(in srgb, var(--rcl-progress-fill) 28%, var(--rcl-progress-track));
    display: inline-flex;
    align-items: center;
    gap: var(--space-sm, .75rem);
    min-inline-size: 12rem;
    color: var(--color-foreground, #0f172a);
    font: var(--text-label, 600 .75rem/1.25rem system-ui, sans-serif);
  }
  [data-rcl-progress][data-size="sm"] { --rcl-progress-thickness: .25rem; }
  [data-rcl-progress][data-size="md"] { --rcl-progress-thickness: .5rem; }
  [data-rcl-progress][data-size="lg"] { --rcl-progress-thickness: .75rem; }
  [data-rcl-progress-track] {
    position: relative;
    flex: 1 1 auto;
    min-inline-size: 0;
    overflow: hidden;
    block-size: var(--rcl-progress-thickness);
    border-radius: var(--radius-pill, 999px);
    background: var(--rcl-progress-track);
    isolation: isolate;
  }
  [data-rcl-progress-layer] {
    position: absolute;
    inset-block: 0;
    inset-inline-start: 0;
    inline-size: 100%;
    border-radius: inherit;
    transform-origin: left center;
    transition: transform var(--dur-moderate, 280ms) var(--ease-standard, cubic-bezier(.2,.8,.2,1));
  }
  [data-rcl-progress-buffer] { background: var(--rcl-progress-buffer); z-index: 0; }
  [data-rcl-progress-fill] { background: var(--rcl-progress-fill); z-index: 1; }
  [data-rcl-progress][data-mode="indeterminate"] [data-rcl-progress-fill] {
    inline-size: 42%;
    animation: rcl-progress-sweep var(--dur-slow, 900ms) var(--ease-standard, cubic-bezier(.2,.8,.2,1)) infinite;
  }
  [data-rcl-progress-segments] { display: flex; gap: var(--space-3xs, .25rem); position: absolute; inset: 0; z-index: 2; }
  [data-rcl-progress-segment] { flex: 1 1 0; background: var(--rcl-progress-track); border-radius: var(--radius-pill, 999px); }
  [data-rcl-progress-segment][data-filled="true"] { background: var(--rcl-progress-fill); }
  [data-rcl-progress][data-shape="circular"] { min-inline-size: auto; }
  [data-rcl-progress-circle] { transform: rotate(-90deg); overflow: visible; }
  [data-rcl-progress-circle-track], [data-rcl-progress-circle-fill] { fill: none; stroke-width: 8; }
  [data-rcl-progress-circle-track] { stroke: var(--rcl-progress-track); }
  [data-rcl-progress-circle-fill] {
    stroke: var(--rcl-progress-fill);
    stroke-linecap: round;
    transition: stroke-dashoffset var(--dur-moderate, 280ms) var(--ease-standard, cubic-bezier(.2,.8,.2,1));
  }
  [data-rcl-progress-value] { min-inline-size: 2.75rem; color: var(--color-muted-foreground, #64748b); text-align: end; font-variant-numeric: tabular-nums; }
  @keyframes rcl-progress-sweep { 0% { transform: translateX(-110%); } 100% { transform: translateX(260%); } }
  @media (prefers-reduced-motion: reduce) {
    [data-rcl-progress-layer], [data-rcl-progress-circle-fill] { transition: none; }
    [data-rcl-progress][data-mode="indeterminate"] [data-rcl-progress-fill] { animation: none; transform: translateX(0); }
  }
`;

const toneColors: Record<NonNullable<ProgressProps["tone"]>, string> = {
  accent: "var(--color-primary, #2563eb)",
  success: "var(--color-success, #15803d)",
  warning: "var(--color-warning, #b45309)",
  danger: "var(--color-danger, #dc2626)",
};

function clamp(value: number, max: number) {
  return Math.max(0, Math.min(max, Number.isFinite(value) ? value : 0));
}

export function Progress({
  value = 0,
  max = 100,
  bufferedValue,
  segments = 5,
  shape = "linear",
  mode = "determinate",
  label = "Progress",
  showValue = true,
  tone = "accent",
  size = "md",
  style,
  ...props
}: ProgressProps) {
  const reducedMotion = useReducedMotion();
  const safeMax = Math.max(1, max);
  const percentage = (clamp(value, safeMax) / safeMax) * 100;
  const bufferedPercentage =
    (clamp(bufferedValue ?? value, safeMax) / safeMax) * 100;
  const displayValue = `${Math.round(percentage)}%`;
  const common = {
    "aria-label": label,
    "aria-valuemin": 0,
    "aria-valuemax": safeMax,
    ...(mode === "indeterminate"
      ? {}
      : { "aria-valuenow": clamp(value, safeMax) }),
  };
  const mergedStyle = {
    "--rcl-progress-fill": toneColors[tone],
    ...style,
  } as CSSProperties;

  return (
    <>
      <style
        data-rcl-progress-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <div
        {...props}
        {...common}
        data-rcl-progress="true"
        data-mode={mode}
        data-shape={shape}
        data-size={size}
        role="progressbar"
        style={mergedStyle}
      >
        {shape === "circular" ? (
          <svg
            data-rcl-progress-circle
            width="72"
            height="72"
            viewBox="0 0 72 72"
            aria-hidden="true"
          >
            <circle data-rcl-progress-circle-track cx="36" cy="36" r="28" />
            <circle
              data-rcl-progress-circle-fill
              cx="36"
              cy="36"
              r="28"
              strokeDasharray={`${2 * Math.PI * 28}`}
              strokeDashoffset={
                mode === "indeterminate"
                  ? `${2 * Math.PI * 28 * 0.28}`
                  : `${2 * Math.PI * 28 * (1 - percentage / 100)}`
              }
              style={reducedMotion ? { transition: "none" } : undefined}
            />
          </svg>
        ) : (
          <div data-rcl-progress-track>
            {mode === "buffered" && (
              <span
                data-rcl-progress-layer
                data-rcl-progress-buffer
                style={{ transform: `scaleX(${bufferedPercentage / 100})` }}
              />
            )}
            {mode === "segmented" ? (
              <span data-rcl-progress-segments>
                {Array.from(
                  { length: Math.max(2, Math.min(12, segments)) },
                  (_, index) => (
                    <span
                      data-rcl-progress-segment
                      data-filled={
                        index < Math.round((percentage / 100) * segments)
                      }
                      key={index}
                    />
                  ),
                )}
              </span>
            ) : (
              <span
                data-rcl-progress-layer
                data-rcl-progress-fill
                style={{
                  transform:
                    mode === "indeterminate"
                      ? undefined
                      : `scaleX(${percentage / 100})`,
                }}
              />
            )}
          </div>
        )}
        {showValue && (
          <span data-rcl-progress-value aria-hidden="true">
            {mode === "indeterminate" ? "Working" : displayValue}
          </span>
        )}
      </div>
    </>
  );
}
