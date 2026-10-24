import { describe, it, expect, vi, beforeEach } from "vitest";
import { act, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";

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

function renderDetail(sessionId = "session-1") {
  return render(
    <MemoryRouter initialEntries={[`/sessions/${sessionId}`]}>
      <Routes>
        <Route path="/sessions/:id" element={<SessionDetailPage />} />
        <Route path="/sessions" element={<div data-testid="list-page" />} />
      </Routes>
    </MemoryRouter>,
  );
}

let SessionDetailPage: typeof import("./SessionDetailPage")["SessionDetailPage"];

beforeEach(async () => {
  vi.clearAllMocks();
  const module = await import("./SessionDetailPage");
  SessionDetailPage = module.SessionDetailPage;
  const { useSessionStore } = await import("../store/sessionStore");
  useSessionStore.getState().reset();
});

describe("SessionDetailPage", () => {
  it("renders the loading state before the session resolves", async () => {
    const api = await import("../lib/api/sessions");
    (api.getSession as ReturnType<typeof vi.fn>).mockReturnValueOnce(new Promise(() => {}));
    renderDetail();
    expect(screen.getByText(/Loading session/i)).toBeInTheDocument();
  });

  it("renders the session with connecting state once loaded", async () => {
    const api = await import("../lib/api/sessions");
    (api.getSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText("demo")).toBeInTheDocument();
    });
    expect(screen.getByText(/Connecting to desktop/i)).toBeInTheDocument();
  });

  it("transitions to the connected state when the store status changes", async () => {
    const api = await import("../lib/api/sessions");
    (api.getSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);
    renderDetail();
    await waitFor(() => {
      expect(screen.getByText("demo")).toBeInTheDocument();
    });
    const { useSessionStore } = await import("../store/sessionStore");
    act(() => {
      useSessionStore.getState().setConnectionStatus("connected");
    });
    await waitFor(() => {
      expect(screen.queryByText(/Connecting to desktop/i)).not.toBeInTheDocument();
    });
  });

  it("shows an error message when the session load fails", async () => {
    const api = await import("../lib/api/sessions");
    (api.getSession as ReturnType<typeof vi.fn>).mockRejectedValueOnce(new Error("nope"));
    renderDetail();
    expect(await screen.findByText(/Unable to load session/i)).toBeInTheDocument();
    expect(await screen.findByText(/nope/)).toBeInTheDocument();
  });

  it("does not render a modal/drawer dialog", async () => {
    const api = await import("../lib/api/sessions");
    (api.getSession as ReturnType<typeof vi.fn>).mockResolvedValueOnce(sampleSession);
    const { container } = renderDetail();
    await waitFor(() => {
      expect(screen.getByText("demo")).toBeInTheDocument();
    });
    expect(container.querySelector('[role="dialog"]')).toBeNull();
    expect(container.querySelector('[data-vaul-drawer]')).toBeNull();
    expect(container.querySelector('[data-drawer]')).toBeNull();
  });
});
