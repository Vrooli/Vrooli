import { ARTIFACT_VIEWER_MIN_WIDTH, type FileViewerProps, type PreviewState } from "@vrooli/file-viewer";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import WorkspaceFilePreview from "./WorkspaceFilePreview";

const mocks = vi.hoisted(() => ({
  controller: {
    state: { status: "idle", open: false, requestedPath: null, model: null, text: null, listing: null, error: null, loadingMore: false, stack: [] } as PreviewState,
    openPreview: vi.fn(),
    reopen: vi.fn(),
    reportError: vi.fn(),
    navigateTo: vi.fn(),
    navigateBack: vi.fn(),
    loadMore: vi.fn(),
    setListOptions: vi.fn(),
  },
}));

// The viewer itself is the package's; this test is about how Web Console wires
// it: the session source, the dock, and the actions it adds to the header.
vi.mock("@vrooli/file-viewer", async (original) => ({
  ...(await original<typeof import("@vrooli/file-viewer")>()),
  useFilePreviewController: () => mocks.controller,
  FileViewer: (props: FileViewerProps) => (
    <div data-testid="mock-file-viewer" data-presentation={String(props.presentation ?? "drawer")} data-scope={props.source.scope}>
      {props.renderHeaderActions?.({ path: "/resolved/file.md", model: { kind: "markdown", archiveMember: false } as never, text: null })}
      <button type="button" onClick={() => props.onFocus?.()}>focus</button>
      <button type="button" onClick={() => props.onRestore?.()}>restore</button>
      <button type="button" onClick={props.onClose}>close</button>
      <div role="separator" onPointerDown={props.onResizeStart} />
    </div>
  ),
}));

function renderPreview(overrides: Partial<React.ComponentProps<typeof WorkspaceFilePreview>> = {}) {
  const props: React.ComponentProps<typeof WorkspaceFilePreview> = {
    request: { sessionId: "session-1", path: "/workspace/file.md", source: "message_link" },
    mobile: false,
    width: 480,
    focusMode: false,
    onWidthChange: vi.fn(),
    onFocus: vi.fn(),
    onRestore: vi.fn(),
    onClose: vi.fn(),
    onHandoff: vi.fn(),
    ...overrides,
  };
  return { props, ...render(<WorkspaceFilePreview {...props} />) };
}

describe("WorkspaceFilePreview", () => {
  it("renders nothing without a request and opens requested files when mounted", () => {
    const { container } = renderPreview({ request: null });
    expect(container).toBeEmptyDOMElement();

    renderPreview();
    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-presentation", "dock");
    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-scope", "session-1");
    expect(mocks.controller.openPreview).toHaveBeenCalledWith("/workspace/file.md", "message_link");
  });

  it("uses the drawer presentation on mobile and forwards handoff and close", () => {
    const onHandoff = vi.fn();
    const onClose = vi.fn();
    renderPreview({ mobile: true, onHandoff, onClose });

    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-presentation", "drawer");
    fireEvent.click(screen.getByTestId("handoff-file-viewer"));
    fireEvent.click(screen.getByRole("button", { name: "close" }));
    expect(onHandoff).toHaveBeenCalledWith("session-1", "/resolved/file.md");
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("clamps desktop resize to the supported viewer width", () => {
    const onWidthChange = vi.fn();
    renderPreview({ width: 480, onWidthChange });
    const separator = screen.getByRole("separator");

    fireEvent.pointerDown(separator, { clientX: 100 });
    fireEvent.pointerMove(window, { clientX: 700 });
    expect(onWidthChange).toHaveBeenCalledWith(ARTIFACT_VIEWER_MIN_WIDTH);

    fireEvent.pointerMove(window, { clientX: -600 });
    expect(onWidthChange).toHaveBeenLastCalledWith(900);
    fireEvent.pointerUp(window);
  });

  it("routes focus controls through the desktop viewer", () => {
    const onFocus = vi.fn();
    const onRestore = vi.fn();
    renderPreview({ onFocus, onRestore, focusMode: false });
    fireEvent.click(screen.getByRole("button", { name: "focus" }));
    expect(onFocus).toHaveBeenCalledOnce();

    fireEvent.click(screen.getByRole("button", { name: "restore" }));
    expect(onRestore).toHaveBeenCalledOnce();
  });
});
