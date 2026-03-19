// [REQ:REQ-P0-005] Config Validation UI
import { screen, waitFor, fireEvent } from "@testing-library/react";
import { vi } from "vitest";
import { renderWithQueryClient, mockFetchSuccess, mockFetchError, mockFetchPending } from "../../test-utils";
import { StepReview } from "./StepReview";

function renderComponent(selected: Set<string>, onRemove?: (name: string) => void, onGoBack?: () => void) {
  return renderWithQueryClient(<StepReview selected={selected} onRemove={onRemove} onGoBack={onGoBack} />);
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
    renderWithQueryClient(<StepReview selected={new Set()} onGoBack={onGoBack} />);
    const goBackBtn = screen.getByTestId("review-go-back");
    expect(goBackBtn).toBeInTheDocument();
    expect(goBackBtn).toHaveAttribute("aria-label", "Go back to select resources");
    fireEvent.click(goBackBtn);
    expect(onGoBack).toHaveBeenCalledOnce();
  });

  it("shows validation loading state when resources selected", async () => {
    mockFetchPending();
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("validation-loading")).toBeInTheDocument();
    });
  });

  it("shows validation success for valid config", async () => {
    mockFetchSuccess({ valid: true, results: [] });
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
    mockFetchError();
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("validation-error")).toBeInTheDocument();
    });
  });
});
