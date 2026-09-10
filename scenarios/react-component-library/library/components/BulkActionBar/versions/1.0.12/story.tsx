import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useState, type ReactNode } from "react";
import { BulkActionBar, type BulkActionBarStatus } from "./BulkActionBar";

const shell = {
  display: "grid",
  alignContent: "start",
  gap: "var(--space-md)",
  width: "min(100%, 720px)",
  minHeight: 360,
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background:
    "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-raised)))",
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
  const libraryStrings = useStrings();
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
          {libraryStrings("data-display.bulk-action-bar.selection-workflow", "Selection workflow")}
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

const common = {
  selectedCount: 8,
  totalCount: 24,
  actionLabel: "Archive selected",
  description: "This action can be undone from the activity stream for the next 30 days.",
  onClear: () => undefined,
};

export function Default() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings("data-display.bulk-action-bar.title", "Context follows selection")}
      detail="The action surface keeps count, scope, and recovery affordances in one deliberate rhythm."
    >
      <BulkActionBar {...common} />
    </Showcase>
  );
}

export function Submitting() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "data-display.bulk-action-bar.title.progress-stays-attached-to-the-decision-detail-a",
        "Progress stays attached to the decision",
      )}
      detail="A long-running action reports concrete progress without replacing the selection context."
    >
      <BulkActionBar
        {...common}
        status="submitting"
        progress={{
          completed: 5,
          total: 8,
          label: "Archiving 5 of 8 records…",
        }}
      />
    </Showcase>
  );
}

export function Success() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "data-display.bulk-action-bar.title.completion-closes-the-loop",
        "Completion closes the loop",
      )}
      detail="The bar confirms the result instead of disappearing and leaving the user to infer what happened."
    >
      <BulkActionBar
        {...common}
        status="success"
        successMessage="Eight records were archived and remain recoverable."
      />
    </Showcase>
  );
}

export function Partial() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "data-display.bulk-action-bar.title.partial-failure-names-the-work-left-behind-detai",
        "Partial failure names the work left behind",
      )}
      detail="A mixed result identifies failed records and makes retry scope explicit."
    >
      <BulkActionBar
        {...common}
        status="partial"
        failedItems={["Untitled release notes", "Arabic workspace handoff"]}
        errorMessage="Six records were updated; these two need attention."
      />
    </Showcase>
  );
}

export function RequestError() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "data-display.bulk-action-bar.title.transport-failure-preserves-intent",
        "Transport failure preserves intent",
      )}
      detail="Nothing claims to have changed, and the selection remains available for a truthful retry."
    >
      <BulkActionBar
        {...common}
        status="request-error"
        errorMessage="The archive service did not respond. Nothing was changed."
      />
    </Showcase>
  );
}

export function Retry() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "data-display.bulk-action-bar.title.retry-is-a-continuation-not-a-reset-detail-the-s",
        "Retry is a continuation, not a reset",
      )}
      detail="The same selection and scope remain visible while recovery is offered."
    >
      <BulkActionBar {...common} status="retry" />
    </Showcase>
  );
}

export function Interactive() {
  const libraryStrings = useStrings();
  const [status, setStatus] = useState<BulkActionBarStatus>("default");
  return (
    <Showcase
      title={libraryStrings(
        "data-display.bulk-action-bar.title.a-real-bulk-action-lifecycle",
        "A real bulk action lifecycle",
      )}
      detail="Run the action to see pending progress, then success without changing the selected count."
    >
      <BulkActionBar
        {...common}
        status={status}
        successMessage="Eight records were archived."
        progress={
          status === "submitting"
            ? { completed: 4, total: 8, label: "Archiving 4 of 8 records…" }
            : undefined
        }
        onAction={async () => {
          setStatus("submitting");
          await new Promise((resolve) => window.setTimeout(resolve, 320));
          setStatus("success");
        }}
        onRetry={() => setStatus("submitting")}
      />
    </Showcase>
  );
}
