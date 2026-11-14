import { createRef } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../../../test-utils";
import { previewModel, previewProps } from "../../../test-utils/filePreview";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { usePresentation } from "../usePresentation";
import { presentationFrame } from "../presentationFrame";
import type { PresentationView } from "../publicationData";
import { PresentationPreview } from "../renderers/PresentationPreview";
vi.mock("../usePresentation", () => ({ usePresentation: vi.fn() }));
const post = vi.fn<(message: object) => void>(), slides = [{ title: "Opening", text: "Original slide text", notes: "A speaker note", hidden: false }, { title: "Results", text: "Next text", notes: "", hidden: true }];
function props() { return previewProps({ model: previewModel({ kind: "presentation", basename: "review.pptx" }) }); }
function state(): ReturnType<typeof usePresentation> { return { frame: createRef<HTMLIFrameElement>(), source: presentationFrame("// owned renderer"), metadata: { slides, width: 960, height: 540, warnings: false }, error: null, warning: false, post }; }
function lastView() { const command = post.mock.lastCall?.[0]; if (!command) throw Error("Missing view command"); return (command as { view: PresentationView }).view; }
beforeEach(() => { vi.clearAllMocks(); vi.mocked(usePresentation).mockReturnValue(state()); });
describe("PowerPoint inspection controls", () => {
  it("isolates deck content and keeps host controls outside the opaque-origin frame", async () => {
    render(<PresentationPreview {...props()} />);
    const iframe = screen.getByTestId(selectors.presentationPreview.frame);
    expect(iframe).toHaveAttribute("sandbox", "allow-scripts"); expect(iframe).toHaveAttribute("referrerpolicy", "no-referrer"); expect(iframe.getAttribute("srcdoc")).toContain("connect-src 'none'"); expect(iframe.getAttribute("srcdoc")).toContain("media-src 'none'"); expect(iframe.getAttribute("srcdoc")).not.toContain("allow-same-origin");
    // Opaque frames cannot send the parent's access cookie to an asset server.
    const document = new DOMParser().parseFromString(iframe.getAttribute("srcdoc") ?? "", "text/html");
    expect(document.querySelector("script[src]")).toBeNull();
    await waitFor(() => { expect(lastView()).toMatchObject({ index: 0, zoom: 1 }); });
    expect(screen.getByRole("button", { name: strings.presentationPreview.previous })).toBeDisabled();
  });
  it("navigates slides, shows empty notes and resets settings for a new file", async () => {
    const c = strings.presentationPreview, { rerender } = render(<PresentationPreview {...props()} />);
    fireEvent.click(screen.getByRole("button", { name: c.notes })); expect(screen.getByText("A speaker note")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: c.next })); expect(screen.getByText(c.noNotes)).toBeInTheDocument(); expect(screen.getByRole("button", { name: c.next })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: c.notes })); fireEvent.click(screen.getByRole("button", { name: c.zoomIn })); await waitFor(() => { expect(lastView()).toMatchObject({ index: 1, zoom: 1.25 }); });
    rerender(<PresentationPreview {...props()} model={{ ...props().model, previewId: "another" }} />); await waitFor(() => { expect(lastView()).toMatchObject({ index: 0, zoom: 1 }); });
  });
  it("provides source-aware recovery and original download on failure", () => {
    vi.mocked(usePresentation).mockReturnValue({ ...state(), error: "fetch" });
    const onReopen = vi.fn(); render(<PresentationPreview {...props()} onReopen={onReopen} />); expect(screen.getByText(strings.presentationPreview.fetch)).toBeInTheDocument(); fireEvent.click(screen.getByRole("button", { name: strings.messagesFileViewer.reopen })); expect(onReopen).toHaveBeenCalledOnce(); expect(screen.getByTestId("file-preview-download")).toHaveAttribute("download", "review.pptx");
  });
});
