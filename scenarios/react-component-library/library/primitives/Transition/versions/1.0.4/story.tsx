import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2.0.0";
import { Transition } from "./Transition";

export function Default() {
  return (
    <Transition
      present
      kind="scale"
      aria-label={useStrings("motion.transition.aria-label", "Transition example")}
    >
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
          {useStrings("motion.transition.shared-transition", "Shared transition")}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {useStrings("motion.transition.a-single-motion-grammar", "A single motion grammar")}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {useStrings(
            "motion.transition.fade-scale-slide-blur-and-crossfade-share-one-lifecycle-and-one-motion-policy",
            "Fade, scale, slide, blur, and crossfade share one lifecycle and one motion policy.",
          )}
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
        aria-label={useStrings(
          "motion.transition.aria-label.interactive-transition",
          "Interactive transition",
        )}
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
            {useStrings("motion.transition.interruptible-by-design", "Interruptible by design")}
          </strong>
          <span
            style={{
              color: "var(--color-muted-foreground)",
              font: "var(--text-body)",
            }}
          >
            {useStrings(
              "motion.transition.the-lifecycle-can-reverse-without-a-second-animation-implementation",
              "The lifecycle can reverse without a second animation implementation.",
            )}
          </span>
        </div>
      </Transition>
    </section>
  );
}
