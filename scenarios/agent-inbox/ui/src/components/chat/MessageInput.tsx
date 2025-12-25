import { useState, useCallback, useRef, useEffect } from "react";
import { Send, Loader2, Paperclip, Smile } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";

interface MessageInputProps {
  onSend: (content: string) => void;
  isGenerating: boolean;
  placeholder?: string;
}

export function MessageInput({
  onSend,
  isGenerating,
  placeholder = "Type a message...",
}: MessageInputProps) {
  const [message, setMessage] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // Auto-resize textarea
  useEffect(() => {
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = "auto";
      textarea.style.height = `${Math.min(textarea.scrollHeight, 200)}px`;
    }
  }, [message]);

  const handleSubmit = useCallback(() => {
    const trimmedMessage = message.trim();
    if (trimmedMessage && !isGenerating) {
      onSend(trimmedMessage);
      setMessage("");
      // Reset height
      if (textareaRef.current) {
        textareaRef.current.style.height = "auto";
      }
    }
  }, [message, isGenerating, onSend]);

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" && !e.shiftKey) {
        e.preventDefault();
        handleSubmit();
      }
    },
    [handleSubmit]
  );

  return (
    <div className="p-4 border-t border-white/10 bg-slate-950/50" data-testid="message-input-container">
      <div className="flex items-end gap-2">
        {/* Input Area */}
        <div className="flex-1 relative">
          <textarea
            ref={textareaRef}
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={isGenerating}
            rows={1}
            className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 pr-12 text-sm text-white placeholder:text-slate-500 resize-none focus:outline-none focus:ring-2 focus:ring-indigo-500/50 disabled:opacity-50"
            data-testid="message-input"
          />
          <div className="absolute right-3 bottom-3 flex items-center gap-1">
            {!isGenerating && message.length > 0 && (
              <span className="text-xs text-slate-600">{message.length}</span>
            )}
          </div>
        </div>

        {/* Send Button */}
        <Tooltip content={isGenerating ? "AI is responding..." : "Send message (Enter)"}>
          <Button
            onClick={handleSubmit}
            disabled={!message.trim() || isGenerating}
            size="icon"
            className="h-11 w-11 shrink-0"
            data-testid="send-message-button"
          >
            {isGenerating ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Send className="h-4 w-4" />
            )}
          </Button>
        </Tooltip>
      </div>

      {/* Keyboard hint */}
      <div className="flex items-center justify-between mt-2 px-1">
        <p className="text-xs text-slate-600">
          Press <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Enter</kbd> to
          send, <kbd className="px-1.5 py-0.5 rounded bg-white/10 text-slate-400">Shift+Enter</kbd>{" "}
          for new line
        </p>
        {isGenerating && (
          <span className="text-xs text-indigo-400 flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            AI is responding...
          </span>
        )}
      </div>
    </div>
  );
}
