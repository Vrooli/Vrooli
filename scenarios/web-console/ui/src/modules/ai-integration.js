import { elements } from "./state.js";

/** @typedef {import("./types.d.ts").TerminalTab} TerminalTab */

/** @type {() => TerminalTab | null} */
let getActiveTabFn = () => null;
/** @type {(tab: TerminalTab, value: string, meta?: Record<string, unknown>) => boolean} */
let transmitInputFn = () => false;
/** @type {(tab: TerminalTab, value: string, meta?: Record<string, unknown>) => void} */
let queueInputFn = () => {};
/** @type {(tab: TerminalTab, options?: Record<string, unknown>) => Promise<unknown>} */
let startSessionFn = () => Promise.resolve();

/**
 * @param {{
 *  getActiveTab?: () => TerminalTab | null;
 *  transmitInput?: (tab: TerminalTab, value: string, meta?: Record<string, unknown>) => boolean;
 *  queueInput?: (tab: TerminalTab, value: string, meta?: Record<string, unknown>) => void;
 *  startSession?: (tab: TerminalTab, options?: Record<string, unknown>) => Promise<unknown>;
 * }} [options]
 */
export function configureAIIntegration({
  getActiveTab,
  transmitInput,
  queueInput,
  startSession,
} = {}) {
  getActiveTabFn =
    typeof getActiveTab === "function" ? getActiveTab : () => null;
  transmitInputFn =
    typeof transmitInput === "function" ? transmitInput : () => false;
  queueInputFn = typeof queueInput === "function" ? queueInput : () => {};
  startSessionFn =
    typeof startSession === "function" ? startSession : () => Promise.resolve();
}

export function initializeAIIntegration() {
  if (
    typeof window !== "undefined" &&
    (document.readyState === "complete" ||
      document.readyState === "interactive")
  ) {
    initAICommandAsync();
  } else {
    window.addEventListener("DOMContentLoaded", initAICommandAsync);
  }
}

async function initAICommandAsync() {
  try {
    const { generateAICommand, showSnackbar, initializeMobileToolbar } =
      await import("./mobile-toolbar/index.js");

    const getActiveTabLocal = () => getActiveTabFn();

    /**
     * @param {string} key
     */
    const sendKeyToTerminal = (key) => {
      const tab = getActiveTabFn();
      if (!tab) {
        console.warn("No active terminal to send key to");
        return;
      }

      if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
        transmitInputFn(tab, key, { appendNewline: false, clearError: true });
      } else {
        queueInputFn(tab, key, { appendNewline: false });
        if (tab.phase === "idle" || tab.phase === "closed") {
          startSessionFn(tab, { reason: "ai-command-input" }).catch((error) => {
            console.error("Unable to start terminal session", error);
            showSnackbar("Unable to start terminal session", "error", 4000);
          });
        }
      }
    };

    /** @type {any} */
    const mobileToolbar = initializeMobileToolbar(
      getActiveTabLocal,
      sendKeyToTerminal,
    );

    const generateBtn =
      elements.aiGenerateBtn instanceof HTMLButtonElement
        ? elements.aiGenerateBtn
        : null;
    const commandInput =
      elements.aiCommandInput instanceof HTMLInputElement
        ? elements.aiCommandInput
        : null;

    if (generateBtn && commandInput) {
      generateBtn.addEventListener("click", async () => {
        const prompt = commandInput.value.trim();
        if (!prompt) return;

        const iconElement = generateBtn.querySelector(".ai-icon");
        const originalIcon = iconElement ? iconElement.outerHTML : "";
        generateBtn.disabled = true;
        generateBtn.classList.add("loading");
        generateBtn.innerHTML = '<span class="loading-spinner"></span>';

        try {
          const result = await generateAICommand(
            prompt,
            getActiveTabLocal,
            sendKeyToTerminal,
          );

          if (result.success) {
            commandInput.value = "";
            showSnackbar("Command generated successfully", "success", 2000);
          } else {
            showSnackbar(
              result.error || "Failed to generate command",
              "error",
              4000,
            );
          }
        } finally {
          generateBtn.disabled = false;
          generateBtn.classList.remove("loading");
          generateBtn.innerHTML = originalIcon;
          if (window.lucide) {
            window.lucide.createIcons();
          }
        }
      });

      commandInput.addEventListener("keydown", (e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          generateBtn.click();
        }
      });
    }

    const debugToolbarToggle = document.getElementById("debugToolbarToggle");
    if (debugToolbarToggle instanceof HTMLInputElement && mobileToolbar) {
      debugToolbarToggle.addEventListener("change", (e) => {
        const target = e.target instanceof HTMLInputElement ? e.target : null;
        const isEnabled = target?.checked ?? false;
        localStorage.setItem("debugToolbarEnabled", isEnabled.toString());

        if (isEnabled) {
          mobileToolbar.setMode("floating", true);
          mobileToolbar.show();
        } else {
          mobileToolbar.setMode("disabled", false);
          mobileToolbar.hide();
        }
      });
    }
  } catch (error) {
    console.error("Failed to initialize AI command generation:", error);
  }
}
