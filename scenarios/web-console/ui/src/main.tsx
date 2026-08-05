import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import App from "./App";
import { registerVoiceTransport } from "./audio-integration";
import "./i18n";
import "./design-tokens.css";
import "./styles.css";

const queryClient = new QueryClient();

registerVoiceTransport();

// INTEROP-CRITICAL: iframe bridge must be initialized before React mount
// so parent scenarios receive the ready signal and can coordinate routing.
if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "web-console" });
}

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

const rootEl = document.getElementById("root");
if (!rootEl) throw new Error("Missing #root element in index.html");

ReactDOM.createRoot(rootEl).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>,
);
