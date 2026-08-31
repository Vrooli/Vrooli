import { useState, type ReactNode } from "react";
import { ValidationSummary } from "./ValidationSummary";
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
  const [store] = useState(() => {
    const nextStore = createFormStore({
      initialValues: { workspace: "", email: "" },
    });
    nextStore.setError("workspace", "Choose a workspace name.");
    nextStore.setError("email", "Enter a reachable email address.");
    return nextStore;
  });
  return (
    <Showcase
      title="A map back to progress"
      detail="Each error names the field and gives keyboard and pointer users a direct route back to it."
    >
      <ValidationSummary
        store={store}
        fieldLabels={{ workspace: "Workspace name", email: "Email address" }}
      />
    </Showcase>
  );
}

export function CustomTitle() {
  return (
    <Showcase
      title="A focused recovery message"
      detail="The summary can be used with server errors or externally managed validation maps."
    >
      <ValidationSummary
        errors={{
          plan: "Choose a plan before continuing.",
          consent: "Confirm that you agree to the terms.",
        }}
        fieldLabels={{ plan: "Plan", consent: "Terms" }}
        title="Almost there"
      />
    </Showcase>
  );
}

export function Empty() {
  return (
    <Showcase
      title="No errors"
      detail="The summary stays out of the layout when there is nothing to recover."
    >
      <span
        style={{ color: "var(--color-success)", font: "var(--text-label)" }}
      >
        Everything looks good.
      </span>
      <ValidationSummary errors={{}} />
    </Showcase>
  );
}
