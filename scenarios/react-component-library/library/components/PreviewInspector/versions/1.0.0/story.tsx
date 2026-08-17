import { PreviewInspector } from "./PreviewInspector";

export function Default() {
  return (
    <div style={{ display: "flex", justifyContent: "flex-end", padding: "var(--space-xl)", background: "var(--color-surface-muted)" }}>
      <PreviewInspector title="Preview inspector" description="Adjust the active story without leaving the workbench." onClose={() => undefined}>
        <div style={{ display: "grid", gap: "var(--space-sm)" }}>
          <label style={{ display: "grid", gap: "var(--space-3xs)", font: "var(--text-body-sm)" }}>
            Story label
            <input aria-label="Story label" defaultValue="Ready state" style={{ minHeight: "var(--tap-target-min)", paddingInline: "var(--space-sm)", border: "var(--border-hairline) solid var(--color-border)", borderRadius: "var(--radius-control)", background: "var(--color-surface)", color: "var(--color-foreground)" }} />
          </label>
          <p role="status" style={{ margin: 0, color: "var(--color-muted-foreground)", font: "var(--text-caption)" }}>Changes apply to this specimen only.</p>
        </div>
      </PreviewInspector>
    </div>
  );
}
