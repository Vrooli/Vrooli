import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import type { ReindexCancel, ReindexStatus } from "../../api/reindex";

vi.mock("../../api/reindex", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/reindex")>();
  const reindexStatus = vi.fn();
  const reindexCancel = vi.fn();
  return { ...actual, reindexStatus, reindexCancel };
});

import { JobDetailPage } from "./JobDetailPage";
import { reindexCancel, reindexStatus } from "../../api/reindex";

beforeEach(() => {
  vi.mocked(reindexStatus).mockReset();
  vi.mocked(reindexCancel).mockReset();
});

describe("JobDetailPage", () => {
  it("renders running-job metadata with a cancel button", async () => {
    vi.mocked(reindexStatus).mockResolvedValue({
      jobId: "job-1",
      state: "running",
      processed: 2,
      total: 5,
      error: "",
    } satisfies ReindexStatus);
    renderWithProviders(
      <Routes>
        <Route path="/reindex/:jobId" element={<JobDetailPage />} />
      </Routes>,
      { routerEntries: ["/reindex/job-1"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.reindex.detail.meta)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.reindex.detail.progress)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.reindex.detail.cancel)).toBeInTheDocument();
  });

  it("hides the cancel button once the job reaches a terminal state", async () => {
    vi.mocked(reindexStatus).mockResolvedValue({
      jobId: "job-2",
      state: "succeeded",
      processed: 5,
      total: 5,
      error: "",
    } satisfies ReindexStatus);
    renderWithProviders(
      <Routes>
        <Route path="/reindex/:jobId" element={<JobDetailPage />} />
      </Routes>,
      { routerEntries: ["/reindex/job-2"] },
    );
    await waitFor(() =>
      expect(screen.getByTestId(selectors.reindex.detail.meta)).toBeInTheDocument(),
    );
    expect(screen.queryByTestId(selectors.reindex.detail.cancel)).not.toBeInTheDocument();
  });

  it("invokes ReindexCancel when the cancel button is clicked", async () => {
    vi.mocked(reindexStatus).mockResolvedValue({
      jobId: "job-3",
      state: "running",
      processed: 0,
      total: 1,
      error: "",
    } satisfies ReindexStatus);
    vi.mocked(reindexCancel).mockResolvedValue({
      jobId: "job-3",
      cancelled: true,
    } satisfies ReindexCancel);
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route path="/reindex/:jobId" element={<JobDetailPage />} />
      </Routes>,
      { routerEntries: ["/reindex/job-3"] },
    );
    const cancelButton = await screen.findByTestId(selectors.reindex.detail.cancel);
    await user.click(cancelButton);
    await waitFor(() => expect(reindexCancel).toHaveBeenCalledWith("job-3"));
  });

  it("renders the error envelope when the status API throws", async () => {
    vi.mocked(reindexStatus).mockRejectedValueOnce(new Error("nope"));
    renderWithProviders(
      <Routes>
        <Route path="/reindex/:jobId" element={<JobDetailPage />} />
      </Routes>,
      { routerEntries: ["/reindex/job-x"] },
    );
    expect(await screen.findByTestId(selectors.reindex.detail.error)).toBeInTheDocument();
  });
});
