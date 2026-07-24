import { Fragment } from "react";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { buildJourneySteps, type JourneyId } from "./journeySteps";

const callbacks = () => ({
  onOpenResource: vi.fn(),
  onRefetchVulnerabilities: vi.fn(),
  onRefreshReadiness: vi.fn(),
  onManifestRequest: vi.fn(),
  onSetDeploymentScenario: vi.fn(),
  onSetDeploymentTier: vi.fn(),
  onSetResourceInput: vi.fn(),
  onSetProvisionResourceInput: vi.fn(),
  onSetProvisionSecretKey: vi.fn(),
  onSetProvisionSecretValue: vi.fn(),
  onProvisionSubmit: vi.fn(),
  onNavigateTab: vi.fn(),
  onStartTutorial: vi.fn()
});

function build(journey: JourneyId, overrides = {}) {
  const handlers = callbacks();
  const steps = buildJourneySteps(journey, {
    heroStats: { vault_configured: 2, vault_total: 4, missing_secrets: 1, risk_score: 75 },
    vulnerabilitySummary: { critical: 2, high: 1, medium: 3, low: 4 },
    tierReadiness: [
      { tier: "tier-1-local", label: "Tier 1", ready_percent: 50, strategized: 1, total: 2 },
      { tier: "tier-2-desktop", label: "Tier 2", ready_percent: 100, strategized: 2, total: 2 },
      { tier: "tier-3-mobile", label: "Tier 3", ready_percent: 80, strategized: 2, total: 3 },
      { tier: "tier-4-saas", label: "Tier 4", ready_percent: 60, strategized: 1, total: 3 }
    ],
    deploymentScenario: "secrets-manager",
    deploymentTier: "tier-2-desktop",
    resourceInput: "vault",
    provisionResourceInput: "vault",
    provisionSecretKey: "VAULT_TOKEN",
    provisionSecretValue: "value",
    provisionIsLoading: false,
    provisionIsSuccess: false,
    manifestIsLoading: false,
    manifestIsError: false,
    ...handlers,
    ...overrides
  });
  return { handlers, steps };
}

function renderSteps(journey: JourneyId, overrides = {}) {
  const result = build(journey, overrides);
  renderWithProviders(
    <>{result.steps.map((step) => <Fragment key={step.title}>{step.content}</Fragment>)}</>
  );
  return result;
}

describe("buildJourneySteps", () => {
  afterEach(cleanup);

  it("builds orientation and configuration actions", () => {
    const orientation = renderSteps("orientation");
    expect(screen.getByText("Readiness Snapshot")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Open Resources" }));
    fireEvent.click(screen.getByRole("button", { name: "Go to Compliance" }));
    fireEvent.click(screen.getByRole("button", { name: "Open Deployment" }));
    expect(orientation.handlers.onNavigateTab).toHaveBeenNthCalledWith(1, "resources");
    expect(orientation.handlers.onNavigateTab).toHaveBeenNthCalledWith(2, "compliance");
    expect(orientation.handlers.onNavigateTab).toHaveBeenNthCalledWith(3, "deployment");

    cleanup();
    const configure = renderSteps("configure-secrets");
    expect(screen.getByText("Coverage ratio")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start guided tutorial" }));
    fireEvent.click(screen.getByRole("button", { name: "View Resources tab" }));
    fireEvent.click(screen.getByRole("button", { name: "Jump to By Tier" }));
    fireEvent.click(screen.getByRole("button", { name: "View resource table" }));
    fireEvent.click(screen.getByRole("button", { name: "Open top blocker" }));
    expect(configure.handlers.onStartTutorial).toHaveBeenCalledWith("configure-secrets");
    expect(configure.handlers.onOpenResource).toHaveBeenCalledWith();
  });

  it("builds vulnerability and deployment actions for blocked and ready tiers", () => {
    const vulnerabilities = renderSteps("fix-vulnerabilities");
    expect(screen.getByText("Critical / High")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start guided tutorial" }));
    fireEvent.click(screen.getByRole("button", { name: "Open Compliance tab" }));
    fireEvent.click(screen.getByRole("button", { name: "Re-run scan" }));
    fireEvent.click(screen.getByRole("button", { name: "Jump to compliance overview" }));
    fireEvent.click(screen.getByRole("button", { name: "Go to findings table" }));
    fireEvent.click(screen.getByRole("button", { name: "Refresh scan results" }));
    expect(vulnerabilities.handlers.onRefetchVulnerabilities).toHaveBeenCalledTimes(2);

    cleanup();
    const deployment = renderSteps("prep-deployment");
    expect(screen.getByText(/Blocked: Tier 1, Tier 3, Tier 4/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start guided tutorial" }));
    fireEvent.click(screen.getByRole("button", { name: "Open Deployment tab" }));
    fireEvent.click(screen.getByRole("button", { name: "Go to campaigns" }));
    fireEvent.click(screen.getByRole("button", { name: "Open top blocker" }));
    expect(deployment.handlers.onStartTutorial).toHaveBeenCalledWith("prep-deployment");
    expect(deployment.handlers.onOpenResource).toHaveBeenCalledWith();

    cleanup();
    renderSteps("prep-deployment", {
      tierReadiness: [{ tier: "tier-1-local", label: "Tier 1", ready_percent: 100, strategized: 2, total: 2 }]
    });
    expect(screen.getByText(/All tiers show full strategy coverage/)).toBeInTheDocument();
  });
});
