import { LibraryStringsProvider } from "@vrooli/react-component-library/useLocale/1";
import { BaseStyles } from "@vrooli/react-component-library/BaseStyles/1";
import { i18n } from "./i18n";
import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import { initSpatialNav } from "@vrooli/iframe-bridge/spatial";
import { installChunkReloadGuard } from "@vrooli/api-base";
import App from "./App";
import { onProfilerRender } from "./lib/profiler";
import "./styles.css";

// Code-split routes use lazy(); after a rebuild the old hashed chunks are
// gone, so a tab opened before the deploy would crash on its next
// navigation. This guard reloads once (rate-limited) instead.
// INTEROP-CRITICAL: preserve the host's keyboard/gamepad focus contract.
installChunkReloadGuard();
initSpatialNav();

function registerServiceWorker() {
  if (!import.meta.env.PROD || !('serviceWorker' in navigator)) {
    return;
  }

  window.addEventListener('load', () => {
    void navigator.serviceWorker.register('/sw.js').catch((error: unknown) => {
      console.warn('Service worker registration failed', error);
    });
  });
}

registerServiceWorker();

declare global {
  interface Window {
    __landingManagerBridgeInitialized?: boolean;
  }
}

const queryClient = new QueryClient();

if (typeof window !== 'undefined' && window.parent !== window && !window.__landingManagerBridgeInitialized) {
  let parentOrigin: string | undefined;
  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch {
    // Unable to parse parent origin - will use default
  }

  initIframeBridgeChild({
    parentOrigin,
    appId: "landing-page-business-suite",
    captureLogs: {
      enabled: true,
      streaming: true,
      levels: ['log', 'info', 'warn', 'error', 'debug'],
      bufferSize: 100
    },
    captureNetwork: {
      enabled: true,
      streaming: true,
      bufferSize: 100
    }
  });
  window.__landingManagerBridgeInitialized = true;
}

const root = document.getElementById("root");
if (!root) {
  throw new Error("Root element not found");
}

ReactDOM.createRoot(root).render(
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
    </LibraryStringsProvider>,
    // vrooli:library-strings-provider end
  );
