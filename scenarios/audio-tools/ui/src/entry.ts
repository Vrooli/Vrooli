import { shimStorage } from "@vrooli/iframe-bridge/storage-shim";

// Storage access can throw in sandboxed frames. Run the tiny shim before
// importing application modules, some of which inspect localStorage at load.
shimStorage();

async function start() {
  if (typeof window !== "undefined" && window.parent !== window) {
    const { initIframeBridgeChild } = await import("@vrooli/iframe-bridge/child");
    initIframeBridgeChild({ appId: "audio-tools" });
  }

  await import("./main");
}

void start();
