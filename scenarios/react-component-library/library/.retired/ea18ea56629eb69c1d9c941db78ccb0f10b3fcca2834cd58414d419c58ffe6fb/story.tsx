import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useState, type ReactNode } from "react";
import { Progress } from "./Progress";

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
        display: "grid",
        gap: "var(--space-md)",
        maxInlineSize: 560,
        padding: "var(--space-xl)",
        border: "1px solid var(--color-border)",
        borderRadius: "var(--radius-panel)",
        background: "var(--color-surface-raised)",
        boxShadow: "var(--elev-raised)",
      }}
    >
      <div style={{ display: "grid", gap: "var(--space-2xs)" }}>
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

export function Determinate() {
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title",
        "Uploading your workspace",
      )}
      detail="The progress remains stable while the transfer completes."
    >
      <Progress
        value={68}
        label={resolveStrings("primitives.progress.label", "Workspace upload")}
      />
    </Showcase>
  );
}
export function Buffered() {
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title.preparing-playback",
        "Preparing playback",
      )}
      detail="Buffered work is visible without competing with the active position."
    >
      <Progress
        value={42}
        bufferedValue={76}
        mode="buffered"
        label={resolveStrings(
          "primitives.progress.label.playback-preparation",
          "Playback preparation",
        )}
      />
    </Showcase>
  );
}
export function Segmented() {
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title.step-3-of-5",
        "Step 3 of 5",
      )}
      detail="Segmented progress communicates discrete milestones at a glance."
    >
      <Progress
        value={60}
        mode="segmented"
        segments={5}
        label={resolveStrings(
          "primitives.progress.label.setup-progress",
          "Setup progress",
        )}
      />
    </Showcase>
  );
}
export function Indeterminate() {
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title.finding-the-best-match",
        "Finding the best match",
      )}
      detail="The system is working, but the completion time is not yet known."
    >
      <Progress
        mode="indeterminate"
        label={resolveStrings(
          "primitives.progress.label.searching",
          "Searching",
        )}
      />
    </Showcase>
  );
}
export function Circular() {
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title.sync-status",
        "Sync status",
      )}
      detail="A compact circular treatment works well beside a primary status."
    >
      <Progress
        shape="circular"
        value={82}
        label={resolveStrings(
          "primitives.progress.label.sync-progress",
          "Sync progress",
        )}
      />
    </Showcase>
  );
}
export function Success() {
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title.export-complete",
        "Export complete",
      )}
      detail="Success remains explicit through tone and text, not color alone."
    >
      <Progress
        value={100}
        tone="success"
        label={resolveStrings(
          "primitives.progress.label.export-complete",
          "Export complete",
        )}
      />
    </Showcase>
  );
}
export function Error() {
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title.export-paused",
        "Export paused",
      )}
      detail="The user can see exactly how much work is preserved before retrying."
    >
      <Progress
        value={54}
        tone="danger"
        label={resolveStrings(
          "primitives.progress.label.export-paused",
          "Export paused",
        )}
      />
    </Showcase>
  );
}

export function ToggleProgress() {
  const [value, setValue] = useState(42);
  return (
    <Showcase
      title={resolveStrings(
        "primitives.progress.title.a-living-status",
        "A living status",
      )}
      detail="Advance the task to verify stable geometry and motion."
    >
      <Progress
        value={value}
        label={resolveStrings(
          "primitives.progress.label.task-progress",
          "Task progress",
        )}
      />
      <button
        type="button"
        onClick={() =>
          setValue((current) => (current >= 100 ? 42 : current + 19))
        }
        style={{
          justifySelf: "start",
          minBlockSize: 44,
          border: 0,
          borderRadius: "var(--radius-control)",
          paddingInline: "var(--space-md)",
          background: "var(--color-primary)",
          color: "var(--color-primary-foreground)",
          font: "var(--text-label)",
        }}
      >
        {resolveStrings("primitives.progress.advance-task", "Advance task")}
      </button>
    </Showcase>
  );
}
