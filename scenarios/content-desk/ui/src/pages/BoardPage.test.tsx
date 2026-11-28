import { fireEvent, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "@vrooli/api-base/testing";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { BoardPage } from "./BoardPage";

const activeDraftId = "draft-active";
const terminalDraftId = "draft-done";
const campaignName = "Aquila";
const gapCapabilityName = "Generated video";
const healthyCapabilityName = "Copy writing";
const offerScenario = "web-console";
const offerNextAction = "Shipped; verify the release record.";

const mocks = vi.hoisted(() => ({
  listDrafts: vi.fn(),
  listCampaigns: vi.fn(),
  listCapabilities: vi.fn(),
  readBoard: vi.fn(),
}));

vi.mock("../api/artifacts", () => ({ artifactsClient: { listDrafts: mocks.listDrafts } }));
vi.mock("../api/campaigns", () => ({ campaignsClient: { listCampaigns: mocks.listCampaigns } }));
vi.mock("../api/capabilities", () => ({ capabilitiesClient: { listCapabilities: mocks.listCapabilities } }));
vi.mock("../api/board", () => ({ readBoard: mocks.readBoard }));

describe("BoardPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.listDrafts.mockResolvedValue({ drafts: [] });
    mocks.listCampaigns.mockResolvedValue({ campaigns: [] });
    mocks.listCapabilities.mockResolvedValue({ capabilities: [] });
    mocks.readBoard.mockResolvedValue({
      program: "content-desk.board-read",
      version: "1",
      status: "ok",
      phase: "report",
      signals: {},
      evidence: [],
      errors: [],
    });
  });

  it("shows non-terminal work with its next action and hides terminal drafts", async () => {
    mocks.listDrafts.mockResolvedValue({
      drafts: [
        { id: activeDraftId, status: "reviewed", campaignId: "campaign-1", channel: "x" },
        { id: terminalDraftId, status: "published", campaignId: "campaign-1", channel: "x" },
      ],
    });
    mocks.listCampaigns.mockResolvedValue({
      campaigns: [{ id: "campaign-1", name: campaignName, status: "active" }],
    });

    renderWithProviders(<BoardPage />);
    fireEvent.click(screen.getByTestId(selectors.pages.boardRefresh));

    await screen.findByText(activeDraftId);
    expect(screen.queryByText(terminalDraftId)).not.toBeInTheDocument();
    expect(screen.getByText(strings.board.nextAction.reviewed)).toBeInTheDocument();
    expect(screen.getAllByText(campaignName).length).toBeGreaterThan(0);
  });

  it("lists capability readiness gaps and omits healthy capabilities", async () => {
    mocks.listCapabilities.mockResolvedValue({
      capabilities: [
        {
          id: "cap-gap",
          name: gapCapabilityName,
          owner: "asset-studio",
          nextAction: "qualify output",
          definitionStatus: "documented",
          implementationStatus: "implemented",
          operationalReadiness: "unverified",
          outputQuality: "accepted-by-review",
          distributionConnectivity: "not-applicable",
        },
        {
          id: "cap-ok",
          name: healthyCapabilityName,
          owner: "prose-studio",
          nextAction: "",
          definitionStatus: "documented",
          implementationStatus: "implemented",
          operationalReadiness: "qualified-in-environment",
          outputQuality: "accepted-by-review",
          distributionConnectivity: "connected",
        },
      ],
    });

    renderWithProviders(<BoardPage />);
    fireEvent.click(screen.getByTestId(selectors.pages.boardRefresh));

    await screen.findByText(gapCapabilityName);
    expect(screen.queryByText(healthyCapabilityName)).not.toBeInTheDocument();
    expect(screen.getByText(/board\.dimensions\.operational/)).toBeInTheDocument();
  });

  it("shows the empty state when no work or gaps remain", async () => {
    renderWithProviders(<BoardPage />);
    fireEvent.click(screen.getByTestId(selectors.pages.boardRefresh));

    await screen.findAllByText(strings.board.emptyTitle);
    expect(screen.getByTestId(selectors.pages.board)).toBeInTheDocument();
  });

  it("shows offer release readiness targets from the board read", async () => {
    mocks.readBoard.mockResolvedValue({
      program: "content-desk.board-read",
      version: "1",
      status: "ok",
      phase: "report",
      signals: {
        offer_read_status: "read",
        offer_readiness: {
          ladder_entries: 14,
          launch_scenarios: [offerScenario],
          launch_targets: [
            {
              scenario: offerScenario,
              node_id: "node-1",
              status: "TRIGGER_MET",
              release_rank: 1,
              readiness_goal_reported: false,
              readiness_goal_exists: null,
              readiness_goal_closed: null,
              readiness_approved_commit: "",
              blocked_by: [{ name: "ai-gateway", status: "IDEA" }],
              next_action: offerNextAction,
            },
          ],
          unknown_goal_state: 1,
        },
      },
      evidence: [],
      errors: [],
    });

    renderWithProviders(<BoardPage />);
    fireEvent.click(screen.getByTestId(selectors.pages.boardRefresh));

    await screen.findByText(offerScenario);
    expect(screen.getByText(offerNextAction)).toBeInTheDocument();
    expect(screen.getByText(/ai-gateway/)).toBeInTheDocument();
  });

  it("keeps the owner board visible when the board read is unavailable", async () => {
    mocks.readBoard.mockRejectedValue(new Error("board read unavailable"));

    renderWithProviders(<BoardPage />);
    fireEvent.click(screen.getByTestId(selectors.pages.boardRefresh));

    await screen.findByText(strings.board.offerUnavailable);
    expect(screen.getByTestId(selectors.pages.board)).toBeInTheDocument();
  });
});
