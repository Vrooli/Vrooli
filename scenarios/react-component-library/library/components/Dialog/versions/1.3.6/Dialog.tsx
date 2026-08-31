/**
 * @libraryId react-component-library:Dialog
 * @displayName Dialog
 * @version 1.3.6
 * @tags ["overlay","interactive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { X } from "lucide-react";
import { type ReactNode, useId, type RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
export const dialogStyles = `
[data-rcl-dialog] { position: fixed; inset: 0; z-index: var(--layer-modal); display: flex; align-items: end; justify-content: center; padding: var(--space-2xs) var(--space-xs) var(--space-xs); background: color-mix(in srgb, var(--color-shell) 60%, transparent); }
[data-rcl-dialog] .rcl-dialog__backdrop { position: absolute; inset: 0; border: 0; background: transparent; cursor: default; }
[data-rcl-dialog] .rcl-dialog__surface { position: relative; z-index: 1; display: flex; min-block-size: 0; max-block-size: calc(100dvh - var(--space-sm)); inline-size: min(100%, 32rem); flex-direction: column; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-modal); }
[data-rcl-dialog] .rcl-dialog__header { display: flex; align-items: start; justify-content: space-between; gap: var(--space-xs); border-block-end: var(--border-hairline) solid var(--color-border); padding: var(--space-sm) var(--space-md); }
[data-rcl-dialog] .rcl-dialog__heading { min-inline-size: 0; }
[data-rcl-dialog] .rcl-dialog__title { margin: 0; color: var(--color-foreground); font-family: var(--font-sans); font-size: var(--text-heading-size); font-weight: 650; line-height: var(--text-heading-line); }
[data-rcl-dialog] .rcl-dialog__description { margin-block-start: var(--space-3xs); color: var(--color-muted-foreground); font-family: var(--font-sans); font-size: var(--text-body-sm-size); line-height: var(--text-body-sm-line); }
[data-rcl-dialog] button { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); border: 0; border-radius: var(--radius-control); background: transparent; color: var(--color-muted-foreground); cursor: pointer; }
[data-rcl-dialog] .rcl-dialog__surface :is(button, a[href], input, select, textarea, [role="button"]) { min-block-size: var(--tap-target-min); min-inline-size: var(--tap-target-min); }
[data-rcl-dialog] button:hover { background: var(--color-surface-muted); color: var(--color-foreground); }
[data-rcl-dialog] .rcl-dialog__body { min-block-size: 0; flex: 1; overflow: auto; padding: var(--space-md); }
[data-rcl-dialog] .rcl-dialog__footer { border-block-start: var(--border-hairline) solid var(--color-border); padding: var(--space-sm) var(--space-md); }
[data-rcl-dialog] :focus-visible { outline: 3px solid color-mix(in srgb, var(--color-focus) 38%, transparent); outline-offset: 2px; }
@media (min-width: 768px) { [data-rcl-dialog] { align-items: center; padding: var(--space-md); } }
@media (prefers-reduced-motion: reduce) { [data-rcl-dialog] *, [data-rcl-dialog] *::before, [data-rcl-dialog] *::after { transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-dialog] .rcl-dialog__surface { border-color: CanvasText; background: Canvas; color: CanvasText; } [data-rcl-dialog] .rcl-dialog__backdrop { background: Canvas; opacity: .8; } }
`;
export const DIALOG_MODES = ["controlled", "uncontrolled"] as const;
export const DIALOG_PARTS = [
  "trigger",
  "overlay",
  "content",
  "header",
  "title",
  "description",
  "body",
  "footer",
  "close",
] as const;

export interface DialogProps {
  open: boolean;
  title: string;
  children: ReactNode;
  onClose: () => void;
  closeLabel: string;
  description?: string;
  footer?: ReactNode;
  className?: string;
  onOpenChange?: (open: boolean) => void;
  initialFocusRef?: RefObject<HTMLElement | null>;
  returnFocusRef?: RefObject<HTMLElement | null>;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
  testId?: string;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

export const Dialog = withClassName(function Dialog({
  open,
  title,
  description,
  children,
  footer,
  onClose,
  closeLabel,
  className,
  onOpenChange,
  initialFocusRef,
  returnFocusRef,
  panelClassName,
  contentClassName,
  backdropClassName,
  testId = "overlays.dialog",
}: DialogProps) {
  useLibraryStyleSheet("dialog-1.3.0", dialogStyles);
  const id = useId();
  const titleID = `${id}-title`;
  const descriptionID = `${id}-description`;
  const overlay = useOverlaySurface({
    open,
    onOpenChange: (next) => {
      onOpenChange?.(next);
      if (!next) onClose();
    },
    modal: true,
    kind: "dialog",
    initialFocusRef,
    returnFocusRef,
  });
  if (!overlay.present) return null;

  return (
    <Portal>
      <div data-rcl-dialog className={cn("rcl-dialog", className)} data-state={overlay.state}>
        <button
          data-testid={`${testId}.backdrop`}
          type="button"
          aria-label={closeLabel}
          className={cn("rcl-dialog__backdrop", backdropClassName)}
          {...overlay.backdropProps}
        />
        <section
          ref={overlay.surfaceRef}
          data-testid={testId}
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleID}
          aria-describedby={description ? descriptionID : undefined}
          className={cn("rcl-dialog__surface", panelClassName)}
        >
          <header className="rcl-dialog__header">
            <div className="rcl-dialog__heading">
              <h2 id={titleID} className="rcl-dialog__title">
                {title}
              </h2>
              {description && (
                <p id={descriptionID} className="rcl-dialog__description">
                  {description}
                </p>
              )}
            </div>
            <button
              data-testid={`${testId}.close`}
              type="button"
              aria-label={closeLabel}
              className="rcl-dialog__close"
              onClick={overlay.close}
            >
              <X aria-hidden width="20" height="20" />
            </button>
          </header>
          <div className={cn("rcl-dialog__body", contentClassName)}>{children}</div>
          {footer && <footer className="rcl-dialog__footer">{footer}</footer>}
        </section>
      </div>
    </Portal>
  );
});
