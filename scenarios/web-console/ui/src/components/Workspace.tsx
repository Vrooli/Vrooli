// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback, useEffect, useRef } from "react";
import type { PointerEvent as ReactPointerEvent } from "react";
import { Plus } from "lucide-react";
import { SPLITTER_SIZE_PX, MIN_COLUMN_PX, MIN_ROW_PX } from "../consts/config";
import { useSessionManager } from "../hooks/useSessionManager";
import { useVirtualKeyboard } from "../hooks/useVirtualKeyboard";
import { useWorkspaceStore } from "../stores/useWorkspaceStore";
import {
  resolveWorkspaceLayout,
  reconcileTrackFractions,
  buildGridTrackTemplate,
  updateAdjacentFractions,
} from "../lib/gridLayout";
import { cn } from "../lib/classnames";
import { Button } from "./ui/button";
import ErrorBanner from "./ErrorBanner";
import ErrorBoundary from "./ErrorBoundary";
import TerminalPane from "./TerminalPane";
import TerminalHeader from "./TerminalHeader";
import GridSplitter from "./GridSplitter";
import TerminalLauncher from "./TerminalLauncher";
import MobileToolbar from "./MobileToolbar";
import AiInput from "./AiInput";
import FloatingToolbar from "./FloatingToolbar";
import WorkspaceMinimap from "./WorkspaceMinimap";
import SessionsModal from "./SessionsModal";
import SettingsModal from "./SettingsModal";

type ActiveResize = {
  axis: "column" | "row";
  index: number;
  startPointer: number;
  containerSize: number;
  startValues: number[];
};

type ActiveArrangeDrag = { paneId: string; dropIndex: number };

/**
 * ── STABLE CORE: Pane layout and control wiring. ──
 * This component owns ONLY visual layout (grid, header, empty state)
 * and wires child components to the session lifecycle hook.
 *
 * Session orchestration lives in useSessionManager.
 * Error display lives in ErrorBanner.
 * Shortcut data lives in consts/shortcuts.ts.
 *
 * [REQ:P0-001a] Responsive Pane Grid Layout
 * [REQ:P0-001b] Independent Pane Session Lifecycle
 */
export default function Workspace() {
  const {
    panes: sessionPanes,
    isHydrated,
    isCreating,
    createError,
    clearError,
    launchSession,
    handleTerminalReady,
    removePane: removeSessionPane,
    handleExit,
    sendToActiveTerminal,
    registerTerminalRef,
  } = useSessionManager();

  const store = useWorkspaceStore();
  const gridRef = useRef<HTMLDivElement>(null);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const activeResizeRef = useRef<ActiveResize | null>(null);
  const { keyboardHeight } = useVirtualKeyboard();

  const [launcherOpen, setLauncherOpen] = useState(false);

  // --- Mobile single-column layout ---
  const [isMobile, setIsMobile] = useState(
    () => typeof window !== "undefined" && window.innerWidth < 768,
  );
  useEffect(() => {
    const onResize = () => setIsMobile(window.innerWidth < 768);
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, []);

  // --- Pane drag-and-drop reordering ---
  const [activeArrangeDrag, setActiveArrangeDrag] =
    useState<ActiveArrangeDrag | null>(null);

  const startArrangeDrag = useCallback(
    (paneId: string, e: ReactPointerEvent) => {
      const idx = store.panes.findIndex((p) => p.sessionId === paneId);
      if (idx === -1) return;
      e.preventDefault();
      e.stopPropagation();
      (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
      setActiveArrangeDrag({ paneId, dropIndex: idx });
    },
    [store.panes],
  );

  useEffect(() => {
    if (activeArrangeDrag === null) return;

    const handleMove = (e: PointerEvent) => {
      const el = document.elementFromPoint(e.clientX, e.clientY);
      const paneEl = el?.closest("[data-pane-index]");
      if (paneEl) {
        const idx = Number(paneEl.getAttribute("data-pane-index"));
        if (Number.isFinite(idx)) {
          setActiveArrangeDrag((prev) =>
            prev ? { ...prev, dropIndex: idx } : null,
          );
        }
      }
    };

    const handleUp = () => {
      setActiveArrangeDrag((prev) => {
        if (prev) store.movePaneToIndex(prev.paneId, prev.dropIndex);
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
  }, [activeArrangeDrag, store]);

  // Reconcile session manager panes with workspace store.
  // Only remove stale store panes after session hydration completes —
  // otherwise the initial empty sessionPanes would nuke persisted metadata.
  useEffect(() => {
    const sessionIds = new Set(sessionPanes.map((p) => p.session.id));
    const storeIds = new Set(store.panes.map((p) => p.sessionId));

    // Add new sessions to store
    for (const sp of sessionPanes) {
      if (!storeIds.has(sp.session.id)) {
        store.addPane(sp.session.id, sp.session.shell ?? "terminal");
      }
    }
    // Remove deleted sessions from store (only after hydration)
    if (isHydrated) {
      for (const sid of storeIds) {
        if (!sessionIds.has(sid)) {
          store.removePane(sid);
        }
      }
    }
  }, [sessionPanes, store, isHydrated]);

  // Auto-set active pane if none is set
  useEffect(() => {
    if (store.activePane === null && store.panes.length > 0) {
      const lastPane = store.panes[store.panes.length - 1];
      if (lastPane) store.setActivePane(lastPane.sessionId);
    }
  }, [store]);

  const openLauncher = useCallback(() => setLauncherOpen(true), []);
  const closeLauncher = useCallback(() => setLauncherOpen(false), []);

  const handleLaunch = useCallback(
    async (command?: string) => {
      const session = await launchSession(command);
      if (session) setLauncherOpen(false);
    },
    [launchSession],
  );

  const handleRetry = useCallback(() => {
    clearError();
    handleLaunch();
  }, [clearError, handleLaunch]);

  const handleRemovePane = useCallback(
    (sessionId: string) => {
      removeSessionPane(sessionId);
      store.removePane(sessionId);
    },
    [removeSessionPane, store],
  );

  const handleSendToTerminal = useCallback(
    (data: string) => {
      sendToActiveTerminal(data, store.activePane ?? undefined);
    },
    [sendToActiveTerminal, store.activePane],
  );

  // --- Resize logic ---
  const startResize = useCallback(
    (axis: "column" | "row", index: number) =>
      (e: ReactPointerEvent) => {
        const container = gridRef.current;
        if (!container) return;
        const rect = container.getBoundingClientRect();
        activeResizeRef.current = {
          axis,
          index,
          startPointer: axis === "column" ? e.clientX : e.clientY,
          containerSize: axis === "column" ? rect.width : rect.height,
          startValues:
            axis === "column"
              ? store.columnFractions
              : store.rowFractions,
        };
      },
    [store.columnFractions, store.rowFractions],
  );

  useEffect(() => {
    const handleMove = (e: PointerEvent) => {
      const resize = activeResizeRef.current;
      if (!resize) return;
      e.preventDefault();
      const delta =
        (resize.axis === "column" ? e.clientX : e.clientY) -
        resize.startPointer;
      const splitterCount = resize.startValues.length - 1;
      const updated = updateAdjacentFractions({
        startValues: resize.startValues,
        index: resize.index,
        delta,
        containerSize: resize.containerSize,
        splitterCount,
        minTrackPx: resize.axis === "column" ? MIN_COLUMN_PX : MIN_ROW_PX,
        splitterSize: SPLITTER_SIZE_PX,
      });
      if (resize.axis === "column") {
        store.setColumnFractions(updated);
      } else {
        store.setRowFractions(updated);
      }
    };

    const handleUp = () => {
      activeResizeRef.current = null;
    };

    window.addEventListener("pointermove", handleMove, { passive: false });
    window.addEventListener("pointerup", handleUp);
    return () => {
      window.removeEventListener("pointermove", handleMove);
      window.removeEventListener("pointerup", handleUp);
    };
  }, [store]);

  // Compute layout
  const orderedPanes = store.panes;
  const maxColumns = isMobile ? 1 : 2;
  const layout = resolveWorkspaceLayout(orderedPanes.length, maxColumns);
  const colFractions = reconcileTrackFractions(
    store.columnFractions,
    layout.columns,
  );
  const rowFractions = reconcileTrackFractions(
    store.rowFractions,
    layout.rows,
  );

  // Persist reconciled fractions if they differ
  useEffect(() => {
    if (
      colFractions.length !== store.columnFractions.length ||
      colFractions.some((f, i) => f !== store.columnFractions[i])
    ) {
      store.setColumnFractions(colFractions);
    }
  }, [colFractions, store]);

  useEffect(() => {
    if (
      rowFractions.length !== store.rowFractions.length ||
      rowFractions.some((f, i) => f !== store.rowFractions[i])
    ) {
      store.setRowFractions(rowFractions);
    }
  }, [rowFractions, store]);

  const colTemplate = buildGridTrackTemplate(colFractions, SPLITTER_SIZE_PX);
  const rowTemplate = buildGridTrackTemplate(rowFractions, SPLITTER_SIZE_PX);

  // Scale grid min-height by row count so each row can grow up to viewport height.
  // Without this, fr tracks share a single-viewport container and panes shrink
  // as rows are added instead of extending the scrollable surface.
  const viewportPaneHeight = typeof window === "undefined"
    ? MIN_ROW_PX
    : Math.max(MIN_ROW_PX, window.innerHeight);
  const rowSplittersHeight = Math.max(0, layout.rows - 1) * SPLITTER_SIZE_PX;
  const minimumGridHeightPx = (viewportPaneHeight * layout.rows) + rowSplittersHeight;

  // Empty state
  if (sessionPanes.length === 0) {
    return (
      <div className="flex h-screen items-center justify-center bg-wc-surface-base text-wc-text-primary">
        <div className="text-center">
          <h1 className="text-2xl font-semibold mb-4">Web Console</h1>
          <p className="text-wc-text-muted mb-6">
            Browser terminal with PTY-backed sessions
          </p>
          {createError && (
            <ErrorBanner
              error={createError}
              onDismiss={clearError}
              onRetry={createError.retry ? handleRetry : undefined}
              className="mb-4"
            />
          )}
          <Button
            data-testid="new-terminal-button"
            onClick={openLauncher}
            disabled={isCreating}
            size="lg"
          >
            <Plus className="mr-2 h-5 w-5" />
            {isCreating ? "Creating..." : "New Terminal"}
          </Button>
        </div>
        <TerminalLauncher
          open={launcherOpen}
          onClose={closeLauncher}
          onLaunch={handleLaunch}
          isCreating={isCreating}
        />
      </div>
    );
  }

  // Build splitter elements
  const columnSplitters: React.ReactNode[] = [];
  for (let i = 0; i < colFractions.length - 1; i++) {
    const gridCol = `${2 + i * 2}`;
    columnSplitters.push(
      <GridSplitter
        key={`col-${i}`}
        axis="column"
        gridColumn={gridCol}
        gridRow={`1 / -1`}
        onPointerDown={startResize("column", i)}
        label={`Resize column ${i + 1}`}
      />,
    );
  }

  const rowSplitters: React.ReactNode[] = [];
  for (let i = 0; i < rowFractions.length - 1; i++) {
    const gridRow = `${2 + i * 2}`;
    rowSplitters.push(
      <GridSplitter
        key={`row-${i}`}
        axis="row"
        gridColumn={`1 / -1`}
        gridRow={gridRow}
        onPointerDown={startResize("row", i)}
        label={`Resize row ${i + 1}`}
      />,
    );
  }

  // Map panes into grid cells
  const isDragging = activeArrangeDrag !== null;
  const paneCells = orderedPanes.map((paneMeta, idx) => {
    const col = idx % layout.columns;
    const row = Math.floor(idx / layout.columns);
    // Grid positions account for splitter tracks: content is at odd positions (1, 3, 5, ...)
    const gridColumn = `${1 + col * 2}`;
    const gridRow = `${1 + row * 2}`;

    const isBeingDragged = activeArrangeDrag?.paneId === paneMeta.sessionId;
    const isDropTarget =
      isDragging &&
      !isBeingDragged &&
      activeArrangeDrag?.dropIndex === idx;

    return (
      <div
        key={paneMeta.sessionId}
        data-testid="terminal-pane-container"
        data-session-id={paneMeta.sessionId}
        data-pane-index={idx}
        {...(isDropTarget ? { "data-drop-target": "" } : {})}
        className={cn(
          "relative flex flex-col rounded border overflow-hidden min-w-0 min-h-0",
          store.activePane === paneMeta.sessionId
            ? "border-wc-accent"
            : "border-wc-default",
          isBeingDragged && "opacity-40",
          isDropTarget && "ring-2 ring-blue-400/60 ring-inset",
        )}
        style={{ gridColumn, gridRow }}
        onClick={() => store.setActivePane(paneMeta.sessionId)}
      >
        <TerminalHeader
          sessionId={paneMeta.sessionId}
          name={paneMeta.name}
          headerColor={paneMeta.headerColor}
          isActive={store.activePane === paneMeta.sessionId}
          onClose={() => handleRemovePane(paneMeta.sessionId)}
          onFocus={() => store.setActivePane(paneMeta.sessionId)}
          onDragStart={startArrangeDrag}
        />
        <div className="flex-1 min-h-0">
          <ErrorBoundary region="terminal">
            <TerminalPane
              sessionId={paneMeta.sessionId}
              onExit={handleExit}
              onReady={() => handleTerminalReady(paneMeta.sessionId)}
              ref={(handle) =>
                registerTerminalRef(paneMeta.sessionId, handle)
              }
            />
          </ErrorBoundary>
        </div>
      </div>
    );
  });

  return (
    <div className="flex h-screen flex-col bg-wc-surface-base text-wc-text-primary">
      {/* Floating toolbar */}
      <FloatingToolbar
        onOpenSessions={() => store.setSessionsModalOpen(true)}
        onOpenSettings={() => store.setSettingsModalOpen(true)}
        onOpenAi={() => store.setAiModalOpen(true)}
        onNewTerminal={() => handleLaunch()}
        onOpenLauncher={openLauncher}
        isCreating={isCreating}
      />

      {/* Error banner */}
      {createError && (
        <ErrorBanner
          error={createError}
          onDismiss={clearError}
          onRetry={createError.retry ? handleRetry : undefined}
          className="border-b border-wc-error"
        />
      )}

      {/* Pane grid + minimap wrapper */}
      <div className="relative flex-1 min-h-0">
        <div
          ref={scrollContainerRef}
          className={cn("absolute inset-0 overflow-auto wc-hide-scrollbar", store.isMinimapVisible && "right-[34px]")}
        >
          <div
            ref={gridRef}
            data-testid="pane-grid"
            className={cn(
              "grid gap-0 p-1",
              isDragging && "select-none cursor-grabbing [&_.xterm]:pointer-events-none",
            )}
            style={{
              gridTemplateColumns: colTemplate,
              gridTemplateRows: rowTemplate,
              height: `${minimumGridHeightPx}px`,
              minHeight: `${minimumGridHeightPx}px`,
            }}
          >
            {paneCells}
            {columnSplitters}
            {rowSplitters}
          </div>
        </div>

        {/* Minimap */}
        <WorkspaceMinimap scrollRef={scrollContainerRef} rowCount={layout.rows} />
      </div>

      {/* Bottom bar — offset for virtual keyboard on mobile */}
      <div
        className="shrink-0 transition-transform duration-150"
        style={
          keyboardHeight > 0
            ? { transform: `translateY(-${keyboardHeight}px)` }
            : undefined
        }
      >
        {/* Mobile toolbar */}
        <MobileToolbar onInput={handleSendToTerminal} />
      </div>

      {/* Terminal Launcher */}
      <TerminalLauncher
        open={launcherOpen}
        onClose={closeLauncher}
        onLaunch={handleLaunch}
        isCreating={isCreating}
      />

      {/* Sessions Modal */}
      <SessionsModal
        sessions={sessionPanes}
        onDeleteSession={handleRemovePane}
      />

      {/* Settings Modal */}
      <SettingsModal />

      {/* AI Modal */}
      <AiInput onExecute={handleSendToTerminal} />
    </div>
  );
}
