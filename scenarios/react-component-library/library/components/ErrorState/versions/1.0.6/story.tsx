import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { ReactNode } from "react";
import { ErrorState } from "./ErrorState";

function Showcase({ children }: { children: ReactNode }) {
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
            color: "var(--color-danger)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          {useStrings("feedback.error-state.recovery-surface", "Recovery surface")}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {useStrings(
            "feedback.error-state.failure-without-a-dead-end",
            "Failure without a dead end",
          )}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          The message is specific, the next action is clear, and the surrounding hierarchy stays
          intact.
        </span>
      </div>
      {children}
    </section>
  );
}

export function Error() {
  return (
    <Showcase>
      <ErrorState
        title={useStrings("feedback.error-state.title", "Sync could not finish")}
        message="The service did not respond. Your local changes are safe."
        onRetry={() => undefined}
      />
    </Showcase>
  );
}
