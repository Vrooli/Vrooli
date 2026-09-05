import { useState, type ReactNode } from "react";
import { FormActions } from "./FormActions";
import { createFormStore } from "@vrooli/react-component-library/FormStore/1.0.0";

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
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
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
      title="Actions with a clear priority"
      detail="The primary action is visually decisive while reset remains available and quiet."
    >
      <FormActions resetLabel="Reset" cancelLabel="Cancel" />
    </Showcase>
  );
}

export function Pending() {
  const [store] = useState(() => {
    const nextStore = createFormStore({
      initialValues: { workspace: "Aurora" },
    });
    nextStore.setPhase("submitting");
    return nextStore;
  });
  return (
    <Showcase
      title="Feedback while work is in motion"
      detail="Actions stay in place while the pending state explains what the system is doing."
    >
      <FormActions store={store} resetLabel="Reset" />
    </Showcase>
  );
}

export function Compact() {
  return (
    <Showcase
      title="Compact composition"
      detail="Use the same semantics in a denser toolbar or a narrow mobile footer."
    >
      <FormActions align="between" submitLabel="Continue" cancelLabel="Back" />
    </Showcase>
  );
}
