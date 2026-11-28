import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { OffersPage } from "./OffersPage";
import { ProposalsPage } from "./ProposalsPage";
import { TriggersPage } from "./TriggersPage";
import { DashboardPage } from "./DashboardPage";
import { SettingsPage } from "./SettingsPage";
import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

const api = vi.hoisted(() => ({
  fetchBoard: vi.fn(),
  fetchNodes: vi.fn(),
  fetchEdges: vi.fn(),
  fetchCatalogVerification: vi.fn(),
  fetchProposals: vi.fn(),
  evaluateTriggers: vi.fn(),
  promoteNode: vi.fn(),
  createNode: vi.fn(),
  transition: vi.fn(),
  createEdge: vi.fn(),
  declareTrigger: vi.fn(),
  addFact: vi.fn(),
}));

vi.mock("../api/offers", () => api);

afterEach(() => {
  cleanup();
  window.sessionStorage.clear();
  window.history.replaceState({}, "", "/");
});

beforeEach(() => {
  vi.clearAllMocks();
  api.fetchBoard.mockResolvedValue({
    entries: [{ title: "Example offer", status: "ACTIVE", actualMinor: 0n }],
    availability: [{ source: "money-ledger", reason: "stale" }],
    position: { runwayMonths: 4.2 },
  });
  api.fetchNodes.mockResolvedValue({ nodes: [{ id: "offer-1", name: "Example offer", status: "ACTIVE" }] });
  api.fetchEdges.mockResolvedValue({ edges: [{ id: "edge-1", kind: "MEMBERSHIP" }] });
  api.fetchCatalogVerification.mockResolvedValue({ files: [], duplicateIdentities: [], orphanEdgeIds: [], extraNodeIds: [], totalDrift: 0, reconciled: true });
  api.fetchProposals.mockResolvedValue({ proposals: [{ id: "proposal-1", nodeId: "offer-1", actor: "agent", requestedStatus: "ACTIVE", reason: "Ready for review", evidenceReference: "catalog/node/offer-1", createdAt: { seconds: 1n, nanos: 0 }, declineHistory: [] }] });
  api.evaluateTriggers.mockResolvedValue({ evaluations: [{ id: "eval-1", factName: "revenue", verdict: "UNKNOWN" }] });
  api.promoteNode.mockResolvedValue({
    proposal: { id: "proposal-1", nodeId: "offer-1", actor: "agent", rationale: "Awaiting operator review" },
  });
  api.createNode.mockResolvedValue({ node: { id: "offer-2", name: "New offer", status: "IDEA" } });
  api.transition.mockResolvedValue({ node: { id: "offer-1", name: "Example offer", status: "SHIPPED" } });
  api.createEdge.mockResolvedValue({ edge: { id: "edge-2", fromId: "offer-1", toId: "offer-1", kind: "feeds" } });
  api.declareTrigger.mockResolvedValue({ trigger: { id: "trigger-1", nodeId: "offer-1" } });
  api.addFact.mockResolvedValue({ fact: { name: "revenue", value: 100 } });
});

describe("offer desk authored states", () => {
  it("renders every ranked board entry and the default-alive gap [REQ:UI-001]", async () => {
    api.fetchBoard.mockResolvedValue({
      entries: [
        { nodeId: "offer-1", title: "First offer", status: "ACTIVE", rankReason: "active and earning nothing", actualMinor: 0n, actualsAvailable: false, availability: [{ source: "money-ledger.actuals", reason: "connection refused" }] },
        { nodeId: "offer-2", title: "Second offer", status: "CANDIDATE", rankReason: "blocked: trigger not met", actualMinor: 200n, actualsAvailable: true, availability: [] },
      ],
      position: { runwayMonths: 3.4 },
      postureSource: "money-ledger.position",
      postureAgeSeconds: 12n,
      goals: [{ goal: { id: "goal-1", name: "default-alive" }, met: false, explanation: "revenue is incomplete" }],
      evaluation: { nodesScored: 2, ageSeconds: 18n, degraded: true, reason: "evaluation is stale" },
      defaultAliveGap: "No default-alive offer is earning yet.",
      availability: [],
    });
    window.history.replaceState({}, "", "/");
    renderWithProviders(<DashboardPage />);

    expect((await screen.findAllByText(/First offer/)).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/Second offer/).length).toBeGreaterThan(0);
    expect(screen.getByText(/No default-alive offer is earning yet/)).toBeVisible();
    expect(screen.getByTestId(selectors.pages.boardRanking)).toHaveTextContent("pages.dashboard.actualUnavailable");
    expect(screen.getByTestId(selectors.pages.evaluationCondition)).toHaveTextContent("pages.dashboard.evaluationDegraded");
    expect(screen.getByTestId(selectors.pages.postureBasis)).toHaveTextContent("money-ledger.position");
    expect(screen.getByTestId(selectors.pages.postureGoalVerdicts)).toHaveTextContent("default-alive");
  });

  it("keeps an agent promotion non-active and renders the returned proposal [REQ:UI-002]", async () => {
    window.history.replaceState({}, "", "/offers");
    renderWithProviders(<OffersPage />);

    fireEvent.click(await screen.findByTestId(selectors.pages.catalogViewToggle));
    fireEvent.click(await screen.findByTestId("offer-promote"));

    expect(api.promoteNode).toHaveBeenCalledWith(expect.objectContaining({ nodeId: "offer-1", role: "agent" }));
    expect(await screen.findByText(/Awaiting operator review/)).toBeVisible();
    expect(screen.getByTestId(selectors.pages.offerStatus)).toHaveTextContent("ACTIVE");
  });

  it("writes catalog nodes, legal transitions, and relationships", async () => {
    window.history.replaceState({}, "", "/offers");
    renderWithProviders(<OffersPage />);

    fireEvent.change(document.getElementById("offer-node-name")!, { target: { value: "New offer" } });
    fireEvent.submit(screen.getByTestId(selectors.pages.offerCreateNode));
    await waitFor(() => expect(api.createNode).toHaveBeenCalledWith(expect.objectContaining({ name: "New offer" })));

    fireEvent.change(document.getElementById("offer-transition-node")!, { target: { value: "offer-1" } });
    fireEvent.change(document.getElementById("offer-transition-status")!, { target: { value: "5" } });
    fireEvent.submit(screen.getByTestId(selectors.pages.offerTransition));
    await waitFor(() => expect(api.transition).toHaveBeenCalledWith({ nodeId: "offer-1", status: 5, actor: "operator" }));

    fireEvent.change(document.getElementById("offer-edge-from")!, { target: { value: "offer-1" } });
    fireEvent.change(document.getElementById("offer-edge-to")!, { target: { value: "offer-1" } });
    fireEvent.submit(screen.getByTestId(selectors.pages.offerCreateEdge));
    await waitFor(() => expect(api.createEdge).toHaveBeenCalledWith(expect.objectContaining({ fromId: "offer-1", toId: "offer-1", kind: "belongs_to" })));
  });

  it("renders a refusal as an explicit request error with its remediation [REQ:UI-001] [REQ:UI-002]", async () => {
    api.fetchNodes.mockRejectedValue(new Error("promotion requires an operator"));
    renderWithProviders(<OffersPage />);

    await waitFor(() => expect(screen.getByTestId("page-offers")).toHaveAttribute("data-experience-state", "request-error"));
    expect(screen.getByTestId("offer-refusal-reason")).toHaveAttribute("role", "alert");
    expect(screen.getByTestId("offer-refusal-remedy")).toBeVisible();
    expect(screen.getByTestId("offer-audit-trail")).toBeVisible();
  });

  it("keeps trigger parse failure and missing-fact guidance visible", async () => {
    api.evaluateTriggers.mockRejectedValue(new Error("trigger expression is invalid"));
    renderWithProviders(<TriggersPage />);

    await waitFor(() => expect(screen.getByTestId("page-triggers")).toHaveAttribute("data-experience-state", "request-error"));
    expect(screen.getByTestId("trigger-parse-error")).toHaveAttribute("role", "alert");
    expect(screen.getByTestId("trigger-missing-fact")).toBeVisible();
    expect(screen.getByTestId("evaluation-freshness")).toBeVisible();
  });

  it("declares composed triggers, adds dated facts, and runs a dry evaluation", async () => {
    window.history.replaceState({}, "", "/triggers");
    renderWithProviders(<TriggersPage />);

    fireEvent.change(document.getElementById("trigger-node")!, { target: { value: "offer-1" } });
    fireEvent.change(document.getElementById("trigger-fact")!, { target: { value: "revenue" } });
    fireEvent.change(document.getElementById("trigger-threshold")!, { target: { value: "100" } });
    fireEvent.change(document.getElementById("trigger-second-fact")!, { target: { value: "margin" } });
    fireEvent.change(document.getElementById("trigger-second-threshold")!, { target: { value: "20" } });
    fireEvent.submit(screen.getByTestId(selectors.pages.triggerDeclare));
    await waitFor(() => expect(api.declareTrigger).toHaveBeenCalledWith(expect.objectContaining({ nodeId: "offer-1", factName: "revenue", threshold: 100, clauses: [{ factName: "margin", operator: ">=", threshold: 20 }] })));

    fireEvent.change(document.getElementById("fact-name")!, { target: { value: "revenue" } });
    fireEvent.change(document.getElementById("fact-value")!, { target: { value: "250" } });
    fireEvent.submit(screen.getByTestId(selectors.pages.triggerAddFact));
    await waitFor(() => expect(api.addFact).toHaveBeenCalledWith(expect.objectContaining({ name: "revenue", value: 250, staleAfterDays: 30 })));

    fireEvent.click(screen.getByTestId(selectors.pages.triggerDryRun));
    await waitFor(() => expect(api.evaluateTriggers).toHaveBeenLastCalledWith(true));
  });

  it("explains an empty proposal queue instead of inventing a proposal [REQ:UI-002]", async () => {
    api.fetchNodes.mockResolvedValue({ nodes: [] });
    api.fetchProposals.mockResolvedValue({ proposals: [] });
    renderWithProviders(<ProposalsPage />);

    await waitFor(() => expect(screen.getByTestId("page-proposals")).toHaveAttribute("data-experience-state", "empty"));
    expect(screen.getByTestId("proposals-empty-guidance")).toBeVisible();
    expect(screen.getByTestId("proposal-operator-only")).toBeVisible();
  });

  it("lists proposal evidence and keeps acceptance operator-gated", async () => {
    window.history.replaceState({}, "", "/proposals");
    const view = renderWithProviders(<ProposalsPage />);

    expect(await screen.findByTestId(selectors.pages.proposalTable)).toBeVisible();
    expect(screen.getByTestId(selectors.pages.proposalAccept)).toBeDisabled();
    expect(screen.getByTestId(selectors.pages.proposalEvidence)).toBeVisible();

    window.sessionStorage.setItem("vrooli.role", "operator");
    view.rerender(<ProposalsPage />);
    fireEvent.click(await screen.findByTestId(selectors.pages.proposalAccept));
    await waitFor(() => expect(api.promoteNode).toHaveBeenCalledWith({ nodeId: "offer-1", actor: "operator", role: "operator" }));
  });

  it("renders incomplete proposal metadata and records an operator decline", async () => {
    window.sessionStorage.setItem("vrooli.role", "operator");
    api.fetchNodes.mockResolvedValue({ nodes: [] });
    api.fetchProposals.mockResolvedValue({ proposals: [{ id: "proposal-2", nodeId: "missing-node", actor: "agent", requestedStatus: 99, reason: "", evidenceReference: "", declineHistory: [{ actor: "operator", reason: "Needs more evidence" }] }] });
    renderWithProviders(<ProposalsPage />);

    expect(await screen.findByTestId("proposal-row-proposal-2")).toBeVisible();
    expect(screen.getByTestId("proposal-row-proposal-2")).toHaveTextContent("Needs more evidence");
    fireEvent.change(screen.getByTestId(selectors.pages.proposalDeclineReason), { target: { value: "Not ready" } });
    fireEvent.click(screen.getByTestId(selectors.pages.proposalDecline));
    await waitFor(() => expect(api.transition).toHaveBeenCalledWith({ nodeId: "missing-node", status: 6, actor: "operator:decline:Not ready" }));
  });

  it("renders successful live board, catalog, and trigger reads", async () => {
    window.history.replaceState({}, "", "/");
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByTestId("board-ranking")).toBeVisible();

    window.history.replaceState({}, "", "/offers");
    cleanup();
    renderWithProviders(<OffersPage />);
    expect(await screen.findByTestId("offer-status")).toBeVisible();

    window.history.replaceState({}, "", "/triggers");
    cleanup();
    renderWithProviders(<TriggersPage />);
    expect(await screen.findByTestId("trigger-fact-trace")).toBeVisible();

    window.history.replaceState({}, "", "/proposals");
    cleanup();
    renderWithProviders(<ProposalsPage />);
    expect(await screen.findByTestId("proposal-proposer")).toBeVisible();
  });

  it("searches every adopted table without hiding column coverage", async () => {
    const dashboardView = renderWithProviders(<DashboardPage />);
    await screen.findByTestId(selectors.pages.boardRanking);
    fireEvent.change(screen.getByPlaceholderText(/pages\.dashboard\.boardOffer/), { target: { value: "no matching offer" } });

    dashboardView.unmount();
    const offersView = renderWithProviders(<OffersPage />);
    await screen.findByTestId(selectors.pages.catalogViewToggle);
    fireEvent.click(screen.getByTestId(selectors.pages.catalogViewToggle));
    await screen.findByTestId(selectors.pages.offerTable);
    fireEvent.change(screen.getByPlaceholderText(/pages\.offers\.offerLabel/), { target: { value: "no matching offer" } });

    offersView.unmount();
    const triggersView = renderWithProviders(<TriggersPage />);
    await screen.findByTestId(selectors.pages.factRegistry);
    fireEvent.change(screen.getByPlaceholderText(/pages\.triggers\.factNameLabel/), { target: { value: "no matching fact" } });

    triggersView.unmount();
    renderWithProviders(<ProposalsPage />);
    await screen.findByTestId(selectors.pages.proposalTable);
    fireEvent.change(screen.getByPlaceholderText(/pages\.proposals\.proposer/), { target: { value: "no matching proposal" } });
  });

  it("drives the dashboard partial state from mocked availability", async () => {
    api.fetchBoard.mockResolvedValue({ entries: [], availability: [{ source: "money-ledger", reason: "stale" }] });
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId("page-dashboard")).toHaveAttribute("data-experience-state", "partial"));
  });

  it("renders settings controls without fixture state injection", async () => {
    const disconnectedView = renderWithProviders(<SettingsPage />);
    expect(screen.getByTestId("page-settings")).toHaveAttribute("data-experience-state", "ready");
    expect(screen.getByTestId("settings-ledger-connection-reason")).toBeVisible();

    disconnectedView.unmount();
    api.fetchBoard.mockResolvedValue({ entries: [], availability: [], position: { runwayMonths: 2 } });
    const connectedView = renderWithProviders(<SettingsPage />);
    expect(await screen.findByTestId("settings-ledger-connection")).toHaveTextContent("pages.settings.ledgerConnected");
    expect(screen.getByTestId("settings-ledger-connection-reason")).toHaveClass("sr-only");

    connectedView.unmount();
    api.fetchBoard.mockRejectedValue(new Error("ledger unavailable"));
    renderWithProviders(<SettingsPage />);
    await waitFor(() => expect(screen.getByTestId("settings-ledger-connection-reason")).toHaveAttribute("role", "alert"));
  });
});
