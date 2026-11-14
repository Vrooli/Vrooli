import { useMemo, forwardRef, useRef, useEffect } from "react";
import { Search, Loader2, Inbox, Star, Archive, X } from "lucide-react";
import type { Chat, Label, SearchResult } from "../../lib/api";
import type { View } from "../../hooks/useChats";
import { useSearch } from "../../hooks/useSearch";
import { ChatListItem } from "./ChatListItem";

interface ChatListProps {
  chats: Chat[];
  labels: Label[];
  selectedChatId: string | null;
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
  const itemRefs = useRef<Map<number, HTMLDivElement>>(new Map());

  useEffect(() => {
    if (focusedIndex >= 0) {
      const element = itemRefs.current.get(focusedIndex);
      if (element && typeof element.scrollIntoView === "function") {
        element.scrollIntoView({ block: "nearest", behavior: "smooth" });
      }
    }
  }, [focusedIndex]);

  const search = useSearch({ debounceMs: 300, limit: 20 });

  const displayChats = useMemo(() => {
    if (!search.isActive) return chats;
    return search.results.map((r) => r.chat);
  }, [chats, search.isActive, search.results]);

  const searchResultsMap = useMemo(() => {
    const map = new Map<string, SearchResult>();
    if (search.isActive) {
      for (const result of search.results) {
        if (!map.has(result.chat.id)) map.set(result.chat.id, result);
      }
    }
    return map;
  }, [search.isActive, search.results]);

  const formatTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    if (diffDays === 0) return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    if (diffDays === 1) return "Yesterday";
    if (diffDays < 7) return date.toLocaleDateString([], { weekday: "short" });
    return date.toLocaleDateString([], { month: "short", day: "numeric" });
  };

  const getLabelById = (id: string) => labels.find((l) => l.id === id);

  const viewLabels: Record<View, { title: string; icon: typeof Inbox; emptyMessage: string }> = {
    inbox: { title: "Inbox", icon: Inbox, emptyMessage: "No chats yet. Start a new conversation to get going." },
    starred: { title: "Starred", icon: Star, emptyMessage: "No starred chats. Star important conversations to find them quickly." },
    archived: { title: "Archived", icon: Archive, emptyMessage: "No archived chats. Archive conversations you want to keep but hide from inbox." },
  };

  const { title, icon: ViewIcon, emptyMessage } = viewLabels[currentView];

  return (
    <div className="w-full lg:w-80 border-r border-white/10 flex flex-col bg-slate-950/50 shrink-0" data-testid="chat-list-panel">
      <div className="p-4 border-b border-white/10">
        <div className="flex items-center gap-2 mb-3">
          <ViewIcon className="h-4 w-4 text-slate-400" />
          <h2 className="text-sm font-semibold text-white">{search.isActive ? "Search Results" : title}</h2>
          <span className="text-xs text-slate-500">({search.isActive ? search.results.length : displayChats.length})</span>
          {search.isSearching && <Loader2 className="h-3 w-3 animate-spin text-slate-400" />}
        </div>
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <input ref={ref} type="text" value={search.query} onChange={(e) => search.setQuery(e.target.value)} placeholder="Search messages... (Ctrl+K)" className="w-full bg-white/5 border border-white/10 rounded-lg pl-9 pr-8 py-2 text-sm text-white placeholder:text-slate-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/50" data-testid="chat-search-input" />
          {search.query && (
            <button onClick={search.clear} className="absolute right-2 top-1/2 -translate-y-1/2 p-1 rounded hover:bg-white/10 text-slate-500 hover:text-white" data-testid="clear-search-button"><X className="h-3 w-3" /></button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-y-auto" data-testid="chat-list">
        {isLoading && !search.isActive ? (
          <div className="flex flex-col items-center justify-center py-12 text-slate-500"><Loader2 className="h-6 w-6 animate-spin mb-2" /><p className="text-sm">Loading chats...</p></div>
        ) : displayChats.length === 0 ? (
          <div className="p-6 text-center">
            {search.isActive ? (
              <>
                <Search className="h-10 w-10 mx-auto mb-3 text-slate-600" />
                <p className="text-sm text-slate-500">{search.isSearching ? "Searching..." : `No results for "${search.query}"`}</p>
                <button onClick={search.clear} className="mt-2 text-sm text-indigo-400 hover:text-indigo-300">Clear search</button>
              </>
            ) : (
              <>
                <ViewIcon className="h-10 w-10 mx-auto mb-3 text-slate-600" />
                <p className="text-sm text-slate-500 mb-4">{emptyMessage}</p>
                {currentView === "inbox" && <button onClick={onNewChat} className="text-sm text-indigo-400 hover:text-indigo-300 font-medium">Create your first chat</button>}
              </>
            )}
          </div>
        ) : (
          displayChats.map((chat, index) => {
            const searchResult = searchResultsMap.get(chat.id);
            return (
              <ChatListItem
                key={chat.id}
                ref={(el) => { if (el) itemRefs.current.set(index, el); else itemRefs.current.delete(index); }}
                chat={chat}
                labels={chat.label_ids.map(getLabelById).filter(Boolean) as Label[]}
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
