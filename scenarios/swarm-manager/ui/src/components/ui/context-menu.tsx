/**
 * ContextMenu — right-click / long-press action menu anchored to a pointer
 * position. Shares the canonical Popover surface and ActionMenuItems rows used
 * by every other menu in the app, so a context menu looks identical to the
 * click-triggered ActionMenu.
 *
 * Pair with the {@link useContextMenu} hook:
 * ```tsx
 * const menu = useContextMenu();
 * return (
 *   <>
 *     <div {...menu.triggerProps}>…row…</div>
 *     <ContextMenu origin={menu.origin} onClose={menu.close} items={items} />
 *   </>
 * );
 * ```
 */

import { Popover } from "./popover";
import { ActionMenuItems, type ActionMenuItem } from "./action-menu";
import type { ContextMenuOrigin } from "./use-context-menu";

export interface ContextMenuProps {
  origin: ContextMenuOrigin | null;
  onClose: () => void;
  items: ActionMenuItem[];
  testId?: string;
}

/**
 * Renders the context menu at the captured pointer origin. Nothing is rendered
 * while `origin` is null or there are no items.
 */
export function ContextMenu({ origin, onClose, items, testId = "context-menu" }: ContextMenuProps) {
  if (items.length === 0) return null;
  return (
    <Popover
      isOpen={origin !== null}
      onClose={onClose}
      x={origin?.x}
      y={origin?.y}
      // Right-click fires a mousedown; delay the click-outside listener so the
      // menu doesn't instantly close on the very event that opened it.
      delayClickOutside
      className="min-w-[200px] overflow-hidden py-1"
      testId={testId}
    >
      <ActionMenuItems items={items} onItemSelected={onClose} role="menu" itemRole="menuitem" />
    </Popover>
  );
}
