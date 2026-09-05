import { useState, type ReactNode } from "react";
import { AsyncBoundary } from "./AsyncBoundary";

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
        gap: "var(--space-lg)",
        width: "min(100%, 720px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background:
          "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 4%, var(--color-surface-raised)))",
        boxShadow: "var(--elev-raised)",
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
          Resilient data surface
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            maxWidth: "58ch",
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

function DashboardContent() {
  return (
    <div style={{ display: "grid", gap: "var(--space-md)" }}>
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          alignItems: "baseline",
          justifyContent: "space-between",
          gap: "var(--space-sm)",
        }}
      >
        <div style={{ display: "grid", gap: "var(--space-3xs)" }}>
          <strong style={{ font: "var(--text-subtitle)" }}>
            Revenue overview
          </strong>
          <span
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-caption)",
            }}
          >
            Last updated 2 minutes ago
          </span>
        </div>
        <strong
          style={{ color: "var(--color-success)", font: "var(--text-title)" }}
        >
          $48,240
        </strong>
      </div>
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fit, minmax(min(100%, 7rem), 1fr))",
          gap: "var(--space-xs)",
        }}
      >
        {["+12.4%", "186 orders", "94% retained"].map((value) => (
          <span
            key={value}
            style={{
              minWidth: 0,
              overflowWrap: "anywhere",
              padding: "var(--space-sm)",
              border: "1px solid var(--color-border)",
              borderRadius: "var(--radius-control)",
              background: "var(--color-surface)",
              color: "var(--color-muted-foreground)",
              font: "var(--text-label)",
            }}
          >
            {value}
          </span>
        ))}
      </div>
    </div>
  );
}

export function Default() {
  return (
    <Showcase
      title="A boundary that stays calm under change"
      detail="The content region remains a product surface, while the boundary communicates freshness and recovery around it."
    >
      <AsyncBoundary status="success" aria-label="Revenue overview">
        <DashboardContent />
      </AsyncBoundary>
    </Showcase>
  );
}

export function Pending() {
  return (
    <Showcase
      title="No flash, no blank rectangle"
      detail="The first-load skeleton waits briefly, then mirrors the shape of the content it is replacing."
    >
      <AsyncBoundary
        status="pending"
        pending={
          <>
            <strong style={{ font: "var(--text-subtitle)" }}>
              Preparing your workspace
            </strong>
            <span
              style={{
                color: "var(--color-muted-foreground)",
                font: "var(--text-body)",
              }}
            >
              Loading the latest revenue data.
            </span>
          </>
        }
      />
    </Showcase>
  );
}

export function Refreshing() {
  return (
    <Showcase
      title="Refresh without losing your place"
      detail="Background work is visible, but the last useful result never disappears while fresh data is requested."
    >
      <AsyncBoundary
        status="refreshing"
        aria-label="Refreshing revenue overview"
      >
        <DashboardContent />
      </AsyncBoundary>
    </Showcase>
  );
}

export function Stale() {
  return (
    <Showcase
      title="Useful before perfect"
      detail="When freshness is temporarily unavailable, the user still gets the answer they came for and an honest explanation."
    >
      <AsyncBoundary status="stale" aria-label="Saved revenue overview">
        <DashboardContent />
      </AsyncBoundary>
    </Showcase>
  );
}

export function PartialError() {
  return (
    <Showcase
      title="One failed section should not take down the page"
      detail="Partial failure preserves successful regions and gives the product room to recover independently."
    >
      <AsyncBoundary
        status="partial-error"
        aria-label="Revenue overview with a partial error"
      >
        <DashboardContent />
      </AsyncBoundary>
    </Showcase>
  );
}

export function Offline() {
  return (
    <Showcase
      title="Offline is a context, not a dead end"
      detail="Cached content remains legible while the connection is unavailable, so the user can keep making decisions."
    >
      <AsyncBoundary status="offline" aria-label="Offline revenue overview">
        <DashboardContent />
      </AsyncBoundary>
    </Showcase>
  );
}

export function Error() {
  return (
    <Showcase
      title="An error that leaves a way forward"
      detail="Failure copy is specific, the retry target is obvious, and the surrounding layout remains composed."
    >
      <AsyncBoundary
        status="error"
        error="The revenue service did not respond. Your saved view is safe."
        retry={() => undefined}
        aria-label="Revenue overview error"
      />
    </Showcase>
  );
}

export function ErrorRecovery() {
  const [status, setStatus] = useState<"error" | "success">("error");
  return (
    <Showcase
      title="Recovery has a clear next step"
      detail="A failure explains what happened, keeps the same visual footprint, and makes retry a first-class keyboard action."
    >
      <AsyncBoundary
        status={status}
        error="The revenue service did not respond. Your saved view is safe."
        retry={() => setStatus("success")}
        aria-label="Revenue overview"
      >
        {status === "success" ? <DashboardContent /> : undefined}
      </AsyncBoundary>
    </Showcase>
  );
}
