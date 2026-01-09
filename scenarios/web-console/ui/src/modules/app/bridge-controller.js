import { state } from "../state.js";
import { initializeIframeBridge } from "../bridge.js";
import { getActiveTab } from "../tab-manager.js";
import { stopSession } from "../session-service.js";

export function initializeBridge() {
  state.bridge = initializeIframeBridge({
    "init-session": () => {
      state.bridge?.emit("ready", { timestamp: Date.now() });
    },
    "end-session": async () => {
      const tab = getActiveTab();
      if (tab) {
        await stopSession(tab);
      }
      state.bridge?.emit("session-ended", { requestedByParent: true });
    },
    "request-screenshot": async () => {
      if (window.html2canvas) {
        const canvas = await window.html2canvas(document.body, {
          backgroundColor: "#0f172a",
        });
        state.bridge?.emit("screenshot", {
          image: canvas.toDataURL("image/png", 0.9),
          requestedAt: Date.now(),
        });
      } else {
        state.bridge?.emit("error", {
          type: "request-screenshot",
          message: "html2canvas not available",
        });
      }
    },
    "request-transcript": () => {
      const tab = getActiveTab();
      state.bridge?.emit("transcript", {
        transcript: tab ? tab.transcript : [],
        requestedAt: Date.now(),
        tabId: tab?.id ?? null,
      });
    },
    "request-logs": () => {
      const tab = getActiveTab();
      state.bridge?.emit("logs", {
        logs: tab ? tab.events.slice(-100) : [],
        requestedAt: Date.now(),
        tabId: tab?.id ?? null,
      });
    },
  });
}
