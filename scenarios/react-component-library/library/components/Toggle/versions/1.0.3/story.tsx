import { useState, type ReactNode } from "react";
import { Toggle } from "./Toggle";

function Showcase({ children }: { children: ReactNode }) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 640px)",
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
          Pressed state
        </span>
        <strong style={{ font: "var(--text-title)" }}>Editor tools</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          A toggle is for a persistent choice that changes the current view or editing mode.
        </span>
      </div>
      {children}
    </section>
  );
}

export function Default() {
  return (
    <Showcase>
      <Toggle aria-label="Pin to dashboard">Pin to dashboard</Toggle>
    </Showcase>
  );
}

export function Interactive() {
  const [pressed, setPressed] = useState(false);
  return (
    <Showcase>
      <Toggle aria-label="Focus mode" pressed={pressed} onPressedChange={setPressed}>
        Focus mode
      </Toggle>
      <div
        role="status"
        aria-label="Toggle state"
        style={{
          color: "var(--color-muted-foreground)",
          font: "var(--text-caption)",
        }}
      >
        Focus mode is {pressed ? "on" : "off"}.
      </div>
    </Showcase>
  );
}

export function Pressed() {
  return (
    <Showcase>
      <Toggle aria-label="Show grid" defaultPressed>
        Show grid
      </Toggle>
    </Showcase>
  );
}

export function Disabled() {
  return (
    <Showcase>
      <Toggle aria-label="Auto-layout" defaultPressed disabled>
        Auto-layout
      </Toggle>
    </Showcase>
  );
}
