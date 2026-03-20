import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";

const mockListThoughts = vi.fn();
const mockCreateThought = vi.fn();
const mockDeleteThought = vi.fn();
const mockListEdges = vi.fn();
const mockCreateEdge = vi.fn();
const mockDeleteEdge = vi.fn();

vi.mock("../lib/api", () => ({
  listThoughts: (...args: unknown[]) => mockListThoughts(...args) as unknown,
  createThought: (...args: unknown[]) => mockCreateThought(...args) as unknown,
  deleteThought: (...args: unknown[]) => mockDeleteThought(...args) as unknown,
  listEdges: (...args: unknown[]) => mockListEdges(...args) as unknown,
  createEdge: (...args: unknown[]) => mockCreateEdge(...args) as unknown,
  deleteEdge: (...args: unknown[]) => mockDeleteEdge(...args) as unknown,
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

import { GraphView } from "./GraphView";

function renderComponent(schemeId = "scheme-1") {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <GraphView schemeId={schemeId} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockListThoughts.mockReset();
  mockCreateThought.mockReset();
  mockDeleteThought.mockReset();
  mockListEdges.mockReset();
  mockCreateEdge.mockReset();
  mockDeleteEdge.mockReset();
});

// [REQ:P0-003] [REQ:P0-004] Dual-view thought graph with edge linking
describe("GraphView", () => {
  it("renders the graph view container", () => {
    mockListThoughts.mockResolvedValue([]);
    renderComponent();
    expect(screen.getByTestId("graph-view")).toBeInTheDocument();
  });

  it("shows empty state when no thoughts exist", async () => {
    mockListThoughts.mockResolvedValue([]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText(/No thoughts yet/)).toBeInTheDocument();
    });
  });

  it("renders thought nodes from API", async () => {
    mockListThoughts.mockResolvedValue([
      { id: "t1", scheme_id: "scheme-1", title: "First Thought", body: "", canvas_x: 50, canvas_y: 50, created_at: "", updated_at: "" },
      { id: "t2", scheme_id: "scheme-1", title: "Second Thought", body: "Some body", canvas_x: 200, canvas_y: 100, created_at: "", updated_at: "" },
    ]);
    mockListEdges.mockResolvedValue([]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("First Thought")).toBeInTheDocument();
      expect(screen.getByText("Second Thought")).toBeInTheDocument();
    });
  });

  it("renders create thought input and button", () => {
    mockListThoughts.mockResolvedValue([]);
    renderComponent();
    expect(screen.getByTestId("thought-title-input")).toBeInTheDocument();
    expect(screen.getByTestId("create-thought-btn")).toBeInTheDocument();
  });

  it("creates a thought when clicking the add button", async () => {
    mockListThoughts.mockResolvedValue([]);
    const newThought = {
      id: "t-new",
      scheme_id: "scheme-1",
      title: "My Thought",
      body: "",
      canvas_x: 100,
      canvas_y: 100,
      created_at: "",
      updated_at: "",
    };
    mockCreateThought.mockResolvedValue(newThought);
    renderComponent();
    fireEvent.change(screen.getByTestId("thought-title-input"), {
      target: { value: "My Thought" },
    });
    fireEvent.click(screen.getByTestId("create-thought-btn"));
    await waitFor(() => {
      expect(mockCreateThought).toHaveBeenCalled();
    });
  });

  it("creates a thought on Enter key in input", async () => {
    mockListThoughts.mockResolvedValue([]);
    mockCreateThought.mockResolvedValue({
      id: "t-new", scheme_id: "scheme-1", title: "Enter Thought", body: "",
      canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "",
    });
    renderComponent();
    const input = screen.getByTestId("thought-title-input");
    fireEvent.change(input, { target: { value: "Enter Thought" } });
    fireEvent.keyDown(input, { key: "Enter" });
    await waitFor(() => {
      expect(mockCreateThought).toHaveBeenCalled();
    });
  });

  it("renders link mode button", () => {
    mockListThoughts.mockResolvedValue([]);
    renderComponent();
    expect(screen.getByTestId("link-mode-btn")).toBeInTheDocument();
  });

  it("renders edge items when edges exist", async () => {
    mockListThoughts.mockResolvedValue([
      { id: "t1", scheme_id: "scheme-1", title: "A", body: "", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
      { id: "t2", scheme_id: "scheme-1", title: "B", body: "", canvas_x: 100, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    mockListEdges.mockResolvedValue([
      { id: "e1", source_id: "t1", target_id: "t2", label: "", created_at: "" },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getAllByTestId("edge-item").length).toBe(1);
    });
  });
});
