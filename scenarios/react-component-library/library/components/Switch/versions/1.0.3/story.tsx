import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { ReactNode } from "react";
import { Switch } from "./Switch";

function Showcase({ children }: { children: ReactNode }) {
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
          {resolveStrings("controls.switch.immediate-setting", "Immediate setting")}
        </span>
        <strong style={{ font: "var(--text-title)" }}>{resolveStrings("controls.switch.notification-preferences", "Notification preferences")}</strong>
        <span
          style={{
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          {resolveStrings("controls.switch.use-a-switch-when-the-setting-takes-effect-immediately-and-can-be-understood-as-on-or-off", "Use a switch when the setting takes effect immediately and can be understood as on or off.")}
        </span>
      </div>
      {children}
    </section>
  );
}

export function Default() {
  return (
    <Showcase>
      <Switch label={resolveStrings("controls.switch.label", "Quiet hours")} description={resolveStrings("controls.switch.description", "Pause non-critical notifications after 6:00 PM.")} />
    </Showcase>
  );
}

export function Enabled() {
  return (
    <Showcase>
      <Switch
        label={resolveStrings("controls.switch.label.automatic-updates", "Automatic updates")}
        description={resolveStrings("controls.switch.description.keep-workspace-tools-current-overnight", "Keep workspace tools current overnight.")}
        defaultChecked
      />
    </Showcase>
  );
}
