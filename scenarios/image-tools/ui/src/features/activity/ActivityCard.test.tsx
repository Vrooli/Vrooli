import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { JobLane, JobState } from "../../api/jobs";
import { makeApiError } from "../../api/client";
import {
  asProgressStream,
  makeCancelJobResponse,
  makeJob,
  makeListJobsResponse,
  makeProgressEvent,
} from "../jobs/mocks/factories";
import { selectors } from "../../consts/selectors";
import { resetWorkspaceIntent, takeWorkspaceIntent } from "../workspace/workspaceIntent";

vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return {
    ...actual,
    jobsClient: { listJobs: vi.fn(), cancelJob: vi.fn(), watchJob: vi.fn() },
  };
});

// `fetchBlob` backs the open-output reopen path; stub it so the click doesn't
// reach the network. `blobUrl` (a pure URL builder for thumbnails) stays real.
const fetchBlob = vi.fn();
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, fetchBlob: (...args: unknown[]) => fetchBlob(...args) };
});

import { ActivityCard } from "./ActivityCard";
import { jobsClient } from "../../api/jobs";

describe("ActivityCard", () => {
  beforeEach(() => {
    resetWorkspaceIntent();
    fetchBlob.mockResolvedValue(new Blob([new Uint8Array([1])], { type: "image/png" }));
    // Default: no live stream (each test that wants one overrides this).
    vi.mocked(jobsClient.watchJob).mockReturnValue(asProgressStream([]));
  });

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

  it("shows a loading state while the jobs query is in flight", () => {
    vi.mocked(jobsClient.listJobs).mockReturnValue(new Promise(() => {}));
    renderWithProviders(<ActivityCard />);
    expect(screen.getByTestId(selectors.activity.loading)).toBeInTheDocument();
  });

  it("shows an error state when the jobs query fails", async () => {
    vi.mocked(jobsClient.listJobs).mockRejectedValue(makeApiError("INTERNAL", "down", 500));
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.error)).toBeInTheDocument();
    });
  });

  it("renders the CPU lane chip, progress bar, and a status message for an active job", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [
          makeJob({
            id: "run1",
            state: JobState.RUNNING,
            lane: JobLane.CPU,
            progress: 40,
            message: "Falling back to CPU…",
            operation: "upscale",
          }),
        ],
      }),
    );
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.list)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.jobs.lane)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.jobs.progress)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.jobs.message)).toHaveTextContent("Falling back to CPU…");
    // An active (running) job exposes a cancel control but no open-output.
    expect(screen.getByTestId(selectors.jobs.cancelButton)).toBeInTheDocument();
  });

  it("overlays live WatchJob progress on an active job", async () => {
    vi.mocked(jobsClient.watchJob).mockReturnValue(
      asProgressStream([
        makeProgressEvent({ jobId: "run1", state: JobState.RUNNING, progress: 75, message: "75%" }),
      ]),
    );
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "run1", state: JobState.RUNNING, progress: 10, message: "queued", operation: "upscale" })],
      }),
    );
    renderWithProviders(<ActivityCard />);
    // Once the stream replays an event, the live badge appears and progress jumps.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.liveBadge)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.jobs.progress)).toHaveAttribute("value", "75");
    expect(jobsClient.watchJob).toHaveBeenCalled();
  });

  it("cancels an active job through the cancel button", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "run1", state: JobState.RUNNING, operation: "upscale" })],
      }),
    );
    vi.mocked(jobsClient.cancelJob).mockResolvedValue(makeCancelJobResponse());
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.cancelButton)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.jobs.cancelButton));
    await waitFor(() => {
      expect(jobsClient.cancelJob).toHaveBeenCalledWith({ id: "run1" });
    });
  });

  it("surfaces an error when cancelling a job fails", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "run1", state: JobState.RUNNING, operation: "upscale" })],
      }),
    );
    vi.mocked(jobsClient.cancelJob).mockRejectedValue(makeApiError("INTERNAL", "cannot cancel", 500));
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.jobs.cancelButton)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.jobs.cancelButton));
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.error)).toBeInTheDocument();
    });
  });

  it("open-output reopens a succeeded result in the workspace", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "ok", state: JobState.SUCCEEDED, resultRef: "out/ok.png", operation: "upscale" })],
      }),
    );
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.openOutput({ index: 1 }))).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.activity.openOutput({ index: 1 })));
    await waitFor(() => {
      expect(fetchBlob).toHaveBeenCalledWith("out/ok.png");
    });
    await waitFor(() => {
      expect(takeWorkspaceIntent()?.mode).toBe("edit");
    });
  });

  it("hides a result thumbnail whose image fails to load", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "ok", state: JobState.SUCCEEDED, resultRef: "out/ok.png", operation: "upscale" })],
      }),
    );
    renderWithProviders(<ActivityCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.activity.list)).toBeInTheDocument();
    });
    const img = screen.getByTestId(selectors.activity.list).querySelector("img");
    expect(img).not.toBeNull();
    fireEvent.error(img as HTMLImageElement);
    expect((img as HTMLImageElement).style.display).toBe("none");
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
