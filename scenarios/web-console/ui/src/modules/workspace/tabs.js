import { state } from "../state.js";
import {
  createTerminalTab,
  findTab,
  setActiveTab,
  destroyTerminalTab,
  renderTabs,
  applyTabAppearance,
  getActiveTab,
  refreshTabButton,
} from "../tab-manager.js";
import { appendEvent } from "../event-feed.js";
import { proxyToApi } from "../utils.js";
import { showToast } from "../notifications.js";
import {
  reconnectSession,
  updateSessionActions,
  setTabPhase,
  setTabSocketState,
} from "../session-service.js";
import { workspaceAnomalyState, debugWorkspace } from "./constants.js";
import { queueSessionOverviewRefresh } from "./callbacks.js";

export function ensureSessionPlaceholder(tab, sessionId) {
  if (!tab || !sessionId) return;
  if (!tab.session || tab.session.id !== sessionId) {
    tab.session = { id: sessionId };
  }
  tab.wasDetached = true;
  refreshTabButton(tab);
}

export function sanitizeWorkspaceTabsFromServer(rawTabs) {
  const duplicates = [];
  const invalid = [];
  const seen = new Set();
  const sanitized = [];

  rawTabs.forEach((entry) => {
    if (!entry || typeof entry !== "object") {
      return;
    }
    const id = typeof entry.id === "string" ? entry.id.trim() : "";
    if (!id) {
      invalid.push(entry.id || "(missing)");
      return;
    }
    if (seen.has(id)) {
      duplicates.push(id);
      return;
    }
    seen.add(id);
    sanitized.push({
      id,
      label: entry.label || `Terminal ${sanitized.length + 1}`,
      colorId: entry.colorId || "sky",
      order: sanitized.length,
      sessionId: entry.sessionId || null,
    });
  });

  return {
    tabs: sanitized,
    duplicateIds: duplicates,
    invalidIds: invalid,
  };
}

export function pruneDuplicateLocalTabs() {
  const seen = new Set();
  const filtered = [];
  const removedIds = [];

  state.tabs.slice().forEach((tab) => {
    if (!tab || typeof tab !== "object" || typeof tab.id !== "string") {
      return;
    }
    if (seen.has(tab.id)) {
      removedIds.push(tab.id);
      destroyTerminalTab(tab);
      return;
    }
    seen.add(tab.id);
    filtered.push(tab);
  });

  state.tabs = filtered;
  if (removedIds.length > 0) {
    renderTabs();
  }
  return { tabs: filtered, removedIds };
}

export function reportWorkspaceAnomaly({
  duplicateIds = [],
  invalidIds = [],
  localDuplicates = [],
} = {}) {
  if (!duplicateIds.length && !invalidIds.length && !localDuplicates.length) {
    return;
  }

  const parts = [];
  if (duplicateIds.length) {
    parts.push(`duplicate tab IDs: ${duplicateIds.join(", ")}`);
  }
  if (invalidIds.length) {
    parts.push(`invalid entries: ${invalidIds.join(", ")}`);
  }
  if (localDuplicates.length) {
    parts.push(`pruned local duplicates: ${localDuplicates.join(", ")}`);
  }

  const summary = parts.join("; ");

  if (debugWorkspace) {
    console.warn("Workspace anomalies detected and repaired:", summary);
  }

  const activeTab = getActiveTab();
  if (activeTab) {
    appendEvent(activeTab, "workspace-sanitized", {
      summary,
      duplicateIds,
      invalidIds,
      localDuplicates,
    });
  }

  if (!workspaceAnomalyState.toastShown) {
    workspaceAnomalyState.toastShown = true;
    const maybePromise = showToast(
      "Workspace layout was repaired to remove duplicate tabs.",
      "warning",
      4200,
    );
    if (maybePromise && typeof maybePromise.catch === "function") {
      maybePromise.catch(() => {});
    }
  }
}

export async function syncTabToWorkspace(tab) {
  try {
    await proxyToApi("/api/v1/workspace/tabs", {
      method: "POST",
      json: {
        id: tab.id,
        label: tab.label,
        colorId: tab.colorId,
        order: state.tabs.indexOf(tab),
        sessionId: tab.session && tab.session.id ? tab.session.id : null,
      },
    });
  } catch (error) {
    console.error("Failed to sync tab to workspace:", error);
  }
}

export async function deleteTabFromWorkspace(tabId) {
  try {
    await proxyToApi(`/api/v1/workspace/tabs/${tabId}`, {
      method: "DELETE",
    });
  } catch (error) {
    console.error("Failed to delete tab from server:", error);
  }
}

export function syncActiveTabState() {
  const { tabs: uniqueTabs, removedIds } = pruneDuplicateLocalTabs();
  if (removedIds.length > 0) {
    reportWorkspaceAnomaly({ localDuplicates: removedIds });
  }

  let activeId = state.activeTabId;
  if (activeId && !uniqueTabs.some((tab) => tab.id === activeId)) {
    activeId = uniqueTabs.length > 0 ? uniqueTabs[0].id : null;
    state.activeTabId = activeId;
  }

  proxyToApi("/api/v1/workspace", {
    method: "PUT",
    json: {
      activeTabId: activeId,
      tabs: uniqueTabs.map((t, idx) => ({
        id: t.id,
        label: t.label,
        colorId: t.colorId,
        order: idx,
        sessionId: t.session && t.session.id ? t.session.id : null,
      })),
    },
  }).catch((error) => {
    console.error("Failed to sync active tab to server:", error);
  });
}

export function handleTabAdded(payload) {
  const existing = findTab(payload.id);
  if (existing) return;

  createTerminalTab({
    focus: false,
    id: payload.id,
    label: payload.label,
    colorId: payload.colorId,
  });
}

export function handleTabUpdated(payload) {
  const tab = findTab(payload.id);
  if (!tab) return;

  tab.label = payload.label;
  tab.colorId = payload.colorId;
  applyTabAppearance(tab);
  renderTabs();
  if (tab.id === state.activeTabId) {
    updateSessionActions();
  }
}

export function handleTabRemoved(payload) {
  const tab = findTab(payload.id);
  if (!tab) return;

  destroyTerminalTab(tab);
  state.tabs = state.tabs.filter((entry) => entry.id !== payload.id);

  if (state.activeTabId === payload.id) {
    const fallback = state.tabs[state.tabs.length - 1] || state.tabs[0] || null;
    state.activeTabId = fallback ? fallback.id : null;
  }

  renderTabs();
  if (state.activeTabId) {
    const active = getActiveTab();
    if (active) {
      setActiveTab(active.id);
    }
  }
}

export function handleActiveTabChanged(payload) {
  if (payload.id && payload.id !== state.activeTabId) {
    setActiveTab(payload.id);
  }
}

export function handleSessionAttached(payload) {
  const tab = findTab(payload.tabId);
  if (!tab || !payload.sessionId) return;

  if (!tab.session || tab.session.id !== payload.sessionId) {
    reconnectSession(tab, payload.sessionId);
  }
}

export function handleSessionDetached(payload) {
  const tab = findTab(payload.tabId);
  if (!tab) return;

  if (tab.session) {
    tab.session = null;
    setTabPhase(tab, "idle");
    setTabSocketState(tab, "disconnected");
  }
  tab.wasDetached = true;
  refreshTabButton(tab);
  queueSessionOverviewRefresh(250);
  updateSessionActions();
}

export async function getWorkspaceState() {
  try {
    const response = await proxyToApi("/api/v1/workspace");
    if (!response.ok) return null;
    return await response.json();
  } catch (error) {
    console.error("Failed to get workspace state:", error);
    return null;
  }
}
