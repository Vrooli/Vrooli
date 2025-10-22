import { state } from "../state.js";
import { refreshTabButton } from "../tab-manager.js";
import { notifyActiveTabUpdate } from "./callbacks.js";

export function setTabPhase(tab, phase) {
  if (!tab) return;
  tab.phase = phase;
  refreshTabButton(tab);
  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate();
  }
}

export function setTabSocketState(tab, socketState) {
  if (!tab) return;
  tab.socketState = socketState;
  refreshTabButton(tab);
  if (tab.id === state.activeTabId) {
    notifyActiveTabUpdate();
  }
}
