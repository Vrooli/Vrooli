import type { ReactNode } from "react";
import { Button } from "../../../Button/versions/1.0.0/Button";
import { Popover, PopoverParts } from "./Popover";

function Showcase({
  children,
  eyebrow,
  title,
  detail,
}: {
  children: ReactNode;
  eyebrow: string;
  title: string;
  detail: string;
}) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-sm)",
        width: "min(100%, 620px)",
        minHeight: "360px",
        padding: "var(--space-lg)",
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
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          {eyebrow}
        </span>
        <strong style={{ font: "var(--text-title)" }}>{title}</strong>
        <span
          style={{
            maxWidth: "44ch",
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

function MenuContent({ showHint = true }: { showHint?: boolean }) {
  return (
    <div
      style={{
        display: "grid",
        gap: "var(--space-xs)",
        padding: "var(--space-sm)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-3xs)" }}>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-caption)",
          }}
        >
          Workspace
        </span>
        <strong style={{ font: "var(--text-subtitle)" }}>Design system</strong>
      </div>
      <div style={{ display: "grid", gap: "var(--space-3xs)" }}>
        <Button variant="secondary">Open workspace</Button>
        <Button variant="secondary">Invite a collaborator</Button>
      </div>
      {showHint ? (
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-caption)",
          }}
        >
          Press Escape to close
        </span>
      ) : null}
    </div>
  );
}

export function Default() {
  return (
    <Showcase
      eyebrow="Anchored surface"
      title="Keep context close to the action"
      detail="Anchored, collision-aware, keyboard-safe, and ready for small screens."
    >
      <Popover defaultOpen placement="bottom-start">
        <PopoverParts.Trigger aria-label="Open workspace menu">
          Workspace menu
        </PopoverParts.Trigger>
        <PopoverParts.Content
          aria-label="Workspace actions"
          initialFocus="none"
        >
          <MenuContent showHint={false} />
        </PopoverParts.Content>
      </Popover>
    </Showcase>
  );
}

export function Interactive() {
  return (
    <Showcase
      eyebrow="Layer-aware interaction"
      title="Dismissal should feel inevitable"
      detail="Open the menu, move through its actions, then click outside or press Escape. Focus returns to the trigger."
    >
      <Popover placement="bottom-start">
        <PopoverParts.Trigger aria-label="Open workspace menu">
          Workspace menu
        </PopoverParts.Trigger>
        <PopoverParts.Content aria-label="Workspace actions">
          <MenuContent />
        </PopoverParts.Content>
      </Popover>
    </Showcase>
  );
}
