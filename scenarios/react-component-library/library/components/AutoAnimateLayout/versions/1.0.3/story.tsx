import { useState } from "react";
import { AutoAnimateLayout } from "./AutoAnimateLayout";

const frame = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 620px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

const actionStyle = {
  minBlockSize: "var(--tap-target-min)",
  padding: "var(--space-2xs) var(--space-sm)",
  border: "var(--border-hairline) solid currentColor",
  borderRadius: "var(--radius-control)",
  background: "transparent",
  color: "var(--color-primary, #2563eb)",
  font: "var(--text-label)",
  cursor: "pointer",
} as const;

export function Default() {
  const [items, setItems] = useState(["Research", "Design", "Launch"]);
  return (
    <section style={frame}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Layout motion
        </span>
        <strong style={{ font: "var(--text-title)" }}>Movement follows the cause</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Keyed rows animate their displacement through a shared FLIP boundary; layout properties
          themselves are never tweened.
        </span>
      </div>
      <AutoAnimateLayout>
        <div style={{ display: "grid", gap: "var(--space-xs)" }}>
          {items.map((item) => (
            <div
              key={item}
              data-layout-key={item}
              style={{
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
                gap: "var(--space-sm)",
                minBlockSize: "var(--tap-target-min)",
                padding: "var(--space-xs) var(--space-sm)",
                border: "var(--border-hairline) solid var(--color-border)",
                borderRadius: "var(--radius-panel)",
                background: "var(--color-surface-muted)",
                font: "var(--text-label)",
              }}
            >
              <span>{item}</span>
              <span
                style={{
                  color: "var(--color-muted-foreground)",
                  font: "var(--text-caption)",
                }}
              >
                Keyed
              </span>
            </div>
          ))}
        </div>
      </AutoAnimateLayout>
      <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-2xs)" }}>
        <button
          style={actionStyle}
          type="button"
          onClick={() => setItems((current) => [...current, "Review"])}
        >
          Add row
        </button>
        <button
          style={actionStyle}
          type="button"
          onClick={() => setItems((current) => [...current].reverse())}
        >
          Reverse order
        </button>
      </div>
    </section>
  );
}
