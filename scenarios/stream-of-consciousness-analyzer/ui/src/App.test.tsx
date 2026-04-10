import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";
import App from "./App";

const mockListSchemes = vi.fn();

// Mock the API module
vi.mock("./lib/api", () => ({
  fetchHealth: vi.fn().mockResolvedValue({ status: "ok" }),
  listSchemes: (...args: unknown[]) => mockListSchemes(...args) as unknown,
  createScheme: vi.fn().mockResolvedValue({ id: "1", name: "Test" }),
  getScheme: vi.fn().mockResolvedValue({ id: "1", name: "Test" }),
  updateScheme: vi.fn().mockResolvedValue({ id: "1", name: "Test" }),
  deleteScheme: vi.fn().mockResolvedValue(undefined),
  listInformation: vi.fn().mockResolvedValue([]),
  createInformation: vi.fn().mockResolvedValue({ id: "1" }),
  updateInformation: vi.fn().mockResolvedValue({ id: "1" }),
  deleteInformation: vi.fn().mockResolvedValue(undefined),
  listThoughts: vi.fn().mockResolvedValue([]),
  createThought: vi.fn().mockResolvedValue({ id: "1" }),
  updateThought: vi.fn().mockResolvedValue({ id: "1" }),
  deleteThought: vi.fn().mockResolvedValue(undefined),
  listEdges: vi.fn().mockResolvedValue([]),
  createEdge: vi.fn().mockResolvedValue({ id: "1" }),
  deleteEdge: vi.fn().mockResolvedValue(undefined),
  exportScheme: vi.fn().mockResolvedValue({ scheme: {}, information: [], thoughts: [], edges: [], export_format: "vrooli-graph-v1" }),
  listProviders: vi.fn().mockResolvedValue([]),
  generateSuggestions: vi.fn().mockResolvedValue({ suggestions: [], provider: "ollama" }),
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

// Mock @vrooli/api-base
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

beforeEach(() => {
  mockListSchemes.mockReset();
  mockListSchemes.mockResolvedValue([]);
});

function renderApp() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  return render(
    <QueryClientProvider client={qc}>
      <App />
    </QueryClientProvider>
  );
}

// [REQ:P0-001] App renders with scheme selector ready for interaction
describe("App", () => {
  it("renders the app root", () => {
    renderApp();
    expect(screen.getByTestId("app-root")).toBeInTheDocument();
  });

  // [REQ:P0-002] Quick text entry - app shows prompt to select scheme
  it("shows empty state when no scheme selected", () => {
    renderApp();
    expect(screen.getByText("Select or create a scheme")).toBeInTheDocument();
  });

  // [REQ:P0-001] Scheme list is present for zero-friction capture
  it("renders scheme list sidebar", () => {
    renderApp();
    expect(screen.getByTestId("scheme-list")).toBeInTheDocument();
  });

  // [REQ:P0-001] Create scheme button is accessible
  it("renders create scheme button", () => {
    renderApp();
    expect(screen.getByTestId("create-scheme-btn")).toBeInTheDocument();
  });

  // [REQ:P0-001] Does not show view toggle buttons when no scheme is selected
  it("hides view toggle buttons when no scheme is selected", () => {
    renderApp();
    expect(screen.queryByTestId("view-canvas-btn")).not.toBeInTheDocument();
    expect(screen.queryByTestId("view-graph-btn")).not.toBeInTheDocument();
  });

  // [REQ:P0-003] Shows view toggle and content panels after selecting a scheme
  it("shows view toggle buttons and canvas when a scheme is selected", async () => {
    mockListSchemes.mockResolvedValue([
      { id: "s1", name: "My Scheme", created_at: "", updated_at: "" },
    ]);
    renderApp();

    // Click on the scheme to select it
    const schemeBtn = await screen.findByText("My Scheme");
    fireEvent.click(schemeBtn);

    await waitFor(() => {
      expect(screen.getByTestId("view-canvas-btn")).toBeInTheDocument();
      expect(screen.getByTestId("view-graph-btn")).toBeInTheDocument();
    });

    // Canvas view should be the default
    expect(screen.getByTestId("view-canvas-btn")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("view-graph-btn")).toHaveAttribute("aria-pressed", "false");

    // Canvas view should be rendered
    expect(screen.getByTestId("canvas-view")).toBeInTheDocument();
  });

  // [REQ:P0-003] Switching between canvas and graph view modes
  it("switches to graph view when graph button is clicked", async () => {
    mockListSchemes.mockResolvedValue([
      { id: "s1", name: "My Scheme", created_at: "", updated_at: "" },
    ]);
    renderApp();

    const schemeBtn = await screen.findByText("My Scheme");
    fireEvent.click(schemeBtn);

    await waitFor(() => {
      expect(screen.getByTestId("view-graph-btn")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("view-graph-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("view-graph-btn")).toHaveAttribute("aria-pressed", "true");
      expect(screen.getByTestId("view-canvas-btn")).toHaveAttribute("aria-pressed", "false");
      expect(screen.getByTestId("graph-view")).toBeInTheDocument();
    });
  });

  // [REQ:P0-001] Displays the selected scheme name in the header
  it("shows selected scheme name in header", async () => {
    mockListSchemes.mockResolvedValue([
      { id: "s1", name: "Deep Thoughts", created_at: "", updated_at: "" },
    ]);
    renderApp();

    // Before selection, shows default title
    expect(screen.getByText("Stream of Consciousness")).toBeInTheDocument();

    const schemeBtn = await screen.findByText("Deep Thoughts");
    fireEvent.click(schemeBtn);

    await waitFor(() => {
      // The scheme name should appear in the header (h1)
      const header = screen.getByRole("heading", { level: 1 });
      expect(header).toHaveTextContent("Deep Thoughts");
    });
  });

  // [REQ:P0-003] Export button appears when a scheme is selected
  it("shows export button when a scheme is selected", async () => {
    mockListSchemes.mockResolvedValue([
      { id: "s1", name: "Export Test", created_at: "", updated_at: "" },
    ]);
    renderApp();

    const schemeBtn = await screen.findByText("Export Test");
    fireEvent.click(schemeBtn);

    await waitFor(() => {
      expect(screen.getByTestId("export-btn")).toBeInTheDocument();
    });
  });
});
