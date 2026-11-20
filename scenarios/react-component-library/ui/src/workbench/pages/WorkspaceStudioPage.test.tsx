import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import {
  DisplayDevice,
  StateStatus,
  WorkspaceProvenance,
  deleteWorkspaceAsset,
  forkWorkspace,
  getWorkspace,
  listWorkspaceAssets,
  saveWorkspace,
  uploadWorkspaceAsset,
  type Workspace,
  type WorkspaceAsset,
} from "../../api/workspaces";
import { setLocale } from "../../i18n";
import { renderWithProviders } from "../../test-utils";
import { proposeWorkspacePatch } from "../studio/workspaceAssist";
import { WorkspaceStudioPage } from "./WorkspaceStudioPage";

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return {
    ...actual,
    getWorkspace: vi.fn(),
    saveWorkspace: vi.fn(),
    listWorkspaceAssets: vi.fn(),
    uploadWorkspaceAsset: vi.fn(),
    deleteWorkspaceAsset: vi.fn(),
    forkWorkspace: vi.fn(),
  };
});

vi.mock("../templates/TemplateRenderer", () => ({
  TemplateRenderer: () => <div data-testid="template-renderer" />,
}));

vi.mock("../studio/workspaceAssist", () => ({ proposeWorkspacePatch: vi.fn() }));

const mockedGet = vi.mocked(getWorkspace);
const mockedSave = vi.mocked(saveWorkspace);
const mockedListAssets = vi.mocked(listWorkspaceAssets);
const mockedUploadAsset = vi.mocked(uploadWorkspaceAsset);
const mockedDeleteAsset = vi.mocked(deleteWorkspaceAsset);
const mockedPropose = vi.mocked(proposeWorkspacePatch);
const mockedFork = vi.mocked(forkWorkspace);

function asset(overrides: Partial<WorkspaceAsset> = {}): WorkspaceAsset {
  return {
    id: "asset_hero",
    workspaceId: "ws_harbor",
    name: "hero.png",
    mime: "image/png",
    bytes: 2048,
    width: 800,
    height: 600,
    fidelity: "reference",
    addedAt: "2026-11-19T12:00:00.000Z",
    url: "/api/workspaces/ws_harbor/assets/asset_hero",
    ...overrides,
  };
}

function state(id: string, name: string) {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceState" as const,
    id,
    name,
    argsJson: "{}",
    scenarioDataJson: "{}",
    status: StateStatus.READY,
    captureId: "",
    sortOrder: 0,
  };
}

const workspace: Workspace = {
  $typeName: "vrooli.react_component_library.v1.workspace.Workspace",
  id: "ws_harbor",
  name: "Harbor onboarding",
  summary: "Four-step onboarding",
  provenance: WorkspaceProvenance.AUTHORED,
  tags: ["onboarding"],
  bindingScenario: "",
  bindingPage: "",
  themeJson: "{}",
  optionsJson: "{}",
  revision: 3,
  createdAt: "2026-11-19T10:00:00Z",
  updatedAt: "2026-11-19T11:00:00Z",
  originWorkspaceId: "",
  pages: [
    {
      $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
      id: "pg_welcome",
      name: "Welcome",
      route: "/onboarding",
      pageTypeId: "templates.onboarding",
      sortOrder: 0,
      specJson: "{}",
      displays: [
        {
          $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
          id: "dp_desktop",
          device: DisplayDevice.DESKTOP,
          width: 1180,
          height: 780,
          sortOrder: 0,
          states: [state("st_default", "Default")],
        },
        {
          $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
          id: "dp_phone",
          device: DisplayDevice.PHONE,
          width: 390,
          height: 844,
          sortOrder: 1,
          states: [state("st_phone", "Default")],
        },
      ],
    },
    {
      $typeName: "vrooli.react_component_library.v1.workspace.WorkspacePage",
      id: "pg_profile",
      name: "Profile",
      route: "/onboarding/profile",
      pageTypeId: "",
      sortOrder: 1,
      specJson: "{}",
      displays: [
        {
          $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceDisplay",
          id: "dp_profile_phone",
          device: DisplayDevice.PHONE,
          width: 390,
          height: 844,
          sortOrder: 0,
          states: [state("st_profile", "Default"), state("st_error", "Validation error")],
        },
      ],
    },
  ],
};

beforeEach(async () => {
  await setLocale("en");
  mockedGet.mockResolvedValue(workspace);
  mockedSave.mockImplementation((next) => Promise.resolve({ ...next, revision: next.revision + 1 }));
  mockedListAssets.mockResolvedValue([]);
  mockedDeleteAsset.mockResolvedValue(undefined);
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkspaceStudioPage", () => {
  it("renders pages as tabs and switches the page being edited", async () => {
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    expect(await screen.findByRole("tab", { name: /Welcome/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    await userEvent.click(screen.getByRole("tab", { name: /Profile/ }));
    expect(screen.getByDisplayValue(/onboarding\/profile/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Validation error/ })).toBeInTheDocument();
  });

  it("adds a state to the active display and autosaves through the revision guard", async () => {
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });
    await userEvent.click(screen.getByRole("button", { name: /Add state/i }));
    expect(await screen.findByRole("button", { name: /State 2/ })).toBeInTheDocument();
    await waitFor(() => expect(mockedSave).toHaveBeenCalled(), { timeout: 3000 });
    const [saved, revision] = mockedSave.mock.calls[0] ?? [];
    expect(revision).toBe(3);
    expect(saved?.pages[0]?.displays[0]?.states).toHaveLength(2);
  });

  it("stops at a conflict instead of overwriting the other writer", async () => {
    mockedSave.mockRejectedValue(new ConnectError("revision conflict", Code.Aborted));
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });
    await userEvent.click(screen.getByRole("button", { name: /Add state/i }));
    expect(await screen.findByRole("alert")).toHaveTextContent(/changed elsewhere/i);
    expect(screen.getByRole("button", { name: /Keep mine/i })).toBeInTheDocument();
  });

  it("shows workspace references and imports one into the active workspace", async () => {
    mockedListAssets.mockResolvedValue([asset()]);
    mockedUploadAsset.mockResolvedValue(asset({ id: "asset_new", name: "capture.png" }));
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });

    await userEvent.click(screen.getByRole("tab", { name: /Assets/ }));
    expect(await screen.findByText(/hero\.png/)).toBeInTheDocument();

    const input = document.querySelector<HTMLInputElement>(".studio-file-input");
    expect(input).not.toBeNull();
    if (!input) return;
    const file = new File(["pixels"], "capture.png", { type: "image/png" });
    await userEvent.upload(input, file);
    await waitFor(() =>
      expect(mockedUploadAsset).toHaveBeenCalledWith("ws_harbor", file, "reference"),
    );
    expect(await screen.findByText(/capture\.png/)).toBeInTheDocument();
  });

  it("places a reference as an image layer and removes it on request", async () => {
    mockedListAssets.mockResolvedValue([asset()]);
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });

    await userEvent.click(screen.getByRole("tab", { name: /Assets/ }));
    await userEvent.click(await screen.findByRole("button", { name: /Place hero\.png/ }));
    expect(await screen.findByText(/Placed hero\.png/)).toBeInTheDocument();

    // The fixture's stored composition is empty: the first edit must write the
    // composition the person was looking at, not silently do nothing.
    await waitFor(() => expect(mockedSave).toHaveBeenCalled(), { timeout: 3000 });
    const placed = mockedSave.mock.calls.at(-1);
    expect(placed).toBeDefined();
    if (placed) {
      const args = JSON.parse(placed[0].pages[0]?.displays[0]?.states[0]?.argsJson ?? "{}") as {
        spec?: { children?: Array<{ props?: { src?: string } }> };
      };
      expect(JSON.stringify(args.spec)).toContain("/api/workspaces/ws_harbor/assets/asset_hero");
    }

    await userEvent.click(screen.getByRole("button", { name: /Remove hero\.png/ }));
    await waitFor(() =>
      expect(mockedDeleteAsset).toHaveBeenCalledWith("ws_harbor", "asset_hero"),
    );
  });

  it("applies an AI proposal to the active state as one undoable save", async () => {
    mockedPropose.mockResolvedValue({
      summary: "Warmer accent",
      operations: [{ kind: "setTheme", key: "accent", value: "#ff5500" }],
      revision: 3,
      operationId: "op-1",
      state: "completed",
      provider: "fake",
      model: "test-model",
    });
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });

    await userEvent.click(screen.getByRole("tab", { name: /AI/ }));
    await userEvent.click(screen.getByRole("button", { name: /Whole page/ }));
    await userEvent.type(screen.getByLabelText(/What should change/), "warmer accent");
    await userEvent.click(screen.getByRole("button", { name: /Propose a patch/ }));
    await userEvent.click(await screen.findByRole("button", { name: /Apply as one step/ }));

    await waitFor(() => expect(mockedSave).toHaveBeenCalled(), { timeout: 3000 });
    const call = mockedSave.mock.calls.at(-1);
    expect(call).toBeDefined();
    if (!call) return;
    expect(JSON.parse(call[0].themeJson)).toMatchObject({ accent: "#ff5500" });
    expect(mockedPropose).toHaveBeenCalledWith(
      "ws_harbor",
      expect.objectContaining({ pageId: "pg_welcome" }),
      expect.anything(),
    );
  });

  it("refuses a proposal that changed after it was requested", async () => {
    mockedPropose.mockResolvedValue({
      summary: "Warmer accent",
      operations: [{ kind: "setTheme", key: "accent", value: "#ff5500" }],
      revision: 3,
      operationId: "op-1",
      state: "completed",
    });
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });
    await userEvent.click(screen.getByRole("tab", { name: /AI/ }));
    await userEvent.click(screen.getByRole("button", { name: /Whole page/ }));
    await userEvent.type(screen.getByLabelText(/What should change/), "warmer accent");
    await userEvent.click(screen.getByRole("button", { name: /Propose a patch/ }));
    // A local edit after the request invalidates the proposal.
    await userEvent.click(screen.getByRole("tab", { name: /Profile/ }));
    expect(await screen.findByText(/changed after this proposal/i)).toBeInTheDocument();
  });

  it("does not write when only the visible state changes", async () => {
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });
    await userEvent.click(screen.getByRole("tab", { name: /Profile/ }));
    await userEvent.click(screen.getByRole("button", { name: /Validation error/ }));
    await new Promise((resolve) => globalThis.setTimeout(resolve, 1100));
    expect(mockedSave).not.toHaveBeenCalled();
  });

  it("previews a curated template read-only and forks it on request", async () => {
    mockedGet.mockResolvedValue({ ...workspace, provenance: WorkspaceProvenance.CURATED });
    mockedFork.mockResolvedValue({
      ...workspace,
      id: "ws_fork",
      provenance: WorkspaceProvenance.FORKED,
      originWorkspaceId: workspace.id,
    });
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);

    expect(
      await screen.findByRole("status", { name: /Curated template/i }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Save workspace/ })).toBeDisabled();
    // Editing panes are inert; the canvas stays inspectable so the template can
    // be judged before forking.
    expect(document.querySelector(".studio-left")).toHaveAttribute("inert");
    expect(document.querySelector(".studio-center")).not.toHaveAttribute("inert");

    await userEvent.click(screen.getByRole("button", { name: /Fork to edit/ }));
    await waitFor(() => expect(mockedFork).toHaveBeenCalledWith("ws_harbor"));
  });

  it("reports an unavailable workspace without pretending it loaded", async () => {
    mockedGet.mockResolvedValue(null);
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_missing" />);
    expect(await screen.findByText(/Workspace unavailable/i)).toBeInTheDocument();
  });
});
