import { describe, expect, it, vi } from "vitest";
import { Route, Routes } from "react-router-dom";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
import { planWorkshopService } from "../services/plan-workshop-service";
import { createTestQueryClient, renderWithProviders } from "../test-utils";

vi.mock("../hooks/useStorePolling", () => ({ useStorePolling: vi.fn() }));
vi.mock("../services/backlog-service", () => ({
  backlogService: {
    get: vi.fn().mockResolvedValue({ name: "test-idea", title: "Test idea", description: "A plan workshop subject", status: "backlog", priority: 2, tags: [], suggestedSkills: [], created: "2026-01-20T00:00:00Z", updated: "2026-01-20T00:00:00Z", kind: "idea" }),
    getFiles: vi.fn().mockResolvedValue([]),
    listBySpawnedFrom: vi.fn().mockResolvedValue([]),
    getMaturitySummary: vi.fn().mockResolvedValue({ items: [] }),
    getBacklogSummary: vi.fn().mockResolvedValue({ maturity: { items: [] } }),
    getArchiveTargets: vi.fn().mockResolvedValue({ targets: [], requirements: [], has_archive: false }),
    batchReview: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    archiveItem: vi.fn(),
    unarchiveItem: vi.fn(),
    getNextAction: vi.fn().mockResolvedValue({ id: "accept_plan", compactLabel: "Accept plan", expandedLabel: "Accept plan", enabled: true, reason: "canonical plan has not been explicitly accepted", blockers: [], target: "plan_accept", effect: "state_change" }),
  },
}));
vi.mock("../services/execution-service", () => ({ executionService: { list: vi.fn().mockResolvedValue([]) } }));
vi.mock("../services/review-service", () => ({ reviewService: { listRounds: vi.fn().mockResolvedValue([]) } }));
vi.mock("../services/plan-workshop-service", () => ({ planWorkshopService: { acceptPlan: vi.fn() } }));

describe("BacklogDetailsPage", () => {
  it("presents the consolidated Decide tab rather than legacy workshop controls", async () => {
    renderWithProviders(
      <Routes><Route path="/backlog/:kind/:name" element={<BacklogDetailsPage />} /></Routes>,
      { queryClient: createTestQueryClient(), initialEntries: ["/backlog/idea/test-idea"] },
    );

    expect(await screen.findByTestId("backlog-details-page")).toBeInTheDocument();
    expect(await screen.findByRole("tab", { name: /Decide/i })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Workshop$/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /^Finalize$/i })).toBeNull();
  });

  // The header CTA must perform the acceptance, not merely open the Plan tab:
  // when that tab was already open the button did nothing visible.
  it("accepts the plan from the header's Accept plan button", async () => {
    vi.mocked(planWorkshopService.acceptPlan).mockResolvedValue({ plan_acceptance: { actor: "operator", accepted_at: "2026-09-13T00:00:00Z", plan_content_hash: "hash", subject_version: "sha256:v" } });
    renderWithProviders(
      <Routes><Route path="/backlog/:kind/:name" element={<BacklogDetailsPage />} /></Routes>,
      { queryClient: createTestQueryClient(), initialEntries: ["/backlog/idea/test-idea?tab=prompt"] },
    );

    fireEvent.click(await screen.findByRole("button", { name: /Accept plan/i }));

    await waitFor(() => expect(planWorkshopService.acceptPlan).toHaveBeenCalledWith("idea", "test-idea"));
  });
});
