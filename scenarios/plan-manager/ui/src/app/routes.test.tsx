/**
 * Routing smoke — for each canonical path the matching page selector is in the
 * document. Page-internal behaviour is exercised in per-page tests; this file's
 * job is to assert the router config. Query-backed boards are mocked at the
 * api/* boundary so no real fetch fires during the routing smoke.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

vi.mock("../api/plans", () => ({
  listPlans: vi.fn().mockResolvedValue([]),
  listTemplates: vi.fn().mockResolvedValue([]),
  getPlan: vi.fn().mockResolvedValue(undefined),
  getGraph: vi.fn().mockResolvedValue([]),
  renderPlan: vi.fn().mockResolvedValue(""),
  archivePlan: vi.fn().mockResolvedValue(undefined),
  createFromTemplate: vi.fn().mockResolvedValue(undefined),
}));
vi.mock("../api/execution", () => ({
  listCandidateFindings: vi.fn().mockResolvedValue([]),
  getVelocity: vi.fn().mockResolvedValue([]),
}));

import { TestAppRouter } from "./routes";

const cases: { path: string; selector: string }[] = [
  { path: "/", selector: selectors.pages.dashboard },
  { path: "/plans", selector: selectors.pages.plans },
  { path: "/authoring", selector: selectors.pages.authoring },
  { path: "/execution", selector: selectors.pages.execution },
  { path: "/validation", selector: selectors.pages.validation },
  { path: "/triage", selector: selectors.pages.triage },
  { path: "/velocity", selector: selectors.pages.velocity },
  { path: "/settings", selector: selectors.pages.settings },
];

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it.each(cases)("renders the page at $path", async ({ path, selector }) => {
    renderWithProviders(<TestAppRouter initialEntries={[path]} />, { withoutRouter: true });
    expect(await screen.findByTestId(selector)).toBeInTheDocument();
  });

  it("renders the plan detail page at /plans/:planId", async () => {
    renderWithProviders(<TestAppRouter initialEntries={["/plans/some-id"]} />, {
      withoutRouter: true,
    });
    // The detail page renders the empty/notFound state when the plan is absent;
    // its root section selector is present regardless.
    expect(await screen.findByTestId(`${selectors.pages.planDetail}-empty`)).toBeInTheDocument();
  });
});
