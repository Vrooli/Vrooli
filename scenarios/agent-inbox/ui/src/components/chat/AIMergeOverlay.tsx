/**
 * Overlay shown when selecting a template while message input has existing text.
 * Offers options to overwrite, merge with AI, or cancel.
 */

import { useCallback, useEffect, useRef } from "react";
import { X, Replace, Blend, Loader2 } from "lucide-react";
import type { MergeAction } from "@/lib/types/templates";

interface AIMergeOverlayProps {
  isOpen: boolean;
  existingMessage: string;
  templateName: string;
  isMerging: boolean;
  onAction: (action: MergeAction) => void;
}

export function AIMergeOverlay({
  isOpen,
  existingMessage,
  templateName,
  isMerging,
  onAction,
}: AIMergeOverlayProps) {
  const overlayRef = useRef<HTMLDivElement>(null);

  // Handle escape key to cancel
  useEffect(() => {
    if (!isOpen) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape" && !isMerging) {
        onAction("cancel");
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [isOpen, isMerging, onAction]);

  // Focus trap
  useEffect(() => {
    if (isOpen && overlayRef.current) {
      overlayRef.current.focus();
    }
  }, [isOpen]);

  const handleOverwrite = useCallback(() => {
    if (!isMerging) {
      onAction("overwrite");
    }
  }, [isMerging, onAction]);

  const handleMerge = useCallback(() => {
    if (!isMerging) {
      onAction("merge");
    }
  }, [isMerging, onAction]);

  const handleCancel = useCallback(() => {
    if (!isMerging) {
      onAction("cancel");
    }
  }, [isMerging, onAction]);

  if (!isOpen) return null;

  // Truncate message for preview
  const truncatedMessage =
    existingMessage.length > 100
      ? existingMessage.slice(0, 100) + "..."
      : existingMessage;

  return (
    <div
      ref={overlayRef}
      tabIndex={-1}
      className="absolute inset-0 z-10 flex items-center justify-center rounded-xl overflow-hidden"
      role="dialog"
      aria-modal="true"
      aria-labelledby="merge-dialog-title"
    >
      {/* Blurred background showing existing message */}
      <div className="absolute inset-0 bg-slate-900/80 backdrop-blur-sm" />

      {/* Dialog content */}
      <div className="relative bg-slate-800 border border-white/10 rounded-lg p-4 mx-4 max-w-md w-full shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between mb-3">
          <h3
            id="merge-dialog-title"
            className="text-sm font-medium text-white"
          >
            Template: {templateName}
          </h3>
          <button
            onClick={handleCancel}
            disabled={isMerging}
            className="p-1 rounded hover:bg-white/10 text-slate-400 hover:text-white transition-colors disabled:opacity-50"
            aria-label="Cancel"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Existing message preview */}
        <div className="mb-4 p-2 rounded bg-slate-700/50 border border-white/5">
          <p className="text-xs text-slate-400 mb-1">Current message:</p>
          <p className="text-sm text-slate-300 italic">{truncatedMessage}</p>
        </div>

        {/* Description */}
        <p className="text-xs text-slate-400 mb-4">
          You have existing text in your message. How would you like to apply
          this template?
        </p>

        {/* Action buttons */}
        <div className="flex flex-col gap-2">
          <button
            onClick={handleOverwrite}
            disabled={isMerging}
            className="flex items-center justify-center gap-2 w-full px-4 py-2 rounded-lg bg-indigo-600 text-white text-sm font-medium hover:bg-indigo-500 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Replace className="h-4 w-4" />
            Overwrite with template
          </button>

          <button
            onClick={handleMerge}
            disabled={isMerging}
            className="flex items-center justify-center gap-2 w-full px-4 py-2 rounded-lg bg-white/10 text-white text-sm font-medium hover:bg-white/20 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isMerging ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Merging...
              </>
            ) : (
              <>
                <Blend className="h-4 w-4" />
                Merge with AI
              </>
            )}
          </button>

          <button
            onClick={handleCancel}
            disabled={isMerging}
            className="w-full px-4 py-2 text-sm text-slate-400 hover:text-white transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Cancel
          </button>
        </div>

        {/* Keyboard hint */}
        <p className="mt-3 text-xs text-slate-500 text-center">
          Press <kbd className="px-1 py-0.5 bg-slate-700 rounded">Esc</kbd> to
          cancel
        </p>
      </div>
    </div>
  );
}
