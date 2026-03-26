import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import BrandDetailPage from "./BrandDetailPage";
import type { Brand, BrandVersion } from "../lib/api";

// [REQ:BM-REQ-UI-DASHBOARD] [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-UI-THEME] [REQ:BM-REQ-UI-APPLY]

const testBrand: Brand = {
  id: "b1",
  name: "Test Brand",
  description: "A test brand for detail page",
  version: 3,
  created_at: "2026-01-15T00:00:00Z",
  updated_at: "2026-02-01T00:00:00Z",
  colors: {
    primary: "#1a365d",
    secondary: "#2d3748",
    accent: "#e53e3e",
    background: "#ffffff",
    surface: "#f7fafc",
    text: "#1a202c",
    error: "#e53e3e",
  },
  identity: {
    display_name: "Test Display",
    tagline: "The best test brand",
  },
  typography: {
    heading_font: "Inter",
    body_font: "Roboto",
    mono_font: "JetBrains Mono",
  },
  voice: {
    tone: "Professional",
    style: "Concise",
    keywords: ["innovation", "quality"],
  },
};

const testVersions: BrandVersion[] = [
  { id: "v3", brand_id: "b1", version: 3, snapshot: "{}", created_at: "2026-02-01T00:00:00Z" },
  { id: "v2", brand_id: "b1", version: 2, snapshot: "{}", created_at: "2026-01-20T00:00:00Z" },
  { id: "v1", brand_id: "b1", version: 1, snapshot: "{}", created_at: "2026-01-15T00:00:00Z" },
];

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchBrand: vi.fn(),
    fetchVersions: vi.fn(),
    deleteBrand: vi.fn(),
    fetchThemePreview: vi.fn(),
    fetchApplyPreview: vi.fn(),
  };
});

import { fetchBrand, fetchVersions, deleteBrand, fetchThemePreview } from "../lib/api";
const mockFetchBrand = vi.mocked(fetchBrand);
const mockFetchVersions = vi.mocked(fetchVersions);
const _mockDeleteBrand = vi.mocked(deleteBrand);
const mockFetchThemePreview = vi.mocked(fetchThemePreview);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("BrandDetailPage", () => {
  const onNavigate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchThemePreview.mockResolvedValue({
      brand_id: "b1", css: "", tokens: { primary: "#1a365d" }, mode: "light",
    });
  });

  // [REQ:BM-REQ-CRUD-READ]
  it("renders loading state initially", () => {
    mockFetchBrand.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);
    expect(screen.getByText("Loading brand...")).toBeTruthy();
  });

  // [REQ:BM-REQ-CRUD-READ] [REQ:BM-REQ-UI-DASHBOARD]
  it("renders brand name and description when loaded", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue(testVersions);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Test Brand")).toBeTruthy();
    });
    expect(screen.getByText("A test brand for detail page")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-DASHBOARD]
  it("renders color swatches when colors exist", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-colors-section")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-UI-DASHBOARD]
  it("renders identity section when identity exists", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-identity-section")).toBeTruthy();
    });
    expect(screen.getByText("Test Display")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-DASHBOARD]
  it("renders typography section when typography exists", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-typography-section")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-UI-DASHBOARD]
  it("renders voice section with keywords", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-voice-section")).toBeTruthy();
    });
    expect(screen.getByText("innovation")).toBeTruthy();
    expect(screen.getByText("quality")).toBeTruthy();
  });

  // [REQ:BM-REQ-CRUD-READ]
  it("renders version history when versions exist", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue(testVersions);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-versions-section")).toBeTruthy();
    });
    expect(screen.getByText("Version 3")).toBeTruthy();
    expect(screen.getByText("Version 1")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-DASHBOARD]
  it("shows edit button that navigates to edit page", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("edit-brand-btn")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("edit-brand-btn"));
    expect(onNavigate).toHaveBeenCalledWith("/brands/b1/edit");
  });

  // [REQ:BM-REQ-UI-DASHBOARD]
  it("shows back to library link", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("back-to-brands")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("back-to-brands"));
    expect(onNavigate).toHaveBeenCalledWith("/brands");
  });

  // [REQ:BM-REQ-CRUD-READ]
  it("shows error state on API failure", async () => {
    mockFetchBrand.mockRejectedValue(new Error("Network error"));
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-detail-error")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-UI-THEME]
  it("renders theme preview section", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("theme-preview-section")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-UI-APPLY]
  it("renders apply preview section", async () => {
    mockFetchBrand.mockResolvedValue(testBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("apply-preview-section")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-UI-DASHBOARD]
  it("hides optional sections when data is absent", async () => {
    const minimalBrand: Brand = {
      id: "b2", name: "Minimal", version: 1,
      created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z",
    };
    mockFetchBrand.mockResolvedValue(minimalBrand);
    mockFetchVersions.mockResolvedValue([]);
    renderWithQuery(<BrandDetailPage brandId="b2" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Minimal")).toBeTruthy();
    });

    expect(screen.queryByTestId("brand-colors-section")).toBeNull();
    expect(screen.queryByTestId("brand-identity-section")).toBeNull();
    expect(screen.queryByTestId("brand-typography-section")).toBeNull();
    expect(screen.queryByTestId("brand-voice-section")).toBeNull();
  });
});
