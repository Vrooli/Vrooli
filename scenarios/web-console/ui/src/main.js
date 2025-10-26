import "./styles.css";
import "@xterm/xterm/css/xterm.css";

import { bootstrapApp } from "./modules/app/bootstrap.js";

const previousCleanup =
  typeof globalThis !== "undefined" &&
  typeof globalThis.__WEB_CONSOLE_APP_CLEANUP__ === "function"
    ? globalThis.__WEB_CONSOLE_APP_CLEANUP__
    : null;

if (previousCleanup) {
  try {
    previousCleanup();
  } catch (error) {
    console.error("Failed to run previous web-console cleanup:", error);
  }
}

const teardown = bootstrapApp();

let disposed = false;

function runTeardown() {
  if (disposed) {
    return;
  }
  disposed = true;
  try {
    if (typeof teardown === "function") {
      teardown();
    }
  } finally {
    if (typeof globalThis !== "undefined") {
      delete globalThis.__WEB_CONSOLE_APP_CLEANUP__;
    }
  }
}

if (typeof globalThis !== "undefined") {
  globalThis.__WEB_CONSOLE_APP_CLEANUP__ = runTeardown;
}

if (import.meta.hot) {
  import.meta.hot.dispose(runTeardown);
}

if (typeof window !== "undefined") {
  window.addEventListener("pagehide", runTeardown, { once: true });
}
