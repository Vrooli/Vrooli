/**
 * @libraryId react-component-library:SampleSeriesProbe
 * @displayName SampleSeriesProbe
 * @description Ingested from command-center:ui/src/components/SampleSeries.tsx
 * @version 0.1.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

export interface SampleSeriesProps {
  series: number[];
  /** An authored series draws dashed with hollow points; it is a drawing, not a trend. */
  illustrative: boolean;
  className?: string;
}

const styles = `
  [data-rcl-sample-series] { display: block; margin: 0.35em 0 0.3em; color: var(--color-muted-foreground, #94a3b8); }
  [data-rcl-sample-series="illustrative"] { color: color-mix(in srgb, var(--color-foreground, #e8ecf3) 55%, var(--color-provenance-gap, #b7a6ff)); }
`;

/** A six-point series beside a figure. Authored series are stamped illustrative and drawn dashed so nothing downstream mistakes them for a trend. */
export const SampleSeries = withClassName(function SampleSeries({ series, illustrative, className }: SampleSeriesProps) {
  if (series.length < 2) return null;
  const width = 120;
  const height = 28;
  const min = Math.min(...series);
  const max = Math.max(...series);
  const span = max - min || 1;
  const points = series.map((value, i) => [4 + (i / (series.length - 1)) * (width - 8), height - 4 - ((value - min) / span) * (height - 8)] as const);
  const path = points.map(([x, y], i) => `${i === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`).join(" ");
  return (
    <>
      <StyleSheet name="sample-series-1" css={styles} />
      <svg data-rcl-sample-series={illustrative ? "illustrative" : "measured"} data-illustrative={illustrative || undefined} className={className} viewBox={`0 0 ${width} ${height}`} width={width} height={height} aria-hidden="true">
        <path d={path} fill="none" stroke="currentColor" strokeWidth="1.4" strokeDasharray={illustrative ? "3 4" : undefined} strokeLinecap="round" />
        {points.map(([x, y], i) => (
          <circle key={i} cx={x} cy={y} r="2" fill={illustrative ? "none" : "currentColor"} stroke="currentColor" strokeWidth="1" />
        ))}
      </svg>
    </>
  );
});