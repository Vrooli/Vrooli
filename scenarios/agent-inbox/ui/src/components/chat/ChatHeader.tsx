import { useState } from "react";
import {
  Mail,
  MailOpen,
  Star,
  Archive,
  Trash2,
  Edit3,
  MoreVertical,
  Download,
  FileText,
  FileJson,
  File,
  ChevronLeft,
  Menu,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { Dropdown, DropdownItem, DropdownSeparator } from "../ui/dropdown";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "../ui/dialog";
import { Input } from "../ui/input";
import { ChatStatusIcon } from "./ChatStatusIcon";
import { ModeSelector, type ChatMode } from "./ModeSelector";
import { selectorsManifest } from "../../consts/selectors";
import { exportChat } from "../../lib/api";
import type { Chat, Model, Label, ExportFormat, AgentModeStatus } from "../../lib/api";
import type { AgentMetric } from "./agent/AgentEventList";

interface ChatHeaderProps {
  chat: Chat;
  models: Model[];
  labels: Label[];
  /** Current chat mode — model selector is hidden in agent mode. */
  chatMode?: "llm" | "agent";
  onUpdateChat: (data: { name?: string; model?: string }) => void;
  onToggleRead: () => void;
  onToggleStar: () => void;
  onToggleArchive: () => void;
  onDelete: () => void;
  onAssignLabel: (labelId: string) => void;
  onRemoveLabel: (labelId: string) => void;
  /** Whether agent is currently running */
  isAgentActive?: boolean;
  /** Live agent status from WebSocket */
  agentStatus?: AgentModeStatus | null;
  /** Aggregated agent metrics */
  agentMetrics?: AgentMetric[];
  /** Agent API error (start/stop failures) */
  agentError?: { message: string; recovery?: string } | null;
  /** Stop the running agent */
  onStopAgent?: () => void;
  /** Mobile: go back to chat list */
  onBackToList?: () => void;
  /** Whether the viewport is mobile-sized */
  isMobile?: boolean;
  /** Mobile: open the sidebar */
  onOpenSidebar?: () => void;
  /** Whether the chat has any messages yet */
  hasMessages?: boolean;
  /** Callback when chat mode changes (only used when hasMessages is false) */
  onModeChange?: (mode: ChatMode) => void;
  /** Open settings focused on agent tab */
  onOpenAgentSettings?: () => void;
}

export function ChatHeader({
  chat,
  models,
  labels,
  chatMode,
  onUpdateChat,
  onToggleRead,
  onToggleStar,
  onToggleArchive,
  onDelete,
  onAssignLabel,
  onRemoveLabel,
  isAgentActive = false,
  agentStatus,
  agentMetrics,
  agentError,
  onStopAgent,
  onBackToList,
  isMobile,
  onOpenSidebar,
  hasMessages = true,
  onModeChange,
  onOpenAgentSettings,
}: ChatHeaderProps) {
  const chatHeaderTestIds = {
    container: selectorsManifest.selectors["chatView.header"]?.testId ?? "chat-header",
    renameChatButton: selectorsManifest.selectors["chatHeader.renameChatButton"]?.testId ?? "rename-chat-button",
    toggleReadButton: selectorsManifest.selectors["chatHeader.toggleReadButton"]?.testId ?? "toggle-read-button",
    toggleStarButton: selectorsManifest.selectors["chatHeader.toggleStarButton"]?.testId ?? "toggle-star-button",
    toggleArchiveButton: selectorsManifest.selectors["chatHeader.toggleArchiveButton"]?.testId ?? "toggle-archive-button",
    moreActionsButton: selectorsManifest.selectors["chatHeader.moreActionsButton"]?.testId ?? "chat-more-actions",
    mobileActionsButton: selectorsManifest.selectors["chatHeader.mobileActionsButton"]?.testId ?? "chat-mobile-actions",
    confirmDeleteButton: selectorsManifest.selectors["chatHeader.confirmDeleteButton"]?.testId ?? "confirm-delete-button",
  };
  const [showRenameDialog, setShowRenameDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showExportDialog, setShowExportDialog] = useState(false);
  const [_exportFormat, _setExportFormat] = useState<ExportFormat>("markdown");
  const [isExporting, setIsExporting] = useState(false);
  const [newName, setNewName] = useState(chat.name ?? "");

  const handleExport = async (format: ExportFormat) => {
    try {
      setIsExporting(true);
      await exportChat(chat.id, format);
      setShowExportDialog(false);
    } catch (error) {
      console.error("Export failed:", error);
    } finally {
      setIsExporting(false);
    }
  };
  const chatLabelIds = chat.label_ids || [];
  const assignedLabels = labels.filter((l) => chatLabelIds.includes(l.id));
  const availableLabels = labels.filter((l) => !chatLabelIds.includes(l.id));

  const handleRename = () => {
    if (newName.trim() && newName !== chat.name) {
      onUpdateChat({ name: newName.trim() });
    }
    setShowRenameDialog(false);
  };

  const handleDelete = () => {
    onDelete();
    setShowDeleteDialog(false);
  };

  return (
    <>
      <header className="pl-2 lg:pl-4 pr-3 lg:pr-4 h-14 border-b border-white/10 bg-slate-950/50 flex items-center" data-testid={chatHeaderTestIds.container}>
        {/* Mobile toggle button - inline in header */}
        {isMobile && (
          <Button
            variant="ghost"
            size="icon"
            onClick={onBackToList ?? onOpenSidebar}
            className="h-9 w-9 shrink-0 mr-1 lg:hidden"
          >
            {onBackToList ? <ChevronLeft className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
          </Button>
        )}
        {/* Left Section - Title */}
        <div className="min-w-0 flex-1 flex items-center gap-2">
          <h2 className="text-sm sm:text-base font-semibold text-white truncate">{chat.name}</h2>
          {/* Assigned labels - inline badges */}
          {assignedLabels.length > 0 && (
            <div className="hidden sm:flex items-center gap-1 overflow-hidden">
              {assignedLabels.slice(0, 2).map((label) => (
                <span
                  key={label.id}
                  className="w-2 h-2 rounded-full shrink-0"
                  style={{ backgroundColor: label.color }}
                  title={label.name}
                />
              ))}
              {assignedLabels.length > 2 && (
                <span className="text-xs text-slate-500">+{assignedLabels.length - 2}</span>
              )}
            </div>
          )}
        </div>

        {/* Right Section - Actions */}
        <div className="flex items-center gap-1 shrink-0">
          {/* Mode selector (empty chat) or status icon (chat with messages) */}
          {!hasMessages && onModeChange ? (
            <ModeSelector
              mode={chatMode ?? "llm"}
              onModeChange={onModeChange}
              isAgentActive={isAgentActive}
              onOpenAgentSettings={onOpenAgentSettings}
            />
          ) : (
            <ChatStatusIcon
              chatMode={chatMode ?? "llm"}
              model={chat.model}
              models={models}
              isAgentActive={isAgentActive}
              agentStatus={agentStatus}
              agentMetrics={agentMetrics}
              agentError={agentError}
              onStopAgent={onStopAgent}
            />
          )}

          {/* Ellipsis menu - all actions consolidated here */}
          <Dropdown
            trigger={
              <Tooltip content="More actions">
                <Button
                  variant="ghost"
                  size="icon"
                  data-testid={chatHeaderTestIds.moreActionsButton}
                  aria-label="Open more chat actions"
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </Tooltip>
            }
            align="right"
          >
            {/* Labels section */}
            <div className="px-3 py-2 border-b border-white/10">
              <p className="text-xs text-slate-500 mb-2">Labels</p>
              {assignedLabels.length > 0 && (
                <div className="flex flex-wrap gap-1 mb-2">
                  {assignedLabels.map((label) => (
                    <button
                      key={label.id}
                      onClick={() => onRemoveLabel(label.id)}
                      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs hover:bg-white/10 transition-colors"
                      title={`Remove ${label.name}`}
                    >
                      <span
                        className="w-2 h-2 rounded-full"
                        style={{ backgroundColor: label.color }}
                      />
                      <span className="text-slate-300">{label.name}</span>
                      <span className="text-slate-500 ml-0.5">×</span>
                    </button>
                  ))}
                </div>
              )}
              {availableLabels.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {availableLabels.map((label) => (
                    <button
                      key={label.id}
                      onClick={() => onAssignLabel(label.id)}
                      className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs text-slate-400 hover:text-white hover:bg-white/10 transition-colors"
                      title={`Add ${label.name}`}
                    >
                      <span
                        className="w-2 h-2 rounded-full opacity-50"
                        style={{ backgroundColor: label.color }}
                      />
                      <span>{label.name}</span>
                    </button>
                  ))}
                </div>
              ) : labels.length === 0 ? (
                <p className="text-xs text-slate-500">No labels yet</p>
              ) : (
                <p className="text-xs text-slate-500">All labels assigned</p>
              )}
            </div>

            {/* Actions */}
            <DropdownItem onClick={onToggleRead} testId={chatHeaderTestIds.toggleReadButton}>
              {chat.is_read ? <Mail className="h-4 w-4" /> : <MailOpen className="h-4 w-4" />}
              {chat.is_read ? "Mark as unread" : "Mark as read"}
            </DropdownItem>
            <DropdownItem onClick={onToggleStar} testId={chatHeaderTestIds.toggleStarButton}>
              <Star className={`h-4 w-4 ${chat.is_starred ? "text-yellow-500 fill-yellow-500" : ""}`} />
              {chat.is_starred ? "Remove star" : "Star chat"}
            </DropdownItem>
            <DropdownItem onClick={onToggleArchive} testId={chatHeaderTestIds.toggleArchiveButton}>
              <Archive className="h-4 w-4" />
              {chat.is_archived ? "Unarchive" : "Archive"}
            </DropdownItem>
            <DropdownSeparator />
            <DropdownItem
              onClick={() => {
                setNewName(chat.name);
                setShowRenameDialog(true);
              }}
              testId={chatHeaderTestIds.renameChatButton}
            >
              <Edit3 className="h-4 w-4" />
              Rename chat
            </DropdownItem>
            <DropdownSeparator />
            <DropdownItem onClick={() => setShowExportDialog(true)} testId="export-chat-button">
              <Download className="h-4 w-4" />
              Export chat
            </DropdownItem>
            <DropdownSeparator />
            <DropdownItem destructive onClick={() => setShowDeleteDialog(true)}>
              <Trash2 className="h-4 w-4" />
              Delete chat
            </DropdownItem>
          </Dropdown>
        </div>
      </header>

      {/* Rename Dialog */}
      <Dialog open={showRenameDialog} onClose={() => setShowRenameDialog(false)}>
        <DialogHeader onClose={() => setShowRenameDialog(false)}>Rename Chat</DialogHeader>
        <DialogBody>
          <Input
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Enter chat name..."
            autoFocus
            onKeyDown={(e) => {
              if (e.key === "Enter") handleRename();
            }}
            data-testid="rename-chat-input"
          />
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => setShowRenameDialog(false)}>
            Cancel
          </Button>
          <Button onClick={handleRename} disabled={!newName.trim()}>
            Save
          </Button>
        </DialogFooter>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={showDeleteDialog} onClose={() => setShowDeleteDialog(false)}>
        <DialogHeader onClose={() => setShowDeleteDialog(false)}>Delete Chat</DialogHeader>
        <DialogBody>
          <p className="text-slate-300">
            Are you sure you want to delete <strong>"{chat.name}"</strong>? This action cannot be
            undone.
          </p>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => setShowDeleteDialog(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={handleDelete} data-testid={chatHeaderTestIds.confirmDeleteButton}>
            Delete
          </Button>
        </DialogFooter>
      </Dialog>

      {/* Export Dialog */}
      <Dialog open={showExportDialog} onClose={() => setShowExportDialog(false)}>
        <DialogHeader onClose={() => setShowExportDialog(false)}>Export Chat</DialogHeader>
        <DialogBody>
          <p className="text-slate-400 text-sm mb-4">
            Choose a format to export "{chat.name}"
          </p>
          <div className="space-y-2">
            <button
              onClick={() => handleExport("markdown")}
              disabled={isExporting}
              className="w-full flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors text-left"
              data-testid="export-markdown-button"
            >
              <FileText className="h-5 w-5 text-indigo-400" />
              <div>
                <div className="font-medium text-white">Markdown (.md)</div>
                <div className="text-xs text-slate-500">Best for documentation and readability</div>
              </div>
            </button>
            <button
              onClick={() => handleExport("json")}
              disabled={isExporting}
              className="w-full flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors text-left"
              data-testid="export-json-button"
            >
              <FileJson className="h-5 w-5 text-emerald-400" />
              <div>
                <div className="font-medium text-white">JSON (.json)</div>
                <div className="text-xs text-slate-500">Complete data with all metadata</div>
              </div>
            </button>
            <button
              onClick={() => handleExport("txt")}
              disabled={isExporting}
              className="w-full flex items-center gap-3 p-3 rounded-lg bg-white/5 hover:bg-white/10 transition-colors text-left"
              data-testid="export-txt-button"
            >
              <File className="h-5 w-5 text-slate-400" />
              <div>
                <div className="font-medium text-white">Plain Text (.txt)</div>
                <div className="text-xs text-slate-500">Simple format for any text editor</div>
              </div>
            </button>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => setShowExportDialog(false)}>
            Cancel
          </Button>
        </DialogFooter>
      </Dialog>
    </>
  );
}
