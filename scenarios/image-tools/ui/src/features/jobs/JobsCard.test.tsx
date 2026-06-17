/**
 * JobsCard tests — focused on the jobs-card surface only.
 *
 * Renders <JobsCard /> directly so failures point at jobs-feature
 * behaviour, not shell composition. Follows the canonical mock-builder
 * pattern from the co-located `./mocks/jobs`.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { JobState } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";
import { makeJob, makeListJobsResponse } from "./mocks/factories";
import { makeJobsMocks } from "./mocks/jobs";

vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return { ...actual, ...makeJobsMocks() };
});

import { JobsCard } from "./JobsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("JobsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when listJobs resolves with no jobs", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockResolvedValueOnce(makeListJobsResponse());

    renderWithProviders(<JobsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.jobs.list)).not.toBeInTheDocument();
  });

  it("renders the list with operation, lane, state, and progress", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockResolvedValueOnce(
      makeListJobsResponse({
        jobs: [
          makeJob({ id: "job-a", operation: "upscale", progress: 50 }),
          makeJob({ id: "job-b", operation: "resize", state: JobState.SUCCEEDED, progress: 100 }),
        ],
      }),
    );

    renderWithProviders(<JobsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.jobs.list);
    expect(list.textContent).toContain("job-a");
    expect(list.textContent).toContain("upscale");
    expect(list.textContent).toContain("resize");
    expect(screen.getAllByTestId(selectors.jobs.operation)).toHaveLength(2);
    expect(screen.getAllByTestId(selectors.jobs.progress)[0]?.textContent).toContain("50");
  });

  it("renders the error state when listJobs rejects", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockRejectedValueOnce(new Error("jobs unavailable"));

    renderWithProviders(<JobsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.error)).toBeInTheDocument();
    });
  });

  it("shows a cancel button for in-flight jobs and invokes cancelJob", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({ jobs: [makeJob({ id: "job-run", state: JobState.RUNNING })] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<JobsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.cancelButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.jobs.cancelButton));

    await waitFor(() => {
      expect(jobsClient.cancelJob).toHaveBeenCalledWith({ id: "job-run" });
    });
  });

  it("hides the cancel button for terminal jobs", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockResolvedValueOnce(
      makeListJobsResponse({ jobs: [makeJob({ id: "job-done", state: JobState.SUCCEEDED })] }),
    );

    renderWithProviders(<JobsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.list)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.jobs.cancelButton)).not.toBeInTheDocument();
  });
});
