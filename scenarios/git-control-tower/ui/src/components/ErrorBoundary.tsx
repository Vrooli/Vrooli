import React from "react";

interface ErrorBoundaryState {
  error: Error | null;
}

export class ErrorBoundary extends React.Component<
  { children: React.ReactNode },
  ErrorBoundaryState
> {
  state: ErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("[ErrorBoundary] Uncaught render error:", error, info.componentStack);
  }

  handleClearAndReload = () => {
    try {
      // Clear all gct-prefixed localStorage entries
      const keys: string[] = [];
      for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key?.startsWith("gct.") || key?.startsWith("gct-")) {
          keys.push(key);
        }
      }
      keys.forEach((k) => localStorage.removeItem(k));
    } catch {
      // localStorage itself may be broken — try clearing everything
      try {
        localStorage.clear();
      } catch {
        // Nothing more we can do
      }
    }
    window.location.replace(window.location.pathname);
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (!this.state.error) {
      return this.props.children;
    }

    const { error } = this.state;

    return (
      <div
        style={{
          minHeight: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: "#020617",
          color: "#e2e8f0",
          fontFamily: "system-ui, -apple-system, sans-serif",
          padding: "1.5rem",
        }}
      >
        <div style={{ maxWidth: "28rem", width: "100%", textAlign: "center" }}>
          <h1
            style={{
              fontSize: "1.25rem",
              fontWeight: 600,
              marginBottom: "0.75rem",
            }}
          >
            Something went wrong
          </h1>
          <p
            style={{
              fontSize: "0.875rem",
              color: "#94a3b8",
              marginBottom: "1.5rem",
            }}
          >
            The app crashed during rendering. This is often caused by stale
            browser data. Try reloading, or clear saved data and reload.
          </p>

          <div
            style={{
              display: "flex",
              gap: "0.75rem",
              justifyContent: "center",
              marginBottom: "1.5rem",
            }}
          >
            <button
              type="button"
              onClick={this.handleReload}
              style={{
                padding: "0.5rem 1rem",
                borderRadius: "0.375rem",
                border: "1px solid #334155",
                backgroundColor: "#1e293b",
                color: "#e2e8f0",
                fontSize: "0.875rem",
                cursor: "pointer",
              }}
            >
              Reload
            </button>
            <button
              type="button"
              onClick={this.handleClearAndReload}
              style={{
                padding: "0.5rem 1rem",
                borderRadius: "0.375rem",
                border: "none",
                backgroundColor: "#2563eb",
                color: "#ffffff",
                fontSize: "0.875rem",
                cursor: "pointer",
              }}
            >
              Clear data &amp; reload
            </button>
          </div>

          <details
            style={{
              textAlign: "left",
              backgroundColor: "#0f172a",
              borderRadius: "0.375rem",
              padding: "0.75rem",
              border: "1px solid #1e293b",
            }}
          >
            <summary
              style={{
                cursor: "pointer",
                fontSize: "0.75rem",
                color: "#64748b",
              }}
            >
              Error details
            </summary>
            <pre
              style={{
                marginTop: "0.5rem",
                fontSize: "0.6875rem",
                color: "#f87171",
                whiteSpace: "pre-wrap",
                wordBreak: "break-word",
                maxHeight: "12rem",
                overflow: "auto",
              }}
            >
              {error.message}
              {error.stack && `\n\n${error.stack}`}
            </pre>
          </details>
        </div>
      </div>
    );
  }
}
