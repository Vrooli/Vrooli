import { ChevronDown, ChevronRight, FolderCog, Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";

interface GroupContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  group: TabGroupMeta;
  onNewSession?: () => void;
  onToggleCollapse: () => void;
  /** Open the Manage Groups drawer (rename/recolor/delete live there). */
  onManageGroups: () => void;
  onDismiss: () => void;
}

/**
 * Quick ephemeral actions on a group header. Management operations (rename,
 * recolor, ungroup, delete) live in the Manage Groups drawer — this menu only
 * keeps the in-place toggles plus a deep link into that drawer.
 */
export default function GroupContextMenu({
  position,
  group,
  onNewSession,
  onToggleCollapse,
  onManageGroups,
  onDismiss,
}: GroupContextMenuProps) {
  const { t } = useTranslation();

  const handleAction = (action: () => void) => {
    action();
    onDismiss();
  };

  return (
    <ContextMenuBase position={position} onClose={onDismiss} data-testid="group-ctx-menu">
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

      <button
        data-testid="group-ctx-manage-groups"
        className={contextMenuItemClass}
        onClick={() => handleAction(onManageGroups)}
      >
        <FolderCog className="h-4 w-4 shrink-0" />
        {t(strings.manageGroups.menuItem)}
      </button>
    </ContextMenuBase>
  );
}
