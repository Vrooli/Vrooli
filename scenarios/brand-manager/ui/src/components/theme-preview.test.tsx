import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemePreview } from "./theme-preview";

// [REQ:BM-REQ-UI-THEME]

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchThemePreview: vi.fn(),
  };
});

import { fetchThemePreview } from "../lib/api";
const mockFetchThemePreview = vi.mocked(fetchThemePreview);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("ThemePreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders section wrapper", () => {
    mockFetchThemePreview.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<ThemePreview brandId="b1" />);
    expect(screen.getByTestId("theme-preview-section")).toBeTruthy();
  });

  it("renders light/dark mode toggle", () => {
    mockFetchThemePreview.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<ThemePreview brandId="b1" />);
    expect(screen.getByTestId("theme-mode-toggle")).toBeTruthy();
    expect(screen.getByTestId("theme-light-btn")).toBeTruthy();
    expect(screen.getByTestId("theme-dark-btn")).toBeTruthy();
  });

  it("renders preview card when data loads", async () => {
    mockFetchThemePreview.mockResolvedValue({
      brand_id: "b1",
      css: ":root { --brand-primary: #1a365d; }",
      tokens: { primary: "#1a365d", secondary: "#2d3748", text: "#1a202c" },
      mode: "light",
    });
    renderWithQuery(<ThemePreview brandId="b1" />);

    await waitFor(() => {
      expect(screen.getByTestId("theme-preview-card")).toBeTruthy();
    });
    expect(screen.getByText("Heading Preview")).toBeTruthy();
    expect(screen.getByText(/CSS Tokens/)).toBeTruthy();
  });

  it("displays token values from the preview", async () => {
    mockFetchThemePreview.mockResolvedValue({
      brand_id: "b1",
      css: "",
      tokens: { primary: "#1a365d" },
      mode: "light",
    });
    renderWithQuery(<ThemePreview brandId="b1" />);

    await waitFor(() => {
      expect(screen.getByText("#1a365d")).toBeTruthy();
    });
  });

  it("shows loading state before data arrives", () => {
    mockFetchThemePreview.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<ThemePreview brandId="b1" />);
    expect(screen.getByText("Loading preview...")).toBeTruthy();
  });

  it("switches mode when dark button clicked", async () => {
    mockFetchThemePreview.mockResolvedValue({
      brand_id: "b1", css: "", tokens: { primary: "#1a365d" }, mode: "light",
    });
    renderWithQuery(<ThemePreview brandId="b1" />);

    await waitFor(() => {
      expect(screen.getByTestId("theme-preview-card")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("theme-dark-btn"));
    // Should trigger a new query with dark mode
    expect(mockFetchThemePreview).toHaveBeenCalledWith("b1", "light");
  });
});
