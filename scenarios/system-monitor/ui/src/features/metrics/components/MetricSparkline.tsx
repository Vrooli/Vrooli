import { useId, useMemo, useState, useCallback, useRef } from 'react';
import { formatTime } from '../../../shared/utils/formatters';
import type { ChartDataPoint } from '../../../types';

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
}

interface ComputedPoint {
  x: number;
  y: number;
  raw: number;
  timestamp: string;
}

/** Vertical inset of the plot band inside the SVG, in viewBox units. */
const PLOT_PADDING = 4;

/** Fractions of the plot band that carry a graticule division. */
const GRID_DIVISIONS = [0, 0.25, 0.5, 0.75, 1] as const;

/** Width of the SVG's coordinate system, in viewBox units. */
const VIEWBOX_WIDTH = 100;

/**
 * Column resolution, expressed in the chart's own coordinate system rather
 * than as a bare vertex count: one vertex per 0.7 viewBox units. On a card
 * around 470px wide that is roughly three device pixels per vertex — fine
 * enough that the downsampling is invisible, coarse enough that the line stops
 * overprinting itself into a solid mass.
 */
const COLUMN_UNITS = 0.7;

/**
 * A near-constant series has almost no range of its own. Scaling it to the
 * plot height would magnify a rounding wobble into a mountain range and print
 * two near-identical rail bounds as though they framed a real span, so an
 * auto-derived domain is widened to at least this much before it is used.
 */
const MIN_DOMAIN_SPAN_RATIO = 0.1;
const MIN_DOMAIN_SPAN_ABSOLUTE = 1;

/** Build a smooth SVG path using monotone cubic (Catmull-Rom → bezier) interpolation. */
function smoothPath(pts: ComputedPoint[]): string {
  if (pts.length === 0) return '';
  if (pts.length === 1) {
    const [p] = pts;
    if (!p) return '';
    return `M${p.x},${p.y}`;
  }
  if (pts.length === 2) {
    const [p0, p1] = pts;
    if (!p0 || !p1) return '';
    return `M${p0.x},${p0.y}L${p1.x},${p1.y}`;
  }

  const n = pts.length;
  const dxArr: number[] = [];
  const dyArr: number[] = [];
  const mArr: number[] = [];

  for (let i = 0; i < n - 1; i++) {
    const cur = pts[i];
    const next = pts[i + 1];
    if (!cur || !next) {
      continue;
    }
    const segDx = next.x - cur.x;
    const segDy = next.y - cur.y;
    dxArr.push(segDx);
    dyArr.push(segDy);
    mArr.push(segDy / (segDx || 1));
  }

  const slopes = Array.from({ length: n }, () => 0);
  slopes[0] = mArr[0] ?? 0;
  slopes[n - 1] = mArr[n - 2] ?? 0;
  for (let i = 1; i < n - 1; i++) {
    const prev = mArr[i - 1] ?? 0;
    const cur = mArr[i] ?? 0;
    if (prev * cur <= 0) {
      slopes[i] = 0;
    } else {
      slopes[i] = (prev + cur) / 2;
    }
  }

  const first = pts[0];
  if (!first) return '';
  let d = `M${first.x.toFixed(2)},${first.y.toFixed(2)}`;
  for (let i = 0; i < n - 1; i++) {
    const cur = pts[i];
    const next = pts[i + 1];
    if (!cur || !next) {
      continue;
    }
    const segLen = (dxArr[i] ?? 0) / 3;
    const cp1x = cur.x + segLen;
    const cp1y = cur.y + (slopes[i] ?? 0) * segLen;
    const cp2x = next.x - segLen;
    const cp2y = next.y - (slopes[i + 1] ?? 0) * segLen;
    d += `C${cp1x.toFixed(2)},${cp1y.toFixed(2)},${cp2x.toFixed(2)},${cp2y.toFixed(2)},${next.x.toFixed(2)},${next.y.toFixed(2)}`;
  }
  return d;
}

/**
 * Largest-Triangle-Three-Buckets downsampling.
 *
 * Every vertex it returns is a REAL sample — LTTB selects, it never averages —
 * so the rendered line states only values the host actually reported, and the
 * card can never show an unlabelled aggregate posing as the series. Within
 * each bucket it keeps the sample forming the largest triangle with the
 * previously kept point and the next bucket's centroid, which is the sample
 * that carries the most of the series' visual shape; peaks and troughs survive
 * where a mean would have flattened them.
 *
 * The first and last samples are always retained, so the trace begins and ends
 * on real readings — the endpoint marker the reader ties to the card's
 * headline figure is therefore the true latest sample, never an aggregate.
 */
function largestTriangleThreeBuckets(points: ComputedPoint[], threshold: number): ComputedPoint[] {
  const n = points.length;
  if (threshold >= n || threshold < 3) {
    return points;
  }

  const first = points[0];
  const last = points[n - 1];
  if (!first || !last) {
    return points;
  }

  const sampled: ComputedPoint[] = [first];
  // Width of one bucket, excluding the two endpoints that are always kept.
  const bucketSize = (n - 2) / (threshold - 2);
  let anchor = first;

  for (let i = 0; i < threshold - 2; i++) {
    // Centroid of the NEXT bucket — the third corner of the triangle.
    const nextStart = Math.floor((i + 1) * bucketSize) + 1;
    const nextEnd = Math.min(Math.floor((i + 2) * bucketSize) + 1, n);
    let avgX = 0;
    let avgY = 0;
    let avgCount = 0;
    for (let j = nextStart; j < nextEnd; j++) {
      const point = points[j];
      if (!point) continue;
      avgX += point.x;
      avgY += point.y;
      avgCount += 1;
    }
    if (avgCount > 0) {
      avgX /= avgCount;
      avgY /= avgCount;
    } else {
      avgX = last.x;
      avgY = last.y;
    }

    const rangeStart = Math.floor(i * bucketSize) + 1;
    const rangeEnd = Math.min(Math.floor((i + 1) * bucketSize) + 1, n - 1);
    let best = points[rangeStart] ?? anchor;
    let bestArea = -1;
    for (let j = rangeStart; j < rangeEnd; j++) {
      const point = points[j];
      if (!point) continue;
      const area = Math.abs(
        (anchor.x - avgX) * (point.y - anchor.y) - (anchor.x - point.x) * (avgY - anchor.y)
      ) / 2;
      if (area > bestArea) {
        bestArea = area;
        best = point;
      }
    }
    sampled.push(best);
    anchor = best;
  }

  sampled.push(last);
  return sampled;
}

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
  ariaLabel
}: MetricSparklineProps) => {
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const gradientId = useId().replace(/:/g, '-');
  const width = VIEWBOX_WIDTH;

  // The resolved vertical domain is needed twice — to place the samples and to
  // label the rail — so it is returned alongside them rather than recomputed.
  const { points, vertices, isDownsampled, domainMin, domainMax } = useMemo(() => {
    const empty = {
      points: [] as ComputedPoint[],
      vertices: [] as ComputedPoint[],
      isDownsampled: false,
      domainMin: 0,
      domainMax: 1
    };
    if (!data.length) {
      return empty;
    }

    const values = data.map(point => point.value);
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

    const computed = data.map((point, idx) => {
      const ratio = data.length > 1 ? idx / (data.length - 1) : 1;
      return {
        x: ratio * width,
        y: scaleY(point.value),
        raw: point.value,
        timestamp: point.timestamp
      };
    });

    // A 60-minute window at the poll interval is hundreds of samples drawn
    // across a few hundred pixels. One vertex per sample makes an opaque
    // stroke fill the whole band and the chart stops being a chart. LTTB caps
    // the vertex count while keeping every vertex a real reading.
    const maxVertices = Math.max(3, Math.round(width / COLUMN_UNITS));
    const downsampled = computed.length > maxVertices;

    return {
      points: computed,
      vertices: downsampled ? largestTriangleThreeBuckets(computed, maxVertices) : computed,
      isDownsampled: downsampled,
      domainMin: min,
      domainMax: max
    };
  }, [data, height, valueDomain]);

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
        <span className="readout-trace__empty-label">Collecting data…</span>
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

  const curvePath = vertices.length === 1
    ? `M0,${latest.y.toFixed(2)}L${width.toFixed(2)},${latest.y.toFixed(2)}`
    : smoothPath(vertices);

  // The fill is the line's body, not a bar — it fades to nothing before it
  // reaches the baseline, so the stroke stays the datum.
  const areaPath = `${curvePath}L${lastPoint.x.toFixed(2)},${height}L${firstPoint.x.toFixed(2)},${height}Z`;

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

  const hoverPoint = hoverIndex !== null ? vertices[hoverIndex] : null;

  // Only a short unit rides along on the rail; a long one ("connections/s")
  // would out-measure the figure it annotates.
  const trimmedUnit = unit?.trim() ?? '';
  const railUnit = trimmedUnit.length > 0 && trimmedUnit.length <= 2 ? trimmedUnit : '';

  // LTTB always retains the final sample, so the line ends exactly here. This
  // marker is what the reader ties to the card's headline figure, so it is
  // pinned to the true latest reading rather than to anything derived.
  const endpoint = latest;

  const asPercent = (x: number) => `${((x / width) * 100).toFixed(4)}%`;

  return (
    <div className={className} data-sm-style="sm-style-6344dc8492">
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
            data-render-mode={isDownsampled ? 'lttb' : 'line'}
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
            <g data-testid="sparkline-graticule">{graticule}</g>
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
          {hoverIndex === null && (
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

          {/* Hover tooltip label */}
          {hoverPoint && (
            <div
              className="readout-trace__tooltip"
              style={{ left: asPercent(hoverPoint.x) }}
            >
              <span data-sm-style="sm-style-2d5d655385">
                {withUnit(formatValue(hoverPoint.raw))}
              </span>
              <span data-sm-style="sm-style-89cc5f7662">
                {formatTime(hoverPoint.timestamp)}
              </span>
            </div>
          )}
        </div>

        {/* Vertical extent, stated. Without it the reader cannot tell whether
            the trace spans two points or two hundred. */}
        <div className="readout-trace__rail" data-testid="sparkline-rail" aria-hidden="true">
          <span>{`${formatRailValue(domainMax, domainMax - domainMin)}${railUnit}`}</span>
          <span>{`${formatRailValue(domainMin, domainMax - domainMin)}${railUnit}`}</span>
        </div>
      </div>

      {windowLabel && (
        <div className="readout-trace__window">
          {windowLabel}
        </div>
      )}
    </div>
  );
};
