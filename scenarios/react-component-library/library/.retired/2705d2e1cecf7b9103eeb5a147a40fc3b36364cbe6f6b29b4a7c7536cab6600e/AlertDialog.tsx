/**
 * @libraryId react-component-library:AlertDialog
 * @displayName AlertDialog
 * @description The focused confirmation surface for consequential decisions, explaining impact, separating safe from destructive actions, and resisting accidental dismissal.
 * @version 2.0.8
 * @tags ["overlay","confirmation","destructive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:AlertDialog */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import {
  useId,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export type AlertDialogStatus = "default" | "error";

export interface AlertDialogProps {
  open: boolean;
  title: string;
  description: string;
  children?: ReactNode;
  status?: AlertDialogStatus;
  errorMessage?: string;
  destructive?: boolean;
  busy?: boolean;
  onConfirm: () => void | Promise<void>;
  onCancel: () => void;
  confirmLabel?: string;
  cancelLabel?: string;
  busyLabel?: string;
  closeLabel?: string;
  style?: CSSProperties;
  testId?: string;
  testIdPrefix?: string;
}

const styles = `
[data-rcl-alert-dialog-layer] { position: fixed; inset: 0; z-index: var(--layer-alert, 700); display: grid; place-items: center; box-sizing: border-box; padding: var(--space-lg, 32px); background: var(--color-scrim, color-mix(in srgb, var(--color-shell) 52%, transparent)); animation: rcl-alert-dialog-in var(--dur-moderate, 280ms) var(--ease-standard, cubic-bezier(.2, 0, 0, 1)) both; }
[data-rcl-alert-dialog] { width: min(100%, 32rem); max-height: min(42rem, calc(100dvh - 2 * var(--space-lg, 32px))); box-sizing: border-box; overflow: auto; border: 1px solid var(--color-border-strong, color-mix(in srgb, var(--color-border) 72%, var(--color-foreground))); border-radius: var(--radius-overlay, 1rem); background: var(--color-surface, #ffffff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-modal, 0 4px 12px rgba(9, 18, 22, .10), 0 16px 48px rgba(9, 18, 22, .18)); }
[data-rcl-alert-dialog-header] { display: flex; gap: var(--space-sm, 16px); padding: var(--space-lg, 32px) var(--space-lg, 32px) var(--space-sm, 16px); }
[data-rcl-alert-dialog-mark] { flex: 0 0 auto; display: grid; place-items: center; inline-size: 2.5rem; block-size: 2.5rem; border-radius: 50%; background: var(--color-danger-subtle, color-mix(in srgb, var(--color-danger) 12%, var(--color-surface))); color: var(--color-danger-foreground, color-mix(in srgb, var(--color-danger) 78%, var(--color-foreground))); font-weight: 900; }
[data-rcl-alert-dialog-title] { margin: 0; font-size: var(--font-size-lg, 18px); line-height: 1.25; letter-spacing: -.02em; }
[data-rcl-alert-dialog-description] { margin: 6px 0 0; color: var(--color-muted-foreground, #64748b); font-size: var(--font-size-sm, 14px); line-height: 1.5; }
[data-rcl-alert-dialog-body] { padding: 0 var(--space-lg, 32px) var(--space-md, 24px); }
[data-rcl-alert-dialog-body] > :first-child { margin-top: 0; }
[data-rcl-alert-dialog-body] > :last-child { margin-bottom: 0; }
[data-rcl-alert-dialog-error] { margin: var(--space-sm, 16px) 0 0; padding: var(--space-sm, 16px); border: 1px solid var(--color-danger-border, color-mix(in srgb, var(--color-danger) 38%, var(--color-border))); border-radius: var(--radius-control, 0.375rem); background: var(--color-danger-subtle, color-mix(in srgb, var(--color-danger) 12%, var(--color-surface))); color: var(--color-danger-foreground, color-mix(in srgb, var(--color-danger) 78%, var(--color-foreground))); font-size: 13px; line-height: 1.45; }
[data-rcl-alert-dialog-actions] { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: var(--space-2xs, 8px); padding: var(--space-sm, 16px) var(--space-lg, 32px) var(--space-lg, 32px); }
[data-rcl-alert-dialog-actions] button { min-block-size: 2.75rem; border-radius: var(--radius-control, 0.375rem); padding: 0 var(--space-sm, 16px); font: inherit; font-size: var(--font-size-sm, 14px); font-weight: 750; cursor: pointer; }
[data-rcl-alert-dialog-actions] button:focus-visible { outline: 3px solid var(--color-focus-ring, var(--color-focus)); outline-offset: 2px; }
[data-rcl-alert-dialog-cancel] { border: 1px solid var(--color-border, #cbd5e1); background: transparent; color: inherit; }
[data-rcl-alert-dialog-confirm] { border: 0; background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #ffffff); }
[data-rcl-alert-dialog-confirm][data-destructive="true"] { background: var(--color-danger, #dc2626); color: var(--color-danger-foreground-inverse, var(--color-primary-foreground)); }
[data-rcl-alert-dialog-actions] button:disabled { cursor: wait; opacity: var(--opacity-disabled, .40); }
@keyframes rcl-alert-dialog-in { from { opacity: 0; transform: translateY(6px) scale(.99); } to { opacity: 1; transform: none; } }
@media (max-width: 480px) { [data-rcl-alert-dialog-layer] { align-items: end; padding: 0; } [data-rcl-alert-dialog] { max-height: calc(100dvh - 12px); border-radius: var(--radius-overlay, 1rem) var(--radius-overlay, 1rem) 0 0; } [data-rcl-alert-dialog-actions] { display: grid; grid-template-columns: 1fr; } [data-rcl-alert-dialog-actions] button { width: 100%; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-alert-dialog-layer] { animation: none; } }
`;

export const AlertDialog = withClassName(function AlertDialog({
  open,
  title,
  description,
  children,
  status = "default",
  errorMessage,
  destructive = false,
  busy = false,
  onConfirm,
  onCancel,
  confirmLabel = destructive ? "Delete" : "Confirm",
  cancelLabel = "Cancel",
  busyLabel = "Working…",
  closeLabel = "Confirmation dialog",
  style,
  testId = "overlays.alert-dialog",
  testIdPrefix,
}: AlertDialogProps) {
  useLibraryStyleSheet("alert-dialog-2.0.6", styles);
  const titleId = useId();
  const descriptionId = useId();
  const errorId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);
  const [pending, setPending] = useState(false);
  const isBusy = busy || pending;
  const surfaceTestId = testIdPrefix ? `${testIdPrefix}-dialog` : testId;
  const cancelTestId = testIdPrefix
    ? `${testIdPrefix}-cancel`
    : `${testId}.cancel`;
  const confirmTestId = testIdPrefix
    ? `${testIdPrefix}-confirm`
    : `${testId}.confirm`;

  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      if (!next) onCancel();
    },
    modal: true,
    kind: "alertdialog",
    dismiss: { escape: true, backdrop: false },
    initialFocusRef: cancelRef,
  });
  if (!overlay.present) return null;

  const confirm = async () => {
    setPending(true);
    try {
      await onConfirm();
    } finally {
      setPending(false);
    }
  };

  return (
    <Portal>
      <div
        data-rcl-alert-dialog-layer
        style={style}
        aria-label={closeLabel}
        data-state={overlay.state}
      >
        <div
          ref={(node) => {
            overlay.surfaceRef.current = node;
          }}
          data-testid={surfaceTestId}
          data-rcl-alert-dialog
          role="alertdialog"
          aria-modal="true"
          aria-labelledby={titleId}
          aria-describedby={
            status === "error" ? `${descriptionId} ${errorId}` : descriptionId
          }
        >
          <div data-rcl-alert-dialog-header>
            <div data-rcl-alert-dialog-mark aria-hidden="true">
              !
            </div>
            <div>
              <h2 id={titleId} data-rcl-alert-dialog-title>
                {title}
              </h2>
              <p id={descriptionId} data-rcl-alert-dialog-description>
                {description}
              </p>
            </div>
          </div>
          {children && (
            <div data-rcl-alert-dialog-body>
              {children}
              {status === "error" && errorMessage && (
                <p id={errorId} data-rcl-alert-dialog-error role="alert">
                  {errorMessage}
                </p>
              )}
            </div>
          )}
          {!children && status === "error" && errorMessage && (
            <div data-rcl-alert-dialog-body>
              <p id={errorId} data-rcl-alert-dialog-error role="alert">
                {errorMessage}
              </p>
            </div>
          )}
          <div data-rcl-alert-dialog-actions>
            <button
              data-testid={cancelTestId}
              ref={cancelRef}
              type="button"
              data-rcl-alert-dialog-cancel
              onClick={onCancel}
              disabled={isBusy}
            >
              {cancelLabel}
            </button>
            <button
              data-testid={confirmTestId}
              type="button"
              data-rcl-alert-dialog-confirm
              data-destructive={destructive ? "true" : "false"}
              onClick={() => void confirm()}
              disabled={isBusy}
            >
              {isBusy ? busyLabel : confirmLabel}
            </button>
          </div>
        </div>
      </div>
    </Portal>
  );
});
