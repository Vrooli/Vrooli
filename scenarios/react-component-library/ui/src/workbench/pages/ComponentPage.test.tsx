import { fireEvent, screen, waitFor } from "@testing-library/react";
import { Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { runComponentTest } from "../../api/componentTests";
import { getWorkspace, WorkspaceProvenance } from "../../api/workspaces";
import { renderWithProviders } from "../../test-utils";
import type { ComponentBundle, WorkbenchStory } from "../data";
import { useComponentBundle } from "../data";
import { createComponentWorkspace } from "../workspaces/componentContext";
import { ComponentPage } from "./ComponentPage";

type HeaderProps = {
  mode: "fixtures" | "context";
  stories: WorkbenchStory[];
  selectedStory?: WorkbenchStory;
  canOpenWorkspace: boolean;
  copied: boolean;
  openingWorkspace: boolean;
  runningEvidence: boolean;
  onShare: () => void;
  onOpenWorkspace: () => void;
  onRunEvidence: () => void;
  onOpenDrawer: (
    tab: "overview" | "source" | "versions" | "experience" | "tests" | "relationships",
  ) => void;
  onChooseMode: (mode: "fixtures" | "context") => void;
  onChooseStory: (story: WorkbenchStory) => void;
  onRerunStory: () => void;
  onToggleDrawer: () => void;
};

vi.mock("../data", () => ({
  assetVersion: (asset: { latestVersion?: string; version?: string }) =>
    asset.latestVersion || asset.version || "",
  useComponentBundle: vi.fn(),
  wbKeys: { component: (id: string) => ["component", id] },
}));
vi.mock("../../api/componentTests", () => ({ runComponentTest: vi.fn() }));
vi.mock("../../api/workspaces", () => ({
  getWorkspace: vi.fn(),
  WorkspaceProvenance: { CURATED: "CURATED" },
}));
vi.mock("../workspaces/useWorkspaces", () => ({
  useWorkspaces: () => ({
    data: [
      {
        id: "template-1",
        name: "Clean studio",
        summary: "A curated layout",
        provenance: "CURATED",
      },
    ],
  }),
}));
vi.mock("../workspaces/componentContext", () => ({
  DEFAULT_COMPONENT_WORKSPACE_ID: "template-1",
  curatedWorkspaceIdForTemplate: (id: string) => id,
  workspaceTemplate: () => ({
    id: "template-spec",
    spec: {
      id: "page",
      type: "page",
      props: {},
      children: [
        { id: "action-slot", type: "slot", role: "action", accepts: ["button"], props: { minH: 40 } },
      ],
    },
  }),
  createComponentWorkspace: vi.fn(),
}));
vi.mock("../component/ComponentPageHeader", () => ({
  ComponentPageHeader: (props: HeaderProps) => (
    <div
      data-testid="page-header"
      data-mode={props.mode}
      data-story={props.selectedStory?.id}
      data-can-open={String(props.canOpenWorkspace)}
    >
      <button onClick={() => props.onChooseStory(props.stories[1]!)}>Select secondary story</button>
      <button onClick={() => props.onChooseMode("context")}>Open context mode</button>
      <button onClick={() => props.onChooseMode("fixtures")}>Open fixture mode</button>
      <button onClick={() => props.onOpenDrawer("versions")}>Open versions</button>
      <button onClick={() => props.onOpenDrawer("tests")}>Open tests</button>
      <button onClick={props.onToggleDrawer}>Toggle drawer</button>
      <button onClick={props.onRunEvidence} disabled={props.runningEvidence}>
        Run evidence
      </button>
      <button
        onClick={props.onOpenWorkspace}
        disabled={!props.canOpenWorkspace || props.openingWorkspace}
      >
        Use in design
      </button>
      <button onClick={props.onRerunStory}>Rerun story</button>
      <button onClick={props.onShare}>{props.copied ? "Copied" : "Share"}</button>
    </div>
  ),
}));
vi.mock("../component/FixtureBench", () => ({
  FixtureBench: (props: {
    selectedId: string | null;
    stories: WorkbenchStory[];
    reloads: Record<string, number>;
    onSelect: (story: WorkbenchStory) => void;
    onStatus: (id: string, status: "ready" | "loading" | "error" | "applied") => void;
  }) => (
    <div
      data-testid="fixture-bench"
      data-selected={props.selectedId}
      data-reloads={JSON.stringify(props.reloads)}
    >
      <button onClick={() => props.onSelect(props.stories[1]!)}>Bench selects story</button>
      <button onClick={() => props.onStatus("secondary", "ready")}>Ready fixture</button>
    </div>
  ),
}));
vi.mock("../component/InContextStage", () => ({
  InContextStage: (props: {
    device: string;
    onDeviceChange: (device: "desktop" | "tablet" | "phone") => void;
    onReport: (report: {
      rows: Array<{
        id: string;
        label: string;
        detail: string;
        verdict: string;
        evidence: string;
        scored: boolean;
      }>;
      passed: number;
      scored: number;
      decision: { chosen: undefined };
      device: "desktop" | "tablet" | "phone";
      viewport: number;
      status: "ready";
      instruments: Record<string, boolean>;
      storyId: string | null;
      hasStory: boolean;
    }) => void;
    onTemplateChange: (templateId: string) => void;
    story?: WorkbenchStory;
  }) => (
    <div data-testid="context-stage" data-device={props.device} data-story={props.story?.id}>
      <button onClick={() => props.onDeviceChange("phone")}>Use phone frame</button>
      <button onClick={() => props.onTemplateChange("another-template")}>Choose template</button>
      <button
        onClick={() =>
          props.onReport({
            rows: [
              {
                id: "measured",
                label: "Measured layout",
                detail: "within bounds",
                verdict: "pass",
                evidence: "measured",
                scored: true,
              },
            ],
            passed: 1,
            scored: 2,
            decision: { chosen: undefined },
            device: "phone",
            viewport: 390,
            status: "ready",
            instruments: {},
            storyId: props.story?.id ?? null,
            hasStory: true,
          })
        }
      >
        Report measured context
      </button>
    </div>
  ),
}));
vi.mock("../component/ComponentPageInspector", () => ({
  ComponentPageInspector: (props: {
    mode: string;
    selectedStory?: WorkbenchStory;
    fixtureRows: unknown[];
    contextRows: unknown[];
    canOpenWorkspace: boolean;
    statuses: Record<string, "loading" | "ready" | "error" | "applied">;
    onOpenWorkspace: () => void;
  }) => (
    <output
      data-testid="page-inspector"
      data-mode={props.mode}
      data-story={props.selectedStory?.id}
      data-fixture-rows={props.fixtureRows.length}
      data-context-rows={props.contextRows.length}
      data-can-open={String(props.canOpenWorkspace)}
      data-status-secondary={props.statuses.secondary ?? ""}
    >
      <button onClick={props.onOpenWorkspace}>Inspector opens workspace</button>
    </output>
  ),
}));
vi.mock("../component/AssetDataDrawer", () => ({
  AssetDataDrawer: (props: {
    tab: string;
    selectedTestReportID: string | null;
    onTab: (tab: "source" | "versions" | "tests") => void;
    onSelectTestReport: (id?: string) => void;
    onSourceSaved: (version: string, filePath: string, sha256: string) => void;
  }) => (
    <div
      data-testid="asset-drawer"
      data-tab={props.tab}
      data-report={props.selectedTestReportID ?? ""}
    >
      <button onClick={() => props.onSelectTestReport("report-42")}>Select saved report</button>
      <button onClick={() => props.onTab("source")}>Drawer source tab</button>
      <button onClick={() => props.onTab("versions")}>Drawer versions tab</button>
      <button onClick={() => props.onSourceSaved("1.2.0", "Button.tsx", "sha-1")}>
        Save source
      </button>
    </div>
  ),
}));

const stories: WorkbenchStory[] = [
  {
    key: "asset@1:primary",
    id: "primary",
    name: "Primary",
    role: "main",
    args: {},
    fieldsJson: "{}",
    version: "1.0.0",
    sourcePath: "Button.tsx",
    contractId: "asset@1",
  },
  {
    key: "asset@1:secondary",
    id: "secondary",
    name: "Secondary",
    role: "alternate",
    args: {},
    fieldsJson: "{}",
    version: "1.0.0",
    sourcePath: "Button.tsx",
    contractId: "asset@1",
  },
];
const asset = {
  id: "asset-7",
  libraryId: "react-component-library:Button",
  displayName: "Button",
  version: "1.0.0",
  latestVersion: "1.2.0",
};
const bundle: ComponentBundle = {
  asset: asset as ComponentBundle["asset"],
  versions: [],
  stories,
  experience: {
    componentId: "asset-7",
    libraryId: "react-component-library:Button",
    version: "1.2.0",
    contractId: "asset@1",
    title: "Button",
    purpose: "Action",
    evidenceStatus: "fresh",
    evidenceMessage: "",
    states: [
      { id: "default", exampleName: "Default", description: "Normal" },
      { id: "disabled", exampleName: "Disabled", description: "Disabled" },
    ],
    claims: [
      {
        id: "keyboard",
        type: "behavior",
        statement: "Keyboard support",
        tier: "core",
        states: ["default"],
      },
    ],
    evidence: [
      {
        claimId: "keyboard",
        verdict: "pass",
        stateId: "default",
        exampleName: "Default",
        captureRef: "capture-1",
        checkedAt: "2026-01-01T00:00:00Z",
        message: "Pass",
        viewport: "desktop",
        viewportWidth: 1280,
        viewportHeight: 800,
      },
    ],
  },
  testReports: [],
  partial: [],
};
const curatedWorkspace = {
  id: "template-1",
  name: "Clean studio",
  provenance: WorkspaceProvenance.CURATED,
};

function LocationProbe() {
  const location = useLocation();
  return <output data-testid="location">{`${location.pathname}${location.search}`}</output>;
}

function NextAssetProbe() {
  const navigate = useNavigate();
  return <button onClick={() => void navigate("/assets/asset-8")}>Go to next asset</button>;
}

function renderComponentPage(path = "/assets/asset-7") {
  return renderWithProviders(
    <>
      <LocationProbe />
      <NextAssetProbe />
      <Routes>
        <Route path="/assets/:id" element={<ComponentPage />} />
        <Route path="/design" element={<div data-testid="design-route">Design route</div>} />
      </Routes>
    </>,
    { routerEntries: [path] },
  );
}

describe("ComponentPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useComponentBundle).mockReturnValue({
      data: bundle,
      isLoading: false,
      isError: false,
      error: null,
      refetch: vi.fn(),
    } as never);
    vi.mocked(getWorkspace).mockResolvedValue(curatedWorkspace as never);
    vi.mocked(createComponentWorkspace).mockResolvedValue({ id: "workspace-created" } as never);
    vi.mocked(runComponentTest).mockResolvedValue({} as never);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("does not carry a release version or story state into the next asset", async () => {
    renderComponentPage("/assets/asset-7?story=primary");
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-story", "primary"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Go to next asset" }));

    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent("/assets/asset-8"),
    );
    const calls = vi.mocked(useComponentBundle).mock.calls;
    const firstForNextAsset = calls.find(([assetId]) => assetId === "asset-8");
    // The first query for the new asset must not name a release version. The
    // page may select that asset's own latest version after the bundle loads.
    expect(firstForNextAsset?.[1]).toBeUndefined();
  });

  it("loads a deep-linked story and keeps unrelated URL parameters while selecting a new mode and story", async () => {
    renderComponentPage("/assets/asset-7?story=primary&foo=keep");
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );
    expect(screen.getByTestId("page-header")).toHaveAttribute("data-story", "primary");
    expect(screen.getByTestId("fixture-bench")).toHaveAttribute("data-selected", "primary");
    fireEvent.click(screen.getByRole("button", { name: "Ready fixture" }));
    expect(screen.getByTestId("page-inspector")).toHaveAttribute("data-status-secondary", "ready");

    fireEvent.click(screen.getByRole("button", { name: "Select secondary story" }));
    expect(screen.getByTestId("page-header")).toHaveAttribute("data-story", "secondary");
    expect(screen.getByTestId("location")).toHaveTextContent("story=secondary");
    expect(screen.getByTestId("location")).toHaveTextContent("foo=keep");

    fireEvent.click(screen.getByRole("button", { name: "Open context mode" }));
    expect(screen.getByTestId("context-stage")).toHaveAttribute("data-story", "secondary");
    expect(screen.getByTestId("location")).toHaveTextContent("mode=context");
    expect(screen.getByTestId("page-inspector")).toHaveAttribute("data-mode", "context");
    fireEvent.click(screen.getByRole("button", { name: "Use phone frame" }));
    expect(screen.getByTestId("context-stage")).toHaveAttribute("data-device", "phone");
    expect(screen.getByTestId("location")).toHaveTextContent("device=phone");
    fireEvent.click(screen.getByRole("button", { name: "Report measured context" }));
    expect(screen.getByTestId("page-inspector")).toHaveAttribute("data-context-rows", "1");
  });

  it("runs evidence and opens the test report drawer, then reflects a selected report in the URL", async () => {
    renderComponentPage("/assets/asset-7?story=secondary");
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Run evidence" }));

    await waitFor(() =>
      expect(runComponentTest).toHaveBeenCalledWith({
        componentId: "asset-7",
        version: "1.2.0",
        includeClosure: false,
      }),
    );
    expect(await screen.findByTestId("asset-drawer")).toHaveAttribute("data-tab", "tests");
    fireEvent.click(screen.getByRole("button", { name: "Select saved report" }));
    expect(screen.getByTestId("asset-drawer")).toHaveAttribute("data-report", "report-42");
    expect(screen.getByTestId("location")).toHaveTextContent("tab=tests");
    expect(screen.getByTestId("location")).toHaveTextContent("testReport=report-42");
  });

  it("updates drawer URL state through page callbacks", async () => {
    renderComponentPage();
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Open versions" }));
    expect(screen.getByTestId("asset-drawer")).toHaveAttribute("data-tab", "versions");
    expect(screen.getByTestId("location")).toHaveTextContent("tab=versions");
    fireEvent.click(screen.getByRole("button", { name: "Toggle drawer" }));
    expect(screen.queryByTestId("asset-drawer")).not.toBeInTheDocument();
  });

  it("creates a workspace from the selected story and navigates to its studio", async () => {
    renderComponentPage("/assets/asset-7?story=secondary");
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Use in design" }));

    await waitFor(() =>
      expect(createComponentWorkspace).toHaveBeenCalledWith({
        templateWorkspace: curatedWorkspace,
        componentId: "asset-7",
        storyId: "secondary",
        componentVersion: "1.0.0",
        preferredSlotId: "action-slot",
        preferredSlotRole: "action",
      }),
    );
    expect(await screen.findByTestId("design-route")).toBeInTheDocument();
    expect(screen.getByTestId("location")).toHaveTextContent("/design?workspace=workspace-created");
  });

  it("updates context template and source revision, then opens from the inspector", async () => {
    renderComponentPage("/assets/asset-7?mode=context");
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Choose template" }));
    expect(screen.getByTestId("location")).toHaveTextContent("template=another-template");
    await waitFor(() => expect(getWorkspace).toHaveBeenCalledWith("another-template"));
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Open versions" }));
    fireEvent.click(screen.getByRole("button", { name: "Drawer source tab" }));
    expect(screen.getByTestId("location")).toHaveTextContent("tab=source");
    fireEvent.click(screen.getByRole("button", { name: "Save source" }));
    fireEvent.click(screen.getByRole("button", { name: "Save source" }));

    fireEvent.click(screen.getByRole("button", { name: "Inspector opens workspace" }));
    await waitFor(() => expect(createComponentWorkspace).toHaveBeenCalledOnce());
  });

  it("surfaces workspace creation failures without leaving the component page", async () => {
    const failureMessage = "template cannot be copied";
    vi.mocked(createComponentWorkspace).mockRejectedValue(new Error(failureMessage));
    renderComponentPage();
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Use in design" }));

    expect(await screen.findByRole("alert")).toHaveTextContent(failureMessage);
    expect(screen.getByTestId("component-page")).toBeInTheDocument();
  });

  it("copies a stable preview link and reruns the selected story on demand", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", Object.assign(Object.create(null), { clipboard: { writeText } }));
    renderComponentPage("/assets/asset-7?story=secondary");
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Share" }));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith(`${window.location.origin}/assets/asset-7/preview`),
    );
    expect(screen.getByRole("button", { name: "Copied" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Rerun story" }));
    expect(screen.getByTestId("fixture-bench")).toHaveAttribute("data-reloads", '{"secondary":1}');
  });

  it("reports bundle errors with a retry action", () => {
    const refetch = vi.fn();
    vi.mocked(useComponentBundle).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
      error: new Error("catalog offline"),
      refetch,
    } as never);
    renderComponentPage();

    expect(screen.getByRole("alert")).toHaveTextContent("catalog offline");
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(refetch).toHaveBeenCalledOnce();
  });

  it("shows loading feedback until the component bundle is ready", () => {
    vi.mocked(useComponentBundle).mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
      error: null,
      refetch: vi.fn(),
    } as never);
    renderComponentPage();

    expect(document.querySelector(".wb-page")).toBeInTheDocument();
    expect(screen.queryByTestId("page-header")).not.toBeInTheDocument();
  });

  it("explains when the curated context template is unavailable and preserves its deep link", async () => {
    vi.mocked(getWorkspace).mockResolvedValue(null);
    renderComponentPage("/assets/asset-7?mode=context&template=missing-template");

    await waitFor(() => expect(document.querySelector(".wb-empty-inline")).toBeInTheDocument());
    expect(screen.getByTestId("page-inspector")).toHaveAttribute("data-can-open", "false");
    expect(screen.getByTestId("location")).toHaveTextContent("template=missing-template");
  });

  it("shows evidence failures while keeping the page usable", async () => {
    vi.mocked(runComponentTest).mockRejectedValue(new Error("evidence service unavailable"));
    renderComponentPage();
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Run evidence" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("evidence service unavailable");
    expect(screen.getByTestId("component-page")).toBeInTheDocument();
  });

  it("keeps share available when the browser clipboard rejects the write", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("clipboard blocked"));
    vi.stubGlobal("navigator", Object.assign(Object.create(null), { clipboard: { writeText } }));
    renderComponentPage();
    await waitFor(() =>
      expect(screen.getByTestId("page-header")).toHaveAttribute("data-can-open", "true"),
    );

    fireEvent.click(screen.getByRole("button", { name: "Share" }));

    await waitFor(() => expect(writeText).toHaveBeenCalledOnce());
    expect(screen.getByRole("button", { name: "Share" })).toBeInTheDocument();
  });
});
