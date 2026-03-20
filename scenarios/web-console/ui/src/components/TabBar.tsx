import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight, Plus, X } from "lucide-react";
import { useWorkspaceStore, type PaneMetadata } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import { useLongPress } from "../hooks/useLongPress";
import TabContextMenu from "./TabContextMenu";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { useConversationStore } from "../stores/useConversationStore";

interface TabBarProps {
  panes: PaneMetadata[];
  activePane: string | null;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  onClosePane: (sessionId: string) => void;
  isCreating?: boolean;
  /** Extra action buttons rendered before the plus button (e.g. settings on mobile). */
  trailingActions?: React.ReactNode;
}

export default function TabBar({
  panes,
  activePane,
  onNewTerminal,
  onOpenLauncher,
  onClosePane,
  isCreating,
  trailingActions,
}: TabBarProps) {
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const displayMode = useWorkspaceStore((s) => s.displayMode);
  const groups = useWorkspaceStore((s) => s.groups);
  const tabContextMenu = useWorkspaceStore((s) => s.tabContextMenu);
  const setTabContextMenu = useWorkspaceStore((s) => s.setTabContextMenu);
  const setPaneGroup = useWorkspaceStore((s) => s.setPaneGroup);
  const toggleGroupCollapsed = useWorkspaceStore((s) => s.toggleGroupCollapsed);
  const addGroup = useWorkspaceStore((s) => s.addGroup);
  const updateGroup = useWorkspaceStore((s) => s.updateGroup);
  const conversationSessions = useConversationStore((s) => s.sessions);
  const { syncPaneOrder, syncActivePane, syncCreateGroup, syncPaneUpdate, syncUpdateGroup } = useWorkspaceSync();

  // Inline rename state for tabs
  const [editingTabId, setEditingTabId] = useState<string | null>(null);
  const [editTabName, setEditTabName] = useState("");
  const editTabInputRef = useRef<HTMLInputElement>(null);

  // Inline rename state for groups
  const [editingGroupId, setEditingGroupId] = useState<string | null>(null);
  const [editGroupName, setEditGroupName] = useState("");
  const editGroupInputRef = useRef<HTMLInputElement>(null);

  /** Start inline rename for a tab. */
  const startTabRename = useCallback((sessionId: string, currentName: string) => {
    setEditingTabId(sessionId);
    setEditTabName(currentName);
  }, []);

  /** Commit tab rename. */
  const commitTabRename = useCallback(() => {
    if (editingTabId && editTabName.trim()) {
      const trimmed = editTabName.trim();
      renamePaneById(editingTabId, trimmed);
      syncPaneUpdate(editingTabId, { name: trimmed });
    }
    setEditingTabId(null);
    setEditTabName("");
  }, [editingTabId, editTabName, renamePaneById, syncPaneUpdate]);

  /** Start inline rename for a group. */
  const startGroupRename = useCallback((groupId: string, currentName: string) => {
    setEditingGroupId(groupId);
    setEditGroupName(currentName);
  }, []);

  /** Commit group rename. */
  const commitGroupRename = useCallback(() => {
    if (editingGroupId && editGroupName.trim()) {
      const trimmed = editGroupName.trim();
      updateGroup(editingGroupId, { name: trimmed });
      syncUpdateGroup(editingGroupId, { name: trimmed });
    }
    setEditingGroupId(null);
    setEditGroupName("");
  }, [editingGroupId, editGroupName, updateGroup, syncUpdateGroup]);

  // Auto-focus rename inputs when they appear
  useEffect(() => {
    if (editingTabId && editTabInputRef.current) {
      editTabInputRef.current.focus();
      editTabInputRef.current.select();
    }
  }, [editingTabId]);

  useEffect(() => {
    if (editingGroupId && editGroupInputRef.current) {
      editGroupInputRef.current.focus();
      editGroupInputRef.current.select();
    }
  }, [editingGroupId]);

  // Long-press detection for opening context menu on tabs.
  // The timer sets longPressReady; the menu only opens on pointerUp
  // so that drag gestures take priority over long-press.
  const longPressTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const longPressFired = useRef(false);
  const longPressReady = useRef(false);
  const longPressPaneId = useRef<string | null>(null);

  const plusButtonBehavior = useWorkspaceStore((s) => s.plusButtonBehavior);
  const plusHandlers = useLongPress({
    onPress: plusButtonBehavior === "launcher" ? onOpenLauncher : onNewTerminal,
    onLongPress: plusButtonBehavior === "launcher" ? onNewTerminal : onOpenLauncher,
  });

  const tabContainerRef = useRef<HTMLDivElement>(null);
  const activeTabRef = useRef<HTMLDivElement>(null);

  // Drag state for tab reordering
  const [dragState, setDragState] = useState<{
    paneId: string;
    dropIndex: number;
  } | null>(null);

  // Pending drag: tracks pointer-down position before movement threshold is met
  const dragStartRef = useRef<{
    paneId: string;
    x: number;
    y: number;
    pointerId: number;
    target: HTMLElement;
  } | null>(null);

  /** Movement threshold (px) before a pointer-down becomes a drag. */
  const DRAG_THRESHOLD = 5;

  const activatePane = useCallback((sessionId: string) => {
    setActivePane(sessionId);
    syncActivePane(useWorkspaceStore.getState().panes.map((p) => p.sessionId), sessionId);
  }, [setActivePane, syncActivePane]);

  // Auto-scroll active tab into view
  useEffect(() => {
    if (activeTabRef.current && tabContainerRef.current) {
      activeTabRef.current.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
        inline: "nearest",
      });
    }
  }, [activePane]);

  // Keyboard shortcuts for tab navigation (only in tabs mode)
  useEffect(() => {
    if (displayMode !== "tabs") return;

    const handleKeyDown = (e: KeyboardEvent) => {
      const activeIdx = panes.findIndex((p) => p.sessionId === activePane);

      // Ctrl+Tab / Ctrl+Shift+Tab - cycle tabs
      if (e.ctrlKey && e.key === "Tab") {
        e.preventDefault();
        if (panes.length === 0) return;

        const direction = e.shiftKey ? -1 : 1;
        const nextIdx = (activeIdx + direction + panes.length) % panes.length;
        const nextPane = panes[nextIdx];
        if (nextPane) {
          activatePane(nextPane.sessionId);
        }
        return;
      }

      // Ctrl+1-9 - jump to tab by index
      if (e.ctrlKey && !e.shiftKey && !e.altKey && /^[1-9]$/.test(e.key)) {
        const idx = parseInt(e.key, 10) - 1;
        if (idx < panes.length) {
          e.preventDefault();
          const targetPane = panes[idx];
          if (targetPane) {
            activatePane(targetPane.sessionId);
          }
        }
        return;
      }

      // Ctrl+W - close active tab
      if (e.ctrlKey && !e.shiftKey && !e.altKey && e.key === "w") {
        e.preventDefault();
        if (activePane) {
          onClosePane(activePane);
        }
        return;
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [displayMode, panes, activePane, activatePane, onClosePane]);

  // Initiate actual drag once movement exceeds threshold
  const commitDrag = useCallback(
    (paneId: string, target: HTMLElement, pointerId: number) => {
      // Cancel any pending long-press when drag starts
      if (longPressTimer.current) {
        clearTimeout(longPressTimer.current);
        longPressTimer.current = null;
      }
      longPressReady.current = false;
      longPressPaneId.current = null;

      const idx = panes.findIndex((p) => p.sessionId === paneId);
      if (idx === -1) return;
      target.setPointerCapture(pointerId);
      setDragState({ paneId, dropIndex: idx });
    },
    [panes],
  );

  // Global pointermove/pointerup to detect drag threshold and manage active drag
  useEffect(() => {
    const handleMove = (e: PointerEvent) => {
      // Check if we need to promote a pending drag
      const pending = dragStartRef.current;
      if (pending && e.pointerId === pending.pointerId) {
        const dx = e.clientX - pending.x;
        const dy = e.clientY - pending.y;
        if (Math.sqrt(dx * dx + dy * dy) > DRAG_THRESHOLD) {
          commitDrag(pending.paneId, pending.target, pending.pointerId);
          dragStartRef.current = null;
        }
        return;
      }

      // Update drop target for active drag
      if (dragState === null) return;
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const tabEl = el?.closest("[data-tab-index]");
      if (tabEl) {
        const idx = Number(tabEl.getAttribute("data-tab-index"));
        if (Number.isFinite(idx)) {
          setDragState((prev) =>
            prev ? { ...prev, dropIndex: idx } : null,
          );
        }
      }
    };

    const handleUp = () => {
      dragStartRef.current = null;
      setDragState((prev) => {
        if (prev) {
          movePaneToIndex(prev.paneId, prev.dropIndex);
          // Zustand set() is synchronous, so getState() reflects the new order
          const { panes: updated, activePane: active } = useWorkspaceStore.getState();
          syncPaneOrder(updated.map((p) => p.sessionId), active);
        }
        return null;
      });
    };

    window.addEventListener("pointermove", handleMove);
    window.addEventListener("pointerup", handleUp);
    window.addEventListener("pointercancel", handleUp);
    return () => {
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleUp);
      window.removeEventListener("pointercancel", handleUp);
    };
  }, [dragState, movePaneToIndex, syncPaneOrder, commitDrag]);

  const isDragging = dragState !== null;

  /** Open the tab context menu at the given position. */
  const openContextMenu = (sessionId: string, x: number, y: number) => {
    setTabContextMenu({ sessionId, position: { x, y } });
  };

  // Group tabs by their groupId, preserving the pane array order.
  // Build a rendering list of: group labels (with collapse toggle) interleaved with tabs.
  type RenderItem =
    | { kind: "group-label"; group: (typeof groups)[0]; tabCount: number }
    | { kind: "tab"; pane: PaneMetadata; globalIndex: number };

  const renderItems: RenderItem[] = [];
  const groupMap = new Map(groups.map((g) => [g.id, g]));
  let lastGroupId: string | null | undefined = undefined; // sentinel to detect group transitions

  panes.forEach((pane, idx) => {
    const gid = pane.groupId;

    // Emit group label when entering a new group
    if (gid && gid !== lastGroupId) {
      const group = groupMap.get(gid);
      if (group) {
        const tabCount = panes.filter((p) => p.groupId === gid).length;
        renderItems.push({ kind: "group-label", group, tabCount });
      }
    }
    lastGroupId = gid;

    // Collapsed group: skip individual tabs (the label shows the count)
    const group = gid ? groupMap.get(gid) : undefined;
    if (group?.isCollapsed) return;

    renderItems.push({ kind: "tab", pane, globalIndex: idx });
  });

  return (
    <div
      data-testid="tab-bar"
      className="flex items-stretch h-9 border-b border-wc-default bg-wc-surface-header shrink-0"
      role="tablist"
    >
      <div
        ref={tabContainerRef}
        className="flex-1 flex items-stretch overflow-x-auto wc-hide-scrollbar"
      >
        {renderItems.map((item) => {
          if (item.kind === "group-label") {
            const { group, tabCount } = item;
            return (
              <button
                key={`group-${group.id}`}
                data-testid={`tab-group-${group.id}`}
                className="flex items-center gap-1 px-2 text-xs shrink-0 border-r border-wc-default text-wc-text-secondary hover:bg-wc-surface-raised transition-colors"
                onClick={() => toggleGroupCollapsed(group.id)}
                title={group.isCollapsed ? `Expand ${group.name}` : `Collapse ${group.name}`}
              >
                <span
                  className="h-2.5 w-2.5 rounded-full shrink-0"
                  style={{ backgroundColor: group.color }}
                />
                {editingGroupId === group.id ? (
                  <input
                    ref={editGroupInputRef}
                    data-testid={`group-rename-input-${group.id}`}
                    className="bg-wc-surface-input text-wc-text-primary text-xs px-1 rounded w-[80px] outline-none ring-1 ring-wc-accent font-medium"
                    value={editGroupName}
                    onChange={(e) => setEditGroupName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === "Enter") commitGroupRename();
                      if (e.key === "Escape") { setEditingGroupId(null); setEditGroupName(""); }
                    }}
                    onBlur={commitGroupRename}
                    onClick={(e) => e.stopPropagation()}
                  />
                ) : (
                  <span className="truncate max-w-[80px] font-medium">{group.name}</span>
                )}
                {group.isCollapsed && (
                  <span className="text-[10px] bg-wc-surface-input rounded px-1">{tabCount}</span>
                )}
                {group.isCollapsed ? (
                  <ChevronRight className="h-3 w-3 shrink-0" />
                ) : (
                  <ChevronDown className="h-3 w-3 shrink-0" />
                )}
              </button>
            );
          }

          const { pane, globalIndex: idx } = item;
          const isActive = pane.sessionId === activePane;
          const isBeingDragged = dragState?.paneId === pane.sessionId;
          const isDropTarget =
            isDragging && !isBeingDragged && dragState?.dropIndex === idx;
          const paneGroup = pane.groupId ? groupMap.get(pane.groupId) : undefined;
          const supportsMessagesView = pane.supportsMessagesView;
          const unreadCount = (() => {
            if (!supportsMessagesView) return 0;
            const session = conversationSessions[pane.sessionId];
            if (!session) return 0;
            return session.events.filter((event) => event.role === "assistant" && event.sequence > session.cursor.lastSeenSequence).length;
          })();

          return (
              <div
                key={pane.sessionId}
                ref={isActive ? activeTabRef : undefined}
                data-testid={`tab-${pane.sessionId}`}
                data-tab-index={idx}
                role="tab"
                tabIndex={0}
                aria-selected={isActive}
                className={cn(
                  "group relative flex items-center gap-1.5 h-full px-3 text-xs shrink-0 border-r border-wc-default transition-colors",
                  "hover:bg-wc-surface-raised focus:outline-none focus-visible:ring-1 focus-visible:ring-wc-accent",
                isActive
                  ? "bg-wc-surface-base text-wc-text-primary font-medium shadow-[inset_0_-2px_0_0_rgb(var(--wc-accent))]"
                  : "bg-wc-surface-header text-wc-text-secondary",
                isBeingDragged && "opacity-40",
                isDropTarget && "ring-2 ring-blue-400/60 ring-inset",
              )}
              style={
                paneGroup
                  ? { borderBottomColor: paneGroup.color, borderBottomWidth: "2px" }
                  : undefined
              }
              onClick={() => {
                // Suppress click if a drag just completed
                if (isDragging) return;
                activatePane(pane.sessionId);
              }}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  if (!isDragging) {
                    activatePane(pane.sessionId);
                  }
                }
              }}
              onContextMenu={(e) => {
                e.preventDefault();
                if (longPressTimer.current) {
                  clearTimeout(longPressTimer.current);
                  longPressTimer.current = null;
                }
                longPressFired.current = true;
                openContextMenu(pane.sessionId, e.clientX, e.clientY);
              }}
              onPointerDown={(e) => {
                // Left-click on mouse: start tracking for drag threshold
                if (e.button === 0 && e.pointerType === "mouse") {
                  dragStartRef.current = {
                    paneId: pane.sessionId,
                    x: e.clientX,
                    y: e.clientY,
                    pointerId: e.pointerId,
                    target: e.currentTarget as HTMLElement,
                  };
                  return;
                }
                // Start long-press timer for touch/pen.
                // Sets longPressReady flag; context menu opens on pointerUp so
                // drag gestures can cancel the long-press.
                if (e.pointerType !== "mouse" && e.button === 0) {
                  longPressFired.current = false;
                  longPressReady.current = false;
                  longPressPaneId.current = pane.sessionId;
                  dragStartRef.current = {
                    paneId: pane.sessionId,
                    x: e.clientX,
                    y: e.clientY,
                    pointerId: e.pointerId,
                    target: e.currentTarget as HTMLElement,
                  };
                  longPressTimer.current = setTimeout(() => {
                    longPressReady.current = true;
                  }, 500);
                }
              }}
              onPointerUp={(e) => {
                if (longPressTimer.current) {
                  clearTimeout(longPressTimer.current);
                  longPressTimer.current = null;
                }
                // Long-press ready + no drag: open context menu on touch-up
                if (longPressReady.current && !isDragging && longPressPaneId.current) {
                  longPressFired.current = true;
                  longPressReady.current = false;
                  openContextMenu(longPressPaneId.current, e.clientX, e.clientY);
                  longPressPaneId.current = null;
                  return;
                }
                longPressReady.current = false;
                longPressPaneId.current = null;
                // Activate tab immediately on pointer-up rather than waiting
                // for onClick, which mobile browsers may delay or suppress
                // when the element is inside a scrollable container.
                if (!longPressFired.current && !isDragging) {
                  activatePane(pane.sessionId);
                }
              }}
              onPointerCancel={() => {
                if (longPressTimer.current) {
                  clearTimeout(longPressTimer.current);
                  longPressTimer.current = null;
                }
                longPressReady.current = false;
                longPressPaneId.current = null;
                dragStartRef.current = null;
              }}
            >
              {/* Color indicator */}
              {pane.headerColor !== "transparent" && (
                <span
                  className="absolute left-0 top-0 bottom-0 w-0.5"
                  style={{ backgroundColor: pane.headerColor }}
                />
              )}

              {/* Tab name (inline editable) */}
              {editingTabId === pane.sessionId ? (
                <input
                  ref={editTabInputRef}
                  data-testid={`tab-rename-input-${pane.sessionId}`}
                  className="bg-wc-surface-input text-wc-text-primary text-xs px-1 rounded w-[100px] outline-none ring-1 ring-wc-accent"
                  value={editTabName}
                  onChange={(e) => setEditTabName(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") commitTabRename();
                    if (e.key === "Escape") { setEditingTabId(null); setEditTabName(""); }
                  }}
                  onBlur={commitTabRename}
                  onClick={(e) => e.stopPropagation()}
                />
              ) : (
                <span className="truncate max-w-[120px]">{pane.name}</span>
              )}

              {unreadCount > 0 && (
                <span className="rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-black">
                  {unreadCount}
                </span>
              )}

              {/* Close button - visible on hover or when active */}
              <button
                type="button"
                className={cn(
                  "flex items-center justify-center h-4 w-4 rounded hover:bg-wc-surface-input",
                  "opacity-0 group-hover:opacity-100 focus:opacity-100",
                  isActive && "opacity-100",
                )}
                onClick={(e) => {
                  e.stopPropagation();
                  onClosePane(pane.sessionId);
                }}
                title="Close tab"
                aria-label={`Close ${pane.name}`}
              >
                <X className="h-3 w-3" />
              </button>
            </div>
          );
        })}
      </div>

      {/* Extra trailing actions (e.g. settings icon on mobile tab mode) */}
      {trailingActions}

      {/* New tab button */}
      <Button
        data-testid="tab-bar-new"
        variant="ghost"
        size="icon"
        className="h-7 w-7 shrink-0 mx-1 self-center"
        disabled={isCreating}
        title={plusButtonBehavior === "launcher" ? "Open launcher (long-press for empty terminal)" : "New terminal (long-press for launcher)"}
        onPointerDown={plusHandlers.onPointerDown}
        onPointerUp={plusHandlers.onPointerUp}
        onPointerCancel={plusHandlers.onPointerCancel}
        onContextMenu={plusHandlers.onContextMenu}
      >
        <Plus className="h-4 w-4" />
      </Button>

      {/* Tab context menu */}
      {tabContextMenu && (() => {
        const pane = panes.find((p) => p.sessionId === tabContextMenu.sessionId);
        if (!pane) return null;
        return (
          <TabContextMenu
            position={tabContextMenu.position}
            sessionId={tabContextMenu.sessionId}
            currentGroupId={pane.groupId}
            groups={groups}
            onRename={() => {
              const p = panes.find((p) => p.sessionId === tabContextMenu.sessionId);
              if (p) startTabRename(p.sessionId, p.name);
            }}
            onCustomize={() => setAppearanceModalPane(tabContextMenu.sessionId)}
            onAddToGroup={(groupId) => {
              setPaneGroup(tabContextMenu.sessionId, groupId);
              syncPaneUpdate(tabContextMenu.sessionId, { group_id: groupId });
            }}
            onRemoveFromGroup={() => {
              setPaneGroup(tabContextMenu.sessionId, null);
              syncPaneUpdate(tabContextMenu.sessionId, { group_id: null });
            }}
            onCreateGroup={async () => {
              const targetSessionId = tabContextMenu.sessionId;
              try {
                const serverGroup = await syncCreateGroup("New Group", "#3b82f6");
                addGroup({
                  id: serverGroup.id,
                  name: serverGroup.name,
                  color: serverGroup.color,
                  isCollapsed: false,
                });
                setPaneGroup(targetSessionId, serverGroup.id);
                syncPaneUpdate(targetSessionId, { group_id: serverGroup.id });
                // Immediately enter rename mode so user can name the group
                startGroupRename(serverGroup.id, serverGroup.name);
              } catch (err) {
                console.error("Failed to create group:", err);
              }
            }}
            onClose={onClosePane}
            onDismiss={() => setTabContextMenu(null)}
          />
        );
      })()}
    </div>
  );
}
