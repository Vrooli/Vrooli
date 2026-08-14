import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";

import { OffersPage } from "./OffersPage";
import { ProposalsPage } from "./ProposalsPage";
import { TriggersPage } from "./TriggersPage";
import { DashboardPage } from "./DashboardPage";
import { SettingsPage } from "./SettingsPage";
import { renderWithProviders } from "../test-utils";

const api = vi.hoisted(() => ({
  fetchBoard: vi.fn(),
  fetchNodes: vi.fn(),
  fetchEdges: vi.fn(),
  evaluateTriggers: vi.fn(),
}));

vi.mock("../api/offers", () => api);

afterEach(() => {
  cleanup();
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
  api.evaluateTriggers.mockResolvedValue({ evaluations: [{ id: "eval-1", factName: "revenue", verdict: "UNKNOWN" }] });
});

function fixture(path: string, name: string) {
  window.history.replaceState({}, "", `${path}?fixture=${name}`);
}

describe("offer desk authored states", () => {
  it("renders a refusal as an explicit error with its remediation [REQ:UI-001] [REQ:UI-002]", () => {
    fixture("/offers", "promotion-blocked");
    renderWithProviders(<OffersPage />);

    expect(screen.getByTestId("page-offers")).toHaveAttribute("data-experience-state", "error");
    expect(screen.getByTestId("offer-refusal-reason")).toHaveAttribute("role", "alert");
    expect(screen.getByTestId("offer-refusal-remedy")).toBeVisible();
    expect(screen.getByTestId("offer-audit-trail")).toBeVisible();
  });

  it("keeps trigger parse failure and missing-fact guidance visible", () => {
    fixture("/triggers", "parse-error");
    renderWithProviders(<TriggersPage />);

    expect(screen.getByTestId("page-triggers")).toHaveAttribute("data-experience-state", "error");
    expect(screen.getByTestId("trigger-parse-error")).toHaveAttribute("role", "alert");
    expect(screen.getByTestId("trigger-missing-fact")).toBeVisible();
    expect(screen.getByTestId("evaluation-freshness")).toBeVisible();
  });

  it("explains an empty proposal queue instead of inventing a proposal [REQ:UI-002]", () => {
    fixture("/proposals", "empty");
    renderWithProviders(<ProposalsPage />);

    expect(screen.getByTestId("page-proposals")).toHaveAttribute("data-experience-state", "empty");
    expect(screen.getByTestId("proposals-empty-guidance")).toBeVisible();
    expect(screen.getByTestId("proposal-operator-only")).toBeVisible();
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

  it("exercises the dashboard, catalog, and trigger state matrix [REQ:UI-001] [REQ:GATE-004]", () => {
    for (const state of ["empty", "loading", "pending", "source-degraded", "all-sources-healthy", "posture-present", "posture-partial", "ledger-unavailable"]) {
      fixture("/", state);
      renderWithProviders(<DashboardPage />);
      expect(screen.getByTestId("page-dashboard")).toHaveAttribute("data-experience-state");
      cleanup();
    }
    for (const state of ["empty", "transition-refused", "promotion-blocked", "ready"]) {
      fixture("/offers", state);
      renderWithProviders(<OffersPage />);
      expect(screen.getByTestId("page-offers")).toHaveAttribute("data-experience-state");
      cleanup();
    }
    for (const state of ["empty", "parse-error", "ready"]) {
      fixture("/triggers", state);
      renderWithProviders(<TriggersPage />);
      expect(screen.getByTestId("page-triggers")).toHaveAttribute("data-experience-state");
      cleanup();
    }
  });

  it("surfaces settings connection states without hiding their remediation", () => {
    window.history.replaceState({}, "", "/settings?fixture=schedule-paused");
    renderWithProviders(<SettingsPage />);
    expect(screen.getByTestId("settings-schedule-paused")).toBeVisible();
    expect(screen.getByTestId("settings-ledger-connection-reason")).toHaveClass("sr-only");

    cleanup();
    window.history.replaceState({}, "", "/settings?fixture=ledger-disconnected");
    renderWithProviders(<SettingsPage />);
    expect(screen.getByTestId("settings-schedule-paused")).toHaveClass("sr-only");
    expect(screen.getByTestId("settings-ledger-connection-reason")).toBeVisible();
  });
});
