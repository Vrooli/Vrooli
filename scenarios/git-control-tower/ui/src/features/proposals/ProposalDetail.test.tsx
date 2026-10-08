import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { MutationPreviewSchema } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import { ApplyProposalResponseSchema, FreshnessState } from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import * as api from "../../lib/api-proposals";
import { ProposalDetail } from "./ProposalDetail";
import { proposalFixture } from "../../test-utils/proposals";

vi.mock("../../lib/api-proposals", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../lib/api-proposals")>();
  return {
    ...actual,
    editProposal: vi.fn(),
    withdrawProposal: vi.fn(),
    refreshProposal: vi.fn(),
    prepareProposalApproval: vi.fn(),
    applyApprovedProposal: vi.fn(),
  };
});

const preview = create(MutationPreviewSchema, { repositoryId: "1", operation: "repo.apply_proposal", branch: "agi", expectedRevision: "0e1f1210438", subjectDigest: "digest-1" });

function renderDetail(overrides: Parameters<typeof proposalFixture>[0] = {}, props: Partial<Parameters<typeof ProposalDetail>[0]> = {}) {
  const handlers = { onBack: vi.fn(), onOpenDiff: vi.fn(), onCommitted: vi.fn() };
  renderWithProviders(<ProposalDetail proposal={proposalFixture(overrides)} repoId="1" canApprove {...handlers} {...props} />);
  return handlers;
}

beforeEach(() => {
  vi.clearAllMocks();
  window.sessionStorage.clear();
});

describe("ProposalDetail", () => {
  it("lists files with flags, opens a file's diff and shows trailer chips", () => {
    const { onOpenDiff } = renderDetail();
    const files = screen.getByTestId("proposal-files");
    expect(files).toHaveTextContent("scenarios/bas/api/replay.go");
    expect(files).toHaveTextContent("mixed");
    expect(files).toHaveTextContent("D");
    fireEvent.click(screen.getByRole("button", { name: "scenarios/bas/api/old_template.go" }));
    expect(onOpenDiff).toHaveBeenCalledWith("scenarios/bas/api/old_template.go");
    expect(screen.getByTestId("proposal-trailers")).toHaveTextContent("Vrooli-Epoch: effort:browser-automation-studio-rehabilitation#E27");
  });

  it("saves message edits against the reviewed revision and blocks approval until saved", async () => {
    vi.mocked(api.editProposal).mockResolvedValue(proposalFixture({ revision: 3, message: { subject: "bas: clearer subject (E27)" } }));
    renderDetail();
    fireEvent.change(screen.getByTestId("proposal-subject"), { target: { value: "bas: clearer subject (E27)" } });
    expect(screen.getByTestId("proposal-approve")).toBeDisabled();
    expect(screen.getByTestId("proposal-approve-blocker")).toHaveTextContent(/save your edits/i);
    fireEvent.click(screen.getByTestId("proposal-save"));
    await waitFor(() => expect(api.editProposal).toHaveBeenCalled());
    expect(vi.mocked(api.editProposal).mock.calls[0]?.[0]).toEqual(expect.objectContaining({ id: "gctp-0123456789ab", expectedRevision: 2, subject: "bas: clearer subject (E27)" }));
  });

  it("approves as one reviewed action: exact preview, then a single apply", async () => {
    vi.mocked(api.prepareProposalApproval).mockResolvedValue(preview);
    vi.mocked(api.applyApprovedProposal).mockResolvedValue(create(ApplyProposalResponseSchema, { success: true, commitOid: "abc123def4567", commitVerified: true }));
    const { onCommitted } = renderDetail();

    fireEvent.click(screen.getByTestId("proposal-approve"));
    expect(await screen.findByTestId("proposal-confirm")).toHaveTextContent("Stage exactly 2 files");
    expect(api.applyApprovedProposal).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("proposal-authorize"));
    expect(await screen.findByTestId("proposal-committed")).toHaveTextContent("Committed abc123def4");
    expect(api.applyApprovedProposal).toHaveBeenCalledTimes(1);
    expect(api.applyApprovedProposal).toHaveBeenCalledWith(expect.objectContaining({ id: "gctp-0123456789ab", revision: 2 }), preview, { skipPrecommit: false });
    expect(onCommitted).toHaveBeenCalledWith("abc123def4567");
  });

  it("applies without the confirm step when the device skips commit confirmation", async () => {
    vi.mocked(api.prepareProposalApproval).mockResolvedValue(preview);
    vi.mocked(api.applyApprovedProposal).mockResolvedValue(create(ApplyProposalResponseSchema, { success: true, commitOid: "abc", commitVerified: true }));
    renderDetail({}, { skipConfirmation: true });
    fireEvent.click(screen.getByTestId("proposal-approve"));
    expect(await screen.findByTestId("proposal-committed")).toBeInTheDocument();
    expect(screen.queryByTestId("proposal-confirm")).not.toBeInTheDocument();
  });

  it("shows a drift refusal with every file it names", async () => {
    vi.mocked(api.prepareProposalApproval).mockResolvedValue(preview);
    vi.mocked(api.applyApprovedProposal).mockResolvedValue(create(ApplyProposalResponseSchema, {
      refusal: { code: "content_drift", paths: ["scenarios/bas/api/replay.go", "scenarios/bas/api/old_template.go"], detail: "content changed since the proposal; refresh it or remove the files" },
    }));
    renderDetail({}, { skipConfirmation: true });
    fireEvent.click(screen.getByTestId("proposal-approve"));
    const refused = await screen.findByTestId("proposal-refused");
    expect(refused).toHaveTextContent("Not committed: content drift");
    expect(refused).toHaveTextContent("scenarios/bas/api/replay.go");
    expect(refused).toHaveTextContent("scenarios/bas/api/old_template.go");
  });

  it("requires a refresh before approving a drifted proposal", async () => {
    vi.mocked(api.refreshProposal).mockResolvedValue(proposalFixture({ revision: 3 }));
    renderDetail({ freshness: { state: FreshnessState.DRIFTED, driftedPaths: ["scenarios/bas/api/replay.go"] } });
    expect(screen.getByTestId("proposal-stale")).toHaveTextContent("scenarios/bas/api/replay.go");
    expect(screen.getByTestId("proposal-approve")).toBeDisabled();
    fireEvent.click(screen.getByTestId("proposal-refresh"));
    await waitFor(() => expect(api.refreshProposal).toHaveBeenCalledWith("gctp-0123456789ab", 2));
  });

  it("never offers approval to callers without human authority", () => {
    renderDetail({}, { canApprove: false, approveDisabledReason: "agent callers may inspect and prepare changes" });
    expect(screen.getByTestId("proposal-approve")).toBeDisabled();
    expect(screen.getByTestId("proposal-approve-blocker")).toHaveTextContent("agent callers may inspect");
  });

  it("withdraws only after an explicit second click", async () => {
    vi.mocked(api.withdrawProposal).mockResolvedValue(proposalFixture());
    const { onBack } = renderDetail();
    fireEvent.click(screen.getByTestId("proposal-withdraw"));
    expect(api.withdrawProposal).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("proposal-withdraw-confirm"));
    await waitFor(() => expect(api.withdrawProposal).toHaveBeenCalledWith("gctp-0123456789ab", "withdrawn by operator"));
    await waitFor(() => expect(onBack).toHaveBeenCalled());
  });

  // GCT-053: leaving the detail (mobile diff subview) and returning keeps edits.
  it("keeps unsaved edits when the view remounts", () => {
    const first = renderDetailOnce();
    fireEvent.change(screen.getByTestId("proposal-body"), { target: { value: "Operator wording." } });
    first.unmount();
    renderDetailOnce();
    expect(screen.getByTestId("proposal-body")).toHaveValue("Operator wording.");
  });
});

function renderDetailOnce() {
  return renderWithProviders(<ProposalDetail proposal={proposalFixture()} repoId="1" canApprove onBack={vi.fn()} onOpenDiff={vi.fn()} />);
}
