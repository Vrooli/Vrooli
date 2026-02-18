import { useMemo, useState, useCallback, useRef } from 'react';
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
  windowLabel?: string;
}

interface ComputedPoint {
  x: number;
  y: number;
  raw: number;
  timestamp: string;
}

/** Build a smooth SVG path using monotone cubic (Catmull-Rom → bezier) interpolation. */
function smoothPath(pts: ComputedPoint[]): string {
  if (pts.length === 0) return '';
  if (pts.length === 1) {
    const p = pts[0]!;
    return `M${p.x},${p.y}`;
  }
  if (pts.length === 2) {
    const p0 = pts[0]!;
    const p1 = pts[1]!;
    return `M${p0.x},${p0.y}L${p1.x},${p1.y}`;
  }

  const n = pts.length;
  const dxArr: number[] = [];
  const dyArr: number[] = [];
  const mArr: number[] = [];

  for (let i = 0; i < n - 1; i++) {
    const cur = pts[i]!;
    const next = pts[i + 1]!;
    const segDx = next.x - cur.x;
    const segDy = next.y - cur.y;
    dxArr.push(segDx);
    dyArr.push(segDy);
    mArr.push(segDy / (segDx || 1));
  }

  const slopes: number[] = new Array(n).fill(0);
  slopes[0] = mArr[0]!;
  slopes[n - 1] = mArr[n - 2]!;
  for (let i = 1; i < n - 1; i++) {
    const prev = mArr[i - 1]!;
    const cur = mArr[i]!;
    if (prev * cur <= 0) {
      slopes[i] = 0;
    } else {
      slopes[i] = (prev + cur) / 2;
    }
  }

  const first = pts[0]!;
  let d = `M${first.x.toFixed(2)},${first.y.toFixed(2)}`;
  for (let i = 0; i < n - 1; i++) {
    const cur = pts[i]!;
    const next = pts[i + 1]!;
    const segLen = dxArr[i]! / 3;
    const cp1x = cur.x + segLen;
    const cp1y = cur.y + slopes[i]! * segLen;
    const cp2x = next.x - segLen;
    const cp2y = next.y - slopes[i + 1]! * segLen;
    d += `C${cp1x.toFixed(2)},${cp1y.toFixed(2)},${cp2x.toFixed(2)},${cp2y.toFixed(2)},${next.x.toFixed(2)},${next.y.toFixed(2)}`;
  }
  return d;
}

export const MetricSparkline = ({
  data = [],
  color = 'var(--color-primary)',
  height = 48,
  className,
  valueDomain,
  threshold,
  unit,
  windowLabel
}: MetricSparklineProps) => {
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);
  const width = 100;

  const points = useMemo(() => {
    if (!data.length) {
      return [] as ComputedPoint[];
    }

    const values = data.map(point => point.value);
    let min = Math.min(...values);
    let max = Math.max(...values);

    if (valueDomain) {
      min = valueDomain[0];
      max = valueDomain[1];
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

    const padding = 4;
    const plotHeight = height - padding * 2;

    return data.map((point, idx) => {
      const ratio = data.length > 1 ? idx / (data.length - 1) : 1;
      const clamped = Math.max(min, Math.min(max, point.value));
      const normalized = (clamped - min) / (max - min);
      const x = ratio * width;
      const y = (height - padding) - normalized * plotHeight;
      return { x, y, raw: point.value, timestamp: point.timestamp };
    });
  }, [data, height, valueDomain]);

  const handleMouseMove = useCallback((e: React.MouseEvent<SVGSVGElement>) => {
    const svg = svgRef.current;
    if (!svg || points.length === 0) return;
    const rect = svg.getBoundingClientRect();
    const mouseX = ((e.clientX - rect.left) / rect.width) * width;

    // Find nearest point
    let nearest = 0;
    let minDist = Math.abs(points[0]!.x - mouseX);
    for (let i = 1; i < points.length; i++) {
      const dist = Math.abs(points[i]!.x - mouseX);
      if (dist < minDist) {
        minDist = dist;
        nearest = i;
      }
    }
    setHoverIndex(nearest);
  }, [points]);

  const handleMouseLeave = useCallback(() => {
    setHoverIndex(null);
  }, []);

  if (!points.length) {
    return (
      <div
        className={className}
        style={{
          height,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--color-text-secondary)',
          fontSize: 'var(--text-xs)',
          letterSpacing: '0.08em'
        }}
      >
        Collecting data…
      </div>
    );
  }

  const latest = points[points.length - 1] ?? points[0];
  if (!latest) {
    return null;
  }

  // Build smooth curve path
  const curvePath = points.length === 1
    ? `M0,${latest.y.toFixed(2)}L${width.toFixed(2)},${latest.y.toFixed(2)}`
    : smoothPath(points);

  // Build area path (curve + bottom edge)
  const lastPt = points[points.length - 1]!;
  const firstPt = points[0]!;
  const areaPath = `${curvePath}L${lastPt.x.toFixed(2)},${height}L${firstPt.x.toFixed(2)},${height}Z`;

  // Gradient ID unique per instance
  const gradientId = useMemo(() => `spark-grad-${Math.random().toString(36).slice(2, 8)}`, []);

  let thresholdLine = null;
  if (typeof threshold === 'number') {
    const values = data.map(point => point.value);
    let min = Math.min(...values);
    let max = Math.max(...values);
    if (valueDomain) {
      min = valueDomain[0];
      max = valueDomain[1];
    }
    if (max === min) {
      max = min + 1;
    }
    const clamped = Math.max(min, Math.min(max, threshold));
    const normalized = (clamped - min) / (max - min);
    const padding = 4;
    const plotHeight = height - padding * 2;
    const y = (height - padding) - normalized * plotHeight;
    thresholdLine = (
      <line
        x1={0}
        x2={width}
        y1={y}
        y2={y}
        stroke="var(--color-warning)"
        strokeDasharray="3 3"
        strokeWidth={0.8}
        opacity={0.6}
      />
    );
  }

  const latestValueLabel = unit
    ? `${latest.raw.toFixed(1)}${unit}`
    : latest.raw.toFixed(1);

  const tooltip = `Latest ${latestValueLabel} at ${formatTime(latest.timestamp)}`;

  const hoverPoint = hoverIndex !== null ? points[hoverIndex] : null;

  return (
    <div className={className} style={{ width: '100%', position: 'relative' }}>
      <svg
        ref={svgRef}
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
        style={{ width: '100%', height, cursor: 'crosshair' }}
        className="metric-sparkline-chart"
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
        <path
          d={areaPath}
          fill={`url(#${gradientId})`}
        />
        <path
          d={curvePath}
          fill="none"
          stroke={color}
          strokeWidth={1.8}
          strokeLinecap="round"
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
            strokeWidth={0.8}
          />
        )}
        {/* Hover dot */}
        {hoverPoint && (
          <circle
            cx={hoverPoint.x}
            cy={hoverPoint.y}
            r={3}
            fill={color}
            stroke="var(--chart-dot-stroke)"
            strokeWidth={1.5}
          />
        )}
        {/* Latest point dot (only when not hovering) */}
        {hoverIndex === null && (
          <circle cx={latest.x} cy={latest.y} r={2.4} fill={color} />
        )}
      </svg>
      {/* Hover tooltip label */}
      {hoverPoint && (
        <div
          style={{
            position: 'absolute',
            left: `${(hoverPoint.x / width) * 100}%`,
            top: -4,
            transform: 'translate(-50%, -100%)',
            background: 'var(--chart-tooltip-bg)',
            border: '1px solid var(--chart-tooltip-border)',
            borderRadius: 'var(--radius-sm)',
            padding: '2px 6px',
            fontSize: '0.6rem',
            color: 'var(--color-text-heading)',
            whiteSpace: 'nowrap',
            pointerEvents: 'none',
            boxShadow: 'var(--chart-tooltip-shadow)',
            zIndex: 10
          }}
        >
          <span style={{ fontWeight: 600 }}>
            {unit ? `${hoverPoint.raw.toFixed(1)}${unit}` : hoverPoint.raw.toFixed(1)}
          </span>
          <span style={{ color: 'var(--color-text-muted)', marginLeft: 4 }}>
            {formatTime(hoverPoint.timestamp)}
          </span>
        </div>
      )}
      {windowLabel && (
        <div
          style={{
            marginTop: 'var(--spacing-xs)',
            color: 'var(--color-text-secondary)',
            fontSize: '0.55rem',
            letterSpacing: '0.08em'
          }}
        >
          {windowLabel}
        </div>
      )}
    </div>
  );
};
