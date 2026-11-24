import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { FixtureCanvas, type FixtureCanvasProps } from "./FixtureCanvas";

vi.mock("./FixtureFrame", () => ({
  FixtureFrame: ({ storyId, version, sourceRevision, override, reloadKey, onStatus }: {
    storyId: string;
    version: string;
    sourceRevision?: string;
    override?: Record<string, unknown> | null;
    reloadKey: number;
    onStatus: (status: "loading" | "ready" | "error" | "applied", message?: string) => void;
  }) => (
    <div data-testid={`fixture-${storyId}`} data-version={version} data-revision={sourceRevision} data-override={JSON.stringify(override)} data-reload={reloadKey}>
      <button type="button" onClick={() => onStatus("applied")}>Mark applied</button>
    </div>
  ),
}));

const asset = { id: "asset-1" } as CatalogAsset;
const stories: WorkbenchStory[] = [
  { key: "asset@1:primary", id: "primary", name: "Primary", role: "main", args: {}, fieldsJson: "{}", version: "1.0.0", sourcePath: "Asset.tsx", contractId: "asset@1" },
  { key: "asset@1:compact", id: "compact", name: "Compact", role: "alternate", args: {}, fieldsJson: "{}", version: "1.0.0", sourcePath: "Asset.tsx", contractId: "asset@1" },
];

function renderCanvas(overrides: Partial<FixtureCanvasProps> = {}) {
  const props: FixtureCanvasProps = {
    asset,
    stories,
    version: "1.0.0",
    sourceRevision: "rev-a",
    selectedId: "primary",
    onSelect: vi.fn(),
    positions: {},
    onMove: vi.fn(),
    overrides: { primary: { label: "Save" } },
    reloads: { primary: 2 },
    statuses: { primary: "ready", compact: "applied" },
    onStatus: vi.fn(),
    ...overrides,
  };
  renderWithProviders(<FixtureCanvas {...props} />);
  return props;
}

describe("FixtureCanvas", () => {
  it("renders fixture state, applies supplied positions, and forwards child status", () => {
    const onStatus = vi.fn();
    renderCanvas({
      positions: { primary: { x: 64, y: 92 } },
      statuses: { primary: "ready", compact: "applied" },
      onStatus,
    });

    expect(document.querySelector(".wb-fixture-canvas")).toHaveAttribute("data-fixtures-ready", "true");
    const primary = screen.getByTestId("fixture-primary");
    expect(primary).toHaveAttribute("data-version", "1.0.0");
    expect(primary).toHaveAttribute("data-revision", "rev-a");
    expect(primary).toHaveAttribute("data-override", '{"label":"Save"}');
    expect(primary).toHaveAttribute("data-reload", "2");
    expect(document.querySelector(".wb-fixture")).toHaveStyle({ left: "64px", top: "92px" });

    fireEvent.click(screen.getAllByRole("button", { name: "Mark applied" })[0]!);
    expect(onStatus).toHaveBeenCalledWith("primary", "applied", undefined);
  });

  it("selects fixture handles by keyboard and moves them in bounded steps", () => {
    const onSelect = vi.fn();
    const onMove = vi.fn();
    renderCanvas({ onSelect, onMove });
    const handle = screen.getAllByRole("button", { name: strings.workbench.component.fixtureHandle })[0]!;

    fireEvent.keyDown(handle, { key: "Enter" });
    fireEvent.keyDown(handle, { key: "ArrowLeft" });
    fireEvent.keyDown(handle, { key: "ArrowDown", shiftKey: true });

    expect(onSelect).toHaveBeenCalledWith(stories[0]);
    expect(onMove).toHaveBeenNthCalledWith(1, "primary", { x: 20, y: 28 });
    expect(onMove).toHaveBeenNthCalledWith(2, "primary", { x: 28, y: 52 });
  });

  it("drags from the fixture header after the movement threshold and clamps the canvas edge", () => {
    const onSelect = vi.fn();
    const onMove = vi.fn();
    renderCanvas({ positions: { primary: { x: 10, y: 20 } }, onSelect, onMove });
    const handle = screen.getAllByRole("button", { name: strings.workbench.component.fixtureHandle })[0]!;

    Object.defineProperties(handle, {
      setPointerCapture: { value: vi.fn() },
      hasPointerCapture: { value: () => false },
      releasePointerCapture: { value: vi.fn() },
    });
    fireEvent.pointerDown(handle, { button: 0, pointerId: 3, clientX: 100, clientY: 100 });
    fireEvent.pointerMove(handle, { pointerId: 3, clientX: 102, clientY: 102 });
    expect(onMove).not.toHaveBeenCalled();
    fireEvent.pointerMove(handle, { pointerId: 3, clientX: 70, clientY: 50 });
    expect(onMove).toHaveBeenCalledWith("primary", { x: 8, y: 8 });
    fireEvent.pointerUp(handle, { pointerId: 3 });
    expect(onSelect).toHaveBeenCalledWith(stories[0]);
  });

  it("explains the empty state when no story contracts exist", () => {
    renderCanvas({ stories: [] });
    expect(screen.getByText(strings.workbench.component.noStories)).toBeInTheDocument();
    expect(document.querySelector(".wb-fixture-canvas")).not.toBeInTheDocument();
  });
});
