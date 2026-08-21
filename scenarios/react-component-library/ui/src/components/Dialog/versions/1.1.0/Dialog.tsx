/**
 * @vrooliComponentSource overlays.dialog
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption 08c09296-b3da-4860-9b64-edd801000ae1
 * @vrooliComponentAppliedAt 2026-08-11T00:47:48Z
 * @vrooliComponentSourceSha256 5ac74dc8e5e06aab4cec656f8b02c3bf4899e42bdc6a7d38993f0339af975c82
 * @vrooliComponentDriftHash 98e3ab91a17675ef62c83064184d2fca2cdf57fd5b0d379b9ef6ad429f6a01d9
 * @vrooliComponentTokenTranslation none
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { X } from "lucide-react";
import { type ReactNode, useEffect, useId } from "react";
import { dialogStyles } from "./styles";
import { useComponentStyles } from "../../../../hooks/useComponentStyles";
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

export function Dialog({
  open,
  title,
  description,
  children,
  footer,
  onClose,
  closeLabel,
  className,
}: DialogProps) {
  // Unconditional: this component early-returns when `open` is false, so the
  // stylesheet request has to sit above that branch to keep hook order stable.
  useComponentStyles("rcl-dialog", dialogStyles);
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
      <button
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
}
