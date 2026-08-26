import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { ReactNode } from "react";
import { LoadingState } from "./LoadingState";

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
          {useStrings("feedback.loading-state.progressive-feedback", "Progressive feedback")}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {useStrings(
            "feedback.loading-state.keep-the-shape-while-work-happens",
            "Keep the shape while work happens",
          )}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {useStrings(
            "feedback.loading-state.a-brief-wait-should-feel-intentional-not-like-the-product-disappeared",
            "A brief wait should feel intentional, not like the product disappeared.",
          )}
        </span>
      </div>
      {children}
    </section>
  );
}

export function Loading() {
  return (
    <Showcase>
      <LoadingState
        label={useStrings("feedback.loading-state.label", "Loading workspace")}
        detail="Preparing your latest activity."
      />
    </Showcase>
  );
}
