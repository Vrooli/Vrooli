/**
 * @libraryId react-component-library:TrendSpark
 * @displayName TrendSpark
 * @description An accessible inline trend visualization for compact history rows.
 * @version 1.0.1
 * @tags ["primitive","data-display","visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource data-display.trend-spark */
import type { SVGAttributes } from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export interface TrendSparkProps
  extends Omit<SVGAttributes<SVGSVGElement>, "children" | "role" | "values"> {
  values: number[];
  label: string;
  tone?: "neutral" | "success" | "warning" | "danger";
}

const toneClass: Record<NonNullable<TrendSparkProps["tone"]>, string> = {
  neutral: "rcl-trend-spark-neutral",
  success: "rcl-trend-spark-success",
  warning: "rcl-trend-spark-warning",
  danger: "rcl-trend-spark-danger",
};

const styles = `
[data-rcl-trend-spark] { display: block; inline-size: 6rem; block-size: 2rem; min-inline-size: 0; overflow: visible; }
[data-rcl-trend-spark] polyline { stroke: currentColor; stroke-linecap: round; stroke-linejoin: round; stroke-width: 2.5; }
.rcl-trend-spark-neutral { color: var(--color-muted-foreground, #64748b); }
.rcl-trend-spark-success { color: var(--color-success, #16803c); }
.rcl-trend-spark-warning { color: var(--color-warning, #b45309); }
.rcl-trend-spark-danger { color: var(--color-danger, #dc2626); }

`;

function pointsFor(values: number[]): string {
  const points =
    values.slice(-12).length >= 2 ? values.slice(-12) : [values[0] ?? 0, values[0] ?? 0];
  const min = Math.min(...points);
  const max = Math.max(...points);
  const range = max - min || 1;
  return points
    .map(
      (value, index) =>
        `${(index / (points.length - 1)) * 100},${28 - ((value - min) / range) * 24}`,
    )
    .join(" ");
}

export function TrendSpark({
  values,
  label,
  tone = "neutral",
  className,
  ...props
}: TrendSparkProps) {
  const points = pointsFor(values);
  return (
    <>
      <StyleSheet name="trend-spark" css={styles} />
      <svg
        {...props}
        role="img"
        aria-label={label}
        viewBox="0 0 100 32"
        preserveAspectRatio="none"
        className={`${toneClass[tone]} ${className ?? ""}`.trim()}
        data-rcl-trend-spark
      >
        <polyline points={points} fill="none" vectorEffect="non-scaling-stroke" />
      </svg>
    </>
  );
}
