/** @vrooliComponentSource react-component-library:CaptureGrid */
export interface CaptureCell {
  id: string;
  viewport: string;
  theme: "light" | "dark";
  status: "pass" | "missing" | "stale";
  captureRef?: string;
}
export function CaptureGrid({ cells = [] }: { cells?: CaptureCell[] }) {
  return (
    <div
      role="grid"
      aria-label="Evidence captures"
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
}
