import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import type { UseQueryResult } from "@tanstack/react-query";
import { webcrypto } from "node:crypto";
import { useLocation } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";

vi.mock("../workspaces/useWorkspaces", () => ({
  useWorkspaces: vi.fn(),
}));

// The brief flow does not depend on EX9 HTML parsing; isolate the capture
// importer so an unrelated capture-path source failure cannot mask these tests.
vi.mock("../studio/htmlImport", () => ({
  htmlTemplate: vi.fn((_parsed, _regionId, name, sourcePreview) => ({
    id: "html-import-test",
    name,
    tagline: "Imported HTML",
    category: "operations",
    slotHint: "main",
    idea: "Imported fixture",
    palette: {},
    display: "desktop",
    radius: 8,
    spec: {
      id: "page",
      type: "page",
      props: { sourceHtml: sourcePreview },
      children: [
        { id: "incident-title", type: "text", props: { text: "Incident overview" } },
        { id: "incident-status", type: "text", props: { text: "Current status" } },
      ],
    },
    local: true,
  })),
  parseHtml: vi.fn((source: string) => ({
    title: "Incident overview",
    layers: 2,
    sourcePreview: () => source,
    regionSpec: () => ({
      id: "page",
      type: "page",
      props: {},
      children: [
        { id: "incident-title", type: "text", props: { text: "Incident overview" } },
        { id: "incident-status", type: "text", props: { text: "Current status" } },
      ],
    }),
  })),
}));

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return {
    ...actual,
    getWorkspace: vi.fn(),
    forkWorkspace: vi.fn(),
    createWorkspace: vi.fn(),
    saveWorkspace: vi.fn(),
    uploadWorkspaceAsset: vi.fn(),
  };
});

import {
  forkWorkspace,
  createWorkspace,
  saveWorkspace,
  uploadWorkspaceAsset,
  getWorkspace,
  WorkspaceProvenance,
  DisplayDevice,
  type Workspace,
  type WorkspaceSummary,
} from "../../api/workspaces";
import { strings } from "../../consts/strings";
import { i18n, setLocale } from "../../i18n";
import * as workspaceHook from "../workspaces/useWorkspaces";
import { DesignIndexPage } from "./DesignIndexPage";

function queryResult<T>(
  value: T | undefined,
  options: { isLoading?: boolean; isError?: boolean } = {},
): UseQueryResult<T> {
  return {
    data: value,
    error: options.isError ? new Error("unavailable") : null,
    isError: options.isError ?? false,
    isLoading: options.isLoading ?? false,
    isFetching: false,
    isSuccess: !options.isError && value !== undefined,
    refetch: vi.fn(),
  } as unknown as UseQueryResult<T>;
}

const WORKSPACE_NAME = "Harbor Workspace";
const CURATED_NAME = "Curated onboarding";
const FORK_ERROR = "Template service is unavailable.";
const CREATE_ERROR = "Workspace service is unavailable.";
const PENDING_BRIEF_STORAGE_KEY = "rcl.workbench.design-brief.pending.v1";
const DESIGN_PATH_COPY = {
  briefHeading: strings.workbench.designIndex.briefHeading,
  captureHeading: strings.workbench.designIndex.captureHeading,
  pageName: strings.workbench.designIndex.pageName,
  purpose: strings.workbench.designIndex.purposeLabel,
  user: strings.workbench.designIndex.userLabel,
  action: strings.workbench.designIndex.actionLabel,
  manifest: strings.workbench.designIndex.manifestLabel,
  artifacts: strings.workbench.designIndex.artifactsLabel,
  template: strings.workbench.designIndex.reviewedTemplate,
  createRevision: strings.workbench.designIndex.createRevision,
  captureError: strings.workbench.designIndex.captureL0Required,
};

function CurrentRoute() {
  const location = useLocation();
  return <output data-testid="current-route">{`${location.pathname}${location.search}`}</output>;
}

const validCaptureManifest = () => ({
  schema_version: "ux-capture.v1",
  id: "123e4567-e89b-12d3-a456-426614174000",
  captured_at: "2026-11-21T12:00:00Z",
  source: { kind: "scenario", scenario: "demo", route: "/", build_id: "build-1" },
  session: { id: "session-1" },
  viewport: { width: 1280, height: 720 },
  lenses: Object.fromEntries(
    ["L0", "L1", "L2", "L3", "L4", "L5", "L6"].map((lens) => [lens, { status: "present" }]),
  ),
  artifacts: [],
  privacy: { redaction_status: "complete" },
});

function workspaceFixture(overrides: Partial<Workspace> = {}): Workspace {
  return {
    $typeName: "vrooli.react_component_library.v1.workspace.Workspace",
    id: "ws_fixture",
    name: "Fixture workspace",
    summary: "",
    provenance: WorkspaceProvenance.AUTHORED,
    tags: [],
    bindingScenario: "",
    bindingPage: "",
    themeJson: "{}",
    optionsJson: "{}",
    revision: 1,
    createdAt: "2026-11-21T00:00:00Z",
    updatedAt: "2026-11-21T00:00:00Z",
    pages: [],
    originWorkspaceId: "",
    ...overrides,
  };
}

const workspaces: WorkspaceSummary[] = [
  {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceSummary",
    id: "ws_harbor",
    name: WORKSPACE_NAME,
    summary: "Four-step onboarding",
    provenance: WorkspaceProvenance.AUTHORED,
    tags: [],
    bindingScenario: "harbor",
    bindingPage: "welcome",
    pageCount: 4,
    displayCount: 12,
    stateCount: 16,
    revision: 2,
    createdAt: "2026-11-18T10:00:00Z",
    updatedAt: "2026-11-18T12:00:00Z",
    originWorkspaceId: "",
  },
  {
    $typeName: "vrooli.react_component_library.v1.workspace.WorkspaceSummary",
    id: "ws_curated",
    name: CURATED_NAME,
    summary: "",
    provenance: WorkspaceProvenance.CURATED,
    tags: [],
    bindingScenario: "",
    bindingPage: "",
    pageCount: 2,
    displayCount: 6,
    stateCount: 8,
    revision: 1,
    createdAt: "2026-11-17T10:00:00Z",
    updatedAt: "2026-11-17T12:00:00Z",
    originWorkspaceId: "",
  },
];

describe("DesignIndexPage", () => {
  beforeEach(() => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(queryResult(workspaces));
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    window.sessionStorage.removeItem(PENDING_BRIEF_STORAGE_KEY);
    vi.unstubAllGlobals();
    Reflect.deleteProperty(File.prototype, "arrayBuffer");
  });

  it("lists authored workspaces with counts and studio open actions", async () => {
    renderWithProviders(<DesignIndexPage />);

    expect(await screen.findByTestId("design-index-page")).toHaveAttribute(
      "data-experience-surface",
      "design-overview",
    );
    expect(await screen.findByText(WORKSPACE_NAME)).toBeInTheDocument();
    expect(screen.getAllByTestId("design-workspace-row")).toHaveLength(1);
    expect(
      screen.getByText(new RegExp(`12 ${strings.workbench.designIndex.displays}`)),
    ).toBeInTheDocument();

    const openLinks = screen.getAllByRole("link", {
      name: strings.workbench.designIndex.open,
    });
    expect(openLinks[0]).toHaveAttribute("href", "/design?workspace=ws_harbor");
  });

  it("shows curated templates and their fork action", async () => {
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);

    expect(screen.getAllByTestId("design-template-card")).toHaveLength(1);
    expect(screen.getAllByText(CURATED_NAME)).toHaveLength(2);
    expect(
      screen.getByRole("button", { name: strings.workbench.designIndex.useTemplate }),
    ).toBeInTheDocument();
  });

  it("keeps the template preview pending when a fetched workspace has no page", async () => {
    vi.mocked(getWorkspace).mockResolvedValue(
      workspaceFixture({ id: "ws_curated", name: CURATED_NAME }),
    );
    renderWithProviders(<DesignIndexPage />);
    expect(await screen.findByTestId("design-template-card")).toBeInTheDocument();
    await waitFor(() => expect(getWorkspace).toHaveBeenCalledWith("ws_curated"));
    expect(document.querySelector(".wb-template-thumb.is-pending")).toBeInTheDocument();
  });

  it("shows a fork error and lets the person retry using the template", async () => {
    vi.mocked(forkWorkspace)
      .mockRejectedValueOnce(new Error(FORK_ERROR))
      .mockResolvedValueOnce({ id: "ws_forked" } as Workspace);
    renderWithProviders(<DesignIndexPage />);

    const useTemplate = await screen.findByRole("button", {
      name: strings.workbench.designIndex.useTemplate,
    });
    fireEvent.click(useTemplate);

    expect(await screen.findByText(FORK_ERROR)).toBeInTheDocument();
    fireEvent.click(useTemplate);
    await waitFor(() => expect(forkWorkspace).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByText(FORK_ERROR)).toBeNull());
  });

  it("shows a workspace creation error and allows retry", async () => {
    vi.mocked(createWorkspace)
      .mockRejectedValueOnce(new Error(CREATE_ERROR))
      .mockResolvedValueOnce({ id: "ws_created" } as Workspace);
    renderWithProviders(<DesignIndexPage />);

    const createButton = await screen.findByRole("button", {
      name: strings.workbench.designIndex.newWorkspace,
    });
    fireEvent.click(createButton);
    expect(await screen.findByText(CREATE_ERROR)).toBeInTheDocument();

    fireEvent.click(createButton);
    await waitFor(() => expect(createWorkspace).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByText(CREATE_ERROR)).toBeNull());
  });

  it("uses translated fallback errors when workspace operations reject with non-Errors", async () => {
    vi.mocked(createWorkspace).mockRejectedValueOnce("offline");
    renderWithProviders(<DesignIndexPage />);
    fireEvent.click(
      await screen.findByRole("button", { name: strings.workbench.designIndex.newWorkspace }),
    );
    expect(await screen.findByText(strings.workbench.designIndex.createFailed)).toBeInTheDocument();
  });

  it("focuses the named-workspace action for the keyboard shortcut route", async () => {
    renderWithProviders(<DesignIndexPage />, { route: "/design?new=1" });
    const action = await screen.findByRole("button", {
      name: strings.workbench.designIndex.newWorkspace,
    });
    await waitFor(() => expect(action).toHaveFocus());
    expect(createWorkspace).not.toHaveBeenCalled();
  });

  it("moves focus to the curated shelf when its deep link is opened", async () => {
    const scrollIntoView = vi.fn();
    Object.defineProperty(HTMLElement.prototype, "scrollIntoView", {
      configurable: true,
      value: scrollIntoView,
    });
    renderWithProviders(<DesignIndexPage />, { route: "/design#curated-templates" });
    const shelf = await screen.findByRole("region", {
      name: strings.workbench.designIndex.templates,
    });
    await waitFor(() => expect(shelf).toHaveFocus());
    expect(scrollIntoView).toHaveBeenCalledWith({ block: "start" });
  });

  it("rejects duplicate uploaded names before digesting capture artifacts", async () => {
    const manifest = {
      ...validCaptureManifest(),
      artifacts: [
        {
          kind: "pixels",
          path: "screen.png",
          media_type: "image/png",
          size_bytes: 1,
          sha256: "a".repeat(64),
        },
      ],
    };
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.manifest), {
      target: { files: [new File([JSON.stringify(manifest)], "capture.json")] },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.artifacts), {
      target: { files: [new File(["a"], "screen.png"), new File(["b"], "screen.png")] },
    });
    fireEvent.click(screen.getByRole("button", { name: DESIGN_PATH_COPY.createRevision }));
    expect(
      await screen.findByText(
        strings.workbench.designIndex.captureDuplicateArtifact.replace("{{name}}", "screen.png"),
      ),
    ).toBeInTheDocument();
  });

  it("rejects an oversized uploaded evidence set before reading its files", async () => {
    const manifest = {
      ...validCaptureManifest(),
      artifacts: [
        {
          kind: "other",
          path: "evidence.bin",
          media_type: "application/octet-stream",
          size_bytes: 0,
          sha256: "a".repeat(64),
        },
      ],
    };
    const oversized = new File(["x"], "evidence.bin");
    Object.defineProperty(oversized, "size", { value: 64 * 1024 * 1024 + 1 });
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.manifest), {
      target: { files: [new File([JSON.stringify(manifest)], "capture.json")] },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.artifacts), {
      target: { files: [oversized] },
    });
    fireEvent.click(screen.getByRole("button", { name: DESIGN_PATH_COPY.createRevision }));
    expect(
      await screen.findByText(strings.workbench.designIndex.captureArtifactLimit),
    ).toBeInTheDocument();
  });

  it("shows the translated fallback when using a template fails without an Error", async () => {
    vi.mocked(forkWorkspace).mockRejectedValueOnce("offline");
    renderWithProviders(<DesignIndexPage />);
    fireEvent.click(
      await screen.findByRole("button", { name: strings.workbench.designIndex.useTemplate }),
    );
    expect(await screen.findByText(strings.workbench.designIndex.forkFailed)).toBeInTheDocument();
  });

  it("keeps the brief and EX9 revision journeys distinct from the saved-workspace shelf", async () => {
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);

    expect(screen.getByText(DESIGN_PATH_COPY.briefHeading)).toBeInTheDocument();
    expect(screen.getByText(DESIGN_PATH_COPY.captureHeading)).toBeInTheDocument();
    const templateSelect = screen
      .getByText(DESIGN_PATH_COPY.template)
      .parentElement?.querySelector("select");
    expect(templateSelect).toHaveValue("ws_curated");
    expect(screen.getByLabelText(DESIGN_PATH_COPY.purpose)).toBeInTheDocument();
    expect(screen.getByLabelText(DESIGN_PATH_COPY.user)).toBeInTheDocument();
    expect(screen.getByLabelText(DESIGN_PATH_COPY.action)).toBeInTheDocument();
    expect(screen.getByLabelText(DESIGN_PATH_COPY.manifest)).toBeInTheDocument();
    expect(screen.getByLabelText(DESIGN_PATH_COPY.artifacts)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.designIndex.captureNote)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: strings.workbench.designIndex.newWorkspace }),
    ).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.designIndex.templates)).toBeInTheDocument();
  });

  it("shows the exact local filenames selected for an EX9 revision", async () => {
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);

    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.manifest), {
      target: { files: [new File(["{}"], "capture-manifest.json")] },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.artifacts), {
      target: {
        files: [
          new File(["png"], "screen-1440.png"),
          new File(["html"], "frozen-page.html"),
        ],
      },
    });

    const selectedFiles = screen.getByRole("status", {
      name: strings.workbench.designIndex.captureSelectedFiles,
    });
    expect(selectedFiles).toBeInTheDocument();
    expect(within(selectedFiles).getAllByRole("listitem").map((item) => item.textContent)).toEqual([
      "capture-manifest.json",
      "screen-1440.png",
      "frozen-page.html",
    ]);
  });

  it("forks a reviewed RCL workspace and saves the requested purpose, user, and action", async () => {
    const forked = {
      $typeName: "vrooli.react_component_library.v1.workspace.Workspace",
      id: "ws_brief",
      name: "Incident review",
      revision: 1,
      summary: "",
      provenance: WorkspaceProvenance.FORKED,
      tags: [],
      bindingScenario: "",
      bindingPage: "",
      themeJson: "{}",
      optionsJson: "{}",
      createdAt: "2026-11-21T00:00:00Z",
      updatedAt: "2026-11-21T00:00:00Z",
      pages: [],
      originWorkspaceId: "ws_curated",
    } as Workspace;
    vi.mocked(forkWorkspace).mockResolvedValue(forked);
    vi.mocked(saveWorkspace).mockResolvedValue(forked);
    renderWithProviders(<DesignIndexPage />);

    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.pageName), {
      target: { value: "Incident review" },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.purpose), {
      target: { value: "Review incidents" },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.user), {
      target: { value: "On-call engineer" },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.action), {
      target: { value: "Assign an incident" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: strings.workbench.designIndex.buildFromBrief }),
    );

    await waitFor(() =>
      expect(forkWorkspace).toHaveBeenCalledWith("ws_curated", "Incident review"),
    );
    await waitFor(() =>
      expect(saveWorkspace).toHaveBeenCalledWith(
        expect.objectContaining({
          summary: strings.workbench.designIndex.briefSummary,
          tags: expect.arrayContaining(["design-brief"]),
        }),
        1,
      ),
    );
  });

  it("retries a rejected brief save on the same fork", async () => {
    const forked = workspaceFixture({ id: "ws_brief" });
    const refreshed = workspaceFixture({ id: "ws_brief" });
    vi.mocked(forkWorkspace).mockResolvedValue(forked);
    vi.mocked(saveWorkspace).mockRejectedValueOnce("offline");
    vi.mocked(getWorkspace).mockResolvedValue(refreshed);
    renderWithProviders(
      <>
        <CurrentRoute />
        <DesignIndexPage />
      </>,
    );
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.pageName), {
      target: { value: "Review" },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.purpose), {
      target: { value: "Review incidents" },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.user), {
      target: { value: "Operator" },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.action), {
      target: { value: "Assign" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: strings.workbench.designIndex.buildFromBrief }),
    );
    expect(await screen.findByText(strings.workbench.designIndex.briefCreateFailed)).toBeInTheDocument();
    expect(screen.getByLabelText(DESIGN_PATH_COPY.template)).toBeDisabled();
    fireEvent.click(
      screen.getByRole("button", { name: strings.workbench.designIndex.retryBrief }),
    );
    await waitFor(() => expect(saveWorkspace).toHaveBeenCalledTimes(2));
    expect(forkWorkspace).toHaveBeenCalledTimes(1);
    expect(saveWorkspace).toHaveBeenLastCalledWith(
      expect.objectContaining({ id: "ws_brief", tags: expect.arrayContaining(["design-brief"]) }),
      forked.revision,
    );
    await waitFor(() =>
      expect(screen.getByTestId("current-route")).toHaveTextContent("/design?workspace=ws_brief"),
    );
  });

  it("restores the same brief identity and workspace after returning to Design", async () => {
    await setLocale("en");
    const pending = {
      workspaceId: "ws_recovered",
      templateId: "ws_curated",
      templateRevision: 1,
      templateName: CURATED_NAME,
      name: "Incident review",
      purpose: "Review incidents",
      user: "On-call engineer",
      action: "Assign an incident",
    };
    const forked = workspaceFixture({ id: pending.workspaceId, name: pending.name });
    window.sessionStorage.setItem(PENDING_BRIEF_STORAGE_KEY, JSON.stringify(pending));
    vi.mocked(getWorkspace).mockResolvedValue(forked);
    vi.mocked(saveWorkspace).mockResolvedValue({ ...forked, revision: forked.revision + 1 });
    vi.mocked(forkWorkspace).mockClear();

    renderWithProviders(
      <>
        <CurrentRoute />
        <DesignIndexPage />
      </>,
      { route: "/design" },
    );

    const recovery = await screen.findByRole("status", {
      name: "Pending brief recovery",
    });
    expect(recovery).toHaveTextContent(pending.workspaceId);
    expect(recovery).toHaveTextContent(`${CURATED_NAME} (${pending.templateId}@1)`);
    expect(screen.getByLabelText(i18n.t(strings.workbench.designIndex.pageName))).toHaveValue(
      pending.name,
    );
    expect(
      screen.getByLabelText(i18n.t(strings.workbench.designIndex.purposeLabel)),
    ).toHaveValue(pending.purpose);
    expect(screen.getByLabelText(i18n.t(strings.workbench.designIndex.userLabel))).toHaveValue(
      pending.user,
    );
    expect(screen.getByLabelText(i18n.t(strings.workbench.designIndex.actionLabel))).toHaveValue(
      pending.action,
    );

    const resume = screen.getByRole("button", {
      name: "Retry saving this draft",
    });
    await waitFor(() => expect(resume).toBeEnabled());
    fireEvent.click(resume);
    await waitFor(() =>
      expect(saveWorkspace).toHaveBeenCalledWith(
        expect.objectContaining({ id: pending.workspaceId, tags: expect.arrayContaining(["design-brief"]) }),
        forked.revision,
      ),
    );
    expect(forkWorkspace).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(screen.getByTestId("current-route")).toHaveTextContent(
        `/design?workspace=${pending.workspaceId}`,
      ),
    );
    expect(window.sessionStorage.getItem(PENDING_BRIEF_STORAGE_KEY)).toBeNull();
  });

  it.each(["absent", "partial"] as const)(
    "rejects an EX9 capture whose L0 status is %s before creating a workspace",
    async (l0Status) => {
      const lenses = Object.fromEntries(
        ["L0", "L1", "L2", "L3", "L4", "L5", "L6"].map((lens) => [
          lens,
          lens === "L0"
            ? { status: l0Status, reason: "The screenshot is unavailable." }
            : { status: "present" },
        ]),
      );
      const manifest = {
        schema_version: "ux-capture.v1",
        id: "123e4567-e89b-12d3-a456-426614174000",
        captured_at: "2026-11-21T12:00:00Z",
        source: { kind: "scenario", scenario: "demo", route: "/", build_id: "build-1" },
        session: { id: "session-1" },
        viewport: { width: 1280, height: 720 },
        lenses,
        artifacts: [],
        privacy: { redaction_status: "complete" },
      };
      vi.mocked(forkWorkspace).mockClear();
      vi.mocked(saveWorkspace).mockClear();
      vi.mocked(uploadWorkspaceAsset).mockClear();
      renderWithProviders(<DesignIndexPage />);
      await screen.findByText(WORKSPACE_NAME);

      const manifestFile = new File([JSON.stringify(manifest)], "capture.json", {
        type: "application/json",
      });
      Object.defineProperty(manifestFile, "text", {
        value: () => Promise.resolve(JSON.stringify(manifest)),
      });
      fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.manifest), {
        target: { files: [manifestFile] },
      });
      fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.artifacts), {
        target: { files: [new File(["artifact"], "unused.bin")] },
      });
      fireEvent.click(screen.getByRole("button", { name: DESIGN_PATH_COPY.createRevision }));

      expect(await screen.findByText(DESIGN_PATH_COPY.captureError)).toBeInTheDocument();
      expect(forkWorkspace).not.toHaveBeenCalled();
      expect(saveWorkspace).not.toHaveBeenCalled();
      expect(uploadWorkspaceAsset).not.toHaveBeenCalled();
    },
  );

  it("requires L1 frozen-page evidence before starting a capture revision", async () => {
    const lenses = Object.fromEntries(
      ["L0", "L1", "L2", "L3", "L4", "L5", "L6"].map((lens) => [
        lens,
        lens === "L1"
          ? { status: "absent", reason: "The frozen page was not captured." }
          : { status: "present" },
      ]),
    );
    const manifest = {
      schema_version: "ux-capture.v1",
      id: "123e4567-e89b-12d3-a456-426614174000",
      captured_at: "2026-11-21T12:00:00Z",
      source: { kind: "scenario", scenario: "demo", route: "/", build_id: "build-1" },
      session: { id: "session-1" },
      viewport: { width: 1280, height: 720 },
      lenses,
      artifacts: [],
      privacy: { redaction_status: "complete" },
    };
    vi.stubGlobal("crypto", webcrypto);
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);
    const manifestFile = new File([JSON.stringify(manifest)], "capture.json", {
      type: "application/json",
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.manifest), {
      target: { files: [manifestFile] },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.artifacts), {
      target: { files: [new File(["unused"], "unused.bin")] },
    });
    fireEvent.click(screen.getByRole("button", { name: DESIGN_PATH_COPY.createRevision }));

    expect(
      await screen.findByText(strings.workbench.designIndex.captureL1Required),
    ).toBeInTheDocument();
    expect(createWorkspace).not.toHaveBeenCalled();
  });

  it.each([
    [390, DisplayDevice.PHONE],
    [768, DisplayDevice.TABLET],
    [1280, DisplayDevice.DESKTOP],
  ] as const)("creates an editable EX9 revision for a %s pixel viewport", async (width, device) => {
    const pixelBytes = new Uint8Array([1, 2, 3]);
    const htmlText =
      "<!doctype html><html><head><title>Incident overview</title></head><body><main><h1>Incident overview</h1><p>Current status</p></main></body></html>";
    const htmlBytes = new TextEncoder().encode(htmlText);
    const pixels = new File([pixelBytes], "screen.png", { type: "image/png" });
    const html = new File([htmlText], "page.html", { type: "text/html" });
    const digest = async (input: Uint8Array) => {
      const bytes = new Uint8Array(await webcrypto.subtle.digest("SHA-256", input));
      return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
    };
    const manifest = {
      schema_version: "ux-capture.v1",
      id: "123e4567-e89b-12d3-a456-426614174000",
      captured_at: "2026-11-21T12:00:00Z",
      source: { kind: "scenario", scenario: "demo", route: "/incidents", build_id: "build-1" },
      session: { id: "session-1" },
      viewport: { width, height: 720 },
      lenses: Object.fromEntries(
        ["L0", "L1", "L2", "L3", "L4", "L5", "L6"].map((lens) => [lens, { status: "present" }]),
      ),
      artifacts: [
        {
          kind: "pixels",
          path: "artifacts/screen.png",
          media_type: "image/png",
          size_bytes: pixels.size,
          sha256: await digest(pixelBytes),
        },
        {
          kind: "frozen-page",
          path: "artifacts/page.html",
          media_type: "text/html",
          size_bytes: html.size,
          sha256: await digest(htmlBytes),
        },
      ],
      privacy: { redaction_status: "complete" },
    };
    const created = { id: "ws_capture_revision" } as Workspace;
    vi.mocked(createWorkspace).mockResolvedValue(created);
    vi.mocked(uploadWorkspaceAsset).mockResolvedValue({} as never);
    vi.stubGlobal("crypto", webcrypto);
    Object.defineProperty(File.prototype, "arrayBuffer", {
      configurable: true,
      value(this: File) {
        return new Promise<ArrayBuffer>((resolve, reject) => {
          const reader = new FileReader();
          reader.onload = () => resolve(reader.result as ArrayBuffer);
          reader.onerror = () =>
            reject(reader.error ?? new Error("The file could not be read as an ArrayBuffer."));
          reader.readAsArrayBuffer(this);
        });
      },
    });
    renderWithProviders(<DesignIndexPage />);
    await screen.findByText(WORKSPACE_NAME);
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.manifest), {
      target: {
        files: [new File([JSON.stringify(manifest)], "capture.json", { type: "application/json" })],
      },
    });
    fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.artifacts), {
      target: { files: [pixels, html] },
    });
    fireEvent.click(screen.getByRole("button", { name: DESIGN_PATH_COPY.createRevision }));

    await waitFor(() => expect(createWorkspace).toHaveBeenCalledTimes(1));
    const options = vi.mocked(createWorkspace).mock.calls[0]![0];
    expect(options.provenance).toBe(WorkspaceProvenance.IMPORTED);
    expect(options.tags).toContain("ex9-capture");
    const state = options.pages?.[0]?.displays[0]?.states[0];
    const stateArgs = JSON.parse(state?.argsJson ?? "{}") as {
      spec?: { children?: Array<{ props?: { text?: string } }> };
    };
    const spec = stateArgs.spec ?? {};
    expect(state?.status).toBeDefined();
    expect(state?.captureId).toBe(manifest.id);
    expect(options.pages?.[0]?.displays[0]?.width).toBe(width);
    expect(options.pages?.[0]?.displays[0]?.device).toBe(device);
    expect(spec).toEqual(
      expect.objectContaining({
        props: expect.objectContaining({
          sourceHtml: expect.stringContaining("Incident overview"),
        }),
      }),
    );
    expect(JSON.stringify(spec)).toContain("Incident overview");
    expect(JSON.stringify(spec)).toContain("Current status");
    expect(vi.mocked(uploadWorkspaceAsset)).toHaveBeenCalledWith(
      "ws_capture_revision",
      pixels,
      "captured",
    );
  });

  it.each([
    ["a non-object JSON value", "null", strings.workbench.designIndex.captureInvalidManifest],
    [
      "an unsupported schema",
      JSON.stringify({ ...validCaptureManifest(), schema_version: "v2" }),
      strings.workbench.designIndex.captureInvalidManifest,
    ],
    [
      "a malformed capture id",
      JSON.stringify({ ...validCaptureManifest(), id: "capture-1" }),
      strings.workbench.designIndex.captureInvalidManifest,
    ],
    [
      "a non-scenario source",
      JSON.stringify({
        ...validCaptureManifest(),
        source: { ...validCaptureManifest().source, kind: "browser" },
      }),
      strings.workbench.designIndex.captureScenarioRequired,
    ],
    [
      "an empty scenario",
      JSON.stringify({
        ...validCaptureManifest(),
        source: { ...validCaptureManifest().source, scenario: "" },
      }),
      strings.workbench.designIndex.captureScenarioRequired,
    ],
    [
      "a non-string revision",
      JSON.stringify({
        ...validCaptureManifest(),
        source: { ...validCaptureManifest().source, revision: 7 },
      }),
      strings.workbench.designIndex.captureScenarioRequired,
    ],
    [
      "an invalid timestamp",
      JSON.stringify({ ...validCaptureManifest(), captured_at: "yesterday" }),
      strings.workbench.designIndex.captureViewportRequired,
    ],
    [
      "a non-positive viewport",
      JSON.stringify({ ...validCaptureManifest(), viewport: { width: 0, height: 720 } }),
      strings.workbench.designIndex.captureViewportRequired,
    ],
    [
      "incomplete redaction",
      JSON.stringify({ ...validCaptureManifest(), privacy: { redaction_status: "partial" } }),
      strings.workbench.designIndex.captureRedactionRequired,
    ],
    [
      "a malformed privacy flag",
      JSON.stringify({
        ...validCaptureManifest(),
        privacy: { redaction_status: "complete", external: "yes" },
      }),
      strings.workbench.designIndex.captureInvalidManifest,
    ],
    [
      "a missing lens",
      JSON.stringify({
        ...validCaptureManifest(),
        lenses: { ...validCaptureManifest().lenses, L4: undefined },
      }),
      strings.workbench.designIndex.captureLensInvalid,
    ],
    [
      "an unsupported lens status",
      JSON.stringify({
        ...validCaptureManifest(),
        lenses: { ...validCaptureManifest().lenses, L2: { status: "unknown" } },
      }),
      strings.workbench.designIndex.captureLensInvalid,
    ],
    [
      "a partial lens without a reason",
      JSON.stringify({
        ...validCaptureManifest(),
        lenses: { ...validCaptureManifest().lenses, L3: { status: "partial" } },
      }),
      strings.workbench.designIndex.captureLensInvalid,
    ],
    [
      "an invalid artifact declaration",
      JSON.stringify({ ...validCaptureManifest(), artifacts: [null] }),
      strings.workbench.designIndex.captureArtifactInvalid,
    ],
    [
      "a path traversal artifact",
      JSON.stringify({
        ...validCaptureManifest(),
        artifacts: [
          {
            kind: "pixels",
            path: "../screen.png",
            media_type: "image/png",
            size_bytes: 1,
            sha256: "a".repeat(64),
          },
        ],
      }),
      strings.workbench.designIndex.captureArtifactInvalid,
    ],
    [
      "an empty artifact list",
      JSON.stringify({ ...validCaptureManifest(), artifacts: [] }),
      strings.workbench.designIndex.captureArtifactsRequired,
    ],
    [
      "optional privacy and source metadata",
      JSON.stringify({
        ...validCaptureManifest(),
        source: { ...validCaptureManifest().source, revision: "r1" },
        privacy: { redaction_status: "complete", external: true, ephemeral: false },
      }),
      strings.workbench.designIndex.captureArtifactsRequired,
    ],
    [
      "a documented partial lens",
      JSON.stringify({
        ...validCaptureManifest(),
        lenses: {
          ...validCaptureManifest().lenses,
          L2: { status: "partial", reason: "Not available in this capture." },
        },
      }),
      strings.workbench.designIndex.captureArtifactsRequired,
    ],
  ] as const)(
    "rejects %s without mutating workspace data",
    async (_label, manifestText, expectedError) => {
      const manifestFile = new File([manifestText], "capture.json", { type: "application/json" });
      const artifactFile = new File(["unused"], "unused.bin");
      renderWithProviders(<DesignIndexPage />);
      await screen.findByText(WORKSPACE_NAME);
      fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.manifest), {
        target: { files: [manifestFile] },
      });
      fireEvent.change(screen.getByLabelText(DESIGN_PATH_COPY.artifacts), {
        target: { files: [artifactFile] },
      });
      fireEvent.click(screen.getByRole("button", { name: DESIGN_PATH_COPY.createRevision }));

      expect(await screen.findByText(expectedError)).toBeInTheDocument();
      expect(createWorkspace).not.toHaveBeenCalled();
      expect(uploadWorkspaceAsset).not.toHaveBeenCalled();
    },
  );

  it("shows a loading state while the workspace list is pending", () => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(
      queryResult<WorkspaceSummary[]>(undefined, { isLoading: true }),
    );
    renderWithProviders(<DesignIndexPage />);
    expect(screen.getByTestId("design-index-loading")).toBeInTheDocument();
  });

  it("shows an honest empty state when no authored workspace exists", async () => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(queryResult([]));
    renderWithProviders(<DesignIndexPage />);
    expect(await screen.findByText(strings.workbench.designIndex.emptyTitle)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.designIndex.noTemplates)).toBeInTheDocument();
  });

  it("labels imported and forked workspace provenance in the saved shelf", async () => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(
      queryResult([
        {
          ...workspaces[0]!,
          id: "ws_imported",
          name: "Imported capture",
          provenance: WorkspaceProvenance.IMPORTED,
          pageCount: 1,
          displayCount: 1,
          stateCount: 1,
        },
        {
          ...workspaces[0]!,
          id: "ws_forked",
          name: "Forked design",
          provenance: WorkspaceProvenance.FORKED,
        },
      ]),
    );
    renderWithProviders(<DesignIndexPage />);
    expect(await screen.findAllByTestId("design-workspace-row")).toHaveLength(2);
    expect(screen.getByText(strings.workbench.designIndex.provenanceImported)).toBeInTheDocument();
    expect(screen.getByText(strings.workbench.designIndex.provenanceForked)).toBeInTheDocument();
  });

  it("reports an unavailable workspace service instead of an empty list", async () => {
    vi.mocked(workspaceHook.useWorkspaces).mockReturnValue(
      queryResult<WorkspaceSummary[]>(undefined, { isError: true }),
    );
    renderWithProviders(<DesignIndexPage />);
    expect(await screen.findByText(strings.workbench.designIndex.errorTitle)).toBeInTheDocument();
  });
});
