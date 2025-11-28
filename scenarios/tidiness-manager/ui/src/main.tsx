import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge/child";
import { ToastProvider } from "./components/ui/toast";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Disable retries for faster failure feedback
      retry: false,
      // Shorter staleTime for testing environments
      staleTime: 0,
      // Always refetch on mount to ensure fresh data
      refetchOnMount: true,
      // Keep data in cache for 5 minutes
      gcTime: 5 * 60 * 1000,
    },
  },
});

// Initialize iframe bridge before React render
const BRIDGE_FLAG = '__vrooli_iframe_bridge_initialized__';
if (
  typeof window !== 'undefined' &&
  window.parent !== window &&
  !((window as unknown as Record<string, unknown>)[BRIDGE_FLAG] ?? false)
) {
  let parentOrigin: string | undefined;

  try {
    if (document.referrer) {
      parentOrigin = new URL(document.referrer).origin;
    }
  } catch (error) {
    console.warn('[tidiness-manager] Unable to determine parent origin for iframe bridge', error);
  }

  initIframeBridgeChild({
    parentOrigin,
    appId: 'tidiness-manager',
    captureLogs: { enabled: true, streaming: true },
    captureNetwork: { enabled: true, streaming: true },
  });
  (window as unknown as Record<string, unknown>)[BRIDGE_FLAG] = true;
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <ToastProvider>
        <App />
      </ToastProvider>
    </QueryClientProvider>
  </React.StrictMode>
);
