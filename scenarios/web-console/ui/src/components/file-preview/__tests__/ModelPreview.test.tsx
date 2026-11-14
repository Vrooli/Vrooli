import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewProps, previewModel } from "../../../test-utils/filePreview";
import { ModelPreview } from "../renderers/ModelPreview";
import { useModelPreview } from "../useModelPreview";
import { strings } from "../../../consts/strings";
vi.mock("../useModelPreview");
const stage = { options: vi.fn(), view: vi.fn(), zoom: vi.fn(), orbit: vi.fn(), dispose: vi.fn() };
const state = () => ({ stats: { meshes: 5, triangles: 4768, dimensions: [2, 2, 2] as [number, number, number] }, error: null, partial: false, stage: { current: stage } });
const props = () => previewProps({ model: previewModel({ kind: "model3d", basename: "model.glb", resolvedPath: "/model.glb" }) });
beforeEach(() => { vi.clearAllMocks(); vi.mocked(useModelPreview).mockReturnValue(state()); });
describe("3D artifact inspector", () => {
  it("keeps the stage free of an extra content card and provides accessible touch/keyboard tools", () => {
    render(<ModelPreview {...props()} />); const c = strings.modelPreview;
    expect(screen.getByRole("img")).toHaveAttribute("tabindex", "0"); expect(screen.getByTestId("model-preview-stage").className).not.toMatch(/rounded|border/);
    fireEvent.click(screen.getByRole("button", { name: c.zoomIn })); expect(stage.zoom).toHaveBeenCalledWith(0.8);
    fireEvent.change(screen.getByRole("combobox", { name: c.view }), { target: { value: "top" } }); expect(stage.view).toHaveBeenCalledWith("top");
    fireEvent.keyDown(screen.getByRole("img"), { key: "ArrowRight" }); expect(stage.orbit).toHaveBeenCalledWith(0.15, 0);
    fireEvent.keyDown(screen.getByRole("img"), { key: "Home" }); expect(stage.view).toHaveBeenLastCalledWith("perspective");
  });
  it("toggles inspection modes and resets settings for a different file", () => {
    const c = strings.modelPreview, { rerender } = render(<ModelPreview {...props()} />);
    fireEvent.click(screen.getByRole("button", { name: c.wireframe })); expect(stage.options).toHaveBeenLastCalledWith(expect.objectContaining({ wireframe: true }));
    fireEvent.click(screen.getByRole("button", { name: c.rotate })); expect(stage.options).toHaveBeenLastCalledWith(expect.objectContaining({ rotate: true }));
    rerender(<ModelPreview {...props()} model={{ ...props().model, previewId: "next" }} />); expect(stage.options).toHaveBeenLastCalledWith(expect.objectContaining({ wireframe: false, rotate: false }));
  });
  it("applies a preset selected while the model is still loading", () => {
    vi.mocked(useModelPreview).mockReturnValue({ ...state(), stats: null, stage: { current: null } });
    const c = strings.modelPreview, { rerender } = render(<ModelPreview {...props()} />);
    fireEvent.click(screen.getByRole("button", { name: c.options })); fireEvent.click(screen.getByRole("radio", { name: c.top }));
    vi.mocked(useModelPreview).mockReturnValue(state()); rerender(<ModelPreview {...props()} />); expect(stage.view).toHaveBeenLastCalledWith("top");
  });
  it("preserves download and source renewal when loading fails", () => {
    vi.mocked(useModelPreview).mockReturnValue({ ...state(), stats: null, error: "fetch" }); const onReopen = vi.fn(); render(<ModelPreview {...props()} onReopen={onReopen} />);
    expect(screen.getByText(strings.modelPreview.fetch)).toBeInTheDocument(); fireEvent.click(screen.getByRole("button", { name: strings.messagesFileViewer.reopen })); expect(onReopen).toHaveBeenCalledOnce(); expect(screen.getByTestId("file-preview-download")).toHaveAttribute("download", "model.glb");
    expect(screen.queryByTestId("model-preview-navigation")).not.toBeInTheDocument();
  });
});
