import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  FileChangeKind,
  FreshnessState,
  ProposalSchema,
  ProposalState,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";

// One open proposal shaped like an accepted epoch, for proposal UI tests.
type ProposalInit = Exclude<MessageInitShape<typeof ProposalSchema>, Proposal>;

export function proposalFixture(overrides: ProposalInit = {}): Proposal {
  return create(ProposalSchema, {
    id: "gctp-0123456789ab",
    repositoryId: "1",
    state: ProposalState.OPEN,
    revision: 2,
    branch: "agi",
    baseHead: "0e1f1210438aaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    createdBy: { subject: "orchestrator", kind: "vrooli-agent", verified: true },
    work: { effortRef: "effort:browser-automation-studio-rehabilitation", epoch: "E27", runIds: ["70dd81be-0c15-4ef3-9b05-e18aa6ca7ccb"] },
    message: {
      subject: "bas: one ReplaySpec renderer (E27)",
      body: "Standalone replay archives use one renderer.",
      rendered: "bas: one ReplaySpec renderer (E27)\n\nStandalone replay archives use one renderer.\n\nVrooli-Epoch: effort:browser-automation-studio-rehabilitation#E27",
      trailers: [{ key: "Vrooli-Epoch", value: "effort:browser-automation-studio-rehabilitation#E27", kind: "epoch", known: true }],
    },
    files: [
      { path: "scenarios/bas/api/replay.go", kind: FileChangeKind.MODIFIED, flags: [{ code: "mixed_prior_uncommitted" }] },
      { path: "scenarios/bas/api/old_template.go", kind: FileChangeKind.DELETED, deleted: true, flags: [] },
    ],
    freshness: { state: FreshnessState.FRESH },
    ...overrides,
  });
}
