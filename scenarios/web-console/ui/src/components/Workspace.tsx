// DOC: docs/concepts/ARCHITECTURE.md#system-layers
// DOC: docs/internal/SEAMS.md#1-entry--presentation
import { useState, useCallback } from "react";
import { Plus, Menu, X, Settings, List } from "lucide-react";
import type { Route } from "../hooks/useHashRoute";
import { PANE_MIN_WIDTH_PX, PANE_MIN_HEIGHT_PX } from "../consts/config";
import { useSessionManager } from "../hooks/useSessionManager";
import { cn } from "../lib/classnames";
import { pluralize } from "../lib/pluralize";
import { Button } from "./ui/button";
import ErrorBanner from "./ErrorBanner";
import ErrorBoundary from "./ErrorBoundary";
import TerminalPane from "./TerminalPane";
import TerminalLauncher from "./TerminalLauncher";
import MobileToolbar from "./MobileToolbar";
import AiInput from "./AiInput";
import SessionDrawer from "./SessionDrawer";

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
export default function Workspace({ onNavigate }: { onNavigate?: (to: Route) => void }) {
  const {
    panes,
    isCreating,
    createError,
    activePane,
    setActivePane,
    clearError,
    launchSession,
    handleTerminalReady,
    removePane,
    handleExit,
    sendToActiveTerminal,
    registerTerminalRef,
  } = useSessionManager();

  const [launcherOpen, setLauncherOpen] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  const openLauncher = useCallback(() => setLauncherOpen(true), []);
  const closeLauncher = useCallback(() => setLauncherOpen(false), []);

  // [REQ:P0-006b] Configurable Shortcut Entries
  const handleLaunch = useCallback(
    async (command?: string) => {
      const session = await launchSession(command);
      if (session) setLauncherOpen(false);
    },
    [launchSession],
  );

  // Single retry handler shared by both empty-state and header ErrorBanners
  const handleRetry = useCallback(() => {
    clearError();
    handleLaunch();
  }, [clearError, handleLaunch]);

  if (panes.length === 0) {
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

  return (
    <div className="flex h-screen flex-col bg-wc-surface-base text-wc-text-primary">
      {/* Header bar - [REQ:P0-001c] Pane Management Controls */}
      <div className="flex items-center justify-between border-b border-wc-default px-4 py-2">
        <div className="flex items-center gap-2">
          {/* [REQ:P0-008a] Drawer toggle */}
          <Button
            data-testid="drawer-toggle"
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={() => setDrawerOpen((prev) => !prev)}
          >
            <Menu className="h-4 w-4" />
          </Button>
          <span className="text-sm text-wc-text-muted">
            {panes.length} {pluralize(panes.length, "terminal")}
          </span>
        </div>
        <div className="flex items-center gap-1">
          {onNavigate && (
            <>
              <Button
                data-testid="nav-sessions"
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={() => onNavigate("sessions")}
                title="Sessions"
              >
                <List className="h-4 w-4" />
              </Button>
              <Button
                data-testid="nav-settings"
                variant="ghost"
                size="icon"
                className="h-8 w-8"
                onClick={() => onNavigate("settings")}
                title="Settings"
              >
                <Settings className="h-4 w-4" />
              </Button>
            </>
          )}
          <Button
            data-testid="new-terminal-button"
            variant="ghost"
            size="sm"
            onClick={openLauncher}
            disabled={isCreating}
          >
            <Plus className="mr-1 h-4 w-4" />
            New
          </Button>
        </div>
      </div>

      {/* Error banner for session creation failures */}
      {createError && (
        <ErrorBanner
          error={createError}
          onDismiss={clearError}
          onRetry={createError.retry ? handleRetry : undefined}
          className="border-b border-wc-error"
        />
      )}

      {/* Pane grid - responsive: 2 columns desktop, 1 column mobile */}
      <div
        data-testid="pane-grid"
        className="flex-1 grid gap-1 p-1 overflow-hidden"
        style={{
          gridTemplateColumns:
            panes.length === 1
              ? "1fr"
              : `repeat(auto-fit, minmax(min(100%, ${PANE_MIN_WIDTH_PX}px), 1fr))`,
          gridAutoRows: panes.length <= 2 ? "1fr" : `minmax(${PANE_MIN_HEIGHT_PX}px, 1fr)`,
        }}
      >
        {panes.map((pane) => (
          <div
            key={pane.session.id}
            data-testid="terminal-pane-container"
            data-session-id={pane.session.id}
            className={cn("relative rounded border overflow-hidden", activePane === pane.session.id ? "border-wc-accent" : "border-wc-default")}
            onClick={() => setActivePane(pane.session.id)}
          >
            <Button
              data-testid={`terminal-close-${pane.session.id}`}
              variant="ghost"
              size="icon"
              className="absolute right-1 top-1 z-10 h-6 w-6 text-wc-text-faint"
              onClick={(e) => {
                e.stopPropagation();
                removePane(pane.session.id);
              }}
              title="Close terminal"
            >
              <X className="h-3 w-3" />
            </Button>
            <ErrorBoundary region="terminal">
              <TerminalPane
                sessionId={pane.session.id}
                onExit={handleExit}
                onReady={() => handleTerminalReady(pane.session.id)}
                ref={(handle) => registerTerminalRef(pane.session.id, handle)}
              />
            </ErrorBoundary>
          </div>
        ))}
      </div>

      {/* [REQ:P0-005b] AI Input Component */}
      <AiInput
        onExecute={sendToActiveTerminal}
        hasActiveTerminal={activePane !== null}
      />

      {/* [REQ:P0-007a] Mobile floating toolbar */}
      <MobileToolbar onInput={sendToActiveTerminal} />

      {/* [REQ:P0-006a] Terminal Launcher */}
      <TerminalLauncher
        open={launcherOpen}
        onClose={closeLauncher}
        onLaunch={handleLaunch}
        isCreating={isCreating}
      />

      {/* [REQ:P0-008a] Session Drawer */}
      <SessionDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        sessions={panes}
        onDeleteSession={removePane}
        onSelectSession={(id) => {
          setActivePane(id);
          setDrawerOpen(false);
        }}
      />
    </div>
  );
}
