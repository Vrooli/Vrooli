import { ChevronDown, ChevronRight, FolderCog, FolderX, Plus } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ContextMenu } from "@vrooli/react-component-library/ContextMenu/1";
import { strings } from "../consts/strings";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";

interface GroupContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  group: TabGroupMeta;
  onNewSession?: () => void;
  onToggleCollapse: () => void;
  /** Open the Manage Groups drawer (rename/recolor live there). */
  onManageGroups: () => void;
  /**
   * Open the close confirmation for this group.
   *
   * Closing a group used to be reachable only by opening the manager and
   * finding the row, so the way it was actually done was closing every
   * session by hand — which left the group behind anyway, since a group
   * outlives its members.
   */
  onCloseGroup: () => void;
  onDismiss: () => void;
}

/**
 * Quick actions on a group header. Bulk administration (rename, recolor,
 * multi-select) stays in the Manage Groups drawer; what lives here is what an
 * operator wants to do to THIS group without leaving the list it is in.
 */
export default function GroupContextMenu({
  position,
  group,
  onNewSession,
  onToggleCollapse,
  onManageGroups,
  onCloseGroup,
  onDismiss,
}: GroupContextMenuProps) {
  const { t } = useTranslation();

  return (
    <ContextMenu
      open
      position={position}
      title={t(strings.manageGroups.menuItem)}
      closeLabel={t(strings.manageGroups.menuItem)}
      testId="group-ctx-menu"
      onOpenChange={(next) => {
        if (!next) onDismiss();
      }}
      items={[
        ...(onNewSession
          ? [{ id: "new-session", label: t(strings.groupContextMenu.newSession), icon: <Plus className="h-4 w-4 shrink-0" />, testId: "group-ctx-new-session", onSelect: onNewSession }]
          : []),
        {
          id: "toggle-collapse",
          label: group.isCollapsed ? t(strings.groupContextMenu.expand) : t(strings.groupContextMenu.collapse),
          icon: group.isCollapsed ? <ChevronDown className="h-4 w-4 shrink-0" /> : <ChevronRight className="h-4 w-4 shrink-0" />,
          testId: "group-ctx-toggle-collapse",
          onSelect: onToggleCollapse,
        },
        {
          id: "manage-groups",
          label: t(strings.manageGroups.menuItem),
          icon: <FolderCog className="h-4 w-4 shrink-0" />,
          testId: "group-ctx-manage-groups",
          separatorBefore: true,
          onSelect: onManageGroups,
        },
        {
          id: "close-group",
          // The ellipsis is a promise: this opens a confirmation, because
          // the operator may also want the sessions closed, and that half is
          // not undoable the way closing the group is.
          label: t(strings.groupContextMenu.closeGroupMenu),
          icon: <FolderX className="h-4 w-4 shrink-0" />,
          testId: "group-ctx-close-group",
          destructive: true,
          separatorBefore: true,
          onSelect: onCloseGroup,
        },
      ]}
    />
  );
}
