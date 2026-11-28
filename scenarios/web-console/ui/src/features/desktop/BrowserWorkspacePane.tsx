import { useCallback, useEffect, useState, type ReactNode } from "react";
import { Globe, HelpCircle, Loader2, X } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { ButtonGroup } from "@vrooli/react-component-library/ButtonGroup";
import { IconButton } from "@vrooli/react-component-library/IconButton";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog";
import { API_BASE_WITH_SUFFIX } from "../../api/client";
import { getAccessToken } from "../../lib/auth";
import { PaneEmptyState } from "../../components/PaneEmptyState";
import { BrowserHubPane } from "./BrowserHubPane";
import type { BrowserPaneSession } from "./BrowserPane";
import { WorkspacePaneToolbar, WorkspaceSurfacePane, type WorkspacePaneState } from "./WorkspaceSurface";

type PaneDiscovery = { sessions: Array<{ pane_handle: string; mode: string; can_control: boolean }>; run_finished?: boolean };

type BrowserPaneState =
  | { kind: "checking" }
  | { kind: "ready"; session: BrowserPaneSession }
  | { kind: "no-session" }
  | { kind: "multiple"; count: number }
  | { kind: "unauthorized" }
  | { kind: "owner-offline" }
  | { kind: "run-finished" };

export type BrowserWorkspacePaneVariant = "overlay" | "view";

const VALID_MODES: ReadonlySet<string> = new Set(["managed", "installed", "user-attached", "remote"]);

async function discoverBrowserPane(webConsoleSessionID: string): Promise<BrowserPaneState> {
  try {
    const token = await getAccessToken();
    const response = await fetch(`${API_BASE_WITH_SUFFIX}/browser/pane/sessions`, {
      method: "POST", cache: "no-store", credentials: "same-origin",
      headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({ web_console_session_id: webConsoleSessionID }),
    });
    if (response.status === 401 || response.status === 403) return { kind: "unauthorized" };
    if (!response.ok) return { kind: "owner-offline" };
    const result = await response.json() as PaneDiscovery;
    if (!result || !Array.isArray(result.sessions)) return { kind: "owner-offline" };
    if (result.run_finished) return { kind: "run-finished" };
    if (result.sessions.length === 0) return { kind: "no-session" };
    if (result.sessions.length > 1) return { kind: "multiple", count: result.sessions.length };
    const active = result.sessions[0];
    if (!active || !VALID_MODES.has(active.mode)) return { kind: "unauthorized" };
    return {
      kind: "ready",
      session: {
        sessionId: active.pane_handle,
        mode: active.mode as BrowserPaneSession["mode"],
        state: "viewer",
        controller: false,
        association: { source: "agent-run", verified: true },
        capabilities: { observe: true, control: active.can_control, frames: true },
      },
    };
  } catch {
    return { kind: "owner-offline" };
  }
}

/** Discovery details remain local; presentation state uses the shared pane vocabulary. */
export function projectBrowserDiscoveryState(state: BrowserPaneState): WorkspacePaneState {
  switch (state.kind) {
    case "checking": return "loading";
    case "ready": return state.session.controller ? "controlling" : "observe";
    case "no-session": return "closed";
    case "run-finished": return "closed";
    case "multiple":
    case "unauthorized": return "permission-blocked";
    case "owner-offline": return "offline";
  }
}

function statusDetail(state: BrowserPaneState): string | undefined {
  switch (state.kind) {
    case "no-session": return "Owner: Agent Manager · Session: not linked";
    case "multiple": return `Linked sessions: ${String(state.count)}`;
    case "unauthorized": return "Permission: not granted to this Web Console session";
    case "owner-offline": return "Owner: unreachable";
    case "run-finished": return "Owner: Agent Manager · Run: finished";
    default: return undefined;
  }
}

/** Workspace mount. It never opens a browser: it asks the owner chain to
 * attest an already active session and renders one honest state. */
export function BrowserWorkspacePane({
  webConsoleSessionID,
  onClose,
  variant = "overlay",
  headerActions,
}: {
  webConsoleSessionID: string;
  onClose: () => void;
  variant?: BrowserWorkspacePaneVariant;
  headerActions?: ReactNode;
}) {
  const [state, setState] = useState<BrowserPaneState>({ kind: "checking" });
  const [retryKey, setRetryKey] = useState(0);
  const [helpOpen, setHelpOpen] = useState(false);
  const revalidateOwner = useCallback(async (reason: "stream-lost" | "owner-refused") => {
    const next = await discoverBrowserPane(webConsoleSessionID);
    if (next.kind !== "ready") {
      setState(next);
      return false;
    }
    if (reason === "stream-lost") {
      // Replace the expiring cached selector with a fresh owner-discovered
      // alias. The pane effect then reattaches through BAS's receipt exchange.
      setState(next);
      return true;
    }
    // A refusal after a reconnect attempt is authoritative for that attempt.
    // Keep its error visible and remove retry; do not spin on fresh aliases.
    return false;
  }, [webConsoleSessionID]);

  useEffect(() => {
    let canceled = false;
    setState({ kind: "checking" });
    void discoverBrowserPane(webConsoleSessionID).then((next) => { if (!canceled) setState(next); });
    return () => { canceled = true; };
  }, [retryKey, webConsoleSessionID]);

  const retry = () => { setRetryKey((value) => value + 1); };
  const returnToRun = async () => {
    if (state.kind !== "ready") throw new Error("The browser pane is no longer authorized.");
    const token = await getAccessToken();
    const response = await fetch(`${API_BASE_WITH_SUFFIX}/browser/pane/return`, {
      method: "POST", cache: "no-store", credentials: "same-origin",
      headers: { "Content-Type": "application/json", ...(token ? { Authorization: `Bearer ${token}` } : {}) },
      body: JSON.stringify({ pane_handle: state.session.sessionId }),
    });
    if (!response.ok) throw new Error("Agent Manager has not confirmed return to the run.");
  };

  const isView = variant === "view";
  const detail = statusDetail(state);
  const emptyTitle = state.kind === "checking"
    ? "Checking for a browser"
    : state.kind === "no-session"
      ? "No browser is attached"
      : state.kind === "multiple"
        ? "More than one browser is linked"
        : state.kind === "run-finished"
          ? "The agent run has finished"
          : state.kind === "unauthorized"
          ? "Browser access isn't available"
          : "Browser service is unreachable";
  const emptyDescription = state.kind === "checking"
    ? "Verifying whether this conversation has an authorized browser session."
    : state.kind === "no-session"
      ? "Browser view opens a live session owned by an agent run. This conversation does not have one yet."
      : state.kind === "multiple"
        ? "Aquila will not guess which run you meant. Close one in its run, then check again."
        : state.kind === "run-finished"
          ? "This run is no longer active, so its previous browser session cannot be reconnected."
          : state.kind === "unauthorized"
          ? "The browser owner could not verify an active session for this conversation. The browser may be running in another view."
          : "The service that owns the browser isn't answering. Your terminal is unaffected.";

  return (
    <>
      <div
        data-testid="browser-workspace-pane"
        data-variant={variant}
        className={isView
          ? "flex h-full min-h-0 flex-col bg-wc-surface-base"
          : "absolute inset-0 z-50 flex min-h-0 flex-col bg-wc-surface-base"}
      >
        {state.kind === "ready" ? (
          <div className="min-h-0 flex-1">
            <BrowserHubPane
              session={state.session}
              onClose={onClose}
              closeLabel="Close pane"
              onControlReturned={returnToRun}
              onRevalidateOwner={revalidateOwner}
              headerActions={headerActions}
            />
          </div>
        ) : (
          <div className="min-h-0 flex-1 overflow-auto">
            <WorkspaceSurfacePane label="Browser workspace pane" title="Browser" owner="bas" state={projectBrowserDiscoveryState(state)} summary={detail}>
              <PaneEmptyState
                testId="browser-pane-empty"
                icon={state.kind === "checking" ? <Loader2 className="animate-spin motion-reduce:animate-none" /> : <Globe />}
                title={emptyTitle}
                description={emptyDescription}
                footnote={detail}
                {...(detail ? { footnoteTestId: "browser-pane-detail" } : {})}
                actions={(
                  <ButtonGroup align="center">
                    {state.kind !== "checking" && state.kind !== "run-finished" && (
                      <Button
                        type="button"
                        variant="primary"
                        data-testid="browser-pane-retry"
                        onClick={retry}
                      >
                        {state.kind === "owner-offline" ? "Retry" : "Check again"}
                      </Button>
                    )}
                    {(state.kind === "no-session" || state.kind === "multiple" || state.kind === "unauthorized") && (
                      <Button
                        type="button"
                        variant="secondary"
                        data-testid="browser-pane-help"
                        onClick={() => { setHelpOpen(true); }}
                      >
                        <HelpCircle className="h-4 w-4" aria-hidden="true" />
                        How to enable browser view
                      </Button>
                    )}
                    {(state.kind === "checking" || state.kind === "owner-offline" || state.kind === "run-finished") && (
                      <Button type="button" variant="secondary" onClick={onClose}>
                        Close
                      </Button>
                    )}
                  </ButtonGroup>
                )}
              />
              {(headerActions || !isView) && (
                <WorkspacePaneToolbar label="Browser pane controls" state={projectBrowserDiscoveryState(state)}>
                  {headerActions}
                  {!isView && <IconButton data-testid="browser-pane-close" aria-label="Close browser pane" title="Close browser pane" surface="ghost" size="xs" shape="circle" onClick={onClose}><X /></IconButton>}
                </WorkspacePaneToolbar>
              )}
            </WorkspaceSurfacePane>
          </div>
        )}
      </div>

      <ResponsiveDialog
        open={helpOpen}
        onOpenChange={setHelpOpen}
        onClose={() => { setHelpOpen(false); }}
        title="How to enable browser view"
        closeLabel="Close"
        size="md"
        contentPadding="comfortable"
        testId="browser-help-dialog"
      >
        <p className="text-sm text-wc-text-secondary">
          Browser sessions are created by an agent run. Aquila does not launch a browser by itself, and it never
          guesses which run should own the view.
        </p>
        <ol className="mt-4 space-y-3 text-sm text-wc-text-secondary">
          <li className="flex gap-3">
            <span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-wc-accent/15 text-xs font-bold text-wc-accent">1</span>
            <span>Start or resume an <b className="text-wc-text-primary">agent run that owns a browser</b>.</span>
          </li>
          <li className="flex gap-3">
            <span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-wc-accent/15 text-xs font-bold text-wc-accent">2</span>
            <span>Wait until that run shows an active browser session.</span>
          </li>
          <li className="flex gap-3">
            <span className="grid h-6 w-6 shrink-0 place-items-center rounded-full bg-wc-accent/15 text-xs font-bold text-wc-accent">3</span>
            <span>Return here and press <b className="text-wc-text-primary">Check again</b>.</span>
          </li>
        </ol>
        <div className="mt-6 flex flex-wrap justify-end gap-2">
          <Button type="button" variant="secondary" onClick={() => { setHelpOpen(false); }}>
            Close
          </Button>
          <Button
            type="button"
            variant="primary"
            onClick={() => { setHelpOpen(false); retry(); }}
          >
            Check again
          </Button>
        </div>
      </ResponsiveDialog>
    </>
  );
}
