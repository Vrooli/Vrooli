/**
 * @libraryId react-component-library:ConflictResolutionFlow
 * @displayName ConflictResolutionFlow
 * @description The concurrency workflow comparing local and remote change, explaining what conflicts, supporting field-level resolution, and retrying against the current version.
 * @version 1.0.11
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:ConflictResolutionFlow
 * @vrooliComponentSourceSlot patterns.conflict-resolution-flow */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import { useCallback, useMemo, useState, type CSSProperties, type ReactNode } from "react";
import { DiffViewer } from "@vrooli/react-component-library/DiffViewer/1";
import { Form } from "@vrooli/react-component-library/Form/1";
import { useAnnounce } from "@vrooli/react-component-library/useAnnounce/1";

export interface ConflictField<T = unknown> {
  id: string;
  label: string;
  local: T;
  remote: T;
  format?: (value: T) => string;
  description?: ReactNode;
}

export type ConflictResolutionStatus =
  | "default"
  | "loading"
  | "submitting"
  | "success"
  | "request-error"
  | "retry";

export interface ConflictResolutionFlowProps<T = unknown> {
  fields: ConflictField<T>[];
  status?: ConflictResolutionStatus;
  onResolve?: (values: Record<string, T>) => void | Promise<void>;
  onRetry?: () => void | Promise<void>;
  onCancel?: () => void;
  title?: ReactNode;
  description?: ReactNode;
  resolveLabel?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-conflict-flow] { display: grid; gap: var(--space-md); min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-conflict-flow-header] { display: grid; gap: var(--space-2xs); }
[data-rcl-conflict-flow-title] { font: var(--text-title); }
[data-rcl-conflict-flow-description] { max-inline-size: 66ch; color: var(--color-muted-foreground); font: var(--text-body); }
[data-rcl-conflict-flow-list] { display: grid; gap: var(--space-sm); }
[data-rcl-conflict-flow-field] { display: grid; gap: var(--space-xs); min-inline-size: 0; padding: var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); }
[data-rcl-conflict-flow-field] legend { padding-inline: var(--space-2xs); font: var(--text-label); }
[data-rcl-conflict-flow-field-description] { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-conflict-flow-options] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-xs); }
[data-rcl-conflict-flow-option] { display: grid; gap: var(--space-3xs); min-block-size: var(--tap-target-min); padding: var(--space-xs); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-muted); color: inherit; text-align: start; cursor: pointer; font: var(--text-label); transition: border-color var(--dur-quick) var(--ease-standard), background var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-conflict-flow-option]:hover { border-color: var(--color-primary); transform: translateY(-1px); }
[data-rcl-conflict-flow-option][aria-pressed="true"] { border-color: var(--color-primary); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-surface-raised)); box-shadow: inset 0 0 0 1px var(--color-primary); }
[data-rcl-conflict-flow-option-label] { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-conflict-flow-option-value] { overflow-wrap: anywhere; }
[data-rcl-conflict-flow-status] { min-block-size: 1.35rem; color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-conflict-flow-status="error"] { color: var(--color-danger); }
[data-rcl-conflict-flow-status="success"] { color: var(--color-success); }
[data-rcl-conflict-flow-actions] { display: flex; flex-wrap: wrap; gap: var(--space-xs); align-items: center; }
[data-rcl-conflict-flow-actions] button { min-block-size: var(--tap-target-min); padding: var(--space-2xs) var(--space-sm); border: var(--border-hairline) solid currentColor; border-radius: var(--radius-control); background: transparent; color: var(--color-primary); cursor: pointer; font: var(--text-label); }
[data-rcl-conflict-flow-actions] button[type="submit"] { background: var(--color-primary); color: var(--color-on-primary); }
[data-rcl-conflict-flow-actions] button:disabled { cursor: not-allowed; opacity: .52; }
@media (max-width: 38rem) { [data-rcl-conflict-flow-options] { grid-template-columns: 1fr; } [data-rcl-conflict-flow-actions] button { flex: 1 1 12rem; } }


`;

function printable<T>(field: ConflictField<T>, value: T) {
  return field.format
    ? field.format(value)
    : typeof value === "string"
      ? value
      : JSON.stringify(value);
}

export const ConflictResolutionFlow = withClassName(function ConflictResolutionFlow<T>({
  fields,
  status,
  onResolve,
  onRetry,
  onCancel,
  title,
  description,
  resolveLabel = "Save resolved version",
  className,
  style,
}: ConflictResolutionFlowProps<T>) {
  const libraryStrings = useStrings();
  description =
    description ??
    libraryStrings(
      "patterns.conflict-resolution-flow.another-version-changed-while-you-were-working-c",
      "Another version changed while you were working. Choose the value that should survive for each field.",
    );
  title =
    title ??
    libraryStrings(
      "patterns.conflict-resolution-flow.resolve-concurrent-changes",
      "Resolve concurrent changes",
    );
  const announce = useAnnounce();
  const [choices, setChoices] = useState<Record<string, "local" | "remote">>(() =>
    Object.fromEntries(fields.map((field) => [field.id, "local"])),
  );
  const [localStatus, setLocalStatus] = useState<ConflictResolutionStatus>("default");
  const resolvedStatus = status ?? localStatus;
  const values = useMemo(
    () =>
      Object.fromEntries(
        fields.map((field) => [
          field.id,
          choices[field.id] === "remote" ? field.remote : field.local,
        ]),
      ) as Record<string, T>,
    [choices, fields],
  );

  const choose = useCallback(
    (field: ConflictField<T>, choice: "local" | "remote") => {
      setChoices((current) => ({ ...current, [field.id]: choice }));
      announce(
        `${field.label}: ${choice === "local" ? "your version" : "the remote version"} selected.`,
      );
    },
    [announce],
  );

  const submit = useCallback(async () => {
    setLocalStatus("submitting");
    announce("Submitting conflict resolution.");
    try {
      await onResolve?.(values);
      setLocalStatus("success");
      announce("Conflict resolved and saved.");
    } catch {
      setLocalStatus("request-error");
      announce("Conflict resolution failed. Your choices are preserved for retry.", {
        priority: "assertive",
      });
    }
  }, [announce, onResolve, values]);

  const retry = useCallback(async () => {
    setLocalStatus("submitting");
    try {
      await onRetry?.();
      await submit();
    } catch {
      setLocalStatus("request-error");
    }
  }, [onRetry, submit]);

  const busy = resolvedStatus === "loading" || resolvedStatus === "submitting";
  const statusText =
    resolvedStatus === "loading"
      ? "Checking the current version…"
      : resolvedStatus === "submitting"
        ? "Saving your resolution…"
        : resolvedStatus === "success"
          ? "Conflict resolved and saved."
          : resolvedStatus === "request-error" || resolvedStatus === "retry"
            ? "The current version changed again. Your choices are still here; retry when ready."
            : "Your choices are kept on this device until you save.";

  return (
    <div data-rcl-conflict-flow className={className} style={style}>
      <StyleSheet
        libraryId="react-component-library:ConflictResolutionFlow"
        version="1.0.10"
        css={styles}
      />
      <Form
        title={<span data-rcl-conflict-flow-title>{title}</span>}
        description={<span data-rcl-conflict-flow-description>{description}</span>}
        aria-label={libraryStrings(
          "patterns.conflict-resolution-flow.conflict-resolution-form-onsubmit-void-submit-fo",
          "Conflict resolution form",
        )}
        onSubmit={() => void submit()}
        footer={
          <div data-rcl-conflict-flow-actions>
            <button
              data-testid="patterns.conflict-resolution-flow"
              type="button"
              onClick={onCancel}
              disabled={busy}
            >
              {libraryStrings("patterns.conflict-resolution-flow.keep-editing", "Keep editing")}
            </button>
            {resolvedStatus === "request-error" || resolvedStatus === "retry" ? (
              <button
                data-testid="patterns.conflict-resolution-flow"
                type="button"
                onClick={() => void retry()}
                disabled={busy}
              >
                {libraryStrings(
                  "patterns.conflict-resolution-flow.retry-resolution",
                  "Retry resolution",
                )}
              </button>
            ) : null}
            <button
              data-testid="patterns.conflict-resolution-flow"
              type="submit"
              disabled={busy || fields.length === 0}
            >
              {busy ? "Saving…" : resolveLabel}
            </button>
          </div>
        }
      >
        <div data-rcl-conflict-flow-list>
          {resolvedStatus === "loading" ? (
            <div data-rcl-conflict-flow-status role="status">
              {libraryStrings(
                "patterns.conflict-resolution-flow.checking-the-latest-version",
                "Checking the latest version…",
              )}
            </div>
          ) : null}
          {fields.map((field) => {
            const selected = choices[field.id] ?? "local";
            return (
              <fieldset data-rcl-conflict-flow-field key={field.id} disabled={busy}>
                <legend>{field.label}</legend>
                {field.description ? (
                  <div data-rcl-conflict-flow-field-description>{field.description}</div>
                ) : null}
                <DiffViewer
                  before={printable(field, field.local)}
                  after={printable(field, field.remote)}
                />
                <div
                  data-rcl-conflict-flow-options
                  role="group"
                  aria-label={`${field.label} resolution`}
                >
                  {(["local", "remote"] as const).map((choice) => (
                    <button
                      data-testid="patterns.conflict-resolution-flow"
                      key={choice}
                      type="button"
                      data-rcl-conflict-flow-option
                      aria-label={`${choice === "local" ? "Your version" : "Remote version"} for ${field.label}`}
                      aria-pressed={selected === choice}
                      onClick={() => choose(field, choice)}
                    >
                      <span data-rcl-conflict-flow-option-label>
                        {choice === "local" ? "Your version" : "Remote version"}
                      </span>
                      <span data-rcl-conflict-flow-option-value>
                        {printable(field, choice === "local" ? field.local : field.remote)}
                      </span>
                    </button>
                  ))}
                </div>
              </fieldset>
            );
          })}
        </div>
        <div
          data-rcl-conflict-flow-status={
            resolvedStatus === "request-error" || resolvedStatus === "retry"
              ? "error"
              : resolvedStatus === "success"
                ? "success"
                : undefined
          }
          role={resolvedStatus === "request-error" ? "alert" : "status"}
          aria-live="polite"
        >
          {statusText}
        </div>
      </Form>
    </div>
  );
});
