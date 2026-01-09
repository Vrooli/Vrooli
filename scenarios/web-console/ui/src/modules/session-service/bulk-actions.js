import { elements, state } from "../state.js";
import { appendEvent, showError } from "../event-feed.js";
import { proxyToApi } from "../utils.js";
import { showToast } from "../notifications.js";
import {
  getActiveTab,
  renderTabs,
  getDetachedTabs,
  createTerminalTab,
  refreshTabButton,
} from "../tab-manager.js";
import {
  queueSessionOverviewRefresh,
  notifySessionActionsChanged,
} from "./callbacks.js";
import { setTabPhase, setTabSocketState } from "./tab-state.js";
import {
  startSession,
  stopSession,
  transmitInputForTab,
  queueInputForTab,
} from "./actions.js";

export async function handleSignOutAllSessions() {
  const button = elements.signOutAllSessions;
  if (!button || button.dataset.busy === "true") return;

  const attachedTabs = state.tabs.filter(
    (tab) => tab.session && tab.phase !== "closed",
  );
  const attachedSessionIds = new Set(attachedTabs.map((tab) => tab.session.id));
  const apiSessions = Array.isArray(state.sessions.items)
    ? state.sessions.items
    : [];
  const detachedSessions = apiSessions.filter(
    (session) =>
      session &&
      typeof session.id === "string" &&
      !attachedSessionIds.has(session.id),
  );

  const totalSessions = attachedTabs.length + detachedSessions.length;
  if (totalSessions === 0) {
    button.blur();
    await showToast("No active sessions to sign out.", "info", 2200);
    return;
  }

  let confirmMessage;
  if (totalSessions === 1) {
    confirmMessage =
      "Sign out the active session? This will close the terminal.";
  } else if (detachedSessions.length > 0 && attachedTabs.length > 0) {
    confirmMessage = `Sign out all ${totalSessions} sessions? (${detachedSessions.length} detached)`;
  } else if (detachedSessions.length > 0) {
    confirmMessage =
      detachedSessions.length === 1
        ? "Sign out the detached session? This will terminate the remote shell."
        : `Sign out all ${detachedSessions.length} detached sessions?`;
  } else {
    confirmMessage = `Sign out all ${totalSessions} sessions? This will close every terminal.`;
  }
  if (!window.confirm(confirmMessage)) {
    return;
  }

  const originalLabel = button.textContent || "";
  button.dataset.busy = "true";
  button.dataset.label = originalLabel;
  button.disabled = true;
  button.textContent = "Signing out…";

  attachedTabs.forEach((tab) => {
    appendEvent(tab, "session-stop-all-requested", { id: tab.session.id });
    setTabPhase(tab, "closing");
    const socketOpen = tab.socket && tab.socket.readyState === WebSocket.OPEN;
    setTabSocketState(tab, socketOpen ? "closing" : "disconnected");
    tab.wasDetached = true;
    refreshTabButton(tab);
  });

  try {
    const response = await proxyToApi("/api/v1/sessions", { method: "DELETE" });
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `API error (${response.status})`);
    }

    let payload;
    const contentType = response.headers.get("Content-Type") || "";
    if (contentType.includes("application/json")) {
      try {
        payload = await response.json();
      } catch (_error) {
        payload = null;
      }
    }

    const eventPayload =
      payload && typeof payload === "object"
        ? payload
        : { status: "terminating_all", terminated: totalSessions };

    attachedTabs.forEach((tab) => {
      appendEvent(tab, "session-stop-all", eventPayload);
      if (tab.socket) {
        try {
          tab.socket.close();
        } catch (_error) {
          // ignore close failures
        }
      }
    });
    queueSessionOverviewRefresh(250);

    if (detachedSessions.length > 0 && totalSessions > attachedTabs.length) {
      const message =
        detachedSessions.length === 1
          ? "Requested sign out for the detached session."
          : `Requested sign out for ${detachedSessions.length} detached sessions.`;
      await showToast(message, "info", 2400);
    }
  } catch (error) {
    const message =
      error instanceof Error ? error.message : "Unable to sign out sessions";
    const activeTab = getActiveTab();
    if (activeTab) {
      showError(activeTab, message);
    }
    attachedTabs.forEach((tab) => {
      appendEvent(tab, "session-stop-all-error", error);
      if (tab.phase === "closing") {
        setTabPhase(tab, tab.session ? "running" : "idle");
      }
      if (tab.socketState === "closing") {
        const socketOpen =
          tab.socket && tab.socket.readyState === WebSocket.OPEN;
        setTabSocketState(tab, socketOpen ? "open" : "disconnected");
      }
      refreshTabButton(tab);
    });
  } finally {
    const label = button.dataset.label || "Sign out all sessions";
    button.textContent = label;
    delete button.dataset.label;
    delete button.dataset.busy;
    button.disabled = false;
    notifySessionActionsChanged();
    queueSessionOverviewRefresh(500);
  }
}

export async function handleCloseDetachedTabs() {
  const button = elements.closeDetachedTabs;
  if (!button || button.dataset.busy === "true") return;

  const tabsToClose = getDetachedTabs().slice();
  if (tabsToClose.length === 0) {
    button.classList.add("hidden");
    return;
  }

  const originalLabel = button.textContent || "Close detached tabs";
  button.dataset.busy = "true";
  button.disabled = true;
  button.textContent = `Closing ${tabsToClose.length}…`;
  notifySessionActionsChanged();

  let closedCount = 0;
  let hadErrors = false;
  for (const tab of tabsToClose) {
    try {
      await stopSession(tab);
      closedCount += 1;
    } catch (error) {
      console.error("Failed to close detached tab", error);
      hadErrors = true;
    }
  }

  button.textContent = originalLabel;
  delete button.dataset.busy;
  button.disabled = false;

  renderTabs();
  notifySessionActionsChanged();

  if (closedCount > 0) {
    queueSessionOverviewRefresh(200);
  }
  if (hadErrors) {
    await showToast(
      "Some detached tabs could not be closed. Check the console for details.",
      "error",
      3500,
    );
  } else if (closedCount > 0) {
    await showToast(
      closedCount === 1
        ? "Closed detached tab."
        : `Closed ${closedCount} detached tabs.`,
      "success",
      2500,
    );
  } else {
    await showToast("No detached tabs to close.", "info", 2500);
  }
}

export function updateSessionActions() {
  const signOutButton = elements.signOutAllSessions;
  if (signOutButton) {
    if (signOutButton.dataset.busy === "true") {
      signOutButton.disabled = true;
    } else {
      const hasLocalSessions = state.tabs.some(
        (entry) => entry.session && entry.phase !== "closed",
      );
      const remoteSessionCount = Array.isArray(state.sessions.items)
        ? state.sessions.items.filter(
            (session) => session && typeof session.id === "string",
          ).length
        : 0;
      signOutButton.disabled = !hasLocalSessions && remoteSessionCount === 0;
    }
  }

  const closeDetachedButton = elements.closeDetachedTabs;
  if (closeDetachedButton) {
    const detachedTabs = getDetachedTabs();
    const detachedCount = detachedTabs.length;
    if (closeDetachedButton.dataset.busy === "true") {
      closeDetachedButton.disabled = true;
      closeDetachedButton.classList.remove("hidden");
    } else {
      const label =
        detachedCount === 1
          ? "Close 1 detached tab"
          : `Close ${detachedCount} detached tabs`;
      closeDetachedButton.textContent = label;
      closeDetachedButton.disabled = detachedCount === 0;
      closeDetachedButton.classList.toggle("hidden", detachedCount === 0);
    }
  }
}

export async function handleShortcutAction({ command, id }) {
  const sourceId = id || "unknown";
  const trimmedCommand = typeof command === "string" ? command.trim() : "";

  if (!trimmedCommand) {
    const tab = getActiveTab();
    if (tab) {
      appendEvent(tab, "shortcut-invalid", { id: sourceId });
    }
    return;
  }

  let tab = getActiveTab();
  if (!tab) {
    tab = createTerminalTab({ focus: true });
    if (tab) {
      startSession(tab, { reason: `shortcut:${sourceId}` }).catch((error) => {
        appendEvent(tab, "session-error", error);
        showError(
          tab,
          error instanceof Error
            ? error.message
            : "Unable to start terminal session",
        );
      });
    }
  }
  if (!tab) return;

  const meta = {
    eventType: "shortcut",
    source: sourceId,
    command: trimmedCommand,
    clearError: true,
    appendNewline: true,
  };

  if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
    const success = transmitInputForTab(tab, trimmedCommand, meta);
    if (!success) {
      showError(tab, "Terminal stream is not connected");
    }
    return;
  }

  queueInputForTab(tab, trimmedCommand, meta);
  appendEvent(tab, "shortcut-queued", {
    id: sourceId,
    command: trimmedCommand,
  });

  if (tab.phase === "idle" || tab.phase === "closed") {
    startSession(tab, { reason: `shortcut:${sourceId}` }).catch((error) => {
      appendEvent(tab, "shortcut-start-error", error);
      showError(tab, "Unable to start terminal session for shortcut");
    });
  }
}
