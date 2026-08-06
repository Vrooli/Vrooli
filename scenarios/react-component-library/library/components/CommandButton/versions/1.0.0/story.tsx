import { useRef, type ReactNode } from "react";
import { CommandButton } from "./CommandButton";

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
        width: "min(100%, 520px)",
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
          Async command
        </span>
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

const pause = (duration: number) =>
  new Promise<void>((resolve) => setTimeout(resolve, duration));

export function Default() {
  return (
    <Showcase
      title="Acknowledge first. Finish gracefully."
      detail="The action confirms the press immediately, holds its footprint during work, and reports completion without shifting the toolbar."
    >
      <CommandButton action={async () => pause(900)}>
        Publish changes
      </CommandButton>
    </Showcase>
  );
}

export function FailureRecovery() {
  const attempts = useRef(0);
  return (
    <Showcase
      title="Recovery is part of the action"
      detail="A failed request leaves the same control in place and turns the next press into a clear retry."
    >
      <CommandButton
        action={async () => {
          attempts.current += 1;
          await pause(120);
          if (attempts.current === 1) throw new Error("Request failed");
        }}
      >
        Sync workspace
      </CommandButton>
    </Showcase>
  );
}

export function Success() {
  return (
    <Showcase
      title="A quiet success state"
      detail="Completion gets a brief, high-signal acknowledgement while preserving the same geometry and focus target."
    >
      <CommandButton action={() => Promise.resolve(undefined)}>
        Save preferences
      </CommandButton>
    </Showcase>
  );
}
