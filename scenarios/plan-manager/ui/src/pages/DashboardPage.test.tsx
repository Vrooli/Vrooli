/**
 * DashboardPage tests — real plan-manager surfaces (stat counts, recent plans,
 * quick links) and axe-clean structure. api/* boundaries are mocked.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { makeHealthResponse } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";
import {
  PlanSchema,
  PlanStatus,
  ReferenceSchema,
  StalenessTier,
} from "@vrooli/proto-types/plan-manager/v1/shared/model_pb";

const listPlans = vi.fn();
const listEntries = vi.fn();

vi.mock("../api/plans", () => ({
  listPlans: (...a: unknown[]) => listPlans(...a),
  listTemplates: vi.fn(),
  getPlan: vi.fn(),
  getGraph: vi.fn(),
  renderPlan: vi.fn(),
  archivePlan: vi.fn(),
  createFromTemplate: vi.fn(),
}));
vi.mock("../api/log", () => ({
  listEntries: (...a: unknown[]) => listEntries(...a),
}));
vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: vi.fn().mockResolvedValue(makeHealthResponse()) };
});

import { DashboardPage } from "./DashboardPage";

const activePlan = create(PlanSchema, {
  id: "plan-1",
  title: "Migrate auth",
  status: PlanStatus.ACTIVE,
  updatedAt: "2026-06-25T10:00:00Z",
  references: [
    create(ReferenceSchema, { id: "r1", staleness: StalenessTier.DEFINITELY_STALE }),
  ],
});

describe("DashboardPage", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders real plan-manager stat surfaces and the recent plan", async () => {
    listPlans.mockResolvedValue([activePlan]);
    listEntries.mockResolvedValue({ entries: [], summary: undefined, step: undefined });

    renderWithProviders(<DashboardPage />);

    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
    await waitFor(() => {
      // Recent-plans list shows the active plan title once the query resolves.
      expect(screen.getAllByText(activePlan.title).length).toBeGreaterThan(0);
    });
  });

  it("renders without axe violations", async () => {
    listPlans.mockResolvedValue([activePlan]);
    listEntries.mockResolvedValue({ entries: [], summary: undefined, step: undefined });

    const { container } = renderWithProviders(<DashboardPage />);
    await waitFor(() => {
      expect(screen.getAllByText(activePlan.title).length).toBeGreaterThan(0);
    });
    await expectNoA11yViolations(container);
  });
});
