import { useLayoutEffect, useRef, useState } from "react";
import { ArrowDown, ArrowUp, ChevronRight, ClipboardCopy, FolderPlus, FolderMinus, Palette, Pencil, Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import ContextMenuBase, { contextMenuItemClass } from "./ContextMenuBase";
import { strings } from "../consts/strings";
import type { TabGroupMeta } from "../stores/useWorkspaceStore";
import { useFloatingPosition } from "../hooks/useFloatingPosition";

interface TabContextMenuProps {
  /** Viewport coordinates where the menu should appear. */
  position: { x: number; y: number };
  /** The ID of the pane this context menu was opened for. */
  sessionId: string;
  /** Current group ID of this pane (null if ungrouped). */
  currentGroupId: string | null;
  /** All available groups for the "Add to group" submenu. */
  groups: TabGroupMeta[];
  onRename: () => void;
  onCustomize: () => void;
  onAddToGroup: (groupId: string) => void;
  onRemoveFromGroup: () => void;
  onCreateGroup: () => void;
  /**
   * Reorder this pane one slot earlier/later. Only supplied by the sidebar in
   * manual-sort mode (and omitted at the list boundaries), so these are the
   * touch-friendly equivalent of the hover-only drag handle. When undefined the
   * menu item is hidden.
   */
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  onClose: (sessionId: string) => void;
  onDismiss: () => void;
}

export default function TabContextMenu({
  position,
  sessionId,
  currentGroupId,
  groups,
  onRename,
  onCustomize,
  onAddToGroup,
  onRemoveFromGroup,
  onCreateGroup,
  onMoveUp,
  onMoveDown,
  onClose,
  onDismiss,
}: TabContextMenuProps) {
  const { t } = useTranslation();
  const [showGroupSubmenu, setShowGroupSubmenu] = useState(false);
  const addToGroupButtonRef = useRef<HTMLButtonElement>(null);
  const groupSubmenuRef = useRef<HTMLDivElement>(null);
  const [groupSubmenuPosition, setGroupSubmenuPosition] = useState<{ x: number; y: number } | null>(null);
  const { computeAnchoredPosition } = useFloatingPosition();

  const handleAction = (action: () => void) => {
    action();
    onDismiss();
  };

  useLayoutEffect(() => {
    if (!showGroupSubmenu) {
      setGroupSubmenuPosition(null);
      return;
    }
    const anchor = addToGroupButtonRef.current;
    const submenu = groupSubmenuRef.current;
    if (!anchor || !submenu) return;
    const position = computeAnchoredPosition({
      anchor: anchor.getBoundingClientRect(),
      size: { width: submenu.offsetWidth, height: submenu.offsetHeight },
      placements: ["right-start", "left-start", "bottom-start", "top-start"],
    });
    setGroupSubmenuPosition({ x: position.x, y: position.y });
  }, [computeAnchoredPosition, groups.length, showGroupSubmenu]);

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

      {/* Group actions */}
      {currentGroupId ? (
        <button
          data-testid="tab-ctx-remove-from-group"
          className={contextMenuItemClass}
          onClick={() => handleAction(onRemoveFromGroup)}
        >
          <FolderMinus className="h-4 w-4 shrink-0" />
          {t(strings.tabContextMenu.removeFromGroup)}
        </button>
      ) : (
        <div
          className="relative"
          onPointerEnter={() => setShowGroupSubmenu(true)}
          onPointerLeave={() => setShowGroupSubmenu(false)}
        >
          <button
            ref={addToGroupButtonRef}
            data-testid="tab-ctx-add-to-group"
            className={contextMenuItemClass}
            onClick={() => setShowGroupSubmenu((prev) => !prev)}
          >
            <FolderPlus className="h-4 w-4 shrink-0" />
            {t(strings.tabContextMenu.addToGroup)}
            <ChevronRight className="h-3 w-3 ml-auto shrink-0" />
          </button>

          {showGroupSubmenu && (
            <div
              ref={groupSubmenuRef}
              data-testid="tab-ctx-group-submenu"
              className="wc-stable-theme fixed z-[60] min-w-[140px] rounded-lg border border-wc-default bg-wc-surface-raised shadow-xl py-1"
              style={
                groupSubmenuPosition
                  ? { left: groupSubmenuPosition.x, top: groupSubmenuPosition.y }
                  : { left: position.x, top: position.y, opacity: 0, pointerEvents: "none" as const }
              }
            >
              {groups.map((group) => (
                <button
                  key={group.id}
                  data-testid={`tab-ctx-group-${group.id}`}
                  className={contextMenuItemClass}
                  onClick={() => handleAction(() => onAddToGroup(group.id))}
                >
                  <span
                    className="h-3 w-3 rounded-full shrink-0"
                    style={{ backgroundColor: group.color }}
                  />
                  {group.name}
                </button>
              ))}
              <button
                data-testid="tab-ctx-new-group"
                className={contextMenuItemClass}
                onClick={() => handleAction(onCreateGroup)}
              >
                <Plus className="h-4 w-4 shrink-0" />
                {t(strings.tabContextMenu.newGroup)}
              </button>
            </div>
          )}
        </div>
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
