import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { JobState } from "../../api/jobs";
import { makeJob, makeListJobsResponse } from "../jobs/mocks/factories";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return {
    ...actual,
    jobsClient: { listJobs: vi.fn(), cancelJob: vi.fn(), watchJob: vi.fn() },
  };
});

import { ActivityCard } from "./ActivityCard";
import { jobsClient } from "../../api/jobs";

describe("ActivityCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists jobs with friendly labels, open-output, and error detail", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [
          makeJob({ id: "j1", state: JobState.SUCCEEDED, resultRef: "out/a.png", operation: "upscale" }),
          makeJob({ id: "j2", state: JobState.FAILED, error: "backend exploded", operation: "resize" }),
        ],
      }),
    );
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.list)).toBeInTheDocument();
    });
    expect(screen.getAllByTestId(selectors.jobs.operation)).toHaveLength(2);
    expect(screen.getByTestId(selectors.activity.openOutput({ index: 1 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.jobs.errorDetail)).toBeInTheDocument();
  });

  it("shows an empty state when there are no jobs", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.empty)).toBeInTheDocument();
    });
  });

  it("has no accessibility violations", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "j1", state: JobState.SUCCEEDED, resultRef: "out/a.png", operation: "upscale" })],
      }),
    );
    const { container } = renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.list)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
