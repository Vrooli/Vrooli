import { useState } from "react";
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";

import { renderWithProviders } from "../../test-utils/renderWithProviders";
import type { DesignDocument, DesignFrame } from "../design-types";
import { node, type TemplateNode } from "../templates/spec";
import { loadComponentBundle, useCatalog } from "../data";
import { WorkspaceStudioInspectorPane as InspectorPane } from "./WorkspaceStudioInspectorPane";

vi.mock("../data", () => ({
  loadComponentBundle: vi.fn(),
  useCatalog: vi.fn(),
  wbKeys: { component: (id: string) => ["component", id] },
}));
vi.mock("../studio/StudioInspector", () => ({
  StudioInspector: (props: {
    selected: TemplateNode | null;
    catalogCandidates: Array<{ id: string; displayName: string }>;
    pendingCatalogAssetId: string;
    onPropChange: (key: string, value: unknown) => void;
    onReplaceNode: (type: "button") => void;
    onReplaceCatalogAsset: (id: string) => void;
  }) => (
    <section>
      <output data-testid="selected-layer">{props.selected?.id ?? "none"}</output>
      <output data-testid="pending-catalog">{props.pendingCatalogAssetId}</output>
      <button onClick={() => props.onPropChange("text", "Updated copy")}>Update text</button>
      <button onClick={() => props.onReplaceNode("button")}>Replace with button</button>
      {props.catalogCandidates.map((candidate) => (
        <button key={candidate.id} onClick={() => props.onReplaceCatalogAsset(candidate.id)}>
          Replace with {candidate.displayName}
        </button>
      ))}
    </section>
  ),
}));

const selectedNode = node("headline", "text", { text: "Hello" });
const replacementNode = node("other", "text", { text: "Other" });
const frame: DesignFrame = {
  id: "phone-frame",
  name: "Phone",
  device: "phone",
  spec: node("page", "page", {}, [selectedNode, replacementNode]),
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
const component = { id: "published-button", displayName: "Published Button" };

type Props = Parameters<typeof InspectorPane>[0];
function renderPane(overrides: Partial<Props> = {}, withSelectionControl = false) {
  const updateFrame = vi.fn();
  const applyProposal = vi.fn();
  const commitFrame = vi.fn<Props["commitFrame"]>();
  const props = {
    studio: { updateFrame, applyProposal, stateByDisplay: {} },
    workspace: { id: "workspace-1" },
    page: { id: "page-1", displays: [] },
    design,
    frame,
    frames: [frame],
    selected: selectedNode,
    commitFrame,
    coverage: { rcl: 0, custom: 2, total: 2, ratio: 0 },
    claims: [],
    audits: {},
    stateGap: { total: 0, covered: 0, missing: [] },
    rightTab: "props",
    onRightTab: vi.fn(),
    aiScope: { kind: "layer", layerId: selectedNode.id },
    onAiScope: vi.fn(),
    selectFrame: vi.fn(),
    flash: vi.fn(),
    rightSheetRef: { current: null },
    inertWhenReadOnly: {},
    onCloseSheet: vi.fn(),
    ...overrides,
  } as unknown as Props;
  function ControlledPane() {
    const [selected, setSelected] = useState(selectedNode);
    return (
      <>
        <button onClick={() => setSelected(replacementNode)}>Change selected layer</button>
        <InspectorPane {...props} selected={selected} />
      </>
    );
  }
  const rendered = withSelectionControl
    ? renderWithProviders(<ControlledPane />)
    : renderWithProviders(<InspectorPane {...props} />);
  return {
    ...rendered,
    props,
    updateFrame,
    commitFrame,
  };
}

describe("WorkspaceStudioInspectorPane", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useCatalog).mockReturnValue({
      data: [component],
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useCatalog>);
  });

  it("routes property and primitive replacement edits through one frame commit", () => {
    const { commitFrame } = renderPane();

    expect(screen.getByTestId("selected-layer")).toHaveTextContent("headline");
    fireEvent.click(screen.getByRole("button", { name: "Update text" }));
    expect(commitFrame).toHaveBeenCalledWith(
      expect.any(Function),
      "workbench.studio.editProp",
      "prop:phone-frame:headline:text",
    );
    const updateProp = commitFrame.mock.calls[0]?.[0];
    expect(updateProp?.(frame.spec).children?.[0]?.props.text).toBe("Updated copy");

    fireEvent.click(screen.getByRole("button", { name: "Replace with button" }));
    expect(commitFrame).toHaveBeenLastCalledWith(
      expect.any(Function),
      "workbench.studio.replaceLayerAction",
      "replace:phone-frame:headline",
    );
  });

  it("loads the published story before replacing the selected layer", async () => {
    vi.mocked(loadComponentBundle).mockResolvedValue({
      stories: [{ id: "default-story" }],
    } as never);
    const { updateFrame } = renderPane();

    fireEvent.click(screen.getByRole("button", { name: "Replace with Published Button" }));

    await waitFor(() => expect(updateFrame).toHaveBeenCalledOnce());
    expect(loadComponentBundle).toHaveBeenCalledWith(component.id);
    expect(updateFrame).toHaveBeenCalledWith(
      frame.id,
      expect.any(Function),
      "workbench.studio.replaceLayerWithComponent",
      "replace:phone-frame:headline:published-button",
    );
    const update = updateFrame.mock.calls[0]?.[1];
    expect(update?.(frame.spec).children?.[0]?.props.componentId).toBe(component.id);
  });

  it("discards a deferred catalog response after the mounted pane changes selection", async () => {
    const pendingComponent = { id: "pending-component", displayName: "Pending Button" };
    vi.mocked(useCatalog).mockReturnValue({
      data: [pendingComponent],
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useCatalog>);
    let resolveBundle: ((bundle: { stories: Array<{ id: string }> }) => void) | undefined;
    vi.mocked(loadComponentBundle).mockReturnValue(
      new Promise((resolve) => {
        resolveBundle = resolve;
      }) as never,
    );
    const { props, updateFrame } = renderPane({}, true);

    fireEvent.click(screen.getByRole("button", { name: "Replace with Pending Button" }));
    expect(loadComponentBundle).toHaveBeenCalledWith(pendingComponent.id);
    expect(screen.getByTestId("pending-catalog")).toHaveTextContent(pendingComponent.id);

    fireEvent.click(screen.getByRole("button", { name: "Change selected layer" }));
    expect(screen.getByTestId("selected-layer")).toHaveTextContent(replacementNode.id);
    act(() => resolveBundle?.({ stories: [{ id: "default-story" }] }));

    await waitFor(() =>
      expect(props.flash).toHaveBeenCalledWith(
        "workbench.studio.catalogReplacementSelectionChanged",
      ),
    );
    expect(updateFrame).not.toHaveBeenCalled();
    expect(screen.getByTestId("pending-catalog")).toBeEmptyDOMElement();
  });

  it("reports missing stories and failed catalog requests without applying a replacement", async () => {
    vi.mocked(loadComponentBundle).mockResolvedValueOnce({ stories: [] } as never);
    const { props, updateFrame } = renderPane();
    fireEvent.click(screen.getByRole("button", { name: "Replace with Published Button" }));
    await waitFor(() =>
      expect(props.flash).toHaveBeenCalledWith("workbench.studio.catalogReplacementNoStory"),
    );
    expect(updateFrame).not.toHaveBeenCalled();

    vi.mocked(loadComponentBundle).mockRejectedValueOnce(new Error("unavailable"));
    fireEvent.click(screen.getByRole("button", { name: "Replace with Published Button" }));
    await waitFor(() =>
      expect(props.flash).toHaveBeenCalledWith("workbench.studio.catalogReplacementError"),
    );
    expect(updateFrame).not.toHaveBeenCalled();
  });
});
