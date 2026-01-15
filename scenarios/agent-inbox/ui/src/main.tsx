import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initIframeBridgeChild } from "@vrooli/iframe-bridge";
import { ErrorBoundary } from "./components/ErrorBoundary";
import App from "./App";
import "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Prevent queries from retrying indefinitely on errors
      retry: 2,
      // Don't refetch on window focus if we have an error
      refetchOnWindowFocus: (query) => query.state.status !== "error",
    },
    mutations: {
      // Prevent mutations from retrying indefinitely
      retry: 1,
    },
  },
});

if (window.top !== window.self) {
  initIframeBridgeChild();
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <ErrorBoundary
      critical
      name="Root"
      onError={(error, errorInfo) => {
        // Log critical errors for debugging
        console.error("[CriticalError] App crashed:", error, errorInfo);
      }}
    >
      <QueryClientProvider client={queryClient}>
        <App />
      </QueryClientProvider>
    </ErrorBoundary>
  </React.StrictMode>
);
