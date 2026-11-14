import { forwardRef, useCallback, useEffect, useImperativeHandle, useRef, useState } from "react";
import type { KeyboardEvent, ReactNode } from "react";
import { Database, ImagePlus, Loader2, MessageSquarePlus, SendHorizontal } from "lucide-react";
import { useAutoResizeTextarea } from "../../hooks/useAutoResizeTextarea";
import type { CaptureAttachment } from "../../hooks/useIndexedDBAttachments";
import { cn } from "../../lib/utils";
import type { AgentSessionContextType } from "../../types";
import { AttachmentPreviewTray } from "./AttachmentPreviewTray";
import { ContextChipTray, type ComposerContextChip } from "./ContextChipTray";
import { MicButton } from "./MicButton";

const ACCEPTED_IMAGE_TYPES = "image/jpeg,image/png,image/gif,image/webp";
const MIN_TEXTAREA_ROWS = 2;
const MAX_TEXTAREA_ROWS = 6;

export interface MessageComposerHandle {
  /**
   * Replace the composer text through the browser's editing pipeline so the
   * user's native undo history survives (Ctrl+Z restores what they had typed).
   * A plain controlled-state set would discard that history.
   */
  replaceText: (text: string) => void;
}

interface MessageComposerProps {
  value: string;
  onChange: (value: string) => void;
  onSubmit: () => void;
  disabled?: boolean;
  isSubmitting?: boolean;
  placeholder?: string;
  submitLabel?: string;
  testId?: string;
  inputTestId?: string;
  attachTestId?: string;
  contextTestId?: string;
  submitTestId?: string;
  className?: string;
  footer?: ReactNode;
  attachments?: CaptureAttachment[];
  onAttachFiles?: (files: File[]) => void;
  onRemoveAttachment?: (id: string) => void;
  contextItems?: ComposerContextChip[];
  onOpenContextPicker?: () => void;
  onRemoveContext?: (type: AgentSessionContextType, ref: string) => void;
  onOpenContext?: (path: string) => void;
  onOpenForm?: (draftText: string) => void;
  canSubmit?: boolean;
  imagePickerRequestKey?: number;
  onTranscript?: (text: string) => void;
  micTestId?: string;
}

export const MessageComposer = forwardRef<MessageComposerHandle, MessageComposerProps>(function MessageComposer({
  value,
  onChange,
  onSubmit,
  disabled = false,
  isSubmitting = false,
  placeholder = "Write a message...",
  submitLabel = "Send",
  testId,
  inputTestId,
  attachTestId,
  contextTestId,
  submitTestId,
  className,
  footer,
  attachments = [],
  onAttachFiles,
  onRemoveAttachment,
  contextItems = [],
  onOpenContextPicker,
  onRemoveContext,
  onOpenContext,
  onOpenForm,
  canSubmit,
  imagePickerRequestKey = 0,
  onTranscript,
  micTestId,
}: MessageComposerProps, ref) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const overlayRef = useRef<HTMLDivElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [interimTranscript, setInterimTranscript] = useState("");
  const materializedInterimRef = useRef("");
  useAutoResizeTextarea(textareaRef, value, { minRows: MIN_TEXTAREA_ROWS, maxRows: MAX_TEXTAREA_ROWS });

  useEffect(() => {
    const textarea = textareaRef.current;
    const overlay = overlayRef.current;
    if (textarea && overlay) overlay.scrollTop = textarea.scrollTop;
  }, [value]);

  const appendInterim = useCallback((base: string, interim: string) => {
    const left = base.trimEnd();
    const right = interim.trim();
    if (!right) return left;
    if (!left) return right;
    return /^[,.!?;:]/.test(right) ? `${left}${right}` : `${left} ${right}`;
  }, []);

  const materializeInterim = useCallback(() => {
    if (!interimTranscript.trim()) return;
    onChange(appendInterim(value, interimTranscript));
    materializedInterimRef.current = interimTranscript.trim();
    setInterimTranscript("");
  }, [appendInterim, interimTranscript, onChange, value]);

  const handleTranscript = useCallback((text: string) => {
    const incoming = text.trim();
    const materialized = materializedInterimRef.current.trim();
    materializedInterimRef.current = "";
    setInterimTranscript("");
    if (!incoming || !onTranscript) return;
    // If the user edited while a partial was visible, that partial is already
    // settled in the textarea. Replace only its overlapping prefix when the
    // provider later emits the corresponding segment-final.
    if (materialized && incoming === materialized) return;
    if (materialized && incoming.startsWith(materialized)) {
      const remainder = incoming.slice(materialized.length).trim();
      if (remainder) onTranscript(remainder);
      return;
    }
    onTranscript(text);
  }, [onTranscript]);

  const handleComposerChange = useCallback((next: string) => {
    if (interimTranscript.trim()) {
      onChange(appendInterim(next, interimTranscript));
      materializedInterimRef.current = interimTranscript.trim();
      setInterimTranscript("");
      return;
    }
    onChange(next);
  }, [appendInterim, interimTranscript, onChange]);

  useImperativeHandle(ref, () => ({
    replaceText: (text: string) => {
      const textarea = textareaRef.current;
      // execCommand routes the edit through the browser's undo stack and
      // fires a native input event, so the controlled state stays in sync.
      if (textarea && !textarea.disabled && typeof document.execCommand === "function") {
        textarea.focus();
        textarea.select();
        try {
          if (document.execCommand("insertText", false, text)) return;
        } catch {
          // Unsupported here — fall through to the controlled set.
        }
      }
      onChange(text);
    },
  }), [onChange]);

  const computedCanSubmit = canSubmit ?? Boolean(value.trim() || attachments.length > 0 || contextItems.length > 0);
  const submitDisabled = !computedCanSubmit || disabled || isSubmitting;

  useEffect(() => {
    if (imagePickerRequestKey <= 0 || disabled || isSubmitting) return;
    fileInputRef.current?.click();
  }, [disabled, imagePickerRequestKey, isSubmitting]);

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLTextAreaElement>) => {
      if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
        event.preventDefault();
        if (!submitDisabled) onSubmit();
      }
      if (event.key === "Escape") {
        textareaRef.current?.blur();
      }
    },
    [onSubmit, submitDisabled],
  );

  const handleFileSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(event.target.files ?? []);
    if (files.length > 0) {
      onAttachFiles?.(files);
    }
    event.target.value = "";
  };

  return (
    <div className="space-y-2">
      {onRemoveAttachment && <AttachmentPreviewTray attachments={attachments} onRemove={onRemoveAttachment} />}
      <ContextChipTray
        items={contextItems}
        onRemove={onRemoveContext}
        onOpen={onOpenContext}
        testId={testId ? `${testId}-context-chips` : undefined}
      />

      <div className={cn("flex min-h-0 items-end gap-2 rounded-lg border border-slate-700 bg-slate-900/70 p-2.5 transition-colors focus-within:border-cyan-500/50 focus-within:ring-1 focus-within:ring-cyan-500/20", className)}>
        {onOpenForm && (
          <button
            type="button"
            onClick={() => onOpenForm(value.trim())}
            disabled={disabled || isSubmitting}
            className="mb-0.5 shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-300 disabled:opacity-50"
            title="Create manually"
          >
            <MessageSquarePlus className="h-5 w-5" />
          </button>
        )}

        <div className="relative min-w-0 flex-1">
          <div
            ref={overlayRef}
            aria-hidden="true"
            className={cn("pointer-events-none absolute inset-0 z-0 min-h-[2.5rem] overflow-hidden whitespace-pre-wrap break-words text-base leading-6 text-slate-200 transition-opacity", !interimTranscript && "opacity-0")}
            data-testid={testId ? `${testId}-overlay` : "composer-overlay"}
          >
            <span>{value}</span>
            {interimTranscript && (
              <span data-testid={testId ? `${testId}-interim` : "composer-interim"} className="border-b border-dotted border-cyan-400/90 text-cyan-200">
                {value && !/[\s,.!?;:]$/.test(value) && !/^[,.!?;:]/.test(interimTranscript) ? " " : ""}
                {interimTranscript}
              </span>
            )}
          </div>
          <textarea
            ref={textareaRef}
            value={value}
            onChange={(event) => handleComposerChange(event.target.value)}
            onFocus={materializeInterim}
            onKeyDown={handleKeyDown}
            onScroll={(event) => {
              if (overlayRef.current) overlayRef.current.scrollTop = event.currentTarget.scrollTop;
            }}
            placeholder={placeholder}
            rows={MIN_TEXTAREA_ROWS}
            disabled={disabled || isSubmitting}
            aria-label={placeholder}
            className={cn("relative z-10 min-h-[2.5rem] w-full resize-none bg-transparent text-base leading-6 caret-slate-200 placeholder:text-slate-500 outline-none selection:bg-cyan-500/30 disabled:opacity-50", interimTranscript ? "text-transparent" : "text-slate-200")}
            data-testid={inputTestId ?? testId}
          />
        </div>

        {onTranscript && (
          <MicButton
            onTranscript={handleTranscript}
            onPartialTranscript={setInterimTranscript}
            disabled={disabled || isSubmitting}
            testId={micTestId ?? (testId ? `${testId}-mic` : undefined)}
          />
        )}

        {onOpenContextPicker && (
          <button
            type="button"
            onClick={onOpenContextPicker}
            disabled={disabled || isSubmitting}
            className="mb-0.5 shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-300 disabled:opacity-50"
            title="Attach context"
            data-testid={contextTestId ?? (testId ? `${testId}-context` : undefined)}
          >
            <Database className="h-4 w-4" />
          </button>
        )}

        {onAttachFiles && (
          <>
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              disabled={disabled || isSubmitting}
              className="mb-0.5 shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-300 disabled:opacity-50"
              title="Attach image"
              data-testid={attachTestId ?? (testId ? `${testId}-attach` : undefined)}
            >
              <ImagePlus className="h-4 w-4" />
            </button>
            <input
              ref={fileInputRef}
              type="file"
              accept={ACCEPTED_IMAGE_TYPES}
              multiple
              className="hidden"
              onChange={handleFileSelect}
            />
          </>
        )}

        <button
          type="button"
          onClick={onSubmit}
          disabled={submitDisabled}
          className="mb-0.5 inline-flex shrink-0 items-center gap-1 rounded px-1.5 py-1 text-cyan-500 transition-colors hover:bg-cyan-500/10 hover:text-cyan-400 disabled:text-slate-600 disabled:hover:bg-transparent"
          title={`${submitLabel} (Ctrl+Enter)`}
          data-testid={submitTestId ?? (testId ? `${testId}-submit` : undefined)}
        >
          {isSubmitting ? <Loader2 className="h-4 w-4 animate-spin" /> : <SendHorizontal className="h-4 w-4" />}
          <span className="sr-only">{submitLabel}</span>
        </button>
      </div>

      {footer}
    </div>
  );
});
