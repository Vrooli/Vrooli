import { useMemo } from 'react';
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

export const MetricSparkline = ({
  data = [],
  color = 'var(--color-accent)',
  height = 48,
  className,
  valueDomain,
  threshold,
  unit,
  windowLabel
}: MetricSparklineProps) => {
  const points = useMemo(() => {
    if (!data.length) {
      return [] as Array<{ x: number; y: number; raw: number; timestamp: string }>;
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
    const width = 100;
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

  if (!points.length) {
    return (
      <div
        className={className}
        style={{
          height,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          color: 'var(--color-text-dim)',
          fontSize: 'var(--font-size-xs)',
          letterSpacing: '0.08em'
        }}
      >
        Collecting data…
      </div>
    );
  }

  const width = 100;
  const latest = points[points.length - 1];
  const linePoints = (() => {
    if (points.length === 1) {
      return `0,${latest.y.toFixed(2)} ${width.toFixed(2)},${latest.y.toFixed(2)}`;
    }
    return points.map(point => `${point.x.toFixed(2)},${point.y.toFixed(2)}`).join(' ');
  })();

  const areaPoints = `${linePoints} ${width.toFixed(2)},${height.toFixed(2)} 0,${height.toFixed(2)}`;

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

  const tooltip = `Latest ${latestValueLabel} at ${new Date(latest.timestamp).toLocaleTimeString()}`;

  return (
    <div className={className} style={{ width: '100%' }}>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
        style={{ width: '100%', height }}
        className="metric-sparkline-chart"
      >
        <title>{tooltip}</title>
        <polygon
          points={areaPoints}
          fill={color}
          fillOpacity={0.12}
        />
        <polyline
          points={linePoints}
          fill="none"
          stroke={color}
          strokeWidth={1.8}
          strokeLinecap="round"
        />
        {thresholdLine}
        <circle cx={latest.x} cy={latest.y} r={2.4} fill={color} />
      </svg>
      {windowLabel && (
        <div
          style={{
            marginTop: 'var(--spacing-xs)',
            color: 'var(--color-text-dim)',
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
