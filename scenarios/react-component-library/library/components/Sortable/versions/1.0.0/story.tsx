import { useState } from "react";
import { Sortable, type SortableItem } from "./Sortable";

const frame = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 660px)",
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
  color: "var(--color-primary)",
  font: "var(--text-label)",
  cursor: "pointer",
} as const;
const initial: SortableItem[] = [
  { id: "research", value: "Research", label: "Research brief" },
  { id: "design", value: "Design", label: "Design review" },
  { id: "launch", value: "Launch", label: "Launch checklist" },
];

export function Default() {
  const [shouldFail, setShouldFail] = useState(false);
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
          Reorderable work
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          Order is an intentional interaction
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Drag handles and keyboard commands share one model; persistence is
          optimistic, visible, and reversible.
        </span>
      </div>
      <Sortable
        items={initial}
        onReorder={async (next) => {
          await new Promise((resolve) => setTimeout(resolve, 80));
          if (shouldFail) throw new Error("Persistence failed");
          void next;
        }}
      />
      <div
        style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-2xs)" }}
      >
        <button
          style={actionStyle}
          type="button"
          onClick={() => setShouldFail((value) => !value)}
        >
          {shouldFail ? "Allow saves" : "Simulate save failure"}
        </button>
      </div>
    </section>
  );
}
