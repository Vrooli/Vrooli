import { useStrings } from "@vrooli/react-component-library/useLocale/1.1.0";
import type { ReactNode } from "react";
import { Switch } from "./Switch";

function Showcase({ children }: { children: ReactNode }) {
  const libraryStrings = useStrings();
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
          {libraryStrings(
            "controls.switch.immediate-setting",
            "Immediate setting",
          )}
        </span>
        <strong style={{ font: "var(--text-title)" }}>
          {libraryStrings(
            "controls.switch.notification-preferences",
            "Notification preferences",
          )}
        </strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {libraryStrings(
            "controls.switch.use-a-switch-when-the-setting-takes-effect-immediately-and-can-be-understood-as-on-or-off",
            "Use a switch when the setting takes effect immediately and can be understood as on or off.",
          )}
        </span>
      </div>
      {children}
    </section>
  );
}

export function Default() {
  const libraryStrings = useStrings();
  return (
    <Showcase>
      <Switch
        label={libraryStrings("controls.switch.label", "Quiet hours")}
        description={libraryStrings(
          "controls.switch.description",
          "Pause non-critical notifications after 6:00 PM.",
        )}
      />
    </Showcase>
  );
}

export function Enabled() {
  const libraryStrings = useStrings();
  return (
    <Showcase>
      <Switch
        label={libraryStrings(
          "controls.switch.label.automatic-updates",
          "Automatic updates",
        )}
        description={libraryStrings(
          "controls.switch.description.keep-workspace-tools-current-overnight",
          "Keep workspace tools current overnight.",
        )}
        defaultChecked
      />
    </Showcase>
  );
}

/**
 * The settings-row shape: the host row already renders the label and hint, so
 * the switch is bare and takes its accessible name from `aria-label`.
 */
export function Bare() {
  const libraryStrings = useStrings();
  const label = libraryStrings(
    "controls.switch.label.adaptive-chrome",
    "Adaptive chrome",
  );
  return (
    <Showcase>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          gap: "var(--space-sm)",
        }}
      >
        <span style={{ font: "var(--text-body)", fontWeight: 650 }}>
          {label}
        </span>
        <Switch aria-label={label} defaultChecked />
      </div>
    </Showcase>
  );
}

export function TrailingLabel() {
  const libraryStrings = useStrings();
  return (
    <Showcase>
      <Switch
        labelPlacement="end"
        label={libraryStrings(
          "controls.switch.label.adaptive-chrome",
          "Adaptive chrome",
        )}
        description={libraryStrings(
          "controls.switch.description.tint-the-chrome-to-match-the-focused-terminal",
          "Tint the chrome to match the focused terminal.",
        )}
        defaultChecked
      />
    </Showcase>
  );
}
