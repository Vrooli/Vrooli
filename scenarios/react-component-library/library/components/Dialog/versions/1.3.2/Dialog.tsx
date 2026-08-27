/**
 * @libraryId react-component-library:Dialog
 * @displayName Dialog
 * @description
 * @version 1.3.2
 * @tags ["overlay","interactive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import { X } from "lucide-react";
import { type ReactNode, useId, type RefObject } from "react";
import { Portal } from "@vrooli/react-component-library/Portal/1.1.1";
import { useOverlaySurface } from "@vrooli/react-component-library/useOverlaySurface/1.1.1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { dialogStyles } from "./styles";
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
