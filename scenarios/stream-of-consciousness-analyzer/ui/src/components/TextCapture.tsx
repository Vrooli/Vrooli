// DOC: docs/concepts/ARCHITECTURE.md#ui-layer
import { useState, useRef, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Send } from "lucide-react";
import { createInformation } from "../lib/api";
import { ErrorBanner } from "./ErrorBanner";
import { INFO_PLACEMENT_WIDTH, INFO_PLACEMENT_HEIGHT, TEXT_CAPTURE_ROWS } from "../lib/config";
import { randomCanvasPosition } from "../lib/utils";

interface Props {
  schemeId: string;
}

export function TextCapture({ schemeId }: Props) {
  const [text, setText] = useState("");
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const qc = useQueryClient();

  // Auto-focus for sub-second text entry readiness [REQ:P0-002]
  useEffect(() => {
    inputRef.current?.focus({ preventScroll: true });
  }, [schemeId]);

  const createMut = useMutation({
    mutationFn: () => {
      const pos = randomCanvasPosition(INFO_PLACEMENT_WIDTH, INFO_PLACEMENT_HEIGHT);
      return createInformation(schemeId, {
        type: "text",
        content: text.trim(),
        canvas_x: pos.x,
        canvas_y: pos.y,
      });
    },
    onSuccess: () => {
      setText("");
      qc.invalidateQueries({ queryKey: ["information", schemeId] });
      inputRef.current?.focus({ preventScroll: true });
    },
  });

  const handleSubmit = () => {
    if (text.trim()) createMut.mutate();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
  };

  return (
    <div data-testid="text-capture" className="border-t border-white/10 bg-slate-900/80 p-3">
      {createMut.error && (
        <div className="mb-2">
          <ErrorBanner
            error={createMut.error}
            onRetry={() => createMut.mutate()}
            onDismiss={() => createMut.reset()}
          />
        </div>
      )}
      <div className="flex items-end gap-2">
        <textarea
          ref={inputRef}
          data-testid="text-capture-input"
          aria-label="Capture a thought"
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Capture a thought... (Enter to send)"
          rows={TEXT_CAPTURE_ROWS}
          className="flex-1 rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-white placeholder-slate-500 resize-none focus:outline-none focus:ring-1 focus:ring-white/30"
        />
        <button
          data-testid="text-capture-send"
          onClick={handleSubmit}
          disabled={!text.trim() || createMut.isPending}
          aria-label="Send thought"
          className="p-2 rounded-lg bg-white/10 text-white hover:bg-white/20 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          <Send className="h-4 w-4" aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
