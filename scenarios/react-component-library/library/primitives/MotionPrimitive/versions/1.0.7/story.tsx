import { useState } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { MotionPrimitive, useMotionValue, type MotionScalar } from "./MotionPrimitive";

function MotionCard({ active }: { active: boolean }) {
  return (
    <MotionPrimitive
      active={active}
      variant="scale"
      duration="moderate"
      role="status"
      aria-label="Motion example"
      style={{
        display: "grid",
        gap: "var(--space-2xs)",
        padding: "var(--space-lg)",
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
        Tokenized transition
      </span>
      <strong style={{ font: "var(--text-subtitle)" }}>Motion that knows when to be quiet</strong>
      <span
        style={{
          color: "var(--color-muted-foreground)",
          font: "var(--text-body)",
        }}
      >
        Shared duration, easing, and reduced-motion policy without layout animation.
      </span>
    </MotionPrimitive>
  );
}

export function Default() {
  return <MotionCard active />;
}

export function Interactive() {
  const [active, setActive] = useState(true);
  const offset = useMotionValue<MotionScalar>("0px");
  const [shifted, setShifted] = useState(false);
  return (
    <section
      style={{
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
      <div style={{ display: "flex", flexWrap: "wrap", gap: "var(--space-xs)" }}>
        <Button onClick={() => setActive((value) => !value)}>
          {active ? "Hide card" : "Reveal card"}
        </Button>
        <Button
          variant="secondary"
          onClick={() => {
            const next = !shifted;
            setShifted(next);
            offset.set(next ? "var(--space-xl)" : "0px");
          }}
        >
          {shifted ? "Reset value" : "Set direct value"}
        </Button>
      </div>
      <MotionPrimitive
        active={active}
        variant="slide-up"
        motionValues={{ "--rcl-motion-demo-offset": offset }}
        style={{
          transform: "translateX(var(--rcl-motion-demo-offset, 0px))",
          display: "grid",
          gap: "var(--space-2xs)",
          padding: "var(--space-md)",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-control)",
          background: "var(--color-surface)",
        }}
        role="status"
        aria-label="Interactive motion value example"
      >
        <strong style={{ font: "var(--text-subtitle)" }}>Direct motion value</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          The offset writes to the DOM style subscription without React frame updates.
        </span>
      </MotionPrimitive>
    </section>
  );
}
