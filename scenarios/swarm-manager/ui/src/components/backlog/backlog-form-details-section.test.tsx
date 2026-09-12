import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";

const mockGet = vi.fn();
vi.mock("../../lib/api-client", () => ({
  defaultApiClient: { get: (...args: unknown[]) => mockGet(...args) },
}));

import { BacklogFormDetailsSection } from "./backlog-form-details-section";

describe("BacklogFormDetailsSection execution contract", () => {
  beforeEach(() => {
    mockGet.mockResolvedValue({ items: [
      { id: "phased-plan-drain", display_name: "Phased plan drain" },
      { id: "adaptive-improvement", display_name: "Adaptive improvement" },
      { id: "goal-session", display_name: "Goal session" },
    ] });
  });

  it("renders reviewed strategy, limits, continuation, and scope fields", async () => {
    const onFieldChange = vi.fn();
    renderWithProviders(<BacklogFormDetailsSection
      description=""
      status="backlog"
      priority={5}
      tagsInput=""
      milestone={undefined}
      dependsOn={undefined}
      effort={undefined}
      acceptanceAllow={undefined}
      acceptanceDeny={undefined}
      executionStrategy="adaptive-improvement"
      executionLimits={{ maxSlices: 12, maxTokens: 1000, maxWallSeconds: 60, maxTurns: 20, maxChargeMicroUsd: 500, maxChildren: 2, maxNodeAttempts: 3, maxRetries: 1 }}
      continuation="until-allowance"
      scopePolicy="extend-with-record"
      isEditMode
      isSubmitting={false}
      onFieldChange={onFieldChange}
      onTagsInputChange={vi.fn()}
      onClearError={vi.fn()}
    />);

    await waitFor(() => expect(screen.getByRole("option", { name: "Goal session" })).toBeInTheDocument());
    expect(screen.getByRole("region", { name: "Execution" })).toHaveTextContent(/Editing them invalidates plan acceptance/);
    expect(screen.getByRole("combobox", { name: "Execution strategy" })).toHaveValue("adaptive-improvement");
    expect(screen.getByRole("radio", { name: /Until allowance is spent/ })).toBeChecked();
    expect(screen.getByRole("radio", { name: /Extend with record/ })).toBeChecked();

    fireEvent.change(screen.getByRole("combobox", { name: "Execution strategy" }), { target: { value: "goal-session" } });
    expect(onFieldChange).toHaveBeenCalledWith("executionStrategy", "goal-session");
    fireEvent.click(screen.getByRole("radio", { name: /Manual/ }));
    expect(onFieldChange).toHaveBeenCalledWith("continuation", "manual");
  });
});
