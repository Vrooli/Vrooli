import { state } from "../state.js";
import {
  renderEventMeta,
  renderEvents,
  renderError,
} from "../event-feed.js";
import { updateSessionActions } from "../session-service.js";
import { renderSessionOverview } from "../session-overview.js";
import { updateDrawerIndicator, resetUnreadEvents } from "../drawer.js";
import { getActiveTab, renderTabs } from "../tab-manager.js";
import { syncActiveTabState, syncTabToWorkspace } from "../workspace.js";

/**
 * @param {import("../types.d.ts").TerminalTab | null} tab
 */
export function emitSessionUpdate(tab) {
  state.bridge?.emit("session-update", {
    tabId: tab?.id ?? null,
    phase: tab?.phase ?? "idle",
    socketState: tab?.socketState ?? "disconnected",
    session: tab?.session ?? null,
    transcriptSize: tab?.transcript?.length ?? 0,
    events: tab ? tab.events.slice(-20) : [],
    suppressed: tab ? { ...tab.suppressed } : {},
    tabs: state.tabs.map((entry) => ({
      id: entry.id,
      label: entry.label,
      phase: entry.phase,
      socketState: entry.socketState,
      hasSession: Boolean(entry.session),
    })),
  });
}

export function refreshActiveTabPanels() {
  const tab = getActiveTab();

  if (!tab) {
    renderEventMeta(null);
    renderEvents(null);
    renderError(null);
    emitSessionUpdate(null);
    updateDrawerIndicator();
    return;
  }

  renderEventMeta(tab);
  renderEvents(tab);
  renderError(tab);
  emitSessionUpdate(tab);
  updateDrawerIndicator();
}

export function updateUI() {
  refreshActiveTabPanels();
  updateSessionActions();
  renderSessionOverview();
}

/**
 * @param {import("../types.d.ts").TerminalTab | null} tab
 * @param {string | null} previousId
 */
export function handleActiveTabChanged(tab, previousId) {
  resetUnreadEvents();
  updateUI();
  syncActiveTabState();
}

/**
 * @param {import("../types.d.ts").TerminalTab} tab
 */
export function handleTabMetadataChanged(tab) {
  syncTabToWorkspace(tab);
  syncActiveTabState();
  renderTabs();
  updateUI();
}

/**
 * @param {string} [_reason]
 */
export function handleActiveTabPanelMutation(_reason) {
  refreshActiveTabPanels();
}
