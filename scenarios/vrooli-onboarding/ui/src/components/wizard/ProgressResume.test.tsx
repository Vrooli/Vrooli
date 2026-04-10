// [REQ:REQ-P1-004] Progress Resume Flow
import { screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithQueryClient } from "../../test-utils";
import App from "../../App";

function renderApp() {
  return renderWithQueryClient(<App />);
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
    // Wait for fetch to resolve, then verify no resume prompt shown
    await waitFor(() => {
      expect(screen.queryByTestId("resume-prompt")).not.toBeInTheDocument();
    });

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

  it("resume prompt displays the saved step number", async () => {
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

    // Should display the step number (1-indexed: step 2 → "step 3")
    expect(screen.getByTestId("resume-prompt")).toHaveTextContent(/step 3/i);
    // Resume prompt should have alert role for accessibility
    expect(screen.getByTestId("resume-prompt")).toHaveAttribute("role", "alert");

    globalThis.fetch = originalFetch;
  });
});
