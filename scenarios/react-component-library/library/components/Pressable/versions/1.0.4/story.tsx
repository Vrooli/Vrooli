import type { ReactNode } from "react";
import { Pressable } from "./Pressable";

function Showcase({ children }: { children: ReactNode }) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 560px)",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background:
          "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 5%, var(--color-surface-raised)))",
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
          Interaction foundation
        </span>
        <strong style={{ font: "var(--text-title)" }}>One press contract</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
            maxWidth: "52ch",
          }}
        >
          Every control keeps its touch target, focus treatment, and geometry while acknowledging
          work in progress.
        </span>
      </div>
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          alignItems: "center",
          gap: "var(--space-sm)",
        }}
      >
        {children}
      </div>
    </section>
  );
}

export function Ready() {
  return (
    <Showcase>
      <Pressable tone="primary">Continue</Pressable>
      <Pressable tone="secondary">Save draft</Pressable>
    </Showcase>
  );
}

export function Pending() {
  return (
    <Showcase>
      <Pressable pending pendingLabel="Saving…">
        Save draft
      </Pressable>
      <Pressable aria-label="Voice input" size="icon" pending>
        {null}
      </Pressable>
    </Showcase>
  );
}
