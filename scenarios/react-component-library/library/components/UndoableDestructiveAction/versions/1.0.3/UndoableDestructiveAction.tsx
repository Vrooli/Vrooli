/**
 * @libraryId react-component-library:UndoableDestructiveAction
 * @displayName UndoableDestructiveAction
 * @description A forgiving destructive-action pattern that completes reversible removal immediately and exposes a clear undo path.
 * @version 1.0.3
 * @tags ["patterns","recovery","destructive","undo","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:UndoableDestructiveAction */
import { translate } from "../../../../hooks/useLocale/versions/1.0.1/useLocale";
import { useState, type CSSProperties, type ReactNode } from "react";
import { UndoBanner } from "../../../UndoBanner/versions/1.0.0/UndoBanner";
import {
  UndoManagerProvider,
  useUndoManager,
} from "../../../../services/UndoManager/versions/1.0.0/UndoManager";

export type UndoableDestructiveActionState = "idle" | "submitting" | "success" | "error";

export interface UndoableDestructiveActionProps {
  itemLabel: string;
  onDelete: () => void | Promise<void>;
  onUndo: () => void | Promise<void>;
  description?: string;
  deleteLabel?: string;
  retryLabel?: string;
  errorMessage?: string;
  successMessage?: string;
  defaultState?: UndoableDestructiveActionState;
  expiresMs?: number;
  className?: string;
  style?: CSSProperties;
  children?: ReactNode;
}

const styles = `
  [data-rcl-undoable-action] { display: grid; gap: var(--space-sm); min-inline-size: 0; box-sizing: border-box; padding: var(--space-md); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-raised); }
  [data-rcl-undoable-action-header] { display: grid; gap: var(--space-3xs); min-inline-size: 0; }
  [data-rcl-undoable-action-title] { color: var(--color-foreground); font: var(--text-heading); letter-spacing: -.01em; }
  [data-rcl-undoable-action-description] { max-inline-size: 60ch; color: var(--color-muted-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; }
  [data-rcl-undoable-action-controls] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs); }
  [data-rcl-undoable-action-button], [data-rcl-undoable-action-retry] { min-block-size: var(--tap-target-min); box-sizing: border-box; border: var(--border-hairline) solid var(--color-danger); border-radius: var(--radius-control); background: var(--color-danger); color: var(--color-danger-foreground-inverse, #fff); padding-inline: var(--space-md); font: var(--text-label); cursor: pointer; transition: background-color var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
  [data-rcl-undoable-action-button]:hover, [data-rcl-undoable-action-retry]:hover { border-color: var(--color-danger-foreground); background: var(--color-danger-foreground); transform: translateY(-1px); }
  [data-rcl-undoable-action-button]:focus-visible, [data-rcl-undoable-action-retry]:focus-visible { outline: var(--border-strong) solid var(--color-focus-ring); outline-offset: var(--space-3xs); }
  [data-rcl-undoable-action-button]:disabled, [data-rcl-undoable-action-retry]:disabled { cursor: wait; opacity: var(--opacity-disabled); }
  [data-rcl-undoable-action-status] { display: grid; gap: var(--space-3xs); min-inline-size: 0; padding: var(--space-sm); border-inline-start: var(--border-strong) solid var(--color-success); border-radius: var(--radius-control); background: color-mix(in srgb, var(--color-success) 10%, var(--color-surface)); color: var(--color-foreground); font: var(--text-body-sm); }
  [data-rcl-undoable-action][data-state="error"] [data-rcl-undoable-action-status] { border-inline-start-color: var(--color-danger); background: color-mix(in srgb, var(--color-danger) 9%, var(--color-surface)); }
  [data-rcl-undoable-action-status] strong { font: var(--text-label); }
  @media (max-width: 38rem) { [data-rcl-undoable-action] { padding: var(--space-sm); } [data-rcl-undoable-action-controls] > button { inline-size: 100%; } }
  @media (prefers-reduced-motion: reduce) { [data-rcl-undoable-action-button]:hover, [data-rcl-undoable-action-retry]:hover { transform: none; } }
  @media (forced-colors: active) { [data-rcl-undoable-action] { border: 2px solid CanvasText; } [data-rcl-undoable-action-button], [data-rcl-undoable-action-retry] { border: 2px solid ButtonText; background: ButtonFace; color: ButtonText; } [data-rcl-undoable-action-status] { border: 2px solid CanvasText; } }
`;

function ActionContent({
  itemLabel,
  onDelete,
  onUndo,
  description,
  deleteLabel,
  retryLabel,
  errorMessage,
  successMessage,
  defaultState,
  expiresMs,
  className,
  style,
  children,
}: UndoableDestructiveActionProps) {
  const manager = useUndoManager();
  const [state, setState] = useState<UndoableDestructiveActionState>(defaultState ?? "idle");
  const [error, setError] = useState<string | null>(null);

  const runDelete = async () => {
    if (state === "submitting") return;
    setState("submitting");
    setError(null);
    try {
      await onDelete();
      setState("success");
      manager.push({
        title: `${itemLabel} removed`,
        detail: "Undo is available for a few seconds.",
        expiresMs,
        successMessage: `${itemLabel} restored`,
        successDetail: "The previous state is back.",
        undo: async () => {
          await onUndo();
          setState("idle");
        },
      });
    } catch (cause) {
      setState("error");
      setError(cause instanceof Error && cause.message ? cause.message : null);
    }
  };

  const resolvedError = error ?? errorMessage;
  const resolvedDescription =
    description ?? `Remove ${itemLabel} now. You can undo this change for a few seconds.`;
  const resolvedSuccess = successMessage ?? `${itemLabel} removed`;

  return (
    <>
      <style data-rcl-undoable-action-styles dangerouslySetInnerHTML={{ __html: styles }} />
      <section
        data-rcl-undoable-action
        data-state={state}
        className={className}
        style={style}
        aria-label={`${itemLabel} destructive action`}
      >
        <div data-rcl-undoable-action-header>
          <strong data-rcl-undoable-action-title>{itemLabel}</strong>
          <span data-rcl-undoable-action-description>{resolvedDescription}</span>
        </div>
        {children}
        {state === "idle" && (
          <div data-rcl-undoable-action-controls>
            <button
              data-testid="patterns.undoable-destructive-action"
              data-rcl-undoable-action-button
              type="button"
              onClick={() => void runDelete()}
            >
              {deleteLabel ?? `Remove ${itemLabel}`}
            </button>
          </div>
        )}
        {state === "submitting" && (
          <div data-rcl-undoable-action-status role="status" aria-live="polite" aria-busy="true">
            <strong>Removing {itemLabel}…</strong>
            <span>
              {translate(
                "patterns.undoable-destructive-action.text.1",
                "Keep this page open while the change is saved.",
              )}
            </span>
          </div>
        )}
        {state === "success" && (
          <div data-rcl-undoable-action-status role="status" aria-live="polite">
            <strong>{resolvedSuccess}</strong>
            <span>
              {translate(
                "patterns.undoable-destructive-action.text.2",
                "The change is complete. A recovery action is visible below.",
              )}
            </span>
          </div>
        )}
        {state === "error" && (
          <div data-rcl-undoable-action-controls>
            <div data-rcl-undoable-action-status role="alert" aria-live="assertive">
              <strong>Could not remove {itemLabel}</strong>
              <span>{resolvedError ?? "The service did not confirm this change."}</span>
            </div>
            <button
              data-testid="patterns.undoable-destructive-action"
              data-rcl-undoable-action-retry
              type="button"
              onClick={() => void runDelete()}
            >
              {retryLabel ?? "Try again"}
            </button>
          </div>
        )}
      </section>
      <UndoBanner />
    </>
  );
}

export function UndoableDestructiveAction(props: UndoableDestructiveActionProps) {
  return (
    <UndoManagerProvider maxVisible={1} defaultExpiresMs={props.expiresMs ?? 8000}>
      <ActionContent {...props} />
    </UndoManagerProvider>
  );
}
