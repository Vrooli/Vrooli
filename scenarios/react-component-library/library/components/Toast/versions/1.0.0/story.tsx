import type { ReactNode } from "react";
import { Toast } from "./Toast";
import {
  ToastManagerProvider,
  useToastManager,
} from "../../../../services/ToastManager/versions/1.0.0/ToastManager";

function Trigger() {
  const manager = useToastManager();
  return (
    <button
      type="button"
      onClick={() =>
        manager.push({
          id: "sync",
          dedupeKey: "sync",
          tone: "success",
          title: "Workspace synced",
          message: "Your latest changes are safe.",
        })
      }
    >
      Show confirmation
    </button>
  );
}

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
        gap: "var(--space-md)",
        width: "min(100%, 560px)",
        minHeight: "280px",
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
          Transient feedback
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
      title="Confirmation that stays out of the way"
      detail="Visible, announced, dismissible, and temporary."
    >
      <ToastManagerProvider
        initialToasts={[
          {
            id: "saved",
            tone: "success",
            title: "Changes saved",
            message: "Your workspace is up to date.",
            durationMs: 0,
          },
        ]}
      >
        <Toast />
      </ToastManagerProvider>
    </Showcase>
  );
}

export function Interactive() {
  return (
    <Showcase
      title="A notification with an owner"
      detail="The manager owns queueing, dedupe, lifetime, and announcement policy."
    >
      <ToastManagerProvider>
        <Trigger />
        <Toast />
      </ToastManagerProvider>
    </Showcase>
  );
}
