import { PreviewDock } from "./PreviewDock";

const button = {
  minHeight: "var(--tap-target-min)",
  paddingInline: "var(--space-sm)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-control)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
  font: "var(--text-body-sm)",
};

export function Default() {
  return (
    <div
      style={{
        display: "grid",
        gap: "var(--space-lg)",
        padding: "var(--space-xl)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Preview chrome
        </span>
        <h1 style={{ margin: 0, font: "var(--text-heading)" }}>
          Controls stay with the workbench
        </h1>
        <p
          style={{
            margin: 0,
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          The dock floats over the stage so viewport controls never steal
          specimen space.
        </p>
      </div>
      <PreviewDock>
        <button
          type="button"
          style={{
            ...button,
            borderColor: "var(--color-primary)",
            background: "var(--color-primary)",
            color: "var(--color-primary-foreground)",
          }}
        >
          Focus
        </button>
        <button type="button" style={button}>
          Canvas
        </button>
        <button type="button" style={button}>
          Desktop · 100%
        </button>
        <span
          role="status"
          style={{
            marginInlineStart: "auto",
            color: "var(--color-muted-foreground)",
            font: "var(--text-caption)",
          }}
        >
          Ready to inspect
        </span>
      </PreviewDock>
    </div>
  );
}
