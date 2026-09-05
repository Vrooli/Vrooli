import { SampleSeries } from "./SampleSeries";

const ground = {
  display: "grid",
  gap: "var(--space-md, 24px)",
  padding: "var(--space-lg, 32px)",
  background: "var(--color-background, #05070e)",
  color: "var(--color-foreground, #e8ecf3)",
  font: "var(--text-caption, 600 0.75rem/1.3 system-ui)",
};

/** An authored series: dashed stroke, hollow points, a drawing of a shape and never a trend. */
export function Illustrative() {
  return (
    <div style={ground}>
      <span>hand-authored, mid-scale, reviewed 2026-09-01</span>
      <SampleSeries series={[8100, 8800, 9600, 10200, 11300, 12400]} illustrative />
    </div>
  );
}

/** A measured series: solid stroke, filled points. */
export function Measured() {
  return (
    <div style={ground}>
      <span>swarm-manager · six weekly readings</span>
      <SampleSeries series={[30, 43, 16, 17, 174, 72]} illustrative={false} />
    </div>
  );
}

/** A single point is not a series; nothing is drawn rather than a misleading flat line. */
export function SinglePoint() {
  return (
    <div style={ground}>
      <span>one reading, no series</span>
      <SampleSeries series={[7]} illustrative />
    </div>
  );
}
