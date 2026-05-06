/**
 * ClarificationMessages — Renders a scrollable chat thread for clarification.
 *
 * Thin domain adapter around the shared chat thread.
 */

import { ChatThread } from "../chat/ChatThread";
import type { ClarificationMessage } from "../../types/domain";
import type { ChatMessageView } from "../chat/chat-types";

interface ClarificationMessagesProps {
  messages: ClarificationMessage[];
  isWaitingForAgent: boolean;
}

export function ClarificationMessages({ messages, isWaitingForAgent }: ClarificationMessagesProps) {
  const chatMessages: ChatMessageView[] = messages.map((msg, index) => ({
    id: `${msg.role}-${index}`,
    role: msg.role,
    content: msg.content,
    attachmentIds: msg.attachment_ids,
  }));

  return (
    <ChatThread
      messages={chatMessages}
      isWaiting={isWaitingForAgent}
      accent="cyan"
      className="px-1 py-2"
      renderAttachmentPreview={(message) =>
        message.attachmentIds && message.attachmentIds.length > 0 ? (
          <div className="mt-2 flex gap-1.5 overflow-x-auto">
            {message.attachmentIds.map((id) => (
              <div key={id} className="h-16 w-16 shrink-0 overflow-hidden rounded border border-slate-600 bg-slate-800/50">
                <div className="flex h-full items-center justify-center text-[9px] text-slate-500">img</div>
              </div>
            ))}
          </div>
        ) : null
      }
    />
  );
}
