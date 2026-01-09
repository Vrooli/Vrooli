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

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */
/** @typedef {import("../types.d.ts").WorkspaceTabSnapshot} WorkspaceTabSnapshot */
/** @typedef {import("../types.d.ts").WorkspaceAnomalySummary} WorkspaceAnomalySummary */
/** @typedef {import("../types.d.ts").WorkspaceAnomalyReport} WorkspaceAnomalyReport */

/**
 * @param {TerminalTab} tab
 * @param {string} sessionId
 */
export function ensureSessionPlaceholder(tab, sessionId) {
  if (!tab || !sessionId) return;
  if (!tab.session || tab.session.id !== sessionId) {
    tab.session = { id: sessionId };
  }
  tab.wasDetached = true;
  refreshTabButton(tab);
}

/**
 * @param {unknown[]} rawTabs
 * @returns {WorkspaceAnomalySummary}
 */
export function sanitizeWorkspaceTabsFromServer(rawTabs) {
  const duplicates = /** @type {string[]} */ ([]);
  const invalid = /** @type {string[]} */ ([]);
  const seen = new Set();
  const sanitized = /** @type {WorkspaceTabSnapshot[]} */ ([]);

  rawTabs.forEach((entry) => {
    if (!entry || typeof entry !== "object") {
      return;
    }
    const record = /** @type {Record<string, unknown>} */ (entry);
    const rawId = typeof record.id === "string" ? record.id.trim() : "";
    const id = rawId;
    if (!id) {
      invalid.push(rawId || "(missing)");
      return;
    }
    if (seen.has(id)) {
      duplicates.push(id);
      return;
    }
    seen.add(id);
    const rawLabel =
      typeof record.label === "string" ? record.label.trim() : "";
    sanitized.push({
      id,
      label: rawLabel || `Terminal ${sanitized.length + 1}`,
      colorId: typeof record.colorId === "string" ? record.colorId : "sky",
      order: sanitized.length,
      sessionId:
        typeof record.sessionId === "string" && record.sessionId.trim()
          ? record.sessionId.trim()
          : null,
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
  const filtered = /** @type {TerminalTab[]} */ ([]);
  const removedIds = /** @type {string[]} */ ([]);

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

/**
 * @param {WorkspaceAnomalyReport} [report]
 */
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

const TAB_SYNC_DEBOUNCE_MS = 150;
const ACTIVE_SYNC_DEBOUNCE_MS = 180;

/**
 * @typedef {{
 *   timer: ReturnType<typeof setTimeout> | null;
 *   makePayload: () => (WorkspaceTabSnapshot & { sessionId: string | null }) | null;
 *   resolvers: Array<(value?: unknown) => void>;
 *   rejectors: Array<(reason?: unknown) => void>;
 * }} PendingTabSync
 */

/** @type {Map<string, PendingTabSync>} */
const pendingTabSyncs = new Map();

/**
 * @typedef {{
 *   timer: ReturnType<typeof setTimeout> | null;
 *   resolvers: Array<(value?: unknown) => void>;
 *   rejectors: Array<(reason?: unknown) => void>;
 * }} PendingActiveSync
 */

/** @type {PendingActiveSync | null} */
let pendingActiveSync = null;

/**
 * Align local tab state with a workspace snapshot and optionally apply the
 * requested active tab.
 * @param {WorkspaceTabSnapshot[]} snapshot
 * @param {{ activeTabId?: string | null }} [options]
 * @returns {Map<string, TerminalTab>}
 */
export function applyWorkspaceSnapshot(snapshot, { activeTabId } = {}) {
  const tabs = Array.isArray(snapshot) ? snapshot : [];
  const serverIds = new Set(tabs.map((tab) => tab.id));

  state.tabs.slice().forEach((tab) => {
    if (!tab) {
      return;
    }
    if (!serverIds.has(tab.id)) {
      handleTabRemoved({ id: tab.id });
    }
  });

  tabs.forEach((tabMeta) => {
    handleTabUpdated(tabMeta);
    const exists = state.tabs.some((tab) => tab.id === tabMeta.id);
    if (!exists) {
      handleTabAdded(tabMeta);
    }
  });

  const orderMap = new Map(tabs.map((tab, index) => [tab.id, index]));
  state.tabs.sort((a, b) => {
    const orderA = orderMap.get(a.id) ?? Number.MAX_SAFE_INTEGER;
    const orderB = orderMap.get(b.id) ?? Number.MAX_SAFE_INTEGER;
    return orderA - orderB;
  });
  renderTabs();

  const requestedActiveId =
    typeof activeTabId === "string" ? activeTabId.trim() : "";
  if (requestedActiveId && serverIds.has(requestedActiveId)) {
    handleActiveTabChanged({ id: requestedActiveId });
  } else if (state.tabs.length > 0) {
    setActiveTab(state.tabs[0].id);
  } else {
    state.activeTabId = null;
  }

  const resolvedTabs = new Map();
  tabs.forEach((tabMeta) => {
    const tab = findTab(tabMeta.id);
    if (tab) {
      resolvedTabs.set(tabMeta.id, tab);
    }
  });

  return resolvedTabs;
}

/**
 * @param {TerminalTab} tab
 */
export function syncTabToWorkspace(tab) {
  if (!tab) {
    return Promise.resolve();
  }

  return scheduleTabSync(tab.id, () => {
    const current = state.tabs.find((entry) => entry.id === tab.id) || tab;
    if (!current) {
      return null;
    }
    const order = state.tabs.indexOf(current);
    return {
      id: current.id,
      label: current.label,
      colorId: current.colorId,
      order: order >= 0 ? order : 0,
      sessionId: current.session && current.session.id ? current.session.id : null,
    };
  });
}

/**
 * @param {string} tabId
 * @param {() => (WorkspaceTabSnapshot & { sessionId: string | null }) | null} makePayload
 */
function scheduleTabSync(tabId, makePayload) {
  let entry = pendingTabSyncs.get(tabId);
  if (entry) {
    if (entry.timer) {
      clearTimeout(entry.timer);
    }
    entry.makePayload = makePayload;
  } else {
    entry = {
      timer: null,
      makePayload,
      resolvers: [],
      rejectors: [],
    };
    pendingTabSyncs.set(tabId, entry);
  }

  const promise = new Promise((resolve, reject) => {
    entry.resolvers.push(resolve);
    entry.rejectors.push(reject);
  });

  entry.timer = setTimeout(async () => {
    pendingTabSyncs.delete(tabId);
    try {
      const payload = entry.makePayload();
      if (payload) {
        await proxyToApi("/api/v1/workspace/tabs", {
          method: "POST",
          json: payload,
        });
      }
      entry.resolvers.forEach((resolve) => resolve());
    } catch (error) {
      console.error("Failed to sync tab to workspace:", error);
      entry.rejectors.forEach((reject) => reject(error));
    }
  }, TAB_SYNC_DEBOUNCE_MS);

  return promise;
}

/**
 * @param {string} tabId
 */
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
  if (pendingActiveSync && pendingActiveSync.timer) {
    clearTimeout(pendingActiveSync.timer);
  } else if (!pendingActiveSync) {
    pendingActiveSync = { timer: null, resolvers: [], rejectors: [] };
  }

  const pending = /** @type {PendingActiveSync} */ (pendingActiveSync);
  const promise = new Promise((resolve, reject) => {
    pending.resolvers.push(resolve);
    pending.rejectors.push(reject);
  });

  pending.timer = setTimeout(() => {
    flushActiveTabSync(pending);
  }, ACTIVE_SYNC_DEBOUNCE_MS);

  return promise;
}

/**
 * @param {PendingActiveSync} pending
 */
async function flushActiveTabSync(pending) {
  pendingActiveSync = null;
  try {
    const { tabs: uniqueTabs, removedIds } = pruneDuplicateLocalTabs();
    if (removedIds.length > 0) {
      reportWorkspaceAnomaly({ localDuplicates: removedIds });
    }

    let activeId = state.activeTabId;
    if (activeId && !uniqueTabs.some((tab) => tab.id === activeId)) {
      activeId = uniqueTabs.length > 0 ? uniqueTabs[0].id : null;
      state.activeTabId = activeId;
    }

    await proxyToApi("/api/v1/workspace", {
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
    });

    pending.resolvers.forEach((resolve) => resolve());
  } catch (error) {
    console.error("Failed to sync active tab to server:", error);
    pending.rejectors.forEach((reject) => reject(error));
  }
}

/**
 * @param {WorkspaceTabSnapshot} payload
 */
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

/**
 * @param {WorkspaceTabSnapshot} payload
 */
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

/**
 * @param {{ id: string }} payload
 */
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

/**
 * @param {{ id?: string | null }} payload
 */
export function handleActiveTabChanged(payload) {
  if (payload.id && payload.id !== state.activeTabId) {
    setActiveTab(payload.id);
  }
}

/**
 * @param {{ tabId: string; sessionId: string }} payload
 */
export function handleSessionAttached(payload) {
  const tab = findTab(payload.tabId);
  if (!tab || !payload.sessionId) return;

  if (!tab.session || tab.session.id !== payload.sessionId) {
    reconnectSession(tab, payload.sessionId);
  }
}

/**
 * @param {{ tabId: string }} payload
 */
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
