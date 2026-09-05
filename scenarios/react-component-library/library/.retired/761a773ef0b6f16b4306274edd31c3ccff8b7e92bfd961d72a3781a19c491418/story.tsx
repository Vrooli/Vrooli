import type { ReactNode } from "react";
import { SelectionControl } from "./SelectionControl";

function Showcase({ children }: { children: ReactNode }) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 620px)",
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
          Selection grammar
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          Workspace preferences
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Native semantics, calm state changes, and enough context to make a
          decision confidently.
        </span>
      </div>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>{children}</div>
    </section>
  );
}

export function Default() {
  return (
    <Showcase>
      <SelectionControl
        kind="checkbox"
        label="Keep me signed in"
        description="Use this only on a private device."
        defaultChecked
      />
    </Showcase>
  );
}

export function States() {
  return (
    <Showcase>
      <SelectionControl
        kind="checkbox"
        label="Email summaries"
        description="A weekly digest of workspace activity."
        defaultChecked
      />
      <SelectionControl
        kind="switch"
        label="Quiet hours"
        description="Pause non-critical notifications after 6:00 PM."
      />
      <SelectionControl
        kind="radio"
        name="density"
        label="Comfortable density"
        description="More breathing room between controls."
        defaultChecked
      />
      <SelectionControl
        kind="checkbox"
        label="Billing alerts"
        error="Choose at least one alert channel."
      />
    </Showcase>
  );
}

export function Bare() {
  return (
    <Showcase>
      <SelectionControl
        kind="switch"
        aria-label="Adaptive chrome"
        defaultChecked
      />
    </Showcase>
  );
}

export function TrailingLabel() {
  return (
    <Showcase>
      <SelectionControl
        kind="switch"
        labelPlacement="end"
        label="Adaptive chrome"
        description="Tint the chrome to match the focused terminal."
        defaultChecked
      />
    </Showcase>
  );
}

export function Activation() {
  return (
    <Showcase>
      <SelectionControl kind="switch" label="Quiet hours" />
    </Showcase>
  );
}
