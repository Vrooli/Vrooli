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

  // [REQ:P0-004] Link mode guidance and accessibility
  it("shows link mode hint when link mode is activated", async () => {
    mockListThoughts.mockResolvedValue([]);
    renderComponent();
    const linkBtn = screen.getByTestId("link-mode-btn");
    fireEvent.click(linkBtn);
    const hint = screen.getByTestId("link-mode-hint");
    expect(hint).toBeInTheDocument();
    expect(hint).toHaveTextContent(/Click a thought to select it as the source/);
  });

  it("hides link mode hint when link mode is deactivated", async () => {
    mockListThoughts.mockResolvedValue([]);
    renderComponent();
    const linkBtn = screen.getByTestId("link-mode-btn");
    fireEvent.click(linkBtn);
    expect(screen.getByTestId("link-mode-hint")).toBeInTheDocument();
    fireEvent.click(linkBtn);
    expect(screen.queryByTestId("link-mode-hint")).not.toBeInTheDocument();
  });

  it("announces link mode state change via aria-live", () => {
    mockListThoughts.mockResolvedValue([]);
    renderComponent();
    const linkBtn = screen.getByTestId("link-mode-btn");
    fireEvent.click(linkBtn);
    const liveRegion = screen.getByRole("status");
    expect(liveRegion).toHaveTextContent(/Link mode active/);
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

  // [REQ:P0-004] Full link flow: activate link mode → select source → select target
  it("completes a link between two thoughts via click sequence", async () => {
    mockListThoughts.mockResolvedValue([
      { id: "t1", scheme_id: "scheme-1", title: "Source", body: "", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
      { id: "t2", scheme_id: "scheme-1", title: "Target", body: "", canvas_x: 100, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    mockListEdges.mockResolvedValue([]);
    mockCreateEdge.mockResolvedValue({ id: "e-new", source_id: "t1", target_id: "t2", label: "", created_at: "" });
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Source")).toBeInTheDocument();
    });

    // Activate link mode
    fireEvent.click(screen.getByTestId("link-mode-btn"));
    expect(screen.getByTestId("link-mode-hint")).toHaveTextContent(/Click a thought to select it as the source/);

    // Click source thought — transitions from WAITING to source selected
    fireEvent.click(screen.getByText("Source"));

    // Hint should now show "click another thought to connect"
    await waitFor(() => {
      expect(screen.getByTestId("link-mode-hint")).toHaveTextContent(/click another thought to connect them/i);
    });

    // Aria-live should announce source selected
    const liveRegion = screen.getByRole("status");
    expect(liveRegion).toHaveTextContent(/Source selected/);

    // Click target thought — creates the edge
    fireEvent.click(screen.getByText("Target"));
    await waitFor(() => {
      expect(mockCreateEdge).toHaveBeenCalledWith("t1", { target_id: "t2", label: "" });
    });
  });

  // [REQ:P0-004] Clicking the same thought as source does not create a self-loop
  it("does not link a thought to itself", async () => {
    mockListThoughts.mockResolvedValue([
      { id: "t1", scheme_id: "scheme-1", title: "Only", body: "", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    mockListEdges.mockResolvedValue([]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Only")).toBeInTheDocument();
    });

    // Activate link mode → select source
    fireEvent.click(screen.getByTestId("link-mode-btn"));
    fireEvent.click(screen.getByText("Only"));

    // Click the same thought again — handleThoughtClick guards against self-link
    fireEvent.click(screen.getByText("Only"));
    expect(mockCreateEdge).not.toHaveBeenCalled();
  });

  // [REQ:P0-004] Delete edge from the connections list
  it("deletes an edge when its delete button is clicked", async () => {
    mockListThoughts.mockResolvedValue([
      { id: "t1", scheme_id: "scheme-1", title: "A", body: "", canvas_x: 0, canvas_y: 0, created_at: "", updated_at: "" },
      { id: "t2", scheme_id: "scheme-1", title: "B", body: "", canvas_x: 100, canvas_y: 0, created_at: "", updated_at: "" },
    ]);
    mockListEdges.mockResolvedValue([
      { id: "e1", source_id: "t1", target_id: "t2", label: "", created_at: "" },
    ]);
    mockDeleteEdge.mockResolvedValue(undefined);
    renderComponent();
    await waitFor(() => {
      expect(screen.getAllByTestId("edge-item").length).toBe(1);
    });
    const edgeItem = screen.getByTestId("edge-item");
    const deleteBtn = edgeItem.querySelector("button");
    expect(deleteBtn).toBeTruthy();
    if (deleteBtn) fireEvent.click(deleteBtn);
    await waitFor(() => {
      expect(mockDeleteEdge).toHaveBeenCalledWith("t1", "e1");
    });
  });

  // [REQ:P0-004] Delete a thought from the graph
  it("deletes a thought when its delete button is clicked", async () => {
    mockListThoughts.mockResolvedValue([
      { id: "t1", scheme_id: "scheme-1", title: "Deletable", body: "", canvas_x: 50, canvas_y: 50, created_at: "", updated_at: "" },
    ]);
    mockListEdges.mockResolvedValue([]);
    mockDeleteThought.mockResolvedValue(undefined);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Deletable")).toBeInTheDocument();
    });
    // ThoughtNode has a delete button with thought title in aria-label
    const deleteBtn = screen.getByLabelText("Delete thought: Deletable");
    fireEvent.click(deleteBtn);
    await waitFor(() => {
      expect(mockDeleteThought).toHaveBeenCalledWith("t1", expect.anything());
    });
  });

  // [REQ:P0-004] Error banner displays when a mutation fails
  it("shows error banner when createThought mutation fails", async () => {
    mockListThoughts.mockResolvedValue([]);
    mockCreateThought.mockRejectedValue(new Error("Network error"));
    renderComponent();
    fireEvent.change(screen.getByTestId("thought-title-input"), { target: { value: "Fail" } });
    fireEvent.click(screen.getByTestId("create-thought-btn"));
    await waitFor(
      () => {
        expect(screen.getByTestId("error-banner")).toBeInTheDocument();
      },
      { timeout: 3000 },
    );
  });

  // [REQ:P0-004] Edge connection list shows source→target names
  it("displays source and target names in the connections list", async () => {
    mockListThoughts.mockResolvedValue([
      { id: "t1", scheme_id: "scheme-1", title: "Start", body: "", canvas_x: 10, canvas_y: 20, created_at: "", updated_at: "" },
      { id: "t2", scheme_id: "scheme-1", title: "End", body: "", canvas_x: 200, canvas_y: 100, created_at: "", updated_at: "" },
    ]);
    mockListEdges.mockResolvedValue([
      { id: "e1", source_id: "t1", target_id: "t2", label: "", created_at: "" },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getAllByTestId("edge-item").length).toBe(1);
    });
    const edgeItem = screen.getByTestId("edge-item");
    // Should show source and target thought titles
    expect(edgeItem).toHaveTextContent("Start");
    expect(edgeItem).toHaveTextContent("End");
    // Delete button should have descriptive aria-label
    const deleteBtn = edgeItem.querySelector("button");
    expect(deleteBtn).toHaveAttribute("aria-label", "Delete connection from Start to End");
  });
});
