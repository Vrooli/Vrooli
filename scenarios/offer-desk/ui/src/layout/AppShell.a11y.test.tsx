/**
 * AppShell accessibility regression test. Renders the full route table through
 * the test-only memory router so axe sees the actual structural composition
 * (header + landmark nav + main + bottom landmark nav). Feature cards keep
 * their own a11y tests.
 */
import { afterEach, beforeEach, describe, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { expectNoA11yViolations, makeHealthResponse, renderWithProviders } from "../test-utils";
import { setLocale } from "../i18n";
import { TestAppRouter } from "../app/routes";

const api = vi.hoisted(() => ({ fetchBoard: vi.fn(), fetchHealth: vi.fn() }));

vi.mock("../api/offers", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/offers")>();
  return { ...actual, fetchBoard: api.fetchBoard };
});

vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth: api.fetchHealth };
});

describe("AppShell accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.clearAllMocks();
    api.fetchBoard.mockResolvedValue({
      entries: [{ nodeId: "a11y-offer", title: "Accessibility sample offer", status: "IDEA", actualMinor: 0n, actualsAvailable: false, availability: [] }],
      availability: [],
      position: undefined,
      evaluation: { degraded: false, nodesScored: 0 },
    });
    api.fetchHealth.mockResolvedValue(makeHealthResponse({ service: "offer-desk-a11y" }));
  });

  afterEach(() => {
    cleanup();
  });

  it("renders the shell without axe violations in English", async () => {
    const { container } = renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );
    await screen.findByText((_, element) => element?.textContent === "Accessibility sample offer");
    await screen.findByText((_, element) => element?.textContent === "offer-desk-a11y");
    await expectNoA11yViolations(container);
  });

  it("exposes exactly one primary navigation landmark", async () => {
    renderWithProviders(
      <TestAppRouter initialEntries={["/"]} />,
      { withoutRouter: true },
    );

    await screen.findByText((_, element) => element?.textContent === "Accessibility sample offer");
    await screen.findByText((_, element) => element?.textContent === "offer-desk-a11y");
    expect(screen.getAllByRole("navigation", { name: "Primary navigation" })).toHaveLength(1);
    expect(screen.getByTestId("layout-nav-link-dashboard-tab")).toBeInTheDocument();
  });
});
