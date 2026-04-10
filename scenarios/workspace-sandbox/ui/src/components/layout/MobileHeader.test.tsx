import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MobileHeader } from "./MobileHeader";
import type { HealthResponse, SandboxStats } from "../../lib/api";

const mockHealth: HealthResponse = {
  status: "healthy",
  service: "Workspace Sandbox API",
  version: "1.0.0",
  readiness: true,
  timestamp: new Date().toISOString(),
  dependencies: { database: "connected", driver: "available" },
};

const mockStats: SandboxStats = {
  total: 10,
  active: 5,
  stopped: 2,
  approved: 1,
  rejected: 1,
  error: 1,
  totalSizeBytes: 1024 * 1024,
};

const defaultProps = {
  health: mockHealth,
  stats: mockStats,
  isLoading: false,
  onRefresh: vi.fn(),
  onCreateClick: vi.fn(),
  onSettingsClick: vi.fn(),
  onCommitClick: vi.fn(),
};

describe("MobileHeader", () => {
  it("renders with data-testid", () => {
    render(<MobileHeader {...defaultProps} />);
    expect(screen.getByTestId("mobile-header")).toBeInTheDocument();
  });

  it("renders refresh button", () => {
    render(<MobileHeader {...defaultProps} />);
    expect(screen.getByTestId("refresh-button")).toBeInTheDocument();
  });

  it("renders create button", () => {
    render(<MobileHeader {...defaultProps} />);
    expect(screen.getByTestId("create-sandbox-button")).toBeInTheDocument();
  });

  it("calls onCreateClick when create button is clicked", async () => {
    const user = userEvent.setup();
    const onCreateClick = vi.fn();
    render(<MobileHeader {...defaultProps} onCreateClick={onCreateClick} />);

    await user.click(screen.getByTestId("create-sandbox-button"));
    expect(onCreateClick).toHaveBeenCalledOnce();
  });

  it("calls onRefresh when refresh button is clicked", async () => {
    const user = userEvent.setup();
    const onRefresh = vi.fn();
    render(<MobileHeader {...defaultProps} onRefresh={onRefresh} />);

    await user.click(screen.getByTestId("refresh-button"));
    expect(onRefresh).toHaveBeenCalledOnce();
  });

  it("renders the more options button", () => {
    render(<MobileHeader {...defaultProps} />);
    expect(screen.getByTestId("mobile-header-more")).toBeInTheDocument();
  });

  it("opens bottom sheet when more button is clicked", async () => {
    const user = userEvent.setup();
    render(<MobileHeader {...defaultProps} />);

    await user.click(screen.getByTestId("mobile-header-more"));
    expect(screen.getByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("Commit Pending Changes")).toBeInTheDocument();
  });
});
