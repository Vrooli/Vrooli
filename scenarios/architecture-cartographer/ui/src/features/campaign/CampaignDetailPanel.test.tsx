import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  CampaignItemStatus,
  CampaignLifecycle,
} from "@vrooli/proto-types/architecture-cartographer/v1/campaign/campaign_pb";

vi.mock("./controllers/useCampaignController", () => ({
  RankProfile: {
    BALANCED: 1,
    FAST: 2,
    LONG_TERM: 3,
  },
  useApplyItem: vi.fn(),
  useCloseCampaign: vi.fn(),
  useCampaignStatus: vi.fn(),
  useNextStep: vi.fn(),
  useReauditCampaign: vi.fn(),
  useResolveItem: vi.fn(),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { CampaignDetailPanel } from "./CampaignDetailPanel";
import {
  useApplyItem,
  useCloseCampaign,
  useCampaignStatus,
  useNextStep,
  useReauditCampaign,
  useResolveItem,
} from "./controllers/useCampaignController";
import { makeCampaign, makeCampaignItem, makeCampaignStatus } from "./flow/fixtures";

afterEach(() => {
  cleanup();
  vi.mocked(useApplyItem).mockReset();
  vi.mocked(useCloseCampaign).mockReset();
  vi.mocked(useCampaignStatus).mockReset();
  vi.mocked(useNextStep).mockReset();
  vi.mocked(useReauditCampaign).mockReset();
  vi.mocked(useResolveItem).mockReset();
  vi.restoreAllMocks();
});

function mockDetailState({
  statusState,
  items = [],
  resolveMutate = vi.fn(),
  applyMutate = vi.fn(),
  reauditMutate = vi.fn(),
  closeMutate = vi.fn(),
  pending = false,
  reauditSuccess = false,
}: {
  statusState?: Partial<ReturnType<typeof useCampaignStatus>>;
  items?: ReturnType<typeof makeCampaignItem>[];
  resolveMutate?: ReturnType<typeof vi.fn>;
  applyMutate?: ReturnType<typeof vi.fn>;
  reauditMutate?: ReturnType<typeof vi.fn>;
  closeMutate?: ReturnType<typeof vi.fn>;
  pending?: boolean;
  reauditSuccess?: boolean;
} = {}) {
  vi.mocked(useCampaignStatus).mockReturnValue({
    isPending: false,
    isError: false,
    data: { status: makeCampaignStatus() },
    error: null,
    refetch: vi.fn(),
    ...statusState,
  } as unknown as ReturnType<typeof useCampaignStatus>);
  vi.mocked(useNextStep).mockReturnValue({
    data: { items },
  } as unknown as ReturnType<typeof useNextStep>);
  vi.mocked(useResolveItem).mockReturnValue({
    isPending: pending,
    mutate: resolveMutate,
  } as unknown as ReturnType<typeof useResolveItem>);
  vi.mocked(useApplyItem).mockReturnValue({
    isPending: pending,
    mutate: applyMutate,
  } as unknown as ReturnType<typeof useApplyItem>);
  vi.mocked(useReauditCampaign).mockReturnValue({
    isPending: pending,
    isSuccess: reauditSuccess,
    data: { validated: ["a"], stillOpen: ["b", "c"], regressions: ["d"] },
    mutate: reauditMutate,
  } as unknown as ReturnType<typeof useReauditCampaign>);
  vi.mocked(useCloseCampaign).mockReturnValue({
    isPending: pending,
    mutate: closeMutate,
  } as unknown as ReturnType<typeof useCloseCampaign>);
  return { resolveMutate, applyMutate, reauditMutate, closeMutate };
}

describe("CampaignDetailPanel", () => {
  it("renders loading, error, and not-found states", () => {
    mockDetailState({ statusState: { isPending: true } });
    const { rerender } = renderWithProviders(<CampaignDetailPanel scenario="demo" campaignId="m-1" />);
    expect(screen.getByTestId(selectors.features.campaign.detail.loading)).toBeInTheDocument();

    const refetch = vi.fn();
    mockDetailState({
      statusState: { isError: true, error: "campaign unavailable" as never, refetch },
    });
    rerender(<CampaignDetailPanel scenario="demo" campaignId="m-1" />);
    expect(screen.getByTestId(selectors.features.campaign.detail.error)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(refetch).toHaveBeenCalledTimes(1);

    mockDetailState({ statusState: { data: { status: undefined } as never } });
    rerender(<CampaignDetailPanel scenario="demo" campaignId="m-1" />);
    expect(screen.getByTestId(selectors.features.campaign.detail.notFound)).toBeInTheDocument();
  });

  it("renders open campaign items and dispatches item actions", () => {
    const promptSpy = vi.spyOn(window, "prompt").mockReturnValue("  done  ");
    const items = [
      makeCampaignItem({
        stableId: "item-detected",
        severity: "blocker",
        status: CampaignItemStatus.DETECTED,
        effort: "small",
        regressed: true,
        suggestion: "move it",
      }),
      makeCampaignItem({
        stableId: "item-assigned",
        severity: "error",
        status: CampaignItemStatus.ASSIGNED,
        locations: [],
        message: "",
      }),
      makeCampaignItem({ stableId: "item-split", severity: "warn", status: CampaignItemStatus.SPLIT }),
      makeCampaignItem({ stableId: "item-resolved", severity: "info", status: CampaignItemStatus.RESOLVED }),
      makeCampaignItem({ stableId: "item-validated", severity: "unknown", status: CampaignItemStatus.VALIDATED }),
      makeCampaignItem({ stableId: "item-committed", status: CampaignItemStatus.COMMITTED }),
      makeCampaignItem({ stableId: "item-force", status: CampaignItemStatus.FORCE_RESOLVED }),
    ];
    const { resolveMutate, applyMutate, closeMutate } = mockDetailState({
      statusState: {
        data: {
          status: makeCampaignStatus({
            campaign: makeCampaign({ status: CampaignLifecycle.OPEN }),
            total: 7,
            open: 4,
            resolved: 2,
            validated: 1,
            regressions: 1,
          }),
        } as never,
      },
      items,
    });

    renderWithProviders(<CampaignDetailPanel scenario="demo" campaignId="m-1" />);

    expect(screen.getByTestId(selectors.features.campaign.detail.root)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.campaign.detail.rollup)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.campaign.detail.itemCard({ stableId: "item-detected" }))).toBeInTheDocument();

    fireEvent.change(screen.getByTestId(selectors.features.campaign.detail.profileSelect), {
      target: { value: "2" },
    });
    expect(useNextStep).toHaveBeenLastCalledWith({ id: "m-1", profile: 2 });

    fireEvent.click(selectorsButton("item-detected", "resolve"));
    expect(promptSpy).toHaveBeenCalledTimes(1);
    expect(resolveMutate).toHaveBeenCalledWith({ stableId: "item-detected", note: "done" });

    fireEvent.click(selectorsButton("item-detected", "apply"));
    expect(applyMutate).toHaveBeenCalledWith("item-detected");

    fireEvent.click(screen.getByTestId(selectors.features.campaign.detail.closeButton));
    expect(closeMutate).toHaveBeenCalledTimes(1);
  });

  it("validates and submits reaudit reports", () => {
    const { reauditMutate } = mockDetailState({
      reauditSuccess: true,
      items: [makeCampaignItem()],
    });
    renderWithProviders(<CampaignDetailPanel scenario="demo" campaignId="m-1" />);

    fireEvent.change(screen.getByTestId(selectors.features.campaign.detail.reauditInput), {
      target: { value: "{not json" },
    });
    fireEvent.click(screen.getByTestId(selectors.features.campaign.detail.reauditSubmit));
    expect(screen.getByText(/Not valid JSON/)).toBeInTheDocument();
    expect(reauditMutate).not.toHaveBeenCalled();

    fireEvent.change(screen.getByTestId(selectors.features.campaign.detail.reauditInput), {
      target: {
        value: JSON.stringify({
          findings: [{ scenario: "demo", source: 1, code: "cycle", severity: 4 }],
        }),
      },
    });
    fireEvent.click(screen.getByTestId(selectors.features.campaign.detail.reauditSubmit));
    expect(reauditMutate).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId(selectors.features.campaign.detail.reauditResult)).toBeInTheDocument();
  });

  it("renders empty worklist and pending mutation labels", () => {
    mockDetailState({ pending: true, items: [] });

    renderWithProviders(<CampaignDetailPanel scenario="demo" campaignId="m-1" />);

    expect(screen.getByTestId(selectors.features.campaign.detail.worklistEmpty)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.features.campaign.detail.reauditSubmit)).toBeDisabled();
    expect(screen.getByTestId(selectors.features.campaign.detail.closeButton)).toBeDisabled();
    expect(screen.getByText("pages.campaign.detail.reauditting")).toBeInTheDocument();
    expect(screen.getByText("pages.campaign.detail.closing")).toBeInTheDocument();
  });

  it("renders a closed campaign without open mutation controls", () => {
    mockDetailState({
      statusState: {
        data: {
          status: makeCampaignStatus({
            campaign: makeCampaign({ status: CampaignLifecycle.CLOSED }),
          }),
        } as never,
      },
      items: [makeCampaignItem({ stableId: "closed-item" })],
    });

    renderWithProviders(<CampaignDetailPanel scenario="demo" campaignId="m-1" />);

    expect(screen.queryByTestId(selectors.features.campaign.detail.reaudit)).not.toBeInTheDocument();
    expect(screen.queryByTestId(selectors.features.campaign.detail.actionButton({ stableId: "closed-item", action: "resolve" }))).not.toBeInTheDocument();
    expect(screen.getByText("pages.campaign.detail.closedNote")).toBeInTheDocument();
  });
});

function selectorsButton(stableId: string, action: "resolve" | "apply") {
  return screen.getByTestId(selectors.features.campaign.detail.actionButton({ stableId, action }));
}
