import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import WorkspaceFilePreview from "./WorkspaceFilePreview";
import { ARTIFACT_VIEWER_MIN_WIDTH } from "./artifactViewerLayout";
import type { PreviewState } from "./types";

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

vi.mock("./useFilePreviewController", () => ({
  useFilePreviewController: () => mocks.controller,
}));

vi.mock("../MessagesFileViewer", () => ({
  default: (props: Record<string, unknown>) => (
    <div data-testid="mock-file-viewer" data-presentation={String(props.presentation ?? "drawer")}>
      <button type="button" onClick={() => (props.onHandoff as ((path: string) => void) | undefined)?.("/resolved/file.md")}>handoff</button>
      <button type="button" onClick={() => (props.onFocus as (() => void) | undefined)?.()}>focus</button>
      <button type="button" onClick={() => (props.onRestore as (() => void) | undefined)?.()}>restore</button>
      <button type="button" onClick={props.onClose as () => void}>close</button>
      <div role="separator" onPointerDown={props.onResizeStart as React.PointerEventHandler<HTMLDivElement>} />
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
    expect(mocks.controller.openPreview).toHaveBeenCalledWith("/workspace/file.md", "message_link");
  });

  it("uses the drawer presentation on mobile and forwards handoff and close", () => {
    const onHandoff = vi.fn();
    const onClose = vi.fn();
    renderPreview({ mobile: true, onHandoff, onClose });

    expect(screen.getByTestId("mock-file-viewer")).toHaveAttribute("data-presentation", "drawer");
    fireEvent.click(screen.getByRole("button", { name: "handoff" }));
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
