// @ts-check

import { elements, state } from "../state.js";
import {
  createTerminalTab,
  getActiveTab,
  closeTabCustomization,
} from "../tab-manager.js";
import {
  startSession,
  handleSignOutAllSessions,
  handleCloseDetachedTabs,
  reconnectSession,
} from "../session-service.js";
import { appendEvent, showError } from "../event-feed.js";
import { syncTabToWorkspace } from "../workspace.js";
import { refreshSessionOverview } from "../session-overview.js";
import { toggleDrawer, closeDrawer } from "../drawer.js";
import { isComposerOpen, closeComposeDialog } from "../composer.js";
import { initializeBridge } from "./bridge-controller.js";
import { requestActiveTerminalFit } from "./terminal-layout.js";
import { fitTerminal } from "./terminal-utils.js";

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

/**
 * @param {KeyboardEvent} event
 */
function handleGlobalKeyDown(event) {
  if (event.key !== "Escape") return;
  if (isComposerOpen()) {
    event.preventDefault();
    closeComposeDialog({ preserveValue: false });
    return;
  }
  if (state.tabMenu.open) {
    event.preventDefault();
    closeTabCustomization();
    return;
  }
  if (state.drawer.open) {
    event.preventDefault();
    closeDrawer();
  }
}

function handleWindowResize() {
  requestActiveTerminalFit();
  if (state.tabMenu.open) {
    closeTabCustomization();
  }
}

function handleVisibilityChange() {
  if (document.hidden) return;

  const activeTab = getActiveTab();
  if (activeTab && activeTab.term) {
    const forceRender = () => {
      try {
        fitTerminal(activeTab);
      } catch (error) {
        console.warn("Failed to refresh terminal on visibility change:", error);
      }
    };

    requestAnimationFrame(forceRender);
    setTimeout(forceRender, 100);
    setTimeout(forceRender, 250);
  }

  state.tabs.forEach((tab) => {
    if (tab.session && tab.phase === "running" && !tab.socket) {
      appendEvent(tab, "visibility-reconnect", { sessionId: tab.session.id });
      reconnectSession(tab, tab.session.id).catch((error) => {
        console.warn("Failed to reconnect on visibility change:", error);
      });
    }
  });
}

export function initializeEventListeners() {
  if (elements.addTabBtn) {
    elements.addTabBtn.addEventListener("click", async () => {
      const tab = /** @type {TerminalTab | null} */ (
        createTerminalTab({ focus: true })
      );
      if (tab) {
        await syncTabToWorkspace(tab);
        startSession(tab, { reason: "new-tab" }).catch((error) => {
          appendEvent(tab, "session-error", error);
          showError(
            tab,
            error instanceof Error
              ? error.message
              : "Unable to start terminal session",
          );
        });
      }
    });
  }

  if (elements.signOutAllSessions) {
    elements.signOutAllSessions.addEventListener(
      "click",
      handleSignOutAllSessions,
    );
  }

  if (elements.closeDetachedTabs) {
    elements.closeDetachedTabs.addEventListener(
      "click",
      handleCloseDetachedTabs,
    );
  }

  if (elements.sessionOverviewRefresh) {
    elements.sessionOverviewRefresh.addEventListener("click", () =>
      refreshSessionOverview({ silent: false })
    );
  }

  if (elements.drawerToggle) {
    elements.drawerToggle.addEventListener("click", () => toggleDrawer());
  }

  if (elements.drawerClose) {
    elements.drawerClose.addEventListener("click", () => closeDrawer());
  }

  if (elements.drawerBackdrop) {
    elements.drawerBackdrop.addEventListener("click", () => closeDrawer());
  }

  document.addEventListener("keydown", handleGlobalKeyDown);
  window.addEventListener("resize", handleWindowResize);
  document.addEventListener("visibilitychange", handleVisibilityChange);

  initializeBridge();
}

export function teardownEventListeners() {
  document.removeEventListener("keydown", handleGlobalKeyDown);
  window.removeEventListener("resize", handleWindowResize);
  document.removeEventListener("visibilitychange", handleVisibilityChange);
}
