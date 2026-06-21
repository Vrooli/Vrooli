/**
 * Dependency-free SVG sparkline. Renders a series of numeric values (oldest →
 * newest) as a small line chart. Degrades to an em-dash when there is nothing
 * to plot. Presentation only — all formatting/labels live in the caller.
 */
export function Sparkline({
  values,
  label,
  width = 220,
  height = 48,
  testId,
}: {
  values: number[];
  label: string;
  width?: number;
  height?: number;
  testId?: string;
}) {
  const points = values.filter((v) => Number.isFinite(v));
  if (points.length === 0) {
    return (
      <span data-testid={testId} className="text-app-muted-foreground">
        —
      </span>
    );
  }
  if (points.length === 1) {
    // A single sample renders as a flat midline so the trend area is never blank.
    points.push(points[0] as number);
  }

  const max = Math.max(...points);
  const min = Math.min(...points);
  const span = max - min || 1;
  const stepX = width / (points.length - 1);

  const coords = points.map((v, i) => {
    const x = i * stepX;
    // y inverted: higher value -> higher on screen (smaller y).
    const y = height - ((v - min) / span) * (height - 4) - 2;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });

  return (
    <svg
      data-testid={testId}
      role="img"
      aria-label={label}
      viewBox={`0 0 ${width} ${height}`}
      width={width}
      height={height}
      className="overflow-visible"
      preserveAspectRatio="none"
    >
      <polyline
        points={coords.join(" ")}
        fill="none"
        stroke="currentColor"
        strokeWidth={1.5}
        className="text-app-primary"
        vectorEffect="non-scaling-stroke"
      />
      <circle
        cx={(points.length - 1) * stepX}
        cy={height - (((points.at(-1) as number) - min) / span) * (height - 4) - 2}
        r={2.5}
        className="fill-app-primary"
      />
    </svg>
  );
}
