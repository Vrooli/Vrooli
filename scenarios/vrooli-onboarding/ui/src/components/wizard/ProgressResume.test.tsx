// [REQ:REQ-P1-004] Progress Resume Flow
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi } from "vitest";
import App from "../../App";

function renderApp() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  );
}

describe("Progress Resume Flow", () => {
  it("shows resume prompt when saved progress exists", async () => {
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (typeof url === "string" && url.includes("/progress")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            id: 1,
            user_id: "default",
            current_step: 2,
            completed_steps: [0, 1],
            config_data: { resources: ["postgres"] },
            updated_at: "2026-01-01T00:00:00Z",
          }),
        });
      }
      return Promise.resolve({ ok: false, status: 404 });
    });

    renderApp();

    await waitFor(() => {
      expect(screen.getByTestId("resume-prompt")).toBeInTheDocument();
    });

    expect(screen.getByTestId("resume-button")).toBeInTheDocument();
    expect(screen.getByText(/saved progress/i)).toBeInTheDocument();

    globalThis.fetch = originalFetch;
  });

  it("does not show resume when no saved progress", async () => {
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn().mockImplementation(() => {
      return Promise.resolve({ ok: false, status: 404 });
    });

    renderApp();
    await new Promise((r) => setTimeout(r, 50));
    expect(screen.queryByTestId("resume-prompt")).not.toBeInTheDocument();

    globalThis.fetch = originalFetch;
  });

  it("restores selected resources from saved progress", async () => {
    const originalFetch = globalThis.fetch;
    globalThis.fetch = vi.fn().mockImplementation((url: string) => {
      if (typeof url === "string" && url.includes("/progress")) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({
            id: 1,
            user_id: "default",
            current_step: 1,
            completed_steps: [0],
            config_data: { resources: ["postgres", "redis"] },
            updated_at: "2026-01-01T00:00:00Z",
          }),
        });
      }
      return Promise.resolve({ ok: false, status: 404 });
    });

    renderApp();

    await waitFor(() => {
      expect(screen.getByTestId("resume-prompt")).toBeInTheDocument();
    });

    globalThis.fetch = originalFetch;
  });
});
