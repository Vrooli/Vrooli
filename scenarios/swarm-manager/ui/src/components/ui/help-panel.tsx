/**
 * HelpPanel - Shared shell for the lens-aware workspace guides.
 *
 * Desktop: a Popover anchored to the header's help button (bottom-end), with
 * a sticky title/close header. Mobile: the shared BottomSheet via Popover's
 * mobileSheet mode (the sheet renders its own header, so the desktop header
 * is skipped there).
 */

import { useEffect, useRef, type ReactNode, type RefObject } from "react";
import { HelpCircle, X } from "lucide-react";
import { Popover } from "./popover";
import { cn } from "../../lib/utils";
import { useIsMobile } from "../../hooks/useMediaQuery";
import { useSpatialNavContext } from "../../hooks/SpatialNavContext";

interface HelpPanelProps {
  isOpen: boolean;
  onClose: () => void;
  /** Guide title ("Plan Guide" / "Graph Guide") */
  title: string;
  /** Header help button to anchor the desktop popover to */
  triggerRef?: RefObject<HTMLElement | null>;
  testId?: string;
  children: ReactNode;
}

export function HelpPanel({ isOpen, onClose, title, triggerRef, testId, children }: HelpPanelProps) {
  const isMobile = useIsMobile();

  // Push a spatial nav modal scope so D-pad navigation is trapped inside.
  // (BottomSheet's Dialog pushes its own scope in sheet mode.)
  const spatialNavRef = useSpatialNavContext();
  const bodyRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const ctrl = spatialNavRef?.current;
    const el = bodyRef.current;
    if (!isOpen || isMobile || !ctrl || !el) return;
    ctrl.pushScope(el);
    return () => { ctrl.popScope(); };
  }, [isMobile, isOpen, spatialNavRef]);

  return (
    <Popover
      isOpen={isOpen}
      onClose={onClose}
      triggerRef={triggerRef}
      placement="bottom-end"
      mobileSheet
      mobileTitle={title}
      testId={testId}
      className="flex max-h-[70vh] w-80 flex-col overflow-hidden rounded-lg border-slate-700/60 bg-slate-900/95 shadow-xl backdrop-blur-sm"
    >
      <div ref={bodyRef} className="flex min-h-0 flex-col">
        {!isMobile && (
          <div className="flex shrink-0 items-center justify-between border-b border-slate-700/60 px-3 py-2">
            <div className="flex items-center gap-2">
              <HelpCircle className="h-4 w-4 text-slate-400" />
              <span className="text-sm font-semibold text-slate-100">{title}</span>
            </div>
            <button
              type="button"
              onClick={onClose}
              className="rounded p-1 text-slate-400 hover:bg-slate-800 hover:text-slate-200"
              aria-label="Close help"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        )}
        {/* Sheet mode already pads its content area; only pad the popover card. */}
        <div className={cn("min-h-0 flex-1 overflow-y-auto", !isMobile && "p-3")}>{children}</div>
      </div>
    </Popover>
  );
}
