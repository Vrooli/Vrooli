/**
 * @libraryId react-component-library:AlertDialog
 * @displayName AlertDialog
 * @version 2.0.10
 * @tags ["overlay","confirmation","destructive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:AlertDialog */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { useId, useRef, useState, type CSSProperties, type ReactNode } from "react";
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
[data-rcl-alert-dialog-layer] { position: fixed; inset: 0; z-index: var(--layer-alert); display: grid; place-items: center; box-sizing: border-box; padding: var(--space-lg); background: var(--color-scrim); animation: rcl-alert-dialog-in var(--dur-moderate) var(--ease-standard) both; }
[data-rcl-alert-dialog] { width: min(100%, 32rem); max-height: min(42rem, calc(100dvh - 2 * var(--space-lg))); box-sizing: border-box; overflow: auto; border: 1px solid var(--color-border-strong); border-radius: var(--radius-overlay); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-modal); }
[data-rcl-alert-dialog-header] { display: flex; gap: var(--space-sm); padding: var(--space-lg) var(--space-lg) var(--space-sm); }
[data-rcl-alert-dialog-mark] { flex: 0 0 auto; display: grid; place-items: center; inline-size: 2.5rem; block-size: 2.5rem; border-radius: 50%; background: var(--color-danger-subtle); color: var(--color-danger-foreground); font-weight: 900; }
[data-rcl-alert-dialog-title] { margin: 0; font-size: var(--font-size-lg); line-height: 1.25; letter-spacing: -.02em; }
[data-rcl-alert-dialog-description] { margin: 6px 0 0; color: var(--color-muted-foreground); font-size: var(--font-size-sm); line-height: 1.5; }
[data-rcl-alert-dialog-body] { padding: 0 var(--space-lg) var(--space-md); }
[data-rcl-alert-dialog-body] > :first-child { margin-top: 0; }
[data-rcl-alert-dialog-body] > :last-child { margin-bottom: 0; }
[data-rcl-alert-dialog-error] { margin: var(--space-sm) 0 0; padding: var(--space-sm); border: 1px solid var(--color-danger-border); border-radius: var(--radius-control); background: var(--color-danger-subtle); color: var(--color-danger-foreground); font-size: 13px; line-height: 1.45; }
[data-rcl-alert-dialog-actions] { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: var(--space-2xs); padding: var(--space-sm) var(--space-lg) var(--space-lg); }
[data-rcl-alert-dialog-actions] button { min-block-size: 2.75rem; border-radius: var(--radius-control); padding: 0 var(--space-sm); font: inherit; font-size: var(--font-size-sm); font-weight: 750; cursor: pointer; }
[data-rcl-alert-dialog-actions] button:focus-visible { outline: 3px solid var(--color-focus-ring); outline-offset: 2px; }
[data-rcl-alert-dialog-cancel] { border: 1px solid var(--color-border); background: transparent; color: inherit; }
[data-rcl-alert-dialog-confirm] { border: 0; background: var(--color-primary); color: var(--color-primary-foreground); }
[data-rcl-alert-dialog-confirm][data-destructive="true"] { background: var(--color-danger); color: var(--color-danger-foreground-inverse); }
[data-rcl-alert-dialog-actions] button:disabled { cursor: wait; opacity: var(--opacity-disabled); }
@keyframes rcl-alert-dialog-in { from { opacity: 0; transform: translateY(6px) scale(.99); } to { opacity: 1; transform: none; } }
@media (max-width: 480px) { [data-rcl-alert-dialog-layer] { align-items: end; padding: 0; } [data-rcl-alert-dialog] { max-height: calc(100dvh - 12px); border-radius: var(--radius-overlay) var(--radius-overlay) 0 0; } [data-rcl-alert-dialog-actions] { display: grid; grid-template-columns: 1fr; } [data-rcl-alert-dialog-actions] button { width: 100%; } }
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
  useLibraryStyleSheet("react-component-library:AlertDialog", "2.0.10",   styles);
  const titleId = useId();
  const descriptionId = useId();
  const errorId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);
  const [pending, setPending] = useState(false);
  const isBusy = busy || pending;
  const surfaceTestId = testIdPrefix ? `${testIdPrefix}-dialog` : testId;
  const cancelTestId = testIdPrefix ? `${testIdPrefix}-cancel` : `${testId}.cancel`;
  const confirmTestId = testIdPrefix ? `${testIdPrefix}-confirm` : `${testId}.confirm`;

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
          aria-describedby={status === "error" ? `${descriptionId} ${errorId}` : descriptionId}
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
