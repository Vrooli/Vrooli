import { useState } from "react";
import {
  Mail,
  MailOpen,
  Star,
  Archive,
  Trash2,
  Edit3,
  ChevronDown,
  Bot,
  Tag,
  MoreVertical,
} from "lucide-react";
import { Button } from "../ui/button";
import { Tooltip } from "../ui/tooltip";
import { Dropdown, DropdownItem, DropdownSeparator } from "../ui/dropdown";
import { Dialog, DialogHeader, DialogBody, DialogFooter } from "../ui/dialog";
import { Input } from "../ui/input";
import { Badge } from "../ui/badge";
import type { Chat, Model, Label } from "../../lib/api";

interface ChatHeaderProps {
  chat: Chat;
  models: Model[];
  labels: Label[];
  onUpdateChat: (data: { name?: string; model?: string }) => void;
  onToggleRead: () => void;
  onToggleStar: () => void;
  onToggleArchive: () => void;
  onDelete: () => void;
  onAssignLabel: (labelId: string) => void;
  onRemoveLabel: (labelId: string) => void;
}

export function ChatHeader({
  chat,
  models,
  labels,
  onUpdateChat,
  onToggleRead,
  onToggleStar,
  onToggleArchive,
  onDelete,
  onAssignLabel,
  onRemoveLabel,
}: ChatHeaderProps) {
  const [showRenameDialog, setShowRenameDialog] = useState(false);
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const [showModelDropdown, setShowModelDropdown] = useState(false);
  const [newName, setNewName] = useState(chat.name);

  const currentModel = models.find((m) => m.id === chat.model);
  const assignedLabels = labels.filter((l) => chat.label_ids.includes(l.id));
  const availableLabels = labels.filter((l) => !chat.label_ids.includes(l.id));

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
      <header className="p-4 border-b border-white/10 bg-slate-950/50" data-testid="chat-header">
        <div className="flex items-start justify-between gap-4">
          {/* Left Section - Title & Info */}
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold text-white truncate">{chat.name}</h2>
              <Tooltip content="Rename chat">
                <button
                  onClick={() => {
                    setNewName(chat.name);
                    setShowRenameDialog(true);
                  }}
                  className="p-1 rounded hover:bg-white/10 text-slate-500 hover:text-white transition-colors"
                  data-testid="rename-chat-button"
                >
                  <Edit3 className="h-3.5 w-3.5" />
                </button>
              </Tooltip>
            </div>

            {/* Model Selector & Mode */}
            <div className="flex items-center gap-3 mt-1.5">
              <Dropdown
                trigger={
                  <button
                    className="flex items-center gap-1.5 text-xs text-slate-400 hover:text-white transition-colors"
                    data-testid="model-selector-button"
                  >
                    <Bot className="h-3.5 w-3.5" />
                    <span>{currentModel?.name || chat.model}</span>
                    <ChevronDown className="h-3 w-3" />
                  </button>
                }
                className="w-72"
              >
                <div className="p-2">
                  <p className="text-xs text-slate-500 px-2 mb-1">Select a model</p>
                  {models.map((model) => (
                    <DropdownItem
                      key={model.id}
                      onClick={() => onUpdateChat({ model: model.id })}
                      className={chat.model === model.id ? "bg-white/10" : ""}
                    >
                      <div>
                        <div className="font-medium">{model.name}</div>
                        {model.description && (
                          <div className="text-xs text-slate-500 mt-0.5">{model.description}</div>
                        )}
                      </div>
                    </DropdownItem>
                  ))}
                </div>
              </Dropdown>

              <span className="text-xs text-slate-600">•</span>
              <span className="text-xs text-slate-500">{chat.view_mode} mode</span>
            </div>

            {/* Labels */}
            {assignedLabels.length > 0 && (
              <div className="flex items-center gap-1.5 mt-2 flex-wrap">
                {assignedLabels.map((label) => (
                  <Badge key={label.id} color={label.color} onRemove={() => onRemoveLabel(label.id)}>
                    {label.name}
                  </Badge>
                ))}
              </div>
            )}
          </div>

          {/* Right Section - Actions */}
          <div className="flex items-center gap-1 shrink-0">
            {/* Label Dropdown */}
            <Dropdown
              trigger={
                <Tooltip content="Add label">
                  <Button variant="ghost" size="icon" data-testid="add-label-button">
                    <Tag className="h-4 w-4" />
                  </Button>
                </Tooltip>
              }
              align="right"
            >
              <div className="p-2">
                {availableLabels.length > 0 ? (
                  <>
                    <p className="text-xs text-slate-500 px-2 mb-1">Add a label</p>
                    {availableLabels.map((label) => (
                      <DropdownItem key={label.id} onClick={() => onAssignLabel(label.id)}>
                        <span
                          className="w-2.5 h-2.5 rounded-full shrink-0"
                          style={{ backgroundColor: label.color }}
                        />
                        {label.name}
                      </DropdownItem>
                    ))}
                  </>
                ) : labels.length === 0 ? (
                  <p className="text-xs text-slate-500 px-2 py-2">
                    No labels yet. Create labels from the sidebar.
                  </p>
                ) : (
                  <p className="text-xs text-slate-500 px-2 py-2">All labels assigned</p>
                )}
              </div>
            </Dropdown>

            <Tooltip content={chat.is_read ? "Mark as unread" : "Mark as read"}>
              <Button variant="ghost" size="icon" onClick={onToggleRead} data-testid="toggle-read-button">
                {chat.is_read ? <Mail className="h-4 w-4" /> : <MailOpen className="h-4 w-4" />}
              </Button>
            </Tooltip>

            <Tooltip content={chat.is_starred ? "Remove star" : "Star chat"}>
              <Button variant="ghost" size="icon" onClick={onToggleStar} data-testid="toggle-star-button">
                <Star
                  className={`h-4 w-4 ${
                    chat.is_starred ? "text-yellow-500 fill-yellow-500" : ""
                  }`}
                />
              </Button>
            </Tooltip>

            <Tooltip content={chat.is_archived ? "Unarchive" : "Archive"}>
              <Button variant="ghost" size="icon" onClick={onToggleArchive} data-testid="toggle-archive-button">
                <Archive className="h-4 w-4" />
              </Button>
            </Tooltip>

            <Dropdown
              trigger={
                <Tooltip content="More actions">
                  <Button variant="ghost" size="icon" data-testid="chat-more-actions">
                    <MoreVertical className="h-4 w-4" />
                  </Button>
                </Tooltip>
              }
              align="right"
            >
              <DropdownItem
                onClick={() => {
                  setNewName(chat.name);
                  setShowRenameDialog(true);
                }}
              >
                <Edit3 className="h-4 w-4" />
                Rename chat
              </DropdownItem>
              <DropdownSeparator />
              <DropdownItem destructive onClick={() => setShowDeleteDialog(true)}>
                <Trash2 className="h-4 w-4" />
                Delete chat
              </DropdownItem>
            </Dropdown>
          </div>
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
          <Button variant="destructive" onClick={handleDelete} data-testid="confirm-delete-button">
            Delete
          </Button>
        </DialogFooter>
      </Dialog>
    </>
  );
}
