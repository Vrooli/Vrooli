import type { Dispatch, ReactNode, SetStateAction } from "react";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import { listAllWorkspaceAssets, type WorkspaceAsset, type Workspace } from "../../api/workspaces";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import type { DesignDocument, DesignFrame, DesignOptions } from "../design-types";
import { node } from "../templates/spec";
import { AssetImportError } from "../studio/assets";
import type { ReferenceUpload } from "../studio/assets";
import type { TemplatePalette } from "../templates/spec";
import { visibleLayerRows } from "../studio/model";
import type { LoadedWorkspaceStudio } from "./workspaceStudioPageTypes";
import type { WorkspaceStudioPageActionSet } from "./workspaceStudioPageActions";
import { WorkspaceStudioLeftPane } from "./WorkspaceStudioLeftPane";

vi.mock("../../api/workspaces", () => ({ listAllWorkspaceAssets: vi.fn() }));
interface LayerProbeProps {
  onSelect: (id: string) => void;
  onToggleCollapse: (id: string) => void;
  onRename: (name: string) => void;
  footer: ReactNode;
}
interface AssetsProbeProps {
  assets: WorkspaceAsset[];
  onImport: (file: File, fidelity: "reference" | "captured") => void;
  onRemove: (id: string) => void;
  onPlace: (asset: ReferenceUpload) => void;
  loadOther: () => Promise<WorkspaceAsset[]>;
}
interface SystemProbeProps {
  onTokenChange: (key: keyof TemplatePalette, value: string) => void;
  onSelectNode: (id: string) => void;
  onSelectFrame: (id: string) => void;
  onOptionsChange: (patch: Partial<DesignOptions>, label: string) => void;
}
interface PageFieldsProbeProps {
  onPageName: (value: string) => void;
  onStateName: (value: string) => void;
  onRemoveState: () => void;
  canRemoveState: boolean;
}

vi.mock("../studio/StudioLayers", () => ({
  StudioLayers: ({ onSelect, onToggleCollapse, onRename, footer }: LayerProbeProps) => (
    <section>
      <button onClick={() => onSelect("layer-1")}>Select layer</button>
      <button onClick={() => onToggleCollapse("group-1")}>Toggle group</button>
      <button onClick={() => onRename("Renamed page")}>Rename page</button>
      {footer}
    </section>
  ),
}));
vi.mock("../studio/StudioAssets", () => ({
  StudioAssets: ({ assets, onImport, onRemove, onPlace, loadOther }: AssetsProbeProps) => (
    <section>
      <output>{assets.length} references</output>
      <button
        onClick={() =>
          onImport(new File(["bad"], "notes.txt", { type: "text/plain" }), "reference")
        }
      >
        Import invalid image
      </button>
      <button onClick={() => onRemove("asset-1")}>Remove reference</button>
      <button
        onClick={() =>
          onPlace({
            id: "asset-1",
            name: "Moodboard",
            url: "/asset",
            bytes: 2,
            fidelity: "reference",
          })
        }
      >
        Place reference
      </button>
      <button
        onClick={() =>
          void loadOther().then((other) => {
            document.body.dataset.otherAssets = other.map((entry) => entry.id).join(",");
          })
        }
      >
        Load other references
      </button>
    </section>
  ),
}));
vi.mock("../studio/StudioSystem", () => ({
  StudioSystem: ({
    onTokenChange,
    onSelectNode,
    onSelectFrame,
    onOptionsChange,
  }: SystemProbeProps) => (
    <section>
      <button onClick={() => onTokenChange("bg", "#123456")}>Change background</button>
      <button onClick={() => onSelectNode("layer-1")}>Select system layer</button>
      <button onClick={() => onSelectFrame("desktop-frame")}>Select desktop</button>
      <button onClick={() => onOptionsChange({ radius: 12 }, "Change radius")}>
        Change radius
      </button>
    </section>
  ),
}));

vi.mock("./WorkspacePageFields", () => ({
  PageFields: ({
    onPageName,
    onStateName,
    onRemoveState,
    canRemoveState,
  }: PageFieldsProbeProps) => (
    <section>
      <button onClick={() => onPageName("Updated page")}>Update page name</button>
      <button onClick={() => onStateName("Empty state")}>Rename state</button>
      <button disabled={!canRemoveState} onClick={onRemoveState}>
        Remove state
      </button>
    </section>
  ),
}));

const frame: DesignFrame = {
  id: "phone-frame",
  name: "Phone",
  device: "phone",
  spec: node("page", "page", {}, [node("layer-1", "text", { text: "Hello" })]),
};
const design: DesignDocument = {
  id: "design-1",
  name: "Design",
  updatedAt: "2025-01-01T00:00:00Z",
  revision: 1,
  frames: [frame],
  theme: {
    bg: "#fff",
    surface: "#eee",
    surfaceAlt: "#ddd",
    ink: "#111",
    muted: "#666",
    line: "#ccc",
    accent: "#06c",
    accentInk: "#fff",
    accentSoft: "#def",
    secondary: "#456",
  },
};
const asset = {
  id: "asset-1",
  workspaceId: "active",
  name: "Moodboard",
  mime: "image/png",
  bytes: 2,
  width: 1,
  height: 1,
  fidelity: "reference" as const,
  addedAt: "today",
  url: "/asset",
};

function renderPane(tab: "layers" | "system" | "assets" = "layers") {
  const importAsset = vi.fn<LoadedWorkspaceStudio["importAsset"]>();
  const removeAsset = vi.fn<LoadedWorkspaceStudio["removeAsset"]>();
  const studio = {
    assets: [asset],
    assetUrl: () => asset.url,
    page: { id: "page-1", name: "Home", route: "/", pageTypeId: "landing" },
    state: { name: "Default" },
    display: { states: [{}, {}] },
    renamePage: vi.fn(),
    updatePage: vi.fn(),
    renameState: vi.fn(),
    removeState: vi.fn(),
    importAsset,
    removeAsset,
    updateTheme: vi.fn(),
    updateOptions: vi.fn(),
  } as unknown as LoadedWorkspaceStudio;
  const actions = {
    selectNode: vi.fn(),
    moveLayer: vi.fn(),
    reparentLayer: vi.fn(),
    duplicateLayer: vi.fn(),
    removeLayer: vi.fn(),
    addLayer: vi.fn(),
    placeAsset: vi.fn(),
    selectFrame: vi.fn(),
  } as unknown as WorkspaceStudioPageActionSet;
  const props = {
    studio,
    workspace: { id: "active" } as Workspace,
    design,
    frames: [frame],
    frame,
    rows: visibleLayerRows(frame.spec, new Set()),
    selected: frame.spec.children?.[0],
    coverage: { rcl: 0, custom: 1, total: 1, ratio: 0 },
    leftTab: tab,
    onLeftTab: vi.fn(),
    leftSheetRef: { current: null },
    inertWhenReadOnly: {},
    onCloseSheet: vi.fn(),
    setCollapsed: vi.fn<Dispatch<SetStateAction<Set<string>>>>(),
    actions,
    flash: vi.fn(),
    token: "bg" as const,
    onSelectToken: vi.fn(),
  };
  return {
    ...renderWithProviders(<WorkspaceStudioLeftPane {...props} />),
    props,
    importAsset,
    removeAsset,
  };
}

describe("WorkspaceStudioLeftPane", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("connects the layer tab to selection, collapse, page fields and close actions", () => {
    const { props } = renderPane();

    fireEvent.click(screen.getByRole("button", { name: "Select layer" }));
    expect(props.actions.selectNode).toHaveBeenCalledWith(frame.id, "layer-1");
    fireEvent.click(screen.getByRole("button", { name: "Toggle group" }));
    expect(props.setCollapsed).toHaveBeenCalledOnce();
    const updateCollapsed = props.setCollapsed.mock.calls[0]?.[0];
    expect(typeof updateCollapsed).toBe("function");
    if (typeof updateCollapsed !== "function") {
      throw new Error("collapse updater was not provided");
    }
    expect(updateCollapsed(new Set())).toEqual(new Set(["group-1"]));
    expect(updateCollapsed(new Set(["group-1"]))).toEqual(new Set());

    fireEvent.click(screen.getByRole("button", { name: "Rename page" }));
    expect(props.studio.renamePage).toHaveBeenCalledWith(
      "Renamed page",
      "workbench.workspace.renamePage",
    );
    fireEvent.click(screen.getByRole("button", { name: "Update page name" }));
    expect(props.studio.updatePage).toHaveBeenCalledWith(
      { name: "Updated page" },
      "workbench.workspace.renamePage",
      "page:page-1:name",
    );
    fireEvent.click(screen.getByRole("button", { name: "Rename state" }));
    expect(props.studio.renameState).toHaveBeenCalledWith("Empty state");
    fireEvent.click(screen.getByRole("button", { name: "Remove state" }));
    expect(props.studio.removeState).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: "workbench.studio.closePanel" }));
    expect(props.onCloseSheet).toHaveBeenCalledOnce();
  });

  it("shows system theme and option controls for the system tab", () => {
    const { props } = renderPane("system");
    fireEvent.click(screen.getByRole("button", { name: "Change background" }));
    expect(props.studio.updateTheme).toHaveBeenCalledWith(
      { bg: "#123456" },
      "workbench.studio.editTokenAction",
      "theme:bg",
    );
    fireEvent.click(screen.getByRole("button", { name: "Select system layer" }));
    expect(props.actions.selectNode).toHaveBeenCalledWith(frame.id, "layer-1");
    fireEvent.click(screen.getByRole("button", { name: "Select desktop" }));
    expect(props.actions.selectFrame).toHaveBeenCalledWith("desktop-frame");
    fireEvent.click(screen.getByRole("button", { name: "Change radius" }));
    expect(props.studio.updateOptions).toHaveBeenCalledWith({ radius: 12 }, "Change radius");
  });

  it("filters other workspace assets and reports import and removal failures", async () => {
    vi.mocked(listAllWorkspaceAssets).mockResolvedValue([
      asset,
      { ...asset, id: "other-asset", workspaceId: "other", name: "Other workspace" },
    ]);
    const { props, importAsset, removeAsset } = renderPane("assets");
    fireEvent.click(screen.getByRole("button", { name: "Load other references" }));
    await waitFor(() => expect(document.body.dataset.otherAssets).toBe("other-asset"));

    importAsset.mockRejectedValue(new AssetImportError("type"));
    fireEvent.click(screen.getByRole("button", { name: "Import invalid image" }));
    await waitFor(() => expect(props.flash).toHaveBeenCalledWith("workbench.studio.assetType"));

    removeAsset.mockRejectedValue(new Error("failed"));
    fireEvent.click(screen.getByRole("button", { name: "Remove reference" }));
    await waitFor(() =>
      expect(props.flash).toHaveBeenCalledWith("workbench.studio.assetRemoveFailed"),
    );
  });
});
