/**
 * ClarificationMessages — Renders a scrollable chat thread for clarification.
 *
 * User messages appear right-aligned, assistant messages left-aligned.
 * Shows a spinner when waiting for an agent response.
 */

import { useEffect, useRef } from "react";
import { Loader2 } from "lucide-react";
import { cn } from "../../lib";
import type { ClarificationMessage } from "../../types/domain";

interface ClarificationMessagesProps {
  messages: ClarificationMessage[];
  isWaitingForAgent: boolean;
}

export function ClarificationMessages({ messages, isWaitingForAgent }: ClarificationMessagesProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages arrive or while waiting.
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length, isWaitingForAgent]);

  return (
    <div className="flex-1 space-y-3 overflow-y-auto px-1 py-2">
      {messages.map((msg, i) => (
        <div
          key={i}
          className={cn(
            "flex",
            msg.role === "user" ? "justify-end" : "justify-start",
          )}
        >
          <div
            className={cn(
              "max-w-[85%] rounded-lg px-3 py-2 text-sm",
              msg.role === "user"
                ? "bg-slate-700/60 text-slate-200"
                : "bg-cyan-500/5 border border-cyan-500/20 text-slate-200",
            )}
          >
            <p className="whitespace-pre-wrap">{msg.content}</p>
            {msg.attachment_ids && msg.attachment_ids.length > 0 && (
              <div className="mt-2 flex gap-1.5 overflow-x-auto">
                {msg.attachment_ids.map((id) => (
                  <div key={id} className="h-16 w-16 shrink-0 overflow-hidden rounded border border-slate-600 bg-slate-800/50">
                    <div className="flex h-full items-center justify-center text-[9px] text-slate-500">
                      img
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      ))}

      {isWaitingForAgent && (
        <div className="flex justify-start">
          <div className="flex items-center gap-2 rounded-lg bg-cyan-500/5 border border-cyan-500/20 px-3 py-2 text-sm text-slate-400">
            <Loader2 className="h-3.5 w-3.5 animate-spin text-cyan-400" />
            Thinking...
          </div>
        </div>
      )}

      <div ref={bottomRef} />
    </div>
  );
}
