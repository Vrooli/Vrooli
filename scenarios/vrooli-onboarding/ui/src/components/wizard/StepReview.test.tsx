// [REQ:REQ-P0-005] Config Validation UI
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { vi } from "vitest";
import { StepReview } from "./StepReview";

function renderComponent(selected: Set<string>, onRemove?: (name: string) => void, onGoBack?: () => void) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <StepReview selected={selected} onRemove={onRemove} onGoBack={onGoBack} />
    </QueryClientProvider>
  );
}

describe("StepReview", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders review heading", () => {
    renderComponent(new Set());
    expect(screen.getByText(/review configuration/i)).toBeInTheDocument();
  });

  it("shows selected resource count", () => {
    renderComponent(new Set(["postgres", "redis"]));
    expect(screen.getByText(/selected resources \(2\)/i)).toBeInTheDocument();
  });

  it("shows resource badges for selected resources", () => {
    renderComponent(new Set(["postgres", "redis"]));
    expect(screen.getByText("postgres")).toBeInTheDocument();
    expect(screen.getByText("redis")).toBeInTheDocument();
  });

  it("shows empty state when no resources selected", () => {
    renderComponent(new Set());
    expect(screen.getByText(/no resources selected/i)).toBeInTheDocument();
    expect(screen.getByTestId("review-empty-state")).toBeInTheDocument();
  });

  it("shows go-back button in empty state when onGoBack is provided", () => {
    const onGoBack = vi.fn();
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    render(
      <QueryClientProvider client={queryClient}>
        <StepReview selected={new Set()} onGoBack={onGoBack} />
      </QueryClientProvider>
    );
    const goBackBtn = screen.getByTestId("review-go-back");
    expect(goBackBtn).toBeInTheDocument();
    expect(goBackBtn).toHaveAttribute("aria-label", "Go back to select resources");
    fireEvent.click(goBackBtn);
    expect(onGoBack).toHaveBeenCalledOnce();
  });

  it("shows validation loading state when resources selected", async () => {
    globalThis.fetch = vi.fn().mockImplementation(() =>
      new Promise(() => {})
    );

    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("validation-loading")).toBeInTheDocument();
    });
  });

  it("shows validation success for valid config", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ valid: true, results: [] }),
    });

    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("validation-success")).toBeInTheDocument();
    });
  });

  it("shows validation help tooltip", () => {
    renderComponent(new Set());
    expect(screen.getByTestId("validation-help")).toBeInTheDocument();
    // Tooltip role is present
    expect(screen.getByRole("tooltip")).toHaveTextContent(/checks for dependency conflicts/i);
  });

  it("shows remove buttons on resource chips when onRemove is provided", () => {
    const onRemove = vi.fn();
    renderComponent(new Set(["postgres", "redis"]), onRemove);
    expect(screen.getByTestId("remove-resource-postgres")).toBeInTheDocument();
    expect(screen.getByTestId("remove-resource-redis")).toBeInTheDocument();
    // Remove buttons have accessible labels
    expect(screen.getByRole("button", { name: /remove postgres/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /remove redis/i })).toBeInTheDocument();
  });

  it("calls onRemove when remove button on chip is clicked", () => {
    const onRemove = vi.fn();
    renderComponent(new Set(["postgres", "redis"]), onRemove);
    fireEvent.click(screen.getByTestId("remove-resource-postgres"));
    expect(onRemove).toHaveBeenCalledWith("postgres");
  });

  it("does not show remove buttons when onRemove is not provided", () => {
    renderComponent(new Set(["postgres"]));
    expect(screen.queryByTestId("remove-resource-postgres")).not.toBeInTheDocument();
  });

  it("shows validation error on API failure", async () => {
    globalThis.fetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
    });

    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("validation-error")).toBeInTheDocument();
    });
  });
});
