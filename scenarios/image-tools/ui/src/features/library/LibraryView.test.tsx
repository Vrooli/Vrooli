import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

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

import { LibraryView } from "./LibraryView";
import { jobsClient } from "../../api/jobs";

const succeeded = (id: string, operation = "upscale") =>
  makeJob({ id, state: JobState.SUCCEEDED, resultRef: `out/${id}.png`, operation });

describe("LibraryView", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a grid of succeeded outputs with open buttons", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({ jobs: [succeeded("j1"), succeeded("j2", "crop")] }),
    );
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.library.item({ index: 1 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.library.item({ index: 2 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.library.open({ index: 1 }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.library.count)).toBeInTheDocument();
  });

  it("shows an empty state when there are no outputs", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.empty)).toBeInTheDocument();
    });
  });

  it("reveals bulk actions once an item is selected", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [succeeded("j1")] }));
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.library.select({ index: 1 })));
    expect(screen.getByTestId(selectors.library.downloadSelected)).toBeInTheDocument();
  });

  it("has no accessibility violations", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [succeeded("j1")] }));
    const { container } = renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
