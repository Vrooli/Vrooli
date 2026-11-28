import { ARTIFACT_VIEWER_MIN_WIDTH, type FileViewerProps, type PreviewState } from "@vrooli/file-viewer";
import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
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
  viewerContext: { path: "/resolved/file.md", model: { kind: "markdown", archiveMember: false }, text: null },
}));

// The viewer itself is the package's; this test is about how Web Console wires
// it: the session source, the dock, and the actions it adds to the header.
vi.mock("@vrooli/file-viewer", async (original) => ({
  ...(await original<typeof import("@vrooli/file-viewer")>()),
  useFilePreviewController: () => mocks.controller,
  FileViewer: (props: FileViewerProps) => (
    <div data-testid="mock-file-viewer" data-presentation={String(props.presentation ?? "drawer")} data-scope={props.source.scope}>
      {props.renderHeaderActions?.(mocks.viewerContext as never)}
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
  beforeEach(() => {
    mocks.controller.openPreview.mockClear();
    mocks.viewerContext = { path: "/resolved/file.md", model: { kind: "markdown", archiveMember: false }, text: null };
  });
  it.each([375, 1280])("uses the shared pane and toolbar contract at %i px", (width) => {
    renderPreview({ width, mobile: width < 600 });
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "loading");
    const toolbar = screen.getByRole("toolbar", { name: "File preview controls" });
    expect(toolbar).toHaveAttribute("data-pane-state", "loading");
    const controls = Array.from(toolbar.querySelectorAll<HTMLButtonElement>("button:not(:disabled)"));
    expect(controls[0]).toHaveAttribute("tabindex", "0");
    controls.slice(1).forEach((control) => { expect(control).toHaveAttribute("tabindex", "-1"); });
    expect(screen.getByRole("status")).toHaveTextContent(/File preview: Surface is loading/);
  });

  it("renders nothing without a request and opens requested files when mounted", () => {
    const { container } = renderPreview({ request: null });
    expect(container).toBeEmptyDOMElement();

    renderPreview();
    const surface = screen.getByTestId("mock-file-viewer").closest("[data-surface-owner]");
    expect(surface).toHaveAttribute("data-surface-owner", "file-preview");
    expect(surface).toHaveAttribute("data-surface-access", "read-only");
    expect(surface).toHaveAttribute("data-surface-lease-held", "false");
    expect(surface).toHaveAttribute("data-surface-can-control", "false");
    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-presentation", "dock");
    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-scope", "session-1");
    expect(screen.getByRole("toolbar", { name: "File preview controls" })).toHaveAttribute("data-pane-state", "loading");
    expect(screen.getByTestId("workspace-pane-state")).toHaveAttribute("data-pane-state", "loading");
    expect(screen.getByRole("status")).toHaveTextContent(/File preview: Surface is loading/);
    expect(mocks.controller.openPreview).toHaveBeenCalledWith("/workspace/file.md", "message_link");
  });

  it("opens authoring accessibly, explains missing source recovery, and returns to the same preview", () => {
    renderPreview();
    const action = screen.getByRole("button", { name: "Open design packet authoring" });
    expect(screen.getByRole("toolbar", { name: "File preview controls" })).toContainElement(action);
    expect(action).toHaveAttribute("tabindex", "0");
    action.focus();
    fireEvent.keyDown(action, { key: "ArrowRight" });
    expect(screen.getByTestId("handoff-file-viewer")).toHaveFocus();
    action.focus();
    fireEvent.click(action);
    const dialog = screen.getByRole("dialog", { name: "Author from retained evidence" });
    expect(dialog).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Close authoring workflow" })).toHaveFocus();
    expect(screen.getByRole("navigation", { name: "Design packet steps" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Inputs & lineage/ })).toHaveAttribute("aria-current", "step");
    expect(screen.getByRole("button", { name: /Mappings & omissions/ })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Validate and build packet" })).toBeDisabled();
    expect(screen.getByRole("toolbar", { name: "File preview controls" })).not.toContainElement(dialog);
    expect(screen.getByLabelText(/Design brief, source map/)).toBeEnabled();
    expect(screen.getByText("The current preview is not a readable HTML source. In Step 1, choose the retained brief, source map, selected-image manifest, referenced images and editable HTML source to continue.")).toHaveAttribute("role", "status");

    fireEvent.keyDown(dialog, { key: "Escape" });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(action).toHaveFocus();
    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-scope", "session-1");
    expect(screen.getByRole("toolbar", { name: "File preview controls" })).toHaveAttribute("data-pane-state", "loading");
    expect(mocks.controller.openPreview).toHaveBeenCalledTimes(1);

    fireEvent.click(action);
    fireEvent.click(screen.getByRole("button", { name: "Close authoring workflow" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(action).toHaveFocus();
    expect(mocks.controller.openPreview).toHaveBeenCalledTimes(1);
  });

  it("uses the drawer presentation on mobile and forwards handoff and close", () => {
    const onHandoff = vi.fn();
    const onClose = vi.fn();
    renderPreview({ mobile: true, onHandoff, onClose });

    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-presentation", "drawer");
    expect(screen.getByRole("toolbar", { name: "File preview controls" })).toHaveAttribute("data-pane-state", "loading");
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
