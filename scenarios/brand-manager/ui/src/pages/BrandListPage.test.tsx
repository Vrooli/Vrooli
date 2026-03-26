import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import BrandListPage from "./BrandListPage";
import type { Brand } from "../lib/api";

// [REQ:BM-REQ-UI-LIBRARY] [REQ:BM-REQ-UI-DASHBOARD]

const testBrands: Brand[] = [
  {
    id: "b1",
    name: "Alpha Brand",
    description: "First brand",
    version: 1,
    created_at: "2026-01-01T00:00:00Z",
    updated_at: "2026-01-01T00:00:00Z",
    colors: { primary: "#ff0000" },
  },
  {
    id: "b2",
    name: "Beta Brand",
    version: 2,
    created_at: "2026-02-01T00:00:00Z",
    updated_at: "2026-02-01T00:00:00Z",
  },
];

// Mock the api module
vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchBrands: vi.fn(),
  };
});

import { fetchBrands } from "../lib/api";
const mockFetchBrands = vi.mocked(fetchBrands);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("BrandListPage", () => {
  const onNavigate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    mockFetchBrands.mockReturnValue(new Promise(() => {})); // never resolves
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    expect(screen.getByText("Loading brands...")).toBeTruthy();
  });

  it("renders brand list when data loads", async () => {
    mockFetchBrands.mockResolvedValue(testBrands);
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Alpha Brand")).toBeTruthy();
    });
    expect(screen.getByText("Beta Brand")).toBeTruthy();
  });

  it("renders empty state when no brands", async () => {
    mockFetchBrands.mockResolvedValue([]);
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-list-empty")).toBeTruthy();
    });
    expect(screen.getByText(/No brands found/)).toBeTruthy();
  });

  it("navigates to new brand page on create button click", async () => {
    mockFetchBrands.mockResolvedValue([]);
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("create-brand-btn")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("create-brand-btn"));
    expect(onNavigate).toHaveBeenCalledWith("/brands/new");
  });

  it("renders search input", async () => {
    mockFetchBrands.mockResolvedValue(testBrands);
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    expect(screen.getByTestId("brand-search-input")).toBeTruthy();
  });

  it("renders error state on API failure", async () => {
    mockFetchBrands.mockRejectedValue(new Error("Network error"));
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-list-error")).toBeTruthy();
    });
  });

  it("renders brand grid container when brands exist", async () => {
    mockFetchBrands.mockResolvedValue(testBrands);
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-list-grid")).toBeTruthy();
    });
  });

  it("has refresh button", () => {
    mockFetchBrands.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    expect(screen.getByTestId("refresh-brands-btn")).toBeTruthy();
  });

  it("renders page title", () => {
    mockFetchBrands.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<BrandListPage onNavigate={onNavigate} />);

    expect(screen.getByText("Brand Library")).toBeTruthy();
  });
});
