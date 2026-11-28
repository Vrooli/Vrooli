/**
 * EvidenceRequestMessages — Renders a scrollable chat thread for evidence requests.
 *
 * Thin domain adapter around the shared chat thread.
 */

import { ChatThread } from "../chat/ChatThread";
import type { RequestMessage } from "../../services/review-service";
import { selectors } from "../../consts/selectors";
import type { ChatMessageView } from "../chat/chat-types";

interface EvidenceRequestMessagesProps {
  messages: RequestMessage[];
  isWaitingForAgent: boolean;
}

export function EvidenceRequestMessages({ messages, isWaitingForAgent }: EvidenceRequestMessagesProps) {
  const chatMessages: ChatMessageView[] = messages.map((msg, index) => ({
    id: `${msg.role}-${index}`,
    role: msg.role,
    content: msg.content,
  }));

  return (
    <ChatThread
      messages={chatMessages}
      isWaiting={isWaitingForAgent}
      accent="violet"
      className="px-3 py-2"
      testId={selectors.evidenceRequest.messageList}
      renderAttachmentPreview={(message) => {
        const idParts = message.id.split("-");
        const sourceMessage = messages[Number(idParts[idParts.length - 1])];
        const count = sourceMessage?.added_evidence_ids?.length ?? 0;
        return count > 0 ? (
          <div className="mt-2 rounded bg-violet-500/10 px-2 py-1 text-xs text-violet-300">
            Added {count} evidence item{count !== 1 ? "s" : ""}
          </div>
        ) : null;
      }}
    />
  );
}
