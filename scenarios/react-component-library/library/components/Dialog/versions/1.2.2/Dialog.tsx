/**
 * @libraryId react-component-library:Dialog
 * @displayName Dialog
 * @description Accessible modal dialog shell with escape/backdrop dismissal and mobile-safe sizing.
 * @version 1.2.2
 * @tags ["overlay","interactive"]
 * @deps {"react":"^18","lucide-react":"^0.424.0"}
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import { X } from "lucide-react";
import { type ReactNode, useEffect, useId } from "react";
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
}: DialogProps) {
  const id = useId();
  const titleID = `${id}-title`;
  const descriptionID = `${id}-description`;
  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onClose]);

  if (!open) {
    return null;
  }

  return (
    <div data-rcl-dialog className="rcl-dialog">
      <StyleSheet name="dialog-1-2-2" css={dialogStyles} />
      <button
        data-testid="overlays.dialog"
        type="button"
        aria-label={closeLabel}
        className="rcl-dialog__backdrop"
        onClick={onClose}
      />
      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleID}
        aria-describedby={description ? descriptionID : undefined}
        className={cn("rcl-dialog__surface", className)}
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
            data-testid="overlays.dialog"
            type="button"
            aria-label={closeLabel}
            className="rcl-dialog__close"
            onClick={onClose}
          >
            <X aria-hidden width="20" height="20" />
          </button>
        </header>
        <div className="rcl-dialog__body">{children}</div>
        {footer && <footer className="rcl-dialog__footer">{footer}</footer>}
      </section>
    </div>
  );
});
