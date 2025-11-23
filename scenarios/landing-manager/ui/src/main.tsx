import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import App from "./App";
import "./styles.css";

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
  } catch (error) {
    // Unable to parse parent origin - will use default
  }

  initIframeBridgeChild({
    parentOrigin,
    appId: "landing-manager",
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

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </React.StrictMode>
);
