/**
 * Quick Capture Input
 *
 * Multi-line auto-resizing text input with image attachment support.
 * Users type raw thoughts, optionally attach images, and submit via
 * the send button or Ctrl+Enter. Classification starts automatically.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { Loader2, MessageSquarePlus, Paperclip, SendHorizontal } from "lucide-react";
import { captureService } from "../../services/capture-service";
import { useCaptureStore } from "../../stores/capture-store";
import { useCaptureAttachments } from "../../hooks/useCaptureAttachments";
import { CaptureAttachmentPreview } from "./capture-attachment-preview";
import { selectors } from "../../consts/selectors";

/** Max visible lines before the textarea scrolls. */
const MAX_VISIBLE_LINES = 4;
/** Approximate line height in px. */
const LINE_HEIGHT_PX = 20;
/** Max textarea height: lines × height + padding. */
const MAX_TEXTAREA_HEIGHT = MAX_VISIBLE_LINES * LINE_HEIGHT_PX + 12;

const DRAFT_KEY = "swarm-capture-draft";
const DRAFT_DEBOUNCE_MS = 300;

interface QuickCaptureInputProps {
  /** Called when the user taps the form icon to create manually. Receives current draft text. */
  onOpenForm?: (draftText: string) => void;
}

export function QuickCaptureInput({ onOpenForm }: QuickCaptureInputProps) {
  const [text, setText] = useState(() => {
    try {
      return localStorage.getItem(DRAFT_KEY) ?? "";
    } catch {
      return "";
    }
  });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const draftTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Persist text draft to localStorage (debounced).
  useEffect(() => {
    if (draftTimerRef.current !== null) clearTimeout(draftTimerRef.current);
    draftTimerRef.current = setTimeout(() => {
      try {
        if (text) {
          localStorage.setItem(DRAFT_KEY, text);
        } else {
          localStorage.removeItem(DRAFT_KEY);
        }
      } catch {
        // Storage full or unavailable — ignore.
      }
    }, DRAFT_DEBOUNCE_MS);
    return () => {
      if (draftTimerRef.current !== null) clearTimeout(draftTimerRef.current);
    };
  }, [text]);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const addCapture = useCaptureStore((s) => s.addCapture);
  const lastSubmitRef = useRef<number>(0);
  const { attachments, addFile, removeFile, clearAll, getFiles } = useCaptureAttachments();

  // Auto-resize textarea based on content.
  useEffect(() => {
    const el = inputRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, MAX_TEXTAREA_HEIGHT)}px`;
  }, [text]);

  const canSubmit = (text.trim() || attachments.length > 0) && !isSubmitting;

  const handleSubmit = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed && attachments.length === 0) return;
    if (isSubmitting) return;

    // Debounce rapid submissions (300ms).
    const now = Date.now();
    if (now - lastSubmitRef.current < 300) return;
    lastSubmitRef.current = now;

    setIsSubmitting(true);
    const savedText = text;
    setText("");

    try {
      const files = getFiles();
      const response = await captureService.create(trimmed, files.length > 0 ? files : undefined);
      addCapture(response.capture);
      clearAll();
      try { localStorage.removeItem(DRAFT_KEY); } catch { /* ignore */ }
    } catch {
      // Restore text on failure so user doesn't lose their thought.
      setText(savedText);
    } finally {
      setIsSubmitting(false);
      inputRef.current?.focus();
    }
  }, [text, attachments.length, isSubmitting, addCapture, getFiles, clearAll]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Ctrl+Enter (or Cmd+Enter) to submit.
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      handleSubmit();
      return;
    }
    if (e.key === "Escape") {
      inputRef.current?.blur();
    }
  };

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const fileList = e.target.files;
    if (!fileList) return;
    Array.from(fileList).forEach(addFile);
    // Reset so the same file can be re-selected.
    e.target.value = "";
  };

  return (
    <div className="relative mb-4" data-testid={selectors.captures.quickInput}>
      <CaptureAttachmentPreview attachments={attachments} onRemove={removeFile} />

      <div className="flex items-end gap-2 rounded-xl border border-slate-700 bg-slate-800/50 p-3 transition-colors focus-within:border-cyan-500/50 focus-within:bg-slate-800">
        <button
          type="button"
          onClick={() => {
            onOpenForm?.(text.trim());
            setText("");
            try { localStorage.removeItem(DRAFT_KEY); } catch { /* ignore */ }
          }}
          className="mb-0.5 shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-300"
          title="Create manually"
        >
          <MessageSquarePlus className="h-5 w-5" />
        </button>

        <textarea
          ref={inputRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="What's on your mind?"
          disabled={isSubmitting}
          className="w-full resize-none bg-transparent text-sm text-slate-200 placeholder-slate-500 outline-none disabled:opacity-50"
          data-testid={selectors.captures.quickInputSubmit}
        />

        {/* Attach image button */}
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          disabled={isSubmitting}
          className="mb-0.5 shrink-0 rounded p-1 text-slate-500 transition-colors hover:bg-slate-700 hover:text-slate-300 disabled:opacity-50"
          title="Attach image"
          data-testid={selectors.captures.quickInputAttach}
        >
          <Paperclip className="h-4 w-4" />
        </button>

        {/* Submit button */}
        <button
          type="button"
          onClick={handleSubmit}
          disabled={!canSubmit}
          className="mb-0.5 shrink-0 rounded p-1 text-cyan-500 transition-colors hover:bg-cyan-500/10 hover:text-cyan-400 disabled:text-slate-600 disabled:hover:bg-transparent"
          title="Send capture (Ctrl+Enter)"
          data-testid={selectors.captures.quickInputSend}
        >
          {isSubmitting ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <SendHorizontal className="h-4 w-4" />
          )}
        </button>

        {/* Hidden file input */}
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png,image/gif,image/webp"
          multiple
          className="hidden"
          onChange={handleFileSelect}
        />
      </div>

      <p className="mt-1 px-1 text-xs text-slate-600">
        Ctrl+Enter to capture
      </p>
    </div>
  );
}
