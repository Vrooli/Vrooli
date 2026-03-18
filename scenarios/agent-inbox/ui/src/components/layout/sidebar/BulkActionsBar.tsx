import {
  Loader2,
  Star,
  Archive,
  Trash2,
  MailOpen,
  MailCheck,
  ArchiveRestore,
} from "lucide-react";
import { Tooltip } from "../../ui/tooltip";
import type { View, BulkOperation } from "./types";

interface BulkActionsBarProps {
  selectedCount: number;
  currentView: View;
  isBulkOperating: boolean;
  onSelectAll: () => void;
  onDeselectAll: () => void;
  onBulkOperation: (operation: BulkOperation) => void;
}

export function BulkActionsBar({
  selectedCount,
  currentView,
  isBulkOperating,
  onSelectAll,
  onDeselectAll,
  onBulkOperation,
}: BulkActionsBarProps) {
  return (
    <div className="px-3 py-2 border-b border-white/10 bg-indigo-500/10 shrink-0" data-testid="bulk-actions-bar">
      <div className="flex items-center gap-2 mb-2">
        <span className="text-sm font-medium text-white">
          {selectedCount} selected
        </span>
        <div className="flex-1" />
        <button
          onClick={onSelectAll}
          className="text-xs text-indigo-400 hover:text-indigo-300"
        >
          Select all
        </button>
        <button
          onClick={onDeselectAll}
          className="text-xs text-slate-400 hover:text-white"
        >
          Clear
        </button>
      </div>
      <div className="flex gap-1 flex-wrap">
        <Tooltip content="Delete selected">
          <button
            onClick={() => onBulkOperation("delete")}
            disabled={isBulkOperating}
            className="p-2 rounded-lg text-red-400 hover:bg-red-500/20 hover:text-red-300 disabled:opacity-50 transition-colors"
            data-testid="bulk-delete"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </Tooltip>
        {currentView !== "archived" ? (
          <Tooltip content="Archive selected">
            <button
              onClick={() => onBulkOperation("archive")}
              disabled={isBulkOperating}
              className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
              data-testid="bulk-archive"
            >
              <Archive className="h-4 w-4" />
            </button>
          </Tooltip>
        ) : (
          <Tooltip content="Unarchive selected">
            <button
              onClick={() => onBulkOperation("unarchive")}
              disabled={isBulkOperating}
              className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
              data-testid="bulk-unarchive"
            >
              <ArchiveRestore className="h-4 w-4" />
            </button>
          </Tooltip>
        )}
        <Tooltip content="Mark as read">
          <button
            onClick={() => onBulkOperation("mark_read")}
            disabled={isBulkOperating}
            className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
            data-testid="bulk-mark-read"
          >
            <MailOpen className="h-4 w-4" />
          </button>
        </Tooltip>
        <Tooltip content="Mark as unread">
          <button
            onClick={() => onBulkOperation("mark_unread")}
            disabled={isBulkOperating}
            className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-white disabled:opacity-50 transition-colors"
            data-testid="bulk-mark-unread"
          >
            <MailCheck className="h-4 w-4" />
          </button>
        </Tooltip>
        <Tooltip content="Star selected">
          <button
            onClick={() => {
              // Toggle star - not directly supported by bulk API
            }}
            disabled={true}
            className="p-2 rounded-lg text-slate-400 hover:bg-white/10 hover:text-yellow-400 disabled:opacity-50 transition-colors"
            data-testid="bulk-star"
          >
            <Star className="h-4 w-4" />
          </button>
        </Tooltip>
      </div>
      {isBulkOperating && (
        <div className="flex items-center gap-2 mt-2 text-xs text-slate-400">
          <Loader2 className="h-3 w-3 animate-spin" />
          Processing...
        </div>
      )}
    </div>
  );
}
