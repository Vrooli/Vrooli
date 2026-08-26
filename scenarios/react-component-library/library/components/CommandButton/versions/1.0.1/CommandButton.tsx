/**
 * @libraryId react-component-library:CommandButton
 * @displayName CommandButton
 * @description An async-aware action button with stable dimensions, acknowledged progress, recovery, and retry.
 * @version 1.0.1
 * @tags ["control","async","recovery","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource controls.command-button */
import { forwardRef, type CSSProperties, type ReactNode } from "react";
import { Button, type ButtonProps } from "@vrooli/react-component-library/Button/2.0.0";
import {
  useAsyncAction,
  type AsyncAction,
} from "@vrooli/react-component-library/useAsyncAction/1.0.0";

const styles = `
[data-rcl-command-button-labels] { display: inline-grid; align-items: center; justify-items: center; }
[data-rcl-command-button-label] { grid-area: 1 / 1; display: inline-flex; align-items: center; gap: var(--space-2xs); opacity: 0; pointer-events: none; }
[data-rcl-command-button-label][data-active="true"] { opacity: 1; pointer-events: auto; }
[data-rcl-command-button-label][data-state="success"] { color: var(--color-success); }
[data-rcl-command-button-label][data-state="error"] { color: var(--color-danger); }
[data-rcl-command-button-spinner] { inline-size: var(--space-sm); block-size: var(--space-sm); flex: 0 0 auto; border: var(--border-strong) solid color-mix(in srgb, currentColor 28%, transparent); border-block-start-color: currentColor; border-radius: var(--radius-pill); animation: rcl-command-button-spin var(--dur-moderate) linear infinite; }
[data-rcl-command-button-status] { position: absolute; inline-size: 1px; block-size: 1px; overflow: hidden; clip-path: inset(50%); white-space: nowrap; }
@keyframes rcl-command-button-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-command-button-spinner] { animation: none; } }
`;

export interface CommandButtonProps
  extends Omit<ButtonProps, "children" | "pending" | "pendingLabel"> {
  children: ReactNode;
  className?: string;
  action?: AsyncAction<unknown> | (() => Promise<unknown>);
  pendingLabel?: ReactNode;
  successLabel?: ReactNode;
  errorLabel?: ReactNode;
  onSuccess?: (value: unknown) => void;
  onError?: (error: unknown) => void;
  statusStyle?: CSSProperties;
}

export const CommandButton = forwardRef<HTMLButtonElement, CommandButtonProps>(
  function CommandButton(
    {
      action,
      children,
      disabled,
      errorLabel = "Try again",
      onClick,
      onError,
      onSuccess,
      pendingLabel = "Working…",
      statusStyle,
      successLabel = "Done",
      ...props
    },
    ref,
  ) {
    const actionController = useAsyncAction(action ?? (() => Promise.resolve(undefined)), {
      onError,
      onSuccess,
    });
    const state = action ? actionController.status : "idle";
    const busy = state === "pending";
    const statusMessage =
      state === "pending"
        ? pendingLabel
        : state === "success"
          ? successLabel
          : state === "error"
            ? errorLabel
            : state === "cancelled"
              ? "Cancelled"
              : undefined;

    return (
      <>
        <style data-rcl-command-button-styles dangerouslySetInnerHTML={{ __html: styles }} />
        <Button data-testid="controls.command-button"
          {...props}
          ref={ref}
          disabled={disabled || busy}
          aria-busy={busy || undefined}
          aria-disabled={disabled || busy || undefined}
          data-rcl-command-button="true"
          data-rcl-command-state={state}
          onClick={(event) => {
            onClick?.(event);
            if (!action || event.defaultPrevented || disabled || busy) return;
            void actionController.run().catch(() => undefined);
          }}
        >
          <span
            data-rcl-command-button-labels
            style={statusStyle}
            data-testid="command-button-labels"
          >
            <span
              data-rcl-command-button-label
              data-state="idle"
              data-active={state === "idle"}
              aria-hidden={state !== "idle"}
              style={{ opacity: state === "idle" ? 1 : 0 }}
            >
              {children}
            </span>
            <span
              data-rcl-command-button-label
              data-state="pending"
              data-active={state === "pending"}
              aria-hidden={state !== "pending"}
              style={{ opacity: state === "pending" ? 1 : 0 }}
            >
              <span aria-hidden="true" data-rcl-command-button-spinner />
              {pendingLabel}
            </span>
            <span
              data-rcl-command-button-label
              data-state="success"
              data-active={state === "success"}
              aria-hidden={state !== "success"}
              style={{ opacity: state === "success" ? 1 : 0 }}
            >
              <span aria-hidden="true">✓</span>
              {successLabel}
            </span>
            <span
              data-rcl-command-button-label
              data-state="error"
              data-active={state === "error"}
              aria-hidden={state !== "error"}
              style={{ opacity: state === "error" ? 1 : 0 }}
            >
              <span aria-hidden="true">!</span>
              {errorLabel}
            </span>
          </span>
          <span data-rcl-command-button-status role="status" aria-live="polite" aria-atomic="true">
            {statusMessage}
          </span>
        </Button>
      </>
    );
  },
);
