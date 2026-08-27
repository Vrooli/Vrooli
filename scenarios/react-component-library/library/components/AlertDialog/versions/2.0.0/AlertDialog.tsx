/**
 * @libraryId react-component-library:AlertDialog
 * @displayName AlertDialog
 * @description
 * @version 2.0.0
 * @tags ["overlay","confirmation","destructive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:AlertDialog */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import { useId, useRef, useState, type CSSProperties, type ReactNode, type RefObject } from "react";
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
}

const styles = `
[data-rcl-alert-dialog-layer] { position: fixed; inset: 0; z-index: var(--layer-modal, 400); display: grid; place-items: center; box-sizing: border-box; padding: var(--space-lg, 24px); background: var(--color-scrim, rgb(15 23 42 / .52)); animation: rcl-alert-dialog-in var(--dur-moderate, 180ms) var(--ease-standard, cubic-bezier(.2,.8,.2,1)) both; }
[data-rcl-alert-dialog] { width: min(100%, 32rem); max-height: min(42rem, calc(100dvh - 2 * var(--space-lg, 24px))); box-sizing: border-box; overflow: auto; border: 1px solid var(--color-border-strong, #b7c3d4); border-radius: var(--radius-overlay, 1rem); background: var(--color-surface, #fff); color: var(--color-foreground, #0f172a); box-shadow: var(--elev-modal, 0 24px 72px rgb(15 23 42 / .24)); }
[data-rcl-alert-dialog-header] { display: flex; gap: var(--space-sm, 12px); padding: var(--space-lg, 24px) var(--space-lg, 24px) var(--space-sm, 12px); }
[data-rcl-alert-dialog-mark] { flex: 0 0 auto; display: grid; place-items: center; inline-size: 2.5rem; block-size: 2.5rem; border-radius: 50%; background: var(--color-danger-subtle, #fde8e7); color: var(--color-danger-foreground, #b42318); font-weight: 900; }
[data-rcl-alert-dialog-title] { margin: 0; font-size: var(--font-size-lg, 18px); line-height: 1.25; letter-spacing: -.02em; }
[data-rcl-alert-dialog-description] { margin: 6px 0 0; color: var(--color-muted-foreground, #64748b); font-size: var(--font-size-sm, 14px); line-height: 1.5; }
[data-rcl-alert-dialog-body] { padding: 0 var(--space-lg, 24px) var(--space-md, 16px); }
[data-rcl-alert-dialog-body] > :first-child { margin-top: 0; }
[data-rcl-alert-dialog-body] > :last-child { margin-bottom: 0; }
[data-rcl-alert-dialog-error] { margin: var(--space-sm, 12px) 0 0; padding: var(--space-sm, 12px); border: 1px solid var(--color-danger-border, #e7a8a8); border-radius: var(--radius-control, .5rem); background: var(--color-danger-subtle, #fff1f0); color: var(--color-danger-foreground, #b42318); font-size: 13px; line-height: 1.45; }
[data-rcl-alert-dialog-actions] { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: var(--space-2xs, 8px); padding: var(--space-sm, 12px) var(--space-lg, 24px) var(--space-lg, 24px); }
[data-rcl-alert-dialog-actions] button { min-block-size: 2.75rem; border-radius: var(--radius-control, .5rem); padding: 0 var(--space-sm, 12px); font: inherit; font-size: var(--font-size-sm, 14px); font-weight: 750; cursor: pointer; }
[data-rcl-alert-dialog-actions] button:focus-visible { outline: 3px solid var(--color-focus-ring, #2563eb); outline-offset: 2px; }
[data-rcl-alert-dialog-cancel] { border: 1px solid var(--color-border, #cbd5e1); background: transparent; color: inherit; }
[data-rcl-alert-dialog-confirm] { border: 0; background: var(--color-primary, #2563eb); color: var(--color-primary-foreground, #fff); }
[data-rcl-alert-dialog-confirm][data-destructive="true"] { background: var(--color-danger, #c9362b); color: var(--color-danger-foreground-inverse, #fff); }
[data-rcl-alert-dialog-actions] button:disabled { cursor: wait; opacity: var(--opacity-disabled, .55); }
@keyframes rcl-alert-dialog-in { from { opacity: 0; transform: translateY(6px) scale(.99); } to { opacity: 1; transform: none; } }
@media (max-width: 480px) { [data-rcl-alert-dialog-layer] { align-items: end; padding: 0; } [data-rcl-alert-dialog] { max-height: calc(100dvh - 12px); border-radius: var(--radius-overlay, 1rem) var(--radius-overlay, 1rem) 0 0; } [data-rcl-alert-dialog-actions] { display: grid; grid-template-columns: 1fr; } [data-rcl-alert-dialog-actions] button { width: 100%; } }
@media (prefers-reduced-motion: reduce) { [data-rcl-alert-dialog-layer] { animation: none; } }
`;

function focusable(container: HTMLElement) {
  return Array.from(
    container.querySelectorAll<HTMLElement>(
      "button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex='-1'])",
    ),
  );
}

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
}: AlertDialogProps) {
  useLibraryStyleSheet("alert-dialog-2.0.0", styles);
  const titleId = useId();
  const descriptionId = useId();
  const errorId = useId();
  const cancelRef = useRef<HTMLButtonElement>(null);
  const [pending, setPending] = useState(false);
  const isBusy = busy || pending;

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
          ref={overlay.surfaceRef as RefObject<HTMLDivElement>}
          data-testid={testId}
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
              data-testid={`${testId}.cancel`}
              ref={cancelRef}
              type="button"
              data-rcl-alert-dialog-cancel
              onClick={onCancel}
              disabled={isBusy}
            >
              {cancelLabel}
            </button>
            <button
              data-testid={`${testId}.confirm`}
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
