// [REQ:GCT-PROPOSAL-WORKSPACE]
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { MutationPreviewSchema } from "@vrooli/proto-types/git-control-tower/v1/human_control/human_control_pb";
import {
  ApplyProposalResponseSchema,
  FreshnessState,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import * as api from "../../lib/api-proposals";
import { ProposalDetail } from "./ProposalDetail";
import { proposalFixture } from "../../test-utils/proposals";

vi.mock("../../lib/api-proposals", async (importOriginal) => {
  const actual =
    await importOriginal<typeof import("../../lib/api-proposals")>();
  return {
    ...actual,
    editProposal: vi.fn(),
    withdrawProposal: vi.fn(),
    refreshProposal: vi.fn(),
    prepareProposalApproval: vi.fn(),
    applyApprovedProposal: vi.fn(),
  };
});

const preview = create(MutationPreviewSchema, {
  repositoryId: "1",
  operation: "repo.apply_proposal",
  branch: "agi",
  expectedRevision: "0e1f1210438",
  subjectDigest: "digest-1",
});

function renderDetail(
  overrides: Parameters<typeof proposalFixture>[0] = {},
  props: Partial<Parameters<typeof ProposalDetail>[0]> = {},
) {
  const handlers = {
    onBack: vi.fn(),
    onOpenRelated: vi.fn(),
    onCommitted: vi.fn(),
  };
  renderWithProviders(
    <ProposalDetail
      proposal={proposalFixture(overrides)}
      repoId="1"
      canApprove
      {...handlers}
      {...props}
    />,
  );
  return handlers;
}

beforeEach(() => {
  vi.clearAllMocks();
  window.sessionStorage.clear();
});

describe("ProposalDetail", () => {
  it("explains a proposal that commits several folded epochs", () => {
    const effort = "effort:browser-automation-studio-rehabilitation";
    renderDetail({
      work: { effortRef: effort, epoch: "E33" },
      message: {
        subject: "bas: E33",
        trailers: ["E31", "E33"].map((epoch) => ({
          key: "Vrooli-Epoch",
          value: `${effort}#${epoch}`,
          kind: "epoch",
          known: true,
        })),
      },
    });
    expect(screen.getByTestId("proposal-folded")).toHaveTextContent(
      "One commit covers E31, E33",
    );
  });

  it("shows no fold note for a single-epoch proposal", () => {
    renderDetail();
    expect(screen.queryByTestId("proposal-folded")).not.toBeInTheDocument();
  });

  it("keeps technical metadata in a disclosure and makes earlier work explicit", () => {
    renderDetail();
    expect(screen.getByTestId("proposal-acknowledge-prior")).not.toBeChecked();
    expect(screen.getByTestId("proposal-approve")).toBeDisabled();
    expect(
      screen.getByText("Commit trailers & metadata").closest("details"),
    ).not.toHaveAttribute("open");
    fireEvent.click(screen.getByText("Commit trailers & metadata"));
    expect(screen.getByTestId("proposal-trailers")).toHaveTextContent(
      "Vrooli-Epoch: effort:browser-automation-studio-rehabilitation#E27",
    );
  });

  it("saves message edits against the reviewed revision and blocks approval until saved", async () => {
    vi.mocked(api.editProposal).mockResolvedValue(
      proposalFixture({
        revision: 3,
        message: { subject: "bas: clearer subject (E27)" },
      }),
    );
    renderDetail();
    fireEvent.change(screen.getByTestId("proposal-subject"), {
      target: { value: "bas: clearer subject (E27)" },
    });
    expect(screen.getByTestId("proposal-approve")).toBeDisabled();
    expect(screen.getByTestId("proposal-approve-blocker")).toHaveTextContent(
      /save your edits/i,
    );
    fireEvent.click(screen.getByTestId("proposal-save"));
    await waitFor(() => expect(api.editProposal).toHaveBeenCalled());
    expect(vi.mocked(api.editProposal).mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({
        id: "gctp-0123456789ab",
        expectedRevision: 2,
        subject: "bas: clearer subject (E27)",
      }),
    );
  });

  it("approves as one reviewed action: exact preview, then a single apply", async () => {
    vi.mocked(api.prepareProposalApproval).mockResolvedValue(preview);
    vi.mocked(api.applyApprovedProposal).mockResolvedValue(
      create(ApplyProposalResponseSchema, {
        success: true,
        commitOid: "abc123def4567",
        commitVerified: true,
      }),
    );
    const { onCommitted } = renderDetail();

    fireEvent.click(screen.getByTestId("proposal-acknowledge-prior"));
    fireEvent.click(screen.getByTestId("proposal-approve"));
    expect(await screen.findByTestId("proposal-confirm")).toHaveTextContent(
      "Commit the saved proposal version of exactly 2 files",
    );
    expect(api.applyApprovedProposal).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("proposal-authorize"));
    expect(await screen.findByTestId("proposal-committed")).toHaveTextContent(
      "Committed abc123def4",
    );
    expect(api.applyApprovedProposal).toHaveBeenCalledTimes(1);
    expect(api.applyApprovedProposal).toHaveBeenCalledWith(
      expect.objectContaining({ id: "gctp-0123456789ab", revision: 2 }),
      preview,
      {
        skipPrecommit: false,
        acknowledgePriorWork: true,
        keepLaterChanges: true,
      },
    );
    expect(onCommitted).toHaveBeenCalledWith("abc123def4567");
  });

  it("applies without the confirm step when the device skips commit confirmation", async () => {
    vi.mocked(api.prepareProposalApproval).mockResolvedValue(preview);
    vi.mocked(api.applyApprovedProposal).mockResolvedValue(
      create(ApplyProposalResponseSchema, {
        success: true,
        commitOid: "abc",
        commitVerified: true,
      }),
    );
    renderDetail({}, { skipConfirmation: true });
    fireEvent.click(screen.getByTestId("proposal-acknowledge-prior"));
    fireEvent.click(screen.getByTestId("proposal-approve"));
    expect(await screen.findByTestId("proposal-committed")).toBeInTheDocument();
    expect(screen.queryByTestId("proposal-confirm")).not.toBeInTheDocument();
  });

  it("shows a drift refusal with every file it names", async () => {
    vi.mocked(api.prepareProposalApproval).mockResolvedValue(preview);
    vi.mocked(api.applyApprovedProposal).mockResolvedValue(
      create(ApplyProposalResponseSchema, {
        refusal: {
          code: "content_drift",
          paths: [
            "scenarios/bas/api/replay.go",
            "scenarios/bas/api/old_template.go",
          ],
          detail:
            "content changed since the proposal; refresh it or remove the files",
        },
      }),
    );
    renderDetail({}, { skipConfirmation: true });
    fireEvent.click(screen.getByTestId("proposal-acknowledge-prior"));
    fireEvent.click(screen.getByTestId("proposal-approve"));
    const refused = await screen.findByTestId("proposal-refused");
    expect(refused).toHaveTextContent("Not committed: content drift");
    expect(refused).toHaveTextContent("scenarios/bas/api/replay.go");
    expect(refused).toHaveTextContent("scenarios/bas/api/old_template.go");
  });

  it("allows the saved version despite later edits and requires explicit confirmation to update it", async () => {
    vi.mocked(api.refreshProposal).mockResolvedValue(
      proposalFixture({ revision: 3 }),
    );
    renderDetail({
      freshness: {
        state: FreshnessState.DRIFTED,
        driftedPaths: ["scenarios/bas/api/replay.go"],
      },
    });
    expect(screen.getByTestId("proposal-stale")).toHaveTextContent(
      "scenarios/bas/api/replay.go",
    );
    fireEvent.click(screen.getByTestId("proposal-acknowledge-prior"));
    expect(screen.getByTestId("proposal-approve")).toBeEnabled();
    fireEvent.click(screen.getByTestId("proposal-refresh"));
    expect(api.refreshProposal).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Update proposal" }));
    await waitFor(() =>
      expect(api.refreshProposal).toHaveBeenCalledWith("gctp-0123456789ab", 2),
    );
  });

  it("never offers approval to callers without human authority", () => {
    renderDetail(
      {},
      {
        canApprove: false,
        approveDisabledReason: "agent callers may inspect and prepare changes",
      },
    );
    expect(screen.getByTestId("proposal-approve")).toBeDisabled();
    expect(screen.getByTestId("proposal-approve-blocker")).toHaveTextContent(
      "agent callers may inspect",
    );
  });

  it("withdraws only after an explicit second click", async () => {
    vi.mocked(api.withdrawProposal).mockResolvedValue(proposalFixture());
    const { onBack } = renderDetail();
    fireEvent.click(screen.getByTestId("proposal-withdraw"));
    expect(api.withdrawProposal).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("proposal-withdraw-confirm"));
    await waitFor(() =>
      expect(api.withdrawProposal).toHaveBeenCalledWith(
        "gctp-0123456789ab",
        "",
        2,
        "",
      ),
    );
    await waitFor(() => expect(onBack).toHaveBeenCalled());
  });

  // GCT-053: leaving the detail (mobile diff subview) and returning keeps edits.
  it("keeps unsaved edits when the view remounts", () => {
    const first = renderDetailOnce();
    fireEvent.change(screen.getByTestId("proposal-body"), {
      target: { value: "Operator wording." },
    });
    first.unmount();
    renderDetailOnce();
    expect(screen.getByTestId("proposal-body")).toHaveValue(
      "Operator wording.",
    );
  });
});

function renderDetailOnce() {
  return renderWithProviders(
    <ProposalDetail
      proposal={proposalFixture()}
      repoId="1"
      canApprove
      onBack={vi.fn()}
    />,
  );
}

it("blocks confirming an old preview after the operator changes the message", async () => {
  vi.mocked(api.prepareProposalApproval).mockResolvedValue(preview);
  renderDetail();
  fireEvent.click(screen.getByTestId("proposal-acknowledge-prior"));
  fireEvent.click(screen.getByTestId("proposal-approve"));
  await screen.findByTestId("proposal-confirm");
  fireEvent.change(screen.getByTestId("proposal-body"), {
    target: { value: "A new unsaved message" },
  });
  expect(screen.getByTestId("proposal-authorize")).toBeDisabled();
  fireEvent.click(screen.getByTestId("proposal-authorize"));
  expect(api.applyApprovedProposal).not.toHaveBeenCalled();
});

it("shows the retained message rather than a current operator draft in historical review", () => {
  window.sessionStorage.setItem(
    "gct.proposals.draft.gctp-0123456789ab",
    JSON.stringify({
      subject: "Unsaved current draft",
      body: "New words",
      revision: 4,
    }),
  );
  renderDetail({}, { historical: true });
  expect(screen.getByTestId("proposal-subject")).toHaveValue(
    "bas: one ReplaySpec renderer (E27)",
  );
  expect(screen.getByTestId("proposal-body")).toHaveValue(
    "Standalone replay archives use one renderer.",
  );
  expect(screen.queryByTestId("proposal-approve")).not.toBeInTheDocument();
  expect(
    window.sessionStorage.getItem("gct.proposals.draft.gctp-0123456789ab"),
  ).toContain("Unsaved current draft");
});

it("blocks approval while captured review cannot be loaded", () => {
  renderDetail(
    { files: [{ path: "review.go" }] },
    { diffDisabledReason: "The captured diff could not be loaded" },
  );
  expect(screen.getByTestId("proposal-approve")).toBeDisabled();
  expect(screen.getByTestId("proposal-approve-blocker")).toHaveTextContent(
    "The captured diff could not be loaded",
  );
});

it("previews omitted replacement paths before closing and preserves code", () => {
  const replacement = proposalFixture({
    id: "gctp-replacement",
    files: [{ path: "scenarios/bas/api/replay.go" }],
    message: { subject: "Later accepted work" },
  });
  renderDetail({}, { relatedProposals: [replacement] });
  fireEvent.click(screen.getByTestId("proposal-withdraw"));
  fireEvent.change(screen.getByLabelText("Replacement proposal"), {
    target: { value: replacement.id },
  });
  expect(screen.getByText(/1 path is absent/)).toBeInTheDocument();
  expect(
    screen.getByText("scenarios/bas/api/old_template.go"),
  ).toBeInTheDocument();
  expect(api.withdrawProposal).not.toHaveBeenCalled();
});

it("opens useful guidance from a blocked commit button by mouse or keyboard", () => {
  renderDetail();
  const wrapper = screen.getByTestId("proposal-approve-explain");
  expect(wrapper).toHaveAttribute("tabindex", "0");
  fireEvent.keyDown(wrapper, { key: "Enter" });
  expect(screen.getByRole("dialog")).toHaveTextContent(
    "These files also contain earlier or unattributed work",
  );
  expect(api.prepareProposalApproval).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Back to review" }));
  fireEvent.click(wrapper);
  fireEvent.click(
    screen.getByRole("button", { name: "Include these whole files" }),
  );
  expect(screen.getByTestId("proposal-acknowledge-prior")).toBeChecked();
  expect(screen.getByTestId("proposal-approve")).toBeEnabled();
});

it("explains saved versions and leaves the proposal unchanged when updating is cancelled", () => {
  renderDetail();
  fireEvent.click(
    screen.getByRole("button", { name: "What will this commit?" }),
  );
  expect(screen.getByRole("dialog")).toHaveTextContent(
    "Edits made afterward stay in your working files, uncommitted",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Use latest working files" }),
  );
  expect(screen.getByRole("dialog")).toHaveTextContent(
    "does not change your code, staged changes, or Git history",
  );
  fireEvent.click(screen.getByRole("button", { name: "Back to review" }));
  expect(api.refreshProposal).not.toHaveBeenCalled();
});

it("guides older proposals to choose working files rather than implying their original contents are saved", () => {
  renderDetail({ captured: false });
  fireEvent.click(screen.getByTestId("proposal-approve-explain"));
  expect(screen.getByRole("dialog")).toHaveTextContent(
    "original contents of this older proposal were not saved",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Use latest working files" }),
  );
  expect(
    screen.getByRole("button", { name: "Update proposal" }),
  ).toBeInTheDocument();
  expect(api.refreshProposal).not.toHaveBeenCalled();
});

it("returns from comparison to the proposal version through blocked-button guidance", () => {
  const onReturnToProposal = vi.fn();
  renderDetail({}, { comparingCurrent: true, onReturnToProposal });
  fireEvent.click(screen.getByTestId("proposal-approve-explain"));
  fireEvent.click(
    screen.getByRole("button", { name: "Show proposal version" }),
  );
  expect(onReturnToProposal).toHaveBeenCalledOnce();
  expect(api.prepareProposalApproval).not.toHaveBeenCalled();
});

it("offers staged-change review for different staged versions without overwriting them", () => {
  const onOpenWorkingChanges = vi.fn();
  renderDetail(
    { files: [{ path: "a.go", flags: [{ code: "staged_version_differs" }] }] },
    { onOpenWorkingChanges },
  );
  fireEvent.click(screen.getByTestId("proposal-approve-explain"));
  expect(screen.getByRole("dialog")).toHaveTextContent(
    "Different staged versions are kept intact",
  );
  fireEvent.click(
    screen.getByRole("button", { name: "Review staged changes" }),
  );
  expect(onOpenWorkingChanges).toHaveBeenCalledOnce();
  expect(api.prepareProposalApproval).not.toHaveBeenCalled();
});

it("does not offer a mutation next step to an agent or read-only caller", () => {
  renderDetail({ captured: false }, { canApprove: false });
  fireEvent.click(screen.getByTestId("proposal-approve-explain"));
  expect(screen.getByRole("dialog")).toHaveTextContent(
    "Approval requires an operator session",
  );
  expect(
    screen.queryByRole("button", { name: "Use latest working files" }),
  ).not.toBeInTheDocument();
});

it("still blocks when newer commits touched proposal files even when working edits also drifted", () => {
  renderDetail({
    files: [{ path: "a.go" }],
    freshness: { state: FreshnessState.DRIFTED, baseChangedPaths: ["a.go"] },
  });
  fireEvent.click(screen.getByTestId("proposal-approve-explain"));
  expect(screen.getByRole("dialog")).toHaveTextContent(
    "avoid replacing that committed work",
  );
  expect(api.prepareProposalApproval).not.toHaveBeenCalled();
});

it("keeps update failures visible inside the explanation dialog", async () => {
  vi.mocked(api.refreshProposal).mockRejectedValueOnce(
    new Error("The proposal changed; check status again"),
  );
  renderDetail({ captured: false });
  fireEvent.click(screen.getByTestId("proposal-approve-explain"));
  fireEvent.click(
    screen.getByRole("button", { name: "Use latest working files" }),
  );
  fireEvent.click(screen.getByRole("button", { name: "Update proposal" }));
  await waitFor(() =>
    expect(screen.getByRole("dialog")).toHaveTextContent(
      "The proposal changed; check status again",
    ),
  );
  expect(screen.getByRole("dialog")).toBeInTheDocument();
  expect(api.prepareProposalApproval).not.toHaveBeenCalled();
});
