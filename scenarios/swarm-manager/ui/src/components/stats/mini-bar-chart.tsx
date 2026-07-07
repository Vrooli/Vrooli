import { useEffect, useRef, useState } from "react";
import { toBarPercent } from "../../lib/stats-format-utils";

interface MiniBarChartPoint {
  key: string;
  label: string;
  value: number;
}

interface MiniBarChartProps {
  points: MiniBarChartPoint[];
  height?: number;
  testId?: string;
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
export function MiniBarChart({ points, height = 112, testId }: MiniBarChartProps) {
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
  const maxValue = Math.max(...points.map((point) => point.value), 1);
  const barGap = 6;
  const barWidth = Math.max(4, (chartWidth - barGap * (points.length + 1)) / Math.max(1, points.length));
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
          const x = barGap + index * (barWidth + barGap);
          const y = baselineY - barHeight;
          const showLabel = index % labelStep === 0 || index === points.length - 1;

          return (
            <g key={point.key}>
              <title>{`${point.label}: ${point.value.toLocaleString()} completed`}</title>
              <rect x={x} y={y} width={barWidth} height={barHeight} rx="3" className="fill-cyan-400/70" />
              <text
                x={x + barWidth / 2}
                y={y - 3}
                textAnchor="middle"
                className="fill-slate-400 text-[10px]"
              >
                {point.value}
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
