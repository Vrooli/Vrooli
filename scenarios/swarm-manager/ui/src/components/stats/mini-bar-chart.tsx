import { useEffect, useRef, useState } from "react";
import { toBarPercent } from "../../lib/stats-format-utils";

export interface MiniBarChartPoint {
  key: string;
  label: string;
  value: number;
  secondaryValue?: number;
}

interface MiniBarChartProps {
  points: MiniBarChartPoint[];
  height?: number;
  testId?: string;
  valueLabel?: string;
  secondaryValueLabel?: string;
  onSelect?: (point: MiniBarChartPoint) => void;
}

/** Format an ISO week-start date to a compact "M/D" axis label. */
function shortWeekLabel(label: string): string {
  const parsed = new Date(label);
  if (Number.isNaN(parsed.getTime())) return label;
  return `${parsed.getUTCMonth() + 1}/${parsed.getUTCDate()}`;
}

/**
 * A compact bar chart whose SVG is sized to its measured container width so
 * bars and text scale 1:1 — no `preserveAspectRatio="none"` stretch. Value
 * labels sit above each bar and week-start labels run along the x-axis.
 */
export function MiniBarChart({
  points,
  height = 112,
  testId,
  valueLabel = "completed",
  secondaryValueLabel,
  onSelect,
}: MiniBarChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;
    const update = () => setWidth(el.clientWidth);
    update();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(update);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  // Fall back to a sensible width before the first measurement (e.g. SSR/tests).
  const chartWidth = width > 0 ? width : Math.max(points.length * 28, 160);
  const valueRowHeight = 14;
  const labelRowHeight = 16;
  const baselineY = height - labelRowHeight;
  const plotTop = valueRowHeight;
  const maxValue = Math.max(...points.flatMap((point) => [point.value, point.secondaryValue ?? 0]), 1);
  const barGap = 6;
  const barWidth = Math.max(4, (chartWidth - barGap * (points.length + 1)) / Math.max(1, points.length));
  const isDualSeries = points.some((point) => typeof point.secondaryValue === "number");
  const seriesGap = isDualSeries ? 2 : 0;
  const seriesBarWidth = isDualSeries ? Math.max(3, (barWidth - seriesGap) / 2) : barWidth;
  // Thin out x-axis labels so they never collide when many weeks are shown.
  const labelStep = Math.max(1, Math.ceil(points.length / 6));

  return (
    <div ref={containerRef} className="w-full">
      <svg
        role="img"
        aria-label="Velocity trend"
        width={chartWidth}
        height={height}
        viewBox={`0 0 ${chartWidth} ${height}`}
        className="overflow-visible"
        data-testid={testId}
      >
        <line x1="0" y1={baselineY} x2={chartWidth} y2={baselineY} className="stroke-slate-700" strokeWidth="1" />
        {points.map((point, index) => {
          const percent = toBarPercent(point.value, maxValue);
          const barHeight = Math.max(2, (percent / 100) * (baselineY - plotTop));
          const secondaryPercent = toBarPercent(point.secondaryValue ?? 0, maxValue);
          const secondaryBarHeight = Math.max(2, (secondaryPercent / 100) * (baselineY - plotTop));
          const x = barGap + index * (barWidth + barGap);
          const y = baselineY - barHeight;
          const secondaryY = baselineY - secondaryBarHeight;
          const showLabel = index % labelStep === 0 || index === points.length - 1;
          const titleParts = [`${point.label}: ${point.value.toLocaleString()} ${valueLabel}`];
          if (typeof point.secondaryValue === "number" && secondaryValueLabel) {
            titleParts.push(`${point.secondaryValue.toLocaleString()} ${secondaryValueLabel}`);
          }

          return (
            <g
              key={point.key}
              onClick={() => onSelect?.(point)}
              className={onSelect ? "cursor-pointer" : undefined}
              role={onSelect ? "button" : undefined}
              tabIndex={onSelect ? 0 : undefined}
              aria-label={onSelect ? `${point.label}: ${point.value} ${valueLabel}` : undefined}
              onKeyDown={(event) => {
                if (onSelect && (event.key === "Enter" || event.key === " ")) {
                  event.preventDefault();
                  onSelect(point);
                }
              }}
            >
              <title>{titleParts.join(", ")}</title>
              <rect x={x} y={y} width={seriesBarWidth} height={barHeight} rx="3" className="fill-cyan-400/70" />
              {typeof point.secondaryValue === "number" && (
                <rect
                  x={x + seriesBarWidth + seriesGap}
                  y={secondaryY}
                  width={seriesBarWidth}
                  height={secondaryBarHeight}
                  rx="3"
                  className="fill-emerald-400/70"
                />
              )}
              <text
                x={x + barWidth / 2}
                y={Math.min(y, secondaryY) - 3}
                textAnchor="middle"
                className="fill-slate-400 text-[10px]"
              >
                {isDualSeries ? `${point.value}/${point.secondaryValue ?? 0}` : point.value}
              </text>
              {showLabel && (
                <text
                  x={x + barWidth / 2}
                  y={height - 4}
                  textAnchor="middle"
                  className="fill-slate-500 text-[10px]"
                >
                  {shortWeekLabel(point.label)}
                </text>
              )}
            </g>
          );
        })}
      </svg>
    </div>
  );
}
