import { describe, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { expectNoA11yViolations } from "@vrooli/api-base/testing";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/reindex", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/reindex")>();
  const reindex = vi.fn(() =>
    Promise.resolve({
      jobId: "job-1",
      plannedUpserts: 2,
      plannedDeletes: 0,
      dryRun: false,
    }),
  );
  const reindexStatus = vi.fn(() =>
    Promise.resolve({
      jobId: "job-1",
      state: "running" as const,
      processed: 1,
      total: 2,
      error: "",
    }),
  );
  return { ...actual, reindex, reindexStatus };
});

import { ReindexPage } from "./ReindexPage";
import { JobDetailPage } from "./JobDetailPage";

beforeEach(() => {
  window.localStorage.clear();
});

describe("Reindex feature accessibility", () => {
  it("ReindexPage has no axe violations (empty jobs state)", async () => {
    const { container } = renderWithProviders(<ReindexPage />);
    await expectNoA11yViolations(container);
  });

  it("ReindexPage has no axe violations with a tracked job", async () => {
    const user = userEvent.setup();
    const { container } = renderWithProviders(<ReindexPage />);
    await user.type(screen.getByTestId(selectors.reindex.scenarioInput), "ui-health");
    await user.click(screen.getByTestId(selectors.reindex.submit));
    await waitFor(() => screen.getByTestId(selectors.reindex.jobsList));
    await expectNoA11yViolations(container);
  });

  it("JobDetailPage has no axe violations", async () => {
    const { container, findByTestId } = renderWithProviders(
      <Routes>
        <Route path="/reindex/:jobId" element={<JobDetailPage />} />
      </Routes>,
      { routerEntries: ["/reindex/job-1"] },
    );
    await findByTestId(selectors.reindex.detail.meta);
    await expectNoA11yViolations(container);
  });
});
