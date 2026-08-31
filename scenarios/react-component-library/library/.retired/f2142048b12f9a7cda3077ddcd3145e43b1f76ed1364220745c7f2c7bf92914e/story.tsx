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
      title="Uploading your workspace"
      detail="The progress remains stable while the transfer completes."
    >
      <Progress value={68} label="Workspace upload" />
    </Showcase>
  );
}
export function Buffered() {
  return (
    <Showcase
      title="Preparing playback"
      detail="Buffered work is visible without competing with the active position."
    >
      <Progress
        value={42}
        bufferedValue={76}
        mode="buffered"
        label="Playback preparation"
      />
    </Showcase>
  );
}
export function Segmented() {
  return (
    <Showcase
      title="Step 3 of 5"
      detail="Segmented progress communicates discrete milestones at a glance."
    >
      <Progress
        value={60}
        mode="segmented"
        segments={5}
        label="Setup progress"
      />
    </Showcase>
  );
}
export function Indeterminate() {
  return (
    <Showcase
      title="Finding the best match"
      detail="The system is working, but the completion time is not yet known."
    >
      <Progress mode="indeterminate" label="Searching" />
    </Showcase>
  );
}
export function Circular() {
  return (
    <Showcase
      title="Sync status"
      detail="A compact circular treatment works well beside a primary status."
    >
      <Progress shape="circular" value={82} label="Sync progress" />
    </Showcase>
  );
}
export function Success() {
  return (
    <Showcase
      title="Export complete"
      detail="Success remains explicit through tone and text, not color alone."
    >
      <Progress value={100} tone="success" label="Export complete" />
    </Showcase>
  );
}
export function Error() {
  return (
    <Showcase
      title="Export paused"
      detail="The user can see exactly how much work is preserved before retrying."
    >
      <Progress value={54} tone="danger" label="Export paused" />
    </Showcase>
  );
}

export function ToggleProgress() {
  const [value, setValue] = useState(42);
  return (
    <Showcase
      title="A living status"
      detail="Advance the task to verify stable geometry and motion."
    >
      <Progress value={value} label="Task progress" />
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
        Advance task
      </button>
    </Showcase>
  );
}
