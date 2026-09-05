import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { ResourceWorkbench } from "./ResourceWorkbench";

const statuses = [
  { resource_name: "vault", secrets_total: 2, secrets_found: 1, secrets_missing: 1, secrets_optional: 0, health_status: "degraded", last_checked: "now" },
  { resource_name: "redis", secrets_total: 1, secrets_found: 1, secrets_missing: 0, secrets_optional: 0, health_status: "healthy", last_checked: "now" }
];

describe("ResourceWorkbench", () => {
  afterEach(cleanup);

  it("shows resource secret strategies and opens selected resources", () => {
    const onOpenResource = vi.fn();
    renderWithProviders(
      <ResourceWorkbench
        resourceInsights={[{ resource_name: "vault", total_secrets: 2, valid_secrets: 1, missing_secrets: 1, secrets: [{ secret_key: "VAULT_TOKEN", secret_type: "string", classification: "service", tier_strategies: { "tier-1": "prompt" } }] }]}
        resourceStatuses={statuses}
        isLoading={false}
        onOpenResource={onOpenResource}
      />
    );
    expect(screen.getByText("VAULT_TOKEN")).toBeInTheDocument();
    expect(screen.getByText("Strategies: tier-1:prompt")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Manage" }));
    expect(onOpenResource).toHaveBeenCalledWith("vault");
    fireEvent.click(screen.getByRole("button", { name: "Show all resources" }));
    expect(screen.getByText("action needed")).toBeInTheDocument();
    expect(screen.getByText("healthy")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /redis/ }));
    expect(onOpenResource).toHaveBeenCalledWith("redis");
    fireEvent.click(screen.getByRole("button", { name: "Help" }));
    expect(screen.getByText("Resource Workbench")).toBeInTheDocument();
  });

  it("handles loading and unavailable resource insights", () => {
    const { rerender } = renderWithProviders(<ResourceWorkbench resourceInsights={[]} resourceStatuses={[]} isLoading onOpenResource={() => {}} />);
    expect(screen.getByText("All resources")).toBeInTheDocument();
    rerender(<ResourceWorkbench resourceInsights={[]} resourceStatuses={[]} isLoading={false} onOpenResource={() => {}} />);
    expect(screen.getByText(/No resource insights available/)).toBeInTheDocument();
    expect(screen.getByText(/0\/0 healthy/)).toBeInTheDocument();
  });
});
