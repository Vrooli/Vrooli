import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useState, type ReactNode } from "react";
import { RadioGroup } from "./RadioGroup";

function Showcase({ children }: { children: ReactNode }) {
  const libraryStrings = useStrings();
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
          {libraryStrings("controls.radio-group.choice-architecture", "Choice architecture")}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {libraryStrings("controls.radio-group.appearance", "Appearance")}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {libraryStrings(
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
  const libraryStrings = useStrings();
  return (
    <Showcase>
      <RadioGroup
        label={libraryStrings("controls.radio-group.label", "Theme")}
        options={options}
        defaultValue="system"
      />
    </Showcase>
  );
}

export function Interactive() {
  const libraryStrings = useStrings();
  const [value, setValue] = useState("system");
  return (
    <Showcase>
      <RadioGroup
        label={libraryStrings("controls.radio-group.label", "Theme")}
        options={options}
        value={value}
        onValueChange={setValue}
      />
      <div
        role="status"
        aria-label={libraryStrings("controls.radio-group.aria-label", "Selected theme")}
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
  const libraryStrings = useStrings();
  return (
    <Showcase>
      <RadioGroup
        label={libraryStrings("controls.radio-group.label", "Theme")}
        description={libraryStrings(
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

const postures = [
  {
    value: "read",
    label: "Read only",
    description: "Read status and telemetry.",
    badge: <span data-testid="story-card-badge">Chosen</span>,
  },
  {
    value: "operate",
    label: "Operate",
    description: "Read and operate governed commands.",
    testId: "story-option-operate",
  },
  {
    value: "full",
    label: "Full control",
    description: "Read, operate, and destructive commands.",
  },
];

/**
 * A choice with a sentence of consequence attached. The whole surface is the
 * target, and the badge rides inside the label rather than beside it.
 */
export function Cards() {
  const [value, setValue] = useState("read");
  return (
    <Showcase>
      <RadioGroup
        label="What should this machine be allowed to do?"
        description="You can change this at any time."
        variant="card"
        options={postures}
        value={value}
        onValueChange={setValue}
      />
    </Showcase>
  );
}
