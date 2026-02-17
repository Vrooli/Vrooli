/**
 * BottomSheet Component
 *
 * Mobile-first sheet overlay for compact actions and contextual panels.
 * Built on Dialog for consistent dismiss behavior (Esc, click-outside, scroll lock).
 */

import { useId, type ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";
import { Dialog } from "./dialog";

interface BottomSheetProps {
  /** Whether the sheet is open */
  isOpen: boolean;
  /** Callback when sheet is closed */
  onClose: () => void;
  /** Optional title */
  title?: string;
  /** Optional description */
  description?: string;
  /** Sheet content */
  children: ReactNode;
  /** Optional footer content */
  footer?: ReactNode;
  /** Optional className for the sheet container */
  className?: string;
  /** Optional className for the content area */
  contentClassName?: string;
  /** data-testid attribute */
  "data-testid"?: string;
}

export function BottomSheet({
  isOpen,
  onClose,
  title,
  description,
  children,
  footer,
  className,
  contentClassName,
  "data-testid": testId,
}: BottomSheetProps) {
  const titleElementId = useId();
  const descriptionElementId = useId();

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="max-w-lg"
      testId={testId}
      titleId={title ? titleElementId : undefined}
      descriptionId={description ? descriptionElementId : undefined}
      className={cn(
        "!items-end !rounded-t-2xl !rounded-b-none !mx-0 !mb-0 !p-0",
        "animate-in slide-in-from-bottom duration-200",
        className,
      )}
    >
      <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
        <div className="space-y-0.5">
          {title && (
            <h2 id={titleElementId} className="text-base font-semibold text-slate-100">
              {title}
            </h2>
          )}
          {description && (
            <p id={descriptionElementId} className="text-xs text-slate-400">
              {description}
            </p>
          )}
        </div>
        <button
          type="button"
          onClick={onClose}
          className="rounded-full p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
          aria-label="Close sheet"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      <div
        className={cn(
          "max-h-[70vh] overflow-y-auto px-4 py-4 pb-[calc(1rem+env(safe-area-inset-bottom))]",
          contentClassName,
        )}
      >
        {children}
      </div>

      {footer && (
        <div className="border-t border-white/10 px-4 py-3">
          {footer}
        </div>
      )}
    </Dialog>
  );
}
