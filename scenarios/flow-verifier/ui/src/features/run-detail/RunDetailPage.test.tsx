import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  return {
    ...actual,
    fetchRun: vi.fn(),
  };
});

import { RunDetailPage } from "./RunDetailPage";
import type { RunRow } from "../../api/inventory";

const baseRun: RunRow = {
  id: "run-abc-123",
  flowId: "notes.attachment-upload.ui",
  flowPath: "ui/src/features/notes/flow/flow.json",
  root: ".",
  mode: "check",
  status: "passed",
  startedAt: "2026-05-10T11:59:58Z",
  finishedAt: "2026-05-10T12:00:00Z",
  durationMs: 2000,
};

const renderAt = (route: string) =>
  renderWithProviders(
    <Routes>
      <Route path="/runs/:runId" element={<RunDetailPage />} />
      <Route path="/runs" element={<RunDetailPage />} />
    </Routes>,
    { routerEntries: [route] },
  );

describe("RunDetailPage", () => {
  beforeEach(async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockReset();
  });
  afterEach(() => cleanup());

  it("renders the missing-id state with no :runId in route", () => {
    renderAt("/runs");
    expect(screen.getByTestId("run-detail-missing")).toBeInTheDocument();
  });

  it("renders the loading state while in flight", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockReturnValue(new Promise(() => {}));
    renderAt("/runs/run-abc-123");
    expect(screen.getByTestId("run-detail-loading")).toBeInTheDocument();
  });

  it("renders the error state + back link on fetch failure", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockRejectedValue(new Error("boom"));
    renderAt("/runs/run-abc-123");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-error")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("run-detail-back")).toHaveAttribute("href", "/flows");
  });

  it("renders metadata header for a passed run with no counterexample", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockResolvedValue(baseRun);
    renderAt("/runs/run-abc-123");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    expect(screen.getByTestId("run-detail-id")).toHaveTextContent("run-abc-123");
    expect(screen.getByTestId("run-detail-status")).toHaveTextContent("passed");
    expect(screen.getByTestId("run-detail-mode")).toHaveTextContent("check");
    expect(screen.getByTestId("run-detail-duration")).toHaveTextContent(
      "runDetail.durationMs",
    );
    expect(screen.getByTestId("run-detail-flow-link")).toHaveAttribute(
      "href",
      "/flows/notes.attachment-upload.ui",
    );
    expect(screen.getByTestId("run-detail-result")).toBeInTheDocument();
  });

  it("counterexample tab shows the empty state for a run with no counterexample", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockResolvedValue(baseRun);
    const user = userEvent.setup();
    renderAt("/runs/run-abc-123");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("run-detail-tab-counterexample"));
    expect(screen.getByTestId("run-detail-no-counterexample")).toBeInTheDocument();
  });

  it("renders the counterexample tree for a failing run", async () => {
    const { fetchRun } = await import("../../api/inventory");
    const failing: RunRow = {
      ...baseRun,
      id: "run-fail-1",
      status: "failed",
      counterexample: JSON.stringify({
        states: [{ state: "draft" }, { state: "uploaded", event: "begin" }],
      }),
    };
    vi.mocked(fetchRun).mockResolvedValue(failing);
    const user = userEvent.setup();
    renderAt("/runs/run-fail-1");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("run-detail-tab-counterexample"));
    expect(screen.getByTestId("run-detail-counterexample")).toBeInTheDocument();
    expect(screen.getByTestId("run-detail-json-root")).toBeInTheDocument();
    expect(
      screen.queryByTestId("run-detail-counterexample-parse-error"),
    ).not.toBeInTheDocument();
  });

  it("surfaces a parse error when counterexample JSON is malformed", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockResolvedValue({
      ...baseRun,
      id: "run-bad-1",
      status: "failed",
      counterexample: "{not json",
    });
    const user = userEvent.setup();
    renderAt("/runs/run-bad-1");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("run-detail-tab-counterexample"));
    expect(
      screen.getByTestId("run-detail-counterexample-parse-error"),
    ).toBeInTheDocument();
  });

  it("the counterexample <details> can be toggled closed by the user", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockResolvedValue({
      ...baseRun,
      id: "run-fail-2",
      status: "failed",
      counterexample: JSON.stringify({ states: [{ state: "x" }] }),
    });
    const user = userEvent.setup();
    renderAt("/runs/run-fail-2");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("run-detail-tab-counterexample"));
    const details = await screen.findByTestId<HTMLDetailsElement>(
      "run-detail-counterexample",
    );
    expect(details.open).toBe(true);
    await user.click(screen.getByTestId("run-detail-counterexample-toggle"));
    expect(details.open).toBe(false);
  });

  it("raw output tab shows captured quint output with a copy button", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockResolvedValue({
      ...baseRun,
      id: "run-raw-1",
      output: "quint verify ok\nstates: 12",
    });
    const user = userEvent.setup();
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });
    renderAt("/runs/run-raw-1");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("run-detail-tab-raw"));
    expect(screen.getByTestId("run-detail-raw-pre")).toHaveTextContent(
      "quint verify ok",
    );
    await user.click(screen.getByTestId("run-detail-raw-copy"));
    expect(writeText).toHaveBeenCalledWith("quint verify ok\nstates: 12");
  });

  it("raw output tab shows the empty state when no output is captured", async () => {
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockResolvedValue(baseRun);
    const user = userEvent.setup();
    renderAt("/runs/run-abc-123");
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    await user.click(screen.getByTestId("run-detail-tab-raw"));
    expect(screen.getByTestId("run-detail-raw-empty")).toBeInTheDocument();
  });
});
