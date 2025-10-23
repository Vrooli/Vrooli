import { appendEvent, showError } from "../event-feed.js";
import {
  transmitInputForTab,
  queueInputForTab,
  startSession,
} from "../session-service.js";
import { updateComposeFeedback } from "../composer.js";

/**
 * @param {{ tab: import("../types.d.ts").TerminalTab | null; value: string; appendNewline: boolean }} params
 * @returns {boolean}
 */
export function handleComposerSubmit({ tab, value, appendNewline }) {
  if (!tab) {
    updateComposeFeedback();
    return false;
  }

  const meta = {
    appendNewline,
    eventType: "composer-send",
    source: "composer",
  };

  let sent = false;
  if (tab.socket && tab.socket.readyState === WebSocket.OPEN) {
    sent = transmitInputForTab(tab, value, meta);
  } else {
    queueInputForTab(tab, value, meta);
    if (tab.phase === "idle" || tab.phase === "closed") {
      startSession(tab, { reason: "composer-input" }).catch((error) => {
        appendEvent(tab, "session-error", error);
        showError(
          tab,
          error instanceof Error
            ? error.message
            : "Unable to start terminal session",
        );
      });
    }
    sent = true;
  }

  if (!sent) {
    showError(tab, "Failed to send message to terminal");
  }

  return sent;
}
