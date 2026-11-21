import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

vi.mock("../lib/api/sessions", () => ({
  listSessions: vi.fn(),
  createSession: vi.fn(),
  destroySession: vi.fn(),
  getSession: vi.fn(),
  heartbeatSession: vi.fn(),
  executeSessionControl: vi.fn(),
  buildVncWsUrl: vi.fn(() => "ws://localhost/ws"),
}));

vi.mock("../lib/bridge", () => ({
  postSessionEvent: vi.fn(() => true),
}));

const sampleSession = {
  id: "session-1",
  scenario_name: "demo",
  state: "running" as const,
  vnc_port: 5900,
  ws_port: 6080,
  width: 1280,
  height: 720,
  created_at: "2026-01-01T00:00:00Z",
  last_heartbeat: "2026-01-01T00:00:00Z",
  is_recording: false,
  network_mode: "normal" as const,
  dark_mode: false,
  app_running: false,
  platform: "linux",
};

function renderWithProviders(initialPath = "/sessions") {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, refetchInterval: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initialPath]}>
        <Routes>
          <Route
            path="/sessions"
            element={<SessionListPage />}
          />
          <Route path="/sessions/:id" element={<div data-testid="detail-page" />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

let SessionListPage: typeof import("./SessionListPage")["SessionListPage"];

beforeEach(async () => {
  vi.clearAllMocks();
  const module = await import("./SessionListPage");
  SessionListPage = module.SessionListPage;
  const { useSessionStore } = await import("../store/sessionStore");
  useSessionStore.getState().reset();
});

describe("SessionListPage", () => {
  it("shows the empty state when the session list is empty", async () => {
    const api = await import("../lib/api/sessions");
    (api.listSessions as ReturnType<typeof vi.fn>).mockResolvedValueOnce([]);
    renderWithProviders();
    expect(
      await screen.findByText(/No active sessions yet/i),
    ).toBeInTheDocument();
  });

  it("renders a row for each session returned by the API", async () => {
    const api = await import("../lib/api/sessions");
    (api.listSessions as ReturnType<typeof vi.fn>).mockResolvedValueOnce([
      sampleSession,
      { ...sampleSession, id: "session-2", scenario_name: "other" },
    ]);
    renderWithProviders();
    expect(await screen.findByText("demo")).toBeInTheDocument();
    expect(await screen.findByText("other")).toBeInTheDocument();
  });

  it("requires a scenario name before calling createSession", async () => {
    const api = await import("../lib/api/sessions");
    (api.listSessions as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    renderWithProviders();
    fireEvent.click(await screen.findByRole("button", { name: /Start Session/i }));
    expect(await screen.findByText(/Scenario name is required/i)).toBeInTheDocument();
    expect(api.createSession).not.toHaveBeenCalled();
  });

  it("creates a new session and navigates to /sessions/:id", async () => {
    const api = await import("../lib/api/sessions");
    (api.listSessions as ReturnType<typeof vi.fn>).mockResolvedValue([]);
    (api.createSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);

    renderWithProviders();
    await screen.findByText(/No active sessions yet/i);

    const input = screen.getByPlaceholderText(/my-app-smoke/i);
    fireEvent.change(input, { target: { value: "my-scenario" } });
    fireEvent.click(screen.getByRole("button", { name: /Start Session/i }));

    await waitFor(() => {
      expect(api.createSession).toHaveBeenCalledWith(
        expect.objectContaining({ scenario_name: "my-scenario", platform: "linux" }),
      );
    });
    await waitFor(() => {
      expect(screen.getByTestId("detail-page")).toBeInTheDocument();
    });
  });
});
