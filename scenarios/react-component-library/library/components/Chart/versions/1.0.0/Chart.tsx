/** @vrooliComponentSource react-component-library:Chart */
import { translate } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

import {
  useId,
  useMemo,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import {
  AsyncBoundary,
  type AsyncBoundaryStatus,
} from "../../../AsyncBoundary/versions/1.0.0/AsyncBoundary";
import { useElementRect } from "../../../../hooks/useElementRect/versions/1.0.0/useElementRect";
import { useLocale } from "../../../../hooks/useLocale/versions/1.0.0/useLocale";

export type ChartPresentation = "contained" | "immersive";
export type ChartLevel = "controller" | "compound" | "convenience";
export type ChartPart = "plot" | "axis" | "legend" | "tooltip" | "annotation";
export type ChartStatus = AsyncBoundaryStatus | "empty";

export interface ChartDatum {
  id: string;
  label: string;
  value: number;
  detail?: string;
}

export interface ChartProps {
  data: ChartDatum[];
  title: string;
  description?: string;
  status?: ChartStatus;
  presentation?: ChartPresentation;
  valueFormatter?: (value: number, locale: string) => string;
  onRetry?: () => void | Promise<void>;
  emptyMessage?: ReactNode;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-chart] { min-inline-size: 0; color: var(--color-foreground, #0f172a); }
[data-rcl-chart-surface] { position: relative; min-inline-size: 0; overflow: hidden; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 16px); background: var(--color-surface, #fff); box-shadow: var(--elev-raised, 0 12px 36px rgb(15 23 42 / .08)); }
[data-rcl-chart-surface][data-presentation="immersive"] { border-inline: 0; border-radius: 0; box-shadow: none; background: transparent; }
[data-rcl-chart-header] { display: flex; align-items: end; justify-content: space-between; gap: var(--space-md, 16px); padding: var(--space-lg, 24px) var(--space-lg, 24px) var(--space-sm, 12px); }
[data-rcl-chart-heading] { min-inline-size: 0; }
[data-rcl-chart-kicker] { display: block; margin-block-end: 6px; color: var(--color-muted-foreground, #64748b); font: var(--text-label, 700 .75rem/1rem system-ui, sans-serif); letter-spacing: .12em; text-transform: uppercase; }
[data-rcl-chart-title] { margin: 0; font: var(--text-heading, 750 1.25rem/1.2 system-ui, sans-serif); letter-spacing: -.03em; }
[data-rcl-chart-description] { max-inline-size: 38rem; margin: 6px 0 0; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .875rem/1.4 system-ui, sans-serif); }
[data-rcl-chart-plot] { position: relative; min-block-size: 16rem; padding: var(--space-sm, 12px) var(--space-lg, 24px) 0; }
[data-rcl-chart-plot] svg { display: block; inline-size: 100%; block-size: 16rem; overflow: visible; }
[data-rcl-chart-grid] { stroke: var(--color-border, #cbd5e1); stroke-opacity: .7; stroke-dasharray: 3 6; vector-effect: non-scaling-stroke; }
[data-rcl-chart-axis] { fill: var(--color-muted-foreground, #64748b); font: 600 11px/1 system-ui, sans-serif; }
[data-rcl-chart-area] { fill: color-mix(in srgb, var(--color-primary, #2563eb) 12%, transparent); }
[data-rcl-chart-line] { fill: none; stroke: var(--color-primary, #2563eb); stroke-linecap: round; stroke-linejoin: round; stroke-width: 3; vector-effect: non-scaling-stroke; }
[data-rcl-chart-point] { fill: var(--color-surface, #fff); stroke: var(--color-primary, #2563eb); stroke-width: 3; vector-effect: non-scaling-stroke; }
[data-rcl-chart-point][data-selected="true"] { fill: var(--color-primary, #2563eb); stroke: var(--color-primary-foreground, #fff); stroke-width: 3; }
[data-rcl-chart-tooltip] { pointer-events: none; position: absolute; inset-block-start: var(--space-md, 16px); inset-inline-end: var(--space-lg, 24px); max-inline-size: 15rem; padding: var(--space-xs, 12px); border: 1px solid var(--color-border-strong, #94a3b8); border-radius: var(--radius-control, 8px); background: var(--color-surface-raised, #fff); box-shadow: var(--elev-overlay, 0 12px 28px rgb(15 23 42 / .16)); color: var(--color-foreground, #0f172a); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
[data-rcl-chart-tooltip] strong { display: block; margin-block-end: 2px; font-size: .8125rem; }
[data-rcl-chart-legend] { display: flex; flex-wrap: wrap; gap: var(--space-2xs, 8px); padding: var(--space-sm, 12px) var(--space-lg, 24px) var(--space-lg, 24px); border-block-start: 1px solid var(--color-border, #cbd5e1); }
[data-rcl-chart-legend] button { min-block-size: 2.75rem; display: inline-flex; align-items: center; gap: 8px; border: 1px solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 8px); padding: 0 10px; background: transparent; color: inherit; font: inherit; font-size: .8125rem; cursor: pointer; }
[data-rcl-chart-legend] button:hover { background: var(--color-surface-muted, #f1f5f9); }
[data-rcl-chart-legend] button:focus-visible { outline: 3px solid var(--color-focus-ring, #2563eb); outline-offset: 2px; }
[data-rcl-chart-legend-mark] { inline-size: 9px; block-size: 9px; border-radius: 50%; background: var(--color-primary, #2563eb); }
[data-rcl-chart-legend-value] { color: var(--color-muted-foreground, #64748b); }
[data-rcl-chart-table] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
[data-rcl-chart-annotation] { margin: 0 var(--space-lg, 24px) var(--space-lg, 24px); padding: var(--space-xs, 12px); border-inline-start: 3px solid var(--color-primary, #2563eb); background: var(--color-surface-muted, #f1f5f9); color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
@media (max-width: 30rem) { [data-rcl-chart-header] { display: grid; align-items: start; padding: var(--space-md, 16px) var(--space-md, 16px) var(--space-sm, 12px); } [data-rcl-chart-title] { overflow-wrap: anywhere; } [data-rcl-chart-tooltip] { position: static; max-inline-size: none; margin-block-start: var(--space-sm, 12px); } [data-rcl-chart-plot] { min-block-size: 13rem; padding-inline: var(--space-sm, 12px); } [data-rcl-chart-plot] svg { block-size: 13rem; } [data-rcl-chart-legend] { padding-inline: var(--space-md, 16px); } [data-rcl-chart-legend] button { flex: 1 1 8rem; justify-content: center; } [data-rcl-chart-annotation] { margin-inline: var(--space-md, 16px); } }
@media (forced-colors: active) { [data-rcl-chart-grid] { stroke: CanvasText; stroke-opacity: .45; } [data-rcl-chart-line], [data-rcl-chart-point] { stroke: Highlight; } [data-rcl-chart-area] { fill: Canvas; } [data-rcl-chart-legend-mark] { background: Highlight; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-chart-point] { transition: none; } }
`;

const defaultFormatter = (value: number, locale: string) =>
  new Intl.NumberFormat(locale, { maximumFractionDigits: 1 }).format(value);

export function Chart({
  data,
  title,
  description = translate("visualization.chart.description.1", "Explore the trend and select a point for its exact value."),
  status = "success",
  presentation = "contained",
  valueFormatter = defaultFormatter,
  onRetry,
  emptyMessage = "There is not enough information to draw this view yet.",
  className,
  style,
}: ChartProps) {
  const locale = useLocale();
  const [plotElement, setPlotElement] = useState<HTMLElement | null>(null);
  const rect = useElementRect(plotElement);
  const [selected, setSelected] = useState<string | null>(
    data.at(-1)?.id ?? null,
  );
  const titleId = useId();
  const boundaryStatus = status === "empty" ? "success" : status;
  const max = Math.max(...data.map((point) => point.value), 1);
  const min = Math.min(...data.map((point) => point.value), 0);
  const range = Math.max(max - min, 1);
  const width = Math.max(480, Math.round(rect?.width ?? 640));
  const height = 240;
  const padding = { left: 42, right: 16, top: 18, bottom: 34 };
  const plotWidth = width - padding.left - padding.right;
  const plotHeight = height - padding.top - padding.bottom;
  const points = data.map((point, index) => ({
    ...point,
    x:
      padding.left +
      (data.length <= 1
        ? plotWidth / 2
        : (index / (data.length - 1)) * plotWidth),
    y: padding.top + ((max - point.value) / range) * plotHeight,
  }));
  const line = points
    .map((point, index) => `${index === 0 ? "M" : "L"} ${point.x} ${point.y}`)
    .join(" ");
  const area =
    points.length > 0
      ? `${line} L ${points.at(-1)?.x ?? 0} ${height - padding.bottom} L ${points[0]?.x ?? 0} ${height - padding.bottom} Z`
      : "";
  const selectedPoint = points.find((point) => point.id === selected);
  const yTicks = useMemo(() => [max, min + range / 2, min], [max, min, range]);

  return (
    <div data-rcl-chart className={className} style={style}>
      <style data-rcl-chart-styles>{styles}</style>
      <AsyncBoundary
        status={boundaryStatus}
        retry={onRetry}
        errorTitle="Chart unavailable"
        error="We could not load this data. Try again when the source is available."
        offline={status === "offline"}
      >
        {status === "success" ||
        status === "refreshing" ||
        status === "stale" ||
        status === "partial-error" ? (
          <div data-rcl-chart-surface data-presentation={presentation}>
            <div data-rcl-chart-header>
              <div data-rcl-chart-heading>
                <span data-rcl-chart-kicker>{translate("visualization.chart.text.3", "Performance overview")}</span>
                <h2 id={titleId} data-rcl-chart-title>
                  {title}
                </h2>
                <p data-rcl-chart-description>{description}</p>
              </div>
              {selectedPoint && (
                <div data-rcl-chart-tooltip role="status">
                  <strong>{selectedPoint.label}</strong>
                  {valueFormatter(selectedPoint.value, locale)}
                  {selectedPoint.detail ? ` · ${selectedPoint.detail}` : ""}
                </div>
              )}
            </div>
            <div
              ref={setPlotElement}
              data-rcl-chart-plot
              data-rcl-chart-part="plot"
            >
              {data.length > 0 ? (
                <svg
                  role="img"
                  aria-labelledby={titleId}
                  viewBox={`0 0 ${width} ${height}`}
                  preserveAspectRatio="none"
                >
                  {yTicks.map((tick) => {
                    const y = padding.top + ((max - tick) / range) * plotHeight;
                    return (
                      <g key={tick} data-rcl-chart-part="axis">
                        <line
                          data-rcl-chart-grid
                          x1={padding.left}
                          x2={width - padding.right}
                          y1={y}
                          y2={y}
                        />
                        <text
                          data-rcl-chart-axis
                          x={padding.left - 8}
                          y={y + 4}
                          textAnchor="end"
                        >
                          {valueFormatter(tick, locale)}
                        </text>
                      </g>
                    );
                  })}
                  {points.map((point) => (
                    <text
                      key={`${point.id}-label`}
                      data-rcl-chart-axis
                      x={point.x}
                      y={height - 8}
                      textAnchor="middle"
                    >
                      {point.label}
                    </text>
                  ))}
                  <path data-rcl-chart-area d={area} aria-hidden="true" />
                  <path data-rcl-chart-line d={line} />
                  {points.map((point) => (
                    <circle
                      key={point.id}
                      data-rcl-chart-point
                      data-rcl-chart-part="plot"
                      data-selected={point.id === selected ? "true" : "false"}
                      cx={point.x}
                      cy={point.y}
                      r={point.id === selected ? 6 : 4}
                      aria-hidden="true"
                    />
                  ))}
                </svg>
              ) : (
                <div data-rcl-chart-annotation data-rcl-chart-part="annotation">
                  {emptyMessage}
                </div>
              )}
            </div>
            {data.length > 0 && (
              <div
                data-rcl-chart-legend
                data-rcl-chart-part="legend"
                aria-label={translate("visualization.chart.aria-label.2", "Chart values")}
              >
                {data.map((point) => (
                  <button data-testid="visualization.chart"
                    key={point.id}
                    type="button"
                    aria-pressed={point.id === selected}
                    onClick={() => setSelected(point.id)}
                  >
                    <span data-rcl-chart-legend-mark aria-hidden="true" />
                    <span>{point.label}</span>
                    <span data-rcl-chart-legend-value>
                      {valueFormatter(point.value, locale)}
                    </span>
                  </button>
                ))}
              </div>
            )}
            <table data-rcl-chart-table>
              <caption>{title} data</caption>
              <thead>
                <tr>
                  <th scope="col">{translate("visualization.chart.text.4", "Period")}</th>
                  <th scope="col">{translate("visualization.chart.text.5", "Value")}</th>
                </tr>
              </thead>
              <tbody>
                {data.map((point) => (
                  <tr key={point.id}>
                    <th scope="row">{point.label}</th>
                    <td>{valueFormatter(point.value, locale)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : status === "empty" ? (
          <div data-rcl-chart-annotation role="status">
            {emptyMessage}
          </div>
        ) : null}
      </AsyncBoundary>
    </div>
  );
}
