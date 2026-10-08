import { useState, useMemo, useCallback, useEffect, Children, isValidElement, useId } from 'react';
import type { ReactNode, ReactElement, CSSProperties } from 'react';
import {
  ResponsiveContainer,
  ComposedChart,
  Line,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend
} from 'recharts';
import { ArrowLeft } from 'lucide-react';

import { formatTimeLabel, formatChartTime } from '../../../shared/utils/formatters';

// ── Interfaces ──────────────────────────────────────────────────────────────

export interface MetricDetailLayoutProps {
  layoutId: string;
  title: string;
  icon: ReactNode;
  headline: string;
  subhead?: string;
  onBack: () => void;
  children: ReactNode;
}

export type DetailDensity = 'compact' | 'comfortable' | 'dense';
export type DetailColumnCount = 1 | 2 | 3;

export interface DetailSectionProps {
  id: string;
  title: string;
  children: ReactNode;
}

export interface DetailLayoutPreference {
  order: string[];
  hidden: string[];
  density: DetailDensity;
  columns: DetailColumnCount;
}

const DETAIL_LAYOUT_VERSION = 1;

export const DetailSection = ({ id, title, children }: DetailSectionProps) => (
  <section
    data-detail-section={id}
    data-detail-title={title}
    className="detail-layout-section"
    aria-label={title}
  >
    {children}
  </section>
);

function isDetailSectionElement(child: ReactNode): child is ReactElement<DetailSectionProps> {
  return isValidElement(child) && child.type === DetailSection;
}

function normalizeDetailPreference(
  raw: unknown,
  sectionIds: string[],
): DetailLayoutPreference {
  const defaults: DetailLayoutPreference = {
    order: sectionIds,
    hidden: [],
    density: 'comfortable',
    columns: 1,
  };
  if (!raw || typeof raw !== 'object') return defaults;
  const candidate = raw as Partial<DetailLayoutPreference> & { version?: unknown };
  if (candidate.version !== undefined && candidate.version !== DETAIL_LAYOUT_VERSION) return defaults;
  const order = Array.isArray(candidate.order)
    ? candidate.order.filter((id): id is string => typeof id === 'string' && sectionIds.includes(id))
    : [];
  const hidden = Array.isArray(candidate.hidden)
    ? candidate.hidden.filter((id): id is string => typeof id === 'string' && sectionIds.includes(id))
    : [];
  const completeOrder = [...order, ...sectionIds.filter(id => !order.includes(id))];
  const density = candidate.density === 'compact' || candidate.density === 'dense' ? candidate.density : 'comfortable';
  const columns = candidate.columns === 2 || candidate.columns === 3 ? candidate.columns : 1;
  return { order: completeOrder, hidden: hidden.filter((id, index) => hidden.indexOf(id) === index), density, columns };
}

function readDetailPreference(key: string, sectionIds: string[]): DetailLayoutPreference {
  try {
    const stored = localStorage.getItem(key);
    return stored ? normalizeDetailPreference(JSON.parse(stored), sectionIds) : normalizeDetailPreference(null, sectionIds);
  } catch {
    return normalizeDetailPreference(null, sectionIds);
  }
}

function DetailLayoutControls({
  sections,
  preference,
  onChange,
  onReset,
}: {
  sections: Array<{ id: string; title: string }>;
  preference: DetailLayoutPreference;
  onChange: (next: DetailLayoutPreference) => void;
  onReset: () => void;
}) {
  const move = (id: string, direction: -1 | 1) => {
    const index = preference.order.indexOf(id);
    const target = index + direction;
    if (index < 0 || target < 0 || target >= preference.order.length) return;
    const order = [...preference.order];
    const current = order[index];
    const next = order[target];
    if (current === undefined || next === undefined) return;
    [order[index], order[target]] = [next, current];
    onChange({ ...preference, order });
  };

  return (
    <div className="detail-layout-controls" aria-label="Detail layout controls" data-testid="detail-layout-controls">
      <div className="detail-layout-controls__row">
        <label>
          Density
          <select
            aria-label="Detail density"
            data-testid="detail-density"
            value={preference.density}
            onChange={event => onChange({ ...preference, density: event.target.value as DetailDensity })}
          >
            <option value="compact">Compact</option>
            <option value="comfortable">Comfortable</option>
            <option value="dense">Dense</option>
          </select>
        </label>
        <label>
          Columns
          <select
            aria-label="Detail columns"
            data-testid="detail-columns"
            value={preference.columns}
            onChange={event => onChange({ ...preference, columns: Number(event.target.value) as DetailColumnCount })}
          >
            <option value="1">One</option>
            <option value="2">Two</option>
            <option value="3">Three</option>
          </select>
        </label>
        <button type="button" className="btn btn-action" data-testid="detail-layout-reset" onClick={onReset}>Reset to default</button>
      </div>
      <ul className="detail-layout-controls__sections">
        {preference.order.map((id, index) => {
          const section = sections.find(candidate => candidate.id === id);
          if (!section) return null;
          const hidden = preference.hidden.includes(id);
          return (
            <li key={id}>
              <label>
                <input
                  type="checkbox"
                  checked={!hidden}
                  onChange={() => onChange({
                    ...preference,
                    hidden: hidden ? preference.hidden.filter(value => value !== id) : [...preference.hidden, id],
                  })}
                />
                {section.title}
              </label>
              <span>
                <button type="button" className="btn btn-icon" aria-label={`Move ${section.title} up`} disabled={index === 0} onClick={() => move(id, -1)}>↑</button>
                <button type="button" className="btn btn-icon" aria-label={`Move ${section.title} down`} disabled={index === preference.order.length - 1} onClick={() => move(id, 1)}>↓</button>
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

interface MetricLineChartLineConfig {
  dataKey: string;
  name: string;
  color: string;
  strokeWidth?: number;
  type?: 'linear' | 'monotone' | 'natural' | 'stepAfter' | 'stepBefore';
  yAxisId?: 'left' | 'right';
}

/**
 * Why a chart has three empty states, not one.
 *
 * `loading` (history not fetched yet), `empty` (fetched, nothing in the
 * window) and `error` (the series could not be loaded) look identical to a
 * user if they render the same sentence — which is how a hard backend failure
 * can sit unnoticed behind what reads as a slow spinner. They are kept
 * distinct here deliberately.
 */
export type MetricChartStatus = 'loading' | 'ready' | 'error';

export interface MetricLineChartProps {
  data: Array<{ timestamp: string } & Record<string, number | string>>;
  lines: MetricLineChartLineConfig[];
  unit?: string;
  height?: number;
  yDomain?: [number | 'auto', number | 'auto'];
  rightYAxisUnit?: string;
  rightYAxisDomain?: [number | 'auto', number | 'auto'];
  yAxisScale?: 'linear' | 'log';
  valueFormatter?: (value: number) => string;
  className?: string;
  style?: CSSProperties;
  /** Defaults to 'ready' so existing call sites keep their behaviour. */
  status?: MetricChartStatus;
  /** Shown under the error title when status is 'error'. */
  errorMessage?: string;
  /** Label used in the empty state, e.g. "memory". */
  seriesLabel?: string;
}

// ── Shared Components ───────────────────────────────────────────────────────

export const MetricDetailLayout = ({ layoutId, title, icon, headline, subhead, onBack, children }: MetricDetailLayoutProps) => {
  const sectionElements = Children.toArray(children).filter(isDetailSectionElement);
  const sectionIds = sectionElements.map(section => section.props.id);
  const preferenceKey = `system-monitor-detail-layout:${layoutId}`;
  const [preference, setPreference] = useState<DetailLayoutPreference>(() => readDetailPreference(preferenceKey, sectionIds));
  const [customizing, setCustomizing] = useState(false);

  useEffect(() => {
    setPreference(current => normalizeDetailPreference(current, sectionIds));
  }, [sectionIds.join('|')]);

  useEffect(() => {
    try {
      localStorage.setItem(preferenceKey, JSON.stringify({ version: DETAIL_LAYOUT_VERSION, ...preference }));
    } catch {
      // Preferences are optional. A storage failure must not affect the page.
    }
  }, [preference, preferenceKey]);

  const reset = () => {
    const defaults = normalizeDetailPreference(null, sectionIds);
    setPreference(defaults);
    try { localStorage.removeItem(preferenceKey); } catch { /* optional storage */ }
  };
  const visibleSections = preference.order
    .filter(id => !preference.hidden.includes(id))
    .map(id => sectionElements.find(section => section.props.id === id))
    .filter((section): section is ReactElement<DetailSectionProps> => Boolean(section));

  return (
  <div className="flex-col-gap-lg">
    <div className="metric-detail-toolbar">
      <button
        type="button"
        className="btn btn-action metric-detail-back-btn"
        onClick={onBack}
      >
        <ArrowLeft size={16} />
        Back To Dashboard
      </button>
      <div className="icon-text gap-md metric-detail-heading">
        <div className="metric-detail-title">
          {icon}
          <span>{title}</span>
        </div>
        <div className="metric-detail-headline">{headline}</div>
      </div>
    </div>
    <div className="detail-layout-toolbar">
      <span className="detail-layout-toolbar__label">{sectionIds.length} detail sections</span>
      <button type="button" className="btn btn-action" data-testid="detail-layout-customize" aria-expanded={customizing} onClick={() => setCustomizing(value => !value)}>
        {customizing ? 'Close layout controls' : 'Customize layout'}
      </button>
    </div>
    {customizing ? (
      <DetailLayoutControls
        sections={sectionElements.map(section => ({ id: section.props.id, title: section.props.title }))}
        preference={preference}
        onChange={setPreference}
        onReset={reset}
      />
    ) : null}
    {subhead && (
      <div className="card-subtitle">
        {subhead}
      </div>
    )}
    <div className="detail-layout-content" data-density={preference.density} data-columns={preference.columns}>
      {sectionElements.length > 0 ? visibleSections : children}
    </div>
  </div>
  );
};

// ── Private Helpers ─────────────────────────────────────────────────────────

const defaultValueFormatter = (unit?: string) => (value: number) => {
  if (!Number.isFinite(value)) {
    return '—';
  }
  if (unit) {
    return `${value.toFixed(2)}${unit}`;
  }
  return value.toFixed(2);
};

// ── Custom Tooltip ──────────────────────────────────────────────────────────

interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{
    value: number;
    name: string;
    color: string;
    dataKey: string;
  }>;
  label?: string;
  formatter: (value: number) => string;
  hiddenSeries: Set<string>;
}

const ChartTooltip = ({ active, payload, label, formatter, hiddenSeries }: CustomTooltipProps) => {
  if (!active || !payload || payload.length === 0) return null;

  const visiblePayload = payload.filter(entry => !hiddenSeries.has(entry.dataKey));
  if (visiblePayload.length === 0) return null;
  const formattedLabel = typeof label === 'string' ? formatTimeLabel(label) : '—';

  return (
    <div
      data-sm-style="sm-style-674a98adf0"
    >
      <div
        data-sm-style="sm-style-f2d36b52d9"
      >
        {formattedLabel}
      </div>
      {visiblePayload.map(entry => (
        <div
          key={entry.dataKey}
          data-sm-style="sm-style-3a3c6a968c"
        >
          <span
            style={{
              width: 8,
              height: 8,
              borderRadius: '50%',
              background: entry.color,
              flexShrink: 0
            }}
          />
          <span data-sm-style="sm-style-2f52d20bb2">{entry.name}</span>
          <span data-sm-style="sm-style-dee1e2e177">
            {Number.isFinite(entry.value) ? formatter(entry.value) : '—'}
          </span>
        </div>
      ))}
    </div>
  );
};

// ── Custom Cursor ───────────────────────────────────────────────────────────

const ChartCursor = (props: { points?: Array<{ x: number }>; height?: number }) => {
  const { points, height } = props;
  if (!points || points.length === 0) return null;
  const firstPoint = points[0];
  if (!firstPoint) return null;
  const x = firstPoint.x;
  return (
    <line
      x1={x}
      x2={x}
      y1={0}
      y2={height ?? 0}
      stroke="var(--chart-cursor-color)"
      strokeWidth={1}
      strokeDasharray="4 3"
    />
  );
};

// ── Chart Placeholders ──────────────────────────────────────────────────────

const ChartSkeleton = ({ height }: { height: number }) => (
  <div
    className="chart-placeholder"
    style={{ height }}
    role="status"
    aria-live="polite"
    aria-label="Loading chart data"
  >
    <div className="chart-skeleton__frame" aria-hidden="true">
      <div className="chart-skeleton__axis-y">
        {[0, 1, 2, 3, 4].map(tick => (
          <span key={tick} className="loading-skeleton__block chart-skeleton__tick" />
        ))}
      </div>
      <div className="chart-skeleton__plot">
        <div className="chart-skeleton__wave" />
      </div>
      <div className="chart-skeleton__axis-x">
        {[0, 1, 2, 3, 4, 5].map(tick => (
          <span key={tick} className="loading-skeleton__block chart-skeleton__tick" />
        ))}
      </div>
    </div>
    <div className="chart-placeholder__caption">Loading timeseries…</div>
  </div>
);

const ChartEmptyState = ({ height, seriesLabel }: { height: number; seriesLabel?: string }) => (
  <div className="chart-placeholder chart-placeholder--empty" style={{ height }} role="status">
    <div className="chart-placeholder__title">No data in this window</div>
    <div className="chart-placeholder__caption">
      {seriesLabel
        ? `No ${seriesLabel} samples were recorded for the selected time range.`
        : 'No samples were recorded for the selected time range.'}
    </div>
  </div>
);

const ChartErrorState = ({ height, errorMessage }: { height: number; errorMessage?: string }) => (
  <div className="chart-placeholder chart-placeholder--error" style={{ height }} role="alert">
    <div className="chart-placeholder__title">Timeseries unavailable</div>
    <div className="chart-placeholder__caption">
      {errorMessage ?? 'The metrics history could not be loaded.'}
    </div>
  </div>
);

// ── MetricLineChart ─────────────────────────────────────────────────────────

export const MetricLineChart = ({
  data,
  lines,
  unit,
  height = 320,
  yDomain = ['auto', 'auto'],
  rightYAxisUnit,
  rightYAxisDomain = ['auto', 'auto'],
  yAxisScale = 'linear',
  valueFormatter,
  className,
  style,
  status = 'ready',
  errorMessage,
  seriesLabel
}: MetricLineChartProps) => {
  const [hiddenSeries, setHiddenSeries] = useState<Set<string>>(new Set());
  const [window, setWindow] = useState({ start: 0, end: Math.max(data.length - 1, 0) });
  const chartId = useId().replace(/:/g, '');

  useEffect(() => {
    setWindow({ start: 0, end: Math.max(data.length - 1, 0) });
  }, [data.length]);

  const visibleData = useMemo(() => data.slice(window.start, window.end + 1), [data, window]);
  const canZoomIn = visibleData.length >= 4;
  const canPan = visibleData.length > 1 && (window.start > 0 || window.end < data.length - 1);
  const zoom = (direction: 'in' | 'out') => {
    if (data.length < 2) return;
    if (direction === 'out') {
      setWindow({ start: 0, end: data.length - 1 });
      return;
    }
    const span = window.end - window.start + 1;
    if (span < 4) return;
    const nextSpan = Math.max(3, Math.ceil(span * 0.75));
    const center = (window.start + window.end) / 2;
    const nextStart = Math.max(0, Math.floor(center - nextSpan / 2));
    setWindow({ start: nextStart, end: Math.min(data.length - 1, nextStart + nextSpan - 1) });
  };
  const pan = (direction: -1 | 1) => {
    const span = window.end - window.start + 1;
    const step = Math.max(1, Math.floor(span * 0.25));
    const start = Math.max(0, Math.min(data.length - span, window.start + direction * step));
    setWindow({ start, end: start + span - 1 });
  };

  const gradientIds = useMemo(
    () => Object.fromEntries(lines.map(line => [line.dataKey, `grad-${chartId}-${line.dataKey}`])),
    [chartId, lines]
  );

  const toggleSeries = useCallback((key: string) => {
    if (!key) return;
    setHiddenSeries(prev => {
      const next = new Set(prev);
      if (next.has(key)) {
        next.delete(key);
      } else {
        if (next.size < lines.length - 1) {
          next.add(key);
        }
      }
      return next;
    });
  }, [lines.length]);

  const fmtr = valueFormatter ?? defaultValueFormatter(unit);
  const summary = data.length === 0
    ? `No ${seriesLabel ?? 'metric'} samples are available in the selected time range.`
    : `${seriesLabel ?? 'Metric'} chart showing ${data.length} samples${visibleData.length !== data.length ? `, ${visibleData.length} currently visible` : ''}. Use the chart controls to zoom, pan, or reset the view.`;

  return (
    <div className={className} style={{ width: '100%', ...(style ?? {}) }}>
      <div className="metric-chart-toolbar" data-testid="metric-chart-controls" aria-label="Chart controls">
        <span className="metric-chart-toolbar__label">{visibleData.length} of {data.length} samples</span>
        <div className="metric-chart-toolbar__actions">
          <button type="button" className="btn btn-icon" aria-label="Zoom in" onClick={() => zoom('in')} disabled={!canZoomIn}>+</button>
          <button type="button" className="btn btn-icon" aria-label="Zoom out" onClick={() => zoom('out')} disabled={!canPan}>−</button>
          <button type="button" className="btn btn-icon" aria-label="Pan chart left" onClick={() => pan(-1)} disabled={window.start === 0}>←</button>
          <button type="button" className="btn btn-icon" aria-label="Pan chart right" onClick={() => pan(1)} disabled={window.end >= data.length - 1}>→</button>
          <button type="button" className="btn btn-action metric-chart-toolbar__reset" aria-label="Reset chart view" onClick={() => setWindow({ start: 0, end: Math.max(data.length - 1, 0) })} disabled={!canPan}>Reset view</button>
        </div>
      </div>
      <p className="sr-only" data-testid="metric-chart-summary">{summary}</p>
      {data.length > 0 ? (
        <ResponsiveContainer width="100%" height={height}>
          <ComposedChart data={visibleData} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
            <defs>
              {lines.map(line => (
                <linearGradient key={line.dataKey} id={gradientIds[line.dataKey]} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="0%" stopColor={line.color} stopOpacity={0.25} />
                  <stop offset="100%" stopColor={line.color} stopOpacity={0} />
                </linearGradient>
              ))}
            </defs>
            <CartesianGrid stroke="var(--chart-grid)" strokeDasharray="4 4" />
            <XAxis
              dataKey="timestamp"
              stroke="var(--color-text-secondary)"
              tickFormatter={formatChartTime}
              minTickGap={20}
              fontSize={11}
            />
            <YAxis
              yAxisId="left"
              stroke="var(--color-text-secondary)"
              domain={yDomain}
              scale={yAxisScale}
              tickFormatter={fmtr}
              fontSize={11}
            />
            {lines.some(line => line.yAxisId === 'right') && (
              <YAxis
                yAxisId="right"
                orientation="right"
                stroke="var(--color-text-secondary)"
                domain={rightYAxisDomain}
                tickFormatter={rightYAxisUnit ? defaultValueFormatter(rightYAxisUnit) : fmtr}
                fontSize={11}
              />
            )}
            <Tooltip
              content={<ChartTooltip formatter={fmtr} hiddenSeries={hiddenSeries} />}
              cursor={<ChartCursor />}
            />
            <Legend
              wrapperStyle={{ color: 'var(--color-text)', fontSize: 'var(--text-xs)' }}
              // Rendered from `lines`, not from the chart's graphical items.
              // Each series contributes a Line *and* a decorative Area, so an
              // inferred legend lists every series twice.
              content={() => (
                <ul className="metric-chart-legend">
                  {lines.map(line => {
                    const isHidden = hiddenSeries.has(line.dataKey);
                    return (
                      <li key={line.dataKey}>
                        <button
                          type="button"
                          className="metric-chart-legend__item"
                          aria-pressed={!isHidden}
							onClick={() => { toggleSeries(line.dataKey); }}
                          style={{ opacity: isHidden ? 0.35 : 1 }}
                        >
                          <span
                            className="metric-chart-legend__swatch"
                            style={{ background: line.color }}
                            aria-hidden="true"
                          />
                          <span style={{ textDecoration: isHidden ? 'line-through' : 'none' }}>
                            {line.name}
                          </span>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              )}
            />
            {lines.map(line => {
              const isHidden = hiddenSeries.has(line.dataKey);
              return (
                <Area
                  key={`area-${line.dataKey}`}
                  type={line.type ?? 'monotone'}
                  dataKey={line.dataKey}
                  yAxisId={line.yAxisId ?? 'left'}
                  fill={`url(#${gradientIds[line.dataKey]})`}
                  stroke="none"
                  isAnimationActive={false}
                  hide={isHidden}
                  // The gradient fill is decoration for the line that shares
                  // its dataKey. Without this it registers a second, unnamed
                  // legend entry per series and doubles the legend.
                  legendType="none"
                />
              );
            })}
            {lines.map(line => {
              const isHidden = hiddenSeries.has(line.dataKey);
              return (
                <Line
                  key={line.dataKey}
                  type={line.type ?? 'monotone'}
                  dataKey={line.dataKey}
                  yAxisId={line.yAxisId ?? 'left'}
                  name={line.name}
                  stroke={line.color}
                  strokeWidth={line.strokeWidth ?? 2}
                  dot={false}
                  activeDot={{
                    r: 4,
                    strokeWidth: 2,
                    stroke: 'var(--chart-dot-stroke)',
                    fill: line.color
                  }}
                  isAnimationActive={false}
                  hide={isHidden}
                />
              );
            })}
          </ComposedChart>
        </ResponsiveContainer>
      ) : status === 'error' ? (
        <ChartErrorState height={height} errorMessage={errorMessage} />
      ) : status === 'loading' ? (
        <ChartSkeleton height={height} />
      ) : (
        <ChartEmptyState height={height} seriesLabel={seriesLabel} />
      )}
    </div>
  );
};

// ── Barrel Re-exports ───────────────────────────────────────────────────────

export { CpuDetailView } from './CpuDetailView';
export type { CpuDetailViewProps } from './CpuDetailView';
export { MemoryDetailView } from './MemoryDetailView';
export type { MemoryDetailViewProps } from './MemoryDetailView';
export { NetworkDetailView } from './NetworkDetailView';
export type { NetworkDetailViewProps } from './NetworkDetailView';
export { DiskDetailView } from './DiskDetailView';
export type { DiskDetailViewProps } from './DiskDetailView';
export { GpuDetailView } from './GpuDetailView';
export type { GpuDetailViewProps } from './GpuDetailView';
