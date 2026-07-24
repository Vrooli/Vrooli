import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../test-utils";
import { CampaignsPanel } from "./CampaignsPanel";
import { DeploymentStepper } from "./DeploymentStepper";
import { DeploymentReadinessPanel, TierReadiness } from "./TierReadiness";
import { OrientationHub } from "./OrientationHub";

const campaigns = [
  {
    id: "gateway",
    scenario: "api-gateway",
    tier: "tier-4-saas",
    status: "ready",
    progress: 100,
    blockers: 0,
    updated_at: "2026-07-23T00:00:00Z",
    summary: { strategized_secrets: 4, total_secrets: 4, requires_action: 0 }
  },
  {
    id: "secrets",
    scenario: "secrets-manager",
    tier: "tier-2-desktop",
    status: "blocked",
    progress: 30,
    blockers: 2,
    updated_at: "2026-07-23T00:00:00Z",
    summary: { strategized_secrets: 1, total_secrets: 3, requires_action: 2 }
  }
];

afterEach(cleanup);

describe("CampaignsPanel", () => {
  it("summarizes readiness, supports search/sort, and opens the selected campaign", () => {
    const onSearchChange = vi.fn();
    const onSelectScenario = vi.fn();
    renderWithProviders(
      <CampaignsPanel
        campaigns={campaigns}
        isLoading={false}
        search=""
        onSearchChange={onSearchChange}
        selectedScenario="secrets-manager"
        onSelectScenario={onSelectScenario}
        defaultBlockedTiers={3}
      />
    );

    expect(screen.getByText("5/7")).toBeInTheDocument();
    expect(screen.getAllByText("2 tiers blocked").length).toBeGreaterThan(0);
    expect(screen.getByText("Ready")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Search scenarios"), { target: { value: "gateway" } });
    expect(onSearchChange).toHaveBeenCalledWith("gateway");

    fireEvent.click(screen.getByText("Sort: A→Z"));
    expect(screen.getByText("Sort: Z→A")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Open"));
    expect(onSelectScenario).toHaveBeenCalledWith("api-gateway");
  });

  it("renders loading, empty, and collapsed campaign states", () => {
    const { rerender } = renderWithProviders(
      <CampaignsPanel
        campaigns={[]}
        isLoading
        search="vault"
        onSearchChange={() => {}}
        onSelectScenario={() => {}}
        defaultBlockedTiers={0}
      />
    );
    expect(screen.getByText("Loading campaigns...")).toBeInTheDocument();

    rerender(
      <CampaignsPanel
        campaigns={[]}
        isLoading={false}
        search="vault"
        onSearchChange={() => {}}
        onSelectScenario={() => {}}
        defaultBlockedTiers={0}
        isCollapsed
      />
    );
    expect(screen.queryByText('No campaigns match "vault".')).not.toBeInTheDocument();
    expect(screen.getByText("0 campaigns")).toBeInTheDocument();
  });

  it("uses fallback blockers and next actions when campaign summaries are unavailable", () => {
    const onSelectScenario = vi.fn();
    const { rerender } = renderWithProviders(
      <CampaignsPanel
        campaigns={[
          { id: "fallback", scenario: "backup", tier: "tier-1-local", status: "blocked", progress: 0, blockers: 0, updated_at: "now" },
          { id: "ready", scenario: "catalog", tier: "tier-4-saas", status: "ready", progress: 100, blockers: 0, updated_at: "now", next_action: "Review handoff" },
          { id: "single", scenario: "gateway", tier: "tier-2-desktop", status: "blocked", progress: 50, blockers: 1, updated_at: "now" }
        ]}
        isLoading={false}
        search=""
        onSearchChange={() => {}}
        selectedScenario="catalog"
        onSelectScenario={onSelectScenario}
        defaultBlockedTiers={2}
      />
    );
    expect(screen.getAllByText("2 tiers blocked").length).toBeGreaterThan(0);
    expect(screen.getByText("1 tier blocked")).toBeInTheDocument();
    expect(screen.getAllByText("—").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Define strategies for blocked tiers").length).toBeGreaterThan(0);
    expect(screen.getByText("Review handoff")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Selected" })).toBeInTheDocument();
    const [firstOpen] = screen.getAllByRole("button", { name: "Open" });
    if (!firstOpen) throw new Error("Expected an open campaign action");
    fireEvent.click(firstOpen);
    expect(onSelectScenario).toHaveBeenCalledWith("backup");

    rerender(
      <CampaignsPanel
        campaigns={[{ id: "fallback", scenario: "backup", tier: "tier-1-local", status: "blocked", progress: 0, blockers: 0, updated_at: "now" }]}
        isLoading={false}
        search=""
        onSearchChange={() => {}}
        selectedScenario="backup"
        onSelectScenario={() => {}}
        defaultBlockedTiers={2}
        isCollapsed
        onToggleCollapse={() => {}}
      />
    );
    expect(screen.getByText("Selected: backup")).toBeInTheDocument();
    expect(screen.getByText("2 blockers")).toBeInTheDocument();
  });
});

describe("DeploymentStepper", () => {
  it("shows manifest readiness and lets operators jump directly between deployment steps", () => {
    const onStepChange = vi.fn();
    renderWithProviders(<DeploymentStepper activeStep={2} hasManifest onStepChange={onStepChange} />);

    expect(screen.getByText("Manifest generated")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Export manifest"));
    expect(onStepChange).toHaveBeenCalledWith(3);
    fireEvent.click(screen.getByText("Manifest primer"));
    expect(onStepChange).toHaveBeenCalledWith(0);
  });
});

describe("OrientationHub", () => {
  it("shows the active orientation step and routes journey navigation controls", () => {
    const onJourneySelect = vi.fn();
    const onJourneyExit = vi.fn();
    const onJourneyNext = vi.fn();
    const onJourneyBack = vi.fn();
    renderWithProviders(
      <OrientationHub
        journeyCards={[{ id: "orientation", badge: "Start", title: "Orientation", description: "Overview", primers: ["5 min"] }]}
        activeJourneyCard={{ id: "orientation", badge: "Start", title: "Orientation", description: "Overview", primers: ["5 min"] }}
        activeJourney="orientation"
        journeySteps={[{ title: "Welcome", description: "Review posture", content: <p>Posture content</p> }]}
        journeyStep={0}
        updatedAt="2026-07-23T00:00:00Z"
        isLoading={false}
        onJourneySelect={onJourneySelect}
        onJourneyExit={onJourneyExit}
        onJourneyNext={onJourneyNext}
        onJourneyBack={onJourneyBack}
        journeyNextDisabled
      />
    );

    expect(screen.getByText("Step 1 of 1")).toBeInTheDocument();
    expect(screen.getByText("Posture content")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Next →" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "← Back" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: "Exit Journey" }));
    expect(onJourneyExit).toHaveBeenCalledOnce();
    fireEvent.click(screen.getByRole("button", { name: /Orientation/ }));
    expect(onJourneySelect).toHaveBeenCalledWith("orientation");
  });

  it("prompts for a journey when none is active", () => {
    renderWithProviders(
      <OrientationHub
        journeyCards={[]}
        activeJourney={null}
        journeySteps={[]}
        journeyStep={0}
        isLoading
        onJourneySelect={() => {}}
        onJourneyExit={() => {}}
        onJourneyNext={() => {}}
        onJourneyBack={() => {}}
      />
    );
    expect(screen.getByText("Select a journey from the left to begin your guided experience.")).toBeInTheDocument();
  });
});

describe("deployment readiness panels", () => {
  const tierReadiness = [
    { tier: "tier-1-local", label: "Tier 1", ready_percent: 50, strategized: 1, total: 2, blocking_secret_sample: ["vault/VAULT_TOKEN"] },
    { tier: "tier-2-desktop", label: "Tier 2", ready_percent: 100, strategized: 2, total: 2, blocking_secret_sample: [] }
  ];
  const manifest = {
    scenario: "secrets-manager",
    tier: "tier-2-desktop",
    resources: ["vault"],
    generated_at: "2026-07-23T00:00:00Z",
    secrets: [],
    summary: {
      total_secrets: 2,
      strategized_secrets: 1,
      requires_action: 1,
      blocking_secrets: ["vault/VAULT_TOKEN"],
      classification_weights: {},
      strategy_breakdown: {},
      scope_readiness: {}
    }
  };

  it("generates a scoped manifest from the readiness panel and routes blocker actions", () => {
    const onSetScenario = vi.fn();
    const onSetTier = vi.fn();
    const onSetResourcesInput = vi.fn();
    const onGenerateManifest = vi.fn();
    const onOpenResource = vi.fn();
    const onStartJourney = vi.fn();
    renderWithProviders(
      <DeploymentReadinessPanel
        tierReadiness={tierReadiness}
        resourceInsights={[{ resource_name: "vault", total_secrets: 1, valid_secrets: 0, missing_secrets: 1 }]}
        onOpenResource={onOpenResource}
        onStartJourney={onStartJourney}
        manifestState={{
          scenario: "secrets-manager",
          tier: "tier-2-desktop",
          resourcesInput: "",
          manifestData: manifest,
          manifestIsLoading: false,
          manifestIsError: false,
          onSetScenario,
          onSetTier,
          onSetResourcesInput,
          onGenerateManifest
        }}
      />
    );

    expect(screen.getByText("1 tier")).toBeInTheDocument();
    expect(screen.getByText("Manifest ready")).toBeInTheDocument();
    expect(screen.getByText("vault/VAULT_TOKEN")).toBeInTheDocument();
    fireEvent.click(screen.getByText("Open top blocker"));
    expect(onOpenResource).toHaveBeenCalledWith("vault");
    fireEvent.click(screen.getByText("Guided prep flow"));
    expect(onStartJourney).toHaveBeenCalledOnce();
    fireEvent.change(screen.getByPlaceholderText("picker-wheel, scenario-name"), { target: { value: "api-gateway" } });
    expect(onSetScenario).toHaveBeenCalledWith("api-gateway");
    fireEvent.change(screen.getByDisplayValue("Tier 2"), { target: { value: "tier-1-local" } });
    expect(onSetTier).toHaveBeenCalledWith("tier-1-local");
    fireEvent.change(screen.getByPlaceholderText("postgres, vault"), { target: { value: "vault, redis" } });
    expect(onSetResourcesInput).toHaveBeenCalledWith("vault, redis");
    fireEvent.click(screen.getByRole("button", { name: "Generate manifest" }));
    expect(onGenerateManifest).toHaveBeenCalledOnce();
  });

  it("presents blocked tiers as actionable and ready tiers as deployment-ready", () => {
    const onOpenResource = vi.fn();
    renderWithProviders(
      <TierReadiness
        tierReadiness={tierReadiness}
        isLoading={false}
        onOpenResource={onOpenResource}
        resourceInsights={[{ resource_name: "vault", total_secrets: 1, valid_secrets: 0, missing_secrets: 1 }]}
        resourceStatuses={[{ resource_name: "vault", secrets_total: 1, secrets_found: 0, secrets_missing: 1, secrets_optional: 0, health_status: "degraded", last_checked: "now" }]}
      />
    );

    expect(screen.getByText("1 secret blocking deployment")).toBeInTheDocument();
    expect(screen.getAllByText("✓ Ready for deployment").length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "Configure for Tier 1 →" }));
    expect(onOpenResource).toHaveBeenCalledWith("vault", undefined, "tier-1-local");
  });

  it("handles loading, absent resources, and blocked tiers without samples", () => {
    const { rerender } = renderWithProviders(
      <TierReadiness
        tierReadiness={[]}
        isLoading
        onOpenResource={() => {}}
        resourceInsights={[{ resource_name: "vault", total_secrets: 1, valid_secrets: 0, missing_secrets: 1 }]}
        resourceStatuses={[]}
      />
    );
    expect(screen.getByText(/0 total required/)).toBeInTheDocument();

    rerender(
      <TierReadiness
        tierReadiness={[
          { tier: "tier-5-enterprise", label: "Tier 5", ready_percent: 0, strategized: 0, total: 2, blocking_secret_sample: [] },
          { tier: "tier-4-saas", label: "Tier 4", ready_percent: 100, strategized: 0, total: 0, blocking_secret_sample: [] }
        ]}
        isLoading={false}
        onOpenResource={() => {}}
        resourceInsights={[{ resource_name: "vault", total_secrets: 1, valid_secrets: 0, missing_secrets: 1 }]}
        resourceStatuses={[{ resource_name: "vault", secrets_total: 2, secrets_found: 0, secrets_missing: 2, secrets_optional: 0, health_status: "degraded", last_checked: "now" }]}
      />
    );
    expect(screen.getByText("2 secrets blocking deployment")).toBeInTheDocument();
    expect(screen.getAllByText("✓ Ready for deployment").length).toBeGreaterThan(0);
    expect(screen.queryByRole("button", { name: /Configure for Tier 5/ })).not.toBeInTheDocument();
  });

  it("shows empty, loading, and failure states for the manifest generator", () => {
    const baseState = {
      scenario: "",
      tier: "",
      resourcesInput: "",
      manifestIsLoading: false,
      manifestIsError: false,
      onSetScenario: vi.fn(),
      onSetTier: vi.fn(),
      onSetResourcesInput: vi.fn(),
      onGenerateManifest: vi.fn()
    };
    const { rerender } = renderWithProviders(
      <DeploymentReadinessPanel
        tierReadiness={[]}
        resourceInsights={[]}
        onOpenResource={() => {}}
        manifestState={baseState}
      />
    );
    expect(screen.getByText("No blocking tiers detected")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Generate manifest" })).toBeDisabled();
    expect(screen.getByText(/No manifest generated yet/)).toBeInTheDocument();
    expect(screen.queryByText("Open top blocker")).not.toBeInTheDocument();

    rerender(
      <DeploymentReadinessPanel
        tierReadiness={tierReadiness}
        resourceInsights={[{ resource_name: "vault", total_secrets: 1, valid_secrets: 0, missing_secrets: 1 }]}
        onOpenResource={() => {}}
        manifestState={{ ...baseState, scenario: "vault", tier: "tier-1-local", manifestIsLoading: true }}
      />
    );
    expect(screen.getByText("Generating...")).toBeDisabled();
    expect(screen.getByText("Analyzing dependencies and strategies...")).toBeInTheDocument();
    expect(screen.getByText("Configure vault")).toBeInTheDocument();

    rerender(
      <DeploymentReadinessPanel
        tierReadiness={tierReadiness}
        resourceInsights={[{ resource_name: "vault", total_secrets: 1, valid_secrets: 0, missing_secrets: 1 }]}
        onOpenResource={() => {}}
        manifestState={{ ...baseState, scenario: "vault", tier: "tier-1-local", manifestIsError: true, manifestError: new Error("manifest API unavailable") }}
      />
    );
    expect(screen.getByText("manifest API unavailable")).toBeInTheDocument();
  });
});
