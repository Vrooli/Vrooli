// [REQ:GCT-PROPOSAL-WORKSPACE]
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ProposalState } from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { proposalFixture } from "../../test-utils/proposals";
import * as api from "../../lib/api-proposals";
import { ProposalWorkspace } from "./ProposalWorkspace";

const device = vi.hoisted(() => ({ mobile: false }));
vi.mock("../../hooks", () => ({ useIsMobile: () => device.mobile }));
vi.mock("../../components/DiffViewer", () => ({
  DiffViewer: ({
    diff,
    selectedFile,
    isProposalMode,
  }: {
    diff?: { raw?: string };
    selectedFile?: string;
    isProposalMode?: boolean;
  }) => (
    <div
      data-testid="captured-diff"
      data-proposal-mode={String(isProposalMode)}
    >
      {selectedFile} {diff?.raw}
    </div>
  ),
}));
vi.mock("@tanstack/react-virtual", () => ({
  useVirtualizer: ({ count }: { count: number }) => ({
    getTotalSize: () => count * 64,
    getVirtualItems: () =>
      Array.from({ length: count }, (_, index) => ({
        index,
        start: index * 64,
        size: 64,
      })),
    scrollToIndex: vi.fn(),
  }),
}));
vi.mock("../../lib/api-proposals", async (load) => ({
  ...(await load<typeof import("../../lib/api-proposals")>()),
  listProposals: vi.fn(),
  getProposal: vi.fn(),
  getProposalDiff: vi.fn(),
  combineProposals: vi.fn(),
  editProposal: vi.fn(),
  withdrawProposal: vi.fn(),
}));

const props = {
  repoId: "1",
  proposalId: "gctp-0123456789ab",
  canApprove: true,
  onOpen: vi.fn(),
  onSelectPath: vi.fn(),
  onExit: vi.fn(),
};

beforeEach(() => {
  device.mobile = false;
  vi.clearAllMocks();
  window.sessionStorage.clear();
  const proposal = proposalFixture();
  vi.mocked(api.getProposal).mockResolvedValue(proposal);
  vi.mocked(api.listProposals).mockResolvedValue({
    proposals: [proposal],
    openCount: 1,
  } as Awaited<ReturnType<typeof api.listProposals>>);
  vi.mocked(api.getProposalDiff).mockImplementation(
    async (_repo, _id, _revision, path, current) => ({
      repo_dir: "/repo",
      path,
      staged: false,
      has_diff: true,
      stats: { additions: 1, deletions: 1, files: 1 },
      raw: current
        ? "unreviewed current contents"
        : "retained reviewed contents",
      timestamp: "",
    }),
  );
});

describe("ProposalWorkspace", () => {
  it("starts with an exact captured diff and keeps working-tree actions out of proposal mode", async () => {
    renderWithProviders(<ProposalWorkspace {...props} />);
    expect(
      await screen.findByText(/retained reviewed contents/),
    ).toBeInTheDocument();
    expect(api.getProposalDiff).toHaveBeenCalledWith(
      "1",
      props.proposalId,
      2,
      "scenarios/bas/api/replay.go",
      false,
    );
    expect(screen.getByTestId("captured-diff")).toHaveAttribute(
      "data-proposal-mode",
      "true",
    );
    expect(
      screen.getByRole("complementary", { name: "Proposed files" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "old_template.go" }));
    expect(props.onSelectPath).toHaveBeenCalledWith(
      "scenarios/bas/api/old_template.go",
    );
    expect(
      screen.queryByRole("button", { name: /^discard/i }),
    ).not.toBeInTheDocument();
  });

  it("makes comparing current work explicit and blocks approval while comparing", async () => {
    renderWithProviders(<ProposalWorkspace {...props} />);
    await screen.findByText(/retained reviewed contents/);
    fireEvent.click(
      screen.getByRole("button", { name: "Compare later edits" }),
    );
    await screen.findByText(/unreviewed current contents/);
    expect(screen.getByTestId("proposal-approve-blocker")).toHaveTextContent(
      "Show the proposal version before committing",
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Show proposal version" }),
    );
    await screen.findByText(/retained reviewed contents/);
  });

  it("never substitutes live bytes when a legacy capture is unavailable", async () => {
    vi.mocked(api.getProposal).mockResolvedValue(
      proposalFixture({ captured: false }),
    );
    renderWithProviders(<ProposalWorkspace {...props} />);
    expect(
      await screen.findByTestId("proposal-capture-unavailable"),
    ).toHaveTextContent("original file contents were not saved");
    expect(api.getProposalDiff).not.toHaveBeenCalled();
    expect(screen.getByTestId("proposal-approve")).toBeDisabled();
  });

  it("supports the mobile overview → files → diff journey without losing identity", async () => {
    device.mobile = true;
    renderWithProviders(<ProposalWorkspace {...props} />);
    await screen.findByTestId("proposal-detail");
    fireEvent.click(screen.getByTestId("proposal-tab-files"));
    fireEvent.click(screen.getByRole("button", { name: "old_template.go" }));
    expect(screen.getByTestId("proposal-tab-diff")).toHaveAttribute(
      "aria-current",
      "page",
    );
    expect(props.onSelectPath).toHaveBeenCalledWith(
      "scenarios/bas/api/old_template.go",
    );
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent(
      "bas: one ReplaySpec renderer",
    );
    fireEvent.click(screen.getByRole("button", { name: "Review & commit" }));
    expect(screen.getByTestId("proposal-detail")).toBeInTheDocument();
  });

  it("excludes a file with an exact revision while explaining that code is kept", async () => {
    vi.mocked(api.editProposal).mockResolvedValue(
      proposalFixture({ revision: 3 }),
    );
    renderWithProviders(<ProposalWorkspace {...props} />);
    await screen.findByTestId("proposal-files");
    expect(
      screen.getByText("Excluding a file leaves its code uncommitted."),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Leave replay.go uncommitted" }),
    );
    await waitFor(() =>
      expect(api.editProposal).toHaveBeenCalledWith({
        id: props.proposalId,
        expectedRevision: 2,
        removePaths: ["scenarios/bas/api/replay.go"],
      }),
    );
  });

  it("shows closed reviews in the inbox and combines only explicitly selected sources", async () => {
    const first = proposalFixture();
    const second = proposalFixture({
      id: "gctp-abcdef123456",
      message: { subject: "Second reviewed work" },
      revision: 7,
    });
    const closed = proposalFixture({
      id: "gctp-000000000000",
      state: ProposalState.WITHDRAWN,
      message: { subject: "Closed work" },
    });
    vi.mocked(api.listProposals).mockResolvedValue({
      proposals: [first, second, closed],
      openCount: 2,
    } as Awaited<ReturnType<typeof api.listProposals>>);
    vi.mocked(api.combineProposals).mockResolvedValue(
      proposalFixture({ id: "gctp-111111111111" }),
    );
    renderWithProviders(<ProposalWorkspace {...props} proposalId="inbox" />);
    await screen.findByText("Second reviewed work");
    expect(screen.queryByText("Closed work")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("checkbox", { name: /Select bas:/ }));
    fireEvent.click(screen.getByRole("checkbox", { name: /Select Second/ }));
    fireEvent.click(screen.getByTestId("proposal-combine"));
    expect(screen.getByTestId("proposal-combine-preview")).toHaveTextContent(
      "r7",
    );
    expect(api.combineProposals).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("proposal-combine-confirm"));
    await waitFor(() =>
      expect(api.combineProposals).toHaveBeenCalledWith(
        [first, second],
        "Review combined work",
        "",
      ),
    );
    expect(props.onOpen).toHaveBeenCalledWith("gctp-111111111111", undefined);
  });
});

it("moves keyboard focus with proposal file selection", async () => {
  renderWithProviders(<ProposalWorkspace {...props} />);
  const files = await screen.findAllByTestId("proposal-file");
  const first = files[0],
    next = files[1];
  if (!first || !next) throw new Error("Expected two proposed files");
  first.focus();
  fireEvent.keyDown(first, { key: "ArrowDown" });
  await waitFor(() => expect(next).toHaveFocus());
  fireEvent.keyDown(next, { key: "ArrowUp" });
  await waitFor(() => expect(first).toHaveFocus());
});
