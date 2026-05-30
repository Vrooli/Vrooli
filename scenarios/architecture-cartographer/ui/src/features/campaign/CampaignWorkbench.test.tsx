import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../../api/campaign", () => ({
  campaignClient: {
    listCampaigns: vi.fn(),
    getCampaignStatus: vi.fn(),
    nextCampaignStep: vi.fn(),
    createCampaign: vi.fn(),
    resolveItem: vi.fn(),
    applyItem: vi.fn(),
    reauditCampaign: vi.fn(),
    closeCampaign: vi.fn(),
  },
}));

import { campaignClient } from "../../api/campaign";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { CampaignWorkbench } from "./CampaignWorkbench";
import { makeCampaign, makeCampaignStatus, makeCampaignItem } from "./flow/fixtures";

afterEach(() => {
  cleanup();
  vi.mocked(campaignClient.listCampaigns).mockReset();
  vi.mocked(campaignClient.getCampaignStatus).mockReset();
  vi.mocked(campaignClient.nextCampaignStep).mockReset();
});

type ListResult = Awaited<ReturnType<typeof campaignClient.listCampaigns>>;
type StatusResult = Awaited<ReturnType<typeof campaignClient.getCampaignStatus>>;
type NextResult = Awaited<ReturnType<typeof campaignClient.nextCampaignStep>>;

describe("CampaignWorkbench", () => {
  it("renders the campaign list on the primary side and an empty detail prompt on the secondary side", async () => {
    vi.mocked(campaignClient.listCampaigns).mockResolvedValue({
      campaigns: [],
    } as unknown as ListResult);

    renderWithProviders(<CampaignWorkbench scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.campaign.workbench.root)).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(selectors.features.campaign.workbench.emptyDetail),
    ).toBeInTheDocument();
  });

  it("renders the detail panel with a worklist when campaignId is provided", async () => {
    vi.mocked(campaignClient.listCampaigns).mockResolvedValue({
      campaigns: [makeCampaign({ id: "m-1" })],
    } as unknown as ListResult);
    vi.mocked(campaignClient.getCampaignStatus).mockResolvedValue({
      status: makeCampaignStatus(),
    } as unknown as StatusResult);
    vi.mocked(campaignClient.nextCampaignStep).mockResolvedValue({
      items: [makeCampaignItem()],
    } as unknown as NextResult);

    renderWithProviders(<CampaignWorkbench scenario="demo" campaignId="m-1" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.campaign.detail.root)).toBeInTheDocument(),
    );
    expect(
      screen.getByTestId(
        selectors.features.campaign.detail.itemCard({ stableId: "afid:abc12345" }),
      ),
    ).toBeInTheDocument();
  });
});
