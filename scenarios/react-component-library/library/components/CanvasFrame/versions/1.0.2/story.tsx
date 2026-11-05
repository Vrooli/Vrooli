import { CanvasFrame } from "./CanvasFrame";

export function Default({ args = {} }: { args?: { mode?: "focus" | "canvas" } }) {
  const mode = args.mode ?? "focus";
  return (
    <div style={{ padding: "var(--space-xl)" }}>
      <CanvasFrame mode={mode}>
        <div
          style={{
            display: "grid",
            placeItems: "center",
            minHeight: "10rem",
            border: "var(--border-hairline) dashed var(--color-primary)",
            borderRadius: "var(--radius-control)",
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Specimen placement surface
        </div>
      </CanvasFrame>
    </div>
  );
}
