/**
 * Quick Capture Input
 *
 * Multi-line auto-resizing text input with image attachment support.
 * Users type raw thoughts, optionally attach images, and submit via
 * the send button or Ctrl+Enter. Classification starts automatically.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { captureService } from "../../services/capture-service";
import { useCaptureStore } from "../../stores/capture-store";
import { useComposerImageAttachments } from "../composer/useComposerImageAttachments";
import { MessageComposer } from "../composer/MessageComposer";
import { selectors } from "../../consts/selectors";

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
  const [submitError, setSubmitError] = useState<string | null>(null);
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
  const addCapture = useCaptureStore((s) => s.addCapture);
  const lastSubmitRef = useRef(0);
  const { attachments, addFile, removeFile, clearAll, getFiles } = useComposerImageAttachments("swarm-capture-attachments");

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
    setSubmitError(null);
    const savedText = text;
    setText("");

    try {
      const files = getFiles();
      const response = await captureService.create(trimmed, files.length > 0 ? files : undefined);
      addCapture(response.capture);
      clearAll();
      try { localStorage.removeItem(DRAFT_KEY); } catch { /* ignore */ }
    } catch (err) {
      // Restore text on failure so user doesn't lose their thought.
      setText(savedText);
      setSubmitError(
        err instanceof Error ? err.message : "Failed to submit capture. Check that the server is running.",
      );
    } finally {
      setIsSubmitting(false);
    }
  }, [text, attachments.length, isSubmitting, addCapture, getFiles, clearAll]);

  return (
    <div className="relative" data-testid={selectors.captures.quickInput}>
      <MessageComposer
        value={text}
        onChange={setText}
        onSubmit={handleSubmit}
        disabled={isSubmitting}
        isSubmitting={isSubmitting}
        placeholder="What's on your mind?"
        submitLabel="Send capture"
        inputTestId={selectors.captures.quickInputSubmit}
        attachTestId={selectors.captures.quickInputAttach}
        submitTestId={selectors.captures.quickInputSend}
        attachments={attachments}
        onAttachFiles={(files) => files.forEach(addFile)}
        onRemoveAttachment={removeFile}
        onOpenForm={
          onOpenForm
            ? (draftText) => {
                onOpenForm(draftText);
                setText("");
                try { localStorage.removeItem(DRAFT_KEY); } catch { /* ignore */ }
              }
            : undefined
        }
        canSubmit={Boolean(canSubmit)}
        onTranscript={(transcribed) => setText((prev) => (prev ? prev.trimEnd() + " " : "") + transcribed)}
      />

      {submitError ? (
        <p className="mt-1 px-1 text-xs text-red-400">{submitError}</p>
      ) : (
        <p className="mt-1 px-1 text-xs text-slate-600">
          Ctrl+Enter to capture
        </p>
      )}
    </div>
  );
}
