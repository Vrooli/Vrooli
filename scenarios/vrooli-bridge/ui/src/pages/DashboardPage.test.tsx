import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { expectNoA11yViolations, makeHealthResponse, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

// DashboardPage composes every fleet surface; stub the on-mount queries so the
// page renders deterministically without the network.
const { listNodes, revokeNode, listQueue, listRuns, getRun, abortRun, fetchHealth } = vi.hoisted(() => ({
  listNodes: vi.fn(),
  revokeNode: vi.fn(),
  listQueue: vi.fn(),
  listRuns: vi.fn(),
  getRun: vi.fn(),
  abortRun: vi.fn(),
  fetchHealth: vi.fn(),
}));

vi.mock("../api/nodes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/nodes")>();
  return { ...actual, nodesClient: { listNodes, revokeNode } };
});
vi.mock("../api/queue", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/queue")>();
  return { ...actual, queueClient: { listQueue } };
});
vi.mock("../api/runs", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/runs")>();
  return { ...actual, runsClient: { listRuns, getRun, abortRun } };
});
vi.mock("../api/health", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/health")>();
  return { ...actual, fetchHealth };
});

import { DashboardPage } from "./DashboardPage";

const follows = (a: Element, b: Element) =>
  Boolean(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);

describe("DashboardPage", () => {
  beforeEach(() => {
    listNodes.mockResolvedValue({ nodes: [] });
    listQueue.mockResolvedValue({ nodes: [] });
    listRuns.mockResolvedValue({ runs: [] });
    fetchHealth.mockResolvedValue(makeHealthResponse());
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("leads with the fleet, then the wizard, run history, and a demoted health card", () => {
    renderWithProviders(<DashboardPage />);

    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: strings.pages.dashboard.title })).toBeInTheDocument();

    const fleet = screen.getByTestId(selectors.fleet.panel);
    const wizard = screen.getByTestId(selectors.fleet.onboard.form);
    const runs = screen.getByTestId(selectors.runs.panel);
    const health = screen.getByTestId(selectors.health.card);
    // Fleet first, then the wizard, then runs, then the (demoted) health card.
    expect(follows(fleet, wizard)).toBe(true);
    expect(follows(wizard, runs)).toBe(true);
    expect(follows(runs, health)).toBe(true);
  });

  it("keeps manual code pairing behind a collapsed disclosure", () => {
    renderWithProviders(<DashboardPage />);
    const disclosure = screen.getByTestId(selectors.fleet.pairing.disclosure);
    expect(disclosure).toBeInTheDocument();
    // The disclosure lives in a <details> that is closed by default.
    expect(disclosure.closest("details")).not.toHaveAttribute("open");
  });

  it("moves focus to the Add-a-node wizard when the empty-state CTA is used", async () => {
    const user = userEvent.setup();
    renderWithProviders(<DashboardPage />);

    await waitFor(() => expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.fleet.onboard.addNode));

    await waitFor(() => expect(document.getElementById("fleet-onboard-heading")).toHaveFocus());
  });

  it("composes without axe violations", async () => {
    const { container } = renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId(selectors.fleet.empty)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
