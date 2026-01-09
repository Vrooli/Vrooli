// @ts-check

import {
  configureTabManager,
  setTerminalEventHandlers,
  initializeTabCustomizationUI,
  initializeShortcutButtons,
  getActiveTab,
  closeTabCustomization,
} from "../tab-manager.js";
import {
  configureSessionService,
  startSession,
  handleTerminalData,
  transmitInputForTab,
  queueInputForTab,
  sendResizeForTab,
  updateSessionActions,
  handleShortcutAction,
} from "../session-service.js";
import {
  configureSessionOverview,
  renderSessionOverview,
  queueSessionOverviewRefresh,
  startSessionOverviewWatcher,
} from "../session-overview.js";
import {
  configureWorkspace,
  initializeWorkspaceSettingsUI,
  initializeWorkspace,
} from "../workspace.js";
import {
  configureComposer,
  initializeComposerUI,
  updateComposeFeedback,
} from "../composer.js";
import {
  applyDrawerState,
  updateDrawerIndicator,
} from "../drawer.js";
import { configureEventFeed } from "../event-feed.js";
import {
  configureAIIntegration,
  initializeAIIntegration,
} from "../ai-integration.js";
import { initializeDiagnostics } from "../diagnostics.js";
import { initializeEventListeners } from "./event-handlers.js";
import {
  handleActiveTabChanged,
  handleTabMetadataChanged,
  updateUI,
  refreshActiveTabPanels,
  emitSessionUpdate,
  handleActiveTabPanelMutation,
} from "./ui-controller.js";
import {
  closeTab,
  handleSessionCloseFromOverview,
  focusSessionTab,
} from "./tab-controller.js";
import { handleComposerSubmit } from "./composer-controller.js";
import { initializeResourceTimingGuard } from "./resource-timing.js";
import { initializeDomElements } from "../state.js";
import {
  initializeTerminalLayoutWatcher,
  requestActiveTerminalFit,
} from "./terminal-layout.js";

export function bootstrapApp() {
  /** @type {Array<() => void>} */
  const cleanupTasks = [];
  initializeDomElements();

  configureSessionService({
    onActiveTabNeedsUpdate: refreshActiveTabPanels,
    onSessionActionsChanged: updateSessionActions,
    queueSessionOverviewRefresh,
  });

  configureTabManager({
    onActiveTabChanged: handleActiveTabChanged,
    onTabCloseRequested: /** @param {import("../types.d.ts").TerminalTab} tab */
    (tab) => closeTab(tab.id),
    onTabMetadataChanged: handleTabMetadataChanged,
    onShortcut: handleShortcutAction,
  });

  setTerminalEventHandlers({
    onResize: sendResizeForTab,
    /**
     * @param {import("../types.d.ts").TerminalTab} tab
     * @param {string} data
     */
    onData(tab, data) {
      handleTerminalData(tab, data);
    },
  });

  configureEventFeed({
    onActiveTabMutation: handleActiveTabPanelMutation,
  });

  configureSessionOverview({
    updateSessionActions,
    closeSessionById: handleSessionCloseFromOverview,
    focusSessionTab,
  });

  configureWorkspace({
    queueSessionOverviewRefresh,
  });

  configureComposer({
    onSubmit: handleComposerSubmit,
    onSubmitError: () => {},
    getActiveTab,
  });

  configureAIIntegration({
    getActiveTab,
    transmitInput: transmitInputForTab,
    queueInput: queueInputForTab,
    startSession,
  });

  initializeTabCustomizationUI();
  initializeShortcutButtons();
  initializeComposerUI();
  initializeAIIntegration();
  initializeDiagnostics();
  initializeResourceTimingGuard();

  const terminalLayoutTeardown = initializeTerminalLayoutWatcher({
    getActiveTab,
    notifyTabSized(tab) {
      const cols = typeof tab.term?.cols === "number" ? tab.term.cols : 0;
      const rows = typeof tab.term?.rows === "number" ? tab.term.rows : 0;
      if (cols > 0 && rows > 0) {
        sendResizeForTab(tab, cols, rows);
      }
    },
    onLayoutChanged: () => {
      closeTabCustomization();
    },
  });
  if (typeof terminalLayoutTeardown === "function") {
    cleanupTasks.push(terminalLayoutTeardown);
  }

  if (typeof document !== "undefined" && document.fonts) {
    const fontSet = document.fonts;
    const handleFontMetricsSettled = () => {
      try {
        requestActiveTerminalFit({ force: true });
      } catch (error) {
        console.warn("Failed to refit terminal after fonts loaded:", error);
      }
    };

    if (fontSet.ready && typeof fontSet.ready.then === "function") {
      fontSet.ready.then(handleFontMetricsSettled).catch(() => {});
    }

    if (
      typeof fontSet.addEventListener === "function" &&
      typeof fontSet.removeEventListener === "function"
    ) {
      fontSet.addEventListener("loadingdone", handleFontMetricsSettled);
      fontSet.addEventListener("loadingerror", handleFontMetricsSettled);
      cleanupTasks.push(() => {
        fontSet.removeEventListener("loadingdone", handleFontMetricsSettled);
        fontSet.removeEventListener("loadingerror", handleFontMetricsSettled);
      });
    }
  }

  applyDrawerState();
  updateDrawerIndicator();
  updateComposeFeedback();

  initializeEventListeners();
  initializeWorkspaceSettingsUI();
  initializeWorkspace();
  startSessionOverviewWatcher();
  updateSessionActions();
  renderSessionOverview();
  updateUI();

  return () => {
    while (cleanupTasks.length > 0) {
      const task = cleanupTasks.pop();
      try {
        task?.();
      } catch (error) {
        console.error("Failed to run cleanup task:", error);
      }
    }
  };
}
