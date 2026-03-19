// [REQ:REQ-P2-004] Glossary Panel Component Tests
import { screen, waitFor, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithQueryClient, mockFetchSuccess, mockFetchPending } from "../../test-utils";
import { GlossaryPanel } from "./GlossaryPanel";

const mockGlossaryData = {
  entries: [
    { term: "Resource", description: "A local service like a database or AI model.", category: "core" },
    { term: "Scenario", description: "A full application built from resources.", category: "core" },
    { term: "Ollama", description: "Local AI model runner.", category: "ai" },
  ],
  count: 3,
};

describe("GlossaryPanel", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders glossary panel container", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    expect(screen.getByTestId("glossary-panel")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    mockFetchPending();
    renderWithQueryClient(<GlossaryPanel />);
    expect(screen.getByTestId("glossary-loading")).toBeInTheDocument();
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("renders glossary entries after loading", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId("glossary-list")).toBeInTheDocument();
    });
    expect(screen.getByText("Resource")).toBeInTheDocument();
    expect(screen.getByText("Scenario")).toBeInTheDocument();
    expect(screen.getByText("Ollama")).toBeInTheDocument();
  });

  it("displays term descriptions", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    await waitFor(() => {
      expect(screen.getByText(/local service like a database/i)).toBeInTheDocument();
    });
  });

  it("renders search input with accessible label", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    const searchInput = screen.getByTestId("glossary-search");
    expect(searchInput).toBeInTheDocument();
    expect(searchInput).toHaveAttribute("aria-label", "Search glossary terms");
    expect(searchInput).toHaveAttribute("type", "search");
  });

  it("shows empty state when no entries match", async () => {
    mockFetchSuccess({ entries: [], count: 0, query: "xyz" });
    renderWithQueryClient(<GlossaryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId("glossary-empty")).toBeInTheDocument();
    });
    expect(screen.getByText(/no matching terms found/i)).toBeInTheDocument();
  });

  it("shows search hint in empty state when search term present", async () => {
    mockFetchSuccess({ entries: [], count: 0 });
    renderWithQueryClient(<GlossaryPanel />);
    // Type something to set search term
    const searchInput = screen.getByTestId("glossary-search");
    fireEvent.change(searchInput, { target: { value: "nonexistent" } });
    await waitFor(() => {
      expect(screen.getByText(/try a different search term/i)).toBeInTheDocument();
    });
  });

  it("uses dl element for glossary list (proper semantics)", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId("glossary-list")).toBeInTheDocument();
    });
    expect(screen.getByTestId("glossary-list").tagName).toBe("DL");
  });

  it("renders category badges for entries", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    await waitFor(() => {
      expect(screen.getAllByText("core")).toHaveLength(2);
    });
    expect(screen.getByText("ai")).toBeInTheDocument();
  });

  it("renders heading with proper hierarchy", () => {
    mockFetchPending();
    renderWithQueryClient(<GlossaryPanel />);
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Glossary");
  });

  it("shows result count after loading entries", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId("glossary-count")).toBeInTheDocument();
    });
    expect(screen.getByText(/3 terms/)).toBeInTheDocument();
  });

  it("shows clear search button in input when search term is present", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    await waitFor(() => {
      expect(screen.getByTestId("glossary-list")).toBeInTheDocument();
    });
    // Clear search button should not exist initially
    expect(screen.queryByTestId("glossary-clear-search")).not.toBeInTheDocument();
    // Type something
    const searchInput = screen.getByTestId("glossary-search");
    fireEvent.change(searchInput, { target: { value: "test" } });
    // Wait for debounce to complete
    await waitFor(() => {
      expect(screen.getByTestId("glossary-clear-search")).toBeInTheDocument();
    });
  });

  it("shows clickable clear search link in empty state", async () => {
    mockFetchSuccess({ entries: [], count: 0 });
    renderWithQueryClient(<GlossaryPanel />);
    const searchInput = screen.getByTestId("glossary-search");
    fireEvent.change(searchInput, { target: { value: "nonexistent" } });
    await waitFor(() => {
      expect(screen.getByText(/clear the search/i)).toBeInTheDocument();
    });
    // The clear search text should be a button
    const clearButton = screen.getByRole("button", { name: /clear the search/i });
    expect(clearButton).toBeInTheDocument();
  });

  it("shows debounce indicator while waiting for search", async () => {
    mockFetchSuccess(mockGlossaryData);
    renderWithQueryClient(<GlossaryPanel />);
    // Wait for initial load
    await waitFor(() => {
      expect(screen.getByTestId("glossary-list")).toBeInTheDocument();
    });
    // Type to trigger debounce
    const searchInput = screen.getByTestId("glossary-search");
    fireEvent.change(searchInput, { target: { value: "test" } });
    // Debounce indicator should appear briefly
    expect(screen.getByTestId("glossary-debounce-indicator")).toBeInTheDocument();
  });
});
