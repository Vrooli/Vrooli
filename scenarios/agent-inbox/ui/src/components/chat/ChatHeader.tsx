import { useState } from "react";
import { Mail, MailOpen, Star, Archive, Trash2, Edit3, MoreVertical, Download, ChevronLeft, Menu } from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { Dropdown, DropdownItem, DropdownSeparator } from "../ui/dropdown";
import { ChatStatusIcon } from "./ChatStatusIcon";
import { ModeSelector, type ChatMode } from "./ModeSelector";
import { RenameDialog, DeleteDialog, ExportDialog } from "./ChatHeaderDialogs";
import { selectorsManifest } from "../../consts/selectors";
import type { Chat, Model, Label, AgentModeStatus } from "../../lib/api";
import type { AgentMetric } from "./agent/AgentEventList";

interface ChatHeaderProps {
  chat: Chat;
  models: Model[];
  labels: Label[];
  chatMode?: "llm" | "agent";
  onUpdateChat: (data: { name?: string; model?: string }) => void;
  onToggleRead: () => void;
  onToggleStar: () => void;
  onToggleArchive: () => void;
  onDelete: () => void;
  onAssignLabel: (labelId: string) => void;
  onRemoveLabel: (labelId: string) => void;
  isAgentActive?: boolean;
  agentStatus?: AgentModeStatus | null;
  agentMetrics?: AgentMetric[];
  agentError?: { message: string; recovery?: string } | null;
  onStopAgent?: () => void;
  onBackToList?: () => void;
  isMobile?: boolean;
  onOpenSidebar?: () => void;
  hasMessages?: boolean;
  onModeChange?: (mode: ChatMode) => void;
  onOpenAgentSettings?: () => void;
}

export function ChatHeader({
  chat, models, labels, chatMode, onUpdateChat, onToggleRead, onToggleStar, onToggleArchive,
  onDelete, onAssignLabel, onRemoveLabel, isAgentActive = false, agentStatus, agentMetrics,
  agentError, onStopAgent, onBackToList, isMobile, onOpenSidebar, hasMessages = true,
  onModeChange, onOpenAgentSettings,
}: ChatHeaderProps) {
  const chatHeaderTestIds = {
    container: selectorsManifest.selectors["chatView.header"]?.testId ?? "chat-header",
    renameChatButton: selectorsManifest.selectors["chatHeader.renameChatButton"]?.testId ?? "rename-chat-button",
    toggleReadButton: selectorsManifest.selectors["chatHeader.toggleReadButton"]?.testId ?? "toggle-read-button",
    toggleStarButton: selectorsManifest.selectors["chatHeader.toggleStarButton"]?.testId ?? "toggle-star-button",
    toggleArchiveButton: selectorsManifest.selectors["chatHeader.toggleArchiveButton"]?.testId ?? "toggle-archive-button",
    moreActionsButton: selectorsManifest.selectors["chatHeader.moreActionsButton"]?.testId ?? "chat-more-actions",
    confirmDeleteButton: selectorsManifest.selectors["chatHeader.confirmDeleteButton"]?.testId ?? "confirm-delete-button",
  };

  const [showRenameDialog, setShowRenameDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showExportDialog, setShowExportDialog] = useState(false);

  const chatLabelIds = chat.label_ids || [];
  const assignedLabels = labels.filter((l) => chatLabelIds.includes(l.id));
  const availableLabels = labels.filter((l) => !chatLabelIds.includes(l.id));

  return (
    <>
      <header className="pl-2 lg:pl-4 pr-3 lg:pr-4 h-14 border-b border-white/10 bg-slate-950/50 flex items-center" data-testid={chatHeaderTestIds.container}>
        {isMobile && (
          <Button variant="ghost" size="icon" onClick={onBackToList ?? onOpenSidebar} className="h-9 w-9 shrink-0 mr-1 lg:hidden">
            {onBackToList ? <ChevronLeft className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </Button>
        )}
        <div className="min-w-0 flex-1 flex items-center gap-2">
          <h2 className="text-sm sm:text-base font-semibold text-white truncate">{chat.name}</h2>
          {assignedLabels.length > 0 && (
            <div className="hidden sm:flex items-center gap-1 overflow-hidden">
              {assignedLabels.slice(0, 2).map((label) => (
                <span key={label.id} className="w-2 h-2 rounded-full shrink-0" style={{ backgroundColor: label.color }} title={label.name} />
              ))}
              {assignedLabels.length > 2 && <span className="text-xs text-slate-500">+{assignedLabels.length - 2}</span>}
            </div>
          )}
        </div>

        <div className="flex items-center gap-1 shrink-0">
          {!hasMessages && onModeChange ? (
            <ModeSelector mode={chatMode ?? "llm"} onModeChange={onModeChange} isAgentActive={isAgentActive} onOpenAgentSettings={onOpenAgentSettings} />
          ) : (
            <ChatStatusIcon chatMode={chatMode ?? "llm"} model={chat.model} models={models} isAgentActive={isAgentActive} agentStatus={agentStatus} agentMetrics={agentMetrics} agentError={agentError} onStopAgent={onStopAgent} />
          )}

          <Dropdown
            trigger={
              <Tooltip content="More actions">
                <Button variant="ghost" size="icon" data-testid={chatHeaderTestIds.moreActionsButton} aria-label="Open more chat actions">
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </Tooltip>
            }
            align="right"
          >
            <div className="px-3 py-2 border-b border-white/10">
              <p className="text-xs text-slate-500 mb-2">Labels</p>
              {assignedLabels.length > 0 && (
                <div className="flex flex-wrap gap-1 mb-2">
                  {assignedLabels.map((label) => (
                    <button key={label.id} onClick={() => onRemoveLabel(label.id)} className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs hover:bg-white/10 transition-colors" title={`Remove ${label.name}`}>
                      <span className="w-2 h-2 rounded-full" style={{ backgroundColor: label.color }} />
                      <span className="text-slate-300">{label.name}</span>
                      <span className="text-slate-500 ml-0.5">x</span>
                    </button>
                  ))}
                </div>
              )}
              {availableLabels.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {availableLabels.map((label) => (
                    <button key={label.id} onClick={() => onAssignLabel(label.id)} className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs text-slate-400 hover:text-white hover:bg-white/10 transition-colors" title={`Add ${label.name}`}>
                      <span className="w-2 h-2 rounded-full opacity-50" style={{ backgroundColor: label.color }} /><span>{label.name}</span>
                    </button>
                  ))}
                </div>
              ) : labels.length === 0 ? <p className="text-xs text-slate-500">No labels yet</p> : <p className="text-xs text-slate-500">All labels assigned</p>}
            </div>

            <DropdownItem onClick={onToggleRead} testId={chatHeaderTestIds.toggleReadButton}>
              {chat.is_read ? <Mail className="h-4 w-4" /> : <MailOpen className="h-4 w-4" />}
              {chat.is_read ? "Mark as unread" : "Mark as read"}
            </DropdownItem>
            <DropdownItem onClick={onToggleStar} testId={chatHeaderTestIds.toggleStarButton}>
              <Star className={`h-4 w-4 ${chat.is_starred ? "text-yellow-500 fill-yellow-500" : ""}`} />
              {chat.is_starred ? "Remove star" : "Star chat"}
            </DropdownItem>
            <DropdownItem onClick={onToggleArchive} testId={chatHeaderTestIds.toggleArchiveButton}>
              <Archive className="h-4 w-4" />{chat.is_archived ? "Unarchive" : "Archive"}
            </DropdownItem>
            <DropdownSeparator />
            <DropdownItem onClick={() => setShowRenameDialog(true)} testId={chatHeaderTestIds.renameChatButton}>
              <Edit3 className="h-4 w-4" />Rename chat
            </DropdownItem>
            <DropdownSeparator />
            <DropdownItem onClick={() => setShowExportDialog(true)} testId="export-chat-button">
              <Download className="h-4 w-4" />Export chat
            </DropdownItem>
            <DropdownSeparator />
            <DropdownItem destructive onClick={() => setShowDeleteDialog(true)}>
              <Trash2 className="h-4 w-4" />Delete chat
            </DropdownItem>
          </Dropdown>
        </div>
      </header>

      <RenameDialog open={showRenameDialog} onClose={() => setShowRenameDialog(false)} chatName={chat.name ?? ""} onRename={(name) => onUpdateChat({ name })} />
      <DeleteDialog open={showDeleteDialog} onClose={() => setShowDeleteDialog(false)} chatName={chat.name} onDelete={onDelete} confirmTestId={chatHeaderTestIds.confirmDeleteButton} />
      <ExportDialog open={showExportDialog} onClose={() => setShowExportDialog(false)} chatId={chat.id} chatName={chat.name} />
    </>
  );
}
