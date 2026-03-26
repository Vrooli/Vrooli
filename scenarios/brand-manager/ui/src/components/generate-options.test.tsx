import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { GenerateOptions } from "./generate-options";

// [REQ:BM-REQ-UI-GENERATE]

vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchGenerateOptions: vi.fn(),
  };
});

import { fetchGenerateOptions } from "../lib/api";
const mockFetchGenerateOptions = vi.mocked(fetchGenerateOptions);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>,
  );
}

describe("GenerateOptions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchGenerateOptions.mockReturnValue(new Promise(() => {}));
    renderWithQuery(<GenerateOptions />);
    expect(screen.getByText("Loading options...")).toBeTruthy();
  });

  it("renders available providers", async () => {
    mockFetchGenerateOptions.mockResolvedValue({
      providers: [
        {
          id: "ollama", name: "Ollama", description: "Local LLM inference",
          available: true, capabilities: ["text", "color_palette"],
        },
        {
          id: "openrouter", name: "OpenRouter", description: "Cloud LLM routing",
          available: false, capabilities: ["text", "image"], requires: "OPENROUTER_API_KEY",
        },
      ],
      elements: ["colors", "typography", "voice"],
    });

    renderWithQuery(<GenerateOptions />);

    await waitFor(() => {
      expect(screen.getByTestId("generate-options-section")).toBeTruthy();
    });

    expect(screen.getByTestId("provider-ollama")).toBeTruthy();
    expect(screen.getByTestId("provider-openrouter")).toBeTruthy();
    expect(screen.getByText("Ollama")).toBeTruthy();
    expect(screen.getByText("OpenRouter")).toBeTruthy();
  });

  it("shows availability status for providers", async () => {
    mockFetchGenerateOptions.mockResolvedValue({
      providers: [
        { id: "ollama", name: "Ollama", description: "Local", available: true, capabilities: ["text"] },
        { id: "openrouter", name: "OpenRouter", description: "Cloud", available: false, capabilities: ["text"], requires: "API_KEY" },
      ],
      elements: ["colors"],
    });

    renderWithQuery(<GenerateOptions />);

    await waitFor(() => {
      expect(screen.getByText("Available")).toBeTruthy();
      expect(screen.getByText("Not configured")).toBeTruthy();
    });
  });

  it("displays capability badges", async () => {
    mockFetchGenerateOptions.mockResolvedValue({
      providers: [
        { id: "p1", name: "Provider", description: "Test", available: true, capabilities: ["text", "image"] },
      ],
      elements: [],
    });

    renderWithQuery(<GenerateOptions />);

    await waitFor(() => {
      expect(screen.getByText("text")).toBeTruthy();
      expect(screen.getByText("image")).toBeTruthy();
    });
  });

  it("displays available elements list", async () => {
    mockFetchGenerateOptions.mockResolvedValue({
      providers: [],
      elements: ["colors", "typography", "voice"],
    });

    renderWithQuery(<GenerateOptions />);

    await waitFor(() => {
      expect(screen.getByText(/colors, typography, voice/)).toBeTruthy();
    });
  });

  it("shows requirements hint for unavailable providers", async () => {
    mockFetchGenerateOptions.mockResolvedValue({
      providers: [
        { id: "p1", name: "Provider", description: "Test", available: false, capabilities: [], requires: "SOME_API_KEY" },
      ],
      elements: [],
    });

    renderWithQuery(<GenerateOptions />);

    await waitFor(() => {
      expect(screen.getByText(/Requires: SOME_API_KEY/)).toBeTruthy();
    });
  });

  it("renders nothing when no options data", async () => {
    mockFetchGenerateOptions.mockResolvedValue(undefined as never);
    const { container } = renderWithQuery(<GenerateOptions />);

    // Wait for loading to finish
    await waitFor(() => {
      expect(screen.queryByText("Loading options...")).toBeNull();
    });
    // Should render null (empty)
    expect(container.querySelector("[data-testid='generate-options-section']")).toBeNull();
  });
});
