/**
 * @libraryId react-component-library:CaptureGrid
 * @displayName CaptureGrid
 * @description A responsive matrix of declared viewport and theme evidence captures.
 * @version 1.0.6
 * @tags ["visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CaptureGrid */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

export interface CaptureCell {
  id: string;
  viewport: string;
  theme: "light" | "dark";
  status: "pass" | "missing" | "stale";
  captureRef?: string;
}
export const CaptureGrid = withClassName(function CaptureGrid({
  cells = [],
}: {
  cells?: CaptureCell[];
}) {
  const strings = useStrings();
  return (
    <div
      data-testid="visualization.capture-grid"
      role="grid"
      aria-label={strings("visualization.capture-grid.evidence-captures", "Evidence captures")}
      style={{
        display: "grid",
        gridTemplateColumns: "repeat(auto-fit, minmax(12rem, 1fr))",
        gap: "var(--space-xs)",
      }}
    >
      {cells.map((cell) => (
        <div
          role="gridcell"
          key={cell.id}
          data-status={cell.status}
          style={{
            border: "1px solid var(--color-border)",
            borderRadius: "var(--radius-control)",
            padding: "var(--space-xs)",
          }}
        >
          <strong>{cell.viewport}</strong>
          <span> · {cell.theme}</span>
          <p>{cell.status}</p>
          {cell.captureRef ? <small>{cell.captureRef}</small> : null}
        </div>
      ))}
    </div>
  );
});
