import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { ThemeProvider } from "./contexts/ThemeContext";
import { AppStateProvider } from "./contexts/AppStateContext";
import { WebSocketProvider } from "./contexts/WebSocketContext";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { TooltipProvider } from "./components/ui/tooltip";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30000, // 30 seconds
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

if (window.top !== window.self) {
  initIframeBridgeChild();
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <AppStateProvider>
            <WebSocketProvider>
              <TooltipProvider delayDuration={0} skipDelayDuration={0}>
                <App />
              </TooltipProvider>
            </WebSocketProvider>
          </AppStateProvider>
        </ThemeProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
