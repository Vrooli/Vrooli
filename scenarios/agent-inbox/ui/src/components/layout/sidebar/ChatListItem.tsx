import { useState, useEffect, useRef, forwardRef } from "react";
import {
  Star,
  MessageSquare,
  Check,
  FileText,
  CheckSquare,
  Square,
  X,
  Bot,
} from "lucide-react";
import { Badge } from "../../ui/badge";
import { SnippetHighlight } from "./SnippetHighlight";
import type { Chat, Label, SearchResult } from "./types";

interface ChatListItemProps {
  chat: Chat;
  labels: Label[];
  isSelected: boolean;
  isFocused?: boolean;
  onClick: () => void;
  onRename?: (newName: string) => void;
  formatTime: (date: string) => string;
  searchResult?: SearchResult;
  selectionMode?: boolean;
  isChecked?: boolean;
  onToggleSelect?: (e: React.MouseEvent) => void;
}

export const ChatListItem = forwardRef<HTMLDivElement, ChatListItemProps>(function ChatListItem(
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

  const handleClick = (e: React.MouseEvent) => {
    if (isEditing) return;

    if ((e.shiftKey || e.ctrlKey || e.metaKey) && onToggleSelect) {
      onToggleSelect(e);
      return;
    }

    if (selectionMode && onToggleSelect) {
      onToggleSelect(e);
      return;
    }

    onClick();
  };

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
          /* Chat Icon - Show agent icon for agent mode chats */
          <div className={`mt-0.5 p-1.5 rounded-lg shrink-0 ${
            chat.chat_mode === "agent"
              ? isSelected ? "bg-blue-500/30" : "bg-blue-500/10"
              : isSelected ? "bg-indigo-500/30" : "bg-white/10"
          }`}>
            {chat.chat_mode === "agent" ? (
              <Bot className="h-3.5 w-3.5 text-blue-400" />
            ) : (
              <MessageSquare className="h-3.5 w-3.5 text-slate-400" />
            )}
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
              <span className="text-[10px] text-slate-400">{formatTime(chat.updated_at)}</span>
            </div>
          </div>

          {/* Preview or Search Snippet */}
          {searchResult?.snippet ? (
            <div className="mt-1">
              <div className="flex items-center gap-1 text-[10px] text-indigo-400 mb-0.5">
                <FileText className="h-3 w-3" />
                <span>{searchResult.match_type === "message_content" ? "Message" : "Name"}</span>
              </div>
              <p className="text-xs text-slate-400 line-clamp-2 break-all" data-testid="search-snippet">
                <SnippetHighlight snippet={searchResult.snippet} matchStart={searchResult.match_start} matchEnd={searchResult.match_end} />
              </p>
            </div>
          ) : (
            <p className="text-xs text-slate-400 truncate mt-0.5">
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
                <span className="text-[9px] text-slate-400">+{labels.length - 2}</span>
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
