/**
 * @libraryId react-component-library:BulkActionBar
 * @displayName BulkActionBar
 * @description
 * @version 1.0.4
 * @tags ["data-display","selection","async","recovery","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource data-display.bulk-action-bar */
import { useStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import { useId, useState, type CSSProperties, type ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2.0.0";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup/1.0.0";

export type BulkActionBarStatus =
  | "default"
  | "submitting"
  | "success"
  | "partial"
  | "request-error"
  | "retry";

export interface BulkActionProgress {
  completed: number;
  total: number;
  label?: string;
}

export interface BulkActionBarProps {
  selectedCount: number;
  totalCount?: number;
  status?: BulkActionBarStatus;
  defaultStatus?: BulkActionBarStatus;
  actionLabel?: string;
  retryLabel?: string;
  clearLabel?: string;
  selectAllLabel?: string;
  selectionLabel?: string;
  scopeLabel?: string;
  progress?: BulkActionProgress;
  failedItems?: string[];
  description?: ReactNode;
  successMessage?: ReactNode;
  errorMessage?: ReactNode;
  onAction?: () => void | Promise<void>;
  onRetry?: () => void | Promise<void>;
  onClear?: () => void;
  onSelectAll?: () => void;
  disabled?: boolean;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-bulk-action-bar] { display: grid; gap: var(--space-sm); min-inline-size: 0; padding: var(--space-sm) var(--space-md); border: var(--border-hairline) solid var(--color-border-strong); border-inline-start: var(--border-strong) solid var(--color-primary); border-radius: var(--radius-panel); background: linear-gradient(110deg, color-mix(in srgb, var(--color-primary) 10%, var(--color-surface-raised)), var(--color-surface-raised) 48%); color: var(--color-foreground); box-shadow: var(--elev-raised); }
[data-rcl-bulk-action-bar-header] { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-sm); min-inline-size: 0; flex-wrap: wrap; }
[data-rcl-bulk-action-bar-title] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
[data-rcl-bulk-action-bar-title] strong { overflow-wrap: anywhere; font: var(--text-label); }
[data-rcl-bulk-action-bar-title] span { color: var(--color-muted-foreground); font: var(--text-caption); overflow-wrap: anywhere; }
[data-rcl-bulk-action-bar-scope] { color: var(--color-primary); font: var(--text-caption); font-weight: 700; }
[data-rcl-bulk-action-bar-progress] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
[data-rcl-bulk-action-bar-progress] progress { inline-size: 100%; block-size: var(--space-2xs); accent-color: var(--color-primary); }
[data-rcl-bulk-action-bar-progress] output { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-bulk-action-bar-message] { margin: 0; color: var(--color-muted-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; }
[data-rcl-bulk-action-bar-status] { display: grid; gap: var(--space-3xs); min-inline-size: 0; padding: var(--space-xs) var(--space-sm); border-radius: var(--radius-control); background: color-mix(in srgb, var(--color-surface) 76%, transparent); color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-bulk-action-bar-status][data-tone="success"] { color: var(--color-success); }
[data-rcl-bulk-action-bar-status][data-tone="danger"] { border: var(--border-hairline) solid color-mix(in srgb, var(--color-danger) 32%, var(--color-border)); color: var(--color-danger); }
[data-rcl-bulk-action-bar-failures] { display: grid; gap: var(--space-3xs); margin: 0; padding-inline-start: var(--space-md); max-block-size: 8rem; overflow: auto; }
[data-rcl-bulk-action-bar-failures] li { overflow-wrap: anywhere; }
[data-rcl-bulk-action-bar-actions] { justify-content: end; }
[data-rcl-bulk-action-bar-actions] [data-control-slot="label"] { overflow: visible; text-overflow: clip; white-space: nowrap; }
@media (max-width: 38rem) { [data-rcl-bulk-action-bar] { padding: var(--space-sm); } [data-rcl-bulk-action-bar-header] { display: grid; } [data-rcl-bulk-action-bar-actions] { justify-content: stretch; } [data-rcl-bulk-action-bar-actions] > * { flex: 1 1 100%; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-bulk-action-bar] *, [data-rcl-bulk-action-bar] *::before, [data-rcl-bulk-action-bar] *::after { animation-duration: 0.01ms; transition-duration: 0.01ms; } }
@media (forced-colors: active) { [data-rcl-bulk-action-bar] { border-color: CanvasText; border-inline-start-color: Highlight; background: Canvas; color: CanvasText; box-shadow: none; } [data-rcl-bulk-action-bar-status] { border: var(--border-hairline) solid CanvasText; background: Canvas; color: CanvasText; } }
`;

function statusCopy(status: BulkActionBarStatus) {
  switch (status) {
    case "submitting":
      return {
        label: "Working through the selection",
        tone: "neutral" as const,
      };
    case "success":
      return { label: "All selected items updated", tone: "success" as const };
    case "partial":
      return { label: "Some items need attention", tone: "danger" as const };
    case "request-error":
      return { label: "Nothing was changed", tone: "danger" as const };
    case "retry":
      return {
        label: "Ready to retry the selection",
        tone: "neutral" as const,
      };
    default:
      return undefined;
  }
}

export const BulkActionBar = withClassName(function BulkActionBar({
  selectedCount,
  totalCount,
  status,
  defaultStatus = "default",
  actionLabel = "Archive selected",
  retryLabel = "Retry failed items",
  clearLabel = "Clear selection",
  selectAllLabel = "Select all",
  selectionLabel,
  scopeLabel,
  progress,
  failedItems = [],
  description,
  successMessage = "The selected records are now up to date.",
  errorMessage = "The request did not complete. Your selection is still available.",
  onAction,
  onRetry,
  onClear,
  onSelectAll,
  disabled = false,
  className,
  style,
}: BulkActionBarProps) {
  const strings = useStrings();
  const titleId = useId();
  const [localStatus, setLocalStatus] = useState(defaultStatus);
  const resolvedStatus = status ?? localStatus;
  const busy = disabled || resolvedStatus === "submitting";
  const hasSelection = selectedCount > 0;
  const allSelected = totalCount !== undefined && selectedCount >= totalCount;
  const statusState = statusCopy(resolvedStatus);
  const progressValue = progress
    ? Math.min(Math.max(progress.completed, 0), Math.max(progress.total, 1))
    : undefined;
  const run = async (kind: "action" | "retry") => {
    if (busy || !hasSelection) return;
    if (!status) setLocalStatus("submitting");
    try {
      await (kind === "action" ? onAction?.() : onRetry?.());
      if (!status) setLocalStatus("success");
    } catch {
      if (!status) setLocalStatus("request-error");
    }
  };

  return (
    <section
      data-testid="data-display.bulk-action-bar"
      data-rcl-bulk-action-bar
      data-status={resolvedStatus}
      aria-labelledby={titleId}
      aria-busy={resolvedStatus === "submitting"}
      className={className}
      style={style}
    >
      <style data-rcl-bulk-action-bar-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <header data-rcl-bulk-action-bar-header>
        <div data-rcl-bulk-action-bar-title>
          <strong id={titleId}>
            {selectionLabel ??
              `${selectedCount} ${selectedCount === 1 ? "item" : "items"} selected`}
          </strong>
          <span>
            {scopeLabel ??
              (allSelected
                ? `All ${totalCount} visible items are included.`
                : "Choose an action for this selection.")}
          </span>
        </div>
        {onSelectAll && totalCount !== undefined && !allSelected ? (
          <Button type="button" variant="ghost" size="sm" onClick={onSelectAll} disabled={busy}>
            {selectAllLabel}
          </Button>
        ) : null}
      </header>

      {description ? <p data-rcl-bulk-action-bar-message>{description}</p> : null}

      {progress && resolvedStatus === "submitting" ? (
        <div data-rcl-bulk-action-bar-progress>
          <progress
            max={Math.max(progress.total, 1)}
            value={progressValue}
            aria-label={progress.label ?? "Bulk action progress"}
          />
          <output aria-live="polite">
            {progress.label ?? `${progress.completed} of ${progress.total} complete`}
          </output>
        </div>
      ) : null}

      {statusState ? (
        <div
          data-rcl-bulk-action-bar-status
          data-tone={statusState.tone}
          role={statusState.tone === "danger" ? "alert" : "status"}
          aria-live="polite"
        >
          <strong>{statusState.label}</strong>
          {resolvedStatus === "success" ? <span>{successMessage}</span> : null}
          {resolvedStatus === "request-error" ? <span>{errorMessage}</span> : null}
          {resolvedStatus === "partial" && failedItems.length > 0 ? (
            <>
              <span>{failedItems.length} items were not updated:</span>
              <ul data-rcl-bulk-action-bar-failures>
                {failedItems.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </>
          ) : null}
        </div>
      ) : null}

      <ButtonGroup
        label={strings("data-display.bulk-action-bar.selection-actions", "Selection actions")}
        data-rcl-bulk-action-bar-actions
      >
        {resolvedStatus === "success" ? null : (
          <Button
            type="button"
            variant={resolvedStatus === "partial" ? "secondary" : "danger"}
            pending={resolvedStatus === "submitting"}
            pendingLabel="Updating selection…"
            disabled={busy || !hasSelection}
            onClick={() =>
              void run(
                resolvedStatus === "partial" ||
                  resolvedStatus === "retry" ||
                  resolvedStatus === "request-error"
                  ? "retry"
                  : "action",
              )
            }
          >
            {resolvedStatus === "partial" ||
            resolvedStatus === "retry" ||
            resolvedStatus === "request-error"
              ? retryLabel
              : actionLabel}
          </Button>
        )}
        {onClear && resolvedStatus !== "submitting" ? (
          <Button type="button" variant="ghost" disabled={disabled} onClick={onClear}>
            {clearLabel}
          </Button>
        ) : null}
      </ButtonGroup>
    </section>
  );
});
