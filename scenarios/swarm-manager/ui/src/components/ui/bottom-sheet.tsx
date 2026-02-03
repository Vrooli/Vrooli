/**
 * BottomSheet Component
 *
 * Mobile-first sheet overlay for compact actions and contextual panels.
 */

import type { ReactNode } from "react";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";

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
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-end justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/60 backdrop-blur-sm"
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Sheet */}
      <div
        className={cn(
          "relative z-10 w-full max-w-lg rounded-t-2xl border border-white/10 bg-slate-900 shadow-2xl",
          className
        )}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? "bottom-sheet-title" : undefined}
        aria-describedby={description ? "bottom-sheet-description" : undefined}
        data-testid={testId}
      >
        <div className="flex items-center justify-between border-b border-white/10 px-4 py-3">
          <div className="space-y-0.5">
            {title && (
              <h2 id="bottom-sheet-title" className="text-base font-semibold text-slate-100">
                {title}
              </h2>
            )}
            {description && (
              <p id="bottom-sheet-description" className="text-xs text-slate-400">
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
            contentClassName
          )}
        >
          {children}
        </div>

        {footer && (
          <div className="border-t border-white/10 px-4 py-3">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}
