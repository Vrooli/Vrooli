import { ArrowDown, ArrowUp, ClipboardCopy, Dot, FolderCog, FolderMinus, FolderPlus, MailOpen, Palette, Pencil, Trash2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ContextMenu, type ContextMenuItem } from "@vrooli/react-component-library/ContextMenu/1";
import { strings } from "../consts/strings";
import { writeText } from "../lib/clipboard";

interface TabContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  /** The ID of the pane this context menu was opened for. */
  sessionId: string;
  /** Current group ID of this pane (null if ungrouped). */
  currentGroupId: string | null;
  /** Whether this pane currently carries the manual unread flag. */
  isManuallyUnread: boolean;
  /** Toggle the manual unread flag. */
  onToggleManuallyUnread: () => void;
  onRename: () => void;
  onCustomize: () => void;
  onRemoveFromGroup: () => void;
  /**
   * Open the anchored group picker for this session.
   *
   * Assignment used to open the whole Manage Groups drawer, which is why
   * putting one session in one group felt like a detour through an
   * administration surface.
   */
  onAssignToGroup: () => void;
  /**
   * Reorder this pane one slot earlier/later. Only supplied by the sidebar in
   * manual-sort mode (and omitted at the list boundaries), so these are the
   * touch-friendly equivalent of the hover-only drag handle. When undefined the
   * menu item is hidden.
   */
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  onClose: (sessionId: string) => void;
  onDeletePermanently: (sessionId: string) => void;
  onDismiss: () => void;
}

export default function TabContextMenu({
  position,
  sessionId,
  currentGroupId,
  isManuallyUnread,
  onToggleManuallyUnread,
  onRename,
  onCustomize,
  onRemoveFromGroup,
  onAssignToGroup,
  onMoveUp,
  onMoveDown,
  onClose,
  onDeletePermanently,
  onDismiss,
}: TabContextMenuProps) {
  const { t } = useTranslation();
  const items: ContextMenuItem[] = [
    {
      id: "rename",
      label: t(strings.tabContextMenu.rename),
      icon: <Pencil className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-rename",
      onSelect: onRename,
    },
    {
      id: "toggle-unread",
      label: t(isManuallyUnread ? strings.tabContextMenu.markAsRead : strings.tabContextMenu.markAsUnread),
      icon: isManuallyUnread ? <MailOpen className="h-4 w-4 shrink-0" /> : <Dot className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-toggle-unread",
      onSelect: onToggleManuallyUnread,
    },
    {
      id: "customize",
      label: t(strings.tabContextMenu.customizeAppearance),
      icon: <Palette className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-customize",
      onSelect: onCustomize,
    },
  ];
  if (currentGroupId) {
    items.push(
      {
        id: "remove-from-group",
        label: t(strings.tabContextMenu.removeFromGroup),
        icon: <FolderMinus className="h-4 w-4 shrink-0" />,
        testId: "tab-ctx-remove-from-group",
        separatorBefore: true,
        onSelect: onRemoveFromGroup,
      },
      {
        id: "move-to-group",
        label: t(strings.tabContextMenu.moveToGroup),
        icon: <FolderCog className="h-4 w-4 shrink-0" />,
        testId: "tab-ctx-move-to-group",
        onSelect: onAssignToGroup,
      },
    );
  } else {
    items.push({
      id: "add-to-group",
      label: t(strings.tabContextMenu.addToGroup),
      icon: <FolderPlus className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-add-to-group",
      separatorBefore: true,
      onSelect: onAssignToGroup,
    });
  }
  if (onMoveUp) {
    items.push({
      id: "move-up",
      label: t(strings.tabContextMenu.moveUp),
      icon: <ArrowUp className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-move-up",
      separatorBefore: true,
      onSelect: onMoveUp,
    });
  }
  if (onMoveDown) {
    items.push({
      id: "move-down",
      label: t(strings.tabContextMenu.moveDown),
      icon: <ArrowDown className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-move-down",
      separatorBefore: !onMoveUp,
      onSelect: onMoveDown,
    });
  }
  items.push(
    {
      id: "close",
      label: t(strings.tabContextMenu.closeTab),
      icon: <X className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-close",
      separatorBefore: true,
      onSelect: () => { onClose(sessionId); },
    },
    {
      id: "delete-permanently",
      label: t(strings.tabContextMenu.deletePermanently),
      icon: <Trash2 className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-delete-permanently",
      destructive: true,
      onSelect: () => { onDeletePermanently(sessionId); },
    },
    {
      id: "copy-debug-log",
      label: t(strings.tabContextMenu.copyDebugLog),
      icon: <ClipboardCopy className="h-4 w-4 shrink-0" />,
      testId: "tab-ctx-copy-debug-log",
      separatorBefore: true,
      onSelect: () => {
        const probe = (window as unknown as {
          __wc_terminal_output?: Record<string, string>;
        }).__wc_terminal_output;
        const data = probe?.[sessionId] ?? "";
        const payload = data || "(empty probe)";
        void writeText(payload).then((result) => { alert(result.ok ? `Copied ${data.length} chars` : `Clipboard ${result.reason}`); });
      },
    },
  );

  return (
    <ContextMenu
      open
      position={position}
      title={t(strings.tabContextMenu.closeTab)}
      closeLabel={t(strings.tabContextMenu.closeTab)}
      items={items}
      onOpenChange={(next) => {
        if (!next) onDismiss();
      }}
    />
  );
}
