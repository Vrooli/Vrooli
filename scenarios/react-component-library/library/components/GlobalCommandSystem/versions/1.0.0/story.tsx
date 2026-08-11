import { useMemo, useState, type ReactNode } from "react";
import { GlobalCommandSystem } from "./GlobalCommandSystem";
import type { Command } from "../../../../services/CommandRegistry/versions/1.0.0/CommandRegistry";

const frame = {
  display: "grid",
  gap: "var(--space-lg)",
  width: "min(100%, 760px)",
  minWidth: 0,
  boxSizing: "border-box",
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
} as const;

function Showcase({
  title,
  detail,
  children,
}: {
  title: string;
  detail: string;
  children: ReactNode;
}) {
  return (
    <section style={frame}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          Global command system
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

const commands: Command[] = [
  {
    id: "open-release",
    label: "Open release brief",
    description: "Jump to the latest release workspace.",
    group: "Navigate",
    keywords: ["release", "workspace", "brief"],
    shortcut: "G R",
    run: () => undefined,
  },
  {
    id: "create-note",
    label: "Create a planning note",
    description: "Capture a decision while the context is fresh.",
    group: "Create",
    keywords: ["note", "decision", "write"],
    shortcut: "N",
    run: () => undefined,
  },
  {
    id: "toggle-density",
    label: "Switch to comfortable density",
    description: "Give dense workspaces a little more breathing room.",
    group: "Preferences",
    keywords: ["density", "comfortable", "layout"],
    run: () => undefined,
  },
];

export function Default() {
  return (
    <Showcase
      title="One command language, everywhere"
      detail="Registration, shortcuts, discovery, and execution share one scoped source of truth."
    >
      <GlobalCommandSystem commands={commands} />
    </Showcase>
  );
}

export function OpenRecent() {
  const registryCommands = useMemo(
    () =>
      commands.map((command) =>
        command.id === "open-release"
          ? { ...command, run: () => undefined }
          : command,
      ),
    [],
  );
  return (
    <Showcase
      title="Returning users start ahead"
      detail="The palette opens focused with grouped actions and familiar shortcut hints already in reach."
    >
      <GlobalCommandSystem commands={registryCommands} defaultOpen />
    </Showcase>
  );
}

export function Interactive() {
  const [lastCommand, setLastCommand] = useState("No command run yet");
  return (
    <Showcase
      title="Keyboard-first execution"
      detail="Open the palette, search by intent, and run a command without leaving the current surface."
    >
      <GlobalCommandSystem
        commands={commands}
        onExecuted={(command) => setLastCommand(`Ran: ${command.label}`)}
      />
      <span role="status" aria-label="Last command">
        {lastCommand}
      </span>
    </Showcase>
  );
}

export function RequestError() {
  return (
    <Showcase
      title="Recovery without losing intent"
      detail="A failed command refresh is explicit and retryable; the surrounding work remains visible."
    >
      <GlobalCommandSystem
        commands={commands}
        defaultOpen
        status="request-error"
        errorMessage="The command index could not be refreshed. Retry when the connection is ready."
        onRetry={() => undefined}
      />
    </Showcase>
  );
}
