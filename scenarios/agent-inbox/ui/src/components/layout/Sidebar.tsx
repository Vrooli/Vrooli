import { useState, useMemo, forwardRef, useRef, useEffect, useCallback } from "react";
import {
  Plus,
  Mail,
  Star,
  Archive,
  Settings,
  Loader2,
  MessageSquare,
  Keyboard,
  Search,
  Inbox,
  X,
  Check,
  FileText,
  CheckSquare,
  Square,
  Trash2,
  MailOpen,
  MailCheck,
  ArchiveRestore,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { Badge } from "../ui/badge";
import { useSearch } from "../../hooks/useSearch";
import type { View } from "../../hooks/useChats";
import type { Chat, Label, SearchResult, BulkOperation } from "../../lib/api";

interface SidebarProps {
  currentView: View;
  onViewChange: (view: View) => void;
  onNewChat: () => void;
  onManageLabels: () => void;
  onOpenSettings: () => void;
  onShowKeyboardShortcuts: () => void;
  isCreatingChat: boolean;
  labels: Label[];
  chatCounts?: {
    inbox: number;
    starred: number;
    archived: number;
  };
  // Chat list props
  chats: Chat[];
  selectedChatId: string | null;
  focusedIndex?: number;
  isLoadingChats: boolean;
  onSelectChat: (chatId: string, messageId?: string) => void;
  onRenameChat?: (chatId: string, newName: string) => void;
  // Bulk selection props
  onBulkOperate?: (chatIds: string[], operation: BulkOperation, labelId?: string) => void;
  isBulkOperating?: boolean;
}

const navItems: { id: View; label: string; icon: typeof Mail }[] = [
  { id: "inbox", label: "Inbox", icon: Inbox },
  { id: "starred", label: "Starred", icon: Star },
  { id: "archived", label: "Archived", icon: Archive },
];

export const Sidebar = forwardRef<HTMLInputElement, SidebarProps>(function Sidebar(
  {
    currentView,
    onViewChange,
    onNewChat,
    onManageLabels,
    onOpenSettings,
    onShowKeyboardShortcuts,
    isCreatingChat,
    labels,
    chatCounts,
    chats,
    selectedChatId,
    focusedIndex = -1,
    isLoadingChats,
    onSelectChat,
    onRenameChat,
    onBulkOperate,
    isBulkOperating = false,
  },
  ref
) {
  // Refs for each chat item to enable scroll-into-view on focus
  const itemRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  // Bulk selection state
  const [selectionMode, setSelectionMode] = useState(false);
  const [selectedChatIds, setSelectedChatIds] = useState<Set<string>>(new Set());

  // Exit selection mode when view changes
  useEffect(() => {
    setSelectionMode(false);
    setSelectedChatIds(new Set());
  }, [currentView]);

  // Toggle selection for a chat
  const toggleChatSelection = useCallback((chatId: string, event: React.MouseEvent) => {
    event.stopPropagation();
    setSelectedChatIds((prev) => {
      const next = new Set(prev);
      if (next.has(chatId)) {
        next.delete(chatId);
      } else {
        next.add(chatId);
      }
      return next;
    });
  }, []);

  // Select all visible chats
  const selectAll = useCallback(() => {
    const displayChatIds = chats.map((c) => c.id);
    setSelectedChatIds(new Set(displayChatIds));
  }, [chats]);

  // Deselect all
  const deselectAll = useCallback(() => {
    setSelectedChatIds(new Set());
  }, []);

  // Exit selection mode
  const exitSelectionMode = useCallback(() => {
    setSelectionMode(false);
    setSelectedChatIds(new Set());
  }, []);

  // Execute bulk operation
  const handleBulkOperation = useCallback(
    (operation: BulkOperation) => {
      if (!onBulkOperate || selectedChatIds.size === 0) return;
      onBulkOperate(Array.from(selectedChatIds), operation);
      exitSelectionMode();
    },
    [onBulkOperate, selectedChatIds, exitSelectionMode]
  );

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
    return search.results.map((r) => r.chat);
  }, [chats, search.isActive, search.results]);

  // Build a map of search results by chat ID for snippet display
  const searchResultsMap = useMemo(() => {
    const map = new Map<string, SearchResult>();
    if (search.isActive) {
      for (const result of search.results) {
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

  const viewLabels: Record<View, { emptyMessage: string }> = {
    inbox: {
      emptyMessage: "No chats yet. Start a new conversation!",
    },
    starred: {
      emptyMessage: "No starred chats. Star important conversations to find them quickly.",
    },
    archived: {
      emptyMessage: "No archived chats.",
    },
  };

  const { emptyMessage } = viewLabels[currentView];

  return (
    <aside
      className="w-80 border-r border-white/10 flex flex-col bg-slate-950 shrink-0 h-full"
      data-testid="sidebar"
    >
      {/* Header with Logo + New Chat */}
      <div className="p-3 border-b border-white/10 shrink-0">
        <div className="flex items-center gap-2 mb-3">
          <MessageSquare className="h-5 w-5 text-indigo-400" />
          <h1 className="text-base font-semibold text-white">Agent Inbox</h1>
          <div className="flex-1" />
          {onBulkOperate && displayChats.length > 0 && !search.isActive && (
            <Tooltip content={selectionMode ? "Cancel selection" : "Select multiple"}>
              <button
                onClick={() => {
                  if (selectionMode) {
                    exitSelectionMode();
                  } else {
                    setSelectionMode(true);
                  }
                }}
                className={`p-1.5 rounded-lg transition-colors ${
                  selectionMode
                    ? "bg-indigo-500/20 text-indigo-400"
                    : "text-slate-500 hover:text-white hover:bg-white/10"
                }`}
                data-testid="toggle-selection-mode"
              >
                <CheckSquare className="h-4 w-4" />
              </button>
            </Tooltip>
          )}
        </div>
        <Button
          onClick={onNewChat}
          disabled={isCreatingChat || selectionMode}
          className="w-full justify-center gap-2"
          data-testid="new-chat-button"
        >
          {isCreatingChat ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <Plus className="h-4 w-4" />
          )}
          New Chat
        </Button>
      </div>

      {/* Bulk Actions Bar - shown when items are selected */}
      {selectionMode && selectedChatIds.size > 0 && (
        <div className="px-3 py-2 border-b border-white/10 bg-indigo-500/10 shrink-0" data-testid="bulk-actions-bar">
          <div className="flex items-center gap-2 mb-2">
            <span className="text-sm font-medium text-white">
              {selectedChatIds.size} selected
            </span>
            <div className="flex-1" />
            <button
              onClick={selectAll}
              className="text-xs text-indigo-400 hover:text-indigo-300"
            >
              Select all
            </button>
            <button
              onClick={deselectAll}
              className="text-xs text-slate-400 hover:text-white"
            >
              Clear
            </button>
          </div>
          <div className="flex gap-1 flex-wrap">
            <Tooltip content="Delete selected">
              <button
                onClick={() => handleBulkOperation("delete")}
                disabled={isBulkOperating}
                className="p-2 rounded-lg text-red-400 hover:bg-red-500/20 hover:text-red-300 disabled:opacity-50 transition-colors"
                data-testid="bulk-delete"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </Tooltip>
            {currentView !== "archived" ? (
              <Tooltip content="Archive selected">
                <button
                  onClick={() => handleBulkOperation("archive")}
                  disabled={isBulkOperating}
                  className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
                  data-testid="bulk-archive"
                >
                  <Archive className="h-4 w-4" />
                </button>
              </Tooltip>
            ) : (
              <Tooltip content="Unarchive selected">
                <button
                  onClick={() => handleBulkOperation("unarchive")}
                  disabled={isBulkOperating}
                  className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
                  data-testid="bulk-unarchive"
                >
                  <ArchiveRestore className="h-4 w-4" />
                </button>
              </Tooltip>
            )}
            <Tooltip content="Mark as read">
              <button
                onClick={() => handleBulkOperation("mark_read")}
                disabled={isBulkOperating}
                className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
                data-testid="bulk-mark-read"
              >
                <MailOpen className="h-4 w-4" />
              </button>
            </Tooltip>
            <Tooltip content="Mark as unread">
              <button
                onClick={() => handleBulkOperation("mark_unread")}
                disabled={isBulkOperating}
                className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
                data-testid="bulk-mark-unread"
              >
                <MailCheck className="h-4 w-4" />
              </button>
            </Tooltip>
            <Tooltip content="Star selected">
              <button
                onClick={() => {
                  // Toggle star - not directly supported by bulk API, but we can use archive as a workaround
                  // For now, just show a toast or skip
                }}
                disabled={true}
                className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-yellow-400 disabled:opacity-50 transition-colors"
                data-testid="bulk-star"
              >
                <Star className="h-4 w-4" />
              </button>
            </Tooltip>
          </div>
          {isBulkOperating && (
            <div className="flex items-center gap-2 mt-2 text-xs text-slate-400">
              <Loader2 className="h-3 w-3 animate-spin" />
              Processing...
            </div>
          )}
        </div>
      )}

      {/* Navigation Tabs */}
      <div className="px-3 py-2 border-b border-white/10 shrink-0" data-testid="sidebar-nav">
        <div className="flex gap-1">
          {navItems.map(({ id, label, icon: Icon }) => {
            const count = chatCounts?.[id];
            const isActive = currentView === id;

            return (
              <button
                key={id}
                onClick={() => onViewChange(id)}
                className={`flex-1 flex items-center justify-center gap-1.5 px-2 py-2 rounded-lg text-xs font-medium transition-colors ${
                  isActive
                    ? "bg-white/10 text-white"
                    : "text-slate-400 hover:text-white hover:bg-white/5"
                }`}
                data-testid={`nav-${id}`}
              >
                <Icon className={`h-3.5 w-3.5 ${isActive && id === "starred" ? "text-yellow-400" : ""}`} />
                <span className="hidden sm:inline">{label}</span>
                {count !== undefined && count > 0 && (
                  <span
                    className={`text-[10px] px-1 py-0.5 rounded-full ${
                      isActive ? "bg-white/20" : "bg-white/10"
                    }`}
                  >
                    {count}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Search */}
      <div className="px-3 py-2 shrink-0">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input
            ref={ref}
            type="text"
            value={search.query}
            onChange={(e) => search.setQuery(e.target.value)}
            placeholder="Search... (/ or Ctrl+K)"
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
          {search.isSearching && (
            <Loader2 className="absolute right-2 top-1/2 -translate-y-1/2 h-3 w-3 animate-spin text-slate-400" />
          )}
        </div>
      </div>

      {/* Chat List */}
      <div className="flex-1 overflow-y-auto" data-testid="chat-list">
        {isLoadingChats && !search.isActive ? (
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
                <MessageSquare className="h-10 w-10 mx-auto mb-3 text-slate-600" />
                <p className="text-sm text-slate-500 mb-4">{emptyMessage}</p>
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
                selectionMode={selectionMode}
                isChecked={selectedChatIds.has(chat.id)}
                onToggleSelect={(e) => toggleChatSelection(chat.id, e)}
              />
            );
          })
        )}
      </div>

      {/* Labels Section (collapsed into a row) */}
      {labels.length > 0 && (
        <div className="px-3 py-2 border-t border-white/10 shrink-0">
          <div className="flex items-center gap-2 overflow-x-auto scrollbar-hide">
            <span className="text-[10px] text-slate-500 uppercase tracking-wide shrink-0">Labels:</span>
            {labels.slice(0, 4).map((label) => (
              <Badge
                key={label.id}
                color={label.color}
                className="text-[10px] py-0.5 shrink-0 cursor-pointer hover:opacity-80"
                onClick={onManageLabels}
              >
                {label.name}
              </Badge>
            ))}
            {labels.length > 4 && (
              <button
                onClick={onManageLabels}
                className="text-[10px] text-slate-500 hover:text-white shrink-0"
              >
                +{labels.length - 4}
              </button>
            )}
          </div>
        </div>
      )}

      {/* Footer */}
      <div className="p-3 border-t border-white/10 shrink-0">
        <div className="flex items-center justify-center gap-1">
          <Tooltip content="Manage labels">
            <button
              onClick={onManageLabels}
              className="p-2 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
              data-testid="sidebar-labels-button"
            >
              <Badge color="#6366f1" className="h-4 w-4 p-0 flex items-center justify-center text-[8px]">
                {labels.length}
              </Badge>
            </button>
          </Tooltip>
          <Tooltip content="Keyboard shortcuts (?)">
            <button
              onClick={onShowKeyboardShortcuts}
              className="p-2 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
              data-testid="sidebar-shortcuts-button"
            >
              <Keyboard className="h-4 w-4" />
            </button>
          </Tooltip>
          <Tooltip content="Settings">
            <button
              onClick={onOpenSettings}
              className="p-2 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
              data-testid="sidebar-settings-button"
            >
              <Settings className="h-4 w-4" />
            </button>
          </Tooltip>
        </div>
      </div>
    </aside>
  );
});

interface ChatListItemProps {
  chat: Chat;
  labels: Label[];
  isSelected: boolean;
  isFocused?: boolean;
  onClick: () => void;
  onRename?: (newName: string) => void;
  formatTime: (date: string) => string;
  searchResult?: SearchResult;
  // Bulk selection props
  selectionMode?: boolean;
  isChecked?: boolean;
  onToggleSelect?: (e: React.MouseEvent) => void;
}

const ChatListItem = forwardRef<HTMLDivElement, ChatListItemProps>(function ChatListItem(
  { chat, labels, isSelected, isFocused, onClick, onRename, formatTime, searchResult, selectionMode, isChecked, onToggleSelect },
  ref
) {
  const [isEditing, setIsEditing] = useState(false);
  const [editValue, setEditValue] = useState(chat.name);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus();
      inputRef.current.select();
    }
  }, [isEditing]);

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

  const handleClick = selectionMode && onToggleSelect
    ? onToggleSelect
    : isEditing
    ? undefined
    : onClick;

  return (
    <div
      ref={ref}
      role="button"
      tabIndex={0}
      onClick={handleClick}
      onKeyDown={(e) => {
        if (!isEditing && (e.key === "Enter" || e.key === " ")) {
          e.preventDefault();
          if (selectionMode && onToggleSelect) {
            onToggleSelect(e as unknown as React.MouseEvent);
          } else {
            onClick();
          }
        }
      }}
      className={`w-full px-3 py-2.5 border-b border-white/5 text-left transition-colors cursor-pointer ${
        isChecked
          ? "bg-indigo-500/20"
          : isSelected
          ? "bg-indigo-500/20 border-l-2 border-l-indigo-500"
          : !chat.is_read
          ? "bg-indigo-500/5 hover:bg-indigo-500/10"
          : "hover:bg-white/5"
      } ${isFocused ? "ring-2 ring-indigo-400 ring-inset" : ""}`}
      data-testid={`chat-item-${chat.id}`}
      data-focused={isFocused}
    >
      <div className="flex items-start gap-2.5">
        {/* Checkbox in selection mode */}
        {selectionMode ? (
          <div
            className={`mt-0.5 p-1.5 rounded-lg shrink-0 transition-colors ${
              isChecked ? "bg-indigo-500 text-white" : "bg-white/10 text-slate-400 hover:bg-white/20"
            }`}
            data-testid={`chat-checkbox-${chat.id}`}
          >
            {isChecked ? (
              <CheckSquare className="h-3.5 w-3.5" />
            ) : (
              <Square className="h-3.5 w-3.5" />
            )}
          </div>
        ) : (
          /* Chat Icon */
          <div className={`mt-0.5 p-1.5 rounded-lg shrink-0 ${isSelected ? "bg-indigo-500/30" : "bg-white/10"}`}>
            <MessageSquare className="h-3.5 w-3.5 text-slate-400" />
          </div>
        )}

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
            <div className="flex items-center gap-1 shrink-0">
              {chat.is_starred && (
                <Star className="h-3 w-3 text-yellow-500 fill-yellow-500" />
              )}
              <span className="text-[10px] text-slate-500">{formatTime(chat.updated_at)}</span>
            </div>
          </div>

          {/* Preview or Search Snippet */}
          {searchResult?.snippet ? (
            <div className="mt-1">
              <div className="flex items-center gap-1 text-[10px] text-indigo-400 mb-0.5">
                <FileText className="h-3 w-3" />
                <span>{searchResult.match_type === "message_content" ? "Message" : "Name"}</span>
              </div>
              <p
                className="text-xs text-slate-400 line-clamp-2 [&>mark]:bg-yellow-500/30 [&>mark]:text-yellow-200 [&>mark]:px-0.5 [&>mark]:rounded"
                dangerouslySetInnerHTML={{ __html: searchResult.snippet }}
                data-testid="search-snippet"
              />
            </div>
          ) : (
            <p className="text-xs text-slate-500 truncate mt-0.5">
              {chat.preview || "No messages yet"}
            </p>
          )}

          {/* Labels */}
          {labels.length > 0 && (
            <div className="flex items-center gap-1 mt-1.5 flex-wrap">
              {labels.slice(0, 2).map((label) => (
                <Badge key={label.id} color={label.color} className="text-[9px] py-0">
                  {label.name}
                </Badge>
              ))}
              {labels.length > 2 && (
                <span className="text-[9px] text-slate-500">+{labels.length - 2}</span>
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
