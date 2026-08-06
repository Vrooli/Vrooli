import { useState, type ReactNode } from "react";
import { FilterBar, type FilterOption } from "./FilterBar";

const options: FilterOption[] = [
  { id: "ready", label: "Ready", count: 24 },
  { id: "review", label: "Needs review", count: 7 },
  { id: "archived", label: "Archived", count: 3 },
];

function Showcase({
  children,
  title,
  detail,
}: {
  children: ReactNode;
  title: string;
  detail: string;
}) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-md)",
        width: "min(100%, 720px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          Data workspace
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}

export function Default() {
  return (
    <Showcase
      title="Find the signal quickly"
      detail="Query, narrow, and reset without losing the context of the collection you are exploring."
    >
      <FilterBar options={options} defaultActiveFilterIds={["ready"]} />
    </Showcase>
  );
}

export function Interactive({ log }: StoryHarnessProps<Record<string, never>>) {
  const [lastApplied, setLastApplied] = useState("No filters applied yet");
  return (
    <Showcase
      title="A filter surface with an honest state"
      detail="Every change remains keyboard reachable, while apply and reset stay explicit actions."
    >
      <FilterBar
        options={options}
        onApply={({ query, activeFilterIds }) => {
          const summary = `${query || "All records"} · ${activeFilterIds.length} active`;
          setLastApplied(summary);
          log("apply", summary);
        }}
        onReset={() => {
          setLastApplied("Reset to all records");
          log("reset", "all");
        }}
      />
      <div
        role="status"
        style={{
          color: "var(--color-muted-foreground)",
          font: "var(--text-label)",
        }}
      >
        Last applied: {lastApplied}
      </div>
    </Showcase>
  );
}
