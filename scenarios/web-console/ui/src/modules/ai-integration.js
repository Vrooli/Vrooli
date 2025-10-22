import { elements } from "./state.js";

let getActiveTabFn = () => null;
let transmitInputFn = () => false;
let queueInputFn = () => {};
let startSessionFn = () => Promise.resolve();

export function configureAIIntegration({
  getActiveTab,
  transmitInput,
  queueInput,
  startSession,
}) {
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
      await import("./mobile-toolbar.js");

    const getActiveTabLocal = () => getActiveTabFn();

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

    const mobileToolbar = initializeMobileToolbar(
      getActiveTabLocal,
      sendKeyToTerminal,
    );

    if (elements.aiGenerateBtn && elements.aiCommandInput) {
      elements.aiGenerateBtn.addEventListener("click", async () => {
        const prompt = elements.aiCommandInput.value.trim();
        if (!prompt) return;

        const iconElement = elements.aiGenerateBtn.querySelector(".ai-icon");
        const originalIcon = iconElement ? iconElement.outerHTML : "";
        elements.aiGenerateBtn.disabled = true;
        elements.aiGenerateBtn.classList.add("loading");
        elements.aiGenerateBtn.innerHTML =
          '<span class="loading-spinner"></span>';

        try {
          const result = await generateAICommand(
            prompt,
            getActiveTabLocal,
            sendKeyToTerminal,
          );

          if (result.success) {
            elements.aiCommandInput.value = "";
            showSnackbar("Command generated successfully", "success", 2000);
          } else {
            showSnackbar(
              result.error || "Failed to generate command",
              "error",
              4000,
            );
          }
        } finally {
          elements.aiGenerateBtn.disabled = false;
          elements.aiGenerateBtn.classList.remove("loading");
          elements.aiGenerateBtn.innerHTML = originalIcon;
          if (window.lucide) {
            window.lucide.createIcons();
          }
        }
      });

      elements.aiCommandInput.addEventListener("keydown", (e) => {
        if (e.key === "Enter") {
          e.preventDefault();
          elements.aiGenerateBtn.click();
        }
      });
    }

    const debugToolbarToggle = document.getElementById("debugToolbarToggle");
    if (debugToolbarToggle && mobileToolbar) {
      debugToolbarToggle.addEventListener("change", (e) => {
        const isEnabled = e.target.checked;
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
