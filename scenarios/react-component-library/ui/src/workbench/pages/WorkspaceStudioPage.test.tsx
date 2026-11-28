import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";
import type { UseQueryResult } from "@tanstack/react-query";

import {
  DisplayDevice,
  StateStatus,
  WorkspaceProvenance,
  captureWorkspaceState,
  deleteWorkspaceAsset,
  forkWorkspace,
  getWorkspace,
  listWorkspaceAssets,
  saveWorkspace,
  uploadWorkspaceAsset,
  type Workspace,
  type WorkspaceAsset,
  type WorkspaceSummary,
  WorkspaceCaptureError,
} from "../../api/workspaces";
import { i18n, setLocale } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { proposeWorkspacePatch } from "../studio/workspaceAssist";
import { readStateSpec, writeStateSpec } from "../workspaces/spec";
import * as workspaceHook from "../workspaces/useWorkspaces";
import { useComponentBundle } from "../data";
import { WorkspaceStudioPage } from "./WorkspaceStudioPage";
import App from "../../App";

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return {
    ...actual,
    getWorkspace: vi.fn(),
    captureWorkspaceState: vi.fn(),
    saveWorkspace: vi.fn(),
    listWorkspaceAssets: vi.fn(),
    uploadWorkspaceAsset: vi.fn(),
    deleteWorkspaceAsset: vi.fn(),
    forkWorkspace: vi.fn(),
  };
});

vi.mock("../templates/TemplateRenderer", () => ({
  TemplateRenderer: ({ slot }: { slot?: ReactNode }) => (
    <div data-testid="template-renderer">{slot}</div>
  ),
}));

vi.mock("../data", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../data")>();
  return { ...actual, useComponentBundle: vi.fn() };
});

vi.mock("../component/FixtureFrame", () => ({
  FixtureFrame: ({
    assetId,
    storyId,
    version,
  }: {
    assetId: string;
    storyId: string;
    version: string;
  }) => (
    <div data-testid="workspace-component-slot">
      {assetId}/{storyId}@{version}
    </div>
  ),
}));

vi.mock("../studio/workspaceAssist", () => ({ proposeWorkspacePatch: vi.fn() }));

// The capture-history panel has its own transport and focused tests. Keeping it
// out of this editor test prevents its real Connect requests from generating
// unrelated alerts when this suite verifies save conflicts.
vi.mock("../studio/VisualDriftPanel", () => ({ VisualDriftPanel: () => null }));
vi.mock("../studio/htmlImport", () => ({ htmlTemplate: vi.fn(), parseHtml: vi.fn() }));

vi.mock("../workspaces/useWorkspaces", () => ({ useWorkspaces: vi.fn() }));
vi.mock("../WorkbenchShell", async () => {
  const { Outlet } = await import("react-router-dom");
  return { WorkbenchShell: () => <Outlet /> };
});

const mockedGet = vi.mocked(getWorkspace);
const mockedCaptureState = vi.mocked(captureWorkspaceState);
const mockedSave = vi.mocked(saveWorkspace);
const mockedListAssets = vi.mocked(listWorkspaceAssets);
const mockedUploadAsset = vi.mocked(uploadWorkspaceAsset);
const mockedDeleteAsset = vi.mocked(deleteWorkspaceAsset);
const mockedPropose = vi.mocked(proposeWorkspacePatch);
const mockedFork = vi.mocked(forkWorkspace);
const mockedComponentBundle = vi.mocked(useComponentBundle);
const mockedUseWorkspaces = vi.mocked(workspaceHook.useWorkspaces);

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
  mockedCaptureState.mockResolvedValue({
    workspaceRevision: 4,
    captureId: "bas-exec-1",
    evidence: {
      schemaVersion: "rcl-workspace-state-capture/v1",
      requestToken: "rcl-state-token",
      attemptId: "attempt-1",
      sourceRevision: 3,
      pageId: "pg_welcome",
      displayId: "dp_desktop",
      stateId: "st_default",
      executionId: "bas-exec-1",
      capturedAt: "2026-11-20T12:00:00Z",
      screenshot: {
        reference: "bas://exec-1/screenshot",
        width: 1180,
        height: 780,
        scope: "selected workspace composition",
      },
      accessibility: { reference: "bas://exec-1/accessibility", scope: "page-wide" },
    },
  });
  mockedSave.mockImplementation((next) =>
    Promise.resolve({ ...next, revision: next.revision + 1 }),
  );
  mockedListAssets.mockResolvedValue([]);
  mockedDeleteAsset.mockResolvedValue(undefined);
  mockedComponentBundle.mockReturnValue({
    isLoading: false,
    data: {
      asset: { id: "cmp_button", latestVersion: "2.0.0", version: "2.0.0" },
      stories: [{ id: "story_primary", name: "Primary", version: "1.4.0" }],
    },
  } as ReturnType<typeof useComponentBundle>);
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

  it("captures the saved selected state and reloads captured status", async () => {
    const latest: Workspace = {
      ...workspace,
      revision: 4,
      pages: workspace.pages.map((page) => ({
        ...page,
        displays: page.displays.map((display) => ({
          ...display,
          states: display.states.map((entry) =>
            entry.id === "st_default"
              ? { ...entry, status: StateStatus.CAPTURED, captureId: "bas-exec-1" }
              : entry,
          ),
        })),
      })),
    };
    mockedGet.mockResolvedValueOnce(workspace).mockResolvedValueOnce(latest);
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    const capture = await screen.findByRole("button", { name: /Capture state/ });
    await userEvent.click(capture);
    await waitFor(() => expect(mockedCaptureState).toHaveBeenCalledTimes(1));
    expect(mockedCaptureState).toHaveBeenCalledWith(
      "ws_harbor",
      expect.objectContaining({
        expectedRevision: 3,
        pageId: "pg_welcome",
        displayId: "dp_desktop",
        stateId: "st_default",
        attemptId: expect.any(String),
      }),
    );
    expect(await screen.findByRole("button", { name: /Capture again/ })).toBeInTheDocument();
    expect(await screen.findByTestId(selectors.workspaceStudio.notice)).toBeInTheDocument();
  });

  it("keeps the attempt token after an unresolved response so a retry recovers the same capture", async () => {
    mockedCaptureState
      .mockRejectedValueOnce(
        new WorkspaceCaptureError("Capture is still being reconciled.", "in_progress"),
      )
      .mockResolvedValueOnce({
        workspaceRevision: 4,
        captureId: "bas-exec-1",
        evidence: {
          schemaVersion: "rcl-workspace-state-capture/v1",
          requestToken: "token",
          attemptId: "attempt",
          sourceRevision: 3,
          pageId: "pg_welcome",
          displayId: "dp_desktop",
          stateId: "st_default",
          executionId: "bas-exec-1",
          capturedAt: "2026-11-20T12:00:00Z",
          screenshot: {
            reference: "bas://exec-1/screenshot",
            width: 1180,
            height: 780,
            scope: "selected workspace composition",
          },
          accessibility: { reference: "bas://exec-1/accessibility", scope: "page-wide" },
        },
      });
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    const capture = await screen.findByRole("button", { name: /Capture state/ });
    await userEvent.click(capture);
    const pending = await screen.findByTestId(selectors.workspaceStudio.notice);
    expect(pending).toHaveTextContent("Capture is still being reconciled.");
    expect(pending.closest('[role="status"]')).toBeInTheDocument();
    await userEvent.click(capture);
    await waitFor(() => expect(mockedCaptureState).toHaveBeenCalledTimes(2));
    expect(mockedCaptureState.mock.calls[1]?.[1].attemptId).toBe(
      mockedCaptureState.mock.calls[0]?.[1].attemptId,
    );
  });

  it("starts a fresh attempt after BAS reports a terminal failure", async () => {
    mockedCaptureState
      .mockRejectedValueOnce(new WorkspaceCaptureError("BAS capture failed.", "failed"))
      .mockResolvedValueOnce({
        workspaceRevision: 4,
        captureId: "bas-exec-retry",
        evidence: {
          schemaVersion: "rcl-workspace-state-capture/v1",
          requestToken: "token",
          attemptId: "attempt",
          sourceRevision: 3,
          pageId: "pg_welcome",
          displayId: "dp_desktop",
          stateId: "st_default",
          executionId: "bas-exec-retry",
          capturedAt: "2026-11-20T12:00:00Z",
          screenshot: {
            reference: "bas://exec-retry/screenshot",
            width: 1180,
            height: 780,
            scope: "selected workspace composition",
          },
          accessibility: { reference: "bas://exec-retry/accessibility", scope: "page-wide" },
        },
      });
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    const capture = await screen.findByRole("button", { name: /Capture state/ });
    await userEvent.click(capture);
    const failed = await screen.findByTestId(selectors.workspaceStudio.notice);
    expect(failed).toHaveTextContent("BAS capture failed.");
    expect(failed.closest('[role="status"]')).toBeInTheDocument();
    await userEvent.click(capture);
    await waitFor(() => expect(mockedCaptureState).toHaveBeenCalledTimes(2));
    expect(mockedCaptureState.mock.calls[1]?.[1].attemptId).not.toBe(
      mockedCaptureState.mock.calls[0]?.[1].attemptId,
    );
  });

  it("keeps layer selection reachable and operable from the keyboard", async () => {
    const user = userEvent.setup();
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tree", { name: /Layers/i });
    const rows = screen.getAllByRole("treeitem");
    const firstRow = rows[0];
    if (!firstRow) throw new Error("The workspace canvas must expose its root layer");
    const layerControl = firstRow.querySelector("button.studio-tree-select");
    if (!(layerControl instanceof HTMLButtonElement)) {
      throw new Error("Each layer must have a keyboard-operable selection button");
    }

    layerControl.focus();
    expect(layerControl).toHaveFocus();
    await user.keyboard("{Enter}");
    expect(firstRow).toHaveAttribute("aria-selected", "true");

    const frames = document.querySelectorAll<HTMLButtonElement>(".studio-frame-select");
    const secondaryFrame = frames[1];
    if (!secondaryFrame) throw new Error("The workspace canvas must expose its alternate frame");
    secondaryFrame.focus();
    expect(secondaryFrame).toHaveFocus();
    await user.keyboard("{Enter}");
    expect(secondaryFrame).toHaveAttribute("aria-pressed", "true");
  });

  it("reopens the component and story persisted on a workspace slot", async () => {
    const slot = {
      id: "root",
      type: "page",
      props: {},
      children: [
        {
          id: "slot-primary",
          type: "slot",
          props: {
            componentId: "cmp_button",
            storyId: "story_primary",
            componentVersion: "1.4.0",
          },
        },
      ],
    };
    mockedGet.mockResolvedValue({
      ...workspace,
      pages: [
        { ...workspace.pages[0]!, specJson: JSON.stringify(slot) },
        ...workspace.pages.slice(1),
      ],
    });

    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);

    await waitFor(() => expect(mockedComponentBundle).toHaveBeenCalledWith("cmp_button", "1.4.0"));
    expect(screen.getAllByTestId("workspace-component-slot")).toHaveLength(2);
    expect(screen.getAllByTestId("workspace-component-slot")[0]).toHaveTextContent(
      "cmp_button/story_primary@1.4.0",
    );
    expect(mockedComponentBundle).toHaveBeenCalledWith("cmp_button", "1.4.0");
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

  it("clones the active composition into a new device display without its capture receipt", async () => {
    const source = structuredClone(workspace);
    const page = source.pages[0]!;
    const desktop = page.displays[0]!;
    page.displays = [desktop];
    desktop.states[0] = {
      ...desktop.states[0]!,
      argsJson: '{"mode":"preview"}',
      scenarioDataJson: '{"message":"Keep this composition"}',
      status: StateStatus.CAPTURED,
      captureId: "capture_source",
    };
    writeStateSpec(desktop.states[0], {
      id: "source-root",
      type: "page",
      props: { css: "--accent: #cc6622" },
      children: [],
    });
    mockedGet.mockResolvedValue(source);

    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);
    await screen.findByRole("tab", { name: /Welcome/ });
    await userEvent.click(screen.getByRole("button", { name: "Phone" }));
    await waitFor(() => expect(mockedSave).toHaveBeenCalled(), { timeout: 3000 });

    const saved = mockedSave.mock.calls[0]?.[0];
    const phone = saved?.pages[0]?.displays.find((entry) => entry.device === DisplayDevice.PHONE);
    expect(phone).toBeDefined();
    if (!phone) throw new Error("New phone display was not saved");
    expect(phone.states).toHaveLength(1);
    const sourceState = desktop.states[0];
    const clonedState = phone.states[0];
    if (!clonedState) throw new Error("New phone display has no default state");
    expect(clonedState).toMatchObject({
      name: sourceState.name,
      scenarioDataJson: '{"message":"Keep this composition"}',
      status: StateStatus.DRAFT,
      captureId: "",
    });
    expect(JSON.parse(clonedState.argsJson).mode).toBe("preview");
    expect(clonedState.id).not.toBe(sourceState.id);
    expect(readStateSpec(clonedState)).toMatchObject({
      type: "page",
      id: expect.not.stringMatching(/^source-root$/),
      props: { css: "--accent: #cc6622" },
    });
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
    await waitFor(() => expect(mockedDeleteAsset).toHaveBeenCalledWith("ws_harbor", "asset_hero"));
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

    expect(await screen.findByRole("status", { name: /Curated template/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Save workspace/ })).toBeDisabled();
    // Editing panes are inert; the canvas stays inspectable so the template can
    // be judged before forking.
    expect(document.querySelector(".studio-left")).toHaveAttribute("inert");
    expect(document.querySelector(".studio-center")).not.toHaveAttribute("inert");

    await userEvent.click(screen.getByRole("button", { name: /Fork to edit/ }));
    await waitFor(() => expect(mockedFork).toHaveBeenCalledWith("ws_harbor"));
  });

  it("explains that an EX9 capture is evidence beside an editable draft", async () => {
    mockedGet.mockResolvedValue({
      ...workspace,
      tags: ["ex9-capture", "editable-reference"],
      summary: "EX9 capture demo /incidents · build build-1 · viewport 1280×720",
    });
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);

    expect(await screen.findByText(/EX9 capture demo \/incidents/)).toBeInTheDocument();
    expect(screen.getByText(/L0 screenshot is captured evidence/i)).toBeInTheDocument();
    expect(screen.getByText(/draft, not a composited screenshot/i)).toBeInTheDocument();
  });

  it("keeps the design brief visible while the page is being edited", async () => {
    mockedGet.mockResolvedValue({
      ...workspace,
      tags: ["design-brief"],
      summary: "Purpose: Review incidents · Target user: On-call engineer · Main action: Assign an incident",
    });
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_harbor" />);

    expect(await screen.findByRole("status", { name: "Design brief" })).toBeInTheDocument();
    expect(
      screen.getByText(/Purpose: Review incidents · Target user: On-call engineer/),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/Keep the purpose, target user, and main action in view/),
    ).toBeInTheDocument();
  });

  it("takes a brief through App routing into the persisted Studio brief banner", async () => {
    const curatedSummary = {
      $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceSummary" as const,
      id: "ws_curated",
      name: "Reviewed starter",
      summary: "A reviewed workspace template",
      provenance: WorkspaceProvenance.CURATED,
      tags: [],
      bindingScenario: "",
      bindingPage: "",
      pageCount: 2,
      displayCount: 2,
      stateCount: 2,
      revision: 1,
      createdAt: "2026-11-21T00:00:00Z",
      updatedAt: "2026-11-21T00:00:00Z",
      originWorkspaceId: "",
    } satisfies WorkspaceSummary;
    mockedUseWorkspaces.mockReturnValue({
      data: [curatedSummary],
      error: null,
      isError: false,
      isLoading: false,
    } as unknown as UseQueryResult<WorkspaceSummary[]>);

    const forked = {
      ...workspace,
      id: "ws_brief",
      name: "Incident review",
      summary: "",
      tags: [],
      revision: 1,
      originWorkspaceId: "ws_curated",
    };
    const savedBrief = {
      ...forked,
      tags: ["design-brief"],
      summary:
        "Reviewed RCL workspace template ws_curated@1. Purpose: Review incidents Target user: On-call engineer Main action: Assign an incident",
    };
    mockedFork.mockResolvedValue(forked);
    mockedSave.mockImplementation((next) =>
      Promise.resolve({ ...next, revision: next.revision + 1 }),
    );
    mockedGet.mockImplementation((id) => Promise.resolve(id === "ws_brief" ? savedBrief : workspace));

    renderWithProviders(<App />, { routerEntries: ["/design"] });
    await screen.findByTestId("design-index-page");
    fireEvent.change(screen.getByLabelText(i18n.t(strings.workbench.designIndex.pageName)), {
      target: { value: "Incident review" },
    });
    fireEvent.change(screen.getByLabelText(i18n.t(strings.workbench.designIndex.purposeLabel)), {
      target: { value: "Review incidents" },
    });
    fireEvent.change(screen.getByLabelText(i18n.t(strings.workbench.designIndex.userLabel)), {
      target: { value: "On-call engineer" },
    });
    fireEvent.change(screen.getByLabelText(i18n.t(strings.workbench.designIndex.actionLabel)), {
      target: { value: "Assign an incident" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: i18n.t(strings.workbench.designIndex.buildFromBrief) }),
    );

    expect(await screen.findByRole("status", { name: "Design brief" })).toBeInTheDocument();
    expect(
      screen.getByText(/Purpose: Review incidents Target user: On-call engineer Main action: Assign an incident/),
    ).toBeInTheDocument();
    expect(mockedFork).toHaveBeenCalledTimes(1);
    expect(mockedSave).toHaveBeenCalledWith(
      expect.objectContaining({ id: "ws_brief", tags: expect.arrayContaining(["design-brief"]) }),
      1,
    );
  });

  it("reports an unavailable workspace without pretending it loaded", async () => {
    mockedGet.mockResolvedValue(null);
    renderWithProviders(<WorkspaceStudioPage workspaceId="ws_missing" />);
    expect(await screen.findByText(/Workspace unavailable/i)).toBeInTheDocument();
  });
});
