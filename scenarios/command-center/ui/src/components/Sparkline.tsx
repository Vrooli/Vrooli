interface SparklineProps {
  series: number[];
  /** An authored series draws dashed with hollow points; it is a drawing, not a trend. */
  illustrative: boolean;
}

export function Sparkline({ series, illustrative }: SparklineProps) {
  if (series.length < 2) return null;
  const width = 120;
  const height = 28;
  const min = Math.min(...series);
  const max = Math.max(...series);
  const span = max - min || 1;
  const points = series.map((value, i) => [4 + (i / (series.length - 1)) * (width - 8), height - 4 - ((value - min) / span) * (height - 8)] as const);
  const path = points.map(([x, y], i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`).join(" ");
  return (
    <svg className="cc-sparkline" viewBox={`0 0 ${width} ${height}`} width={width} height={height} aria-hidden="true" data-illustrative={illustrative || undefined}>
      <path d={path} fill="none" stroke="currentColor" strokeWidth="1.4" strokeDasharray={illustrative ? "3 4" : undefined} strokeLinecap="round" />
      {points.map(([x, y], i) => (
        <circle key={i} cx={x} cy={y} r="2" fill={illustrative ? "none" : "currentColor"} stroke="currentColor" strokeWidth="1" />
      ))}
    </svg>
  );
}
