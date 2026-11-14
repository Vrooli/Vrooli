import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import { ImagePreview } from "../renderers/ImagePreview";

vi.mock("@vrooli/react-component-library/useResizeObserver", () => ({ useResizeObserver: () => ({ ref: () => {}, rect: { width: 400, height: 300 } }) }));
const c = strings.artifactPreview;
const props = previewProps({ model: previewModel({ basename: "drawing.svg", kind: "svg", mimeType: "image/svg+xml" }) });
function load() {
  const image = screen.getByRole("img", { name: "drawing.svg" });
  Object.defineProperties(image, { naturalWidth: { value: 1000 }, naturalHeight: { value: 800 } });
  fireEvent.load(image);
  return image;
}

describe("image inspector", () => {
  it("shows original dimensions, supports Fit/100%, rotation and transparency backgrounds", () => {
    render(<ImagePreview {...props} />);
    const image = load();
    expect(image).toHaveStyle({ transform: "translate(-50%, -50%) translate3d(0px, 0px, 0) scale(0.375) rotate(0deg)" });
    expect(screen.getByText(/1000 × 800/)).toBeInTheDocument();
    fireEvent.change(screen.getByRole("combobox", { name: c.imageZoom }), { target: { value: "1" } });
    expect(image.style.transform).toContain("scale(1)");
    fireEvent.click(screen.getByRole("button", { name: c.rotate }));
    expect(image.style.transform).toContain("rotate(90deg)");
    expect(image.style.transform).toContain("scale(0.3)");
    fireEvent.click(screen.getByRole("button", { name: c.background }));
    expect(screen.getByTestId("artifact-image-stage")).toHaveClass("bg-white");
    fireEvent.click(screen.getByRole("button", { name: c.background }));
    expect(screen.getByTestId("artifact-image-stage")).toHaveClass("bg-black");
  });
  it("keeps SVG as an inert image resource and exposes keyboard inspection", () => {
    render(<ImagePreview {...props} />);
    const image = load();
    expect(image).toHaveAttribute("src", "/preview/blob");
    const stage = screen.getByTestId("artifact-image-stage");
    fireEvent.keyDown(stage, { key: "+" });
    expect(image.style.transform).toContain("scale(0.5625)");
    fireEvent.keyDown(stage, { key: "0" });
    expect(screen.getByRole("combobox", { name: c.imageZoom })).toHaveValue("fit");
    expect(document.querySelector("iframe")).toBeNull();
    expect(document.querySelector("script")).toBeNull();
  });
  it("preserves file actions on failure and resets inspection for a different source", () => {
    const { rerender } = render(<ImagePreview {...props} />);
    const image = load();
    fireEvent.error(image);
    expect(screen.getByRole("alert")).toHaveTextContent(strings.messagesFileViewer.mediaLoadError);
    expect(screen.getByTestId("file-preview-download")).toHaveAttribute("href", "/preview/blob");
    rerender(<ImagePreview {...props} model={{ ...props.model, previewId: "second", blobHref: "/second" }} />);
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(screen.getByRole("img", { name: "drawing.svg" })).toHaveAttribute("src", "/second");
    expect(screen.getByRole("combobox", { name: c.imageZoom })).toHaveValue("fit");
  });
});
