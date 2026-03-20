import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";

const mockListSchemes = vi.fn();
const mockCreateScheme = vi.fn();
const mockDeleteScheme = vi.fn();

vi.mock("../lib/api", () => ({
  listSchemes: (...args: unknown[]) => mockListSchemes(...args) as unknown,
  createScheme: (...args: unknown[]) => mockCreateScheme(...args) as unknown,
  deleteScheme: (...args: unknown[]) => mockDeleteScheme(...args) as unknown,
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

import { SchemeList } from "./SchemeList";

function renderComponent(props: { activeSchemeId?: string | null; onSelect?: () => void } = {}) {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <SchemeList
        activeSchemeId={props.activeSchemeId ?? null}
        onSelect={props.onSelect ?? vi.fn()}
      />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockListSchemes.mockReset();
  mockCreateScheme.mockReset();
  mockDeleteScheme.mockReset();
});

// [REQ:P0-001] Scheme CRUD sidebar
describe("SchemeList", () => {
  it("renders the scheme list container", async () => {
    mockListSchemes.mockResolvedValue([]);
    renderComponent();
    expect(screen.getByTestId("scheme-list")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    mockListSchemes.mockReturnValue(new Promise(() => {})); // never resolves
    renderComponent();
    expect(screen.getByText("Loading...")).toBeInTheDocument();
  });

  it("renders schemes from API", async () => {
    mockListSchemes.mockResolvedValue([
      { id: "s1", name: "First", created_at: "", updated_at: "" },
      { id: "s2", name: "Second", created_at: "", updated_at: "" },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("First")).toBeInTheDocument();
      expect(screen.getByText("Second")).toBeInTheDocument();
    });
  });

  it("highlights the active scheme", async () => {
    mockListSchemes.mockResolvedValue([
      { id: "s1", name: "Active", created_at: "", updated_at: "" },
    ]);
    renderComponent({ activeSchemeId: "s1" });
    await waitFor(() => {
      const item = screen.getByTestId("scheme-item");
      expect(item.className).toContain("bg-white/10");
    });
  });

  it("calls onSelect when a scheme is clicked", async () => {
    const scheme = { id: "s1", name: "Click Me", created_at: "", updated_at: "" };
    mockListSchemes.mockResolvedValue([scheme]);
    const onSelect = vi.fn();
    renderComponent({ onSelect });
    await waitFor(() => {
      expect(screen.getByText("Click Me")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("scheme-item"));
    expect(onSelect).toHaveBeenCalledWith(scheme);
  });

  it("calls createScheme when create button is clicked", async () => {
    mockListSchemes.mockResolvedValue([]);
    const newScheme = { id: "new", name: "Untitled", created_at: "", updated_at: "" };
    mockCreateScheme.mockResolvedValue(newScheme);
    const onSelect = vi.fn();
    renderComponent({ onSelect });
    fireEvent.click(screen.getByTestId("create-scheme-btn"));
    await waitFor(() => {
      expect(mockCreateScheme).toHaveBeenCalledWith("Untitled");
    });
  });

  it("shows error banner when list fails", async () => {
    mockListSchemes.mockRejectedValue(new Error("Network error"));
    renderComponent();
    await waitFor(
      () => {
        expect(screen.getByTestId("error-banner")).toBeInTheDocument();
      },
      { timeout: 3000 },
    );
  });
});
