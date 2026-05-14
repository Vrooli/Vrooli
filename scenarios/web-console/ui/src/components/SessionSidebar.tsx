import { useCallback, useEffect, useRef, useState, type RefObject } from "react";
import { MessageSquareText, Plus, Settings, TerminalSquare, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../consts/strings";
import { cn } from "../lib/classnames";
import type { WorkspaceNavigationItem } from "../lib/workspaceNavigation";
import { useLongPress } from "../hooks/useLongPress";
import { useResizablePanel } from "../hooks/useResizablePanel";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { Button } from "./ui/button";
import TabContextMenu from "./TabContextMenu";

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
  onOpenSettings,
}: SessionSidebarProps) {
  const { t } = useTranslation();
  const sidebarRef = useRef<HTMLElement>(null);
  const tabContextMenu = useWorkspaceStore((s) => s.tabContextMenu);
  const setTabContextMenu = useWorkspaceStore((s) => s.setTabContextMenu);
  const groups = useWorkspaceStore((s) => s.groups);
  const addGroup = useWorkspaceStore((s) => s.addGroup);
  const renamePaneById = useWorkspaceStore((s) => s.renamePaneById);
  const setPaneGroup = useWorkspaceStore((s) => s.setPaneGroup);
  const toggleGroupCollapsed = useWorkspaceStore((s) => s.toggleGroupCollapsed);
  const setAppearanceModalPane = useWorkspaceStore((s) => s.setAppearanceModalPane);
  const plusButtonBehavior = useWorkspaceStore((s) => s.plusButtonBehavior);
  const { syncCreateGroup, syncPaneUpdate } = useWorkspaceSync();
  const [editingPaneId, setEditingPaneId] = useState<string | null>(null);
  const [editingPaneName, setEditingPaneName] = useState("");
  const editingInputRef = useRef<HTMLInputElement>(null);

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

  const plusHandlers = useLongPress({
    onPress: plusButtonBehavior === "launcher" ? onOpenLauncher : onNewTerminal,
    onLongPress: plusButtonBehavior === "launcher" ? onNewTerminal : onOpenLauncher,
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
      <div className="flex h-11 items-center gap-2 border-b border-wc-default px-3">
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

      <div className="flex-1 overflow-y-auto p-2">
        {items.map((item) => {
          if (item.kind === "group-label") {
            const { group, tabCount } = item;
            return (
              <button
                key={`group-${group.id}`}
                data-testid={`sidebar-group-${group.id}`}
                className="mb-1 flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-xs text-wc-text-secondary hover:bg-wc-surface-raised"
                onClick={() => toggleGroupCollapsed(group.id)}
                title={group.isCollapsed ? t(strings.tabBar.expandGroup, { name: group.name }) : t(strings.tabBar.collapseGroup, { name: group.name })}
              >
                <span className="h-2.5 w-2.5 rounded-full" style={{ backgroundColor: group.color }} />
                <span className="min-w-0 flex-1 truncate font-medium">{group.name}</span>
                <span className="rounded bg-wc-surface-input px-1 text-[10px]">{tabCount}</span>
              </button>
            );
          }

          const { pane, activityLabel, previewText, unreadCount, viewMode, isActive, group } = item;
          const accentColor = pane.headerColor !== "transparent" ? pane.headerColor : group?.color;
          return (
            <div
              key={pane.sessionId}
              role="button"
              tabIndex={0}
              data-testid={`sidebar-session-${pane.sessionId}`}
              className={cn(
                "group relative mb-1 flex w-full items-start gap-2 rounded border px-2 py-2 text-left transition-colors",
                "focus:outline-none focus-visible:ring-1 focus-visible:ring-wc-accent",
                isActive
                  ? "border-wc-accent bg-wc-surface-raised text-wc-text-primary"
                  : "border-transparent text-wc-text-secondary hover:border-wc-default hover:bg-wc-surface-raised",
              )}
              onClick={() => activate(pane.sessionId)}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  activate(pane.sessionId);
                }
              }}
              onContextMenu={(event) => {
                event.preventDefault();
                setTabContextMenu({ sessionId: pane.sessionId, position: { x: event.clientX, y: event.clientY } });
              }}
            >
              {accentColor && (
                <span className="mt-1 h-8 w-1 shrink-0 rounded-full" style={{ backgroundColor: accentColor }} />
              )}
              <span className="min-w-0 flex-1">
                <span className="flex min-w-0 items-center gap-2">
                  {editingPaneId === pane.sessionId ? (
                    <input
                      ref={editingInputRef}
                      data-testid={`sidebar-rename-input-${pane.sessionId}`}
                      className="min-w-0 flex-1 rounded bg-wc-surface-input px-1 text-sm font-medium text-wc-text-primary outline-none ring-1 ring-wc-accent"
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
                    <span className="rounded-full bg-wc-accent px-1.5 py-0.5 text-[10px] font-semibold text-black">
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
              <span
                role="button"
                tabIndex={0}
                className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-wc-text-muted opacity-0 hover:bg-wc-surface-input hover:text-wc-text-primary group-hover:opacity-100 focus:opacity-100"
                onClick={(event) => {
                  event.stopPropagation();
                  onClosePane(pane.sessionId);
                }}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    event.stopPropagation();
                    onClosePane(pane.sessionId);
                  }
                }}
                title={t(strings.tabBar.closeTabTitle)}
                aria-label={t(strings.tabBar.closeTabAria, { name: pane.name })}
              >
                <X className="h-3.5 w-3.5" />
              </span>
            </div>
          );
        })}
      </div>

      <div className="flex items-center gap-2 border-t border-wc-default p-2">
        <Button
          data-testid="workspace-sidebar-new"
          variant="outline"
          size="sm"
          className="h-8 flex-1 justify-start"
          disabled={isCreating}
          title={plusButtonBehavior === "launcher" ? t(strings.floatingToolbar.launcherFirstTitle) : t(strings.floatingToolbar.terminalFirstTitle)}
          onPointerDown={plusHandlers.onPointerDown}
          onPointerUp={plusHandlers.onPointerUp}
          onPointerCancel={plusHandlers.onPointerCancel}
          onContextMenu={plusHandlers.onContextMenu}
        >
          <Plus className="mr-2 h-3.5 w-3.5" />
          {t(strings.workspace.newTerminalButton)}
        </Button>
        <Button
          data-testid="workspace-sidebar-settings"
          variant="ghost"
          size="icon"
          className="h-8 w-8"
          onClick={onOpenSettings}
          title={t(strings.workspace.settingsTitle)}
        >
          <Settings className="h-4 w-4" />
        </Button>
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
            "relative hidden shrink-0 flex-col border-r border-wc-default bg-wc-surface-header md:flex",
            isResizing && "select-none",
          )}
          style={{ width: size }}
        >
          {sidebarContent}
          <div
            data-testid="workspace-sidebar-resize-handle"
            className="absolute right-[-6px] top-0 z-20 h-full w-3 cursor-col-resize"
            {...resizeHandleProps}
          >
            <div className="mx-auto h-full w-px bg-transparent transition-colors hover:bg-wc-accent" />
          </div>
        </aside>
      )}

      {isMobile && mobileOpen && (
        <div className="fixed inset-0 z-50 md:hidden" role="dialog" aria-modal="true" aria-label={t(strings.sessionSidebar.title)}>
          <button
            data-testid="workspace-sidebar-backdrop"
            className="absolute inset-0 bg-black/55"
            onClick={onCloseMobile}
            aria-label={t(strings.sessionSidebar.close)}
          />
          <aside
            data-testid="workspace-sidebar-shell"
            className="absolute inset-y-0 left-0 flex w-[min(22rem,calc(100vw-2rem))] flex-col border-r border-wc-default bg-wc-surface-header shadow-xl"
          >
            {sidebarContent}
          </aside>
        </div>
      )}

      {tabContextMenu && (() => {
        const paneItem = paneItems.find((item) => item.pane.sessionId === tabContextMenu.sessionId);
        if (!paneItem) return null;
        return (
          <TabContextMenu
            position={tabContextMenu.position}
            sessionId={tabContextMenu.sessionId}
            currentGroupId={paneItem.pane.groupId}
            groups={groups}
            onRename={() => {
              startRename(paneItem.pane.sessionId, paneItem.pane.name);
              setTabContextMenu(null);
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
              const serverGroup = await syncCreateGroup("New Group", "#3b82f6");
              addGroup({
                id: serverGroup.id,
                name: serverGroup.name,
                color: serverGroup.color,
                isCollapsed: false,
              });
              setPaneGroup(targetSessionId, serverGroup.id);
              syncPaneUpdate(targetSessionId, { group_id: serverGroup.id });
            }}
            onClose={onClosePane}
            onDismiss={() => setTabContextMenu(null)}
          />
        );
      })()}
    </>
  );
}
