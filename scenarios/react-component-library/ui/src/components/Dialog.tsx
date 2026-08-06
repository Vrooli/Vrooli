/**
 * @vrooliComponentSource react-component-library:Dialog
 * @vrooliComponentVersion 1.1.0
 * @vrooliComponentAdoption b86f1345-9172-494f-a8e8-865724cfa4fd
 * @vrooliComponentAppliedAt 2026-08-06T03:46:15Z
 * @vrooliComponentSourceSha256 985a8ffa01f06fa0649b1164b9e4044db0adf491286313ff72081989f2a38d37
 * @vrooliComponentDriftHash eb41ad021e77f0a4a759ebda21304a1716eb4a7c1167f09b0cfe9cbe9d4b8af5
 * @vrooliComponentTokenTranslation bg-app-shell/60->bg-app-shell/60,bg-app-surface->bg-app-surface,bg-app-surface-muted->bg-app-surface-muted,border-app-border->border-app-border,text-app-foreground->text-app-foreground,text-app-muted-foreground->text-app-muted-foreground
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { X } from "lucide-react";
import { type ReactNode, useEffect } from "react";
export const DIALOG_MODES = ["controlled", "uncontrolled"] as const;
export const DIALOG_PARTS = ["trigger", "overlay", "content", "header", "title", "description", "body", "footer", "close"] as const;

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

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

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
    <div className="fixed inset-0 z-50 flex items-end justify-center bg-app-shell/60 p-3 pt-safe pb-safe md:items-center">
      <button type="button" aria-label={closeLabel} className="absolute inset-0 cursor-default" onClick={onClose} />
      <section
        role="dialog"
        aria-modal="true"
        aria-labelledby="dialog-title"
        aria-describedby={description ? "dialog-description" : undefined}
        className={cn(
          "relative z-10 flex max-h-[calc(100dvh-2rem)] w-full max-w-lg flex-col overflow-hidden rounded-panel border border-app-border bg-app-surface text-app-foreground shadow-xl",
          className,
        )}
      >
        <header className="flex items-start justify-between gap-3 border-b border-app-border px-4 py-3">
          <div className="min-w-0">
            <h2 id="dialog-title" className="text-base font-semibold">{title}</h2>
            {description && (
              <p id="dialog-description" className="mt-1 text-sm text-app-muted-foreground">
                {description}
              </p>
            )}
          </div>
          <button
            type="button"
            aria-label={closeLabel}
            className="touch-target inline-flex items-center justify-center rounded-control text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground"
            onClick={onClose}
          >
            <X aria-hidden className="h-5 w-5" />
          </button>
        </header>
        <div className="min-h-0 flex-1 overflow-auto px-4 py-4">{children}</div>
        {footer && <footer className="border-t border-app-border px-4 py-3">{footer}</footer>}
      </section>
    </div>
  );
}
