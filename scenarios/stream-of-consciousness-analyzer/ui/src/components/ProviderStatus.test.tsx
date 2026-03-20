import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";

const mockListProviders = vi.fn();

vi.mock("../lib/api", () => ({
  listProviders: (...args: unknown[]) => mockListProviders(...args) as unknown,
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

import { ProviderStatus } from "./ProviderStatus";

function renderComponent() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={qc}>
      <ProviderStatus />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockListProviders.mockReset();
});

// [REQ:P2-001] Provider resilience status display
describe("ProviderStatus", () => {
  it("shows active provider name when one is active", async () => {
    mockListProviders.mockResolvedValue([
      { name: "Ollama", url: "http://localhost:11434", active: true, fallback: false },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Ollama")).toBeInTheDocument();
    });
  });

  it("shows 'No LLM' when no provider is active", async () => {
    mockListProviders.mockResolvedValue([
      { name: "Ollama", url: "http://localhost:11434", active: false, fallback: true },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("No LLM")).toBeInTheDocument();
    });
  });

  it("shows fallback count when multiple providers exist", async () => {
    mockListProviders.mockResolvedValue([
      { name: "Ollama", url: "http://localhost:11434", active: true, fallback: false },
      { name: "OpenRouter", url: "https://openrouter.ai", active: false, fallback: true },
    ]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByText("Ollama")).toBeInTheDocument();
      expect(screen.getByText(/\+1 fallback/)).toBeInTheDocument();
    });
  });

  it("shows error state when API is offline", async () => {
    mockListProviders.mockRejectedValue(new Error("connection refused"));
    renderComponent();
    // Component has retry: 2, so allow time for retries with exponential backoff
    await waitFor(
      () => {
        expect(screen.getByText("API offline")).toBeInTheDocument();
      },
      { timeout: 10000 },
    );
  });

  it("renders provider-status test ID", async () => {
    mockListProviders.mockResolvedValue([]);
    renderComponent();
    await waitFor(() => {
      expect(screen.getByTestId("provider-status")).toBeInTheDocument();
    });
  });
});
