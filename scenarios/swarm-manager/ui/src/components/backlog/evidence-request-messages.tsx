/**
 * EvidenceRequestMessages — Renders a scrollable chat thread for evidence requests.
 *
 * User messages appear right-aligned, assistant messages left-aligned.
 * Uses violet accent to distinguish from clarification threads (cyan).
 * Shows a spinner when waiting for an agent response.
 */

import { useEffect, useRef } from "react";
import { renderMarkdown } from "../../lib/render-markdown";
import { Loader2 } from "lucide-react";
import { cn } from "../../lib";
import type { RequestMessage } from "../../services/review-service";
import { selectors } from "../../consts/selectors";

interface EvidenceRequestMessagesProps {
  messages: RequestMessage[];
  isWaitingForAgent: boolean;
}

export function EvidenceRequestMessages({ messages, isWaitingForAgent }: EvidenceRequestMessagesProps) {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages.length, isWaitingForAgent]);

  return (
    <div className="flex-1 space-y-3 overflow-y-auto px-3 py-2" data-testid={selectors.evidenceRequest.messageList}>
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
                : "border border-violet-500/20 bg-violet-500/5 text-slate-200",
            )}
          >
            <div className="prose-sm-slate" dangerouslySetInnerHTML={{ __html: renderMarkdown(msg.content) }} />
            {msg.added_evidence_ids && msg.added_evidence_ids.length > 0 && (
              <div className="mt-2 rounded bg-violet-500/10 px-2 py-1 text-xs text-violet-300">
                Added {msg.added_evidence_ids.length} evidence item{msg.added_evidence_ids.length !== 1 ? "s" : ""}
              </div>
            )}
          </div>
        </div>
      ))}

      {isWaitingForAgent && (
        <div className="flex justify-start">
          <div className="flex items-center gap-2 rounded-lg border border-violet-500/20 bg-violet-500/5 px-3 py-2 text-sm text-slate-400">
            <Loader2 className="h-3.5 w-3.5 animate-spin text-violet-400" />
            Thinking...
          </div>
        </div>
      )}

      <div ref={bottomRef} />
    </div>
  );
}
