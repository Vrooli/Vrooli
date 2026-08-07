import type { ReactNode } from "react";
import { Checkbox } from "./Checkbox";

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
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".08em",
            textTransform: "uppercase",
          }}
        >
          Preference control
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          Account preferences
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Make independent choices with enough context to understand the
          consequence.
        </span>
      </div>
      {children}
    </section>
  );
}

export function Default() {
  return (
    <Showcase>
      <Checkbox
        label="Keep me signed in"
        description="Use this only on a private device."
        defaultChecked
      />
    </Showcase>
  );
}

export function Invalid() {
  return (
    <Showcase>
      <Checkbox
        label="Billing alerts"
        description="Receive important account notices."
        error="Choose at least one alert channel."
      />
    </Showcase>
  );
}
