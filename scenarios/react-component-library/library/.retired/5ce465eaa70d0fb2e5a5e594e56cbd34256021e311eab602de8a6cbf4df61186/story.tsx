import type { ReactNode } from "react";
import { Tabs } from "./Tabs";

function Showcase({ children }: { children: ReactNode }) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 720px)",
        minHeight: "340px",
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background:
          "linear-gradient(145deg, var(--color-surface-raised), color-mix(in srgb, var(--color-primary) 4%, var(--color-surface-raised)))",
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
          Navigation pattern
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          Workspace overview
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
            maxWidth: "58ch",
          }}
        >
          Move between related views without losing the context of the current
          workspace.
        </span>
      </div>
      <div
        style={{
          display: "grid",
          gap: "var(--space-md)",
          padding: "var(--space-md)",
          border: "1px solid var(--color-border)",
          borderRadius: "var(--radius-panel)",
          background:
            "color-mix(in srgb, var(--color-surface) 82%, transparent)",
        }}
      >
        {children}
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-caption)",
          }}
        >
          Arrow keys move between tabs · Home and End jump to the edges
        </span>
      </div>
    </section>
  );
}

function TabsSpecimen({
  items,
  panels,
  defaultActive,
}: {
  items: string[];
  panels: Record<string, ReactNode>;
  defaultActive?: string;
}) {
  return (
    <Showcase>
      <Tabs
        items={items}
        defaultActive={defaultActive}
        panels={panels}
        ariaLabel="Workspace sections"
      />
    </Showcase>
  );
}

export function Default() {
  return (
    <TabsSpecimen
      items={["Overview", "Activity"]}
      panels={{
        Overview: "A focused overview of the workspace.",
        Activity: "Recent activity appears here.",
      }}
      defaultActive="Overview"
    />
  );
}

export function LongContent() {
  return (
    <TabsSpecimen
      items={[
        "Overview",
        "Activity",
        "Automations",
        "Integrations",
        "Preferences",
      ]}
      panels={{
        Overview: "A focused overview of the workspace.",
        Activity: "Recent activity appears here.",
        Automations: "Automation history.",
        Integrations: "Connected integrations.",
        Preferences: "Workspace preferences.",
      }}
      defaultActive="Overview"
    />
  );
}
