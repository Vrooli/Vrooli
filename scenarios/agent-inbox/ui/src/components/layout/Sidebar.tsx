import { Plus, Mail, Star, Archive, Tag, Settings, Loader2, MessageSquare, Keyboard } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import type { View } from "../../hooks/useChats";
import type { Label } from "../../lib/api";

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
}

const navItems: { id: View; label: string; icon: typeof Mail }[] = [
  { id: "inbox", label: "Inbox", icon: Mail },
  { id: "starred", label: "Starred", icon: Star },
  { id: "archived", label: "Archived", icon: Archive },
];

export function Sidebar({
  currentView,
  onViewChange,
  onNewChat,
  onManageLabels,
  onOpenSettings,
  onShowKeyboardShortcuts,
  isCreatingChat,
  labels,
  chatCounts,
}: SidebarProps) {
  return (
    <aside
      className="w-64 border-r border-white/10 flex flex-col bg-slate-950 shrink-0"
      data-testid="sidebar"
    >
      {/* Header */}
      <div className="p-4 border-b border-white/10">
        <h1 className="text-lg font-semibold flex items-center gap-2 text-white">
          <MessageSquare className="h-5 w-5 text-indigo-400" />
          Agent Inbox
        </h1>
        <p className="text-xs text-slate-500 mt-1">AI-powered conversations</p>
      </div>

      {/* New Chat Button */}
      <div className="p-3">
        <Button
          onClick={onNewChat}
          disabled={isCreatingChat}
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

      {/* Navigation */}
      <nav className="flex-1 px-3 py-2 space-y-1 overflow-y-auto" data-testid="sidebar-nav">
        {navItems.map(({ id, label, icon: Icon }) => {
          const count = chatCounts?.[id];
          const isActive = currentView === id;

          return (
            <button
              key={id}
              onClick={() => onViewChange(id)}
              className={`w-full flex items-center justify-between gap-2 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${
                isActive
                  ? "bg-white/10 text-white"
                  : "text-slate-400 hover:text-white hover:bg-white/5"
              }`}
              data-testid={`nav-${id}`}
            >
              <span className="flex items-center gap-2">
                <Icon className={`h-4 w-4 ${isActive && id === "starred" ? "text-yellow-400" : ""}`} />
                {label}
              </span>
              {count !== undefined && count > 0 && (
                <span
                  className={`text-xs px-1.5 py-0.5 rounded-full ${
                    isActive ? "bg-white/20" : "bg-white/10"
                  }`}
                >
                  {count}
                </span>
              )}
            </button>
          );
        })}

        {/* Labels Section */}
        {labels.length > 0 && (
          <div className="pt-4 mt-4 border-t border-white/10">
            <div className="flex items-center justify-between px-3 mb-2">
              <span className="text-xs font-medium text-slate-500 uppercase tracking-wide">
                Labels
              </span>
              <Tooltip content="Manage labels">
                <button
                  onClick={onManageLabels}
                  className="p-1 rounded hover:bg-white/10 text-slate-500 hover:text-white transition-colors"
                  data-testid="manage-labels-button"
                >
                  <Settings className="h-3.5 w-3.5" />
                </button>
              </Tooltip>
            </div>
            {labels.slice(0, 5).map((label) => (
              <button
                key={label.id}
                className="w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm text-slate-400 hover:text-white hover:bg-white/5 transition-colors"
                data-testid={`label-filter-${label.id}`}
              >
                <span
                  className="w-2.5 h-2.5 rounded-full"
                  style={{ backgroundColor: label.color }}
                />
                <span className="truncate">{label.name}</span>
              </button>
            ))}
            {labels.length > 5 && (
              <button
                onClick={onManageLabels}
                className="w-full px-3 py-2 text-xs text-slate-500 hover:text-white transition-colors text-left"
              >
                +{labels.length - 5} more labels
              </button>
            )}
          </div>
        )}

        {/* Add Labels Button when no labels exist */}
        {labels.length === 0 && (
          <button
            onClick={onManageLabels}
            className="w-full flex items-center gap-2 px-3 py-2.5 rounded-lg text-sm text-slate-500 hover:text-white hover:bg-white/5 transition-colors mt-4"
            data-testid="add-labels-button"
          >
            <Tag className="h-4 w-4" />
            Add labels
          </button>
        )}
      </nav>

      {/* Footer */}
      <div className="p-3 border-t border-white/10">
        <div className="flex items-center justify-center gap-1">
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
        <p className="text-xs text-slate-600 text-center mt-2">
          Powered by AI models
        </p>
      </div>
    </aside>
  );
}
