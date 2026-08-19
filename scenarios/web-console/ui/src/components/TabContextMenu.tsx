import { ArrowDown, ArrowUp, ClipboardCopy, Dot, FolderCog, FolderMinus, FolderPlus, MailOpen, Palette, Pencil, Trash2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";

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
  /** Open the Manage Groups drawer with this session as context. */
  onManageGroups: () => void;
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
  onManageGroups,
  onMoveUp,
  onMoveDown,
  onClose,
  onDeletePermanently,
  onDismiss,
}: TabContextMenuProps) {
  const { t } = useTranslation();

  const handleAction = (action: () => void) => {
    action();
    onDismiss();
  };

  return (
    <ContextMenuBase position={position} onClose={onDismiss}>
      {/* Rename */}
      <button
        data-testid="tab-ctx-rename"
        className={contextMenuItemClass}
        onClick={() => handleAction(onRename)}
      >
        <Pencil className="h-4 w-4 shrink-0" />
        {t(strings.tabContextMenu.rename)}
      </button>

      {/* Mark as unread / read. A deliberate flag, not a derived state: it
          survives being read and works on sessions with no conversation. */}
      <button
        data-testid="tab-ctx-toggle-unread"
        className={contextMenuItemClass}
        onClick={() => handleAction(onToggleManuallyUnread)}
      >
        {isManuallyUnread
          ? <MailOpen className="h-4 w-4 shrink-0" />
          : <Dot className="h-4 w-4 shrink-0" />}
        {t(isManuallyUnread
          ? strings.tabContextMenu.markAsRead
          : strings.tabContextMenu.markAsUnread)}
      </button>

      {/* Customize appearance */}
      <button
        data-testid="tab-ctx-customize"
        className={contextMenuItemClass}
        onClick={() => handleAction(onCustomize)}
      >
        <Palette className="h-4 w-4 shrink-0" />
        {t(strings.tabContextMenu.customizeAppearance)}
      </button>

      {/* Divider */}
      <div className="border-t border-wc-default my-1" />

      {/* Group membership: "Add to Group" opens the Manage Groups drawer with
          this session as context (assign, create, rename, delete all live
          there); grouped panes get a one-tap remove plus the same drawer. */}
      {currentGroupId ? (
        <>
          <button
            data-testid="tab-ctx-remove-from-group"
            className={contextMenuItemClass}
            onClick={() => handleAction(onRemoveFromGroup)}
          >
            <FolderMinus className="h-4 w-4 shrink-0" />
            {t(strings.tabContextMenu.removeFromGroup)}
          </button>
          <button
            data-testid="tab-ctx-manage-groups"
            className={contextMenuItemClass}
            onClick={() => handleAction(onManageGroups)}
          >
            <FolderCog className="h-4 w-4 shrink-0" />
            {t(strings.manageGroups.menuItem)}
          </button>
        </>
      ) : (
        <button
          data-testid="tab-ctx-add-to-group"
          className={contextMenuItemClass}
          onClick={() => handleAction(onManageGroups)}
        >
          <FolderPlus className="h-4 w-4 shrink-0" />
          {t(strings.tabContextMenu.addToGroup)}
        </button>
      )}

      {/* Reorder (touch-friendly stand-in for the hover-only drag handle) */}
      {(onMoveUp || onMoveDown) && (
        <>
          <div className="border-t border-wc-default my-1" />
          {onMoveUp && (
            <button
              data-testid="tab-ctx-move-up"
              className={contextMenuItemClass}
              onClick={() => handleAction(onMoveUp)}
            >
              <ArrowUp className="h-4 w-4 shrink-0" />
              {t(strings.tabContextMenu.moveUp)}
            </button>
          )}
          {onMoveDown && (
            <button
              data-testid="tab-ctx-move-down"
              className={contextMenuItemClass}
              onClick={() => handleAction(onMoveDown)}
            >
              <ArrowDown className="h-4 w-4 shrink-0" />
              {t(strings.tabContextMenu.moveDown)}
            </button>
          )}
        </>
      )}

      {/* Divider */}
      <div className="border-t border-wc-default my-1" />

      {/* Close tab */}
      <button
        data-testid="tab-ctx-close"
        className={contextMenuItemClass}
        onClick={() => handleAction(() => onClose(sessionId))}
      >
        <X className="h-4 w-4 shrink-0" />
        {t(strings.tabContextMenu.closeTab)}
      </button>

      <button
        data-testid="tab-ctx-delete-permanently"
        className={`${contextMenuItemClass} text-red-300 hover:text-red-200`}
        onClick={() => handleAction(() => onDeletePermanently(sessionId))}
      >
        <Trash2 className="h-4 w-4 shrink-0" />
        {t(strings.tabContextMenu.deletePermanently)}
      </button>

      {/* TEMP: remove after the terminal-output-duplication bug is fixed.
       * Copies the last ~12k chars of xterm writes for this session to
       * the clipboard so a phone-only user can share the same data
       * normally pulled from window.__wc_terminal_output via devtools. */}
      <div className="border-t border-wc-default my-1" />
      <button
        data-testid="tab-ctx-copy-debug-log"
        className={contextMenuItemClass}
        onClick={() =>
          handleAction(() => {
            const probe = (window as unknown as {
              __wc_terminal_output?: Record<string, string>;
            }).__wc_terminal_output;
            const data = probe?.[sessionId] ?? "";
            const payload = data || "(empty probe)";
            if (navigator.clipboard?.writeText) {
              navigator.clipboard
                .writeText(payload)
                .then(() => alert(`Copied ${data.length} chars`))
                .catch(() => alert("Clipboard denied"));
            } else {
              alert("Clipboard unavailable");
            }
          })
        }
      >
        <ClipboardCopy className="h-4 w-4 shrink-0" />
        {t(strings.tabContextMenu.copyDebugLog)}
      </button>
    </ContextMenuBase>
  );
}
