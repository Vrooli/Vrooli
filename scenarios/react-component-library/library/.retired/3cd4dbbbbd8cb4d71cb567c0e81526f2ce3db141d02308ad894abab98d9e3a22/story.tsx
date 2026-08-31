import { useState, type ReactNode } from "react";
import {
  ConflictResolutionFlow,
  type ConflictField,
} from "./ConflictResolutionFlow";

const frame = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 700px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;
const fields: ConflictField<string>[] = [
  {
    id: "title",
    label: "Title",
    local: "Release brief",
    remote: "Release brief — revised",
    description:
      "The remote editor clarified the scope while you were offline.",
  },
  {
    id: "owner",
    label: "Owner",
    local: "Mara Chen",
    remote: "Jon Bell",
    description: "Both versions are valid assignments; choose deliberately.",
  },
];

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
          Concurrent editing
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
      title="Make the disagreement understandable"
      detail="Field-level choices keep both versions visible and preserve the user's work until the resolution is saved."
    >
      <ConflictResolutionFlow fields={fields} />
    </Showcase>
  );
}
export function Loading() {
  return (
    <Showcase
      title="Check before choosing"
      detail="The comparison remains stable while the latest server version is fetched."
    >
      <ConflictResolutionFlow fields={fields} status="loading" />
    </Showcase>
  );
}
export function RequestError() {
  return (
    <Showcase
      title="Keep choices through a retry"
      detail="A second conflict does not erase the decisions already made."
    >
      <ConflictResolutionFlow fields={fields} status="request-error" />
    </Showcase>
  );
}
export function Interactive() {
  const [saved, setSaved] = useState(false);
  return (
    <Showcase
      title="Resolution is a real workflow"
      detail="Choose a remote value, then submit the resolved record with a live success state."
    >
      <ConflictResolutionFlow
        fields={fields}
        onResolve={async () => {
          await new Promise((resolve) => setTimeout(resolve, 80));
          setSaved(true);
        }}
      />
      {saved ? <span>Resolved version ready</span> : null}
    </Showcase>
  );
}
