// [REQ:REQ-P0-003] Resource Selection UI
// [REQ:REQ-P2-003] Setup Order UI Suggestions
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithQueryClient, mockFetchPending, mockFetchError } from "../../test-utils";
import { StepSelectResources } from "./StepSelectResources";

const mockResources = [
  { name: "postgres", status: "running", category: "database", installed: true },
  { name: "redis", status: "stopped", category: "database", installed: true },
  { name: "ollama", status: "running", category: "ai", installed: true },
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

  return renderWithQueryClient(<StepSelectResources selected={selected} onToggle={onToggle} />);
}

describe("StepSelectResources", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows loading state initially", () => {
    mockFetchPending();
    renderWithQueryClient(<StepSelectResources selected={new Set()} onToggle={vi.fn()} />);
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
    mockFetchError();
    renderWithQueryClient(<StepSelectResources selected={new Set()} onToggle={vi.fn()} />);

    await waitFor(() => {
      expect(screen.getByTestId("step-resources-error")).toBeInTheDocument();
    });
  });

  it("sets aria-pressed on selected resource cards", async () => {
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    expect(screen.getByTestId("resource-card-postgres")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("resource-card-redis")).toHaveAttribute("aria-pressed", "false");
  });

  it("displays setup order in correct dependency order", async () => {
    renderComponent(new Set(["redis", "postgres", "ollama"]));
    await waitFor(() => {
      expect(screen.getByTestId("setup-order-hint")).toBeInTheDocument();
    });
    // postgres (order 1) → redis (order 2) → ollama (order 3)
    expect(screen.getByTestId("setup-order-hint")).toHaveTextContent("postgres → redis → ollama");
  });

  it("Deselect All calls onToggle for every selected item in category", async () => {
    const onToggle = vi.fn();
    renderComponent(new Set(["postgres", "redis"]), onToggle);
    await waitFor(() => {
      expect(screen.getByTestId("category-toggle-database")).toHaveTextContent("Deselect All");
    });
    fireEvent.click(screen.getByTestId("category-toggle-database"));
    expect(onToggle).toHaveBeenCalledWith("postgres");
    expect(onToggle).toHaveBeenCalledWith("redis");
    expect(onToggle).toHaveBeenCalledTimes(2);
  });

  it("Select Rest only toggles unselected items in category", async () => {
    const onToggle = vi.fn();
    // postgres selected, redis not
    renderComponent(new Set(["postgres"]), onToggle);
    await waitFor(() => {
      expect(screen.getByTestId("category-toggle-database")).toHaveTextContent("Select Rest");
    });
    fireEvent.click(screen.getByTestId("category-toggle-database"));
    // Should only toggle redis (the unselected one)
    expect(onToggle).toHaveBeenCalledWith("redis");
    expect(onToggle).not.toHaveBeenCalledWith("postgres");
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it("resource card includes status badge text", async () => {
    renderComponent(new Set());
    await waitFor(() => {
      expect(screen.getByTestId("resource-card-postgres")).toBeInTheDocument();
    });
    expect(screen.getByTestId("resource-card-postgres")).toHaveTextContent("running");
    expect(screen.getByTestId("resource-card-redis")).toHaveTextContent("stopped");
  });

  it("shows singular 'resource' when exactly one selected", async () => {
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByText(/1 resource selected/i)).toBeInTheDocument();
    });
  });
});
