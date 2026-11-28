/**
 * @libraryId react-component-library:Dialog
 * @displayName Dialog
 * @description The modal or non-modal conversation surface with focus containment, inert background, reference-counted scroll lock, structured regions, async actions, and nested-overlay support.
 * @version 1.3.11-draft.1
 * @tags []
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import {
  OverlayActionBar,
  type OverlayActionList,
} from "@vrooli/react-component-library/OverlayActionBar/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

import { X } from "lucide-react";
import { createContext, type HTMLAttributes, type ReactNode, useContext, useId, type RefObject } from "react";
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
@media (max-width: 767.98px) { [data-rcl-dialog][data-full-screen-mobile] { align-items: stretch; padding: 0; } [data-rcl-dialog][data-full-screen-mobile] .rcl-dialog__surface { inline-size: 100%; block-size: 100dvh; max-block-size: none; border: 0; border-radius: 0; } }
@media (min-width: 768px) { [data-rcl-dialog][data-full-screen-mobile] .rcl-dialog__surface { max-block-size: calc(100dvh - var(--space-lg) * 2); } }
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
  title: ReactNode;
  children: ReactNode;
  onClose: () => void;
  closeLabel: string;
  description?: ReactNode;
  footer?: ReactNode;
  /** Declarative actions use the shared one-row footer contract. */
  actions?: OverlayActionList;
  className?: string;
  onOpenChange?: (open: boolean) => void;
  initialFocusRef?: RefObject<HTMLElement | null>;
  returnFocusRef?: RefObject<HTMLElement | null>;
  panelClassName?: string;
  contentClassName?: string;
  backdropClassName?: string;
  /** Expand the panel to the mobile viewport while keeping the desktop dialog layout. */
  fullScreenMobile?: boolean;
  testId?: string;
}

const cn = (...inputs: Array<string | undefined>) => inputs.filter(Boolean).join(" ");

export const Dialog = withClassName(function Dialog({
  open,
  title,
  description,
  children,
  footer,
  actions,
  onClose,
  closeLabel,
  className,
  onOpenChange,
  initialFocusRef,
  returnFocusRef,
  panelClassName,
  contentClassName,
  backdropClassName,
  fullScreenMobile = false,
  testId = "overlays.dialog",
}: DialogProps) {
  useLibraryStyleSheet("react-component-library:Dialog", "1.3.11-draft.1", dialogStyles);
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
      <div
        {...overlay.rootProps}
        data-rcl-dialog
        className={cn("rcl-dialog", className)}
        data-state={overlay.state}
        data-full-screen-mobile={fullScreenMobile || undefined}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleID}
        aria-describedby={description ? descriptionID : undefined}
      >
        <button
          data-testid={`${testId}.backdrop`}
          data-rcl-selector="overlays.dialog.backdrop"
          type="button"
          aria-label={closeLabel}
          className={cn("rcl-dialog__backdrop", backdropClassName)}
          {...overlay.backdropProps}
        />
        <section
          ref={overlay.surfaceRef}
          data-testid={testId}
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
              data-rcl-selector="overlays.dialog.close"
              type="button"
              aria-label={closeLabel}
              className="rcl-dialog__close"
              onClick={overlay.close}
            >
              <X aria-hidden width="20" height="20" />
            </button>
          </header>
          <div className={cn("rcl-dialog__body", contentClassName)}>{children}</div>
          {(actions?.length || footer) && (
            <footer className="rcl-dialog__footer">
              {actions?.length ? <OverlayActionBar actions={actions} /> : footer}
            </footer>
          )}
        </section>
      </div>
    </Portal>
  );
});

interface CompoundDialogState {
  overlay: ReturnType<typeof useOverlaySurface>;
  titleID: string;
  descriptionID: string;
  closeLabel: string;
  contentClassName?: string;
}

const CompoundDialogContext = createContext<CompoundDialogState | null>(null);

function useCompoundDialog() {
  const context = useContext(CompoundDialogContext);
  if (!context) throw new Error("Dialog parts must be rendered inside DialogRoot.");
  return context;
}

export interface DialogRootProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  children: ReactNode;
  contentClassName?: string;
  fullScreenMobile?: boolean;
  closeLabel?: string;
  className?: string;
}

/** Composition API for consumers that need custom header, body, or footer regions. */
export function DialogRoot({
  open,
  onOpenChange,
  children,
  contentClassName,
  fullScreenMobile = false,
  closeLabel = "Close dialog",
  className,
}: DialogRootProps) {
  useLibraryStyleSheet("react-component-library:Dialog", "1.3.11-draft.1", dialogStyles);
  const id = useId();
  const overlay = useOverlaySurface({
    open,
    onOpenChange,
    modal: true,
    kind: "dialog",
  });
  if (!overlay.present) return null;
  const context = {
    overlay,
    titleID: `${id}-title`,
    descriptionID: `${id}-description`,
    closeLabel,
    contentClassName,
  };

  return (
    <Portal>
      <div
        {...overlay.rootProps}
        data-rcl-dialog
        data-state={overlay.state}
        data-full-screen-mobile={fullScreenMobile || undefined}
        className={cn("rcl-dialog", className)}
      >
        <button
          type="button"
          aria-label={closeLabel}
          className="rcl-dialog__backdrop"
          {...overlay.backdropProps}
        />
        <CompoundDialogContext.Provider value={context}>{children}</CompoundDialogContext.Provider>
      </div>
    </Portal>
  );
}

export interface DialogContentProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  fullScreenMobile?: boolean;
}

/** Optional wrapper retained for semantic grouping inside a custom surface. */
export function DialogContent({ children, ...props }: DialogContentProps) {
  const { overlay, titleID, descriptionID, contentClassName } = useCompoundDialog();
  const { className, fullScreenMobile, ...contentProps } = props;
  return (
    <div
      ref={(node) => { overlay.surfaceRef.current = node; }}
      data-state={overlay.state}
      data-full-screen-mobile={fullScreenMobile || undefined}
      className={cn("rcl-dialog__surface", contentClassName, className)}
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleID}
      aria-describedby={descriptionID}
      {...contentProps}
    >
      {children}
    </div>
  );
}

export interface DialogHeaderProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  onClose?: () => void;
}

export function DialogHeader({ children, onClose, className, ...props }: DialogHeaderProps) {
  const { closeLabel } = useCompoundDialog();
  return (
    <header className={cn("rcl-dialog__header", className)} {...props}>
      <div className="rcl-dialog__heading">{children}</div>
      {onClose && (
        <button type="button" aria-label={closeLabel} onClick={onClose} className="rcl-dialog__close">
          <X aria-hidden width="20" height="20" />
        </button>
      )}
    </header>
  );
}

export function DialogTitle({ children, className, ...props }: HTMLAttributes<HTMLHeadingElement>) {
  const { titleID } = useCompoundDialog();
  return <h2 id={titleID} className={cn("rcl-dialog__title", className)} {...props}>{children}</h2>;
}

export function DialogDescription({ className, ...props }: HTMLAttributes<HTMLParagraphElement>) {
  const { descriptionID } = useCompoundDialog();
  return <p id={descriptionID} className={cn("rcl-dialog__description", className)} {...props} />;
}

export function DialogBody({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("rcl-dialog__body", className)} {...props} />;
}

export function DialogFooter({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <footer className={cn("rcl-dialog__footer", className)} {...props} />;
}
