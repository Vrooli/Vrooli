import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import type { ReactNode } from "react";
import { Stat } from "./Stat";

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
            "data-display.stat.signal-at-a-glance",
            "Signal at a glance",
          )}
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

export function Default() {
  return (
    <Showcase
      title={resolveStrings(
        "data-display.stat.title",
        "Metrics with a point of view",
      )}
      detail="A stat is not just a number: it tells the user what to notice, how it is moving, and when it was measured."
    >
      <div
        style={{
          display: "grid",
          gridTemplateColumns:
            "repeat(auto-fit, minmax(min(100%, 13rem), 1fr))",
          gap: "var(--space-sm)",
        }}
      >
        <Stat
          label={resolveStrings("data-display.stat.label", "Active workspaces")}
          value="42"
          trend="+12.4%"
          trendTone="positive"
          caption="vs. last month"
        />
        <Stat
          label={resolveStrings(
            "data-display.stat.label.tasks-completed",
            "Tasks completed",
          )}
          value="186"
          trend="+8.1%"
          trendTone="positive"
          caption="this week"
        />
      </div>
    </Showcase>
  );
}

export function Negative() {
  return (
    <Showcase
      title={resolveStrings(
        "data-display.stat.title.a-downturn-should-be-legible-not-alarming",
        "A downturn should be legible, not alarming",
      )}
      detail="Semantic tone communicates direction while the neutral caption anchors the comparison in time."
    >
      <Stat
        label={resolveStrings(
          "data-display.stat.label.support-response",
          "Support response",
        )}
        value="2h 14m"
        trend="-4.8%"
        trendTone="negative"
        caption="vs. last week"
        icon="↘"
      />
    </Showcase>
  );
}

export function Loading() {
  return (
    <Showcase
      title={resolveStrings(
        "data-display.stat.title.reserve-the-geometry-before-the-number-arrives",
        "Reserve the geometry before the number arrives",
      )}
      detail="The loading state keeps the card footprint and rhythm stable while the data source catches up."
    >
      <Stat
        label={resolveStrings(
          "data-display.stat.label.pipeline-value",
          "Pipeline value",
        )}
        value="$—"
        loading
      />
    </Showcase>
  );
}
