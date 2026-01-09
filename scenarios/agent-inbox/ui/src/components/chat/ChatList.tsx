import { useState, useMemo, forwardRef, useRef, useEffect } from "react";
import { Search, MessageSquare, Star, Loader2, Inbox, Archive, X, Check, FileText } from "lucide-react";
import type { Chat, Label, SearchResult } from "../../lib/api";
import type { View } from "../../hooks/useChats";
import { useSearch } from "../../hooks/useSearch";
import { Badge } from "../ui/badge";

interface ChatListProps {
  chats: Chat[];
  labels: Label[];
  selectedChatId: string | null;
  /** Index of focused chat for j/k keyboard navigation (-1 = none) */
  focusedIndex?: number;
  currentView: View;
  isLoading: boolean;
  onSelectChat: (chatId: string, messageId?: string) => void;
  onNewChat: () => void;
  onRenameChat?: (chatId: string, newName: string) => void;
}

export const ChatList = forwardRef<HTMLInputElement, ChatListProps>(function ChatList(
  {
    chats,
    labels,
    selectedChatId,
    focusedIndex = -1,
    currentView,
    isLoading,
    onSelectChat,
    onNewChat,
    onRenameChat,
  },
  ref
) {
  // Refs for each chat item to enable scroll-into-view on focus
  const itemRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  // Scroll focused item into view when focusedIndex changes
  useEffect(() => {
    if (focusedIndex >= 0) {
      const element = itemRefs.current.get(focusedIndex);
      if (element && typeof element.scrollIntoView === "function") {
        element.scrollIntoView({ block: "nearest", behavior: "smooth" });
      }
    }
  }, [focusedIndex]);
  // Server-side search
  const search = useSearch({ debounceMs: 300, limit: 20 });

  // When not searching, show chats from props; when searching, show search results
  const displayChats = useMemo(() => {
    if (!search.isActive) return chats;
    // Extract unique chats from search results
    return search.results.map((r) => r.chat);
  }, [chats, search.isActive, search.results]);

  // Build a map of search results by chat ID for snippet display
  const searchResultsMap = useMemo(() => {
    const map = new Map<string, SearchResult>();
    if (search.isActive) {
      for (const result of search.results) {
        // Keep the first (highest ranked) result per chat
        if (!map.has(result.chat.id)) {
          map.set(result.chat.id, result);
        }
      }
    }
    return map;
  }, [search.isActive, search.results]);

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
          <h2 className="text-sm font-semibold text-white">
            {search.isActive ? "Search Results" : title}
          </h2>
          <span className="text-xs text-slate-500">
            ({search.isActive ? search.results.length : displayChats.length})
          </span>
          {search.isSearching && (
            <Loader2 className="h-3 w-3 animate-spin text-slate-400" />
          )}
        </div>

        {/* Search */}
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input
            ref={ref}
            type="text"
            value={search.query}
            onChange={(e) => search.setQuery(e.target.value)}
            placeholder="Search messages... (Ctrl+K)"
            className="w-full bg-white/5 border border-white/10 rounded-lg pl-9 pr-8 py-2 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50"
            data-testid="chat-search-input"
          />
          {search.query && (
            <button
              onClick={search.clear}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-white/10 text-slate-500 hover:text-white"
              data-testid="clear-search-button"
            >
              <X className="h-3 w-3" />
            </button>
          )}
        </div>
      </div>

      {/* Chat List */}
      <div className="flex-1 overflow-y-auto" data-testid="chat-list">
        {isLoading && !search.isActive ? (
          <div className="flex flex-col items-center justify-center py-12 text-slate-500">
            <Loader2 className="h-6 w-6 animate-spin mb-2" />
            <p className="text-sm">Loading chats...</p>
          </div>
        ) : displayChats.length === 0 ? (
          <div className="p-6 text-center">
            {search.isActive ? (
              <>
                <Search className="h-10 w-10 mx-auto mb-3 text-slate-600" />
                <p className="text-sm text-slate-500">
                  {search.isSearching
                    ? "Searching..."
                    : `No results for "${search.query}"`}
                </p>
                <button
                  onClick={search.clear}
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
          displayChats.map((chat, index) => {
            const searchResult = searchResultsMap.get(chat.id);
            return (
              <ChatListItem
                key={chat.id}
                ref={(el) => {
                  if (el) itemRefs.current.set(index, el);
                  else itemRefs.current.delete(index);
                }}
                chat={chat}
                labels={(chat.label_ids || []).map(getLabelById).filter(Boolean) as Label[]}
                isSelected={selectedChatId === chat.id}
                isFocused={focusedIndex === index}
                onClick={() => onSelectChat(chat.id, searchResult?.message_id)}
                onRename={onRenameChat ? (newName) => onRenameChat(chat.id, newName) : undefined}
                formatTime={formatTime}
                searchResult={searchResult}
              />
            );
          })
        )}
      </div>
    </div>
  );
});

interface ChatListItemProps {
  chat: Chat;
  labels: Label[];
  isSelected: boolean;
  /** Whether this item is focused via j/k navigation (shown with ring) */
  isFocused?: boolean;
  onClick: () => void;
  onRename?: (newName: string) => void;
  formatTime: (date: string) => string;
  searchResult?: SearchResult;
}

const ChatListItem = forwardRef<HTMLDivElement, ChatListItemProps>(function ChatListItem(
  { chat, labels, isSelected, isFocused, onClick, onRename, formatTime, searchResult },
  ref
) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(chat.name);
  const inputRef = useRef<HTMLInputElement>(null);

  // Focus input when entering edit mode
  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

  // Sync editValue when chat name changes externally
  useEffect(() => {
    if (!isEditing) {
      setEditValue(chat.name);
    }
  }, [chat.name, isEditing]);

  const handleDoubleClick = (e: React.MouseEvent) => {
    if (!onRename) return;
    e.stopPropagation();
    setIsEditing(true);
  };

  const handleSave = () => {
    const trimmed = editValue.trim();
    if (trimmed && trimmed !== chat.name && onRename) {
      onRename(trimmed);
    }
    setIsEditing(false);
  };

  const handleCancel = () => {
    setEditValue(chat.name);
    setIsEditing(false);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      e.preventDefault();
      handleSave();
    } else if (e.key === "Escape") {
      e.preventDefault();
      handleCancel();
    }
  };

  return (
    <div
      ref={ref}
      role="button"
      tabIndex={0}
      onClick={isEditing ? undefined : onClick}
      onKeyDown={(e) => {
        if (!isEditing && (e.key === "Enter" || e.key === " ")) {
          e.preventDefault();
          onClick();
        }
      }}
      className={`w-full p-3 border-b border-white/5 text-left transition-colors cursor-pointer ${
        isSelected
          ? "bg-indigo-500/20 border-l-2 border-l-indigo-500"
          : !chat.is_read
          ? "bg-indigo-500/5 hover:bg-indigo-500/10"
          : "hover:bg-white/5"
      } ${isFocused ? "ring-2 ring-indigo-400 ring-inset" : ""}`}
      data-testid={`chat-item-${chat.id}`}
      data-focused={isFocused}
    >
      <div className="flex items-start gap-3">
        {/* Chat Icon */}
        <div className={`mt-0.5 p-1.5 rounded-lg ${isSelected ? "bg-indigo-500/30" : "bg-white/10"}`}>
          <MessageSquare className="h-4 w-4 text-slate-400" />
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center justify-between gap-2">
            {isEditing ? (
              <div className="flex items-center gap-1 flex-1 min-w-0" onClick={(e) => e.stopPropagation()}>
                <input
                  ref={inputRef}
                  type="text"
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onKeyDown={handleKeyDown}
                  onBlur={handleSave}
                  className="flex-1 min-w-0 bg-white/10 border border-indigo-500 rounded px-2 py-0.5 text-sm text-white focus:outline-none focus:ring-1 focus:ring-indigo-500"
                  data-testid="inline-rename-input"
                />
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleSave();
                  }}
                  className="p-1 rounded hover:bg-white/10 text-green-400"
                  data-testid="inline-rename-save"
                >
                  <Check className="h-3 w-3" />
                </button>
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleCancel();
                  }}
                  className="p-1 rounded hover:bg-white/10 text-slate-400"
                  data-testid="inline-rename-cancel"
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            ) : (
              <span
                onDoubleClick={handleDoubleClick}
                className={`text-sm truncate cursor-pointer ${
                  !chat.is_read ? "font-semibold text-white" : "font-medium text-slate-300"
                } ${onRename ? "hover:text-indigo-300" : ""}`}
                title={onRename ? "Double-click to rename" : undefined}
                data-testid="chat-name"
              >
                {chat.name}
              </span>
            )}
            <div className="flex items-center gap-1.5 shrink-0">
              {chat.is_starred && (
                <Star className="h-3 w-3 text-yellow-500 fill-yellow-500" />
              )}
              <span className="text-xs text-slate-500">{formatTime(chat.updated_at)}</span>
            </div>
          </div>

          {/* Preview or Search Snippet */}
          {searchResult?.snippet ? (
            <div className="mt-1">
              <div className="flex items-center gap-1 text-[10px] text-indigo-400 mb-0.5">
                <FileText className="h-3 w-3" />
                <span>{searchResult.match_type === "message_content" ? "Message match" : "Name match"}</span>
              </div>
              <p
                className="text-xs text-slate-400 line-clamp-2 [&>mark]:bg-yellow-500/30 [&>mark]:text-yellow-200 [&>mark]:px-0.5 [&>mark]:rounded"
                dangerouslySetInnerHTML={{ __html: searchResult.snippet }}
                data-testid="search-snippet"
              />
            </div>
          ) : (
            <p className="text-xs text-slate-500 truncate mt-1">
              {chat.preview || "No messages yet"}
            </p>
          )}

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
    </div>
  );
});
