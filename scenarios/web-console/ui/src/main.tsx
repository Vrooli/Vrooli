import { i18n } from "./i18n";
import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
import { BaseStyles } from "@vrooli/react-component-library/BaseStyles/1";
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import { registerVoiceTransport } from "./audio-integration";
import "./i18n";
import "./design-tokens.css";
import "./styles.css";
import { onProfilerRender } from "./lib/profiler";

const queryClient = new QueryClient();

registerVoiceTransport();

// INTEROP-CRITICAL: iframe bridge must be initialized before React mount
// so parent scenarios receive the ready signal and can coordinate routing.
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "web-console" });
}

// Keep controller navigation available both standalone and when embedded in a
// host frame. The bridge is process-wide and is disposed with the page.
initSpatialNav();

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

const rootEl = document.getElementById("root");
if (!rootEl) throw new Error("Missing #root element in index.html");

ReactDOM.createRoot(rootEl).render(
    // vrooli:library-strings-provider start
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
      <BaseStyles />
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <React.Profiler id="App" onRender={onProfilerRender}>
        <App />
      </React.Profiler>
    </QueryClientProvider>
  </React.StrictMode>

    </LibraryStringsProvider>
    // vrooli:library-strings-provider end
);
