import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import type { ReactNode } from "react";
import { ErrorState } from "./ErrorState";

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
            color: "var(--color-danger)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          {libraryStrings("feedback.error-state.recovery-surface", "Recovery surface")}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {libraryStrings(
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
  const libraryStrings = useStrings();
  return (
    <Showcase>
      <ErrorState
        title={libraryStrings("feedback.error-state.title", "Sync could not finish")}
        message="The service did not respond. Your local changes are safe."
        onRetry={() => undefined}
      />
    </Showcase>
  );
}

/**
 * The case this version exists for: a transport failure whose verbatim text is
 * six lines of internal vocabulary. The operator reads the sentence; the
 * engineer opens the disclosure.
 */
export function WithDetail() {
  return (
    <Showcase>
      <ErrorState
        title="This machine can't answer configuration questions yet"
        message="Its bridge agent is running an older catalog. Re-apply the profile to update it."
        detail={`node reach transport node="25c7e426-c76c-421a-8351-aaf964589802" verb="vrooli-onboarding"
scenario proxy returned 502 Bad Gateway:
  scenario method api/v2.operator-inputs is not in the governed catalog`}
        correlationId="run_9f3c1a77e4"
        actions={
          <button type="button" data-testid="story-error-action">
            Re-apply profile
          </button>
        }
      />
    </Showcase>
  );
}

/** With no dump to show, the identifier a support conversation asks for still appears. */
export function CorrelationOnly() {
  return (
    <Showcase>
      <ErrorState
        title="The fleet could not be read"
        message="The control plane did not answer. Nothing was changed."
        correlationId="run_2b8d40c1aa"
      />
    </Showcase>
  );
}
