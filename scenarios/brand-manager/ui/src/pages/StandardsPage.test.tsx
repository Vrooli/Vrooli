import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import StandardsPage from "./StandardsPage";

// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-RULES] [REQ:BM-REQ-AUDIT-ENDPOINT]

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchStandards: vi.fn(),
  };
});

import { fetchStandards } from "../lib/api";
const mockFetchStandards = vi.mocked(fetchStandards);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("StandardsPage", () => {
  const onNavigate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders page with title", () => {
    mockFetchStandards.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    expect(screen.getByTestId("standards-page")).toBeTruthy();
    expect(screen.getByText("Brand Standards")).toBeTruthy();
  });

  it("shows loading state", () => {
    mockFetchStandards.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    expect(screen.getByTestId("standards-loading")).toBeTruthy();
  });

  it("renders standards list when loaded", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "has-logo", name: "Logo Required", description: "Every scenario must have a logo", severity: "error" },
        { id: "has-favicon", name: "Favicon Required", description: "Favicon must be set", severity: "warning" },
        { id: "has-colors", name: "Color System", description: "Color system must be defined", severity: "error" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("standards-list")).toBeTruthy();
    });

    expect(screen.getByText("Logo Required")).toBeTruthy();
    expect(screen.getByText("Favicon Required")).toBeTruthy();
    expect(screen.getByText("Color System")).toBeTruthy();
  });

  it("shows severity badges for each rule", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "r1", name: "Rule 1", description: "Desc", severity: "error" },
        { id: "r2", name: "Rule 2", description: "Desc", severity: "warning" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("error")).toBeTruthy();
      expect(screen.getByText("warning")).toBeTruthy();
    });
  });

  it("shows error state on API failure", async () => {
    mockFetchStandards.mockRejectedValue(new Error("API down"));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("standards-error")).toBeTruthy();
    });
  });

  it("shows empty state when no rules", async () => {
    mockFetchStandards.mockResolvedValue({ rules: [] });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("standards-empty")).toBeTruthy();
    });
  });

  it("navigates back to brands", () => {
    mockFetchStandards.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    fireEvent.click(screen.getByTestId("back-to-brands"));
    expect(onNavigate).toHaveBeenCalledWith("/brands");
  });

  it("renders rule descriptions", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "r1", name: "Logo", description: "Must have a logo file", severity: "error" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Must have a logo file")).toBeTruthy();
    });
  });

  it("renders rule IDs", async () => {
    mockFetchStandards.mockResolvedValue({
      rules: [
        { id: "has-logo", name: "Logo", description: "Desc", severity: "error" },
      ],
    });
    renderWithQuery(<StandardsPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("ID: has-logo")).toBeTruthy();
    });
  });
});
