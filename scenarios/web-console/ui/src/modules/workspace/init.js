// @ts-check

import { state } from "../state.js";
import { createTerminalTab, refreshTabButton } from "../tab-manager.js";
import { appendEvent, showError } from "../event-feed.js";
import { proxyToApi, scheduleMicrotask } from "../utils.js";
import { reconnectSession, startSession } from "../session-service.js";
import {
  setWorkspaceLoading,
  isWorkspaceLoading,
  applyIdleTimeoutFromServer,
} from "./idle-timeout.js";
import {
  sanitizeWorkspaceTabsFromServer,
  pruneDuplicateLocalTabs,
  reportWorkspaceAnomaly,
  ensureSessionPlaceholder,
  syncTabToWorkspace,
  syncActiveTabState,
  applyWorkspaceSnapshot,
} from "./tabs.js";
import { connectWorkspaceWebSocket } from "./socket.js";
import { debugWorkspace } from "./constants.js";

/** @typedef {import("../types.d.ts").TerminalTab} TerminalTab */

export async function initializeWorkspace() {
  setWorkspaceLoading(true);
  /** @type {Promise<unknown>[]} */
  const pendingReconnections = [];
  try {
    const response = await proxyToApi("/api/v1/workspace");
    if (!response.ok) {
      throw new Error(`Failed to load workspace: ${response.status}`);
    }
    const workspace = await response.json();

    applyIdleTimeoutFromServer(
      typeof workspace.idleTimeoutSeconds === "number"
        ? workspace.idleTimeoutSeconds
        : 0,
    );

    const rawTabs = Array.isArray(workspace.tabs) ? workspace.tabs : [];
    const {
      tabs: sanitizedTabs,
      duplicateIds,
      invalidIds,
    } = sanitizeWorkspaceTabsFromServer(rawTabs);
    const { removedIds: localDuplicates } = pruneDuplicateLocalTabs();

    if (
      duplicateIds.length > 0 ||
      invalidIds.length > 0 ||
      localDuplicates.length > 0
    ) {
      reportWorkspaceAnomaly({ duplicateIds, invalidIds, localDuplicates });
    }

    const requestedActiveId =
      typeof workspace.activeTabId === "string"
        ? workspace.activeTabId.trim()
        : null;

    if (sanitizedTabs.length > 0) {
      const resolvedTabs = applyWorkspaceSnapshot(sanitizedTabs, {
        activeTabId: requestedActiveId,
      });

      sanitizedTabs.forEach((tabMeta) => {
        const tab = resolvedTabs.get(tabMeta.id);
        if (!tab) {
          return;
        }
        const sessionId =
          typeof tabMeta.sessionId === "string" ? tabMeta.sessionId.trim() : "";
        if (!sessionId) {
          return;
        }
        ensureSessionPlaceholder(tab, sessionId);
        const reconnectPromise = reconnectSession(tab, sessionId).catch(() => {
          if (debugWorkspace) {
            console.log(
              `Session ${sessionId} no longer available for tab ${tabMeta.id}`,
            );
          }
          tab.session = null;
          refreshTabButton(tab);
        });
        pendingReconnections.push(reconnectPromise);
      });

      if (pendingReconnections.length > 0) {
        await Promise.allSettled(pendingReconnections);
      }
    } else {
      const initialTab = /** @type {TerminalTab | null} */ (
        createTerminalTab({ focus: true })
      );
      if (initialTab) {
        await syncTabToWorkspace(initialTab);
        startSession(initialTab, { reason: "initial-tab" }).catch((error) => {
          appendEvent(initialTab, "session-error", error);
          showError(
            initialTab,
            error instanceof Error
              ? error.message
              : "Unable to start terminal session",
          );
        });
      }
    }
  } catch (error) {
    console.error("Failed to initialize workspace:", error);
    applyIdleTimeoutFromServer(0);
    const initialTab = /** @type {TerminalTab | null} */ (
      createTerminalTab({ focus: true })
    );
    if (initialTab) {
      await syncTabToWorkspace(initialTab);
      startSession(initialTab, { reason: "initial-tab" }).catch(
        (startError) => {
          appendEvent(initialTab, "session-error", startError);
          showError(
            initialTab,
            startError instanceof Error
              ? startError.message
              : "Unable to start terminal session",
          );
        },
      );
    }
  } finally {
    setWorkspaceLoading(false);
    if (state.tabs.some((tab) => tab && tab.session && tab.session.id)) {
      scheduleMicrotask(() => {
        if (!isWorkspaceLoading()) {
          syncTabToWorkspaceState();
        }
      });
    }
    connectWorkspaceWebSocket();
  }
}

function syncTabToWorkspaceState() {
  try {
    syncActiveTabState();
  } catch (error) {
    console.error("Failed to sync active tab state:", error);
  }
}
