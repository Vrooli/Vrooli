import { useState, type ReactNode } from "react";
import { DataToolbar, type DataToolbarDensity } from "./DataToolbar";

const frame = {
  display: "grid",
  gap: "var(--space-lg)",
  width: "min(100%, 980px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

function Showcase({
  eyebrow,
  title,
  detail,
  children,
}: {
  eyebrow: string;
  title: string;
  detail: string;
  children: ReactNode;
}) {
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
          {eyebrow}
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
            maxInlineSize: "65ch",
          }}
        >
          {detail}
        </span>
      </div>
      {children}
    </section>
  );
}

const filterOptions = [
  { id: "ready", label: "Ready", count: 24 },
  { id: "attention", label: "Needs attention", count: 7 },
  { id: "archived", label: "Archived", count: 3 },
];

const views = [
  { id: "all", label: "All resources", count: 34 },
  { id: "attention", label: "My attention", count: 7 },
  { id: "recent", label: "Recently changed" },
];

const sortOptions = [
  { id: "updated", label: "Recently updated" },
  { id: "name", label: "Name" },
  { id: "status", label: "Status" },
];

function ExampleToolbar({
  status,
  defaultDensity,
}: {
  status?: "refreshing" | "offline" | "stale";
  defaultDensity?: DataToolbarDensity;
}) {
  const [lastAction, setLastAction] = useState(
    "Ready to refine the collection",
  );
  const [density, setDensity] = useState<DataToolbarDensity>(
    defaultDensity ?? "comfortable",
  );

  return (
    <div style={{ display: "grid", gap: "var(--space-sm)" }}>
      <DataToolbar
        defaultQuery=""
        filterOptions={filterOptions}
        defaultActiveFilterIds={status === "offline" ? ["ready"] : []}
        views={views}
        defaultViewId="all"
        sortOptions={sortOptions}
        defaultSortId="updated"
        defaultDensity={density}
        status={status}
        lastUpdatedLabel={
          status === "offline" ? "Saved 8 minutes ago" : "Updated just now"
        }
        onApply={({ query, activeFilterIds }) =>
          setLastAction(
            `Applied ${activeFilterIds.length} filter${activeFilterIds.length === 1 ? "" : "s"}${query ? ` for “${query}”` : ""}`,
          )
        }
        onReset={() => setLastAction("Filters reset")}
        onViewChange={(id) =>
          setLastAction(
            `View: ${views.find((view) => view.id === id)?.label ?? id}`,
          )
        }
        onSortChange={(id) =>
          setLastAction(
            `Sort: ${sortOptions.find((option) => option.id === id)?.label ?? id}`,
          )
        }
        onDensityChange={setDensity}
        onRefresh={() => setLastAction("Refresh requested")}
        onExport={() => setLastAction("Export requested")}
        onColumns={() => setLastAction("Column settings opened")}
      />
      <span
        role="status"
        aria-label="Last toolbar action"
        style={{
          color: "var(--color-muted-foreground)",
          font: "var(--text-caption)",
        }}
      >
        {lastAction} · {density} density
      </span>
    </div>
  );
}

export function Default() {
  return (
    <Showcase
      eyebrow="Collection controls"
      title="A calm command surface for busy data"
      detail="Search, filters, saved views, sorting, freshness, and actions stay close to the collection they shape—without competing for attention."
    >
      <ExampleToolbar />
    </Showcase>
  );
}

export function Refreshing() {
  return (
    <Showcase
      eyebrow="Live refresh"
      title="Context stays anchored while results update"
      detail="The refreshing state preserves the current controls and makes the in-flight work explicit instead of replacing the whole surface with a spinner."
    >
      <ExampleToolbar status="refreshing" />
    </Showcase>
  );
}

export function Offline() {
  return (
    <Showcase
      eyebrow="Resilient collection"
      title="Useful even when the network is quiet"
      detail="Saved context remains visible, the freshness boundary is honest, and recovery actions retain their familiar position."
    >
      <ExampleToolbar status="offline" defaultDensity="compact" />
    </Showcase>
  );
}
