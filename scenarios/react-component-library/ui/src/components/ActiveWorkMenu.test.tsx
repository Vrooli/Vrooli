import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";

const { listWorkflows, stopWorkflow, retryWorkflow } = vi.hoisted(() => ({
  listWorkflows: vi.fn(),
  stopWorkflow: vi.fn(),
  retryWorkflow: vi.fn(),
}));

vi.mock("../api/workflows", () => ({
  workflowsClient: { listWorkflows, stopWorkflow, retryWorkflow },
}));

import { ActiveWorkMenu } from "./ActiveWorkMenu";

describe("ActiveWorkMenu", () => {
  beforeEach(() => {
    listWorkflows.mockResolvedValue({
      workflows: [
        {
          id: "running",
          assetId: "focus-panel",
          status: "running",
          canStop: true,
          canRetry: false,
          targetScenario: "demo",
        },
        {
          id: "failed",
          assetId: "focus-hook",
          status: "failed",
          canStop: false,
          canRetry: true,
          error: "dispatch unavailable",
        },
      ],
    });
    stopWorkflow.mockResolvedValue({});
    retryWorkflow.mockResolvedValue({});
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("shows attributable workflow state and exposes only its legal controls", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ActiveWorkMenu />);

    await waitFor(() => expect(screen.getByTestId("active-work-menu")).toHaveTextContent("1"));
    await user.click(screen.getByTestId("active-work-menu"));

    expect(screen.getByRole("dialog", { name: "workflows.active" })).toHaveTextContent(
      "focus-panel",
    );
    expect(screen.getByRole("dialog", { name: "workflows.active" })).toHaveTextContent(
      "dispatch unavailable",
    );
    await user.click(screen.getByRole("button", { name: "workflows.stop" }));
    await user.click(screen.getByRole("button", { name: "workflows.retry" }));

    await waitFor(() => expect(stopWorkflow).toHaveBeenCalledWith({ id: "running" }));
    expect(retryWorkflow).toHaveBeenCalledWith(
      expect.objectContaining({
        id: "failed",
        idempotencyKey: expect.stringMatching(/^ui-retry:failed:/),
      }),
    );
  });

  it("treats a response without workflows as an empty workflow list", async () => {
    listWorkflows.mockResolvedValue({});

    renderWithProviders(<ActiveWorkMenu />);

    await waitFor(() => expect(screen.getByTestId("active-work-menu")).toBeInTheDocument());
  });

  it("shows an empty active-work dialog when no workflow history exists", async () => {
    listWorkflows.mockResolvedValue({ workflows: [] });
    const user = userEvent.setup();
    renderWithProviders(<ActiveWorkMenu />);
    await user.click(screen.getByTestId("active-work-menu"));
    expect(await screen.findByText("workflows.none")).toBeInTheDocument();
  });

  it("falls back to the root link for workflow work without an asset", async () => {
    listWorkflows.mockResolvedValue({
      workflows: [
        {
          id: "extract",
          status: "running",
          canStop: false,
          canRetry: false,
          sourceScenario: "demo",
          sourcePath: "ui/Button.tsx",
        },
      ],
    });
    const user = userEvent.setup();
    renderWithProviders(<ActiveWorkMenu />);
    await user.click(screen.getByTestId("active-work-menu"));
    expect(await screen.findByRole("link", { name: "demo" })).toHaveAttribute("href", "/");
  });

  it("preserves the current asset tab on workflow asset links", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ActiveWorkMenu />, { routerEntries: ["/assets/current?tab=files"] });
    await user.click(screen.getByTestId("active-work-menu"));
    expect(await screen.findByRole("link", { name: "focus-panel" })).toHaveAttribute(
      "href",
      "/assets/focus-panel?tab=files",
    );
  });
});
