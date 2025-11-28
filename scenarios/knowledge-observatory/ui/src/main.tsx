import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient();

// Initialize iframe bridge for embedded scenarios
declare global {
  interface Window {
    __knowledgeObservatoryBridgeInitialized?: boolean;
  }
}

if (typeof window !== 'undefined' && window.parent !== window) {
  if (!window.__knowledgeObservatoryBridgeInitialized) {
    let parentOrigin: string | undefined;
    try {
      if (document.referrer) {
        parentOrigin = new URL(document.referrer).origin;
      }
    } catch {
      // Unable to determine parent origin; iframe bridge will use fallback behavior
    }

    try {
      initIframeBridgeChild({
        parentOrigin,
        appId: 'knowledge-observatory',
        captureLogs: { enabled: true },
        captureNetwork: { enabled: true },
      });
      window.__knowledgeObservatoryBridgeInitialized = true;
    } catch (error) {
      console.warn('[knowledge-observatory] Unable to initialize iframe bridge', error);
    }
  }
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
