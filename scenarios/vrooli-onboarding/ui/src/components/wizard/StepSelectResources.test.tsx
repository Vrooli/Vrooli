// [REQ:REQ-P0-003] Resource Selection UI
// [REQ:REQ-P2-003] Setup Order UI Suggestions
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi } from "vitest";
import { StepSelectResources } from "./StepSelectResources";

const mockResources = [
  { name: "postgres", status: "running", category: "database", installed: "true", last_updated: "2026-01-01" },
  { name: "redis", status: "stopped", category: "database", installed: "true", last_updated: "2026-01-01" },
  { name: "ollama", status: "running", category: "ai", installed: "true", last_updated: "2026-01-01" },
];

const mockSetupOrder = {
  setup_order: [
    { name: "postgres", category: "database", order: 1, dependencies: [] },
    { name: "redis", category: "database", order: 2, dependencies: [] },
    { name: "ollama", category: "ai", order: 3, dependencies: [] },
  ],
  total: 3,
};

function renderComponent(selected: Set<string>, onToggle = vi.fn()) {
  globalThis.fetch = vi.fn().mockImplementation((url: string) => {
    if (typeof url === "string" && url.includes("/setup-order")) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockSetupOrder),
      });
    }
    if (typeof url === "string" && url.includes("/resources")) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockResources),
      });
    }
    return Promise.resolve({ ok: false, status: 404 });
  });

  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <StepSelectResources selected={selected} onToggle={onToggle} />
    </QueryClientProvider>
  );
}

describe("StepSelectResources", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows loading state initially", () => {
    globalThis.fetch = vi.fn().mockImplementation(() => new Promise(() => {}));
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <StepSelectResources selected={new Set()} onToggle={vi.fn()} />
      </QueryClientProvider>
    );
    expect(screen.getByTestId("step-resources-loading")).toBeInTheDocument();
  });

  it("renders resource cards after loading", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    expect(screen.getByTestId("resource-card-redis")).toBeInTheDocument();
    expect(screen.getByTestId("resource-card-ollama")).toBeInTheDocument();
  });

  it("shows selected count", async () => {
    renderComponent(new Set(["postgres", "redis"]));
    await waitFor(() => {
      expect(screen.getByText(/2 resources selected/i)).toBeInTheDocument();
    });
  });

  it("calls onToggle when resource card clicked", async () => {
    const onToggle = vi.fn();
    renderComponent(new Set(), onToggle);
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("resource-card-postgres"));
    expect(onToggle).toHaveBeenCalledWith("postgres");
  });

  it("groups resources by category", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByText("database")).toBeInTheDocument();
    });
    expect(screen.getByText("ai")).toBeInTheDocument();
  });

  it("shows setup order hint when resources are selected", async () => {
    renderComponent(new Set(["postgres", "redis"]));
    await waitFor(() => {
      expect(screen.getByTestId("setup-order-hint")).toBeInTheDocument();
    });
  });

  it("hides setup order hint when no resources selected", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("setup-order-hint")).not.toBeInTheDocument();
  });

  it("renders search/filter input", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    expect(screen.getByTestId("resource-search")).toBeInTheDocument();
  });

  it("filters resources by name when searching", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByTestId("resource-search"), { target: { value: "post" } });
    expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    expect(screen.queryByTestId("resource-card-redis")).not.toBeInTheDocument();
    expect(screen.queryByTestId("resource-card-ollama")).not.toBeInTheDocument();
  });

  it("filters resources by category when searching", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByTestId("resource-search"), { target: { value: "ai" } });
    expect(screen.getByTestId("resource-card-ollama")).toBeInTheDocument();
    expect(screen.queryByTestId("resource-card-postgres")).not.toBeInTheDocument();
  });

  it("shows filter count when searching", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByTestId("resource-search"), { target: { value: "data" } });
    expect(screen.getByTestId("resource-filter-count")).toHaveTextContent("2 of 3 shown");
  });

  it("shows no results state when search matches nothing", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    fireEvent.change(screen.getByTestId("resource-search"), { target: { value: "zzzzz" } });
    expect(screen.getByTestId("resource-no-results")).toBeInTheDocument();
  });

  it("shows Select All button per category", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("category-toggle-database")).toBeInTheDocument();
    });
    expect(screen.getByTestId("category-toggle-ai")).toBeInTheDocument();
    expect(screen.getByTestId("category-toggle-database")).toHaveTextContent("Select All");
  });

  it("calls onToggle for all items in category when Select All is clicked", async () => {
    const onToggle = vi.fn();
    renderComponent(new Set(), onToggle);
    await waitFor(() => {
      expect(screen.getByTestId("category-toggle-database")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("category-toggle-database"));
    expect(onToggle).toHaveBeenCalledWith("postgres");
    expect(onToggle).toHaveBeenCalledWith("redis");
    expect(onToggle).toHaveBeenCalledTimes(2);
  });

  it("shows Deselect All when all items in category are selected", async () => {
    renderComponent(new Set(["postgres", "redis"]));
    await waitFor(() => {
      expect(screen.getByTestId("category-toggle-database")).toBeInTheDocument();
    });
    expect(screen.getByTestId("category-toggle-database")).toHaveTextContent("Deselect All");
  });

  it("shows Select Rest when some items in category are selected", async () => {
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("category-toggle-database")).toBeInTheDocument();
    });
    expect(screen.getByTestId("category-toggle-database")).toHaveTextContent("Select Rest");
  });

  it("shows error state on API failure", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
    });

    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <StepSelectResources selected={new Set()} onToggle={vi.fn()} />
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId("step-resources-error")).toBeInTheDocument();
    });
  });
});
