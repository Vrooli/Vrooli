import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState, type ReactNode } from "react";
import { RadioGroup } from "./RadioGroup";

function Showcase({ children }: { children: ReactNode }) {
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 660px)",
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
            "controls.radio-group.choice-architecture",
            "Choice architecture",
          )}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {resolveStrings("controls.radio-group.appearance", "Appearance")}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {resolveStrings(
            "controls.radio-group.choose-one-visual-mode-for-this-workspace-the-setting-is-remembered-for-your-next-visit",
            "Choose one visual mode for this workspace. The setting is remembered for your next visit.",
          )}
        </span>
      </div>
      {children}
    </section>
  );
}

const options = [
  {
    value: "system",
    label: "Use system setting",
    description: "Follow the appearance preference of this device.",
  },
  {
    value: "light",
    label: "Light",
    description: "A bright canvas for daytime work.",
  },
  {
    value: "dark",
    label: "Dark",
    description: "A low-glare canvas for focused sessions.",
  },
];

export function Default() {
  return (
    <Showcase>
      <RadioGroup
        label={resolveStrings("controls.radio-group.label", "Theme")}
        options={options}
        defaultValue="system"
      />
    </Showcase>
  );
}

export function Interactive() {
  const [value, setValue] = useState("system");
  return (
    <Showcase>
      <RadioGroup
        label={resolveStrings("controls.radio-group.label", "Theme")}
        options={options}
        value={value}
        onValueChange={setValue}
      />
      <div
        role="status"
        aria-label={resolveStrings(
          "controls.radio-group.aria-label",
          "Selected theme",
        )}
        style={{
          color: "var(--color-muted-foreground)",
          font: "var(--text-caption)",
        }}
      >
        Selected: {value}
      </div>
    </Showcase>
  );
}

export function Disabled() {
  return (
    <Showcase>
      <RadioGroup
        label={resolveStrings("controls.radio-group.label", "Theme")}
        description={resolveStrings(
          "controls.radio-group.description",
          "The organization administrator controls this setting.",
        )}
        options={options}
        defaultValue="system"
        disabled
      />
    </Showcase>
  );
}
