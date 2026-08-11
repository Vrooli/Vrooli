import { useState, type ReactNode } from "react";
import { ErrorBoundary } from "./ErrorBoundary";

const shell = {
  display: "grid",
  gap: "var(--space-md)",
  width: "min(100%, 640px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

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
    <section style={shell}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Recovery boundary
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

function BrokenModule(): ReactNode {
  throw new Error("The render region failed while rendering");
}

export function RequestError() {
  return (
    <Showcase
      title="The surrounding product stays alive"
      detail="A render failure is contained to its region, named clearly, and paired with a deliberate recovery action."
    >
      <ErrorBoundary contextLabel="Insight module" showDiagnostics>
        <BrokenModule />
      </ErrorBoundary>
    </Showcase>
  );
}

export function Retry() {
  const [failed, setFailed] = useState(true);
  return (
    <Showcase
      title="Retry resets the failed region"
      detail="The boundary can retry without forcing a full-page reload or losing the host application context."
    >
      <ErrorBoundary
        contextLabel="Live insight"
        resetKeys={[failed]}
        onRetry={() => setFailed(false)}
      >
        {failed ? (
          <BrokenModule />
        ) : (
          <div role="status">Insight module restored</div>
        )}
      </ErrorBoundary>
    </Showcase>
  );
}

export function CustomFallback() {
  return (
    <Showcase
      title="Consumers can own the fallback"
      detail="A product can preserve its own visual language while retaining the same boundary and telemetry behavior."
    >
      <ErrorBoundary
        fallback={
          <div
            role="alert"
            style={{
              padding: "var(--space-lg)",
              border: "var(--border-hairline) solid var(--color-danger)",
              borderRadius: "var(--radius-panel)",
              color: "var(--color-danger)",
            }}
          >
            Workspace panel unavailable
          </div>
        }
      >
        <BrokenModule />
      </ErrorBoundary>
    </Showcase>
  );
}
