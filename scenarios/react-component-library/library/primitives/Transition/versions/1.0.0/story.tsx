import { useState } from "react";
import { Button } from "../../../../components/Button/versions/1.0.0/Button";
import { Transition } from "./Transition";

export function Default() {
  return (
    <Transition present kind="scale" aria-label="Transition example">
      <div
        role="status"
        style={{
          display: "grid",
          gap: "var(--space-2xs)",
          width: "min(100%, 480px)",
          padding: "var(--space-xl)",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-panel)",
          background: "var(--color-surface-raised)",
          boxShadow: "var(--elev-raised)",
        }}
      >
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Shared transition
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          A single motion grammar
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Fade, scale, slide, blur, and crossfade share one lifecycle and one
          motion policy.
        </span>
      </div>
    </Transition>
  );
}

export function Interactive() {
  const [present, setPresent] = useState(true);
  return (
    <section
      style={{
        display: "grid",
        gap: "var(--space-md)",
        width: "min(100%, 540px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <Button onClick={() => setPresent((value) => !value)}>
        {present ? "Hide transition" : "Show transition"}
      </Button>
      <Transition
        present={present}
        kind="slide"
        aria-label="Interactive transition"
      >
        <div
          role="status"
          style={{
            display: "grid",
            gap: "var(--space-2xs)",
            padding: "var(--space-md)",
            border: "1px solid var(--color-border)",
            borderRadius: "var(--radius-control)",
            background: "var(--color-surface)",
          }}
        >
          <strong style={{ font: "var(--text-subtitle)" }}>
            Interruptible by design
          </strong>
          <span
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-body)",
            }}
          >
            The lifecycle can reverse without a second animation implementation.
          </span>
        </div>
      </Transition>
    </section>
  );
}
