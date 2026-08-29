import { useStrings } from "@vrooli/react-component-library/useLocale/1.1.0";
import type { ReactNode } from "react";
import { List, type ListItem } from "./List";

function Showcase({
  children,
  title,
  detail,
}: {
  children: ReactNode;
  title: string;
  detail: string;
}) {
  const libraryStrings = useStrings();
  return (
    <section
      style={{
        boxSizing: "border-box",
        display: "grid",
        gap: "var(--space-lg)",
        width: "min(100%, 720px)",
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
          {libraryStrings("data-display.list.activity-stream", "Activity stream")}
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

const activity: ListItem[] = [
  {
    id: "deploy",
    title: "Deploy completed",
    description: "Production · web-console",
    meta: "2m ago",
    tone: "success",
  },
  {
    id: "review",
    title: "A descriptive activity with a longer label that still needs to wrap gracefully",
    description: "Design system review · 3 comments",
    meta: "18m ago",
    tone: "warning",
  },
  {
    id: "backup",
    title: "Backup queued",
    description: "Workspace archive",
    meta: "1h ago",
  },
];

export function Default() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings("data-display.list.title", "A list with hierarchy, not just rows")}
      detail="Each item has a primary action cue, supporting context, and a time marker that remains readable on narrow screens."
    >
      <List
        title={libraryStrings("data-display.list.title.recent-activity", "Recent activity")}
        description={libraryStrings(
          "data-display.list.description",
          "Changes across the design workspace",
        )}
        items={activity}
        label={libraryStrings("data-display.list.label", "Recent activity")}
      />
    </Showcase>
  );
}

export function LongContent() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "data-display.list.title.long-labels-are-part-of-the-product",
        "Long labels are part of the product",
      )}
      detail="The row gives primary content room, lets metadata move below it on mobile, and never relies on truncation to hide meaning."
    >
      <List
        items={activity.slice(1)}
        label={libraryStrings("data-display.list.label.review-activity", "Review activity")}
      />
    </Showcase>
  );
}

export function Empty() {
  const libraryStrings = useStrings();
  return (
    <Showcase
      title={libraryStrings(
        "data-display.list.title.an-empty-list-still-explains-itself",
        "An empty list still explains itself",
      )}
      detail="Empty space becomes a useful state with calm copy and a bounded footprint."
    >
      <List
        title={libraryStrings("data-display.list.title.recent-activity", "Recent activity")}
        empty="No activity has been recorded yet."
        label={libraryStrings("data-display.list.label.recent-activity", "Recent activity")}
      />
    </Showcase>
  );
}
