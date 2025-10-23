// @ts-check

import { state } from "../state.js";
import {
  findTab,
  setActiveTab,
  renderTabs,
  createTerminalTab,
  destroyTerminalTab,
  closeTabCustomization,
  getActiveTab,
} from "../tab-manager.js";
import {
  startSession,
  stopSession,
  updateSessionActions,
} from "../session-service.js";
import { appendEvent, showError } from "../event-feed.js";
import {
  syncActiveTabState,
  syncTabToWorkspace,
  deleteTabFromWorkspace,
} from "../workspace.js";
import {
  renderSessionOverview,
  queueSessionOverviewRefresh,
} from "../session-overview.js";
import { proxyToApi } from "../utils.js";
import { showToast } from "../notifications.js";

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * @param {string} tabId
 */
export async function closeTab(tabId) {
  const tab = findTab(tabId);
  if (!tab) return;

  if (state.tabMenu.open && state.tabMenu.tabId === tabId) {
    closeTabCustomization();
  }

  const sessionStopPromise = Promise.resolve(stopSession(tab));
  destroyTerminalTab(tab);
  state.tabs = state.tabs.filter((entry) => entry.id !== tabId);

  let fallback = null;
  if (state.activeTabId === tabId) {
    fallback = state.tabs[state.tabs.length - 1] || state.tabs[0] || null;
    state.activeTabId = fallback ? fallback.id : null;
  }

  renderTabs();

  if (state.tabs.length === 0) {
    const replacement = /** @type {TerminalTab | null} */ (
      createTerminalTab({ focus: true })
    );
    if (replacement) {
      await syncTabToWorkspace(replacement);
      startSession(replacement, { reason: "replacement-tab" }).catch((error) => {
        appendEvent(replacement, "session-error", error);
        showError(
          replacement,
          error instanceof Error
            ? error.message
            : "Unable to start terminal session",
        );
      });
    }
  } else if (fallback) {
    setActiveTab(fallback.id);
  }

  syncActiveTabState();
  updateSessionActions();
  renderSessionOverview();

  try {
    await deleteTabFromWorkspace(tabId);
  } finally {
    await sessionStopPromise;
  }
}

/**
 * @param {string} sessionId
 * @returns {Promise<boolean>}
 */
export async function handleSessionCloseFromOverview(sessionId) {
  const trimmed = typeof sessionId === "string" ? sessionId.trim() : "";
  if (!trimmed) {
    return false;
  }

  const attached = state.tabs.find(
    (entry) => entry.session && entry.session.id === trimmed,
  );
  if (attached) {
    await closeTab(attached.id);
    return true;
  }

  try {
    const response = await proxyToApi(
      `/api/v1/sessions/${encodeURIComponent(trimmed)}`,
      {
        method: "DELETE",
      },
    );

    if (!response.ok && response.status !== 404) {
      const text = await response.text();
      const message = text || `API error (${response.status})`;
      throw new Error(message);
    }

    updateSessionActions();
    queueSessionOverviewRefresh(250);
    await showToast("Requested session sign out.", "success", 2200);
    return true;
  } catch (error) {
    console.error("Failed to terminate session", error);
    await showToast(
      "Unable to sign out session. Check the console for details.",
      "error",
      3200,
    );
    return false;
  }
}

/**
 * @param {string} tabId
 */
export function focusSessionTab(tabId) {
  const tab = findTab(tabId);
  if (!tab) {
    return;
  }
  if (state.activeTabId !== tab.id) {
    setActiveTab(tab.id);
  } else {
    tab.term?.focus();
  }
}

export function ensureActiveTabReconnects() {
  state.tabs.forEach((tab) => {
    if (tab.session && tab.phase === "running" && !tab.socket) {
      tab.reconnecting = false;
    }
  });
}
