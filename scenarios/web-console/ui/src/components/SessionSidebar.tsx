import { useCallback, useEffect, useRef, useState, type RefObject } from "react";
import { Archive, ArrowDownUp, ArrowLeft, ChevronRight, Circle, GripVertical, MessageSquareText, Pencil, Plus, Search, Settings, TerminalSquare, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import { paneAccentStyle } from "../lib/paneColor";
import type { OriginBucketNavigation } from "../lib/workspaceNavigation";
import type { PaneMetadata, RoleMeta, SidebarOriginTab } from "../stores/useWorkspaceStore";
import { SwipeActions, type SwipeAction } from "@vrooli/react-component-library/SwipeActions";
import { useLongPress } from "../hooks/useLongPress";
import { usePressGesture } from "../hooks/usePressGesture";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { useGroupActions } from "../hooks/useGroupActions";
// Unversioned import: this scenario tracks the library's latest release of
// the shell rather than pinning a major. web-console is the shell's primary
// consumer and is still greenfield, so taking each release as it lands is
// cheaper than discovering the drift at the next deliberate upgrade.
import { SidebarShell } from "@vrooli/react-component-library/SidebarShell";
import { Button } from "./ui/button";
import TabContextMenu from "./TabContextMenu";
import GroupContextMenu from "./GroupContextMenu";
import RoleRow from "./RoleRow";
import { GroupPickerOverlay } from "./launcher/GroupPicker";
import { listArchivedSessions, type ArchivedSession } from "../api/sessions";
import { formatRelativeTime } from "./MessageJumpList.helpers";

const SIDEBAR_LONG_PRESS_MS = 500;
const SIDEBAR_PRESS_MOVE_THRESHOLD = 8;

interface SessionSidebarProps {
  /** Session navigation partitioned into origin buckets (UI-owned / Programmatic
   *  / Remote), non-empty buckets only, in display order. A single "ui" bucket
   *  means only UI-origin sessions exist — the tab strip stays unmounted and the
   *  list renders exactly as it did before origin tabs. */
  buckets: OriginBucketNavigation[];
  containerRef: RefObject<HTMLElement | null>;
  isMobile: boolean;
  mobileOpen: boolean;
  isCreating?: boolean;
  onCloseMobile: () => void;
  /** Opens the drawer from an edge drag. Without it the gesture stays off. */
  onOpenMobile?: () => void;
  onActivatePane: (sessionId: string) => void;
  onClosePane: (sessionId: string) => void;
  onDeletePanePermanently: (sessionId: string) => void;
  onNewTerminal: () => void;
  onOpenLauncher: () => void;
  onNewSessionInGroup: (groupId: string) => void;
  /** Start a waiting role's command. The row becomes a pane once it exists. */
  onStartRole: (role: RoleMeta) => void;
  /** Open the handoff composer aimed at a waiting role. */
  onHandoffToRole: (role: RoleMeta) => void;
  /** Open a waiting role's overflow menu. */
  onOpenRoleMenu: (role: RoleMeta, position: { x: number; y: number }) => void;
  onOpenSettings: () => void;
  onOpenArchiveDrawer?: (sessionId?: string) => void;
}

function modeLabel(viewMode: string): string {
  return viewMode === "messages" ? "Messages" : "Terminal";
}

const ORIGIN_TAB_LABEL = {
  ui: strings.sessionSidebar.originTabUi,
  programmatic: strings.sessionSidebar.originTabProgrammatic,
  remote: strings.sessionSidebar.originTabRemote,
} as const satisfies Record<SidebarOriginTab, string>;

const ARCHIVE_STATE_LABEL = {
  reopenable: strings.sessionSidebar.archiveStateReopenable,
  read_only: strings.sessionSidebar.archiveStateReadOnly,
  nothing_to_restore: strings.sessionSidebar.archiveStateNothing,
} as const;

export default function SessionSidebar({
  buckets,
  containerRef,
  isMobile,
  mobileOpen,
  isCreating,
  onCloseMobile,
  onOpenMobile,
  onActivatePane,
  onClosePane,
  onDeletePanePermanently,
  onNewTerminal,
  onOpenLauncher,
  onNewSessionInGroup,
  onStartRole,
  onHandoffToRole,
  onOpenRoleMenu,
  onOpenSettings,
  onOpenArchiveDrawer,
}: SessionSidebarProps) {
  const { t } = useTranslation();
  const sidebarRef = useRef<HTMLDivElement>(null);
  const tabContextMenu = useWorkspaceStore((s) => s.tabContextMenu);
  const setTabContextMenu = useWorkspaceStore((s) => s.setTabContextMenu);
  const groups = useWorkspaceStore((s) => s.groups);

  // A handoff needs somewhere to go. Offering the control on a role whose
  // group holds nothing else would open a composer with no targets.
  const groupHasOtherMembers = useCallback((role: RoleMeta) => {
    const state = useWorkspaceStore.getState();
    const panesInGroup = state.panes.filter((p) => p.groupId === role.groupId).length;
    const rolesInGroup = state.roles.filter((r) => r.groupId === role.groupId).length;
    return panesInGroup + rolesInGroup > 1;
  }, []);
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const movePaneToIndex = useWorkspaceStore((s) => s.movePaneToIndex);
  const toggleGroupCollapsed = useWorkspaceStore((s) => s.toggleGroupCollapsed);
  const setManageGroupsOpen = useWorkspaceStore((s) => s.setManageGroupsOpen);
  const setCloseGroupTarget = useWorkspaceStore((s) => s.setCloseGroupTarget);
  // Which session the group overlay is open for, if any.
  const [assignPicker, setAssignPicker] = useState<{ sessionId: string } | null>(null);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const sidebarSortMode = useWorkspaceStore((s) => s.sidebarSortMode);
  const setSidebarSortMode = useWorkspaceStore((s) => s.setSidebarSortMode);
  const sidebarOriginTab = useWorkspaceStore((s) => s.sidebarOriginTab);
  const setSidebarOriginTab = useWorkspaceStore((s) => s.setSidebarOriginTab);
  const sidebarView = useWorkspaceStore((s) => s.sidebarView);
  const setSidebarView = useWorkspaceStore((s) => s.setSidebarView);
  const plusButtonBehavior = useWorkspaceStore((s) => s.plusButtonBehavior);
	const viewerCounts = useWorkspaceStore((s) => s.viewerCounts);
  const { syncPaneUpdate, syncPaneMove } = useWorkspaceSync();
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
  // Only one row rests open at a time; opening another closes the last.
  const [swipeOpenPaneId, setSwipeOpenPaneId] = useState<string | null>(null);
  /**
   * The swipe track, drawn from the same set the long-press menu offers.
   *
   * Three at most: the track stops being readable past that, and every entry
   * here must also be reachable without the gesture — a swipe is an accelerator
   * over the menu, never the only route to an action. Ordered nearest-first, so
   * the cheapest reversible action arms soonest and Close sits furthest away.
   *
   * Close is not guarded by a confirmation. It archives the session, and the
   * archive view reopens it, so the destructive-looking action is already
   * reversible through a first-class path; permanent deletion stays menu-only.
   */
  const swipeActionsFor = useCallback(
    (pane: PaneMetadata): SwipeAction[] => [
      {
        id: "unread",
        label: t(
          pane.manuallyUnread
            ? strings.sessionSidebar.swipeMarkRead
            : strings.sessionSidebar.swipeMarkUnread,
        ),
        icon: <Circle className="h-4 w-4" aria-hidden />,
        onSelect: () => { toggleManualUnread(pane.sessionId); },
      },
      {
        id: "rename",
        label: t(strings.tabContextMenu.rename),
        icon: <Pencil className="h-4 w-4" aria-hidden />,
        onSelect: () => {
          setEditingPaneId(pane.sessionId);
          setEditingPaneName(pane.name);
        },
      },
      {
        id: "close",
        label: t(strings.tabContextMenu.closeTab),
        icon: <X className="h-4 w-4" aria-hidden />,
        tone: "destructive",
        onSelect: () => { onClosePane(pane.sessionId); },
      },
    ],
    [onClosePane, t, toggleManualUnread],
  );

  const [editingPaneId, setEditingPaneId] = useState<string | null>(null);
  const [editingPaneName, setEditingPaneName] = useState("");
  const editingInputRef = useRef<HTMLInputElement>(null);
  // Group context menu state.
  const [groupMenu, setGroupMenu] = useState<{ groupId: string; position: { x: number; y: number } } | null>(null);
  // Manual-mode drag-reorder (driven by an explicit grip handle so it never
  // collides with the row tap / long-press gestures).
  const [dragState, setDragState] = useState<{ paneId: string; dropIndex: number } | null>(null);
  const [archivedSessions, setArchivedSessions] = useState<ArchivedSession[]>([]);
  const [archivedTotal, setArchivedTotal] = useState(0);
  const [archiveFilter, setArchiveFilter] = useState("");
  const [archiveLoading, setArchiveLoading] = useState(true);
  const isManualSort = sidebarSortMode === "manual";

  const refreshArchive = useCallback(async () => {
    try {
      const result = await listArchivedSessions();
      setArchivedSessions(result.sessions);
      setArchivedTotal(result.total);
    } catch {
      // The live session list remains usable when the archive endpoint is
      // temporarily unavailable (startup/reconnect). A later archive event or
      // remount retries without turning sidebar navigation into an error state.
    } finally {
      setArchiveLoading(false);
    }
  }, []);

  useEffect(() => {
    void refreshArchive();
    window.addEventListener("web-console:archive-changed", refreshArchive);
    return () => { window.removeEventListener("web-console:archive-changed", refreshArchive); };
  }, [refreshArchive]);

  const visibleArchived = archivedSessions
    .slice(0, 20)
    .filter((session) => !session.awaiting_recovery)
    .filter((session) => session.pane_name.toLocaleLowerCase().includes(archiveFilter.trim().toLocaleLowerCase()));

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

  // Origin tabs appear only when a non-UI-origin session exists; with just UI
  // sessions `buckets` holds a single "ui" entry and the list renders exactly
  // as it did before origin tabs. `activeBucket` resolves the persisted tab
  // against the buckets actually present, so a tab whose bucket has emptied
  // falls back to the first present bucket without mutating the persisted
  // choice (it stays valid for when that origin reappears).
  const showOriginTabs = buckets.some((bucket) => bucket.bucket !== "ui");
  const activeBucket = buckets.find((bucket) => bucket.bucket === sidebarOriginTab) ?? buckets[0];
  const items = activeBucket?.items ?? [];

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
    onTap: (groupId) => { toggleGroupCollapsed(groupId); },
    onLongPress: (groupId, point) => { openGroupMenu(groupId, point.x, point.y); },
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
  }, [dragState, movePaneToIndex, syncPaneMove]);

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
          className="h-11 w-11 shrink-0 md:h-8 md:w-8"
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

      {sidebarView === "list" && showOriginTabs && (
        <div
          role="tablist"
          aria-label={t(strings.sessionSidebar.originTabsAria)}
          data-testid="sidebar-origin-tabs"
          className="flex select-none items-center gap-1 border-b border-wc-default px-2 py-1.5"
        >
          {buckets.map((bucket) => {
            const count = bucket.items.reduce((sum, item) => sum + (item.kind === "pane" ? 1 : 0), 0);
            const isActiveTab = bucket.bucket === activeBucket?.bucket;
            return (
              <button
                key={bucket.bucket}
                type="button"
                role="tab"
                aria-selected={isActiveTab}
                data-testid={`sidebar-origin-tab-${bucket.bucket}`}
                className={cn(
                  "flex min-w-0 flex-1 items-center justify-center gap-1 rounded px-2 py-1 text-[11px] font-medium transition-colors",
                  isActiveTab
                    ? "bg-wc-surface-raised text-wc-text-primary"
                    : "text-wc-text-muted hover:bg-wc-surface-raised/60 hover:text-wc-text-secondary",
                )}
                onClick={() => { setSidebarOriginTab(bucket.bucket); }}
              >
                <span className="truncate">{t(ORIGIN_TAB_LABEL[bucket.bucket])}</span>
                <span className="shrink-0 rounded bg-wc-surface-input px-1 text-[10px]">{count}</span>
              </button>
            );
          })}
        </div>
      )}

      {sidebarView === "list" && <div className="flex select-none items-center gap-1.5 border-b border-wc-default px-3 py-1.5 text-[11px] text-wc-text-muted">
        <ArrowDownUp className="h-3 w-3 shrink-0" />
        <label htmlFor="sidebar-sort-select" className="shrink-0">
          {t(strings.sessionSidebar.sortLabel)}
        </label>
        <select
          id="sidebar-sort-select"
          data-testid="sidebar-sort-select"
          className="min-w-0 flex-1 rounded bg-wc-surface-input px-1 py-0.5 text-[11px] text-wc-text-secondary outline-none focus:ring-1 focus:ring-wc-accent"
          value={sidebarSortMode}
          onChange={(event) => { setSidebarSortMode(event.target.value as typeof sidebarSortMode); }}
        >
          <option value="manual">{t(strings.sessionSidebar.sortManual)}</option>
          <option value="name">{t(strings.sessionSidebar.sortName)}</option>
          <option value="activity">{t(strings.sessionSidebar.sortActivity)}</option>
          <option value="unread">{t(strings.sessionSidebar.sortUnread)}</option>
        </select>
      </div>}

      {sidebarView === "list" ? <div className="flex-1 select-none overflow-y-auto p-2">
        {items.map((item) => {
          if (item.kind === "group-label") {
            const { group, tabCount, waitingCount } = item;
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
                {waitingCount > 0 && (
                  <span
                    data-testid={`sidebar-group-waiting-${group.id}`}
                    className="shrink-0 rounded border border-dashed border-wc-default px-1 text-[10px] text-wc-text-faint"
                    title={t(strings.roles.waitingCount, { count: waitingCount })}
                  >
                    {t(strings.roles.waitingCount, { count: waitingCount })}
                  </span>
                )}
                <span className="shrink-0 rounded bg-wc-surface-input px-1 text-[10px]">{tabCount}</span>
              </div>
            );
          }

          // A waiting role is a member of its group with no pane behind it.
          // It renders between the group's running sessions and the next
          // group, so the block reads as "running here, then not yet".
          if (item.kind === "waiting-role") {
            return (
              <RoleRow
                key={`role-${item.role.id}`}
                role={item.role}
                group={item.group}
                isLastInGroup={item.isLastInGroup}
                onStart={onStartRole}
                onHandoff={groupHasOtherMembers(item.role) ? onHandoffToRole : undefined}
                onOpenMenu={onOpenRoleMenu}
              />
            );
          }

          const { pane, activityLabel, previewText, unreadCount, viewMode, isActive, group, groupPosition, globalIndex } = item;
          const isBeingDragged = dragState?.paneId === pane.sessionId;
          const isDropTarget = dragState !== null && !isBeingDragged && dragState.dropIndex === globalIndex;
          const accentStyle = paneAccentStyle(pane.headerColor, group?.color, "bar");
          const isEditing = editingPaneId === pane.sessionId;
		  const viewerCount = viewerCounts[pane.sessionId] ?? 1;
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
                      onChange={(event) => { setEditingPaneName(event.target.value); }}
                      onKeyDown={(event) => {
                        if (event.key === "Enter") commitRename();
                        if (event.key === "Escape") {
                          setEditingPaneId(null);
                          setEditingPaneName("");
                        }
                        event.stopPropagation();
                      }}
                      onBlur={commitRename}
                      onClick={(event) => { event.stopPropagation(); }}
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
                  {/* Manual flag. Suppressed when real unread messages are
                      already shown — the count is strictly more informative,
                      and two badges on one row reads as two separate alerts. */}
                  {unreadCount === 0 && pane.manuallyUnread && (
                    <span
                      data-testid={`sidebar-manual-unread-${pane.sessionId}`}
                      className="h-2 w-2 shrink-0 rounded-full bg-wc-accent"
                      role="img"
                      aria-label={t(strings.sessionSidebar.manuallyUnreadAria, { name: pane.name })}
                    />
                  )}
				  {viewerCount > 1 && <span data-testid={`sidebar-viewers-${pane.sessionId}`} className="rounded-full border border-wc-default px-1.5 py-0.5 text-[10px]">{viewerCount}</span>}
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
          const rowNode = (
            <div
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
                    onPointerDown={(event) => { startSidebarDrag(pane.sessionId, globalIndex, event); }}
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
                  onClick={() => { onClosePane(pane.sessionId); }}
                  title={t(strings.tabBar.closeTabTitle)}
                  aria-label={t(strings.tabBar.closeTabAria, { name: pane.name })}
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          );

          // Touch only. On a fine pointer the hover cluster already offers
          // these, and a drag there would compete with text selection.
          if (!isMobile) return <div key={pane.sessionId}>{rowNode}</div>;

          return (
            <SwipeActions
              key={pane.sessionId}
              testId={`sidebar-swipe-${pane.sessionId}`}
              // The wrapper clips the action track, so its corners must match
              // the row's. A grouped row squares off the edges it shares with
              // its neighbours, and a flat `rounded` here would let the track
              // show through the corners the row does not actually round.
              actions={swipeActionsFor(pane)}
              label={t(strings.sessionSidebar.swipeActionsAria, { name: pane.name })}
              open={swipeOpenPaneId === pane.sessionId}
              onOpenChange={(next) => {
                setSwipeOpenPaneId(next ? pane.sessionId : null);
              }}
              className={cn(
                "rounded",
                groupPosition === "first" && "rounded-t-none",
                groupPosition === "middle" && "rounded-none",
                (groupPosition === "last" || groupPosition === "single") && "rounded-t-none",
              )}
            >
              {rowNode}
            </SwipeActions>
          );
        })}
      </div> : (
        <div data-testid="sidebar-archive-view" className="flex min-h-0 flex-1 flex-col">
          <div className="border-b border-wc-default p-2">
            <button
              type="button"
              data-testid="sidebar-archive-back"
              className="mb-2 flex items-center gap-1 text-xs text-wc-text-secondary hover:text-wc-text-primary"
              onClick={() => { setSidebarView("list"); }}
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              {t(strings.sessionSidebar.archiveBack)}
            </button>
            <label className="flex items-center gap-2 rounded border border-wc-default bg-wc-surface-input px-2 py-1.5">
              <Search className="h-3.5 w-3.5 text-wc-text-muted" />
              <input
                data-testid="sidebar-archive-filter"
                className="min-w-0 flex-1 bg-transparent text-sm text-wc-text-primary outline-none"
                value={archiveFilter}
                onChange={(event) => { setArchiveFilter(event.target.value); }}
                placeholder={t(strings.sessionSidebar.archiveFilterPlaceholder)}
              />
            </label>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            {archiveLoading && <p className="p-2 text-xs text-wc-text-muted">{t(strings.sessionSidebar.archiveLoading)}</p>}
            {!archiveLoading && visibleArchived.length === 0 && (
              <p className="p-2 text-xs text-wc-text-muted">{t(strings.sessionSidebar.archiveEmpty)}</p>
            )}
            {visibleArchived.map((session) => (
              <button
                key={session.id}
                type="button"
                data-testid={`sidebar-archive-session-${session.id}`}
                onClick={() => onOpenArchiveDrawer?.(session.id)}
                className="group mb-1.5 block w-full overflow-hidden rounded-lg border border-wc-default bg-wc-surface px-2.5 py-2 text-start shadow-sm transition hover:border-wc-accent/40 hover:bg-wc-surface-raised hover:shadow-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent/60"
              >
                <div className="flex items-start gap-2.5">
                  <span className="mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-wc-default bg-wc-surface-input text-[10px] font-semibold uppercase text-wc-text-secondary group-hover:border-wc-accent/30 group-hover:text-wc-text-primary">
                    {session.agent_type === "none" ? <TerminalSquare className="h-3.5 w-3.5" /> : session.agent_type.slice(0, 2)}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="flex items-center justify-between gap-2">
                      <span className="truncate text-sm font-medium text-wc-text-primary">{session.pane_name}</span>
                      <ChevronRight className="h-3.5 w-3.5 shrink-0 text-wc-text-faint transition-transform group-hover:translate-x-0.5 group-hover:text-wc-text-secondary" />
                    </span>
                    <span className="mt-1 flex min-w-0 items-center gap-1.5 text-[10px] text-wc-text-muted">
                      <span className="truncate capitalize">{session.agent_type === "none" ? "shell" : session.agent_type}</span>
                      <span aria-hidden="true">·</span>
                      <span className="shrink-0">{t(strings.sessionSidebar.archiveMessages, { count: session.message_count })}</span>
                    </span>
                  </span>
                </div>
                <div className="mt-2 flex items-center justify-between gap-2 border-t border-wc-default/50 pt-1.5">
                  <span className="truncate rounded-full bg-wc-surface-input px-2 py-0.5 text-[9px] font-medium text-wc-text-muted">{t(ARCHIVE_STATE_LABEL[session.restore_state])}</span>
                  <span className="shrink-0 text-[10px] text-wc-text-faint">{formatRelativeTime(session.archived_at)}</span>
                </div>
              </button>
            ))}
            <button
              type="button"
              data-testid="sidebar-archive-search-all"
              className="mt-2 w-full rounded border border-dashed border-wc-default px-3 py-2 text-xs text-wc-text-secondary hover:bg-wc-surface-raised"
              onClick={() => onOpenArchiveDrawer?.()}
            >
              {t(strings.sessionSidebar.archiveSearchAll, { count: archivedTotal })}
            </button>
          </div>
    </div>
      )}

      <button
        type="button"
        data-testid="sidebar-archive-footer"
        className="flex shrink-0 items-center gap-2 border-t border-wc-default px-3 py-2 text-sm text-wc-text-secondary hover:bg-wc-surface-raised hover:text-wc-text-primary"
        onClick={() => { setSidebarView("archive"); }}
      >
        <Archive className="h-4 w-4" />
        <span className="flex-1 text-start">{t(strings.sessionSidebar.archive)}</span>
        <span className="rounded bg-wc-surface-input px-1.5 text-xs">{archivedTotal}</span>
      </button>
    </>
  );

  return (
    <>
      {/* One shell for both the desktop panel and the mobile drawer. The
          drawer chrome stays in `sidebarContent` — its close control lives in
          the topbar row rather than in a header of its own — so the shell is
          asked for positioning, backdrop, dialog semantics and Escape only. */}
      <SidebarShell
        ref={sidebarRef}
        mode={isMobile ? "overlay" : "persistent"}
        mobileChrome="content"
        testId="workspace-sidebar"
        mobileOpen={mobileOpen}
        onMobileClose={onCloseMobile}
        onMobileOpen={onOpenMobile}
        mobileLabel={t(strings.sessionSidebar.title)}
        // The uncovered strip is the tap-to-dismiss target, so it has to be
        // wide enough to read as one — the previous 2rem gutter looked like a
        // rendering artifact rather than a way out.
        mobileWidth="min(22rem, calc(100% - 3.5rem))"
        desktopLabel={t(strings.sessionSidebar.title)}
        closeLabel={t(strings.sessionSidebar.close)}
        className={cn(
          "wc-chrome-surface wc-chrome-fg flex flex-col border-e border-wc-default pt-[var(--wc-safe-top,0px)] ps-[var(--wc-safe-left,0px)]",
          isMobile ? "z-wc-drawer shadow-xl" : "shrink-0",
        )}
        contentClassName="flex flex-col"
        backdropClassName="fixed inset-0 z-wc-drawer bg-black/55"
        resizable={
          isMobile
            ? undefined
            : {
                containerRef,
                min: 240,
                max: 520,
                defaultSize: 300,
                adjacentMin: 420,
                storageKey: "web-console.sidebar.width.v1",
                panelName: t(strings.sessionSidebar.title),
              }
        }
      >
        {sidebarContent}
      </SidebarShell>

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
          syncPaneMove(tabContextMenu.sessionId);
        };
        const canMoveUp = isManualSort && orderIdx > 0;
        const canMoveDown = isManualSort && orderIdx >= 0 && orderIdx < orderedPanes.length - 1;
        return (
          <TabContextMenu
            position={tabContextMenu.position}
            sessionId={tabContextMenu.sessionId}
            currentGroupId={paneItem.pane.groupId}
            isManuallyUnread={paneItem.pane.manuallyUnread}
            onToggleManuallyUnread={() => { toggleManualUnread(tabContextMenu.sessionId); }}
            onRename={() => {
              startRename(paneItem.pane.sessionId, paneItem.pane.name);
              setTabContextMenu(null);
            }}
            onCustomize={() => { setAppearanceModalPane(tabContextMenu.sessionId); }}
            onRemoveFromGroup={() => { removePaneFromGroup(tabContextMenu.sessionId); }}
            onAssignToGroup={() => {
              setAssignPicker({ sessionId: tabContextMenu.sessionId });
              setTabContextMenu(null);
            }}
            onMoveUp={canMoveUp ? () => { moveTo(orderIdx - 1); } : undefined}
            onMoveDown={canMoveDown ? () => { moveTo(orderIdx + 1); } : undefined}
            onClose={onClosePane}
            onDeletePermanently={onDeletePanePermanently}
            onDismiss={() => { setTabContextMenu(null); }}
          />
        );
      })()}

      {/* The group overlay lives at the top level, NOT inside a sidebar view
          branch: it is opened from the session menu, which is reachable from
          every view, and a picker mounted inside one branch simply never
          appeared in the other. */}
      {assignPicker && (() => {
        const pane = useWorkspaceStore.getState().panes.find((p) => p.sessionId === assignPicker.sessionId);
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

      {groupMenu && (() => {
        const group = groups.find((g) => g.id === groupMenu.groupId);
        if (!group) return null;
        return (
          <GroupContextMenu
            position={groupMenu.position}
            group={group}
            onNewSession={() => { onNewSessionInGroup(group.id); }}
            onToggleCollapse={() => { toggleGroupCollapsed(group.id); }}
            onManageGroups={() => { setManageGroupsOpen(true); }}
            onCloseGroup={() => { setCloseGroupTarget(group.id); }}
            onDismiss={() => { setGroupMenu(null); }}
          />
        );
      })()}
    </>
  );
}
