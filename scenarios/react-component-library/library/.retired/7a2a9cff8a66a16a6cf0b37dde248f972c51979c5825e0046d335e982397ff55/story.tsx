import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { ReactNode } from "react";
import { OfflineState } from "./OfflineState";

function Showcase({ children }: { children: ReactNode }) {
  const libraryStrings = useStrings();
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-md)",
        width: "min(100%, 560px)",
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
            color: "var(--color-warning)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          {libraryStrings(
            "feedback.offline-state.connection-context",
            "Connection context",
          )}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {libraryStrings(
            "feedback.offline-state.offline-still-understandable",
            "Offline, still understandable",
          )}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Connectivity is different from failure: give the user a truthful next
          step while preserving what is still useful.
        </span>
      </div>
      {children}
    </section>
  );
}

export function Offline() {
  const libraryStrings = useStrings();
  return (
    <Showcase>
      <OfflineState onRetry={() => undefined}>
        <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
          <strong>
            {libraryStrings(
              "feedback.offline-state.saved-workspace-snapshot",
              "Saved workspace snapshot",
            )}
          </strong>
          <span
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-body)",
            }}
          >
            {libraryStrings(
              "feedback.offline-state.last-synced-4-minutes-ago",
              "Last synced 4 minutes ago.",
            )}
          </span>
        </div>
      </OfflineState>
    </Showcase>
  );
}
