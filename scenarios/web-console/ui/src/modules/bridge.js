export function initializeIframeBridge(handlers) {
  const CHANNEL = "web-console";
  const hasParent =
    typeof window !== "undefined" && window.parent && window.parent !== window;

  const emit = (type, payload) => {
    if (!hasParent) {
      return;
    }
    try {
      window.parent.postMessage({ channel: CHANNEL, type, payload }, "*");
    } catch (error) {
      console.warn("Failed to post bridge message:", error);
    }
  };

  window.addEventListener("message", async (event) => {
    const message = event.data;
    if (!message || message.channel !== CHANNEL) {
      return;
    }

    try {
      const handler = handlers[message.type];
      if (handler) {
        await handler(message, emit);
      } else if (hasParent && message.type !== "error") {
        emit("error", { type: "unknown-command", payload: message });
      }
    } catch (error) {
      if (hasParent) {
        emit("error", {
          type: message.type,
          message: error instanceof Error ? error.message : "unknown error",
        });
      }
    }
  });

  if (hasParent) {
    emit("bridge-initialized", { timestamp: Date.now() });
  }

  return { emit };
}
