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

// NOTE: StrictMode intentionally double-renders components to detect side effects.
// During rapid state transitions (like fresh chat message send), this can push
// borderline render counts over React's 50-render limit. Temporarily disabled
// while investigating "too many re-renders" issue.
// TODO: Re-enable StrictMode after fixing the render loop issue
const rootElement = document.getElementById("root");
if (!rootElement) {
  throw new Error("Root element not found");
}
ReactDOM.createRoot(rootElement).render(
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
);
