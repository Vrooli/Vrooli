/**
 * Confirmation Dialog Component
 *
 * A modal dialog that requires explicit user confirmation before proceeding with
 * destructive or irreversible actions. Provides a clear warning message and
 * requires intentional user input.
 *
 * [REQ:REQ-P0-008] Strong confirmation dialog for scenario deletion
 */

import { useState, useEffect, useRef, type ReactNode } from "react";
import { AlertTriangle, Check, Copy } from "lucide-react";
import { Button } from "./button";
import { BottomSheet } from "./bottom-sheet";
import { Dialog } from "./dialog";

interface ConfirmDialogProps {
  /** Whether the dialog is open */
  isOpen: boolean;
  /** Callback when dialog is closed */
  onClose: () => void;
  /** Callback when user confirms the action */
  onConfirm: () => void;
  /** Dialog title */
  title: string;
  /** Dialog description/message */
  description: string;
  /** Text to type for confirmation (if required) */
  confirmationText?: string;
  /** Label for the confirm button */
  confirmLabel?: string;
  /** Whether the confirm action is in progress */
  isLoading?: boolean;
  /** Visible error from a failed confirm action (dialog stays open). */
  errorMessage?: string;
  /** Optional checkbox content */
  checkboxContent?: {
    label: string;
    checked: boolean;
    onChange: (checked: boolean) => void;
    testId?: string;
  };
  /** Test IDs for the dialog elements */
  testIds?: {
    dialog?: string;
    confirmButton?: string;
    cancelButton?: string;
    copyButton?: string;
  };
  /** Optional side panel rendered inside the same dialog container */
  sidePanel?: ReactNode;
  /** Use a compact sheet on mobile while retaining a centered desktop overlay. */
  presentation?: "dialog" | "bottom-sheet";
}

export function ConfirmDialog({
  isOpen,
  onClose,
  onConfirm,
  title,
  description,
  confirmationText,
  confirmLabel = "Confirm",
  isLoading = false,
  errorMessage,
  checkboxContent,
  testIds,
  sidePanel,
  presentation = "dialog",
}: ConfirmDialogProps) {
  const [inputValue, setInputValue] = useState("");
  const [copied, setCopied] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  // Reset input when dialog opens
  useEffect(() => {
    if (isOpen) {
      setInputValue("");
      setCopied(false);
      // Focus input if confirmation text is required
      if (confirmationText) {
        setTimeout(() => inputRef.current?.focus(), 100);
      }
    }
  }, [isOpen, confirmationText]);

  // Clear the "copied" affordance after a short delay.
  useEffect(() => {
    if (!copied) return;
    const timer = setTimeout(() => setCopied(false), 1500);
    return () => clearTimeout(timer);
  }, [copied]);

  if (!isOpen) return null;

  const canConfirm = !confirmationText || inputValue === confirmationText;

  const handleCopy = async () => {
    if (!confirmationText) return;
    try {
      await navigator.clipboard.writeText(confirmationText);
    } catch {
      // Clipboard may be unavailable (insecure context / denied permission).
      // Fall back to pre-filling the input so the user can still confirm.
      setInputValue(confirmationText);
    }
    setCopied(true);
  };

  const content = (
    <div
      role="alertdialog"
      aria-labelledby="confirm-dialog-title"
      aria-describedby="confirm-dialog-description"
    >
      <div className={sidePanel ? "grid max-h-[85vh] gap-0 overflow-hidden lg:grid-cols-[1fr_340px]" : ""}>
        <div className={sidePanel ? "overflow-y-auto p-6 lg:p-7" : ""}>
          {/* Warning icon */}
          <div className="mb-4 flex items-center gap-3">
            <div className="rounded-full bg-red-500/20 p-3">
              <AlertTriangle className="h-6 w-6 text-red-400" />
            </div>
            <h2
              id="confirm-dialog-title"
              className="text-xl font-semibold text-slate-100"
            >
              {title}
            </h2>
          </div>

          {/* Description */}
          <p
            id="confirm-dialog-description"
            className="mb-4 text-slate-400"
          >
            {description}
          </p>

          {/* Optional checkbox */}
          {checkboxContent && (
            <label className="mb-4 flex cursor-pointer items-center gap-3 rounded-lg bg-slate-800/50 p-3">
              <input
                type="checkbox"
                checked={checkboxContent.checked}
                onChange={(e) => checkboxContent.onChange(e.target.checked)}
                className="h-4 w-4 rounded border-slate-600 bg-slate-700 text-cyan-500 focus:ring-cyan-500 focus:ring-offset-slate-900"
                data-testid={checkboxContent.testId}
              />
              <span className="text-sm text-slate-300">{checkboxContent.label}</span>
            </label>
          )}

          {/* Confirmation input */}
          {confirmationText && (
            <div className="mb-4">
              <div className="mb-2 flex items-center gap-2 text-sm text-slate-400">
                <span>
                  Type <code className="rounded bg-slate-800 px-1 py-0.5 text-red-400">{confirmationText}</code> to confirm:
                </span>
                <button
                  type="button"
                  onClick={handleCopy}
                  disabled={isLoading}
                  className="inline-flex items-center gap-1 rounded border border-slate-700 px-1.5 py-0.5 text-xs text-slate-300 hover:border-slate-500 hover:text-slate-100 disabled:opacity-50"
                  data-testid={testIds?.copyButton}
                  aria-label={copied ? "Copied" : "Copy confirmation text"}
                  title="Copy to clipboard"
                >
                  {copied ? <Check className="h-3 w-3 text-green-400" /> : <Copy className="h-3 w-3" />}
                  {copied ? "Copied" : "Copy"}
                </button>
              </div>
              <input
                ref={inputRef}
                type="text"
                value={inputValue}
                onChange={(e) => setInputValue(e.target.value)}
                className="w-full rounded-lg border border-slate-700 bg-slate-800 px-3 py-2 text-slate-100 placeholder-slate-500 focus:border-red-500 focus:outline-none focus:ring-1 focus:ring-red-500"
                placeholder={confirmationText}
                disabled={isLoading}
              />
            </div>
          )}

          {errorMessage && (
            <p role="alert" className="mb-4 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-300">
              {errorMessage}
            </p>
          )}

          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={onClose} disabled={isLoading} data-testid={testIds?.cancelButton}>Cancel</Button>
            <Button variant="destructive" onClick={onConfirm} disabled={!canConfirm || isLoading} data-testid={testIds?.confirmButton}>
              {isLoading ? "Processing..." : confirmLabel}
            </Button>
          </div>
        </div>
        {sidePanel && <aside className="border-t border-white/10 bg-slate-900/90 p-5 lg:border-l lg:border-t-0 lg:p-6">{sidePanel}</aside>}
      </div>
    </div>
  );

  if (presentation === "bottom-sheet") {
    return (
      <BottomSheet
        isOpen={isOpen}
        onClose={onClose}
        data-testid={testIds?.dialog}
        contentClassName="p-4"
      >
        {content}
      </BottomSheet>
    );
  }

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      maxWidth={sidePanel ? "max-w-5xl" : "max-w-md"}
      isLoading={isLoading}
      testId={testIds?.dialog}
      className={sidePanel ? "overflow-hidden p-0" : undefined}
    >
      {content}
    </Dialog>
  );
}
