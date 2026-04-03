/**
 * FileActionMenu
 *
 * Context menu / action menu items for file operations,
 * extracted from BacklogFileBrowser.
 */

import { useCallback } from "react";
import { ArrowRightLeft, Copy, Edit, Lock, Trash2 } from "lucide-react";
import { cn } from "../../lib";
import type { FileActionType } from "./backlog-file-browser";
import type { BacklogFile } from "../../types";

export interface FileActionMenuProps {
  onOpenActionDialog: (action: FileActionType, target: BacklogFile) => void;
}

/**
 * Hook that returns a stable `renderFileActionItems` callback.
 */
export function useFileActionMenuRenderer({ onOpenActionDialog }: FileActionMenuProps) {
  const renderFileActionItems = useCallback(
    (target: BacklogFile, closeMenu: () => void) => {
      const isProtected = target.path === "spec.json";
      const rowClass =
        "flex w-full items-center justify-start gap-2 px-3 py-2 text-sm text-slate-100 hover:bg-slate-800/80";
      return (
        <div className="py-1" data-testid="backlog-file-actions-menu">
          <button
            type="button"
            className={rowClass}
            disabled={isProtected}
            onClick={() => {
              closeMenu();
              onOpenActionDialog("rename", target);
            }}
          >
            <Edit className="h-4 w-4 text-slate-300" />
            Rename
          </button>
          <button
            type="button"
            className={rowClass}
            disabled={isProtected}
            onClick={() => {
              closeMenu();
              onOpenActionDialog("move", target);
            }}
          >
            <ArrowRightLeft className="h-4 w-4 text-slate-300" />
            Move
          </button>
          <button
            type="button"
            className={rowClass}
            disabled={isProtected}
            onClick={() => {
              closeMenu();
              onOpenActionDialog("copy", target);
            }}
          >
            <Copy className="h-4 w-4 text-slate-300" />
            Copy
          </button>
          <button
            type="button"
            className={cn(rowClass, "text-red-300 hover:bg-red-500/20")}
            disabled={isProtected}
            onClick={() => {
              closeMenu();
              onOpenActionDialog("delete", target);
            }}
          >
            <Trash2 className="h-4 w-4 text-red-300" />
            Delete
          </button>
          {isProtected && (
            <p className="flex items-center gap-2 px-3 py-2 text-xs text-slate-400">
              <Lock className="h-3.5 w-3.5" />
              `spec.json` is protected.
            </p>
          )}
        </div>
      );
    },
    [onOpenActionDialog],
  );

  return renderFileActionItems;
}
