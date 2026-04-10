import { useState, useEffect, useRef } from "react";
import {
  Plus,
  Settings,
  Loader2,
  MessageSquare,
  Keyboard,
  CheckSquare,
  PanelLeftClose,
  Bot,
  Tag,
  MoreVertical,
  ChevronDown,
} from "lucide-react";
import { Button } from "../../ui/button";
import { Tooltip } from "../../ui/tooltip";
import { Dropdown, DropdownItem, DropdownSeparator } from "../../ui/dropdown";
import type { Chat } from "./types";

interface SidebarHeaderProps {
  onDeselectChat?: () => void;
  onToggleCollapsed?: () => void;
  /** Only used for truthiness check to show/hide bulk selection UI */
  onBulkOperate?: boolean;
  displayChats: Chat[];
  searchIsActive: boolean;
  selectionMode: boolean;
  exitSelectionMode: () => void;
  setSelectionMode: (value: boolean) => void;
  onNewChat: () => void;
  onNewAgentChat?: () => void;
  isCreatingChat: boolean;
  onManageLabels: () => void;
  onShowKeyboardShortcuts: () => void;
  onOpenSettings: () => void;
  sidebarTestIds: {
    newChatButton: string;
    mobileActionsButton: string;
    manageLabelsButton: string;
  };
}

export function SidebarHeader({
  onDeselectChat,
  onToggleCollapsed,
  onBulkOperate,
  displayChats,
  searchIsActive,
  selectionMode,
  exitSelectionMode,
  setSelectionMode,
  onNewChat,
  onNewAgentChat,
  isCreatingChat,
  onManageLabels,
  onShowKeyboardShortcuts,
  onOpenSettings,
  sidebarTestIds,
}: SidebarHeaderProps) {
  const [showNewChatMenu, setShowNewChatMenu] = useState(false);
  const newChatMenuRef = useRef<HTMLDivElement>(null);

  // Close new chat menu when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (newChatMenuRef.current && !newChatMenuRef.current.contains(event.target as Node)) {
        setShowNewChatMenu(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  return (
    <div className="p-3 border-b border-white/10 shrink-0">
      <div className="flex items-center gap-2 mb-3">
        <button
          onClick={onDeselectChat}
          className="flex items-center gap-2 hover:opacity-80 transition-opacity"
          title="Go to home"
          data-testid="sidebar-home-button"
        >
          <MessageSquare className="h-5 w-5 text-indigo-400" />
          <h1 className="text-base font-semibold text-white">Agent Inbox</h1>
        </button>
        <div className="flex-1" />
        {/* Collapse button - desktop only */}
        {onToggleCollapsed && (
          <Tooltip content="Collapse sidebar">
            <button
              onClick={onToggleCollapsed}
              className="hidden lg:flex p-1.5 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
              data-testid="collapse-sidebar-button"
              aria-label="Collapse sidebar"
            >
              <PanelLeftClose className="h-4 w-4" />
            </button>
          </Tooltip>
        )}
        {onBulkOperate && displayChats.length > 0 && !searchIsActive && (
          <Tooltip content={selectionMode ? "Cancel selection" : "Select multiple"}>
            <button
              onClick={() => {
                if (selectionMode) {
                  exitSelectionMode();
                } else {
                  setSelectionMode(true);
                }
              }}
              className={`hidden lg:flex p-1.5 rounded-lg transition-colors ${
                selectionMode
                  ? "bg-indigo-500/20 text-indigo-400"
                  : "text-slate-500 hover:text-white hover:bg-white/10"
              }`}
              data-testid="toggle-selection-mode"
              aria-label="Toggle selection mode"
            >
              <CheckSquare className="h-4 w-4" />
            </button>
          </Tooltip>
        )}
        <div className="lg:hidden">
          <Dropdown
            align="right"
            trigger={
              <button
                className="p-1.5 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors"
                aria-label="Open sidebar actions"
                data-testid={sidebarTestIds.mobileActionsButton}
              >
                <MoreVertical className="h-4 w-4" />
              </button>
            }
          >
            {onBulkOperate && displayChats.length > 0 && !searchIsActive && (
              <>
                <DropdownItem
                  onClick={() => {
                    if (selectionMode) {
                      exitSelectionMode();
                    } else {
                      setSelectionMode(true);
                    }
                  }}
                >
                  <CheckSquare className="h-4 w-4" />
                  {selectionMode ? "Cancel selection mode" : "Select multiple chats"}
                </DropdownItem>
                <DropdownSeparator />
              </>
            )}
            <DropdownItem onClick={onManageLabels} testId={sidebarTestIds.manageLabelsButton}>
              <Tag className="h-4 w-4" />
              Manage labels
            </DropdownItem>
            <DropdownItem onClick={onShowKeyboardShortcuts}>
              <Keyboard className="h-4 w-4" />
              Keyboard shortcuts
            </DropdownItem>
            <DropdownItem onClick={onOpenSettings}>
              <Settings className="h-4 w-4" />
              Settings
            </DropdownItem>
          </Dropdown>
        </div>
      </div>
      {/* New Chat button with optional agent mode dropdown */}
      <div className="relative" ref={newChatMenuRef}>
        <div className="flex gap-1">
          <Button
            onClick={onNewChat}
            disabled={isCreatingChat || selectionMode}
            className="flex-1 justify-center gap-2"
            data-testid={sidebarTestIds.newChatButton}
          >
            {isCreatingChat ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <Plus className="h-4 w-4" />
            )}
            New Chat
          </Button>
          {onNewAgentChat && (
            <Button
              onClick={() => setShowNewChatMenu(!showNewChatMenu)}
              disabled={isCreatingChat || selectionMode}
              className="px-2"
              data-testid="new-chat-dropdown"
              aria-label="Choose chat type"
            >
              <ChevronDown className="h-4 w-4" />
            </Button>
          )}
        </div>
        {showNewChatMenu && onNewAgentChat && (
          <div className="absolute top-full left-0 right-0 mt-1 bg-slate-900 border border-white/10 rounded-lg shadow-lg z-50 overflow-hidden">
            <button
              onClick={() => {
                onNewChat();
                setShowNewChatMenu(false);
              }}
              className="w-full flex items-center gap-3 px-3 py-2.5 text-left text-sm text-slate-300 hover:bg-white/10 transition-colors"
              data-testid="new-llm-chat-option"
            >
              <MessageSquare className="h-4 w-4 text-indigo-400" />
              <div>
                <div className="font-medium text-white">LLM Chat</div>
                <div className="text-xs text-slate-500">Chat with AI assistant</div>
              </div>
            </button>
            <button
              onClick={() => {
                onNewAgentChat();
                setShowNewChatMenu(false);
              }}
              className="w-full flex items-center gap-3 px-3 py-2.5 text-left text-sm text-slate-300 hover:bg-white/10 transition-colors border-t border-white/5"
              data-testid="new-agent-chat-option"
            >
              <Bot className="h-4 w-4 text-blue-400" />
              <div>
                <div className="font-medium text-white">Agent Chat</div>
                <div className="text-xs text-slate-500">Agentic coding with tools</div>
              </div>
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
