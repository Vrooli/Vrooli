/**
 * @libraryId react-component-library:UndoBanner
 * @displayName UndoBanner
 * @version 1.0.8
 * @tags ["feedback","recovery","undo","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource feedback.undo-banner */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useEffect, useState, type CSSProperties } from "react";
import { Presence } from "@vrooli/react-component-library/Presence/1";
import { Surface } from "@vrooli/react-component-library/Surface/1";
import { useUndoManager, type UndoRecord } from "@vrooli/react-component-library/UndoManager/1";

const styles = `
  [data-rcl-undo-viewport] { position: fixed; z-index: var(--layer-toast, 500); inset-inline: var(--space-lg); inset-block: auto calc(var(--space-lg) + env(safe-area-inset-bottom)); display: grid; justify-items: center; pointer-events: none; }
  [data-rcl-undo-banner] { pointer-events: auto; display: grid; grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; gap: var(--space-md); inline-size: min(100%, 38rem); padding: var(--space-md); border: var(--border-hairline) solid color-mix(in srgb, var(--color-primary) 24%, var(--color-border)); }
  [data-rcl-undo-banner][data-status="available"] { border-inline-start: var(--border-strong) solid var(--color-primary); }
  [data-rcl-undo-banner][data-status="submitting"] { border-inline-start: var(--border-strong) solid var(--color-warning); }
  [data-rcl-undo-banner][data-status="success"] { border-inline-start: var(--border-strong) solid var(--color-success); }
  [data-rcl-undo-banner][data-status="error"] { border-inline-start: var(--border-strong) solid var(--color-danger); }
  [data-rcl-undo-icon] { display: grid; place-items: center; inline-size: 2.5rem; block-size: 2.5rem; border-radius: var(--radius-pill); background: color-mix(in srgb, var(--color-primary) 12%, var(--color-surface)); color: var(--color-primary); }
  [data-rcl-undo-copy] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
  [data-rcl-undo-title] { color: var(--color-foreground); font: var(--text-label); }
  [data-rcl-undo-detail] { color: var(--color-muted-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; }
  [data-rcl-undo-action] { min-block-size: var(--tap-target-min); padding-inline: var(--space-md); border: 0; border-radius: var(--radius-control); background: var(--color-primary); color: var(--color-on-primary); cursor: pointer; font: var(--text-label); white-space: nowrap; }
  [data-rcl-undo-action]:hover { background: var(--color-primary-hover, color-mix(in srgb, var(--color-primary) 88%, var(--color-foreground))); transform: translateY(-1px); }
  [data-rcl-undo-action]:disabled { cursor: wait; opacity: .7; }
  [data-rcl-undo-dismiss] { align-self: start; inline-size: var(--tap-target-min); block-size: var(--tap-target-min); border: 0; border-radius: var(--radius-pill); background: transparent; color: var(--color-muted-foreground); cursor: pointer; font-size: 1.25rem; }
  [data-rcl-undo-dismiss]:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
  [data-rcl-undo-progress] { grid-column: 1 / -1; block-size: 3px; overflow: hidden; border-radius: var(--radius-pill); background: var(--color-surface-muted); }
  [data-rcl-undo-progress]::before { display: block; block-size: 100%; inline-size: calc(var(--rcl-undo-progress) * 100%); background: var(--color-primary); content: ""; transition: inline-size var(--dur-quick) linear; }
  @media (max-width: 38rem) { [data-rcl-undo-viewport] { inset-inline: var(--space-sm); } [data-rcl-undo-banner] { grid-template-columns: auto minmax(0, 1fr); gap: var(--space-sm); padding: var(--space-sm); } [data-rcl-undo-action] { grid-column: 1 / -1; inline-size: 100%; } [data-rcl-undo-dismiss] { grid-column: 2; grid-row: 1; justify-self: end; } }


`;

function UndoGlyph({ status }: { status: UndoRecord["status"] }) {
  if (status === "success") return <span aria-hidden="true">✓</span>;
  if (status === "error") return <span aria-hidden="true">!</span>;
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 24 24"
      width="22"
      height="22"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M9 14 4 9l5-5" />
      <path d="M4 9h10a6 6 0 0 1 6 6v1" />
    </svg>
  );
}

function UndoItem({
  record,
  onUndo,
  onDismiss,
}: {
  record: UndoRecord;
  onUndo: () => void;
  onDismiss: () => void;
}) {
  const strings = useStrings();
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (record.status !== "available") return undefined;
    const timer = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(timer);
  }, [record.status]);
  const progress = Math.max(
    0,
    Math.min(1, (record.expiresAt - now) / Math.max(record.expiresAt - record.createdAt, 1)),
  );
  const title =
    record.status === "success"
      ? (record.successMessage ?? "Change restored")
      : record.status === "error"
        ? "Undo could not be completed"
        : record.title;
  const detail =
    record.status === "error"
      ? (record.error ?? "Try again or dismiss this message.")
      : record.status === "submitting"
        ? "Restoring your change…"
        : record.status === "success"
          ? (record.successDetail ?? record.detail ?? "The previous state is back.")
          : (record.detail ?? "You can restore this change for a few seconds.");
  const liveRole = record.status === "error" ? "alert" : "status";
  return (
    <Presence present duration="quick">
      <Surface
        data-rcl-undo-banner
        data-status={record.status}
        elevation="floating"
        role={liveRole}
        aria-live={record.status === "error" ? "assertive" : "polite"}
        aria-atomic="true"
        aria-busy={record.status === "submitting" || undefined}
      >
        <span data-rcl-undo-icon>
          <UndoGlyph status={record.status} />
        </span>
        <div data-rcl-undo-copy>
          <span data-rcl-undo-title>{title}</span>
          <span data-rcl-undo-detail>{detail}</span>
        </div>
        {record.status === "available" && (
          <button
            data-testid="feedback.undo-banner"
            data-rcl-undo-action
            type="button"
            onClick={() => {
              onUndo();
            }}
          >
            {strings("feedback.undo-banner.undo", "Undo")}
          </button>
        )}
        {record.status === "submitting" && (
          <button data-testid="feedback.undo-banner" data-rcl-undo-action type="button" disabled>
            {strings("feedback.undo-banner.restoring", "Restoring…")}
          </button>
        )}
        {record.status === "error" && (
          <button
            data-testid="feedback.undo-banner"
            data-rcl-undo-action
            type="button"
            onClick={() => {
              onUndo();
            }}
          >
            {strings("feedback.undo-banner.retry-undo", "Retry undo")}
          </button>
        )}
        <button
          data-testid="feedback.undo-banner"
          data-rcl-undo-dismiss
          type="button"
          aria-label={strings("feedback.undo-banner.dismiss-undo-message", "Dismiss undo message")}
          onClick={onDismiss}
        >
          ×
        </button>
        {record.status === "available" && (
          <div
            data-rcl-undo-progress
            aria-hidden="true"
            style={{ "--rcl-undo-progress": progress } as CSSProperties}
          />
        )}
      </Surface>
    </Presence>
  );
}

export interface UndoBannerProps {
  className?: string;
  style?: CSSProperties;
}

export const UndoBanner = withClassName(function UndoBanner({ className, style }: UndoBannerProps) {
  const strings = useStrings();
  const manager = useUndoManager();
  const visible = manager.records.filter((record) => record.status !== "expired");
  return (
    <>
      <StyleSheet name="undobanner-1-0-4-1" css={styles} />
      <div
        data-rcl-undo-viewport
        className={className}
        style={style}
        aria-label={strings("feedback.undo-banner.undo-messages", "Undo messages")}
      >
        {visible.map((record) => (
          <UndoItem
            key={record.id}
            record={record}
            onUndo={() => {
              void (record.status === "error" ? manager.retry(record.id) : manager.undo(record.id));
            }}
            onDismiss={() => manager.dismiss(record.id)}
          />
        ))}
      </div>
    </>
  );
});
