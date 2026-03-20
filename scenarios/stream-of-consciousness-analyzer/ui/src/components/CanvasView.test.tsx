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
});
