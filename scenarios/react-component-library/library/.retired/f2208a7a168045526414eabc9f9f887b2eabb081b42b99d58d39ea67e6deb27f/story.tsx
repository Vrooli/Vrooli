import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState, type ReactNode } from "react";
import { Toolbar, type ToolbarItem } from "./Toolbar";

function Showcase({ children }: { children: ReactNode }) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 760px)",
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
          {resolveStrings(
            "controls.toolbar.command-surface",
            "Command surface",
          )}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {resolveStrings("controls.toolbar.editor-toolbar", "Editor toolbar")}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {resolveStrings(
            "controls.toolbar.actions-stay-grouped-keyboardable-and-usable-when-the-available-width-gets-smaller",
            "Actions stay grouped, keyboardable, and usable when the available width gets smaller.",
          )}
        </span>
      </div>
      {children}
    </section>
  );
}

function items(setStatus: (value: string) => void): ToolbarItem[] {
  return [
    {
      id: "bold",
      label: "Bold",
      kind: "toggle",
      onSelect: () => setStatus("Bold"),
    },
    {
      id: "italic",
      label: "Italic",
      kind: "toggle",
      onSelect: () => setStatus("Italic"),
    },
    {
      id: "link",
      label: "Insert link",
      variant: "ghost",
      onSelect: () => setStatus("Insert link"),
    },
    { id: "comment", label: "Comment", disabled: true },
  ];
}

export function Default() {
  return (
    <Showcase>
      <Toolbar
        label={resolveStrings("controls.toolbar.label", "Editor toolbar")}
        items={items(() => {})}
      />
    </Showcase>
  );
}

export function Interactive() {
  const [status, setStatus] = useState("No action yet");
  return (
    <Showcase>
      <Toolbar
        label={resolveStrings("controls.toolbar.label", "Editor toolbar")}
        items={items(setStatus)}
      />
      <div
        role="status"
        aria-label={resolveStrings(
          "controls.toolbar.aria-label",
          "Last toolbar action",
        )}
        style={{
          color: "var(--color-muted-foreground)",
          font: "var(--text-caption)",
        }}
      >
        Last action: {status}
      </div>
    </Showcase>
  );
}

export function Vertical() {
  return (
    <Showcase>
      <Toolbar
        label={resolveStrings("controls.toolbar.label", "Editor toolbar")}
        orientation="vertical"
        items={items(() => {})}
      />
    </Showcase>
  );
}
