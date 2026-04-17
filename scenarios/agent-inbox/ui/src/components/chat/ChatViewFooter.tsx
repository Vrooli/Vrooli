/**
 * ChatViewFooter - Bottom section of ChatView with MessageInput and agent modals.
 *
 * Extracted from ChatView to keep each module under 300 lines.
 */
import { ErrorBoundary } from "../ErrorBoundary";
import { MessageInput, type MessagePayload } from "./MessageInput";
import { AgentStartModal } from "./AgentStartModal";
import { AttachRunModal } from "./AttachRunModal";
import { uploadAgentAttachment } from "../../lib/api";
import type { Model, Message } from "../../lib/api";
import type { useAgentChatMode } from "./useAgentChatMode";

interface ChatViewFooterProps {
  agent: ReturnType<typeof useAgentChatMode>;
  isGenerating: boolean;
  models: Model[];
  chatModel: string;
  chatId: string;
  chatWebSearchEnabled: boolean;
  editingMessage?: Message | null;
  onCancelEdit?: () => void;
  onSubmitEdit?: (payload: MessagePayload) => void;
  onTemplateActivated?: (templateId: string, toolIds: string[]) => Promise<void>;
  activeTemplateId?: string | null;
  onTemplateDeactivate?: () => void;
}

export function ChatViewFooter({
  agent,
  isGenerating,
  models,
  chatModel,
  chatId,
  chatWebSearchEnabled,
  editingMessage,
  onCancelEdit,
  onSubmitEdit,
  onTemplateActivated,
  activeTemplateId,
  onTemplateDeactivate,
}: ChatViewFooterProps) {
  return (
    <>
      <div className="border-t border-white/10 bg-slate-950/50">
        {agent.queuedMessage && (
          <div className="flex items-center justify-between px-4 py-2 bg-blue-500/10 border-b border-blue-500/20">
            <span className="text-xs text-blue-300">Message queued -- will send when agent finishes</span>
            <button onClick={() => agent.setQueuedMessage(null)} className="text-xs text-blue-400 hover:text-blue-200">Cancel</button>
          </div>
        )}
        <ErrorBoundary name="MessageInput">
          <MessageInput
            onSend={agent.handleSendMessage}
            isLoading={isGenerating || agent.isStartingAgent}
            currentModel={models.find((m) => m.id === chatModel) || null}
            chatId={chatId}
            chatWebSearchDefault={chatWebSearchEnabled}
            editingMessage={editingMessage}
            onCancelEdit={onCancelEdit}
            onSubmitEdit={onSubmitEdit}
            onTemplateActivated={onTemplateActivated}
            activeTemplateId={activeTemplateId}
            onTemplateDeactivate={onTemplateDeactivate}
            customUploadFn={agent.chatMode === "agent" ? uploadAgentAttachment : undefined}
          />
        </ErrorBoundary>
      </div>

      <AgentStartModal
        isOpen={agent.showAgentStartModal}
        onClose={() => { agent.setShowAgentStartModal(false); agent.setPendingAgentMessage(""); }}
        onStart={(config) => { void agent.handleStartAgent(config); }}
        defaultSettings={agent.agentSettings}
        isLoading={agent.isStartingAgent}
        error={agent.agentError}
      />

      <AttachRunModal
        isOpen={agent.showAttachModal}
        onClose={() => agent.setShowAttachModal(false)}
        onAttach={(run) => { void agent.handleAttachRun(run); }}
        isLoading={agent.isAttaching}
      />
    </>
  );
}
