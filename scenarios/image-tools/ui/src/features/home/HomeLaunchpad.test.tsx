import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { makeHealthResponse } from "../../test-utils/factories";
import { makeListJobsResponse } from "../jobs/mocks/factories";
import { selectors } from "../../consts/selectors";
import { resetWorkspaceIntent, takeWorkspaceIntent } from "../workspace/workspaceIntent";

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

import { HomeLaunchpad } from "./HomeLaunchpad";
import { jobsClient } from "../../api/jobs";

describe("HomeLaunchpad", () => {
  beforeEach(() => {
    resetWorkspaceIntent();
    vi.mocked(jobsClient.listJobs).mockResolvedValue(makeListJobsResponse({ jobs: [] }));
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

  it("has no accessibility violations", async () => {
    const { container } = renderWithProviders(<HomeLaunchpad />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.home.recentEmpty)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
