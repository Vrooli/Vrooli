import { useState, useCallback } from "react";
import {
  Trash2,
  AlertTriangle,
  BarChart3,
  MailCheck,
  Archive,
  Loader2,
} from "lucide-react";
import { Button } from "../ui/button";
import { Input } from "../ui/input";
import { SettingsSection } from "./SettingsControls";

interface DataSettingsTabProps {
  onDeleteAllChats: () => Promise<unknown>;
  isDeletingAll: boolean;
  onClearArchived?: () => Promise<unknown>;
  isClearingArchived: boolean;
  onMarkAllAsRead?: () => Promise<unknown>;
  isMarkingAllAsRead: boolean;
  onShowUsageStats: () => void;
  onClose: () => void;
}

export function DataSettingsTab({
  onDeleteAllChats,
  isDeletingAll,
  onClearArchived,
  isClearingArchived,
  onMarkAllAsRead,
  isMarkingAllAsRead,
  onShowUsageStats,
  onClose,
}: DataSettingsTabProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState("");
  const [showClearArchivedConfirm, setShowClearArchivedConfirm] = useState(false);

  const handleDeleteAll = useCallback(async () => {
    if (deleteConfirmText !== "delete all") return;
    await onDeleteAllChats();
    setShowDeleteConfirm(false);
    setDeleteConfirmText("");
  }, [deleteConfirmText, onDeleteAllChats]);

  const handleCancelDelete = useCallback(() => {
    setShowDeleteConfirm(false);
    setDeleteConfirmText("");
  }, []);

  const handleClearArchived = useCallback(async () => {
    if (!onClearArchived) return;
    await onClearArchived();
    setShowClearArchivedConfirm(false);
  }, [onClearArchived]);

  const handleMarkAllAsRead = useCallback(async () => {
    if (!onMarkAllAsRead) return;
    await onMarkAllAsRead();
  }, [onMarkAllAsRead]);

  return (
    <div className="space-y-6">
      <SettingsSection
        title="Usage Statistics"
        description="View token usage, costs, and activity across your chats"
      >
        <Button
          variant="secondary"
          onClick={() => {
            onShowUsageStats();
            onClose();
          }}
          className="w-full justify-start gap-2"
          data-testid="usage-stats-button"
        >
          <BarChart3 className="h-4 w-4" />
          View Usage Statistics
        </Button>
      </SettingsSection>

      {onMarkAllAsRead && (
        <SettingsSection title="Quick Actions">
          <Button
            variant="secondary"
            onClick={() => { void handleMarkAllAsRead(); }}
            disabled={isMarkingAllAsRead}
            className="w-full justify-start gap-2"
            data-testid="mark-all-read-button"
          >
            {isMarkingAllAsRead ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <MailCheck className="h-4 w-4" />
            )}
            Mark All as Read
          </Button>
        </SettingsSection>
      )}

      {onClearArchived && (
        <SettingsSection title="Archived Chats">
          <div className="p-4 rounded-lg border border-white/10 bg-white/5">
            {!showClearArchivedConfirm ? (
              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-sm font-medium text-white">Clear Archived Chats</p>
                  <p className="text-xs text-slate-500 mt-1">
                    Permanently delete all archived chats.
                  </p>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setShowClearArchivedConfirm(true)}
                  data-testid="clear-archived-settings-button"
                >
                  <Archive className="h-4 w-4" />
                </Button>
              </div>
            ) : (
              <div className="space-y-3">
                <p className="text-sm text-slate-300">
                  Are you sure you want to delete all archived chats?
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={() => setShowClearArchivedConfirm(false)}
                    className="flex-1"
                  >
                    Cancel
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={() => { void handleClearArchived(); }}
                    disabled={isClearingArchived}
                    className="flex-1"
                    data-testid="confirm-clear-archived-settings-button"
                  >
                    {isClearingArchived ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    ) : (
                      <Trash2 className="h-4 w-4 mr-2" />
                    )}
                    Delete
                  </Button>
                </div>
              </div>
            )}
          </div>
        </SettingsSection>
      )}

      <section>
        <h3 className="text-sm font-medium text-red-400 mb-3 flex items-center gap-2">
          <AlertTriangle className="h-4 w-4" />
          Danger Zone
        </h3>
        <div className="p-4 rounded-lg border border-red-500/20 bg-red-500/5">
          {!showDeleteConfirm ? (
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="text-sm font-medium text-white">Delete All Chats</p>
                <p className="text-xs text-slate-500 mt-1">
                  Permanently delete all chats and messages. This cannot be undone.
                </p>
              </div>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => setShowDeleteConfirm(true)}
                data-testid="delete-all-button"
              >
                <Trash2 className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-slate-300">
                Type <span className="font-mono text-red-400">delete all</span> to confirm:
              </p>
              <Input
                type="text"
                value={deleteConfirmText}
                onChange={(e) => setDeleteConfirmText(e.target.value)}
                placeholder="delete all"
                autoFocus
                className="focus:ring-red-500/50"
                data-testid="delete-confirm-input"
              />
              <div className="flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={handleCancelDelete}
                  className="flex-1"
                >
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => { void handleDeleteAll(); }}
                  disabled={deleteConfirmText !== "delete all" || isDeletingAll}
                  className="flex-1"
                  data-testid="confirm-delete-all-button"
                >
                  {isDeletingAll ? "Deleting..." : "Delete All"}
                </Button>
              </div>
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
