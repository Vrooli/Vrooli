import { useState } from "react";
import { ChevronDown, ChevronRight, FolderMinus, Palette, Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";
import { HEADER_COLORS } from "../consts/config";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";

interface GroupContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  group: TabGroupMeta;
  onRename: () => void;
  onRecolor: (color: string) => void;
  onNewSession?: () => void;
  onToggleCollapse: () => void;
  onUngroupAll: () => void;
  onDelete: () => void;
  onDismiss: () => void;
}

export default function GroupContextMenu({
  position,
  group,
  onRename,
  onRecolor,
  onNewSession,
  onToggleCollapse,
  onUngroupAll,
  onDelete,
  onDismiss,
}: GroupContextMenuProps) {
  const { t } = useTranslation();
  const [showPalette, setShowPalette] = useState(false);

  const handleAction = (action: () => void) => {
    action();
    onDismiss();
  };

  return (
    <ContextMenuBase position={position} onClose={onDismiss} data-testid="group-ctx-menu">
      {/* Rename */}
      <button
        data-testid="group-ctx-rename"
        className={contextMenuItemClass}
        onClick={() => handleAction(onRename)}
      >
        <Pencil className="h-4 w-4 shrink-0" />
        {t(strings.groupContextMenu.rename)}
      </button>

      {/* Recolor (inline palette) */}
      <button
        data-testid="group-ctx-recolor"
        className={contextMenuItemClass}
        onClick={() => setShowPalette((prev) => !prev)}
      >
        <Palette className="h-4 w-4 shrink-0" />
        {t(strings.groupContextMenu.recolor)}
        <ChevronRight className="h-3 w-3 ml-auto shrink-0" />
      </button>
      {showPalette && (
        <div data-testid="group-ctx-palette" className="flex flex-wrap gap-1.5 px-3 py-2">
          {HEADER_COLORS.map((color) => (
            <button
              key={color}
              type="button"
              data-testid={`group-ctx-color-${color}`}
              className="h-5 w-5 rounded-full border border-wc-default"
              style={{ backgroundColor: color }}
              onClick={() => handleAction(() => onRecolor(color))}
              title={color}
            />
          ))}
        </div>
      )}

      {onNewSession && (
        <button
          data-testid="group-ctx-new-session"
          className={contextMenuItemClass}
          onClick={() => handleAction(onNewSession)}
        >
          <Plus className="h-4 w-4 shrink-0" />
          {t(strings.groupContextMenu.newSession)}
        </button>
      )}

      {/* Collapse / Expand */}
      <button
        data-testid="group-ctx-toggle-collapse"
        className={contextMenuItemClass}
        onClick={() => handleAction(onToggleCollapse)}
      >
        {group.isCollapsed ? (
          <ChevronDown className="h-4 w-4 shrink-0" />
        ) : (
          <ChevronRight className="h-4 w-4 shrink-0" />
        )}
        {group.isCollapsed ? t(strings.groupContextMenu.expand) : t(strings.groupContextMenu.collapse)}
      </button>

      <div className="border-t border-wc-default my-1" />

      {/* Ungroup all */}
      <button
        data-testid="group-ctx-ungroup-all"
        className={contextMenuItemClass}
        onClick={() => handleAction(onUngroupAll)}
      >
        <FolderMinus className="h-4 w-4 shrink-0" />
        {t(strings.groupContextMenu.ungroupAll)}
      </button>

      {/* Delete group */}
      <button
        data-testid="group-ctx-delete"
        className={contextMenuItemClass}
        onClick={() => handleAction(onDelete)}
      >
        <Trash2 className="h-4 w-4 shrink-0" />
        {t(strings.groupContextMenu.delete)}
      </button>
    </ContextMenuBase>
  );
}
