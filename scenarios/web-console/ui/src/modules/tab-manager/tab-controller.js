// @ts-check

import {
  elements,
  state,
  TAB_COLOR_DEFAULT,
  TAB_COLOR_MAP,
  TAB_COLOR_OPTIONS,
} from "../state.js";
import { fitTerminal, TerminalFocusMode } from "../app/terminal-utils.js";
import { registerTabCustomizationHandlers } from "./customization.js";
import { tabCallbacks } from "./callbacks.js";
import { TAB_SESSION_STATE_META } from "./tab-helpers.js";

const COLOR_MAP = /** @type {Record<string, (typeof TAB_COLOR_OPTIONS)[number]>} */ (
  TAB_COLOR_MAP
);

/** @typedef {keyof typeof TAB_SESSION_STATE_META} TabSessionStateKey */

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * @param {TerminalTab | null | undefined} tab
 */
function focusTerminal(tab) {
  if (!tab) return;
  requestAnimationFrame(() => {
    try {
      fitTerminal(tab, { focusMode: TerminalFocusMode.FORCE });
    } catch (error) {
      console.warn("Failed to refresh terminal on focus:", error);
    }
  });
}

/**
 * Set the active tab by identifier.
 * @param {string} tabId
 */
export function setActiveTab(tabId) {
  const tab = findTab(tabId);
  if (!tab) {
    return;
  }
  const previousActiveId = state.activeTabId;
  state.activeTabId = tabId;
  /** @type {TerminalTab[]} */
  const tabs = state.tabs;
  tabs.forEach((entry) => {
    if (!entry.container) return;
    if (entry.id === tabId) {
      entry.container.classList.add("active");
    } else {
      entry.container.classList.remove("active");
    }
    updateTabButtonState(entry);
  });
  focusTerminal(tab);
  const { onActiveTabChanged } = tabCallbacks;
  if (typeof onActiveTabChanged === "function") {
    onActiveTabChanged(tab, previousActiveId);
  }
}

/**
 * @param {TerminalTab} tab
 */
function updateTabSessionIndicator(tab) {
  if (!tab?.domButton) return;
  if (!tab.domStatus) {
    tab.domStatus = tab.domButton.querySelector(".tab-status-icon");
  }

  const hasActiveSession = Boolean(
    tab.session && tab.wasDetached !== true && tab.phase !== "closed",
  );
  const isStarting = tab.phase === "creating";
  const isClosing = tab.phase === "closing";
  const isDetached =
    !hasActiveSession &&
    (tab.wasDetached || isDetachedTab(tab) || tab.phase === "closed");

  /** @type {TabSessionStateKey} */
  let sessionState = "attached";

  if (isStarting) {
    sessionState = "starting";
  } else if (isClosing) {
    sessionState = "closing";
  } else if (hasActiveSession) {
    sessionState = "attached";
  } else if (isDetached) {
    sessionState = "detached";
  } else {
    sessionState = "none";
  }

  const stateMeta =
    TAB_SESSION_STATE_META[sessionState] || TAB_SESSION_STATE_META.attached;
  tab.domButton.dataset.sessionState = sessionState;
  tab.domButton.dataset.sessionAttached = hasActiveSession ? "true" : "false";

  if (tab.domStatus) {
    tab.domStatus.classList.remove(
      "hidden",
      "tab-status-loading",
      "tab-status-closing",
      "tab-status-detached",
      "tab-status-empty",
    );
    tab.domStatus.textContent = "";
    tab.domStatus.innerHTML = "";

    if (stateMeta.icon) {
      const lucide = typeof window !== "undefined" ? window.lucide : null;
      const iconElement = document.createElement("i");
      iconElement.dataset.lucide = stateMeta.icon;
      iconElement.setAttribute("aria-hidden", "true");
      tab.domStatus.appendChild(iconElement);
      if (lucide && typeof lucide.createIcons === "function") {
        lucide.createIcons();
      }

      if (stateMeta.extraClass) {
        tab.domStatus.classList.add(stateMeta.extraClass);
      }
      tab.domStatus.classList.remove("hidden");

      const tooltip = stateMeta.label || "";
      if (tooltip) {
        tab.domStatus.title = tooltip;
      } else {
        tab.domStatus.removeAttribute("title");
      }
    } else {
      tab.domStatus.classList.add("hidden");
      tab.domStatus.removeAttribute("title");
    }
  }

  const baseLabel = tab.label || "";
  const stateLabel = stateMeta.label || "";
  if (baseLabel) {
    const titleText = stateLabel ? `${baseLabel} (${stateLabel})` : baseLabel;
    tab.domButton.title = titleText;
    tab.domButton.setAttribute(
      "aria-label",
      stateLabel ? `${baseLabel} – ${stateLabel}` : baseLabel,
    );
  } else {
    tab.domButton.title = stateLabel;
    if (stateLabel) {
      tab.domButton.setAttribute("aria-label", stateLabel);
    } else {
      tab.domButton.removeAttribute("aria-label");
    }
  }
}

/**
 * @param {TerminalTab} tab
 */
function updateTabButtonState(tab) {
  if (!tab.domButton) return;
  tab.domButton.setAttribute(
    "aria-selected",
    tab.id === state.activeTabId ? "true" : "false",
  );
  tab.domButton.dataset.phase = tab.phase;
  tab.domButton.dataset.socketState = tab.socketState;
  tab.domButton.classList.toggle("tab-running", tab.phase === "running");
  tab.domButton.classList.toggle("tab-error", tab.socketState === "error");
  if (tab.domClose) {
    tab.domClose.disabled = false;
    tab.domClose.removeAttribute("disabled");
  }
  updateTabSessionIndicator(tab);
  applyTabAppearance(tab);
}

/**
 * Update the tab button styling for a specific tab.
 * @param {TerminalTab} tab
 */
export function refreshTabButton(tab) {
  updateTabButtonState(tab);
}

/**
 * Apply color-driven appearance for a tab button.
 * @param {TerminalTab} tab
 */
export function applyTabAppearance(tab) {
  if (!tab?.domButton) return;
  if (!COLOR_MAP[tab.colorId]) {
    tab.colorId = TAB_COLOR_DEFAULT;
  }

  const swatch = COLOR_MAP[tab.colorId]?.styles;
  if (!swatch) return;

  tab.domButton.style.setProperty("--tab-border-color", swatch.border);
  tab.domButton.style.setProperty("--tab-hover", swatch.hover);
  tab.domButton.style.setProperty("--tab-active", swatch.active);
  tab.domButton.style.setProperty("--tab-selected-start", swatch.selectedStart);
  tab.domButton.style.setProperty("--tab-selected-end", swatch.selectedEnd);
  tab.domButton.style.setProperty("--tab-glow", swatch.glow);
  tab.domButton.style.setProperty("--tab-label", swatch.label);
}

/**
 * Render tab buttons in the tab list container.
 */
export function renderTabs() {
  const tabList = elements.tabList;
  if (!tabList) return;

  /** @type {TerminalTab[]} */
  const tabs = state.tabs;
  const knownIds = new Set(tabs.map((tab) => tab.id));

  Array.from(tabList.children).forEach((child) => {
    const tabId =
      child instanceof HTMLElement ? child.dataset.tabId : undefined;
    if (tabId && !knownIds.has(tabId)) {
      tabList.removeChild(child);
    }
  });

  tabs.forEach((tab) => {
    if (!tab.domItem) {
      const item = document.createElement("div");
      item.className = "tab-item";
      item.dataset.tabId = tab.id;

      const selectBtn = document.createElement("button");
      selectBtn.type = "button";
      selectBtn.className = "tab-button";
      selectBtn.dataset.tabId = tab.id;
      selectBtn.setAttribute("role", "tab");
      selectBtn.title = tab.label;
      const labelSpan = document.createElement("span");
      labelSpan.className = "tab-label";
      labelSpan.textContent = tab.label;
      selectBtn.appendChild(labelSpan);

      const statusSpan = document.createElement("span");
      statusSpan.className = "tab-status-icon hidden";
      statusSpan.setAttribute("aria-hidden", "true");
      selectBtn.appendChild(statusSpan);
      selectBtn.addEventListener("click", () => setActiveTab(tab.id));
      registerTabCustomizationHandlers(selectBtn, tab);

      const closeBtn = document.createElement("button");
      closeBtn.type = "button";
      closeBtn.className = "tab-close";
      closeBtn.dataset.tabId = tab.id;
      closeBtn.setAttribute("aria-label", `Close ${tab.label}`);
      closeBtn.textContent = "×";
      closeBtn.disabled = false;
      closeBtn.addEventListener("click", (event) => {
        event.stopPropagation();
        const { onTabCloseRequested } = tabCallbacks;
        if (typeof onTabCloseRequested === "function") {
          onTabCloseRequested(tab);
        }
      });

      item.appendChild(selectBtn);
      item.appendChild(closeBtn);

      const insertionPoint = elements.tabAddSlot || elements.addTabBtn || null;
      if (insertionPoint && tabList.contains(insertionPoint)) {
        tabList.insertBefore(item, insertionPoint);
      } else {
        tabList.appendChild(item);
      }

      tab.domItem = item;
      tab.domButton = selectBtn;
      tab.domClose = closeBtn;
      tab.domLabel = labelSpan;
      tab.domStatus = statusSpan;
    } else {
      const insertionPoint = elements.tabAddSlot || elements.addTabBtn || null;
      if (insertionPoint && tabList.contains(insertionPoint)) {
        tabList.insertBefore(tab.domItem, insertionPoint);
      } else {
        tabList.appendChild(tab.domItem);
      }
    }

    if (!tab.domStatus && tab.domButton) {
      tab.domStatus = tab.domButton.querySelector(".tab-status-icon");
    }

    if (tab.domLabel) {
      tab.domLabel.textContent = tab.label;
    } else if (tab.domButton) {
      tab.domButton.textContent = tab.label;
    }
    if (tab.domButton) {
      tab.domButton.title = tab.label;
    }
    if (tab.domClose) {
      tab.domClose.setAttribute("aria-label", `Close ${tab.label}`);
      tab.domClose.disabled = false;
      tab.domClose.removeAttribute("disabled");
    }
    applyTabAppearance(tab);
    updateTabButtonState(tab);
  });
}

/**
 * Locate the currently active tab.
 * @returns {TerminalTab | null}
 */
export function getActiveTab() {
  if (!state.activeTabId) return null;
  return state.tabs.find((tab) => tab.id === state.activeTabId) || null;
}

/**
 * Locate a tab by identifier.
 * @param {string} tabId
 * @returns {TerminalTab | null}
 */
export function findTab(tabId) {
  return state.tabs.find((tab) => tab.id === tabId) || null;
}

/**
 * Determine if a tab has been detached from any live session.
 * @param {TerminalTab | null | undefined} tab
 * @returns {boolean}
 */
export function isDetachedTab(tab) {
  if (!tab || tab.session) {
    return false;
  }
  if (tab.phase === "creating" || tab.phase === "running") {
    return false;
  }
  if (tab.wasDetached) {
    return true;
  }
  if (tab.phase === "closed") {
    return true;
  }
  if (tab.events && tab.events.length > 0) {
    return true;
  }
  if (
    tab.socketState !== "open" &&
    tab.socketState !== "connecting" &&
    tab.transcript &&
    tab.transcript.length > 0
  ) {
    return true;
  }
  return false;
}

/**
 * Return tabs that are currently detached from any running session.
 * @returns {TerminalTab[]}
 */
export function getDetachedTabs() {
  return state.tabs.filter((tab) => isDetachedTab(tab));
}

/**
 * Snapshot the current tab collection.
 * @returns {TerminalTab[]}
 */
export function getTabs() {
  return state.tabs.slice();
}
