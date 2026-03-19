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

  it("shows validation error on API failure with error message", async () => {
    mockFetchError();
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("validation-error")).toBeInTheDocument();
    });
    // Error state should use alert role for screen readers
    expect(screen.getByRole("alert")).toBeInTheDocument();
    // Should display a meaningful error message (from formatQueryError)
    expect(screen.getByTestId("validation-error")).toHaveTextContent(/failed|error/i);
  });

  it("shows invalid config state with error icon", async () => {
    mockFetchSuccess({ valid: false, errors: [], warnings: [] });
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByTestId("validation-invalid")).toBeInTheDocument();
    });
    expect(screen.getByText(/configuration has issues/i)).toBeInTheDocument();
  });

  it("displays validation error messages", async () => {
    mockFetchSuccess({
      valid: false,
      errors: ["Port 5432 is already in use", "Missing required dependency: libpq"],
      warnings: [],
    });
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByText("Port 5432 is already in use")).toBeInTheDocument();
    });
    expect(screen.getByText("Missing required dependency: libpq")).toBeInTheDocument();
  });

  it("displays validation warning messages", async () => {
    mockFetchSuccess({
      valid: true,
      errors: [],
      warnings: ["Redis is running an older version", "Consider enabling TLS"],
    });
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByText("Redis is running an older version")).toBeInTheDocument();
    });
    expect(screen.getByText("Consider enabling TLS")).toBeInTheDocument();
  });

  it("displays both errors and warnings simultaneously", async () => {
    mockFetchSuccess({
      valid: false,
      errors: ["Critical: port conflict"],
      warnings: ["Recommended: increase memory"],
    });
    renderComponent(new Set(["postgres"]));
    await waitFor(() => {
      expect(screen.getByText("Critical: port conflict")).toBeInTheDocument();
    });
    expect(screen.getByText("Recommended: increase memory")).toBeInTheDocument();
    expect(screen.getByTestId("validation-invalid")).toBeInTheDocument();
  });

  it("shows select resources prompt when no resources and no loading/error", () => {
    renderComponent(new Set());
    expect(screen.getByText(/select resources to validate/i)).toBeInTheDocument();
  });
});
