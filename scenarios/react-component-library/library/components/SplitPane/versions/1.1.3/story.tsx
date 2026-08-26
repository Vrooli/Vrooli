import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { SplitPane } from "./SplitPane";

const frame = {
  display: "grid",
  gap: "var(--space-lg)",
  width: "min(100%, 1040px)",
  minWidth: 0,
  boxSizing: "border-box" as const,
  padding: "var(--space-xl)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-panel)",
  background: "var(--color-surface-raised)",
  boxShadow: "var(--elev-raised)",
};

const pane = {
  display: "grid",
  gap: "var(--space-sm)",
  minWidth: 0,
  padding: "var(--space-lg)",
  border: "var(--border-hairline) solid var(--color-border)",
  borderRadius: "var(--radius-control)",
  background: "var(--color-surface-muted)",
};

function Pane({
  eyebrow,
  title,
  detail,
  value,
}: {
  eyebrow: string;
  title: string;
  detail: string;
  value: string;
}) {
  return (
    <section style={pane}>
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
      <h2 style={{ margin: 0, font: "var(--text-title)" }}>{title}</h2>
      <p
        style={{
          margin: 0,
          color: "var(--color-muted-foreground)",
          font: "var(--text-body)",
        }}
      >
        {detail}
      </p>
      <strong style={{ color: "var(--color-primary)", font: "var(--text-title)" }}>{value}</strong>
    </section>
  );
}

export function Default() {
  return (
    <div style={frame}>
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
        <span
          style={{
            color: "var(--color-primary)",
            font: "var(--text-overline)",
            letterSpacing: ".1em",
            textTransform: "uppercase",
          }}
        >
          {resolveStrings("manipulation.split-pane.operational-workspace", "Operational workspace")}
        </span>
        <h1 style={{ margin: 0, font: "var(--text-heading)" }}>{resolveStrings("manipulation.split-pane.keep-both-decisions-in-view", "Keep both decisions in view")}</h1>
        <p
          style={{
            margin: 0,
            color: "var(--color-muted-foreground)",
            font: "var(--text-body)",
          }}
        >
          Related work stays adjacent on wide screens and stacks without losing its context on
          compact screens.
        </p>
      </div>
      <SplitPane
        primary={
          <Pane
            eyebrow="Primary queue"
            title={resolveStrings("manipulation.split-pane.title", "Needs review")}
            detail="Three items are waiting for an owner."
            value="03"
          />
        }
        secondary={
          <Pane
            eyebrow="Secondary queue"
            title={resolveStrings("manipulation.split-pane.title.ready-to-ship", "Ready to ship")}
            detail="The next handoff is prepared for release."
            value="08"
          />
        }
      />
    </div>
  );
}
