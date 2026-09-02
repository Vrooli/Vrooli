import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
import { i18n } from "./i18n";
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { initSpatialNav } from "@vrooli/iframe-bridge";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";
import "./styles.css";

const queryClient = new QueryClient();

if (window.top !== window.self) {
  initIframeBridgeChild({ appId: "scenario-to-desktop" });
}

initSpatialNav();

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    void navigator.serviceWorker.register("./sw.js", { scope: "./" });
  });
}

const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error(
    'Root element not found. Check that index.html contains a <div id="root"></div>.',
  );
}
ReactDOM.createRoot(rootElement).render(
    // vrooli:library-strings-provider start
    <LibraryStringsProvider translate={(key, fallback) => i18n.t(key, { defaultValue: fallback })}>
<React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <React.Profiler id="scenario-to-desktop" onRender={onProfilerRender}>
        <App />
      </React.Profiler>
    </QueryClientProvider>
  </React.StrictMode>
    </LibraryStringsProvider>,
    // vrooli:library-strings-provider end
  );
