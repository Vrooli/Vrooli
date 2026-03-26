import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import BrandFormPage from "./BrandFormPage";
import type { Brand } from "../lib/api";

// [REQ:BM-REQ-UI-CREATE] [REQ:BM-REQ-CRUD-CREATE] [REQ:BM-REQ-CRUD-UPDATE] [REQ:BM-REQ-UI-GENERATE]

const existingBrand: Brand = {
  id: "b1",
  name: "Existing Brand",
  description: "A brand to edit",
  version: 2,
  created_at: "2026-01-15T00:00:00Z",
  updated_at: "2026-02-01T00:00:00Z",
  colors: { primary: "#ff0000", secondary: "#00ff00" },
  identity: { display_name: "My Brand" },
  typography: { heading_font: "Inter" },
  voice: { tone: "Friendly", style: "Casual", keywords: ["fun", "modern"] },
};

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchBrand: vi.fn(),
    createBrand: vi.fn(),
    updateBrand: vi.fn(),
    fetchGenerateOptions: vi.fn(),
  };
});

import { fetchBrand, createBrand, updateBrand, fetchGenerateOptions } from "../lib/api";
const mockFetchBrand = vi.mocked(fetchBrand);
const mockCreateBrand = vi.mocked(createBrand);
const mockUpdateBrand = vi.mocked(updateBrand);
const mockFetchGenerateOptions = vi.mocked(fetchGenerateOptions);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("BrandFormPage", () => {
  const onNavigate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchGenerateOptions.mockResolvedValue({
      providers: [], elements: ["colors", "typography"],
    });
  });

  // [REQ:BM-REQ-UI-CREATE] [REQ:BM-REQ-CRUD-CREATE]
  it("renders create mode when no brandId", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    expect(screen.getByRole("heading", { name: "Create Brand" })).toBeTruthy();
    expect(screen.getByTestId("brand-form-page")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-CREATE]
  it("renders name input field", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    expect(screen.getByTestId("brand-name-input")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-CREATE]
  it("renders description input field", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    expect(screen.getByTestId("brand-description-input")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-CREATE]
  it("renders color picker inputs", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    expect(screen.getByTestId("color-picker-primary")).toBeTruthy();
    expect(screen.getByTestId("color-picker-secondary")).toBeTruthy();
    expect(screen.getByTestId("color-picker-accent")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-CREATE]
  it("renders typography inputs", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    expect(screen.getByTestId("brand-heading-font-input")).toBeTruthy();
    expect(screen.getByTestId("brand-body-font-input")).toBeTruthy();
    expect(screen.getByTestId("brand-mono-font-input")).toBeTruthy();
  });

  // [REQ:BM-REQ-UI-CREATE]
  it("renders voice inputs", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    expect(screen.getByTestId("brand-tone-input")).toBeTruthy();
    expect(screen.getByTestId("brand-style-input")).toBeTruthy();
    expect(screen.getByTestId("brand-keywords-input")).toBeTruthy();
  });

  // [REQ:BM-REQ-CRUD-CREATE]
  it("shows validation error when name is empty", async () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);

    fireEvent.click(screen.getByTestId("save-brand-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("form-error")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-CRUD-CREATE]
  it("calls createBrand on submit in create mode", async () => {
    const created: Brand = {
      id: "new-1", name: "New Brand", version: 1,
      created_at: "2026-03-01T00:00:00Z", updated_at: "2026-03-01T00:00:00Z",
    };
    mockCreateBrand.mockResolvedValue(created);
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);

    fireEvent.change(screen.getByTestId("brand-name-input"), { target: { value: "New Brand" } });
    fireEvent.click(screen.getByTestId("save-brand-btn"));

    await waitFor(() => {
      expect(mockCreateBrand).toHaveBeenCalled();
    });
    await waitFor(() => {
      expect(onNavigate).toHaveBeenCalledWith("/brands/new-1");
    });
  });

  // [REQ:BM-REQ-CRUD-UPDATE]
  it("renders edit mode when brandId is provided", async () => {
    mockFetchBrand.mockResolvedValue(existingBrand);
    renderWithQuery(<BrandFormPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Edit Brand")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-CRUD-UPDATE]
  it("populates form with existing brand data", async () => {
    mockFetchBrand.mockResolvedValue(existingBrand);
    renderWithQuery(<BrandFormPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      const nameInput = screen.getByTestId("brand-name-input");
      expect(nameInput.getAttribute("value")).toBe("Existing Brand");
    });
  });

  // [REQ:BM-REQ-CRUD-UPDATE]
  it("calls updateBrand on submit in edit mode", async () => {
    mockFetchBrand.mockResolvedValue(existingBrand);
    mockUpdateBrand.mockResolvedValue({ ...existingBrand, version: 3 });
    renderWithQuery(<BrandFormPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("brand-name-input").getAttribute("value")).toBe("Existing Brand");
    });

    fireEvent.click(screen.getByTestId("save-brand-btn"));

    await waitFor(() => {
      expect(mockUpdateBrand).toHaveBeenCalledWith("b1", expect.any(Object));
    });
  });

  // [REQ:BM-REQ-UI-CREATE]
  it("back button navigates to library in create mode", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    fireEvent.click(screen.getByTestId("back-from-form"));
    expect(onNavigate).toHaveBeenCalledWith("/brands");
  });

  // [REQ:BM-REQ-CRUD-UPDATE]
  it("back button navigates to brand detail in edit mode", async () => {
    mockFetchBrand.mockResolvedValue(existingBrand);
    renderWithQuery(<BrandFormPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Edit Brand")).toBeTruthy();
    });

    fireEvent.click(screen.getByTestId("back-from-form"));
    expect(onNavigate).toHaveBeenCalledWith("/brands/b1");
  });

  // [REQ:BM-REQ-UI-GENERATE]
  it("renders generate options section in create mode", async () => {
    mockFetchGenerateOptions.mockResolvedValue({
      providers: [{ id: "ollama", name: "Ollama", description: "Local LLM", available: true, capabilities: ["text"] }],
      elements: ["colors"],
    });
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByTestId("generate-options-section")).toBeTruthy();
    });
  });

  // [REQ:BM-REQ-UI-GENERATE]
  it("hides generate options in edit mode", async () => {
    mockFetchBrand.mockResolvedValue(existingBrand);
    renderWithQuery(<BrandFormPage brandId="b1" onNavigate={onNavigate} />);

    await waitFor(() => {
      expect(screen.getByText("Edit Brand")).toBeTruthy();
    });
    expect(screen.queryByTestId("generate-options-section")).toBeNull();
  });

  // [REQ:BM-REQ-UI-CREATE]
  it("renders save button with correct label", () => {
    renderWithQuery(<BrandFormPage onNavigate={onNavigate} />);
    expect(screen.getByTestId("save-brand-btn").textContent).toContain("Create Brand");
  });
});
