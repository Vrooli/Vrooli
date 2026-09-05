import { useState, type ReactNode } from "react";
import { ResourceDetail, type ResourceDetailStatus } from "./ResourceDetail";

const frame = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 940px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;
const entries = [
  { term: "Owner", description: "Platform operations" },
  { term: "Status", description: "Needs review" },
  { term: "Last changed", description: "8 minutes ago" },
];
const history = [
  { actor: "Mara Chen", action: "Updated the release scope" },
  { actor: "Jon Bell", action: "Reviewed the mobile handoff" },
];

function Detail({
  status = "default",
  children,
}: {
  status?: ResourceDetailStatus;
  children?: ReactNode;
}) {
  const [retried, setRetried] = useState(false);
  return (
    <ResourceDetail
      title="Release brief"
      description="A route-safe record surface with freshness, metadata, history, and recovery."
      entries={entries}
      history={history}
      status={status}
      freshness={
        status === "stale"
          ? "Showing saved version · refresh pending"
          : "Updated 8 minutes ago"
      }
      onRetry={() => setRetried(true)}
    >
      {children}
      {retried ? <span>Latest version requested</span> : null}
    </ResourceDetail>
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
          Resource detail
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
      title="The record stays the product surface"
      detail="Identity, metadata, history, and freshness remain readable without asking the route to coordinate every state."
    >
      <Detail />
    </Showcase>
  );
}
export function Refreshing() {
  return (
    <Showcase
      title="Refresh without losing context"
      detail="The last useful record remains visible while a newer version is requested."
    >
      <Detail status="refreshing" />
    </Showcase>
  );
}
export function Partial() {
  return (
    <Showcase
      title="Partial data is named"
      detail="Usable fields stay available while incomplete information is disclosed at the surface boundary."
    >
      <Detail status="partial" />
    </Showcase>
  );
}
export function Empty() {
  return (
    <Showcase
      title="Empty is a real state"
      detail="The detail surface explains why there is nothing to inspect without leaving a blank shell."
    >
      <Detail status="empty" />
    </Showcase>
  );
}
export function RequestError() {
  return (
    <Showcase
      title="Recovery stays local"
      detail="A failed refresh keeps the record context and exposes a retry path."
    >
      <Detail status="request-error" />
    </Showcase>
  );
}
export function PermissionDenied() {
  return (
    <Showcase
      title="Permission is explained"
      detail="The user learns why the record is unavailable instead of seeing a generic failure."
    >
      <Detail status="permission-denied" />
    </Showcase>
  );
}
