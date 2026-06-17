/**
 * JobsCard accessibility regression tests.
 *
 * Jobs owns its query and mutation states, so the a11y waits and mocks live
 * with the feature instead of leaking into shell-level a11y tests.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { JobState } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";
import { makeJob, makeListJobsResponse } from "./mocks/factories";
import { makeJobsMocks } from "./mocks/jobs";
import { JobsCard } from "./JobsCard";

vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return { ...actual, ...makeJobsMocks() };
});

describe("JobsCard accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state without axe violations", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockResolvedValueOnce(makeListJobsResponse());

    const { container } = renderWithProviders(<JobsCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.empty)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the list state without axe violations", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockResolvedValueOnce(
      makeListJobsResponse({
        jobs: [
          makeJob({ id: "job-a", state: JobState.RUNNING }),
          makeJob({ id: "job-b", state: JobState.SUCCEEDED }),
        ],
      }),
    );

    const { container } = renderWithProviders(<JobsCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.list)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });

  it("renders the error state without axe violations", async () => {
    const { jobsClient } = await import("../../api/jobs");
    vi.mocked(jobsClient.listJobs).mockRejectedValueOnce(new Error("jobs unavailable"));

    const { container } = renderWithProviders(<JobsCard />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.error)).toBeInTheDocument();
    });

    await expectNoA11yViolations(container);
  });
});
