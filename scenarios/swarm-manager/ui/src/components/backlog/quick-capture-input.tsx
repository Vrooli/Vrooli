/**
 * Quick Capture Input
 *
 * A minimal text input for the unified action feed's "All" tab.
 * Users type or speak a raw thought and press Enter to capture it.
 * Classification starts automatically on submit.
 */

import { useCallback, useRef, useState } from "react";
import { MessageSquarePlus } from "lucide-react";
import { captureService } from "../../services/capture-service";
import { useCaptureStore } from "../../stores/capture-store";
import { selectors } from "../../consts/selectors";

export function QuickCaptureInput() {
  const [text, setText] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const addCapture = useCaptureStore((s) => s.addCapture);
  const lastSubmitRef = useRef<number>(0);

  const handleSubmit = useCallback(async () => {
    const trimmed = text.trim();
    if (!trimmed || isSubmitting) return;

    // Debounce rapid submissions (300ms).
    const now = Date.now();
    if (now - lastSubmitRef.current < 300) return;
    lastSubmitRef.current = now;

    setIsSubmitting(true);
    setText("");

    try {
      const response = await captureService.create(trimmed);
      addCapture(response.capture);
    } catch {
      // Restore text on failure so user doesn't lose their thought.
      setText(trimmed);
    } finally {
      setIsSubmitting(false);
      inputRef.current?.focus();
    }
  }, [text, isSubmitting, addCapture]);

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === "Escape") {
      inputRef.current?.blur();
    }
  };

  return (
    <div className="relative mb-4" data-testid={selectors.captures.quickInput}>
      <div className="flex items-start gap-3 rounded-xl border border-slate-700 bg-slate-800/50 p-3 transition-colors focus-within:border-cyan-500/50 focus-within:bg-slate-800">
        <MessageSquarePlus className="mt-1 h-5 w-5 shrink-0 text-slate-500" />
        <textarea
          ref={inputRef}
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="What's on your mind?"
          rows={1}
          disabled={isSubmitting}
          className="w-full resize-none bg-transparent text-sm text-slate-200 placeholder-slate-500 outline-none disabled:opacity-50"
          data-testid={selectors.captures.quickInputSubmit}
        />
        {isSubmitting && (
          <div className="mt-1 h-4 w-4 shrink-0 animate-spin rounded-full border-2 border-cyan-500 border-t-transparent" />
        )}
      </div>
      <p className="mt-1 px-1 text-xs text-slate-600">
        Press Enter to capture. Shift+Enter for new line.
      </p>
    </div>
  );
}
