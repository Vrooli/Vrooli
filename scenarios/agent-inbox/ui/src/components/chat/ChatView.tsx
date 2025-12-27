import { Loader2 } from "lucide-react";
import { ChatHeader } from "./ChatHeader";
import { MessageList } from "./MessageList";
import { MessageInput } from "./MessageInput";
import type { ChatWithMessages, Model, Label } from "../../lib/api";
import type { ActiveToolCall } from "../../hooks/useChats";
import type { ViewMode } from "../settings/Settings";

interface ChatViewProps {
  chatData: ChatWithMessages | null;
  models: Model[];
  labels: Label[];
  isLoading: boolean;
  isGenerating: boolean;
  streamingContent: string;
  activeToolCalls?: ActiveToolCall[];
  scrollToMessageId?: string | null;
  onScrollComplete?: () => void;
  onSendMessage: (content: string) => void;
  onUpdateChat: (data: { name?: string; model?: string }) => void;
  onToggleRead: () => void;
  onToggleStar: () => void;
  onToggleArchive: () => void;
  onDeleteChat: () => void;
  onAssignLabel: (labelId: string) => void;
  onRemoveLabel: (labelId: string) => void;
  viewMode?: ViewMode;
}

export function ChatView({
  chatData,
  models,
  labels,
  isLoading,
  isGenerating,
  streamingContent,
  activeToolCalls = [],
  scrollToMessageId,
  onScrollComplete,
  onSendMessage,
  onUpdateChat,
  onToggleRead,
  onToggleStar,
  onToggleArchive,
  onDeleteChat,
  onAssignLabel,
  onRemoveLabel,
  viewMode,
}: ChatViewProps) {
  if (isLoading) {
    return (
      <div className="flex-1 flex items-center justify-center bg-slate-950" data-testid="chat-view-loading">
        <div className="text-center">
          <Loader2 className="h-8 w-8 animate-spin text-indigo-400 mx-auto mb-3" />
          <p className="text-sm text-slate-500">Loading conversation...</p>
        </div>
      </div>
    );
  }

  if (!chatData) {
    return null;
  }

  return (
    <div className="flex-1 flex flex-col bg-slate-950" data-testid="chat-view">
      <ChatHeader
        chat={chatData.chat}
        models={models}
        labels={labels}
        onUpdateChat={(data) => onUpdateChat(data)}
        onToggleRead={onToggleRead}
        onToggleStar={onToggleStar}
        onToggleArchive={onToggleArchive}
        onDelete={onDeleteChat}
        onAssignLabel={onAssignLabel}
        onRemoveLabel={onRemoveLabel}
      />

      <MessageList
        messages={chatData.messages || []}
        isGenerating={isGenerating}
        streamingContent={streamingContent}
        activeToolCalls={activeToolCalls}
        scrollToMessageId={scrollToMessageId}
        onScrollComplete={onScrollComplete}
        viewMode={viewMode}
      />

      <MessageInput onSend={onSendMessage} isGenerating={isGenerating} />
    </div>
  );
}
