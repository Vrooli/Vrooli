import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { Route, Routes } from "react-router-dom";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { setLocale } from "../../i18n";

vi.mock("../../api/inventory", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/inventory")>();
  return {
    ...actual,
    fetchRun: vi.fn(),
  };
});

import { RunDetailPage } from "./RunDetailPage";

describe("RunDetailPage accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    const { fetchRun } = await import("../../api/inventory");
    vi.mocked(fetchRun).mockResolvedValue({
      id: "run-abc-123",
      flowId: "notes.attachment-upload.ui",
      flowPath: "ui/src/features/notes/flow/flow.json",
      root: ".",
      mode: "check",
      status: "failed",
      startedAt: "2026-05-10T11:59:58Z",
      finishedAt: "2026-05-10T12:00:00Z",
      durationMs: 2000,
      counterexample: JSON.stringify({
        states: [{ state: "draft" }, { state: "uploaded", event: "begin" }],
      }),
    });
  });

  afterEach(() => cleanup());

  it("renders without axe violations", async () => {
    const { container } = renderWithProviders(
      <Routes>
        <Route path="/runs/:runId" element={<RunDetailPage />} />
      </Routes>,
      { routerEntries: ["/runs/run-abc-123"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId("run-detail-page")).toBeInTheDocument(),
    );
    await expectNoA11yViolations(container);
  });
});
