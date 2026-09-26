/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav) for every canonical
 * route — catching landmark-uniqueness regressions when a new page adds its own
 * <nav>/<section> landmarks. Feature cards keep their own a11y tests; the api/*
 * boundaries are mocked so query-backed boards reach a stable state.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";

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
  getVelocity: vi.fn().mockResolvedValue([]),
  startExecution: vi.fn(),
  getStatus: vi.fn(),
  getContext: vi.fn(),
  transitionPhase: vi.fn(),
  completeExecution: vi.fn(),
  getNext: vi.fn(),
  getHandoff: vi.fn(),
}));
vi.mock("../api/log", () => ({
  listEntries: vi.fn().mockResolvedValue({ entries: [], summary: undefined, step: undefined }),
  promoteEntry: vi.fn(),
  updateEntry: vi.fn(),
  addDecision: vi.fn(),
  addFinding: vi.fn(),
}));

import { TestAppRouter } from "../app/routes";

const ROUTES = ["/", "/plans", "/authoring", "/execution", "/validation", "/triage", "/velocity", "/settings"];

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it.each(ROUTES)("renders %s through the shell without axe violations", async (route) => {
    const { container } = renderWithProviders(<TestAppRouter initialEntries={[route]} />, {
      withoutRouter: true,
    });
    await expectNoA11yViolations(container);
  });
});
