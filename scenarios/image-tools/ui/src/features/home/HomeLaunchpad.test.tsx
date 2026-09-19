import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { makeHealthResponse } from "../../test-utils/factories";
import { JobState } from "../../api/jobs";
import { makeJob, makeListJobsResponse } from "../jobs/mocks/factories";
import { selectors } from "../../consts/selectors";
import { resetWorkspaceIntent, takeWorkspaceIntent } from "../workspace/workspaceIntent";
import { DEFAULT_SAMPLE, SAMPLES } from "./samples";

vi.mock("../../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/health")>();
  return { ...actual, fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()) };
});

vi.mock("../../api/jobs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/jobs")>();
  return {
    ...actual,
    jobsClient: { listJobs: vi.fn(), cancelJob: vi.fn(), watchJob: vi.fn() },
  };
});

// `fetchBlob` is exercised by the reopen path; stub it so the recent-rail
// "reopen" click doesn't reach the network. `blobUrl` stays real (a pure URL
// builder used for thumbnails).
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return {
    ...actual,
    fetchBlob: vi.fn().mockResolvedValue(new Blob([new Uint8Array([1])], { type: "image/png" })),
  };
});

// Sample loading does a real `fetch` of a bundled asset; stub the loader so
// clicking a sample tile stages an intent without touching the network.
const loadSampleFile = vi.fn();
vi.mock("./samples", async (importOriginal) => {
  const actual = await importOriginal<typeof import("./samples")>();
  return { ...actual, loadSampleFile: (...args: unknown[]) => loadSampleFile(...args) };
});

import { HomeLaunchpad } from "./HomeLaunchpad";
import { jobsClient } from "../../api/jobs";

const pngFile = () => new File([new Uint8Array([1, 2, 3])], "x.png", { type: "image/png" });

describe("HomeLaunchpad", () => {
  beforeEach(() => {
    resetWorkspaceIntent();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
    loadSampleFile.mockResolvedValue(pngFile());
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the hero, the universal entry, and an intent tile per group", async () => {
    renderWithProviders(<HomeLaunchpad />);
    expect(screen.getByTestId(selectors.home.hero)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.home.dropzone)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.home.chooseButton)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.home.sampleButton)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.home.tile({ name: "crop" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.home.tile({ name: "background_removal" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.home.tile({ name: "text_to_image" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.home.tile({ name: "analyze" }))).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.recentEmpty)).toBeInTheDocument();
    });
  });

  it("a quick-edit tile stages an edit-mode intent for the workspace", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    await user.click(screen.getByTestId(selectors.home.tile({ name: "crop" })));
    const intent = takeWorkspaceIntent();
    expect(intent?.mode).toBe("edit");
    expect(intent?.operation).toBe("crop");
  });

  it("an enhance tile stages an enhance-mode intent carrying the chosen action", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    await user.click(screen.getByTestId(selectors.home.tile({ name: "upscale" })));
    const intent = takeWorkspaceIntent();
    expect(intent?.mode).toBe("enhance");
    expect(intent?.operation).toBe("upscale");
  });

  it("the create tile stages a create-mode text_to_image intent", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    await user.click(screen.getByTestId(selectors.home.tile({ name: "text_to_image" })));
    const intent = takeWorkspaceIntent();
    expect(intent?.mode).toBe("create");
    expect(intent?.operation).toBe("text_to_image");
  });

  it("the analyze tile stages an analyze-mode intent with no operation", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    await user.click(screen.getByTestId(selectors.home.tile({ name: "analyze" })));
    const intent = takeWorkspaceIntent();
    expect(intent?.mode).toBe("analyze");
    expect(intent?.operation).toBeUndefined();
  });

  it("choosing a file from the picker opens it in edit mode and dismisses first-run", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    // First-run hint shows when there are no recent outputs.
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.firstRun)).toBeInTheDocument();
    });
    await user.upload(screen.getByTestId(selectors.home.fileInput), pngFile());
    const intent = takeWorkspaceIntent();
    expect(intent?.mode).toBe("edit");
    expect(intent?.file).toBeInstanceOf(File);
    // Picking a file marks the launchpad seen → first-run hint disappears.
    expect(screen.queryByTestId(selectors.home.firstRun)).not.toBeInTheDocument();
    // The seen marker persists to localStorage so a return visit skips the hint.
    expect(window.localStorage.getItem("lume.home.seen")).toBe("1");
  });

  it("dropping a file onto the dropzone opens it in edit mode", () => {
    renderWithProviders(<HomeLaunchpad />);
    const dropzone = screen.getByTestId(selectors.home.dropzone);
    fireEvent.dragOver(dropzone, { dataTransfer: { files: [] } });
    fireEvent.drop(dropzone, { dataTransfer: { files: [pngFile()] } });
    const intent = takeWorkspaceIntent();
    expect(intent?.mode).toBe("edit");
    expect(intent?.file).toBeInstanceOf(File);
  });

  it("a drop with no file is a no-op (no intent staged)", () => {
    renderWithProviders(<HomeLaunchpad />);
    const dropzone = screen.getByTestId(selectors.home.dropzone);
    fireEvent.drop(dropzone, { dataTransfer: { files: [] } });
    expect(takeWorkspaceIntent()).toBeNull();
  });

  it("pasting an image anywhere on Home opens it", async () => {
    renderWithProviders(<HomeLaunchpad />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.root)).toBeInTheDocument();
    });
    const event = new Event("paste") as ClipboardEvent;
    Object.defineProperty(event, "clipboardData", { value: { files: [pngFile()] } });
    fireEvent(window, event);
    const intent = takeWorkspaceIntent();
    expect(intent?.file).toBeInstanceOf(File);
  });

  it("pasting non-image content is ignored", async () => {
    renderWithProviders(<HomeLaunchpad />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.root)).toBeInTheDocument();
    });
    const textFile = new File(["hi"], "n.txt", { type: "text/plain" });
    const event = new Event("paste") as ClipboardEvent;
    Object.defineProperty(event, "clipboardData", { value: { files: [textFile] } });
    fireEvent(window, event);
    expect(takeWorkspaceIntent()).toBeNull();
  });

  it("the sample button loads the default sample and opens it in its mode", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    await user.click(screen.getByTestId(selectors.home.sampleButton));
    await waitFor(() => {
      expect(loadSampleFile).toHaveBeenCalledWith(DEFAULT_SAMPLE);
    });
    await waitFor(() => {
      const intent = takeWorkspaceIntent();
      expect(intent?.mode).toBe(DEFAULT_SAMPLE.mode);
    });
  });

  it("a sample tile loads that sample (here: the analyze receipt opens analyze mode)", async () => {
    const user = userEvent.setup();
    const receipt = SAMPLES.find((s) => s.key === "receipt");
    expect(receipt).toBeDefined();
    renderWithProviders(<HomeLaunchpad />);
    await user.click(screen.getByTestId(selectors.home.sample({ key: "receipt" })));
    await waitFor(() => {
      expect(loadSampleFile).toHaveBeenCalledWith(receipt);
    });
    await waitFor(() => {
      expect(takeWorkspaceIntent()?.mode).toBe("analyze");
    });
  });

  it("renders the recent rail and reopens an output on click", async () => {
    const user = userEvent.setup();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "r1", state: JobState.SUCCEEDED, resultRef: "out/r1.png", operation: "upscale" })],
      }),
    );
    renderWithProviders(<HomeLaunchpad />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.recentList)).toBeInTheDocument();
    });
    // No recent items ⇒ no first-run hint and no empty state.
    expect(screen.queryByTestId(selectors.home.recentEmpty)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.home.firstRun)).not.toBeInTheDocument();

    const item = screen.getByTestId(selectors.home.recentItem({ index: 1 }));
    await user.click(item);
    // Reopen navigates to the workspace with an edit intent built from the blob.
    await waitFor(() => {
      expect(takeWorkspaceIntent()?.mode).toBe("edit");
    });
  });

  it("hides a recent thumbnail whose image fails to load", async () => {
    vi.mocked(jobsClient.listJobs).mockResolvedValue(
      makeListJobsResponse({
        jobs: [makeJob({ id: "r1", state: JobState.SUCCEEDED, resultRef: "out/r1.png", operation: "upscale" })],
      }),
    );
    renderWithProviders(<HomeLaunchpad />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.recentList)).toBeInTheDocument();
    });
    const img = screen.getByTestId(selectors.home.recentList).querySelector("img");
    expect(img).not.toBeNull();
    fireEvent.error(img as HTMLImageElement);
    // The onError handler hides the <li> ancestor of the broken thumbnail.
    expect((img as HTMLImageElement).closest("li")?.getAttribute("hidden")).toBe("true");
  });

  it("the dropzone, choose, and camera buttons open their hidden file inputs; drag-leave resets state", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);

    // Both the dropzone and the explicit Choose button proxy a click to the
    // hidden file input; the camera button proxies to the camera input.
    const fileInput = screen.getByTestId(selectors.home.fileInput);
    const cameraInput = screen.getByTestId(selectors.home.cameraInput);
    const fileClick = vi.spyOn(fileInput, "click").mockImplementation(() => {});
    const cameraClick = vi.spyOn(cameraInput, "click").mockImplementation(() => {});

    const dropzone = screen.getByTestId(selectors.home.dropzone);
    await user.click(dropzone);
    await user.click(screen.getByTestId(selectors.home.chooseButton));
    await user.click(screen.getByTestId(selectors.home.cameraButton));

    expect(fileClick).toHaveBeenCalledTimes(2);
    expect(cameraClick).toHaveBeenCalledTimes(1);

    // Drag-over activates the dropzone styling; drag-leave clears it (no throw).
    fireEvent.dragOver(dropzone, { dataTransfer: { files: [] } });
    fireEvent.dragLeave(dropzone);

    fileClick.mockRestore();
    cameraClick.mockRestore();
  });

  it("capturing from the camera input opens the image in edit mode", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    await user.upload(screen.getByTestId(selectors.home.cameraInput), pngFile());
    const intent = takeWorkspaceIntent();
    expect(intent?.mode).toBe("edit");
    expect(intent?.file).toBeInstanceOf(File);
  });

  it("the recent empty-state sample action also loads the default sample", async () => {
    const user = userEvent.setup();
    renderWithProviders(<HomeLaunchpad />);
    const empty = await screen.findByTestId(selectors.home.recentEmpty);
    await user.click(within(empty).getByRole("button"));
    await waitFor(() => {
      expect(loadSampleFile).toHaveBeenCalledWith(DEFAULT_SAMPLE);
    });
  });

  it("has no accessibility violations", async () => {
    const { container } = renderWithProviders(<HomeLaunchpad />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.recentEmpty)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
