import React, { useState, useEffect, useMemo, useRef } from "react";
import {
  Loader2,
  Search,
  MessageSquare,
  FileText,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import { ChatListItem } from "./ChatListItem";
import { SnippetHighlight } from "./SnippetHighlight";
import { formatTime, getLabelById } from "./utils";
import { onProfilerRender } from "../../../lib/profiler";
import { useVirtualRows } from "../../../hooks/useVirtualRows";
import type { Chat, Label, SearchResult, ChatSearchMode } from "./types";

type ContentSearchGroup = { chat: Chat; matches: SearchResult[] };
type ContentSearchRow =
  | { type: "group"; group: ContentSearchGroup }
  | { type: "match"; group: ContentSearchGroup; match: SearchResult; matchIndex: number };

interface ChatListProps {
  chats: Chat[];
  displayChats: Chat[];
  isLoadingChats: boolean;
  selectedChatId: string | null;
  focusedIndex: number;
  onSelectChat: (chatId: string, messageId?: string) => void;
  onRenameChat?: (chatId: string, newName: string) => void;
  labels: Label[];
  emptyMessage: string;
  // Search
  searchIsActive: boolean;
  searchQuery: string;
  isSearching: boolean;
  searchResults: SearchResult[];
  searchMode: ChatSearchMode;
  onSearchModeChange: (mode: ChatSearchMode) => void;
  clearSearch: () => void;
  // Bulk selection
  selectionMode: boolean;
  selectedChatIds: Set<string>;
  toggleChatSelection: (chatId: string, index: number, event: React.MouseEvent) => void;
  // Test IDs
  listTestId: string;
  switchToContentSearchTestId: string;
}

export function ChatList({
  displayChats,
  isLoadingChats,
  selectedChatId,
  focusedIndex,
  onSelectChat,
  onRenameChat,
  labels,
  emptyMessage,
  searchIsActive,
  searchQuery,
  isSearching,
  searchResults,
  searchMode,
  onSearchModeChange,
  clearSearch,
  selectionMode,
  selectedChatIds,
  toggleChatSelection,
  listTestId,
  switchToContentSearchTestId,
}: ChatListProps) {
  // Refs for each chat item to enable scroll-into-view on focus
  const itemRefs = useRef<Map<number, HTMLDivElement>>(new Map());
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Build a map of search results by chat ID for snippet display (content mode only)
  const searchResultsMap = useMemo(() => {
    const map = new Map<string, SearchResult>();
    if (searchIsActive && searchMode === "content") {
      for (const result of searchResults) {
        if (!map.has(result.chat.id)) {
          map.set(result.chat.id, result);
        }
      }
    }
    return map;
  }, [searchIsActive, searchResults, searchMode]);

  // Group search results by chat for content mode
  const groupedSearchResults = useMemo(() => {
    if (!searchIsActive || searchMode !== "content") return [];
    const groupMap = new Map<string, { chat: Chat; matches: SearchResult[] }>();
    for (const result of searchResults) {
      const existing = groupMap.get(result.chat.id);
      if (existing) existing.matches.push(result);
      else groupMap.set(result.chat.id, { chat: result.chat, matches: [result] });
    }
    return Array.from(groupMap.values());
  }, [searchIsActive, searchResults, searchMode]);

  const showContentSearch = searchMode === "content" && searchIsActive && groupedSearchResults.length > 0;
  const shouldVirtualizeRows = !showContentSearch && displayChats.length > 40;
  const virtualRows = useVirtualRows({
    count: displayChats.length,
    estimateSize: 74,
    overscan: 8,
    enabled: shouldVirtualizeRows,
    containerRef: scrollContainerRef,
  });

  // Scroll focused item into view when focusedIndex changes
  useEffect(() => {
    if (focusedIndex >= 0) {
      const element = itemRefs.current.get(focusedIndex);
      if (element && typeof element.scrollIntoView === "function") {
        element.scrollIntoView({ block: "nearest", behavior: "smooth" });
      } else if (shouldVirtualizeRows) {
        virtualRows.scrollToIndex(focusedIndex, "center");
      }
    }
  }, [focusedIndex, shouldVirtualizeRows, virtualRows]);

  // Track which chat groups are expanded in content search
  const [expandedSearchGroups, setExpandedSearchGroups] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (searchMode === "content" && groupedSearchResults.length > 0) {
      setExpandedSearchGroups(new Set(groupedSearchResults.map((g) => g.chat.id)));
    }
  }, [groupedSearchResults, searchMode]);

  const contentSearchRows = useMemo<ContentSearchRow[]>(() => {
    if (!showContentSearch) return [];
    const rows: ContentSearchRow[] = [];
    for (const group of groupedSearchResults) {
      rows.push({ type: "group", group });
      if (!expandedSearchGroups.has(group.chat.id)) continue;
      group.matches.forEach((match, matchIndex) => {
        rows.push({ type: "match", group, match, matchIndex });
      });
    }
    return rows;
  }, [expandedSearchGroups, groupedSearchResults, showContentSearch]);

  const shouldVirtualizeContentRows = showContentSearch && contentSearchRows.length > 40;
  const contentVirtualRows = useVirtualRows({
    count: contentSearchRows.length,
    estimateSize: 52,
    overscan: 10,
    enabled: shouldVirtualizeContentRows,
    containerRef: scrollContainerRef,
  });

  const labelsByChatId = useMemo(() => {
    const map = new Map<string, Label[]>();
    for (const chat of displayChats) {
      map.set(chat.id, chat.label_ids.map((id) => getLabelById(labels, id)).filter(Boolean) as Label[]);
    }
    return map;
  }, [displayChats, labels]);

  const renderChatRow = (chat: Chat, index: number, measure?: (element: HTMLDivElement | null) => void) => {
    const searchResult = searchResultsMap.get(chat.id);
    return (
      <ChatListItem
        key={chat.id}
        ref={(el) => {
          if (el) itemRefs.current.set(index, el);
          else itemRefs.current.delete(index);
          measure?.(el);
        }}
        chat={chat}
        labels={labelsByChatId.get(chat.id) ?? []}
        isSelected={selectedChatId === chat.id}
        isFocused={focusedIndex === index}
        onClick={() => onSelectChat(chat.id, searchResult?.message_id)}
        onRename={onRenameChat ? (newName) => onRenameChat(chat.id, newName) : undefined}
        formatTime={formatTime}
        searchResult={searchResult}
        selectionMode={selectionMode}
        isChecked={selectedChatIds.has(chat.id)}
        onToggleSelect={(e) => toggleChatSelection(chat.id, index, e)}
      />
    );
  };

  const toggleSearchGroup = (chatId: string) => {
    setExpandedSearchGroups((prev) => {
      const next = new Set(prev);
      if (next.has(chatId)) next.delete(chatId);
      else next.add(chatId);
      return next;
    });
  };

  const renderContentSearchRow = (row: ContentSearchRow) => {
    if (row.type === "group") {
      const { group } = row;
      const isExpanded = expandedSearchGroups.has(group.chat.id);
      return (
        <div className="border-b border-white/5">
          <button
            type="button"
            className="w-full flex items-center gap-2 px-3 py-2 hover:bg-white/5 transition-colors"
            onClick={() => toggleSearchGroup(group.chat.id)}
          >
            {isExpanded ? (
              <ChevronDown className="h-3 w-3 text-slate-400 shrink-0" />
            ) : (
              <ChevronRight className="h-3 w-3 text-slate-400 shrink-0" />
            )}
            <MessageSquare className="h-3.5 w-3.5 text-slate-400 shrink-0" />
            <span className="text-sm text-slate-200 truncate flex-1 text-left">{group.chat.name}</span>
            <span className="text-[10px] text-slate-500 bg-white/5 rounded-full px-1.5 py-0.5 shrink-0">
              {group.matches.length}
            </span>
          </button>
        </div>
      );
    }

    const { group, match } = row;
    return (
      <button
        type="button"
        className="w-full flex items-start gap-2 pl-8 pr-3 py-2 hover:bg-white/5 transition-colors text-left border-b border-white/5"
        onClick={() => onSelectChat(group.chat.id, match.message_id)}
      >
        <FileText className="h-3 w-3 text-slate-500 mt-0.5 shrink-0" />
        <div className="min-w-0 flex-1">
          <span className="text-[10px] text-slate-500 uppercase tracking-wide">
            {match.match_type === "chat_name" ? "Name" : "Message"}
          </span>
          {match.snippet && (
            <p className="text-xs text-slate-400 line-clamp-2 break-all">
              <SnippetHighlight snippet={match.snippet} matchStart={match.match_start} matchEnd={match.match_end} />
            </p>
          )}
        </div>
      </button>
    );
  };

  return (
    <React.Profiler id="SidebarChatList" onRender={onProfilerRender}>
    <div ref={scrollContainerRef} className="flex-1 overflow-y-auto" data-testid={listTestId}>
      {isLoadingChats && !searchIsActive ? (
        <div className="flex flex-col items-center justify-center py-12 text-slate-500">
          <Loader2 className="h-6 w-6 animate-spin mb-2" />
          <p className="text-sm">Loading chats...</p>
        </div>
      ) : displayChats.length === 0 ? (
        <div className="p-6 text-center">
          {searchIsActive ? (
            <>
              <Search className="h-10 w-10 mx-auto mb-3 text-slate-600" />
              <p className="text-sm text-slate-500">
                {isSearching
                  ? "Searching..."
                  : `No results for "${searchQuery}"`}
              </p>
              {searchMode === "quick" && !isSearching && (
                <button
                  onClick={() => onSearchModeChange("content")}
                  className="mt-2 text-sm text-indigo-400 hover:text-indigo-300"
                  data-testid={switchToContentSearchTestId}
                >
                  Search message content instead
                </button>
              )}
              <button
                onClick={clearSearch}
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
      ) : showContentSearch ? (
        shouldVirtualizeContentRows ? (
          <div className="relative" style={{ height: contentVirtualRows.totalHeight }}>
            {contentVirtualRows.virtualRows.map((virtualRow) => {
              const row = contentSearchRows[virtualRow.index];
              if (!row) return null;
              const key = row.type === "group"
                ? `group-${row.group.chat.id}`
                : `match-${row.match.chat.id}-${row.match.message_id}-${row.matchIndex}`;
              return (
                <div
                  key={key}
                  ref={(element) => contentVirtualRows.measureElement(virtualRow.index, element)}
                  className="absolute left-0 right-0 top-0"
                  style={{ transform: `translateY(${virtualRow.offsetTop}px)` }}
                >
                  {renderContentSearchRow(row)}
                </div>
              );
            })}
          </div>
        ) : (
          contentSearchRows.map((row) => {
            const key = row.type === "group"
              ? `group-${row.group.chat.id}`
              : `match-${row.match.chat.id}-${row.match.message_id}-${row.matchIndex}`;
            return <React.Fragment key={key}>{renderContentSearchRow(row)}</React.Fragment>;
          })
        )
      ) : shouldVirtualizeRows ? (
        <div className="relative" style={{ height: virtualRows.totalHeight }}>
          {virtualRows.virtualRows.map((row) => {
            const chat = displayChats[row.index];
            if (!chat) return null;
            return (
              <div
                key={chat.id}
                ref={(element) => virtualRows.measureElement(row.index, element)}
                className="absolute left-0 right-0 top-0"
                style={{ transform: `translateY(${row.offsetTop}px)` }}
              >
                {renderChatRow(chat, row.index)}
              </div>
            );
          })}
        </div>
      ) : (
        displayChats.map((chat, index) => renderChatRow(chat, index))
      )}
    </div>
    </React.Profiler>
  );
}
