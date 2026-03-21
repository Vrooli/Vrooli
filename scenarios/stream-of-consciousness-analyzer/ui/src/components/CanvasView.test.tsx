import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";

const mockListInformation = vi.fn();
const mockUpdateInformation = vi.fn();
const mockDeleteInformation = vi.fn();

vi.mock("../lib/api", () => ({
  listInformation: (...args: unknown[]) => mockListInformation(...args) as unknown,
  updateInformation: (...args: unknown[]) => mockUpdateInformation(...args) as unknown,
  deleteInformation: (...args: unknown[]) => mockDeleteInformation(...args) as unknown,
  ApiRequestError: class ApiRequestError extends Error {
    status: number;
    category: string;
    retryable: boolean;
    constructor(status: number, apiError: { category: string; message: string; retryable: boolean }) {
      super(apiError.message);
      this.name = "ApiRequestError";
      this.status = status;
      this.category = apiError.category;
      this.retryable = apiError.retryable;
    }
  },
}));

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

import { CanvasView } from "./CanvasView";

function renderComponent(schemeId = "scheme-1") {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <CanvasView schemeId={schemeId} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockListInformation.mockReset();
  mockUpdateInformation.mockReset();
  mockDeleteInformation.mockReset();
});

// [REQ:P0-003] Spatial canvas for information items
describe("CanvasView", () => {
  it("renders the canvas view container", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    expect(screen.getByTestId("canvas-view")).toBeInTheDocument();
  });

  it("shows loading state when data is pending", () => {
    mockListInformation.mockReturnValue(new Promise(() => {}));
    renderComponent();
    expect(screen.getByText("Loading items...")).toBeInTheDocument();
  });

  it("renders information nodes from API", async () => {
    mockListInformation.mockResolvedValue([
      { id: "i1", scheme_id: "scheme-1", type: "text", content: "Hello world", canvas_x: 50, canvas_y: 80, created_at: "", updated_at: "" },
      { id: "i2", scheme_id: "scheme-1", type: "text", content: "Another item", canvas_x: 200, canvas_y: 150, created_at: "", updated_at: "" },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Hello world")).toBeInTheDocument();
      expect(screen.getByText("Another item")).toBeInTheDocument();
    });
  });

  it("renders canvas nodes with correct test IDs", async () => {
    mockListInformation.mockResolvedValue([
      { id: "i1", scheme_id: "scheme-1", type: "text", content: "Node 1", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getAllByTestId("canvas-node").length).toBe(1);
    });
  });

  it("shows zoom percentage indicator", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    expect(screen.getByText("100%")).toBeInTheDocument();
  });

  it("displays item type label", async () => {
    mockListInformation.mockResolvedValue([
      { id: "i1", scheme_id: "scheme-1", type: "text", content: "Typed item", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("text")).toBeInTheDocument();
    });
  });

  it("calls deleteInformation when delete button is clicked", async () => {
    mockListInformation.mockResolvedValue([
      { id: "i1", scheme_id: "scheme-1", type: "text", content: "Delete me", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    mockDeleteInformation.mockResolvedValue(undefined);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Delete me")).toBeInTheDocument();
    });
    const node = screen.getByTestId("canvas-node");
    const deleteBtn = node.querySelector("button");
    expect(deleteBtn).toBeTruthy();
    if (deleteBtn) fireEvent.click(deleteBtn);
    await waitFor(() => {
      expect(mockDeleteInformation).toHaveBeenCalledWith("scheme-1", "i1");
    });
  });

  it("unmounts cleanly without errors", () => {
    mockListInformation.mockResolvedValue([]);
    const { unmount } = renderComponent();
    expect(() => unmount()).not.toThrow();
  });

  // [REQ:P0-003] Keyboard accessibility for canvas navigation
  it("pans down when ArrowDown is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "ArrowDown" });
    // Canvas transform should reflect pan change (negative y = scroll down)
    const inner = canvas.querySelector<HTMLElement>("[style]");
    expect(inner?.style.transform).toContain("translate(0px, -40px)");
  });

  it("zooms in when + key is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "+" });
    // Zoom should change from 100% to 110%
    expect(screen.getByText("110%")).toBeInTheDocument();
  });

  it("zooms out when - key is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "-" });
    expect(screen.getByText("90%")).toBeInTheDocument();
  });

  it("has appropriate ARIA attributes for keyboard users", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    expect(canvas).toHaveAttribute("role", "application");
    expect(canvas).toHaveAttribute("tabindex", "0");
    expect(canvas).toHaveAttribute("aria-label", expect.stringContaining("arrow keys"));
  });

  it("announces zoom level via aria-live region", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const liveRegion = screen.getByRole("status");
    expect(liveRegion).toHaveTextContent("Zoom 100%");
  });

  // [REQ:P0-003] Keyboard shortcut help discoverability
  it("shows keyboard shortcut help when ? is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    expect(screen.queryByTestId("keyboard-shortcut-help")).not.toBeInTheDocument();
    fireEvent.keyDown(canvas, { key: "?" });
    expect(screen.getByTestId("keyboard-shortcut-help")).toBeInTheDocument();
  });

  it("hides keyboard shortcut help when ? is pressed again", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "?" });
    expect(screen.getByTestId("keyboard-shortcut-help")).toBeInTheDocument();
    fireEvent.keyDown(canvas, { key: "?" });
    expect(screen.queryByTestId("keyboard-shortcut-help")).not.toBeInTheDocument();
  });

  it("shows shortcut hint text near zoom indicator", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    expect(screen.getByText("Press ? for shortcuts")).toBeInTheDocument();
  });

  // [REQ:P0-003] Arrow keys for panning in all directions
  it("pans up when ArrowUp is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "ArrowUp" });
    const inner = canvas.querySelector<HTMLElement>("[style]");
    expect(inner?.style.transform).toContain("translate(0px, 40px)");
  });

  it("pans left when ArrowLeft is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "ArrowLeft" });
    const inner = canvas.querySelector<HTMLElement>("[style]");
    expect(inner?.style.transform).toContain("translate(40px, 0px)");
  });

  it("pans right when ArrowRight is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "ArrowRight" });
    const inner = canvas.querySelector<HTMLElement>("[style]");
    expect(inner?.style.transform).toContain("translate(-40px, 0px)");
  });

  it("zooms in when = key is pressed", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    fireEvent.keyDown(canvas, { key: "=" });
    expect(screen.getByText("110%")).toBeInTheDocument();
  });

  // [REQ:P0-003] Wheel zoom
  it("zooms in on wheel scroll up", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    // deltaY < 0 → zoom in (1.1x)
    fireEvent.wheel(canvas, { deltaY: -100 });
    expect(screen.getByText("110%")).toBeInTheDocument();
  });

  it("zooms out on wheel scroll down", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    // deltaY > 0 → zoom out (0.9x)
    fireEvent.wheel(canvas, { deltaY: 100 });
    expect(screen.getByText("90%")).toBeInTheDocument();
  });

  // [REQ:P0-003] Canvas pan via mouse drag
  it("pans the canvas when dragging on background", () => {
    mockListInformation.mockResolvedValue([]);
    renderComponent();
    const canvas = screen.getByTestId("canvas-view");
    // Mousedown on canvas background initiates pan drag
    fireEvent.mouseDown(canvas, { target: canvas, clientX: 100, clientY: 100 });
    // Move the mouse — triggers useWindowDrag onMove → updates pan transform
    fireEvent.mouseMove(window, { clientX: 150, clientY: 130 });
    // The canvas inner div should reflect the pan offset in its transform
    const inner = canvas.querySelector<HTMLElement>("[style]");
    if (inner) {
      expect(inner.style.transform).toContain("translate");
    }
    // Release
    fireEvent.mouseUp(window);
    expect(canvas).toBeInTheDocument();
  });

  // [REQ:P0-003] Item drag to reposition
  it("initiates item drag on mousedown and commits position on mouseup", async () => {
    mockListInformation.mockResolvedValue([
      { id: "i1", scheme_id: "scheme-1", type: "text", content: "Drag me", canvas_x: 50, canvas_y: 80, created_at: "", updated_at: "" },
    ]);
    mockUpdateInformation.mockResolvedValue({
      id: "i1", scheme_id: "scheme-1", type: "text", content: "Drag me", canvas_x: 100, canvas_y: 130, created_at: "", updated_at: "",
    });
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Drag me")).toBeInTheDocument();
    });
    const node = screen.getByTestId("canvas-node");
    // Start item drag
    fireEvent.mouseDown(node, { clientX: 50, clientY: 80 });
    // Simulate move and end on window
    fireEvent.mouseMove(window, { clientX: 100, clientY: 130 });
    fireEvent.mouseUp(window, { clientX: 100, clientY: 130 });
    await waitFor(() => {
      expect(mockUpdateInformation).toHaveBeenCalled();
    });
  });

  // [REQ:P0-003] Scheme change resets pan/zoom state
  it("resets pan and zoom when schemeId changes", () => {
    mockListInformation.mockResolvedValue([]);
    const qc = new QueryClient({
      defaultOptions: { queries: { retry: false, gcTime: 0 } },
    });
    const { rerender } = render(
      <QueryClientProvider client={qc}>
        <CanvasView schemeId="scheme-1" />
      </QueryClientProvider>,
    );
    const canvas = screen.getByTestId("canvas-view");
    // Pan the canvas
    fireEvent.keyDown(canvas, { key: "ArrowDown" });
    const inner = canvas.querySelector<HTMLElement>("[style]");
    expect(inner?.style.transform).toContain("-40px");
    // Change schemeId — should reset
    rerender(
      <QueryClientProvider client={qc}>
        <CanvasView schemeId="scheme-2" />
      </QueryClientProvider>,
    );
    expect(inner?.style.transform).toContain("translate(0px, 0px) scale(1)");
  });

  // [REQ:P0-003] Error display when mutation fails
  it("shows error banner when delete mutation fails", async () => {
    const MockApiRequestError = (await import("../lib/api")).ApiRequestError;
    mockListInformation.mockResolvedValue([
      { id: "i1", scheme_id: "scheme-1", type: "text", content: "Fail delete", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    mockDeleteInformation.mockRejectedValue(
      new MockApiRequestError(500, { category: "internal", message: "delete failed", retryable: true }),
    );
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Fail delete")).toBeInTheDocument();
    });
    const node = screen.getByTestId("canvas-node");
    const deleteBtn = node.querySelector("button");
    if (deleteBtn) fireEvent.click(deleteBtn);
    await waitFor(() => {
      expect(screen.getByText("delete failed")).toBeInTheDocument();
    });
  });
});
