import { CanvasFrame } from "./CanvasFrame";

export function Default() {
  return (
    <div style={{ padding: "var(--space-xl)" }}>
      <CanvasFrame>
        <div style={{ display: "grid", placeItems: "center", minHeight: "10rem", border: "var(--border-hairline) dashed var(--color-primary)", borderRadius: "var(--radius-control)", color: "var(--color-muted-foreground)", font: "var(--text-body)" }}>
          Specimen placement surface
        </div>
      </CanvasFrame>
    </div>
  );
}
