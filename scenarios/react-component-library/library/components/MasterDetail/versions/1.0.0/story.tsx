import { useState, type ReactNode } from "react";
import {
  MasterDetail,
  type MasterDetailItem,
  type MasterDetailStatus,
} from "./MasterDetail";

const frame = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 980px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

const records: MasterDetailItem[] = [
  {
    id: "brief",
    title: "Release brief",
    summary: "Three decisions need a clear owner before Friday.",
    meta: "Updated 8 min ago",
    value: { owner: "Mara Chen", state: "Needs review" },
  },
  {
    id: "handoff",
    title: "Design handoff",
    summary: "The mobile states are ready for engineering review.",
    meta: "Updated yesterday",
    value: { owner: "Jon Bell", state: "In review" },
  },
  {
    id: "checklist",
    title: "Launch checklist",
    summary: "A compact record with one remaining operational check.",
    meta: "Updated Monday",
    value: { owner: "Inez Okafor", state: "Ready" },
  },
];

function Detail({ item }: { item: MasterDetailItem }) {
  const value = item.value as { owner: string; state: string };
  return (
    <div style={{ display: "grid", gap: "var(--space-md)" }}>
      <p
        style={{
          margin: 0,
          color: "var(--color-muted-foreground)",
          font: "var(--text-body)",
        }}
      >
        {item.summary}
      </p>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
          gap: "var(--space-xs)",
        }}
      >
        <div
          style={{
            display: "grid",
            gap: "var(--space-3xs)",
            padding: "var(--space-sm)",
            border: "var(--border-hairline) solid var(--color-border)",
            borderRadius: "var(--radius-control)",
            background: "var(--color-surface-muted)",
          }}
        >
          <span
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-overline)",
              letterSpacing: ".08em",
              textTransform: "uppercase",
            }}
          >
            Owner
          </span>
          <strong style={{ font: "var(--text-label)" }}>{value.owner}</strong>
        </div>
        <div
          style={{
            display: "grid",
            gap: "var(--space-3xs)",
            padding: "var(--space-sm)",
            border: "var(--border-hairline) solid var(--color-border)",
            borderRadius: "var(--radius-control)",
            background: "var(--color-surface-muted)",
          }}
        >
          <span
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-overline)",
              letterSpacing: ".08em",
              textTransform: "uppercase",
            }}
          >
            State
          </span>
          <strong
            style={{ color: "var(--color-primary)", font: "var(--text-label)" }}
          >
            {value.state}
          </strong>
        </div>
      </div>
      <p
        style={{
          margin: 0,
          color: "var(--color-muted-foreground)",
          font: "var(--text-caption)",
        }}
      >
        This detail stays in context on wide screens and becomes a focused
        drill-in on compact screens.
      </p>
    </div>
  );
}

function Showcase({
  title,
  detail,
  children,
}: {
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
          Collection navigation
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

function Surface({
  status = "default",
  selectedId,
  items = records,
}: {
  status?: MasterDetailStatus;
  selectedId?: string;
  items?: MasterDetailItem[];
}) {
  const [selected, setSelected] = useState(selectedId);
  return (
    <MasterDetail
      items={items}
      selectedId={selected ?? null}
      status={status}
      onSelect={(item) => setSelected(item.id)}
      onBack={() => setSelected(undefined)}
      renderDetail={(item) => <Detail item={item} />}
      label="Release workspace"
    />
  );
}

export function Default() {
  return (
    <Showcase
      title="Inspect without losing your place"
      detail="A selected record stays adjacent to its collection on desktop and becomes a focused detail route on mobile."
    >
      <Surface selectedId="brief" />
    </Showcase>
  );
}

export function Collection() {
  return (
    <Showcase
      title="The collection remains the anchor"
      detail="Selection is explicit, keyboard reachable, and ready to become a route decision owned by the consumer."
    >
      <Surface />
    </Showcase>
  );
}

export function Loading() {
  return (
    <Showcase
      title="Loading preserves the workspace frame"
      detail="The shell remains stable while the collection request is in flight."
    >
      <Surface status="loading" items={[]} />
    </Showcase>
  );
}

export function Empty() {
  return (
    <Showcase
      title="An empty collection is still useful"
      detail="The empty state explains what the user is looking at without inventing a detail record."
    >
      <Surface status="empty" items={[]} />
    </Showcase>
  );
}

export function Partial() {
  return (
    <Showcase
      title="Partial data stays honest"
      detail="Known record context remains visible while incomplete fields are clearly marked."
    >
      <Surface status="partial" selectedId="handoff" />
    </Showcase>
  );
}

export function RequestError() {
  return (
    <Showcase
      title="Recovery stays in context"
      detail="A failed collection request is visible and legible instead of collapsing the surrounding workspace."
    >
      <Surface status="request-error" items={[]} />
    </Showcase>
  );
}
