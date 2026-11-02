import { useEffect, useId, useMemo, useState, useCallback, useRef } from 'react';
import { formatTime } from '../../../shared/utils/formatters';
import type { ChartDataPoint } from '../../../types';

export interface ChartInspection {
  selected: { timestamp: string; raw: number } | null;
  frozen: { data: ChartDataPoint[]; start?: string; end?: string; interval?: number; label?: string } | null;
  zoom?: { start: string; end: string } | null;
}

interface MetricSparklineProps {
  data?: ChartDataPoint[];
  color?: string;
  height?: number;
  className?: string;
  valueDomain?: [number, number];
  threshold?: number;
  unit?: string;
  /** Decimal places for read-out values. Counts are integers; rates are not. */
  precision?: number;
  windowLabel?: string;
  ariaLabel?: string;
  sampleIntervalSeconds?: number;
  windowStart?: string;
  windowEnd?: string;
  emptyMessage?: string;
  inspectionStore?: Map<string, ChartInspection>;
  inspectionKey?: string;
}

interface ComputedPoint {
  x: number;
  y: number;
  raw: number;
  timestamp: string;
  segment: number;
}

/** Vertical inset of the plot band inside the SVG, in viewBox units. */
const PLOT_PADDING = 4;

/** Fractions of the plot band that carry a graticule division. */
const GRID_DIVISIONS = [0, 0.25, 0.5, 0.75, 1] as const;

/** Width of the SVG's coordinate system, in viewBox units. */
const VIEWBOX_WIDTH = 100;

/**
 * A near-constant series has almost no range of its own. Scaling it to the
 * plot height would magnify a rounding wobble into a mountain range and print
 * two near-identical rail bounds as though they framed a real span, so an
 * auto-derived domain is widened to at least this much before it is used.
 */
const MIN_DOMAIN_SPAN_RATIO = 0.1;
const MIN_DOMAIN_SPAN_ABSOLUTE = 1;

/**
 * Rail figures: enough precision to be useful, never enough to be noise. The
 * rail annotates the extent of the plot, so its precision follows the span it
 * describes — a 0.6 MB/s span printed as "24 / 23" tells the reader nothing.
 */
function formatRailValue(value: number, span: number): string {
  if (!Number.isFinite(value)) return '—';
  if (span >= 20) return value.toFixed(0);
  if (span >= 2) return value.toFixed(1);
  return value.toFixed(2);
}

export const MetricSparkline = ({
  data = [],
  color = 'var(--color-primary)',
  height = 48,
  className,
  valueDomain,
  threshold,
  unit,
  precision = 1,
  windowLabel,
  ariaLabel,
  sampleIntervalSeconds,
  windowStart,
  windowEnd,
  emptyMessage = 'No observations in this window',
  inspectionStore,
  inspectionKey
}: MetricSparklineProps) => {
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [plotFocused, setPlotFocused] = useState(false);
  const [tableOpen, setTableOpen] = useState(false);
  const [zoom, setZoom] = useState<{ start: string; end: string } | null>(() => inspectionKey ? inspectionStore?.get(inspectionKey)?.zoom ?? null : null);
  const tableSnapshot = useRef<ComputedPoint[]>([]);
  const [selectedSample, setSelectedSample] = useState<ChartInspection['selected']>(() => inspectionKey ? inspectionStore?.get(inspectionKey)?.selected ?? null : null);
  const [frozenView, setFrozenView] = useState<ChartInspection['frozen']>(() => inspectionKey ? inspectionStore?.get(inspectionKey)?.frozen ?? null : null);
  useEffect(() => {
    if (inspectionStore && inspectionKey) {
      if (selectedSample && frozenView) inspectionStore.set(inspectionKey, { selected: selectedSample, frozen: frozenView, zoom });
      else inspectionStore.delete(inspectionKey);
    }
  }, [inspectionStore, inspectionKey, selectedSample, frozenView, zoom]);
  const plottedData = frozenView?.data ?? data;
  const plottedStart = zoom?.start ?? (frozenView ? frozenView.start : windowStart);
  const plottedEnd = zoom?.end ?? (frozenView ? frozenView.end : windowEnd);
  const plottedInterval = frozenView ? frozenView.interval : sampleIntervalSeconds;
  const plottedLabel = frozenView ? frozenView.label : windowLabel;
  const selectedTime = selectedSample?.timestamp ?? null;
  const svgRef = useRef<SVGSVGElement>(null);
  const gradientId = useId().replace(/:/g, '-');
  const width = VIEWBOX_WIDTH;

  // The resolved vertical domain is needed twice — to place the samples and to
  // label the rail — so it is returned alongside them rather than recomputed.
  const { points, vertices, domainMin, domainMax } = useMemo(() => {
    const empty = {
      points: [] as ComputedPoint[],
      vertices: [] as ComputedPoint[],
      domainMin: 0,
      domainMax: 1
    };
    if (!plottedData.length) {
      return empty;
    }

    const startBound = plottedStart ? Date.parse(plottedStart) : NaN;
    const endBound = plottedEnd ? Date.parse(plottedEnd) : NaN;
    const hasBounds = Number.isFinite(startBound) && Number.isFinite(endBound) && endBound > startBound;
    const valid = plottedData.filter(point => Number.isFinite(point.value) && Number.isFinite(Date.parse(point.timestamp)) && (!hasBounds || (Date.parse(point.timestamp) >= startBound && Date.parse(point.timestamp) <= endBound)))
      .sort((a, b) => Date.parse(a.timestamp) - Date.parse(b.timestamp));
    if (!valid.length) return empty;
    const values = valid.map(point => point.value);
    let min = Math.min(...values);
    let max = Math.max(...values);

    if (valueDomain) {
      min = valueDomain[0];
      max = valueDomain[1];
    } else {
      if (!Number.isFinite(min)) {
        min = 0;
      }
      if (!Number.isFinite(max)) {
        max = min + 1;
      }
      // An explicit domain is the caller's contract and is never widened; an
      // auto-derived one is only ever a description of this window's data, so
      // a near-constant window gets a floor rather than a magnified wobble.
      const observedSpan = max - min;
      const magnitude = Math.max(Math.abs(min), Math.abs(max));
      const minimumSpan = Math.max(magnitude * MIN_DOMAIN_SPAN_RATIO, MIN_DOMAIN_SPAN_ABSOLUTE);
      if (observedSpan < minimumSpan) {
        const centre = (min + max) / 2;
        min = centre - minimumSpan / 2;
        max = centre + minimumSpan / 2;
      }
    }

    if (!Number.isFinite(min)) {
      min = 0;
    }
    if (!Number.isFinite(max)) {
      max = min + 1;
    }
    if (max === min) {
      max = min + 1;
    }

    const plotHeight = height - PLOT_PADDING * 2;
    const scaleY = (value: number) => {
      const clamped = Math.max(min, Math.min(max, value));
      const normalized = Number.isFinite(clamped) ? (clamped - min) / (max - min) : 0;
      return (height - PLOT_PADDING) - normalized * plotHeight;
    };

    const firstTime = hasBounds ? startBound : Date.parse(valid[0]!.timestamp);
    const lastTime = hasBounds ? endBound : Date.parse(valid[valid.length - 1]!.timestamp);
    const deltas = valid.slice(1).map((point, index) => Date.parse(point.timestamp) - Date.parse(valid[index]!.timestamp)).filter(delta => delta > 0).sort((a, b) => a - b);
    const typicalMs = plottedInterval ? plottedInterval * 1000 : deltas.length >= 2 ? deltas[Math.floor(deltas.length / 2)]! : 0;
    let segment = 0;
    const computed = valid.map((point, idx) => {
      const time = Date.parse(point.timestamp);
      const previous = idx > 0 ? Date.parse(valid[idx - 1]!.timestamp) : time;
      if (point.gapBefore || (typicalMs && time - previous > typicalMs * 1.5)) segment += 1;
      return {
        x: lastTime === firstTime ? width : ((time - firstTime) / (lastTime - firstTime)) * width,
        y: scaleY(point.value), raw: point.value, timestamp: point.timestamp, segment
      };
    });

    // Draw every measured observation at its actual timestamp. Gaps remain
    // separate segments; no client-side sampling hides intermediate values.
    return {
      points: computed,
      vertices: computed,
      domainMin: min,
      domainMax: max
    };
  }, [plottedData, height, valueDomain, plottedInterval, plottedStart, plottedEnd]);

  const handleMouseMove = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    const svg = svgRef.current;
    if (!svg || vertices.length === 0) return;
    const rect = svg.getBoundingClientRect();
    const mouseX = ((e.clientX - rect.left) / rect.width) * VIEWBOX_WIDTH;

    // Find the nearest rendered vertex — which is a real sample, so the
    // tooltip always reports a value the host actually reported.
    const firstVertex = vertices[0];
    if (!firstVertex) {
      return;
    }
    let nearest = 0;
    let minDist = Math.abs(firstVertex.x - mouseX);
    for (let i = 1; i < vertices.length; i++) {
      const vertex = vertices[i];
      if (!vertex) {
        continue;
      }
      const dist = Math.abs(vertex.x - mouseX);
      if (dist < minDist) {
        minDist = dist;
        nearest = i;
      }
    }
    setHoverIndex(nearest);
  }, [vertices]);

  const handleMouseLeave = useCallback(() => {
    setHoverIndex(null);
  }, []);

  if (!points.length) {
    // "No window yet" is a different claim from "this metric is unavailable"
    // (which the CARD renders, with the collector's reason). Neither invents a
    // value, and neither draws a chart.
    return (
      <div
        className={`readout-trace--empty${className ? ` ${className}` : ''}`}
        style={{ height }}
        data-testid="sparkline-empty"
      >
        <span className="readout-trace__empty-label">{emptyMessage}</span>
        <span className="readout-trace__empty-rule" aria-hidden="true" />
      </div>
    );
  }

  const latest = points[points.length - 1] ?? points[0];
  if (!latest) {
    return null;
  }
  const firstPoint = points[0];
  const lastPoint = points[points.length - 1];
  if (!firstPoint || !lastPoint) {
    return null;
  }

  const formatValue = (value: number) => value.toFixed(precision);
  const withUnit = (text: string) => (unit ? `${text}${unit}` : text);

  // Straight segments connect observations; gaps and single observations do not
  // imply values between samples or outside the captured coverage.
  const groups = vertices.reduce<ComputedPoint[][]>((all, point) => {
    const current = all[all.length - 1];
    if (!current || current[0]?.segment !== point.segment) all.push([point]);
    else current.push(point);
    return all;
  }, []);
  const curvePath = groups.map(group => group.map((point, index) =>
    `${index ? 'L' : 'M'}${point.x.toFixed(2)},${point.y.toFixed(2)}`).join('')).join('');
  const areaPath = groups.filter(group => group.length > 1).map(group => {
    const start = group[0]!, end = group[group.length - 1]!;
    return group.map((point, index) => `${index ? 'L' : 'M'}${point.x.toFixed(2)},${point.y.toFixed(2)}`).join('') +
      `L${end.x.toFixed(2)},${height}L${start.x.toFixed(2)},${height}Z`;
  }).join('');

  const plotHeight = height - PLOT_PADDING * 2;
  const yForFraction = (fraction: number) => (height - PLOT_PADDING) - fraction * plotHeight;

  // The graticule. This is what makes the panel read as an instrument rather
  // than a coloured slab, so it renders under everything else and never moves.
  const graticule = GRID_DIVISIONS.map(fraction => (
    <line
      key={`grid-${String(fraction)}`}
      x1={0}
      x2={width}
      y1={yForFraction(fraction)}
      y2={yForFraction(fraction)}
      stroke={fraction === 0 || fraction === 0.5 ? 'var(--chart-grid-strong)' : 'var(--chart-grid)'}
      strokeWidth={1}
      vectorEffect="non-scaling-stroke"
      shapeRendering="crispEdges"
    />
  ));

  let thresholdLine = null;
  if (typeof threshold === 'number') {
    const clamped = Math.max(domainMin, Math.min(domainMax, threshold));
    const normalized = (clamped - domainMin) / (domainMax - domainMin);
    thresholdLine = (
      <line
        x1={0}
        x2={width}
        y1={yForFraction(normalized)}
        y2={yForFraction(normalized)}
        stroke="var(--color-warning)"
        strokeDasharray="3 3"
        strokeWidth={1}
        vectorEffect="non-scaling-stroke"
        opacity={0.6}
      />
    );
  }

  const tooltip = `Latest ${withUnit(formatValue(latest.raw))} at ${formatTime(latest.timestamp)}`;

  const selectedPoint = selectedTime ? points.find(point => point.timestamp === selectedTime) : undefined;
  const hoverPoint = selectedPoint ?? (hoverIndex !== null ? vertices[hoverIndex] : plotFocused ? latest : null);

  // Only a short unit rides along on the rail; a long one ("connections/s")
  // would out-measure the figure it annotates.
  const trimmedUnit = unit?.trim() ?? '';
  const railUnit = trimmedUnit.length > 0 && trimmedUnit.length <= 2 ? trimmedUnit : '';

  // The line ends at the final sample. This
  // marker is what the reader ties to the card's headline figure, so it is
  // pinned to the true latest reading rather than to anything derived.
  const endpoint = latest;

  const asPercent = (x: number) => `${((x / width) * 100).toFixed(4)}%`;

  return (
    <div className={className} data-sm-style="sm-style-6344dc8492">
      <div className="readout-trace__summary" aria-hidden="true">
        {hoverPoint && <><span>{withUnit(formatValue(hoverPoint.raw))}</span> · <span>{formatTime(hoverPoint.timestamp)}</span></>}
      </div>
      <div className="readout-trace">
        <div className="readout-trace__plot">
          <svg
            ref={svgRef}
            viewBox={`0 0 ${width} ${height}`}
            preserveAspectRatio="none"
            style={{ width: '100%', height, cursor: 'crosshair', display: 'block' }}
            className="metric-sparkline-chart"
            role="img"
            aria-label={ariaLabel ?? tooltip}
            aria-describedby={`${gradientId}-instructions`}
            data-render-mode="line"
            tabIndex={0}
            onFocus={() => { setPlotFocused(true); }}
            onBlur={() => { setPlotFocused(false); setHoverIndex(null); }}
            onClick={event => {
              event.stopPropagation();
              const rect = event.currentTarget.getBoundingClientRect();
              if (!rect.width) return;
              const x = ((event.clientX - rect.left) / rect.width) * width;
              const nearest = points.reduce((best, point) => Math.abs(point.x - x) < Math.abs(best.x - x) ? point : best, points[0]!);
              setFrozenView(previous => previous ?? { data: [...data], start: windowStart, end: windowEnd, interval: sampleIntervalSeconds, label: windowLabel });
              setSelectedSample({ timestamp: nearest.timestamp, raw: nearest.raw });
            }}
            onKeyDown={event => {
              if (event.key === 'Escape') { setSelectedSample(null); setFrozenView(null); setZoom(null); return; }
              if (!['ArrowLeft', 'ArrowRight', 'PageUp', 'PageDown'].includes(event.key)) return;
              event.preventDefault(); event.stopPropagation();
              const index = selectedTime ? points.findIndex(point => point.timestamp === selectedTime) : points.length - 1;
              let next = Math.max(0, Math.min(points.length - 1, index + (event.key === 'ArrowLeft' ? -1 : 1)));
              if (event.key === 'PageUp' || event.key === 'PageDown') {
                const direction = event.key === 'PageUp' ? -1 : 1;
                const currentSegment = points[index]?.segment;
                next = index;
                while (next + direction >= 0 && next + direction < points.length && points[next + direction]?.segment === currentSegment) next += direction;
                if (next + direction >= 0 && next + direction < points.length) next += direction;
              }
              setFrozenView(previous => previous ?? { data: [...data], start: windowStart, end: windowEnd, interval: sampleIntervalSeconds, label: windowLabel });
              setSelectedSample({ timestamp: points[next]!.timestamp, raw: points[next]!.raw });
            }}
            onMouseMove={handleMouseMove}
            onMouseLeave={handleMouseLeave}
          >
            <title>{tooltip}</title>
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={color} stopOpacity="var(--chart-gradient-start-opacity, 0.25)" />
                <stop offset="100%" stopColor={color} stopOpacity="var(--chart-gradient-end-opacity, 0)" />
              </linearGradient>
            </defs>
            <g data-testid="sparkline-graticule" aria-hidden="true">
              {graticule}
              {[0, 0.25, 0.5, 0.75, 1].map(fraction => (
                <line key={`time-grid-${fraction}`} x1={fraction * width} x2={fraction * width}
                  y1={PLOT_PADDING} y2={height - PLOT_PADDING} stroke="var(--chart-grid)"
                  strokeWidth={1} vectorEffect="non-scaling-stroke" shapeRendering="crispEdges" />
              ))}
            </g>
            <path
              d={areaPath}
              fill={`url(#${gradientId})`}
            />
            {/* The viewBox is 100 units wide and stretched to the card's width,
                so a plain stroke-width renders fat horizontally and thin
                vertically. non-scaling-stroke keeps the datum a true, uniform
                weight at every card size. */}
            <path
              d={curvePath}
              data-testid="sparkline-line"
              fill="none"
              stroke={color}
              strokeWidth={1.6}
              strokeLinecap="round"
              strokeLinejoin="round"
              vectorEffect="non-scaling-stroke"
            />
            {groups.filter(group => group.length === 1).map(group => (
              <line key={`${group[0]!.timestamp}-${group[0]!.segment}`} x1={group[0]!.x} x2={group[0]!.x} y1={group[0]!.y - 2} y2={group[0]!.y + 2} stroke={color} strokeWidth={2} vectorEffect="non-scaling-stroke" />
            ))}
            {thresholdLine}
            {/* Hover hairline */}
            {hoverPoint && (
              <line
                x1={hoverPoint.x}
                x2={hoverPoint.x}
                y1={0}
                y2={height}
                stroke="var(--chart-cursor-color)"
                strokeWidth={1}
                vectorEffect="non-scaling-stroke"
              />
            )}
          </svg>

          {/* Dots live in HTML rather than the SVG: the horizontal stretch
              would turn an SVG circle into an ellipse, so there is deliberately
              no <circle> in the chart. */}
          {!hoverPoint && (
            <span
              className="readout-trace__halo"
              data-testid="sparkline-endpoint-halo"
              aria-hidden="true"
              style={{ left: asPercent(endpoint.x), top: endpoint.y }}
            />
          )}
          <span
            className="readout-trace__dot"
            data-testid="sparkline-endpoint"
            aria-hidden="true"
            style={{
              left: asPercent((hoverPoint ?? endpoint).x),
              top: (hoverPoint ?? endpoint).y,
              background: color
            }}
          />


        </div>

        {/* Vertical extent, stated. Without it the reader cannot tell whether
            the trace spans two points or two hundred. */}
        <div className="readout-trace__rail" data-testid="sparkline-rail" aria-hidden="true">
          <span>{`${formatRailValue(domainMax, domainMax - domainMin)}${railUnit}`}</span>
          <span>{`${formatRailValue(domainMin, domainMax - domainMin)}${railUnit}`}</span>
        </div>
      </div>

      <span id={`${gradientId}-instructions`} className="sr-only">Tap the plot or use left and right arrows to select samples; Page Up and Page Down move between segments. Select a sample to zoom, and press Escape to return to latest.</span>
      <span className="sr-only" aria-live="polite" aria-atomic="true" data-testid="sparkline-latest-announcement">{plotFocused && !selectedSample ? `Latest observation: ${withUnit(formatValue(latest.raw))} at ${new Date(latest.timestamp).toLocaleString()}` : ''}</span>
      {plottedStart && plottedEnd && Number.isFinite(Date.parse(plottedStart)) && Number.isFinite(Date.parse(plottedEnd)) && (
        <div className="readout-trace__axis" aria-label={`Window ${new Date(plottedStart!).toLocaleString()} to ${new Date(plottedEnd!).toLocaleString()}`}>
          <span>{formatTime(plottedStart)}</span><span>{formatTime((Date.parse(plottedStart) + Date.parse(plottedEnd)) / 2)}</span><span>{formatTime(plottedEnd)}</span>
        </div>
      )}
      <details onClick={event => { event.stopPropagation(); }} onToggle={event => { const open = event.currentTarget.open; if (open) tableSnapshot.current = points; setTableOpen(open); }}>
        <summary>View observed samples</summary>
        {tableOpen && <table>
          <caption>{ariaLabel ?? 'History'} — observed samples</caption>
          <thead><tr><th scope="col">Time</th><th scope="col">Value</th></tr></thead>
          <tbody>{tableSnapshot.current.map((point, index) => <tr key={`${point.timestamp}-${index}`}><td>{new Date(point.timestamp).toLocaleString()}</td><td>{withUnit(formatValue(point.raw))}</td></tr>)}</tbody>
        </table>}
      </details>
      {selectedSample && <div className="readout-caption" role="status">
        Fixed selection: {withUnit(formatValue(selectedSample.raw))} at {new Date(selectedSample.timestamp).toLocaleString()}
        {!selectedPoint ? ' — no longer in the displayed samples' : ''}
      </div>}
      {selectedTime && <button type="button" onClick={event => { event.stopPropagation(); setSelectedSample(null); setFrozenView(null); setZoom(null); }}>Return to latest</button>}
      {selectedTime && !zoom && <button className="readout-detail__btn" type="button" onClick={event => {
        event.stopPropagation();
        const start = Date.parse(frozenView?.start ?? windowStart ?? points[0]!.timestamp);
        const end = Date.parse(frozenView?.end ?? windowEnd ?? points[points.length - 1]!.timestamp);
        const selected = Date.parse(selectedTime);
        if (!Number.isFinite(start) || !Number.isFinite(end) || !Number.isFinite(selected) || end <= start) return;
        const span = Math.min(15 * 60_000, (end - start) / 4);
        const from = Math.max(start, Math.min(selected - span / 2, end - span));
        setZoom({ start: new Date(from).toISOString(), end: new Date(from + span).toISOString() });
        svgRef.current?.focus();
      }}>Zoom around selected sample</button>}
      {zoom && <div className="readout-caption">Chart zoom: {new Date(zoom.start).toLocaleString()} – {new Date(zoom.end).toLocaleString()} ({Intl.DateTimeFormat().resolvedOptions().timeZone}) <button className="readout-detail__btn" type="button" onClick={event => { event.stopPropagation(); setZoom(null); svgRef.current?.focus(); }}>Reset chart zoom</button></div>}
      <div className="readout-trace__window">
        {plottedLabel ? `${plottedLabel} · observed` : 'Observed'} {new Date(firstPoint.timestamp).toLocaleString()} – {new Date(lastPoint.timestamp).toLocaleString()} ({Intl.DateTimeFormat().resolvedOptions().timeZone})
        {groups.length > 1 ? ` · ${groups.length - 1} gap${groups.length === 2 ? '' : 's'}` : ''}
      </div>
    </div>
  );
};
