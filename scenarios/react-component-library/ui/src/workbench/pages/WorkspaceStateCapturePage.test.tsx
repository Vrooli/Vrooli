import { act, cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Route, Routes } from "react-router-dom";

import { getWorkspace } from "../../api/workspaces";
import { renderWithProviders } from "../../test-utils";
import { useComponentBundle } from "../data";
import { WorkspaceStateCapturePage } from "./WorkspaceStateCapturePage";

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return { ...actual, getWorkspace: vi.fn() };
});

vi.mock("../data", () => ({ useComponentBundle: vi.fn() }));
vi.mock("../component/FixtureFrame", () => ({
  FixtureFrame: ({
    assetId,
    onStatus,
  }: {
    assetId: string;
    onStatus: (status: string) => void;
  }) => (
    <div data-testid="capture-fixture-frame" data-asset-id={assetId}>
      <button onClick={() => onStatus("failed")}>Fail render</button>
      <button onClick={() => onStatus("applied")}>Apply render</button>
    </div>
  ),
}));

const mockedGetWorkspace = vi.mocked(getWorkspace);
const mockedUseComponentBundle = vi.mocked(useComponentBundle);

function makeWorkspace(
  spec: Record<string, unknown> = { id: "root", type: "page", props: {}, children: [] },
  revision = 3,
) {
  return {
    id: "ws_demo",
    revision,
    themeJson: "{}",
    optionsJson: "{}",
    pages: [
      {
        id: "pg_demo",
        name: "Welcome",
        specJson: "{}",
        displays: [
          {
            id: "dp_phone",
            device: 3,
            width: 390,
            height: 844,
            states: [
              {
                id: "st_ready",
                name: "Ready",
                argsJson: JSON.stringify({ spec }),
                scenarioDataJson: "{}",
                status: 2,
                captureId: "",
                sortOrder: 0,
              },
            ],
          },
        ],
      },
    ],
  };
}

function renderCapture() {
  return renderWithProviders(
    <Routes>
      <Route path="/workspace-capture/:id" element={<WorkspaceStateCapturePage />} />
    </Routes>,
    {
      initialEntries: [
        "/workspace-capture/ws_demo?revision=3&page=pg_demo&display=dp_phone&state=st_ready",
      ],
    },
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkspaceStateCapturePage", () => {
  it("does not substitute another story when the stored state story is missing", async () => {
    mockedGetWorkspace.mockResolvedValue(
      makeWorkspace({
        id: "root",
        type: "page",
        props: {},
        children: [
          {
            id: "bound-slot",
            type: "slot",
            props: {
              componentId: "cmp_button",
              storyId: "removed-story",
              componentVersion: "0.9.0",
            },
          },
        ],
      }) as never,
    );
    mockedUseComponentBundle.mockReturnValue({
      isLoading: false,
      data: {
        asset: { id: "cmp_button", latestVersion: "1.0.0", version: "1.0.0" },
        stories: [{ id: "remaining-story", name: "Other story" }],
      },
    } as never);

    renderCapture();

    expect(await screen.findByRole("alert")).toBeInTheDocument();
    expect(mockedUseComponentBundle).toHaveBeenCalledWith("cmp_button", "0.9.0");
    expect(screen.queryByTestId("capture-fixture-frame")).not.toBeInTheDocument();
    expect(screen.getByTestId("workspace-state-composition")).not.toHaveAttribute(
      "data-capture-ready",
      "true",
    );
  });

  it("shows the loading state until the requested workspace resolves", async () => {
    let resolveWorkspace!: (value: unknown) => void;
    mockedGetWorkspace.mockReturnValue(
      new Promise((resolve) => {
        resolveWorkspace = resolve;
      }) as never,
    );

    renderCapture();
    expect(screen.getByRole("status")).toBeInTheDocument();

    await act(async () => {
      resolveWorkspace(makeWorkspace());
      await Promise.resolve();
    });

    expect(await screen.findByTestId("workspace-state-composition")).toHaveAttribute(
      "data-capture-ready",
      "true",
    );
    expect(screen.getByTestId("workspace-state-composition")).toHaveStyle({
      width: "390px",
      height: "844px",
    });
  });

  it.each([
    ["stale revision", makeWorkspace(undefined, 2)],
    ["missing page", { ...makeWorkspace(), pages: [] }],
  ])("blocks capture when the workspace has a %s", async (_label, workspace) => {
    mockedGetWorkspace.mockResolvedValue(workspace as never);
    renderCapture();

    expect(await screen.findByTestId("workspace-capture-error")).toHaveTextContent(
      "The selected workspace revision or state is stale.",
    );
    expect(screen.queryByTestId("workspace-state-composition")).not.toBeInTheDocument();
  });

  it("reports an API error and a non-Error rejection without rendering a composition", async () => {
    mockedGetWorkspace.mockRejectedValueOnce(new Error("capture source offline"));
    renderCapture();
    expect(await screen.findByTestId("workspace-capture-error")).toHaveTextContent(
      "capture source offline",
    );

    cleanup();
    mockedGetWorkspace.mockRejectedValueOnce("unexpected");
    renderCapture();
    expect(await screen.findByTestId("workspace-capture-error")).toHaveTextContent(
      "Workspace state could not be loaded.",
    );
  });

  it("waits for a pinned component bundle and only marks capture ready after its frame applies", async () => {
    mockedGetWorkspace.mockResolvedValue(
      makeWorkspace({
        id: "root",
        type: "page",
        props: {},
        children: [
          {
            id: "bound-slot",
            type: "slot",
            props: { componentId: "cmp_button", storyId: "story_primary" },
          },
        ],
      }) as never,
    );
    mockedUseComponentBundle.mockReturnValue({ isLoading: true, data: undefined } as never);

    renderCapture();
    await waitFor(() =>
      expect(document.querySelector('[data-capture-pending="true"]')).toBeInTheDocument(),
    );
    expect(screen.getByTestId("workspace-state-composition")).not.toHaveAttribute(
      "data-capture-ready",
      "true",
    );

    mockedUseComponentBundle.mockReturnValue({
      isLoading: false,
      data: {
        asset: { id: "cmp_button", latestVersion: "1.0.0", version: "1.0.0" },
        stories: [{ id: "story_primary", name: "Primary", version: "1.0.0" }],
      },
    } as never);
    await waitFor(() =>
      expect(mockedUseComponentBundle).toHaveBeenCalledWith("cmp_button", undefined),
    );
    // Model the query hook's completed response with a fresh render.
    cleanup();
    renderCapture();
    expect(await screen.findByTestId("capture-fixture-frame")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Fail render" }));
    expect(screen.getByTestId("workspace-state-composition")).not.toHaveAttribute(
      "data-capture-ready",
      "true",
    );
    fireEvent.click(screen.getByRole("button", { name: "Apply render" }));
    await waitFor(() =>
      expect(screen.getByTestId("workspace-state-composition")).toHaveAttribute(
        "data-capture-ready",
        "true",
      ),
    );
  });

  it("waits until every bound component frame applies before capture is ready", async () => {
    mockedGetWorkspace.mockResolvedValue(
      makeWorkspace({
        id: "root",
        type: "page",
        props: {},
        children: [
          { id: "first", type: "slot", props: { componentId: "cmp_first", storyId: "first" } },
          { id: "second", type: "slot", props: { componentId: "cmp_second", storyId: "second" } },
        ],
      }) as never,
    );
    mockedUseComponentBundle.mockReturnValue({
      isLoading: false,
      data: {
        asset: { id: "component", latestVersion: "1.0.0", version: "1.0.0" },
        stories: [
          { id: "first", name: "Story", version: "1.0.0" },
          { id: "second", name: "Story", version: "1.0.0" },
        ],
      },
    } as never);

    renderCapture();

    const frames = await screen.findAllByTestId("capture-fixture-frame");
    expect(frames).toHaveLength(2);
    fireEvent.click(within(frames[0] as HTMLElement).getByRole("button", { name: "Apply render" }));
    expect(screen.getByTestId("workspace-state-composition")).not.toHaveAttribute(
      "data-capture-ready",
      "true",
    );
    fireEvent.click(within(frames[1] as HTMLElement).getByRole("button", { name: "Apply render" }));
    await waitFor(() =>
      expect(screen.getByTestId("workspace-state-composition")).toHaveAttribute(
        "data-capture-ready",
        "true",
      ),
    );
  });

  it("keeps capture blocked when the component bundle cannot be loaded", async () => {
    mockedGetWorkspace.mockResolvedValue(
      makeWorkspace({
        id: "root",
        type: "page",
        props: {},
        children: [
          {
            id: "bound-slot",
            type: "slot",
            props: { componentId: "cmp_button" },
          },
        ],
      }) as never,
    );
    mockedUseComponentBundle.mockReturnValue({ isLoading: false, data: undefined } as never);

    renderCapture();

    expect(await screen.findByText(/component/i)).toBeInTheDocument();
    expect(screen.getByTestId("workspace-state-composition")).not.toHaveAttribute(
      "data-capture-ready",
      "true",
    );
    expect(screen.queryByTestId("capture-fixture-frame")).not.toBeInTheDocument();
  });
});
