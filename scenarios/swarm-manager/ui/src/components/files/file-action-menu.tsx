/**
 * FileActionMenu
 *
 * Context menu / action menu items for file operations,
 * extracted from BacklogFileBrowser.
 */

import { useCallback } from "react";
import { ArrowRightLeft, Copy, Edit, Lock, Trash2 } from "lucide-react";
import { ActionMenuItemButton, type ActionMenuItem } from "../ui/action-menu";
import { useFileService } from "../../contexts/FileServiceContext";
import type { FileActionType } from "./backlog-file-browser";
import type { BacklogFile } from "../../types";

export interface FileActionMenuProps {
  onOpenActionDialog: (action: FileActionType, target: BacklogFile) => void;
}

/**
 * Hook that returns a stable `renderFileActionItems` callback.
 */
export function useFileActionMenuRenderer({ onOpenActionDialog }: FileActionMenuProps) {
  const fileService = useFileService();
  const renderFileActionItems = useCallback(
    (target: BacklogFile, closeMenu: () => void) => {
      const isProtected = target.path === fileService.protectedFile;
      const items: ActionMenuItem[] = [
        {
          label: "Rename",
          icon: <Edit />,
          disabled: isProtected,
          onSelect: () => {
            closeMenu();
            onOpenActionDialog("rename", target);
          },
        },
        {
          label: "Move",
          icon: <ArrowRightLeft />,
          disabled: isProtected,
          onSelect: () => {
            closeMenu();
            onOpenActionDialog("move", target);
          },
        },
        {
          label: "Copy",
          icon: <Copy />,
          disabled: isProtected,
          onSelect: () => {
            closeMenu();
            onOpenActionDialog("copy", target);
          },
        },
        {
          label: "Delete",
          icon: <Trash2 />,
          destructive: true,
          disabled: isProtected,
          onSelect: () => {
            closeMenu();
            onOpenActionDialog("delete", target);
          },
        },
      ];

      return (
        <div className="py-1" data-testid="backlog-file-actions-menu">
          {items.map((item) => (
            <ActionMenuItemButton key={item.label} item={item} />
          ))}
          {isProtected && (
            <p className="flex items-center gap-2 px-3 py-2 text-xs text-slate-400">
              <Lock className="h-3.5 w-3.5" />
              {`\`${fileService.protectedFile}\` is protected.`}
            </p>
          )}
        </div>
      );
    },
    [onOpenActionDialog, fileService.protectedFile],
  );

  return renderFileActionItems;
}
