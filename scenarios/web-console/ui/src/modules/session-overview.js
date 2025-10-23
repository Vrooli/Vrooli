import { elements, state } from "./state.js";
import { getDetachedTabs, getActiveTab } from "./tab-manager.js";
import {
  proxyToApi,
  formatRelativeTimestamp,
  formatAbsoluteTime,
} from "./utils.js";

let updateActionsCallback = null;
let closeSessionCallback = null;
let focusTabCallback = null;
const relativeTimeFormatter =
  typeof Intl !== "undefined" && typeof Intl.RelativeTimeFormat === "function"
    ? new Intl.RelativeTimeFormat(undefined, { numeric: "auto" })
    : null;
let listHandlersBound = false;

/**
 * @param {{
 *  updateSessionActions?: () => void;
 *  closeSessionById?: (id: string) => Promise<boolean> | boolean;
 *  focusSessionTab?: (tabId: string) => void;
 * }} [options]
 */
export function configureSessionOverview({
  updateSessionActions,
  closeSessionById,
  focusSessionTab,
} = {}) {
  updateActionsCallback =
    typeof updateSessionActions === "function" ? updateSessionActions : null;
  closeSessionCallback =
    typeof closeSessionById === "function" ? closeSessionById : null;
  focusTabCallback =
    typeof focusSessionTab === "function" ? focusSessionTab : null;
  bindSessionOverviewEvents();
}

export function setSessionOverviewLoading(isLoading) {
  const button = elements.sessionOverviewRefresh;
  if (button) {
    button.disabled = Boolean(isLoading);
    button.classList.toggle("loading", Boolean(isLoading));
    if (isLoading) {
      button.setAttribute("aria-busy", "true");
    } else {
      button.removeAttribute("aria-busy");
    }
  }
}

export function queueSessionOverviewRefresh(delay = 0) {
  if (typeof window === "undefined") return;
  if (state.sessions.refreshTimer) {
    clearTimeout(state.sessions.refreshTimer);
    state.sessions.refreshTimer = null;
  }
  state.sessions.refreshTimer = window.setTimeout(
    () => {
      state.sessions.refreshTimer = null;
      refreshSessionOverview({ silent: true });
    },
    Math.max(0, delay),
  );
}

export async function refreshSessionOverview(options = {}) {
  if (state.sessions.loading) {
    state.sessions.needsRefresh = true;
    return;
  }

  state.sessions.loading = true;
  state.sessions.error = null;
  setSessionOverviewLoading(true);
  renderSessionOverview();

  try {
    const response = await proxyToApi("/api/v1/sessions");
    if (!response.ok) {
      const text = await response.text();
      throw new Error(text || `API error (${response.status})`);
    }
    let data;
    try {
      data = await response.json();
    } catch (_error) {
      data = [];
    }

    if (Array.isArray(data)) {
      state.sessions.items = data;
    } else if (data && typeof data === "object") {
      const list = Array.isArray(data.sessions) ? data.sessions : [];
      state.sessions.items = list;
      if (
        typeof data.capacity === "number" &&
        Number.isFinite(data.capacity) &&
        data.capacity > 0
      ) {
        state.sessions.capacity = data.capacity;
      }
    } else {
      state.sessions.items = [];
    }
    state.sessions.lastFetched = Date.now();
    let capacityHeader = response.headers.get("x-session-capacity");
    if (!capacityHeader) {
      capacityHeader = response.headers.get("X-Session-Capacity");
    }
    if (capacityHeader) {
      const parsedCapacity = parseInt(capacityHeader, 10);
      if (Number.isFinite(parsedCapacity) && parsedCapacity > 0) {
        state.sessions.capacity = parsedCapacity;
      }
    }
  } catch (error) {
    state.sessions.error =
      error instanceof Error ? error : new Error(String(error));
    if (options.silent !== true) {
      console.error("Failed to refresh session overview:", error);
    }
  } finally {
    state.sessions.loading = false;
    setSessionOverviewLoading(false);
    renderSessionOverview();
    if (state.sessions.needsRefresh) {
      state.sessions.needsRefresh = false;
      queueSessionOverviewRefresh(50);
    }
  }
}

export function startSessionOverviewWatcher() {
  if (typeof window === "undefined") return;
  if (state.sessions.pollHandle) {
    clearInterval(state.sessions.pollHandle);
    state.sessions.pollHandle = null;
  }
  refreshSessionOverview({ silent: true });
  state.sessions.pollHandle = window.setInterval(() => {
    if (typeof document !== "undefined" && document.hidden) {
      return;
    }
    refreshSessionOverview({ silent: true });
  }, 20000);
}

export function stopSessionOverviewWatcher() {
  if (state.sessions.pollHandle) {
    clearInterval(state.sessions.pollHandle);
    state.sessions.pollHandle = null;
  }
  if (state.sessions.refreshTimer) {
    clearTimeout(state.sessions.refreshTimer);
    state.sessions.refreshTimer = null;
  }
  state.sessions.needsRefresh = false;
}

if (typeof window !== "undefined") {
  window.addEventListener("beforeunload", () => {
    stopSessionOverviewWatcher();
  });
}

function bindSessionOverviewEvents() {
  if (listHandlersBound) return;
  const listEl = elements.sessionOverviewList;
  if (!listEl) return;
  listEl.addEventListener("click", handleSessionOverviewClick);
  listHandlersBound = true;
}

function handleSessionOverviewClick(event) {
  const target = event.target;
  if (!(target instanceof HTMLElement)) {
    return;
  }

  const closeButton = target.closest('button[data-session-close="true"]');
  if (closeButton instanceof HTMLButtonElement) {
    const sessionId = closeButton.dataset.sessionId || "";
    if (!sessionId || closeButton.dataset.busy === "true") {
      event.stopPropagation();
      event.preventDefault();
      return;
    }

    closeButton.dataset.busy = "true";
    closeButton.disabled = true;
    closeButton.setAttribute("aria-busy", "true");

    const maybePromise = closeSessionCallback
      ? closeSessionCallback(sessionId)
      : null;
    Promise.resolve(maybePromise)
      .catch((error) => {
        console.error("Failed to close session from overview:", error);
      })
      .finally(() => {
        if (closeButton.isConnected) {
          delete closeButton.dataset.busy;
          closeButton.disabled = false;
          closeButton.removeAttribute("aria-busy");
        }
      });

    event.stopPropagation();
    event.preventDefault();
    return;
  }

  const item = target.closest("li[data-session-id]");
  if (!(item instanceof HTMLElement)) {
    return;
  }

  const tabId = item.dataset.tabId || "";
  if (!tabId || !focusTabCallback) {
    return;
  }

  focusTabCallback(tabId, item.dataset.sessionId || "");
}

const SESSION_ITEM_PARTS = Symbol("sessionOverviewItemParts");

/**
 * @typedef {{
 *   id: string;
 *   shortId: string;
 *   statusText: string;
 *   status: "attached" | "orphaned";
 *   tabId: string;
 *   isClickable: boolean;
 *   isActive: boolean;
 *   commandLine: string;
 *   timestamp: string;
 *   isClosable: boolean;
 *   closeLabel: string;
 * }} SessionOverviewEntry
 */
function getSessionItemParts(li) {
  return /** @type {{ idSpan: HTMLSpanElement; statusSpan: HTMLSpanElement; closeButton: HTMLButtonElement; commandLine: HTMLDivElement | null; timestamp: HTMLDivElement | null }} */ (
    li[SESSION_ITEM_PARTS]
  );
}

function updateSessionOverviewItem(li, entry) {
  const parts = getSessionItemParts(li);

  li.dataset.sessionId = entry.id;
  parts.idSpan.textContent = entry.shortId;
  parts.idSpan.title = entry.id;

  if (entry.isClickable) {
    li.dataset.tabId = entry.tabId;
    li.classList.add('session-overview-clickable');
  } else {
    li.removeAttribute('data-tab-id');
    li.classList.remove('session-overview-clickable');
  }

  if (entry.isActive) {
    li.dataset.active = 'true';
  } else {
    delete li.dataset.active;
  }

  li.dataset.sessionState = entry.status;
  parts.statusSpan.textContent = entry.statusText;

  parts.closeButton.dataset.sessionId = entry.id;
  parts.closeButton.setAttribute('aria-label', entry.closeLabel);
  parts.closeButton.disabled = !entry.isClosable;

  if (entry.commandLine) {
    if (!parts.commandLine) {
      parts.commandLine = document.createElement('div');
      parts.commandLine.className = 'session-overview-command';
      li.appendChild(parts.commandLine);
    }
    parts.commandLine.textContent = entry.commandLine;
  } else if (parts.commandLine) {
    parts.commandLine.remove();
    parts.commandLine = null;
  }

  if (entry.timestamp) {
    if (!parts.timestamp) {
      parts.timestamp = document.createElement('div');
      parts.timestamp.className = 'session-overview-timestamp';
      li.appendChild(parts.timestamp);
    }
    parts.timestamp.textContent = entry.timestamp;
  } else if (parts.timestamp) {
    parts.timestamp.remove();
    parts.timestamp = null;
  }
}

function createSessionOverviewItem(entry) {
  const li = document.createElement('li');
  li.className = 'session-overview-item';

  const header = document.createElement('div');
  header.className = 'session-overview-row';
  li.appendChild(header);

  const idSpan = document.createElement('span');
  idSpan.className = 'session-overview-id';
  header.appendChild(idSpan);

  const controls = document.createElement('div');
  controls.className = 'session-overview-controls';
  header.appendChild(controls);

  const statusSpan = document.createElement('span');
  statusSpan.className = 'session-overview-status';
  controls.appendChild(statusSpan);

  const closeButton = document.createElement('button');
  closeButton.type = 'button';
  closeButton.className = 'session-overview-close';
  closeButton.dataset.sessionClose = 'true';
  closeButton.textContent = '\u00D7';
  controls.appendChild(closeButton);

  li[SESSION_ITEM_PARTS] = {
    idSpan,
    statusSpan,
    closeButton,
    commandLine: /** @type {HTMLDivElement | null} */ (null),
    timestamp: /** @type {HTMLDivElement | null} */ (null),
  };

  updateSessionOverviewItem(li, entry);
  return li;
}

function buildTimestampLabel(value) {
  if (!value) return '';
  const relative = formatRelativeTimestamp(value, relativeTimeFormatter);
  const absolute = formatAbsoluteTime(value);
  if (relative && absolute) return `Last active ${relative} (${absolute})`;
  if (relative) return `Last active ${relative}`;
  if (absolute) return `Last active ${absolute}`;
  return '';
}

function buildCommandLabel(command, args) {
  if (!command) return '';
  const parts = [command];
  if (Array.isArray(args) && args.length > 0) {
    parts.push(...args);
  }
  return parts.join(' ');
}

function getCapacityValue() {
  if (
    typeof state.sessions.capacity === 'number' &&
    Number.isFinite(state.sessions.capacity) &&
    state.sessions.capacity > 0
  ) {
    return state.sessions.capacity;
  }
  return null;
}

function buildSessionOverviewEntries() {
  const sessionMap = new Map();
  const apiSessions = Array.isArray(state.sessions.items)
    ? state.sessions.items.slice()
    : [];

  apiSessions.forEach((session) => {
    if (!session || typeof session.id !== 'string') return;
    const id = session.id.trim();
    if (!id) return;
    sessionMap.set(id, { ...session });
  });

  state.tabs.forEach((tab) => {
    if (!tab || !tab.session) return;
    const tabSession = tab.session;
    const id = typeof tabSession.id === 'string' ? tabSession.id.trim() : '';
    if (!id || sessionMap.has(id)) return;
    sessionMap.set(id, {
      id,
      createdAt: tabSession.createdAt || null,
      expiresAt: tabSession.expiresAt || null,
      lastActivity:
        tabSession.lastActivity ||
        tabSession.updatedAt ||
        tabSession.createdAt ||
        null,
      state: tab.phase === 'closed' ? 'closed' : 'active',
      command: tabSession.command || '',
      args: Array.isArray(tabSession.args) ? [...tabSession.args] : [],
    });
  });

  const tabBySessionId = new Map();
  state.tabs.forEach((tab) => {
    if (tab.session && tab.session.id) {
      tabBySessionId.set(tab.session.id, tab);
    }
  });

  const activeTabId = state.activeTabId;
  const activeSessionId = getActiveTab()?.session?.id || null;
  const sessions = Array.from(sessionMap.values());
  sessions.sort((a, b) => {
    const aTime = a && a.lastActivity ? Date.parse(a.lastActivity) : 0;
    const bTime = b && b.lastActivity ? Date.parse(b.lastActivity) : 0;
    return bTime - aTime;
  });

  const entries = /** @type {SessionOverviewEntry[]} */ ([]);
  let orphanCount = 0;

  sessions.forEach((session) => {
    if (!session || typeof session.id !== 'string') {
      return;
    }
    const sessionId = session.id;
    const attachedTab = tabBySessionId.get(sessionId) || null;
    const tabLabel = attachedTab
      ? attachedTab.label || attachedTab.defaultLabel || attachedTab.id
      : '';
    const isAttached = Boolean(attachedTab);
    const isActiveTab = attachedTab ? attachedTab.id === activeTabId : false;
    const isActiveSession = activeSessionId ? sessionId === activeSessionId : false;
    const statusText = isAttached
      ? isActiveTab
        ? 'Active tab'
        : `Attached (${tabLabel})`
      : 'No tab attached';
    const status = isAttached ? 'attached' : 'orphaned';
    if (!isAttached) {
      orphanCount += 1;
    }

    entries.push({
      id: sessionId,
      shortId: sessionId.slice(0, 8) || sessionId,
      statusText,
      status,
      tabId: attachedTab?.id || '',
      isClickable: isAttached,
      isActive: isActiveSession,
      commandLine: buildCommandLabel(session.command, session.args),
      timestamp: buildTimestampLabel(session.lastActivity),
      isClosable: session.state !== 'closed',
      closeLabel: `Sign out session ${sessionId.slice(0, 8)}`,
    });
  });

  return {
    entries,
    orphanCount,
    detachedTabs: getDetachedTabs(),
    capacityValue: getCapacityValue(),
    loading: state.sessions.loading,
    error: state.sessions.error,
    lastFetched: state.sessions.lastFetched,
  };
}

export function renderSessionOverview() {
  const listEl = elements.sessionOverviewList;
  if (!listEl) return;
  bindSessionOverviewEvents();

  const {
    entries,
    orphanCount,
    detachedTabs,
    capacityValue,
    loading,
    error,
    lastFetched,
  } = buildSessionOverviewEntries();

  const countElement = elements.sessionOverviewCount;
  if (countElement) {
    const base = loading ? '…' : error ? '!' : String(entries.length);
    const hasCapacity = capacityValue !== null;
    const displayCapacity = hasCapacity ? capacityValue : '—';
    countElement.textContent = `${base} of ${displayCapacity}`;

    let label;
    if (loading) {
      label = hasCapacity
        ? `Loading active sessions (capacity ${capacityValue})`
        : 'Loading active sessions (capacity unknown)';
    } else if (error) {
      label = hasCapacity
        ? `Unable to load active sessions (capacity ${capacityValue})`
        : 'Unable to load active sessions (capacity unknown)';
    } else {
      label = hasCapacity
        ? `${entries.length} of ${capacityValue} sessions active`
        : `${entries.length} sessions active (capacity unknown)`;
    }
    countElement.setAttribute('aria-label', label);
    countElement.title = label;
  }

  const preserveExistingList = (loading || error) && listEl.childElementCount > 0;
  if (!preserveExistingList) {
    const existing = new Map();
    Array.from(listEl.children).forEach((child) => {
      if (child instanceof HTMLLIElement && child.dataset.sessionId) {
        existing.set(child.dataset.sessionId, child);
      }
    });

    const fragment = document.createDocumentFragment();
    entries.forEach((entry) => {
      let item = existing.get(entry.id) || null;
      if (item) {
        existing.delete(entry.id);
        if (!item[SESSION_ITEM_PARTS]) {
          item.remove();
          item = null;
        }
      }
      if (item) {
        updateSessionOverviewItem(item, entry);
      } else {
        item = createSessionOverviewItem(entry);
      }
      fragment.appendChild(item);
    });

    existing.forEach((li) => li.remove());
    listEl.replaceChildren(fragment);
  }

  const emptyEl = elements.sessionOverviewEmpty;
  if (emptyEl) {
    if (!loading && !error && entries.length === 0) {
      emptyEl.classList.remove('hidden');
    } else {
      emptyEl.classList.add('hidden');
    }
  }

  const alertEl = elements.sessionOverviewAlert;
  if (alertEl) {
    const messageSpan = alertEl.querySelector('span');
    if (messageSpan) {
      if (orphanCount <= 0) {
        messageSpan.textContent = 'Unmapped sessions detected. Use “Sign out all sessions” to clean up.';
      } else if (orphanCount === 1) {
        messageSpan.textContent = '1 session is not mapped to a tab. Use “Sign out all sessions” to clean up.';
      } else {
        messageSpan.textContent = `${orphanCount} sessions are not mapped to tabs. Use “Sign out all sessions” to clean up.`;
      }
    }
    if (!loading && !error && orphanCount > 0) {
      alertEl.classList.remove('hidden');
    } else {
      alertEl.classList.add('hidden');
    }
  }

  const detachedMetaEl = elements.detachedTabsMeta;
  if (detachedMetaEl) {
    if (detachedTabs.length > 0) {
      const labels = detachedTabs.map(
        (tab) => tab.label || tab.defaultLabel || tab.id,
      );
      const joined = labels.slice(0, 3).join(', ');
      const remaining = labels.length > 3 ? `, +${labels.length - 3} more` : '';
      const label =
        detachedTabs.length === 1
          ? `1 tab is open without an active session (${labels[0]}).`
          : `${detachedTabs.length} tabs are open without active sessions (${joined}${remaining}).`;
      detachedMetaEl.textContent = `${label} Use “Close detached tabs” to clean up.`;
      detachedMetaEl.classList.remove('hidden');
    } else {
      detachedMetaEl.textContent = '';
      detachedMetaEl.classList.add('hidden');
    }
  }

  const metaEl = elements.sessionOverviewMeta;
  if (metaEl) {
    metaEl.classList.toggle('error', Boolean(error));
    const capacityText = capacityValue ? `Capacity ${capacityValue}` : null;
    if (error) {
      const errValue =
        error && typeof error === 'object' && 'message' in error
          ? String(/** @type {{ message?: unknown }} */ (error).message ?? '') || 'Unknown error'
          : String(error);
      const segments = [`Unable to load sessions: ${errValue}`];
      segments.push(capacityText || 'Capacity unknown');
      metaEl.textContent = segments.join(' • ');
    } else if (loading) {
      metaEl.textContent = `Loading sessions… • ${capacityText || 'Capacity unknown'}`;
    } else if (lastFetched) {
      const relative = formatRelativeTimestamp(lastFetched, relativeTimeFormatter);
      const absolute = formatAbsoluteTime(lastFetched);
      const segments = [];
      if (relative && absolute) {
        segments.push(`Updated ${relative} (${absolute})`);
      } else if (relative) {
        segments.push(`Updated ${relative}`);
      } else if (absolute) {
        segments.push(`Updated ${absolute}`);
      }
      segments.push(capacityText || 'Capacity unknown');
      metaEl.textContent = segments.join(' • ');
    } else {
      metaEl.textContent = capacityText || 'Capacity unknown';
    }
    metaEl.classList.toggle('hidden', metaEl.textContent === '');
  }

  updateActionsCallback?.();
}
