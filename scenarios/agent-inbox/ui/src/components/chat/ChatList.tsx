import { useState, useMemo, forwardRef } from "react";
import { Search, MessageSquare, Terminal, Star, Loader2, Inbox, Archive, X } from "lucide-react";
import type { Chat, Label } from "../../lib/api";
import type { View } from "../../hooks/useChats";
import { Badge } from "../ui/badge";

interface ChatListProps {
  chats: Chat[];
  labels: Label[];
  selectedChatId: string | null;
  currentView: View;
  isLoading: boolean;
  onSelectChat: (chatId: string) => void;
  onNewChat: () => void;
}

export const ChatList = forwardRef<HTMLInputElement, ChatListProps>(function ChatList(
  {
    chats,
    labels,
    selectedChatId,
    currentView,
    isLoading,
    onSelectChat,
    onNewChat,
  },
  ref
) {
  const [searchQuery, setSearchQuery] = useState("");

  const filteredChats = useMemo(() => {
    if (!searchQuery.trim()) return chats;
    const query = searchQuery.toLowerCase();
    return chats.filter(
      (chat) =>
        chat.name.toLowerCase().includes(query) ||
        chat.preview.toLowerCase().includes(query)
    );
  }, [chats, searchQuery]);

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

    if (diffDays === 0) {
      return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    } else if (diffDays === 1) {
      return "Yesterday";
    } else if (diffDays < 7) {
      return date.toLocaleDateString([], { weekday: "short" });
    } else {
      return date.toLocaleDateString([], { month: "short", day: "numeric" });
    }
  };

  const getLabelById = (id: string) => labels.find((l) => l.id === id);

  const viewLabels: Record<View, { title: string; icon: typeof Inbox; emptyMessage: string }> = {
    inbox: {
      title: "Inbox",
      icon: Inbox,
      emptyMessage: "No chats yet. Start a new conversation to get going.",
    },
    starred: {
      title: "Starred",
      icon: Star,
      emptyMessage: "No starred chats. Star important conversations to find them quickly.",
    },
    archived: {
      title: "Archived",
      icon: Archive,
      emptyMessage: "No archived chats. Archive conversations you want to keep but hide from inbox.",
    },
  };

  const { title, icon: ViewIcon, emptyMessage } = viewLabels[currentView];

  return (
    <div className="w-full lg:w-80 border-r border-white/10 flex flex-col bg-slate-950/50 shrink-0" data-testid="chat-list-panel">
      {/* Header */}
      <div className="p-4 border-b border-white/10">
        <div className="flex items-center gap-2 mb-3">
          <ViewIcon className="h-4 w-4 text-slate-400" />
          <h2 className="text-sm font-semibold text-white">{title}</h2>
          <span className="text-xs text-slate-500">({filteredChats.length})</span>
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input
            ref={ref}
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search chats... (Ctrl+K)"
            className="w-full bg-white/5 border border-white/10 rounded-lg pl-9 pr-8 py-2 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
            data-testid="chat-search-input"
          />
          {searchQuery && (
            <button
              onClick={() => setSearchQuery("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-white/10 text-slate-500 hover:text-white"
            >
              <X className="h-3 w-3" />
            </button>
          )}
        </div>
      </div>

      {/* Chat List */}
      <div className="flex-1 overflow-y-auto" data-testid="chat-list">
        {isLoading ? (
          <div className="flex flex-col items-center justify-center py-12 text-slate-500">
            <Loader2 className="h-6 w-6 animate-spin mb-2" />
            <p className="text-sm">Loading chats...</p>
          </div>
        ) : filteredChats.length === 0 ? (
          <div className="p-6 text-center">
            {searchQuery ? (
              <>
                <Search className="h-10 w-10 mx-auto mb-3 text-slate-600" />
                <p className="text-sm text-slate-500">No chats match "{searchQuery}"</p>
                <button
                  onClick={() => setSearchQuery("")}
                  className="mt-2 text-sm text-indigo-400 hover:text-indigo-300"
                >
                  Clear search
                </button>
              </>
            ) : (
              <>
                <ViewIcon className="h-10 w-10 mx-auto mb-3 text-slate-600" />
                <p className="text-sm text-slate-500 mb-4">{emptyMessage}</p>
                {currentView === "inbox" && (
                  <button
                    onClick={onNewChat}
                    className="text-sm text-indigo-400 hover:text-indigo-300 font-medium"
                  >
                    Create your first chat
                  </button>
                )}
              </>
            )}
          </div>
        ) : (
          filteredChats.map((chat) => (
            <ChatListItem
              key={chat.id}
              chat={chat}
              labels={chat.label_ids.map(getLabelById).filter(Boolean) as Label[]}
              isSelected={selectedChatId === chat.id}
              onClick={() => onSelectChat(chat.id)}
              formatTime={formatTime}
            />
          ))
        )}
      </div>
    </div>
  );
});

interface ChatListItemProps {
  chat: Chat;
  labels: Label[];
  isSelected: boolean;
  onClick: () => void;
  formatTime: (date: string) => string;
}

function ChatListItem({ chat, labels, isSelected, onClick, formatTime }: ChatListItemProps) {
  return (
    <button
      onClick={onClick}
      className={`w-full p-3 border-b border-white/5 text-left transition-colors ${
        isSelected
          ? "bg-indigo-500/20 border-l-2 border-l-indigo-500"
          : !chat.is_read
          ? "bg-indigo-500/5 hover:bg-indigo-500/10"
          : "hover:bg-white/5"
      }`}
      data-testid={`chat-item-${chat.id}`}
    >
      <div className="flex items-start gap-3">
        {/* Chat Icon */}
        <div className={`mt-0.5 p-1.5 rounded-lg ${isSelected ? "bg-indigo-500/30" : "bg-white/10"}`}>
          {chat.view_mode === "terminal" ? (
            <Terminal className="h-4 w-4 text-slate-400" />
          ) : (
            <MessageSquare className="h-4 w-4 text-slate-400" />
          )}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            <span
              className={`text-sm truncate ${
                !chat.is_read ? "font-semibold text-white" : "font-medium text-slate-300"
              }`}
            >
              {chat.name}
            </span>
            <div className="flex items-center gap-1.5 shrink-0">
              {chat.is_starred && (
                <Star className="h-3 w-3 text-yellow-500 fill-yellow-500" />
              )}
              <span className="text-xs text-slate-500">{formatTime(chat.updated_at)}</span>
            </div>
          </div>

          <p className="text-xs text-slate-500 truncate mt-1">
            {chat.preview || "No messages yet"}
          </p>

          {/* Labels */}
          {labels.length > 0 && (
            <div className="flex items-center gap-1 mt-2 flex-wrap">
              {labels.slice(0, 3).map((label) => (
                <Badge key={label.id} color={label.color} className="text-[10px] py-0">
                  {label.name}
                </Badge>
              ))}
              {labels.length > 3 && (
                <span className="text-[10px] text-slate-500">+{labels.length - 3}</span>
              )}
            </div>
          )}
        </div>

        {/* Unread Indicator */}
        {!chat.is_read && !isSelected && (
          <span
            className="w-2 h-2 bg-indigo-500 rounded-full shrink-0 mt-2"
            data-testid="unread-indicator"
          />
        )}
      </div>
    </button>
  );
}
