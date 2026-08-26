/**
 * @libraryId react-component-library:CaptureGrid
 * @displayName CaptureGrid
 * @description A responsive matrix of declared viewport and theme evidence captures.
 * @version 1.0.4
 * @tags ["visualization","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:CaptureGrid */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { withClassName } from "../../../../foundations/ClassMerge/versions/1.0.1/ClassMerge";

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
  return (
    <div
      role="grid"
      aria-label={translate("visualization.capture-grid.aria-label.1", "Evidence captures")}
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
