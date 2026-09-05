import { CirclePlay, MessageSquareText, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ContextMenu } from "@vrooli/react-component-library/ContextMenu/1";

import { strings } from "../consts/strings";
import type { RoleMeta } from "../stores/useWorkspaceStore";

// [REQ:P0-014e] Waiting Roles

interface RoleMenuProps {
  role: RoleMeta;
  position: { x: number; y: number };
  onStart: (role: RoleMeta) => void;
  onRename: (role: RoleMeta) => void;
  onEditPrompt: (role: RoleMeta) => void;
  onDelete: (role: RoleMeta) => void;
  onDismiss: () => void;
}

/**
 * Per-role actions.
 *
 * "Edit incoming message" lives here rather than in a settings panel because
 * the prompt belongs to the RECEIVING role (decision D3): the place to change
 * what an implementer is told is the implementer, not the sender.
 */
export default function RoleMenu({
  role,
  position,
  onStart,
  onRename,
  onEditPrompt,
  onDelete,
  onDismiss,
}: RoleMenuProps) {
  const { t } = useTranslation();

  return (
    <ContextMenu
      open
      position={position}
      title={role.label}
      closeLabel={t(strings.roles.roleOptions, { label: role.label })}
      testId="role-menu"
      onOpenChange={(next) => { if (!next) onDismiss(); }}
      items={[
        {
          id: "start",
          label: t(strings.roles.startRole, { label: role.label }),
          icon: <CirclePlay className="h-4 w-4 shrink-0" />,
          testId: "role-menu-start",
          onSelect: () => { onStart(role); },
        },
        {
          id: "rename",
          label: t(strings.roles.renameRole),
          icon: <Pencil className="h-4 w-4 shrink-0" />,
          testId: "role-menu-rename",
          onSelect: () => { onRename(role); },
        },
        {
          id: "edit-prompt",
          label: t(strings.roles.editPrompt),
          icon: <MessageSquareText className="h-4 w-4 shrink-0" />,
          testId: "role-menu-edit-prompt",
          onSelect: () => { onEditPrompt(role); },
        },
        {
          id: "delete",
          label: t(strings.roles.deleteRole),
          icon: <Trash2 className="h-4 w-4 shrink-0" />,
          testId: "role-menu-delete",
          destructive: true,
          separatorBefore: true,
          onSelect: () => { onDelete(role); },
        },
      ]}
    />
  );
}
