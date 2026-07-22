import { describe, expect, it, vi } from "vitest";
import { Route, Routes } from "react-router-dom";
import { screen } from "@testing-library/react";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
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
  },
}));
vi.mock("../services/execution-service", () => ({ executionService: { list: vi.fn().mockResolvedValue([]) } }));
vi.mock("../services/review-service", () => ({ reviewService: { listRounds: vi.fn().mockResolvedValue([]) } }));
vi.mock("../services/plan-workshop-service", () => ({ planWorkshopService: { acceptPlan: vi.fn() } }));

describe("BacklogDetailsPage", () => {
  it("presents Plan Workshop rather than legacy workshop controls", async () => {
    renderWithProviders(
      <Routes><Route path="/backlog/:kind/:name" element={<BacklogDetailsPage />} /></Routes>,
      { queryClient: createTestQueryClient(), initialEntries: ["/backlog/idea/test-idea"] },
    );

    expect(await screen.findByTestId("backlog-details-page")).toBeInTheDocument();
    expect(await screen.findByTestId("plan-workshop-panel")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Workshop$/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /^Finalize$/i })).toBeNull();
  });
});
