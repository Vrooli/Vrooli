import type { ReactNode } from "react";
import { FormField } from "./FormField";

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
        display: "grid",
        gap: "var(--space-md)",
        width: "min(100%, 480px)",
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

const inputStyle = {
  boxSizing: "border-box",
  width: "100%",
  minHeight: 44,
  border: "1px solid var(--color-border)",
  borderRadius: "var(--radius-control)",
  background: "var(--color-surface)",
  color: "var(--color-foreground)",
  paddingInline: "var(--space-sm)",
  font: "inherit",
} as const;

export function Default() {
  return (
    <Showcase
      title="A field that explains itself"
      detail="The label, help text, and control form one predictable relationship."
    >
      <FormField
        label="Workspace name"
        description="This name appears in navigation and notifications."
        control={<input style={inputStyle} placeholder="e.g. Aurora" />}
      />
    </Showcase>
  );
}
export function Required() {
  return (
    <Showcase
      title="Clear required intent"
      detail="Required state is visible in text and native semantics, not color alone."
    >
      <FormField
        label="Email address"
        required
        control={<input type="email" style={inputStyle} placeholder="you@example.com" />}
      />
    </Showcase>
  );
}
export function Error() {
  return (
    <Showcase
      title="Recovery without blame"
      detail="The error stays associated with the control and leaves the user's input intact."
    >
      <FormField
        label="Project slug"
        required
        error="Use lowercase letters, numbers, and hyphens only."
        control={<input style={inputStyle} defaultValue="Aurora Project" />}
      />
    </Showcase>
  );
}
export function Disabled() {
  return (
    <Showcase
      title="Unavailable, but legible"
      detail="Disabled fields preserve their label and explanation so the surrounding workflow remains understandable."
    >
      <FormField
        label="Billing plan"
        disabled
        description="Managed by your organization administrator."
        control={<input style={inputStyle} value="Team" readOnly />}
      />
    </Showcase>
  );
}
