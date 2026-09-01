import React from "react";
import ReactDOM from "react-dom/client";
import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
// INTEROP-CRITICAL: proxy-aware routing and iframe bridge setup keep the board
// addressable both directly and when embedded by the Vrooli control plane.
import "@fontsource/inter/400.css";
import "@fontsource/inter/600.css";
import "@fontsource/jetbrains-mono/400.css";
import "@fontsource/jetbrains-mono/600.css";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { installChunkReloadGuard } from "@vrooli/api-base";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import App from "./App";
import "./styles.css";
import "./design-tokens.css";

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
installChunkReloadGuard();

if (import.meta.env.PROD && "serviceWorker" in navigator) {
  void navigator.serviceWorker.register("./sw.js");
}

const queryClient = new QueryClient();

if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "command-center" });
}

initSpatialNav();

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element '#root' not found");
}

ReactDOM.createRoot(rootElement).render(
  // vrooli:library-strings-provider start
  <LibraryStringsProvider translate={(_key, fallback) => fallback ?? _key}>
    <React.StrictMode>
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </React.StrictMode>
  </LibraryStringsProvider>
  // vrooli:library-strings-provider end
);
