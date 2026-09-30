import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SecretDetail } from "./SecretDetail";
import { renderWithProviders } from "../../test-utils/renderWithProviders";

const secret = {
  id: "secret-1",
  secret_key: "POSTGRES_PASSWORD",
  secret_type: "password",
  description: "Database password",
  classification: "service",
  required: true,
  owner_team: "platform",
  owner_contact: "platform@example.test",
  tier_strategies: { "tier-1": "prompt" },
  validation_state: "valid"
};

describe("SecretDetail", () => {
  afterEach(cleanup);

  it("edits classification, requirements, strategy, and scenario overrides", () => {
    const onUpdateSecret = vi.fn();
    const handlers = {
      onApplyStrategy: vi.fn(),
      onDeleteOverride: vi.fn(),
      onSetStrategyTier: vi.fn(),
      onSetStrategyHandling: vi.fn(),
      onSetStrategyPrompt: vi.fn(),
      onSetStrategyDescription: vi.fn(),
      onSetOverrideReason: vi.fn(),
      onSetIsOverrideMode: vi.fn()
    };
    renderWithProviders(
      <SecretDetail
        selectedSecret={secret}
        tierReadiness={[{ tier: "tier-1", label: "Tier 1" }, { tier: "tier-2", label: "Tier 2" }]}
        strategyTier="tier-1"
        strategyHandling="prompt"
        strategyPrompt="Database password"
        strategyDescription="Used by the database"
        selectedScenario="secrets-manager"
        isOverrideMode
        overrideReason=""
        currentOverride={{ id: "override-1", scenario_name: "secrets-manager", resource_secret_id: "secret-1", resource_name: "postgres", secret_key: "POSTGRES_PASSWORD", tier: "tier-1", override_reason: "Local", handling_strategy: "generate", created_at: "2026-09-06T00:00:00Z", updated_at: "2026-09-06T00:00:00Z" }}
        onUpdateSecret={onUpdateSecret}
        {...handlers}
      />
    );

    const selects = screen.getAllByRole("combobox");
    const classificationSelect = selects[0];
    const handlingSelect = selects[1];
    if (!classificationSelect || !handlingSelect) throw new Error("expected classification and handling selects");
    fireEvent.change(classificationSelect, { target: { value: "user" } });
    fireEvent.click(screen.getByRole("button", { name: "Mark optional" }));
    fireEvent.change(screen.getByLabelText(/Prompt Label/), { target: { value: "Token" } });
    fireEvent.change(screen.getByLabelText(/Prompt Description/), { target: { value: "Token description" } });
    fireEvent.change(screen.getByLabelText(/Override Reason/), { target: { value: "Different local policy" } });
    fireEvent.change(handlingSelect, { target: { value: "generate" } });
    fireEvent.click(screen.getByRole("button", { name: "Remove override" }));
    fireEvent.click(screen.getByRole("button", { name: "Apply scenario override" }));

    expect(onUpdateSecret).toHaveBeenCalledWith("POSTGRES_PASSWORD", { classification: "user" });
    expect(onUpdateSecret).toHaveBeenCalledWith("POSTGRES_PASSWORD", { required: false });
    expect(handlers.onApplyStrategy).toHaveBeenCalled();
  });

  it("shows the empty selection state and non-prompt strategy guidance", () => {
    const onSetStrategyHandling = vi.fn();
    renderWithProviders(
      <SecretDetail
        tierReadiness={[]}
        strategyTier="tier-1"
        strategyHandling="generate"
        strategyPrompt=""
        strategyDescription=""
        onUpdateSecret={() => undefined}
        onApplyStrategy={() => undefined}
        onSetStrategyTier={() => undefined}
        onSetStrategyHandling={onSetStrategyHandling}
        onSetStrategyPrompt={() => undefined}
        onSetStrategyDescription={() => undefined}
      />
    );
    expect(screen.getByText("Select a secret to view details.")).toBeInTheDocument();
  });
});
