import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { ManifestSummaryBar } from "./ManifestSummaryBar";

describe("ManifestSummaryBar", () => {
  afterEach(cleanup);

  it("shows override counts only when present and toggles the export preview", () => {
    renderWithProviders(
      <ManifestSummaryBar
        summary={{ resourceCount: 2, totalSecrets: 3, strategizedSecrets: 2, blockingSecrets: 1, excludedSecrets: 1, overriddenSecrets: 1 }}
        exportPreview={{
          scenario: "secrets-manager",
          tier: "tier-2-desktop",
          generated_at: "2026-07-23T00:00:00Z",
          resources: ["vault"],
          secrets: [],
          summary: { total_secrets: 0, strategized_secrets: 0, requires_action: 0, blocking_secrets: [], classification_weights: {}, strategy_breakdown: {}, scope_readiness: {} }
        }}
      />
    );
    expect(screen.getByText("Overridden:")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Preview JSON" }));
    expect(screen.getByText(/This is a preview of what will be exported/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Hide JSON" }));
    expect(screen.queryByText(/This is a preview of what will be exported/)).not.toBeInTheDocument();
  });

  it("does not render an override indicator when no override exists", () => {
    renderWithProviders(
      <ManifestSummaryBar
        summary={{ resourceCount: 0, totalSecrets: 0, strategizedSecrets: 0, blockingSecrets: 0, excludedSecrets: 0, overriddenSecrets: 0 }}
        exportPreview={{ scenario: "secrets-manager", tier: "tier-1-local", generated_at: "now", resources: [], secrets: [], summary: { total_secrets: 0, strategized_secrets: 0, requires_action: 0, blocking_secrets: [], classification_weights: {}, strategy_breakdown: {}, scope_readiness: {} } }}
      />
    );
    expect(screen.queryByText("Overridden:")).not.toBeInTheDocument();
  });
});
