import { ChevronDown, ChevronRight, FolderCog, Plus } from "lucide-react";
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
      ]}
    />
  );
}
