import { memo, useCallback, useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronRight, Plus, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useWorkspaceStore, type PaneMetadata, type RoleMeta } from "../stores/useWorkspaceStore";
import { cn } from "../lib/classnames";
import { paneAccentStyle } from "../lib/paneColor";
import { strings } from "../consts/strings";
import { Button } from "./ui/button";
import { useLongPress } from "../hooks/useLongPress";
import { usePressGesture } from "../hooks/usePressGesture";
import TabContextMenu from "./TabContextMenu";
import GroupContextMenu from "./GroupContextMenu";
import RoleRow from "./RoleRow";
import { GroupPickerOverlay } from "./launcher/GroupPicker";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { useGroupActions } from "../hooks/useGroupActions";
import { getSessionUnreadCount, useConversationStore } from "../stores/useConversationStore";
import { buildWorkspaceNavigationItems } from "../lib/workspaceNavigation";

/**
 * Per-tab unread badge. Subscribes to ONLY its own session's unread count
 * (a primitive), so a new message re-renders just this badge — never the
 * whole tab strip or sibling tabs. This is the isolation the multi-session
 * performance overhaul depends on (Layer 0.1).
 */
const TabUnreadBadge = memo(function TabUnreadBadge({
  sessionId,
  supportsMessagesView,
  manuallyUnread,
}: {
  sessionId: string;
  supportsMessagesView: boolean;
  manuallyUnread: boolean;
}) {
  const unreadCount = useConversationStore((state) =>
    supportsMessagesView ? getSessionUnreadCount(state, sessionId) : 0,
  );
  if (unreadCount > 0) {
    return (
      <span className="rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-wc-accent-fg">
        {unreadCount}
      </span>
    );
  }
  // Manual flag: a dot with no number. It means "come back here", not
  // "N things happened", and a real count always outranks it.
  if (manuallyUnread) {
    return (
      <span
        data-testid={`tab-manual-unread-${sessionId}`}
        className="h-2 w-2 shrink-0 rounded-full bg-wc-accent"
      />
    );
  }
  return null;
});

interface TabBarProps {
  panes: PaneMetadata[];
  activePane: string | null;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  onClosePane: (sessionId: string) => void;
  onDeletePanePermanently: (sessionId: string) => void;
  isCreating?: boolean;
  /** Start a waiting role's command from its chip in the strip. */
  onStartRole: (role: RoleMeta) => void;
  /** Open a waiting role's overflow menu. */
  onOpenRoleMenu: (role: RoleMeta, position: { x: number; y: number }) => void;
  /** Extra action buttons rendered before the plus button (e.g. settings on mobile). */
  trailingActions?: React.ReactNode;
}

function TabBar({
  panes,
  activePane,
  onNewTerminal,
  onOpenLauncher,
  onClosePane,
  onDeletePanePermanently,
  isCreating,
  onStartRole,
  onOpenRoleMenu,
  trailingActions,
}: TabBarProps) {
  const { t } = useTranslation();
  const setActivePane = useWorkspaceStore((s) => s.setActivePane);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const groups = useWorkspaceStore((s) => s.groups);
  const roles = useWorkspaceStore((s) => s.roles);
  const tabContextMenu = useWorkspaceStore((s) => s.tabContextMenu);
  const setTabContextMenu = useWorkspaceStore((s) => s.setTabContextMenu);
  const toggleGroupCollapsed = useWorkspaceStore((s) => s.toggleGroupCollapsed);
  const setManageGroupsOpen = useWorkspaceStore((s) => s.setManageGroupsOpen);
  const setCloseGroupTarget = useWorkspaceStore((s) => s.setCloseGroupTarget);
  // Where the anchored assign picker should sit, and for which session.
  const [assignPicker, setAssignPicker] = useState<{ sessionId: string } | null>(null);
  const { syncPaneMove, syncActivePane, syncPaneUpdate } = useWorkspaceSync();
  const { removePaneFromGroup, assignPaneToGroup, createNamedGroup } = useGroupActions();
  const setPaneManuallyUnread = useWorkspaceStore((s) => s.setPaneManuallyUnread);

  /** Flip a pane's manual unread flag and persist it. */
  const toggleManualUnread = useCallback((sessionId: string) => {
    const pane = useWorkspaceStore.getState().panes.find((p) => p.sessionId === sessionId);
    if (!pane) return;
    const next = !pane.manuallyUnread;
    setPaneManuallyUnread(sessionId, next);
    syncPaneUpdate(sessionId, { manually_unread: next });
  }, [setPaneManuallyUnread, syncPaneUpdate]);

  // Group context menu (long-press / right-click on a group label).
  const [groupMenu, setGroupMenu] = useState<{ groupId: string; position: { x: number; y: number } } | null>(null);

  // Inline rename state for tabs
  const [editingTabId, setEditingTabId] = useState<string | null>(null);
  const [editTabName, setEditTabName] = useState("");
  const editTabInputRef = useRef<HTMLInputElement>(null);

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

  // Auto-focus rename inputs when they appear
  useEffect(() => {
    if (editingTabId && editTabInputRef.current) {
      editTabInputRef.current.focus();
      editTabInputRef.current.select();
    }
  }, [editingTabId]);

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
  const suppressPointerUpActivationRef = useRef(false);
  const mouseActivatedPaneRef = useRef<string | null>(null);

  /** Movement threshold (px) before a pointer-down becomes a drag. */
  const DRAG_THRESHOLD = 5;

  const activatePane = useCallback((sessionId: string) => {
    const clearedUnread = setActivePane(sessionId);
    syncActivePane(useWorkspaceStore.getState().panes.map((p) => p.sessionId), sessionId);
    if (clearedUnread) syncPaneUpdate(sessionId, { manually_unread: false });
  }, [setActivePane, syncActivePane, syncPaneUpdate]);

  /** Open the tab context menu at the given position. */
  const openContextMenu = useCallback((sessionId: string, x: number, y: number) => {
    setTabContextMenu({ sessionId, position: { x, y } });
  }, [setTabContextMenu]);

  const tabPressGesture = usePressGesture<string>({
    longPressMs: 500,
    moveThresholdPx: DRAG_THRESHOLD,
    onTap: (sessionId) => {
      activatePane(sessionId);
    },
    onLongPress: (sessionId, point) => {
      openContextMenu(sessionId, point.x, point.y);
    },
  });

  const openGroupMenu = useCallback((groupId: string, x: number, y: number) => {
    setGroupMenu({ groupId, position: { x, y } });
  }, []);

  const groupPressGesture = usePressGesture<string>({
    longPressMs: 500,
    moveThresholdPx: DRAG_THRESHOLD,
    onTap: (groupId) => { toggleGroupCollapsed(groupId); },
    onLongPress: (groupId, point) => { openGroupMenu(groupId, point.x, point.y); },
  });

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

  // Initiate actual drag once movement exceeds threshold
  const commitDrag = useCallback(
    (paneId: string, target: HTMLElement, pointerId: number) => {
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
          suppressPointerUpActivationRef.current = true;
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
          // Zustand set() is synchronous, so syncPaneMove's getState() sees
          // the new order — and the group/color change a drop can carry.
          movePaneToIndex(prev.paneId, prev.dropIndex);
          syncPaneMove(prev.paneId);
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
  }, [dragState, movePaneToIndex, syncPaneMove, commitDrag]);

  const isDragging = dragState !== null;

  const groupMap = new Map(groups.map((g) => [g.id, g]));
  // Unread badges are rendered per-tab via <TabUnreadBadge> so a new message
  // re-renders only the affected badge — not the whole strip. So the strip
  // structure is built without conversation data here.
  const renderItems = buildWorkspaceNavigationItems({
    panes,
    groups,
    roles,
    activePane,
  });

  return (
    <div
      data-testid="tab-bar"
      className="wc-chrome-surface flex items-stretch h-9 border-b border-wc-default shrink-0 ps-[var(--wc-safe-left,0px)] pe-[var(--wc-safe-right,0px)]"
      role="tablist"
    >
      <div
        ref={tabContainerRef}
        className="flex-1 flex items-stretch overflow-x-auto wc-hide-scrollbar"
      >
        {renderItems.map((item) => {
          if (item.kind === "group-label") {
            const { group, tabCount, waitingCount } = item;
            return (
              <button
                key={`group-${group.id}`}
                data-testid={`tab-group-${group.id}`}
                className="flex items-center gap-1 px-2 text-xs shrink-0 border-r border-wc-default text-wc-text-secondary hover:bg-wc-surface-raised transition-colors"
                {...groupPressGesture.getGestureHandlers(group.id)}
                onClick={() => {
                  if (groupPressGesture.shouldSuppressClick(group.id)) return;
                  toggleGroupCollapsed(group.id);
                }}
                title={group.isCollapsed ? t(strings.tabBar.expandGroup, { name: group.name }) : t(strings.tabBar.collapseGroup, { name: group.name })}
              >
                <span
                  className="h-2.5 w-2.5 rounded-full shrink-0"
                  style={{ backgroundColor: group.color }}
                />
                <span className="truncate max-w-[80px] font-medium">{group.name}</span>
                {group.isCollapsed && (
                  <span className="text-[10px] bg-wc-surface-input rounded px-1">{tabCount}</span>
                )}
                {waitingCount > 0 && (
                  <span
                    data-testid={`tab-group-waiting-${group.id}`}
                    className="rounded border border-dashed border-wc-default px-1 text-[10px] text-wc-text-faint"
                  >
                    {waitingCount}
                  </span>
                )}
                {group.isCollapsed ? (
                  <ChevronRight className="h-3 w-3 shrink-0" />
                ) : (
                  <ChevronDown className="h-3 w-3 shrink-0" />
                )}
              </button>
            );
          }

          // A waiting role appears in the strip as a dashed chip. Clicking it
          // starts the role, which is the same gesture as clicking a tab.
          if (item.kind === "waiting-role") {
            return (
              <RoleRow
                key={`role-${item.role.id}`}
                variant="tab"
                role={item.role}
                group={item.group}
                isLastInGroup={item.isLastInGroup}
                onStart={onStartRole}
                onOpenMenu={onOpenRoleMenu}
              />
            );
          }

          const { pane, globalIndex: idx, group } = item;
          const isActive = item.isActive;
          const isBeingDragged = dragState?.paneId === pane.sessionId;
          const isDropTarget =
            isDragging && !isBeingDragged && dragState?.dropIndex === idx;
          const paneGroup = pane.groupId ? groupMap.get(pane.groupId) : undefined;

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
                if (tabPressGesture.shouldSuppressClick(pane.sessionId)) {
                  return;
                }
                if (mouseActivatedPaneRef.current === pane.sessionId) {
                  mouseActivatedPaneRef.current = null;
                  return;
                }
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
              onContextMenu={tabPressGesture.getGestureHandlers(pane.sessionId).onContextMenu}
              onPointerDown={(e) => {
                suppressPointerUpActivationRef.current = false;
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
                tabPressGesture.getGestureHandlers(pane.sessionId).onPointerDown(e);
              }}
              onPointerUp={(e) => {
                const suppressActivation = suppressPointerUpActivationRef.current;
                suppressPointerUpActivationRef.current = false;
                if (e.pointerType !== "mouse") return;
                // Activate tab immediately on pointer-up rather than waiting
                // for onClick, which mobile browsers may delay or suppress
                // when the element is inside a scrollable container.
                if (!isDragging && !suppressActivation) {
                  mouseActivatedPaneRef.current = pane.sessionId;
                  activatePane(pane.sessionId);
                }
              }}
              onPointerCancel={() => {
                suppressPointerUpActivationRef.current = false;
                tabPressGesture.reset();
                dragStartRef.current = null;
              }}
            >
              {/* Color indicator — the pane's own accent, else its group's. */}
              {(() => {
                const accentStyle = paneAccentStyle(pane.headerColor, group?.color, "bar");
                return accentStyle ? (
                  <span className="absolute left-0 top-0 bottom-0 w-1" style={accentStyle} />
                ) : null;
              })()}

              {/* Tab name (inline editable) */}
              {editingTabId === pane.sessionId ? (
                <input
                  ref={editTabInputRef}
                  data-testid={`tab-rename-input-${pane.sessionId}`}
                  className="bg-wc-surface-input text-wc-text-primary text-xs px-1 rounded w-[100px] outline-none ring-1 ring-wc-accent"
                  value={editTabName}
                  onChange={(e) => { setEditTabName(e.target.value); }}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") commitTabRename();
                    if (e.key === "Escape") { setEditingTabId(null); setEditTabName(""); }
                  }}
                  onBlur={commitTabRename}
                  onClick={(e) => { e.stopPropagation(); }}
                />
              ) : (
                <span className="truncate max-w-[120px]">{pane.name}</span>
              )}

              <TabUnreadBadge sessionId={pane.sessionId} supportsMessagesView={pane.supportsMessagesView} manuallyUnread={pane.manuallyUnread} />

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
                title={t(strings.tabBar.closeTabTitle)}
                aria-label={t(strings.tabBar.closeTabAria, { name: pane.name })}
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
        title={plusButtonBehavior === "launcher" ? t(strings.floatingToolbar.launcherFirstTitle) : t(strings.floatingToolbar.terminalFirstTitle)}
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
            isManuallyUnread={pane.manuallyUnread}
            onToggleManuallyUnread={() => { toggleManualUnread(pane.sessionId); }}
            onRename={() => {
              const p = panes.find((p) => p.sessionId === tabContextMenu.sessionId);
              if (p) startTabRename(p.sessionId, p.name);
            }}
            onCustomize={() => { setAppearanceModalPane(tabContextMenu.sessionId); }}
            onRemoveFromGroup={() => { removePaneFromGroup(tabContextMenu.sessionId); }}
            onAssignToGroup={() => {
              setAssignPicker({ sessionId: tabContextMenu.sessionId });
              setTabContextMenu(null);
            }}
            onClose={onClosePane}
            onDeletePermanently={onDeletePanePermanently}
            onDismiss={() => { setTabContextMenu(null); }}
          />
        );
      })()}

      {/* Group context menu */}
      {groupMenu && (() => {
        const group = groups.find((g) => g.id === groupMenu.groupId);
        if (!group) return null;
        return (
          <GroupContextMenu
            position={groupMenu.position}
            group={group}
            onToggleCollapse={() => { toggleGroupCollapsed(group.id); }}
            onManageGroups={() => { setManageGroupsOpen(true); }}
            onCloseGroup={() => { setCloseGroupTarget(group.id); }}
            onDismiss={() => { setGroupMenu(null); }}
          />
        );
      })()}
      {assignPicker && (() => {
        const pane = panes.find((p) => p.sessionId === assignPicker.sessionId);
        return (
          <GroupPickerOverlay
            open
            onClose={() => { setAssignPicker(null); }}
            groups={groups}
            value={pane?.groupId ?? null}
            onChange={(groupId) => {
              if (groupId) assignPaneToGroup(assignPicker.sessionId, groupId);
              else removePaneFromGroup(assignPicker.sessionId);
              setAssignPicker(null);
            }}
            onCreate={(name) => {
              // Server-first, then assign: the group must exist before a pane
              // can point at it.
              void createNamedGroup(name).then((group) => {
                assignPaneToGroup(assignPicker.sessionId, group.id);
                setAssignPicker(null);
              });
            }}
          />
        );
      })()}
    </div>
  );
}

// Memoized so a Workspace re-render (e.g. a conversation event landing in the
// store) does NOT re-render the whole tab strip. Unread badges update through
// their own per-tab subscriptions; everything else here is driven by the
// workspace store / props. Requires the call site to pass stable props.
export default memo(TabBar);
