import { i18n } from "./i18n";
import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
// INTEROP-CRITICAL: interop-sensitive configuration below — do not remove without checking host-frame embedding.
import React from "react";
import ReactDOM from "react-dom/client";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import "./styles.css";
import { onProfilerRender } from "./lib/profiler";

initSpatialNav();

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("./sw.js", { scope: "./" }).catch((error: unknown) => {
      console.warn("Secrets Manager service worker registration failed", error);
    });
  });
}

if (typeof window !== "undefined" && window.parent !== window) {
  initIframeBridgeChild(
    {
      appId: "secrets-manager",
      captureLogs: true,
      // The bridge records request metadata only. Secret-bearing request and
      // response bodies are outside the host observability contract.
      captureNetwork: true
    }
  );
}

const root = document.getElementById("root");
if (!root) {
  throw new Error("Secrets Manager could not find its root element");
}

ReactDOM.createRoot(root).render(
    // vrooli:library-strings-provider start
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
  <React.StrictMode>
    <React.Profiler id="App" onRender={onProfilerRender}>
      <App />
    </React.Profiler>
  </React.StrictMode>

    </LibraryStringsProvider>
    // vrooli:library-strings-provider end
);
