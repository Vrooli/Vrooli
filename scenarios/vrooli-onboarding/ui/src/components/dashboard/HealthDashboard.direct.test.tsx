// [REQ:REQ-P1-002] Health Dashboard Component Tests
import { screen, waitFor, fireEvent } from "../../test-utils";
import { vi } from "vitest";
import { renderWithQueryClient } from "../../test-utils";
import { HealthDashboard } from "./HealthDashboard";

const resourcesApi = vi.hoisted(() => ({ fetchResourceHealth: vi.fn() }));
vi.mock("../../api/resources", () => resourcesApi);

function mockHealthSuccess(body: unknown) { resourcesApi.fetchResourceHealth.mockResolvedValue(body); }
function mockHealthPending() { resourcesApi.fetchResourceHealth.mockImplementation(() => new Promise(() => {})); }

function renderDashboard(props: React.ComponentProps<typeof HealthDashboard> = {}) {
  return renderWithQueryClient(<HealthDashboard {...props} />);
}

const mockHealthData = {
  resources: [
    { name: "postgres", status: "running", category: "database", available: true, last_checked: "2026-01-01T00:00:00Z" },
    { name: "redis", status: "stopped", category: "database", available: false, last_checked: "2026-01-01T00:00:00Z" },
    { name: "ollama", status: "running", category: "ai", available: true, last_checked: "2026-01-01T00:00:00Z" },
  ],
  total: 3,
  healthyCount: 2,
  checked_at: "2026-01-01T00:00:00Z",
};

describe("HealthDashboard", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows loading state initially", () => {
    mockHealthPending();
    renderDashboard();
    expect(screen.getByTestId("health-loading")).toBeInTheDocument();
    expect(screen.getByTestId("health-loading")).toBeInTheDocument();
  });

  it("shows error state on fetch failure", async () => {
    resourcesApi.fetchResourceHealth.mockRejectedValue(new Error("Network error"));
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-error")).toBeInTheDocument();
    });
    expect(screen.getByRole("alert")).toBeInTheDocument();
  });

  it("renders resource health cards after loading", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-card-postgres")).toBeInTheDocument();
    });
    expect(screen.getByTestId("health-card-redis")).toBeInTheDocument();
    expect(screen.getByTestId("health-card-ollama")).toBeInTheDocument();
  });

  it("displays healthy count summary", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-summary")).toBeInTheDocument();
    });
    expect(screen.getByText(/2 of 3/)).toBeInTheDocument();
  });

  it("shows health grid with list role", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-grid")).toBeInTheDocument();
    });
    expect(screen.getByRole("list", { name: /resource health status/i })).toBeInTheDocument();
  });

  it("shows status indicators with descriptive accessible labels", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("status-indicator-postgres")).toBeInTheDocument();
    });
    expect(screen.getByTestId("status-indicator-postgres")).toHaveAttribute("aria-label", "postgres is healthy");
    expect(screen.getByTestId("status-indicator-redis")).toHaveAttribute("aria-label", "redis is unhealthy");
  });

  it("shows empty state when no resources", async () => {
    mockHealthSuccess({ resources: [], total: 0, healthyCount: 0, checkedAt: "" });
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByText(/no resources detected/i)).toBeInTheDocument();
    });
  });

  it("shows auto-refresh indicator", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByText(/auto-refreshes/i)).toBeInTheDocument();
    });
  });

  it("shows reattempt button on error state", async () => {
    resourcesApi.fetchResourceHealth.mockRejectedValue(new Error("Network error"));
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-reattempt")).toBeInTheDocument();
    });
    expect(screen.getByTestId("health-reattempt")).toHaveAttribute("aria-label", "Reattempt loading health data");
  });

  it("shows refresh button when resources loaded", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-refresh")).toBeInTheDocument();
    });
    expect(screen.getByTestId("health-refresh")).toHaveAttribute("aria-label", "Refresh health data");
  });

  it("displays resource categories", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-card-postgres")).toBeInTheDocument();
    });
    expect(screen.getAllByText("database").length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText("ai").length).toBeGreaterThanOrEqual(1);
  });

  it("shows last checked timestamp", async () => {
    mockHealthSuccess(mockHealthData);
    renderDashboard();
    await waitFor(() => {
      expect(screen.getByTestId("health-last-checked")).toBeInTheDocument();
    });
    expect(screen.getByTestId("health-last-checked")).toHaveTextContent(/auto-refreshes/);
  });

  it("shows 'Go to Setup Wizard' button in empty state when callback provided", async () => {
    const onNavigate = vi.fn();
    mockHealthSuccess({ resources: [], total: 0, healthyCount: 0, checkedAt: "2026-01-01T00:00:00Z" });
    renderDashboard({ onNavigateToWizard: onNavigate });
    await waitFor(() => {
      expect(screen.getByTestId("health-go-to-wizard")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("health-go-to-wizard"));
    expect(onNavigate).toHaveBeenCalled();
  });
});
