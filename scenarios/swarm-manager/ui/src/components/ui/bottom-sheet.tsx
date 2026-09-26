/**
 * BottomSheet Component
 *
 * Mobile-first sheet overlay for compact actions and contextual panels.
 * Built on Dialog for consistent dismiss behavior (Esc, click-outside, scroll lock).
 */

import { useId, type ReactNode } from "react";
import { cn } from "../../lib/utils";
import { useIsMobile } from "../../hooks/useMediaQuery";
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
  /** Optional className for the fixed outer container */
  containerClassName?: string;
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
  containerClassName,
  contentClassName,
  "data-testid": testId,
}: BottomSheetProps) {
  const titleElementId = useId();
  const descriptionElementId = useId();
  const isMobile = useIsMobile();

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="max-w-lg"
      testId={testId}
      titleId={title ? titleElementId : undefined}
      descriptionId={description ? descriptionElementId : undefined}
      // Mobile: bottom sheet (anchored to viewport bottom).
      // Desktop: fall through to Dialog's centered (vertical + horizontal) modal.
      containerClassName={cn(isMobile && "!items-end", containerClassName)}
      className={cn(
        "flex flex-col !overflow-hidden !p-0",
        isMobile
          ? [
              "!max-h-[calc(100dvh-0.75rem)]",
              "!mx-0 !mb-0 !rounded-b-none rounded-t-2xl",
              "!animate-none",
              "animate-in slide-in-from-bottom duration-200",
            ]
          : // Desktop keeps Dialog defaults: mx-4, rounded-xl, max-h-[85vh], zoom-in.
            null,
        className,
      )}
    >
      {/* Header with title/description (X button provided by Dialog) */}
      {(title || description) && (
        <div className="border-b border-white/10 px-4 py-3 pr-12">
          {title && (
            <h2 id={titleElementId} className="text-base font-semibold text-slate-100">
              {title}
            </h2>
          )}
          {description && (
            <p id={descriptionElementId} className="mt-0.5 text-xs text-slate-400">
              {description}
            </p>
          )}
        </div>
      )}

      <div
        className={cn(
          "min-h-0 flex-1 overflow-y-auto px-4 py-4 pb-4",
          contentClassName,
        )}
      >
        {children}
      </div>

      {footer && (
        <div className="shrink-0 border-t border-white/10 px-4 pb-[max(2rem,calc(0.75rem+env(safe-area-inset-bottom)))] pt-3">
          {footer}
        </div>
      )}
    </Dialog>
  );
}
