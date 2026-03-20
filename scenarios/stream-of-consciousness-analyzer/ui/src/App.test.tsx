import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi } from "vitest";
import "@testing-library/jest-dom/vitest";
import App from "./App";

// Mock the API module
vi.mock("./lib/api", () => ({
  fetchHealth: vi.fn().mockResolvedValue({ status: "ok" }),
  listSchemes: vi.fn().mockResolvedValue([]),
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
}));

// Mock @vrooli/api-base
vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

function renderApp() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
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
});
