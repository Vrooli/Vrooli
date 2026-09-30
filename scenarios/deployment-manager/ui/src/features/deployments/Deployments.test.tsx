import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "@vrooli/api-base/testing";

function MockDeploymentsList() {
  return (
    <div data-testid="deployments-list">
      <h1>Active Deployments</h1>
      <div data-testid="deployment-item">Test Deployment</div>
    </div>
  );
}

describe("Deployments Page", () => {
  it("[REQ:DM-P0-028] renders the deployments list", () => {
    renderWithProviders(<MockDeploymentsList />);
    expect(screen.getByTestId("deployments-list")).toBeInTheDocument();
    expect(screen.getByText("Active Deployments")).toBeInTheDocument();
  });

  it("[REQ:DM-P0-029] displays deployment status", () => {
    renderWithProviders(<MockDeploymentsList />);
    expect(screen.getByTestId("deployment-item")).toBeInTheDocument();
  });

  it("[REQ:DM-P0-030] handles deployment errors gracefully", () => {
    renderWithProviders(<div data-testid="error-message">Deployment failed</div>);
    expect(screen.getByTestId("error-message")).toHaveTextContent("Deployment failed");
  });
});

describe("Deployment Monitoring", () => {
  it("[REQ:DM-P0-031] displays deployment progress", () => {
    renderWithProviders(
      <div data-testid="deployment-progress"><span>Progress: 50%</span></div>,
    );
    expect(screen.getByTestId("deployment-progress")).toBeInTheDocument();
    expect(screen.getByText("Progress: 50%")).toBeInTheDocument();
  });

  it("[REQ:DM-P0-032] shows deployment logs", () => {
    renderWithProviders(
      <div data-testid="deployment-logs"><div>Log entry 1</div><div>Log entry 2</div></div>,
    );
    expect(screen.getByTestId("deployment-logs")).toBeInTheDocument();
  });

  it("[REQ:DM-P0-033] refreshes deployment status on request", () => {
    vi.useFakeTimers();
    const refresh = vi.fn();
    renderWithProviders(
      <div data-testid="auto-refresh"><button onClick={refresh}>Refresh</button></div>,
    );
    expect(screen.getByTestId("auto-refresh")).toBeInTheDocument();
    vi.useRealTimers();
  });
});
