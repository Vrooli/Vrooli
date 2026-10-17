import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { FindingSeverity, FindingSource } from "@vrooli/proto-types/architecture/v1/findings_pb";

vi.mock("../../api/campaign", () => ({
  campaignClient: {
    createCampaign: vi.fn(),
  },
}));

import { campaignClient } from "../../api/campaign";
import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { CreateCampaignForm } from "./CreateCampaignForm";

afterEach(() => {
  cleanup();
  vi.mocked(campaignClient.createCampaign).mockReset();
});

function reportWithOneFinding(): string {
  return JSON.stringify({
    phases: [
      {
        name: "architecture",
        findings: [
          {
            scenario: "demo",
            source: FindingSource.ARCHITECTURE,
            code: "cycle/cross-domain",
            severity: FindingSeverity.BLOCKER,
            locations: ["api/a.go", "api/b.go"],
            domains: ["a", "b"],
            message: "import cycle",
          },
        ],
      },
    ],
  });
}

describe("CreateCampaignForm", () => {
  it("keeps submit disabled until the pasted report parses", async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <CreateCampaignForm scenario="demo" onCreated={vi.fn()} onCancel={vi.fn()} />,
    );

    const submit = screen.getByTestId(selectors.features.campaign.create.submit);
    expect(submit).toBeDisabled();

    await user.click(screen.getByTestId(selectors.features.campaign.create.reportInput));
    await user.paste("{not json");

    expect(screen.getByTestId(selectors.features.campaign.create.parseError)).toBeInTheDocument();
    expect(submit).toBeDisabled();
    expect(campaignClient.createCampaign).not.toHaveBeenCalled();
  });

  it("submits parsed findings and reports the created campaign id", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    vi.mocked(campaignClient.createCampaign).mockResolvedValue({
      status: { campaign: { id: "camp-1" } },
    } as Awaited<ReturnType<typeof campaignClient.createCampaign>>);

    renderWithProviders(
      <CreateCampaignForm scenario="demo" onCreated={onCreated} onCancel={vi.fn()} />,
    );

    await user.type(screen.getByTestId(selectors.features.campaign.create.nameInput), "Audit drift");
    await user.click(screen.getByTestId(selectors.features.campaign.create.reportInput));
    await user.paste(reportWithOneFinding());

    expect(screen.getByTestId(selectors.features.campaign.create.parsedCount)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.features.campaign.create.submit));

    await waitFor(() => expect(campaignClient.createCampaign).toHaveBeenCalledTimes(1));
    expect(campaignClient.createCampaign).toHaveBeenCalledWith(
      expect.objectContaining({
        scenario: "demo",
        name: "Audit drift",
        findings: [
          expect.objectContaining({
            scenario: "demo",
            code: "cycle/cross-domain",
            locations: ["api/a.go", "api/b.go"],
          }),
        ],
      }),
    );
    await waitFor(() => expect(onCreated).toHaveBeenCalledWith("camp-1"));
  });

  it("invokes cancel without submitting a mutation", async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();

    renderWithProviders(
      <CreateCampaignForm scenario="demo" onCreated={vi.fn()} onCancel={onCancel} />,
    );

    await user.click(screen.getByTestId(selectors.features.campaign.create.cancel));

    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(campaignClient.createCampaign).not.toHaveBeenCalled();
  });
});
