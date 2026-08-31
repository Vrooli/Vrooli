/** @vrooliComponentSource patterns.conflict-resolution-flow */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import {
  useCallback,
  useMemo,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { DiffViewer } from "../../../DiffViewer/versions/1.0.0/DiffViewer";
import { Form } from "../../../Form/versions/1.0.0/Form";
import { useAnnounce } from "../../../../hooks/useAnnounce/versions/1.0.0/useAnnounce";

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
[data-rcl-conflict-flow] { display: grid; gap: var(--space-md, 1rem); min-inline-size: 0; color: var(--color-foreground, #0f172a); }
[data-rcl-conflict-flow-header] { display: grid; gap: var(--space-2xs, .35rem); }
[data-rcl-conflict-flow-title] { font: var(--text-title, 700 1.2rem/1.3 system-ui, sans-serif); }
[data-rcl-conflict-flow-description] { max-inline-size: 66ch; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .9rem/1.45 system-ui, sans-serif); }
[data-rcl-conflict-flow-list] { display: grid; gap: var(--space-sm, .75rem); }
[data-rcl-conflict-flow-field] { display: grid; gap: var(--space-xs, .625rem); min-inline-size: 0; padding: var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 1rem); background: var(--color-surface-raised, #fff); }
[data-rcl-conflict-flow-field] legend { padding-inline: var(--space-2xs, .35rem); font: var(--text-label, 650 .9rem/1.35 system-ui, sans-serif); }
[data-rcl-conflict-flow-field-description] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
[data-rcl-conflict-flow-options] { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: var(--space-xs, .625rem); }
[data-rcl-conflict-flow-option] { display: grid; gap: var(--space-3xs, .2rem); min-block-size: var(--tap-target-min, 44px); padding: var(--space-xs, .625rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .625rem); background: var(--color-surface-muted, #f1f5f9); color: inherit; text-align: start; cursor: pointer; font: var(--text-label, 650 .9rem/1.35 system-ui, sans-serif); transition: border-color var(--dur-quick, 160ms) var(--ease-standard, ease), background var(--dur-quick, 160ms) var(--ease-standard, ease), transform var(--dur-quick, 160ms) var(--ease-standard, ease); }
[data-rcl-conflict-flow-option]:hover { border-color: var(--color-primary, #2563eb); transform: translateY(-1px); }
[data-rcl-conflict-flow-option][aria-pressed="true"] { border-color: var(--color-primary, #2563eb); background: color-mix(in srgb, var(--color-primary, #2563eb) 10%, var(--color-surface-raised, #fff)); box-shadow: inset 0 0 0 1px var(--color-primary, #2563eb); }
[data-rcl-conflict-flow-option-label] { color: var(--color-muted-foreground, #64748b); font: var(--text-overline, 700 .68rem/1.1 system-ui, sans-serif); letter-spacing: .08em; text-transform: uppercase; }
[data-rcl-conflict-flow-option-value] { overflow-wrap: anywhere; }
[data-rcl-conflict-flow-status] { min-block-size: 1.35rem; color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
[data-rcl-conflict-flow-status="error"] { color: var(--color-danger, #b42318); }
[data-rcl-conflict-flow-status="success"] { color: var(--color-success, #16803c); }
[data-rcl-conflict-flow-actions] { display: flex; flex-wrap: wrap; gap: var(--space-xs, .625rem); align-items: center; }
[data-rcl-conflict-flow-actions] button { min-block-size: var(--tap-target-min, 44px); padding: var(--space-2xs, .35rem) var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid currentColor; border-radius: var(--radius-control, .625rem); background: transparent; color: var(--color-primary, #2563eb); cursor: pointer; font: var(--text-label, 650 .9rem/1.35 system-ui, sans-serif); }
[data-rcl-conflict-flow-actions] button[type="submit"] { background: var(--color-primary, #2563eb); color: var(--color-on-primary, #fff); }
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

export function ConflictResolutionFlow<T>({
  fields,
  status,
  onResolve,
  onRetry,
  onCancel,
  title = "Resolve concurrent changes",
  description = "Another version changed while you were working. Choose the value that should survive for each field.",
  resolveLabel = "Save resolved version",
  className,
  style,
}: ConflictResolutionFlowProps<T>) {
  const announce = useAnnounce();
  const [choices, setChoices] = useState<Record<string, "local" | "remote">>(
    () => Object.fromEntries(fields.map((field) => [field.id, "local"])),
  );
  const [localStatus, setLocalStatus] =
    useState<ConflictResolutionStatus>("default");
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
      announce(
        "Conflict resolution failed. Your choices are preserved for retry.",
        { priority: "assertive" },
      );
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
      <StyleSheet name="conflictresolutionflow-1-0-0-1" css={styles} />
      <Form
        title={<span data-rcl-conflict-flow-title>{title}</span>}
        description={
          <span data-rcl-conflict-flow-description>{description}</span>
        }
        aria-label="Conflict resolution form"
        onSubmit={() => void submit()}
        footer={
          <div data-rcl-conflict-flow-actions>
            <button type="button" onClick={onCancel} disabled={busy}>
              Keep editing
            </button>
            {resolvedStatus === "request-error" ||
            resolvedStatus === "retry" ? (
              <button
                type="button"
                onClick={() => void retry()}
                disabled={busy}
              >
                Retry resolution
              </button>
            ) : null}
            <button type="submit" disabled={busy || fields.length === 0}>
              {busy ? "Saving…" : resolveLabel}
            </button>
          </div>
        }
      >
        <div data-rcl-conflict-flow-list>
          {resolvedStatus === "loading" ? (
            <div data-rcl-conflict-flow-status role="status">
              Checking the latest version…
            </div>
          ) : null}
          {fields.map((field) => {
            const selected = choices[field.id] ?? "local";
            return (
              <fieldset
                data-rcl-conflict-flow-field
                key={field.id}
                disabled={busy}
              >
                <legend>{field.label}</legend>
                {field.description ? (
                  <div data-rcl-conflict-flow-field-description>
                    {field.description}
                  </div>
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
                        {printable(
                          field,
                          choice === "local" ? field.local : field.remote,
                        )}
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
}
