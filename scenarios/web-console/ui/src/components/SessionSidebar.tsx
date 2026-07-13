import { useCallback, useEffect, useRef, useState, type RefObject } from "react";
import { ArrowDownUp, GripVertical, MessageSquareText, Plus, Settings, TerminalSquare, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { paneColorStyle } from "../lib/paneColor";
import type { WorkspaceNavigationItem } from "../lib/workspaceNavigation";
import { useLongPress } from "../hooks/useLongPress";
import { usePressGesture } from "../hooks/usePressGesture";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { useGroupActions } from "../hooks/useGroupActions";
import { Button } from "./ui/button";
import TabContextMenu from "./TabContextMenu";
import GroupContextMenu from "./GroupContextMenu";

const SIDEBAR_LONG_PRESS_MS = 500;
const SIDEBAR_PRESS_MOVE_THRESHOLD = 8;

interface SessionSidebarProps {
  items: WorkspaceNavigationItem[];
  containerRef: RefObject<HTMLElement | null>;
  isMobile: boolean;
  mobileOpen: boolean;
  isCreating?: boolean;
  onCloseMobile: () => void;
  onActivatePane: (sessionId: string) => void;
  onClosePane: (sessionId: string) => void;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  onNewSessionInGroup: (groupId: string) => void;
  onOpenSettings: () => void;
}

function modeLabel(viewMode: string): string {
  return viewMode === "messages" ? "Messages" : "Terminal";
}

export default function SessionSidebar({
  items,
  containerRef,
  isMobile,
  mobileOpen,
  isCreating,
  onCloseMobile,
  onActivatePane,
  onClosePane,
  onNewTerminal,
  onOpenLauncher,
  onNewSessionInGroup,
  onOpenSettings,
}: SessionSidebarProps) {
  const { t } = useTranslation();
  const sidebarRef = useRef<HTMLElement>(null);
  const tabContextMenu = useWorkspaceStore((s) => s.tabContextMenu);
  const setTabContextMenu = useWorkspaceStore((s) => s.setTabContextMenu);
  const groups = useWorkspaceStore((s) => s.groups);
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const toggleGroupCollapsed = useWorkspaceStore((s) => s.toggleGroupCollapsed);
  const setManageGroupsTarget = useWorkspaceStore((s) => s.setManageGroupsTarget);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const sidebarSortMode = useWorkspaceStore((s) => s.sidebarSortMode);
  const setSidebarSortMode = useWorkspaceStore((s) => s.setSidebarSortMode);
  const plusButtonBehavior = useWorkspaceStore((s) => s.plusButtonBehavior);
  const { syncPaneUpdate, syncPaneOrder } = useWorkspaceSync();
  const { removePaneFromGroup } = useGroupActions();
  const [editingPaneId, setEditingPaneId] = useState<string | null>(null);
  const [editingPaneName, setEditingPaneName] = useState("");
  const editingInputRef = useRef<HTMLInputElement>(null);
  // Group context menu state.
  const [groupMenu, setGroupMenu] = useState<{ groupId: string; position: { x: number; y: number } } | null>(null);
  // Manual-mode drag-reorder (driven by an explicit grip handle so it never
  // collides with the row tap / long-press gestures).
  const [dragState, setDragState] = useState<{ paneId: string; dropIndex: number } | null>(null);
  const isManualSort = sidebarSortMode === "manual";

  const { size, isResizing, resizeHandleProps } = useResizablePanel({
    containerRef,
    targetRef: sidebarRef,
    minSize: 240,
    maxSize: 520,
    defaultSize: 300,
    adjacentMinSize: 420,
    handleWidth: 12,
    storageKey: "web-console.sidebar.width.v1",
  });

  const handleSidebarNewTerminal = useCallback(() => {
    onNewTerminal();
    if (isMobile) onCloseMobile();
  }, [isMobile, onCloseMobile, onNewTerminal]);

  const handleSidebarOpenLauncher = useCallback(() => {
    if (isMobile) onCloseMobile();
    onOpenLauncher();
  }, [isMobile, onCloseMobile, onOpenLauncher]);

  const plusHandlers = useLongPress({
    onPress: plusButtonBehavior === "launcher" ? handleSidebarOpenLauncher : handleSidebarNewTerminal,
    onLongPress: plusButtonBehavior === "launcher" ? handleSidebarNewTerminal : handleSidebarOpenLauncher,
  });

  const paneItems = items.filter((item) => item.kind === "pane");
  const totalUnread = paneItems.reduce((sum, item) => sum + item.unreadCount, 0);

  useEffect(() => {
    if (!editingPaneId) return;
    editingInputRef.current?.focus();
    editingInputRef.current?.select();
  }, [editingPaneId]);

  const startRename = useCallback((sessionId: string, currentName: string) => {
    setEditingPaneId(sessionId);
    setEditingPaneName(currentName);
  }, []);

  const commitRename = useCallback(() => {
    if (editingPaneId && editingPaneName.trim()) {
      const trimmed = editingPaneName.trim();
      renamePaneById(editingPaneId, trimmed);
      syncPaneUpdate(editingPaneId, { name: trimmed });
    }
    setEditingPaneId(null);
    setEditingPaneName("");
  }, [editingPaneId, editingPaneName, renamePaneById, syncPaneUpdate]);

  const activate = useCallback((sessionId: string) => {
    onActivatePane(sessionId);
    if (isMobile) onCloseMobile();
  }, [isMobile, onActivatePane, onCloseMobile]);

  const openContextMenu = useCallback((sessionId: string, x: number, y: number) => {
    setTabContextMenu({ sessionId, position: { x, y } });
  }, [setTabContextMenu]);

  // --- Group context menu ---------------------------------------------------
  const openGroupMenu = useCallback((groupId: string, x: number, y: number) => {
    setGroupMenu({ groupId, position: { x, y } });
  }, []);

  const groupPressGesture = usePressGesture<string>({
    longPressMs: SIDEBAR_LONG_PRESS_MS,
    moveThresholdPx: SIDEBAR_PRESS_MOVE_THRESHOLD,
    onTap: (groupId) => toggleGroupCollapsed(groupId),
    onLongPress: (groupId, point) => openGroupMenu(groupId, point.x, point.y),
  });

  const startSidebarDrag = useCallback((paneId: string, index: number, event: React.PointerEvent<HTMLElement>) => {
    if (!isManualSort) return;
    event.currentTarget.setPointerCapture?.(event.pointerId);
    setDragState({ paneId, dropIndex: index });
  }, [isManualSort]);

  // Global pointer tracking while a sidebar drag is active.
  useEffect(() => {
    if (!dragState) return;
    const handleMove = (e: PointerEvent) => {
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const row = el?.closest("[data-pane-index]");
      if (!row) return;
      const idx = Number(row.getAttribute("data-pane-index"));
      if (Number.isFinite(idx)) {
        setDragState((prev) => (prev ? { ...prev, dropIndex: idx } : null));
      }
    };
    const handleUp = () => {
      setDragState((prev) => {
        if (prev) {
          movePaneToIndex(prev.paneId, prev.dropIndex);
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
  }, [dragState, movePaneToIndex, syncPaneOrder]);

  const panePressGesture = usePressGesture<string>({
    longPressMs: SIDEBAR_LONG_PRESS_MS,
    moveThresholdPx: SIDEBAR_PRESS_MOVE_THRESHOLD,
    onTap: (sessionId) => {
      activate(sessionId);
    },
    onLongPress: (sessionId, point) => {
      openContextMenu(sessionId, point.x, point.y);
    },
  });

  useEffect(() => {
    if (!mobileOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onCloseMobile();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [mobileOpen, onCloseMobile]);

  const sidebarContent = (
    <>
      <div className="flex h-11 select-none items-center gap-2 border-b border-wc-default px-3">
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-wc-text-primary">
            {t(strings.sessionSidebar.title)}
          </div>
          <div className="truncate text-[11px] text-wc-text-muted">
            {totalUnread > 0
              ? t(strings.sessionSidebar.unreadSummary, { count: totalUnread })
              : t(strings.sessionSidebar.sessionCount, { count: paneItems.length })}
          </div>
        </div>
        <Button
          data-testid="workspace-sidebar-new"
          variant="ghost"
          size="icon"
          className="h-8 w-8 shrink-0"
          disabled={isCreating}
          title={plusButtonBehavior === "launcher" ? t(strings.floatingToolbar.launcherFirstTitle) : t(strings.floatingToolbar.terminalFirstTitle)}
          onPointerDown={plusHandlers.onPointerDown}
          onPointerUp={plusHandlers.onPointerUp}
          onPointerCancel={plusHandlers.onPointerCancel}
          onContextMenu={plusHandlers.onContextMenu}
        >
          <Plus className="h-4 w-4" />
        </Button>
        <Button
          data-testid="workspace-sidebar-settings"
          variant="ghost"
          size="icon"
          className="h-8 w-8 shrink-0"
          onClick={onOpenSettings}
          title={t(strings.workspace.settingsTitle)}
        >
          <Settings className="h-4 w-4" />
        </Button>
        {isMobile && (
          <Button
            data-testid="workspace-sidebar-close"
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onCloseMobile}
            title={t(strings.sessionSidebar.close)}
          >
            <X className="h-4 w-4" />
          </Button>
        )}
      </div>

      <div className="flex select-none items-center gap-1.5 border-b border-wc-default px-3 py-1.5 text-[11px] text-wc-text-muted">
        <ArrowDownUp className="h-3 w-3 shrink-0" />
        <label htmlFor="sidebar-sort-select" className="shrink-0">
          {t(strings.sessionSidebar.sortLabel)}
        </label>
        <select
          id="sidebar-sort-select"
          data-testid="sidebar-sort-select"
          className="min-w-0 flex-1 rounded bg-wc-surface-input px-1 py-0.5 text-[11px] text-wc-text-secondary outline-none focus:ring-1 focus:ring-wc-accent"
          value={sidebarSortMode}
          onChange={(event) => setSidebarSortMode(event.target.value as typeof sidebarSortMode)}
        >
          <option value="manual">{t(strings.sessionSidebar.sortManual)}</option>
          <option value="name">{t(strings.sessionSidebar.sortName)}</option>
          <option value="activity">{t(strings.sessionSidebar.sortActivity)}</option>
          <option value="unread">{t(strings.sessionSidebar.sortUnread)}</option>
        </select>
      </div>

      <div className="flex-1 select-none overflow-y-auto p-2">
        {items.map((item) => {
          if (item.kind === "group-label") {
            const { group, tabCount } = item;
            return (
              <div
                key={`group-${group.id}`}
                data-testid={`sidebar-group-header-${group.id}`}
                className={cn(
                  "mt-1 flex w-full items-center gap-2 border border-wc-default border-b-0 bg-wc-surface-base/70 px-2 py-1.5 text-xs text-wc-text-secondary hover:bg-wc-surface-raised",
                  group.isCollapsed ? "mb-1 rounded" : "rounded-t",
                )}
                style={{ borderLeftColor: group.color }}
                onContextMenu={(event) => {
                  event.preventDefault();
                  openGroupMenu(group.id, event.clientX, event.clientY);
                }}
              >
                <span className="h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: group.color }} />
                <button
                  type="button"
                  data-testid={`sidebar-group-${group.id}`}
                  className="flex min-w-0 flex-1 items-center gap-2 text-start"
                  {...groupPressGesture.getGestureHandlers(group.id)}
                  onClick={() => {
                    if (groupPressGesture.shouldSuppressClick(group.id)) return;
                    toggleGroupCollapsed(group.id);
                  }}
                  title={group.isCollapsed ? t(strings.tabBar.expandGroup, { name: group.name }) : t(strings.tabBar.collapseGroup, { name: group.name })}
                >
                  <span className="min-w-0 flex-1 truncate font-medium">{group.name}</span>
                </button>
                <span className="shrink-0 rounded bg-wc-surface-input px-1 text-[10px]">{tabCount}</span>
              </div>
            );
          }

          const { pane, activityLabel, previewText, unreadCount, viewMode, isActive, group, groupPosition, globalIndex } = item;
          const isBeingDragged = dragState?.paneId === pane.sessionId;
          const isDropTarget = dragState !== null && !isBeingDragged && dragState.dropIndex === globalIndex;
          // Pane color wins; fall back to the group color when the pane is transparent.
          const accentStyle =
            paneColorStyle(pane.headerColor, "bar") ??
            (group?.color ? { backgroundColor: group.color } : undefined);
          const isEditing = editingPaneId === pane.sessionId;
          const rowBody = (
            <>
              {accentStyle && (
                <span className="mt-1 h-8 w-1.5 shrink-0 rounded-full" style={accentStyle} />
              )}
              <span className="min-w-0 flex-1">
                <span className="flex min-w-0 items-center gap-2">
                  {isEditing ? (
                    <input
                      ref={editingInputRef}
                      data-testid={`sidebar-rename-input-${pane.sessionId}`}
                      className="min-w-0 flex-1 select-text rounded bg-wc-surface-input px-1 text-sm font-medium text-wc-text-primary outline-none ring-1 ring-wc-accent"
                      value={editingPaneName}
                      onChange={(event) => setEditingPaneName(event.target.value)}
                      onKeyDown={(event) => {
                        if (event.key === "Enter") commitRename();
                        if (event.key === "Escape") {
                          setEditingPaneId(null);
                          setEditingPaneName("");
                        }
                        event.stopPropagation();
                      }}
                      onBlur={commitRename}
                      onClick={(event) => event.stopPropagation()}
                    />
                  ) : (
                    <span className="min-w-0 flex-1 truncate text-sm font-medium" title={pane.name}>
                      {pane.name}
                    </span>
                  )}
                  {unreadCount > 0 && (
                    <span className="rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-wc-accent-fg">
                      {unreadCount}
                    </span>
                  )}
                </span>
                <span className="mt-0.5 flex items-center gap-1 text-[11px] text-wc-text-muted">
                  {viewMode === "messages" ? <MessageSquareText className="h-3 w-3" /> : <TerminalSquare className="h-3 w-3" />}
                  <span className="truncate">
                    {[activityLabel, modeLabel(viewMode)].filter(Boolean).join(" · ")}
                  </span>
                </span>
                {previewText && (
                  <span className="mt-1 block truncate text-[11px] text-wc-text-faint">
                    {previewText}
                  </span>
                )}
              </span>
            </>
          );
          return (
            <div
              key={pane.sessionId}
              data-pane-index={globalIndex}
              data-group-id={group?.id}
              className={cn(
                "group relative mb-1 flex w-full items-start rounded border text-start transition-colors",
                group && "border-s bg-wc-surface-base/45",
                groupPosition === "first" && "rounded-t-none",
                groupPosition === "middle" && "rounded-none border-y-transparent",
                groupPosition === "last" && "mb-2 rounded-t-none",
                groupPosition === "single" && "mb-2 rounded-t-none",
                isActive
                  ? "border-wc-accent bg-wc-surface-raised text-wc-text-primary"
                  : "border-transparent text-wc-text-secondary hover:border-wc-default hover:bg-wc-surface-raised",
                isBeingDragged && "opacity-40",
                isDropTarget && "ring-2 ring-wc-accent ring-inset",
              )}
              style={group?.color ? { borderLeftColor: group.color } : undefined}
              onContextMenu={(event) => {
                event.preventDefault();
                setTabContextMenu({ sessionId: pane.sessionId, position: { x: event.clientX, y: event.clientY } });
              }}
            >
              {isEditing ? (
                <div
                  data-testid={`sidebar-session-${pane.sessionId}`}
                  className="flex min-w-0 flex-1 items-start gap-2 rounded px-2 py-2 text-start"
                >
                  {rowBody}
                </div>
              ) : (
                <button
                  type="button"
                  data-testid={`sidebar-session-${pane.sessionId}`}
                  className="flex min-w-0 flex-1 items-start gap-2 rounded px-2 py-2 text-start focus:outline-none focus-visible:ring-1 focus-visible:ring-wc-accent"
                  {...panePressGesture.getGestureHandlers(pane.sessionId)}
                  onClick={() => {
                    if (panePressGesture.shouldSuppressClick(pane.sessionId)) {
                      return;
                    }
                    activate(pane.sessionId);
                  }}
                >
                  {rowBody}
                </button>
              )}
              {/* Hover action cluster — fine pointers only. On touch these
                  actions live in the long-press context menu (Move Up/Down,
                  Close), so nothing is reserved here and the title gets the
                  row's full width at rest. The left-fading gradient masks the
                  title edge behind the controls while they're shown. */}
              <div
                className={cn(
                  "pointer-events-none absolute inset-y-0 end-0 hidden items-center gap-0.5 rounded-e pe-1.5 ps-8 opacity-0 transition-opacity",
                  "bg-gradient-to-l from-wc-surface-raised via-wc-surface-raised to-transparent",
                  "[@media(hover:hover)]:flex group-hover:opacity-100 focus-within:opacity-100",
                )}
              >
                {isManualSort && (
                  <button
                    type="button"
                    data-testid={`sidebar-drag-handle-${pane.sessionId}`}
                    className="pointer-events-auto flex h-6 w-5 shrink-0 cursor-grab touch-none items-center justify-center rounded text-wc-text-faint hover:text-wc-text-secondary active:cursor-grabbing"
                    onPointerDown={(event) => startSidebarDrag(pane.sessionId, globalIndex, event)}
                    title={t(strings.sessionSidebar.reorderAria, { name: pane.name })}
                    aria-label={t(strings.sessionSidebar.reorderAria, { name: pane.name })}
                    aria-roledescription="drag handle"
                  >
                    <GripVertical className="h-3.5 w-3.5" />
                  </button>
                )}
                <button
                  type="button"
                  className="pointer-events-auto flex h-6 w-6 shrink-0 items-center justify-center rounded text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary"
                  onClick={() => onClosePane(pane.sessionId)}
                  title={t(strings.tabBar.closeTabTitle)}
                  aria-label={t(strings.tabBar.closeTabAria, { name: pane.name })}
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </>
  );

  return (
    <>
      {!isMobile && (
        <aside
          ref={sidebarRef}
          data-testid="workspace-sidebar-shell"
          className={cn(
            "wc-chrome-surface wc-chrome-fg relative hidden shrink-0 flex-col border-e border-wc-default ps-[var(--wc-safe-left,0px)] md:flex",
            isResizing && "select-none",
          )}
          style={{ width: size }}
        >
          {sidebarContent}
          <div
            data-testid="workspace-sidebar-resize-handle"
            className="absolute end-[-6px] top-0 z-wc-chrome-raised h-full w-3 cursor-col-resize"
            {...resizeHandleProps}
          >
            <div className="mx-auto h-full w-px bg-transparent transition-colors hover:bg-wc-accent" />
          </div>
        </aside>
      )}

      {isMobile && mobileOpen && (
        <div className="fixed inset-0 z-wc-drawer md:hidden" role="dialog" aria-modal="true" aria-label={t(strings.sessionSidebar.title)}>
          <button
            data-testid="workspace-sidebar-backdrop"
            className="absolute inset-0 bg-black/55"
            onClick={onCloseMobile}
            aria-label={t(strings.sessionSidebar.close)}
          />
          <aside
            data-testid="workspace-sidebar-shell"
            className="wc-chrome-surface wc-chrome-fg absolute inset-y-0 start-0 flex w-[min(22rem,calc(100vw-2rem))] flex-col border-e border-wc-default pt-[var(--wc-safe-top,0px)] ps-[var(--wc-safe-left,0px)] shadow-xl"
          >
            {sidebarContent}
          </aside>
        </div>
      )}

      {tabContextMenu && (() => {
        const paneItem = paneItems.find((item) => item.pane.sessionId === tabContextMenu.sessionId);
        if (!paneItem) return null;
        // Touch-friendly reorder: the drag handle is hover-only, so on touch
        // the menu exposes Move Up/Down. Only meaningful in manual sort, and
        // only when the pane isn't already at that boundary of the manual order.
        const orderedPanes = useWorkspaceStore.getState().panes;
        const orderIdx = orderedPanes.findIndex((p) => p.sessionId === tabContextMenu.sessionId);
        const moveTo = (target: number) => {
          movePaneToIndex(tabContextMenu.sessionId, target);
          const { panes: updated, activePane: active } = useWorkspaceStore.getState();
          syncPaneOrder(updated.map((p) => p.sessionId), active);
        };
        const canMoveUp = isManualSort && orderIdx > 0;
        const canMoveDown = isManualSort && orderIdx >= 0 && orderIdx < orderedPanes.length - 1;
        return (
          <TabContextMenu
            position={tabContextMenu.position}
            sessionId={tabContextMenu.sessionId}
            currentGroupId={paneItem.pane.groupId}
            onRename={() => {
              startRename(paneItem.pane.sessionId, paneItem.pane.name);
              setTabContextMenu(null);
            }}
            onCustomize={() => setAppearanceModalPane(tabContextMenu.sessionId)}
            onRemoveFromGroup={() => removePaneFromGroup(tabContextMenu.sessionId)}
            onManageGroups={() => setManageGroupsTarget({ sessionId: tabContextMenu.sessionId })}
            onMoveUp={canMoveUp ? () => moveTo(orderIdx - 1) : undefined}
            onMoveDown={canMoveDown ? () => moveTo(orderIdx + 1) : undefined}
            onClose={onClosePane}
            onDismiss={() => setTabContextMenu(null)}
          />
        );
      })()}

      {groupMenu && (() => {
        const group = groups.find((g) => g.id === groupMenu.groupId);
        if (!group) return null;
        return (
          <GroupContextMenu
            position={groupMenu.position}
            group={group}
            onNewSession={() => onNewSessionInGroup(group.id)}
            onToggleCollapse={() => toggleGroupCollapsed(group.id)}
            onManageGroups={() => setManageGroupsTarget({ sessionId: null })}
            onDismiss={() => setGroupMenu(null)}
          />
        );
      })()}
    </>
  );
}
