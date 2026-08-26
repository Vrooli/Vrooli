import type { ReactNode } from "react";
import { UndoBanner } from "./UndoBanner";
import { UndoManagerProvider } from "@vrooli/react-component-library/UndoManager/1.0.0";

const noopUndo = () => undefined;

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
    <section
      style={{
        position: "relative",
        display: "grid",
        alignContent: "start",
        gap: "var(--space-md)",
        width: "min(100%, 600px)",
        minHeight: 360,
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
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          Reversible action
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

const previewPosition = {
  position: "absolute" as const,
  insetInline: "var(--space-lg)",
  insetBlockEnd: "var(--space-lg)",
};

function Provider({
  status = "available" as const,
}: {
  status?: "available" | "submitting" | "success" | "error";
}) {
  return (
    <UndoManagerProvider
      initialRecords={[
        {
          id: "archive",
          title: "Project brief moved to archive",
          detail: "You have a few seconds to restore it.",
          successMessage: "Project brief restored",
          status,
          error:
            status === "error"
              ? "The archive service is still catching up."
              : undefined,
          undo: noopUndo,
          expiresMs: 12000,
        },
      ]}
    >
      <UndoBanner style={previewPosition} />
    </UndoManagerProvider>
  );
}

export function Default() {
  return (
    <Showcase
      title="A way back, without an interruption"
      detail="Destructive work completes immediately, with a visible and time-bounded reversal."
    >
      <Provider />
    </Showcase>
  );
}
export function Submitting() {
  return (
    <Showcase
      title="Restoring the previous state"
      detail="Progress is explicit while the rollback request is in flight."
    >
      <Provider status="submitting" />
    </Showcase>
  );
}
export function Success() {
  return (
    <Showcase
      title="The change is safely reversed"
      detail="The confirmation stays calm and gives the user closure."
    >
      <Provider status="success" />
    </Showcase>
  );
}
export function RequestError() {
  return (
    <Showcase
      title="Undo needs another try"
      detail="The original action remains understandable, and recovery stays one click away."
    >
      <Provider status="error" />
    </Showcase>
  );
}
export function Interactive() {
  return (
    <Showcase
      title="Undo the archive action"
      detail="Try the action to see the banner move from available to restored."
    >
      <UndoManagerProvider
        initialRecords={[
          {
            id: "archive-interactive",
            title: "Project brief moved to archive",
            detail: "Undo is available for a few seconds.",
            successMessage: "Project brief restored",
            undo: noopUndo,
            expiresMs: 12000,
          },
        ]}
      >
        <UndoBanner style={previewPosition} />
      </UndoManagerProvider>
    </Showcase>
  );
}
