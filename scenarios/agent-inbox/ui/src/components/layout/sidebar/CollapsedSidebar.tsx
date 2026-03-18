import {
  Plus,
  Settings,
  Loader2,
  Keyboard,
  PanelLeft,
} from "lucide-react";
import { Tooltip } from "../../ui/tooltip";
import { navItems } from "./types";
import type { SidebarProps } from "./types";

type CollapsedSidebarProps = Pick<
  SidebarProps,
  | "onToggleCollapsed"
  | "onNewChat"
  | "isCreatingChat"
  | "currentView"
  | "onViewChange"
  | "chatCounts"
  | "onShowKeyboardShortcuts"
  | "onOpenSettings"
>;

export function CollapsedSidebar({
  onToggleCollapsed,
  onNewChat,
  isCreatingChat,
  currentView,
  onViewChange,
  chatCounts,
  onShowKeyboardShortcuts,
  onOpenSettings,
}: CollapsedSidebarProps) {
  return (
    <aside
      className="hidden lg:flex w-16 border-r border-white/10 flex-col bg-slate-950 shrink-0 h-full"
      data-testid="sidebar-collapsed"
    >
      {/* Expand button */}
      <div className="p-2 border-b border-white/10 shrink-0">
        <Tooltip content="Expand sidebar" side="right">
          <button
            onClick={onToggleCollapsed}
            className="w-full p-2.5 rounded-lg text-slate-400 hover:text-white hover:bg-white/10 transition-colors flex items-center justify-center"
            data-testid="expand-sidebar-button"
          >
            <PanelLeft className="h-5 w-5" />
          </button>
        </Tooltip>
      </div>

      {/* New Chat button */}
      <div className="p-2 border-b border-white/10 shrink-0">
        <Tooltip content="New Chat" side="right">
          <button
            onClick={onNewChat}
            disabled={isCreatingChat}
            className="w-full p-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 transition-colors flex items-center justify-center"
            data-testid="new-chat-button-collapsed"
          >
            {isCreatingChat ? (
              <Loader2 className="h-5 w-5 animate-spin text-white" />
            ) : (
              <Plus className="h-5 w-5 text-white" />
            )}
          </button>
        </Tooltip>
      </div>

      {/* Navigation Icons */}
      <div className="p-2 border-b border-white/10 shrink-0 flex flex-col gap-1">
        {navItems.map(({ id, label, icon: Icon }) => {
          const count = chatCounts?.[id];
          const isActive = currentView === id;

          return (
            <Tooltip key={id} content={`${label}${count ? ` (${count})` : ""}`} side="right">
              <button
                onClick={() => onViewChange(id)}
                className={`relative w-full p-2.5 rounded-lg transition-colors flex items-center justify-center ${
                  isActive
                    ? "bg-white/10 text-white"
                    : "text-slate-400 hover:text-white hover:bg-white/5"
                }`}
                data-testid={`nav-${id}-collapsed`}
              >
                <Icon className={`h-5 w-5 ${isActive && id === "starred" ? "text-yellow-400" : ""}`} />
                {count !== undefined && count > 0 && (
                  <span className="absolute -top-0.5 -right-0.5 min-w-[16px] h-4 px-1 text-[10px] font-medium bg-indigo-500 text-white rounded-full flex items-center justify-center">
                    {count > 99 ? "99+" : count}
                  </span>
                )}
              </button>
            </Tooltip>
          );
        })}
      </div>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Footer Icons */}
      <div className="p-2 border-t border-white/10 shrink-0 flex flex-col gap-1">
        <Tooltip content="Keyboard shortcuts" side="right">
          <button
            onClick={onShowKeyboardShortcuts}
            className="w-full p-2.5 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors flex items-center justify-center"
            data-testid="sidebar-shortcuts-button-collapsed"
          >
            <Keyboard className="h-5 w-5" />
          </button>
        </Tooltip>
        <Tooltip content="Settings" side="right">
          <button
            onClick={onOpenSettings}
            className="w-full p-2.5 rounded-lg text-slate-500 hover:text-white hover:bg-white/10 transition-colors flex items-center justify-center"
            data-testid="sidebar-settings-button-collapsed"
          >
            <Settings className="h-5 w-5" />
          </button>
        </Tooltip>
      </div>
    </aside>
  );
}
