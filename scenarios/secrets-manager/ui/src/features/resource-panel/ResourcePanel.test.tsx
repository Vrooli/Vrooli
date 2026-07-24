import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ResourcePanel } from "./ResourcePanel";

const detail = {
  resource_name: "vault",
  valid_secrets: 0,
  missing_secrets: 1,
  total_secrets: 1,
  secrets: [{
    id: "vault-token",
    secret_key: "VAULT_TOKEN",
    secret_type: "token",
    description: "Vault token",
    classification: "service",
    required: true,
    owner_team: "platform",
    owner_contact: "platform@example.test",
    tier_strategies: { "tier-2-desktop": "prompt" },
    validation_state: "missing"
  }],
  open_vulnerabilities: [{
    id: "vuln-1",
    component_type: "resource",
    component_name: "vault",
    file_path: "resources/vault",
    line_number: 1,
    severity: "high" as const,
    type: "configuration",
    title: "Missing policy",
    description: "Add a policy",
    recommendation: "Configure policy",
    can_auto_fix: false,
    discovered_at: "2026-07-23T00:00:00Z"
  }]
};

describe("ResourcePanel", () => {
  it("routes secret editing, scenario overrides, vulnerabilities, resource switching, and close actions", () => {
    const onClose = vi.fn();
    const onSwitchResource = vi.fn();
    const onSelectSecret = vi.fn();
    const onUpdateSecret = vi.fn();
    const onApplyStrategy = vi.fn();
    const onDeleteOverride = vi.fn();
    const onUpdateVulnerabilityStatus = vi.fn();
    const onSetStrategyTier = vi.fn();
    const onSetStrategyHandling = vi.fn();
    const onSetStrategyPrompt = vi.fn();
    const onSetStrategyDescription = vi.fn();
    const onSetOverrideReason = vi.fn();
    const onSetIsOverrideMode = vi.fn();
    renderWithProviders(
      <ResourcePanel
        activeResource="vault"
        resourceDetail={detail}
        isLoading={false}
        isFetching
        selectedSecretKey="VAULT_TOKEN"
        strategyTier="tier-2-desktop"
        strategyHandling="prompt"
        strategyPrompt="Vault credential"
        strategyDescription="Enter the Vault credential"
        overrideReason="desktop isolated"
        isOverrideMode
        currentOverride={{ id: "override-1", scenario_name: "secrets-manager", resource_secret_id: "vault-token", resource_name: "vault", secret_key: "VAULT_TOKEN", tier: "tier-2-desktop", handling_strategy: "prompt", override_reason: "desktop isolated", created_at: "now", updated_at: "now" }}
        selectedScenario="secrets-manager"
        tierReadiness={[{ tier: "tier-1-local", label: "Tier 1" }, { tier: "tier-2-desktop", label: "Tier 2" }]}
        allResources={[{ resource_name: "redis", secrets_total: 1, secrets_found: 1, secrets_missing: 0, health_status: "ready" }, { resource_name: "vault", secrets_total: 1, secrets_found: 0, secrets_missing: 1, health_status: "degraded" }]}
        onClose={onClose}
        onSwitchResource={onSwitchResource}
        onSelectSecret={onSelectSecret}
        onUpdateSecret={onUpdateSecret}
        onApplyStrategy={onApplyStrategy}
        onDeleteOverride={onDeleteOverride}
        onUpdateVulnerabilityStatus={onUpdateVulnerabilityStatus}
        onSetStrategyTier={onSetStrategyTier}
        onSetStrategyHandling={onSetStrategyHandling}
        onSetStrategyPrompt={onSetStrategyPrompt}
        onSetStrategyDescription={onSetStrategyDescription}
        onSetOverrideReason={onSetOverrideReason}
        onSetIsOverrideMode={onSetIsOverrideMode}
      />
    );

    expect(screen.getByText("Syncing…")).toBeInTheDocument();
    const [secretListItem] = screen.getAllByText("VAULT_TOKEN");
    if (!secretListItem) throw new Error("Expected Vault secret list item");
    fireEvent.click(secretListItem);
    expect(onSelectSecret).toHaveBeenCalledWith("VAULT_TOKEN");
    const [resourceSelect, classificationSelect, , handlingSelect] = screen.getAllByRole("combobox");
    if (!resourceSelect || !classificationSelect || !handlingSelect) throw new Error("Expected resource strategy controls");
    fireEvent.change(classificationSelect, { target: { value: "user" } });
    expect(onUpdateSecret).toHaveBeenCalledWith("VAULT_TOKEN", { classification: "user" });
    fireEvent.click(screen.getByText("Mark optional"));
    expect(onUpdateSecret).toHaveBeenCalledWith("VAULT_TOKEN", { required: false });
    fireEvent.change(handlingSelect, { target: { value: "generate" } });
    expect(onSetStrategyHandling).toHaveBeenCalledWith("generate");
    fireEvent.click(screen.getByText("Apply scenario override"));
    expect(onApplyStrategy).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText("Remove override"));
    expect(onDeleteOverride).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByText("Resolve"));
    expect(onUpdateVulnerabilityStatus).toHaveBeenCalledWith("vuln-1", "resolved");
    fireEvent.change(resourceSelect, { target: { value: "redis" } });
    expect(onSwitchResource).toHaveBeenCalledWith("redis");
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("distinguishes unloaded, empty, and multiple-issue resources", () => {
    const props = {
      activeResource: "postgres",
      resourceDetail: {
        ...detail,
        resource_name: "postgres",
        valid_secrets: 0,
        missing_secrets: 0,
        total_secrets: 0,
        secrets: [],
        open_vulnerabilities: []
      },
      isLoading: false,
      isFetching: false,
      selectedSecretKey: null,
      strategyTier: "",
      strategyHandling: "",
      strategyPrompt: "",
      strategyDescription: "",
      overrideReason: "",
      isOverrideMode: false,
      tierReadiness: [],
      allResources: [
        { resource_name: "redis", secrets_total: 1, secrets_found: 1, secrets_missing: 0, health_status: "ready" },
        { resource_name: "vault", secrets_total: 1, secrets_found: 0, secrets_missing: 1, health_status: "degraded" },
        { resource_name: "postgres", secrets_total: 2, secrets_found: 0, secrets_missing: 2, health_status: "degraded" }
      ],
      onClose: vi.fn(),
      onSwitchResource: vi.fn(),
      onSelectSecret: vi.fn(),
      onUpdateSecret: vi.fn(),
      onApplyStrategy: vi.fn(),
      onDeleteOverride: vi.fn(),
      onUpdateVulnerabilityStatus: vi.fn(),
      onSetStrategyTier: vi.fn(),
      onSetStrategyHandling: vi.fn(),
      onSetStrategyPrompt: vi.fn(),
      onSetStrategyDescription: vi.fn(),
      onSetOverrideReason: vi.fn(),
      onSetIsOverrideMode: vi.fn()
    };
    const { rerender, getByText, container } = renderWithProviders(<ResourcePanel {...props} />);

    expect(getByText("1 of 2 needing attention")).toBeInTheDocument();
    expect(getByText("0/0 secrets valid · Missing 0")).toBeInTheDocument();
    expect(getByText("No secrets were returned for this resource.")).toBeInTheDocument();
    expect(container).not.toHaveTextContent("Syncing…");

    rerender(<ResourcePanel {...props} resourceDetail={undefined} isLoading />);
    expect(getByText("Loading secrets…")).toBeInTheDocument();
  });
});
