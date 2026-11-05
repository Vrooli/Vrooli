import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { JobState } from "../../api/jobs";
import { makeJob, makeListJobsResponse } from "../jobs/mocks/factories";
import { makeApiError } from "../../api/client";
import { selectors } from "../../consts/selectors";
import { resetWorkspaceIntent, takeWorkspaceIntent } from "../workspace/workspaceIntent";

vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return {
    ...actual,
    jobsClient: { listJobs: vi.fn(), cancelJob: vi.fn(), watchJob: vi.fn() },
  };
});

// `fetchBlob` is hit by both the bulk-download path and the reopen path; stub
// it so neither reaches the network. `blobUrl` stays real (a pure URL builder).
const fetchBlob = vi.fn();
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, fetchBlob: (...args: unknown[]) => fetchBlob(...args) };
});

import { LibraryView } from "./LibraryView";
import { jobsClient } from "../../api/jobs";

const succeeded = (id: string, operation = "upscale") =>
  makeJob({ id, state: JobState.SUCCEEDED, resultRef: `out/${id}.png`, operation });

describe("LibraryView", () => {
  beforeEach(() => {
    resetWorkspaceIntent();
    fetchBlob.mockResolvedValue(new Blob([new Uint8Array([1])], { type: "image/png" }));
  });

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

  it("shows a loading state while the jobs query is in flight", () => {
    vi.mocked(jobsClient.listJobs).mockReturnValue(new Promise(() => {}));
    renderWithProviders(<LibraryView />);
    expect(screen.getByTestId(selectors.library.loading)).toBeInTheDocument();
  });

  it("shows an error state when the jobs query fails", async () => {
    vi.mocked(jobsClient.listJobs).mockRejectedValue(makeApiError("INTERNAL", "boom", 500));
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.error)).toBeInTheDocument();
    });
  });

  it("filters by operation and shows a no-match message when nothing matches", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({ jobs: [succeeded("j1", "upscale"), succeeded("j2", "crop")] }),
    );
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
    });

    // Filtering by an operation substring narrows the grid to the matching item.
    await user.type(screen.getByTestId(selectors.library.search), "crop");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.item({ index: 1 }))).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.library.item({ index: 2 }))).not.toBeInTheDocument();

    // A query that matches nothing surfaces the no-match copy, not an empty grid.
    await user.clear(screen.getByTestId(selectors.library.search));
    await user.type(screen.getByTestId(selectors.library.search), "zzzznope");
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.noMatches)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.library.grid)).not.toBeInTheDocument();
  });

  it("select-all toggles every item, then clear-selection empties it", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({ jobs: [succeeded("j1"), succeeded("j2", "crop")] }),
    );
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.library.selectAll));
    const cb1 = screen.getByTestId<HTMLInputElement>(selectors.library.select({ index: 1 }));
    const cb2 = screen.getByTestId<HTMLInputElement>(selectors.library.select({ index: 2 }));
    expect(cb1.checked).toBe(true);
    expect(cb2.checked).toBe(true);
    expect(screen.getByTestId(selectors.library.downloadSelected)).toBeInTheDocument();

    // Select-all again (now all-selected) clears the set.
    await user.click(screen.getByTestId(selectors.library.selectAll));
    expect(cb1.checked).toBe(false);
    expect(cb2.checked).toBe(false);

    // Re-select one, then use the explicit clear-selection control.
    await user.click(cb1);
    expect(screen.getByTestId(selectors.library.clearSelection)).toBeInTheDocument();
    await user.click(screen.getByTestId(selectors.library.clearSelection));
    expect(cb1.checked).toBe(false);
    expect(screen.queryByTestId(selectors.library.downloadSelected)).not.toBeInTheDocument();
  });

  it("bulk-download fetches a blob per selected item and triggers a download", async () => {
    const user = userEvent.setup();
    // jsdom doesn't implement object-URL APIs; define only those two methods on
    // the real URL constructor (don't replace `new URL`, which buildApiUrl needs).
    const realCreate = (URL as { createObjectURL?: unknown }).createObjectURL;
    const realRevoke = (URL as { revokeObjectURL?: unknown }).revokeObjectURL;
    const createObjectURL = vi.fn().mockReturnValue("blob:mock");
    const revokeObjectURL = vi.fn();
    (URL as unknown as { createObjectURL: unknown }).createObjectURL = createObjectURL;
    (URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = revokeObjectURL;
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(() => {});

    try {
      vi.mocked(jobsClient.listJobs).mockResolvedValue(
        makeListJobsResponse({ jobs: [succeeded("j1"), succeeded("j2", "crop")] }),
      );
      renderWithProviders(<LibraryView />);
      await waitFor(() => {
        expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
      });

      await user.click(screen.getByTestId(selectors.library.selectAll));
      await user.click(screen.getByTestId(selectors.library.downloadSelected));

      await waitFor(() => {
        expect(fetchBlob).toHaveBeenCalledTimes(2);
      });
      expect(fetchBlob).toHaveBeenCalledWith("out/j1.png");
      expect(fetchBlob).toHaveBeenCalledWith("out/j2.png");
      await waitFor(() => {
        expect(clickSpy).toHaveBeenCalledTimes(2);
      });
      expect(createObjectURL).toHaveBeenCalledTimes(2);
      expect(revokeObjectURL).toHaveBeenCalledTimes(2);
    } finally {
      clickSpy.mockRestore();
      (URL as unknown as { createObjectURL: unknown }).createObjectURL = realCreate;
      (URL as unknown as { revokeObjectURL: unknown }).revokeObjectURL = realRevoke;
    }
  });

  it("the per-item open button reopens the output in the workspace", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [succeeded("j1")] }));
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.library.open({ index: 1 })));
    await waitFor(() => {
      expect(fetchBlob).toHaveBeenCalledWith("out/j1.png");
    });
    await waitFor(() => {
      expect(takeWorkspaceIntent()?.mode).toBe("edit");
    });
  });

  it("hides a thumbnail whose image fails to load", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [succeeded("j1")] }));
    renderWithProviders(<LibraryView />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.library.grid)).toBeInTheDocument();
    });
    const img = screen.getByTestId(selectors.library.grid).querySelector("img");
    expect(img).not.toBeNull();
    fireEvent.error(img as HTMLImageElement);
    expect((img as HTMLImageElement).closest("li")?.getAttribute("hidden")).toBe("true");
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
