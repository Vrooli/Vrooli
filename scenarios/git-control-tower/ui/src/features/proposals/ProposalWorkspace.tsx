import { useEffect, useMemo, useState } from "react";
import {
  ArrowLeft,
  FileCode2,
  GitPullRequestDraft,
  Loader2,
} from "lucide-react";
import { ProposalState } from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { useIsMobile } from "../../hooks";
import { DiffViewer } from "../../components/DiffViewer";
import { Badge } from "../../components/ui/badge";
import {
  useProposal,
  useProposalDiff,
  useProposals,
} from "../../lib/hooks-proposals";
import { ProposalFiles } from "./ProposalFiles";
import { ProposalInbox } from "./ProposalInbox";
import { ProposalDetail } from "./ProposalDetail";
import {
  describeFlag,
  effortLabel,
  freshnessMeta,
  proposalEpochs,
  epochsLabel,
} from "./model";

interface ProposalWorkspaceProps {
  repoId: string | null;
  proposalId: string;
  selectedPath?: string;
  revision?: number;
  canApprove: boolean;
  approveDisabledReason?: string;
  skipConfirmation?: boolean;
  operator?: string;
  onOpen: (id: string, revision?: number) => void;
  onSelectPath: (path: string) => void;
  onExit: () => void;
}

const allStates = [
  ProposalState.OPEN,
  ProposalState.COMMITTED,
  ProposalState.WITHDRAWN,
  ProposalState.SUPERSEDED,
];

/** Owns proposal navigation and composition, keeping it out of App orchestration. */
export function ProposalWorkspace(props: ProposalWorkspaceProps) {
  const isMobile = useIsMobile();
  const [page, setPage] = useState<"overview" | "files" | "diff">("overview");
  const [compareCurrent, setCompareCurrent] = useState(false);
  const inbox = props.proposalId === "inbox";
  const list = useProposals(props.repoId, { states: allStates });
  const current = useProposal(inbox ? null : props.proposalId);
  const archived = useProposal(
    inbox || !props.revision ? null : props.proposalId,
    props.revision,
  );
  const proposal = props.revision ? archived.data : current.data;
  const selectedPath = proposal?.files.some(
    (file) => file.path === props.selectedPath,
  )
    ? props.selectedPath
    : proposal?.files[0]?.path;
  const selectedFile = proposal?.files.find(
    (file) => file.path === selectedPath,
  );
  useEffect(() => setCompareCurrent(false), [proposal?.id, proposal?.revision]);
  const diff = useProposalDiff(proposal, selectedPath, compareCurrent);
  const related = useMemo(
    () =>
      list.data?.proposals.filter(
        (p) =>
          p.id !== proposal?.id &&
          p.work?.effortRef &&
          p.work.effortRef === proposal?.work?.effortRef,
      ) ?? [],
    [list.data, proposal],
  );
  const open = (id: string, revision?: number) => {
    props.onOpen(id, revision);
    setPage("overview");
  };
  const select = (path: string) => {
    props.onSelectPath(path);
    if (isMobile) setPage("diff");
  };
  const status = freshnessMeta(proposal?.freshness?.state);
  const query = props.revision ? archived : current;
  const inspector = proposal && (
    <ProposalDetail
      proposal={proposal}
      repoId={props.repoId}
      canApprove={props.canApprove}
      approveDisabledReason={props.approveDisabledReason}
      comparingCurrent={compareCurrent}
      diffDisabledReason={
        diff.isPending
          ? "Wait for the saved diff to load"
          : diff.isError
            ? "The saved diff could not be loaded"
            : !diff.data
              ? "Select a file to review its saved diff"
              : undefined
      }
      onCompareCurrent={() => setCompareCurrent(true)}
      onReturnToProposal={() => setCompareCurrent(false)}
      onOpenWorkingChanges={props.onExit}
      onRetryReview={() => {
        void query.refetch();
        void diff.refetch();
        void list.refetch();
      }}
      skipConfirmation={props.skipConfirmation}
      onBack={() => open("inbox")}
      onOpenRelated={open}
      relatedProposals={related}
      historical={Boolean(props.revision)}
    />
  );
  const files = proposal && (
    <ProposalFiles
      proposal={proposal}
      selectedPath={selectedPath}
      onSelect={select}
      repoId={props.repoId}
      historical={Boolean(props.revision)}
    />
  );
  const change = proposal?.captured ? (
    <div className="flex h-full min-h-0 flex-col">
      <div className="flex shrink-0 items-center justify-between gap-2 pb-2 text-xs text-slate-400">
        <span>
          {compareCurrent
            ? "Proposal version → latest working files (not part of this commit)"
            : "Changes included in this proposal"}
        </span>
        <button
          className="min-h-10 rounded border border-slate-700 px-3 text-cyan-300"
          onClick={() => setCompareCurrent(!compareCurrent)}
        >
          {compareCurrent ? "Show proposal version" : "Compare later edits"}
        </button>
      </div>
      <div className="mb-2 shrink-0 break-words text-xs text-slate-500">
        {selectedFile?.mode === "120000"
          ? "Symbolic link"
          : selectedFile?.mode === "100755"
            ? "Executable file"
            : selectedFile?.deleted
              ? "Deleted file"
              : "Regular file"}
        {selectedFile?.baseMode !== selectedFile?.mode && (
          <span>
            {" "}
            · mode {selectedFile?.baseMode || "absent"} →{" "}
            {selectedFile?.mode || "absent"}
          </span>
        )}
        {selectedFile?.flags.map((flag) => (
          <span key={flag.code} className="ml-2" title={flag.detail}>
            {describeFlag(flag.code)}
          </span>
        ))}
      </div>
      <div className="min-h-0 flex-1">
        <DiffViewer
          diff={diff.data}
          selectedFile={selectedPath}
          isStaged={false}
          isUntracked={false}
          isLoading={diff.isLoading}
          error={diff.error}
          viewMode="diff"
          onViewModeChange={() => undefined}
          isProposalMode
        />
      </div>
    </div>
  ) : (
    <div
      className="flex h-full flex-col items-center justify-center gap-3 px-6 text-center"
      data-testid="proposal-capture-unavailable"
    >
      <FileCode2 className="h-9 w-9 text-slate-600" />
      <h2 className="text-base font-medium text-slate-200">
        Choose files for this older proposal
      </h2>
      <p className="max-w-md text-sm leading-relaxed text-slate-400">
        The original file contents were not saved. Use the latest working files
        to update this proposal, then review its changes. Your files stay
        untouched.
      </p>
      {isMobile && (
        <button
          className="rounded-lg border border-slate-700 px-4 py-3 text-sm text-cyan-300"
          onClick={() => setPage("overview")}
        >
          Review details
        </button>
      )}
    </div>
  );

  return (
    <div
      className="flex h-full min-h-0 flex-col bg-slate-950 text-slate-100"
      data-testid="proposal-workspace"
    >
      <header className="shrink-0 border-b border-slate-800 px-4 pb-3 pt-[max(12px,env(safe-area-inset-top))] sm:px-6">
        <div className="flex items-center justify-between gap-3">
          <button
            className="inline-flex min-h-10 items-center gap-2 text-sm text-slate-400 hover:text-slate-100"
            onClick={inbox ? props.onExit : () => open("inbox")}
            data-testid="proposal-workspace-back"
          >
            <ArrowLeft className="h-4 w-4" />{" "}
            {inbox ? "Working changes" : "Proposal inbox"}
          </button>
          {!inbox && (
            <button
              className="min-h-10 text-xs text-slate-400 hover:text-slate-100"
              onClick={props.onExit}
            >
              Back to working changes
            </button>
          )}
          {inbox && <GitPullRequestDraft className="h-5 w-5 text-cyan-400" />}
        </div>
        <div className="mt-1 flex flex-wrap items-start justify-between gap-2">
          <h1 className="min-w-0 text-lg font-semibold leading-snug sm:text-xl">
            {inbox
              ? "Commit proposals"
              : proposal?.message?.subject || "Loading proposal…"}
          </h1>
          {proposal && (
            <Badge variant={proposal.captured ? status.tone : "warning"}>
              {props.revision
                ? `Retained r${props.revision}`
                : proposal.state === ProposalState.COMMITTED
                  ? "Committed"
                  : proposal.state === ProposalState.WITHDRAWN
                    ? "Closed"
                    : proposal.state === ProposalState.SUPERSEDED
                      ? "Replaced"
                      : !proposal.captured
                        ? "Needs file review"
                        : status.label}
            </Badge>
          )}
        </div>
        {proposal && (
          <div className="mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-slate-400">
            <span>
              {effortLabel(proposal.work?.effortRef ?? "") || "Combined work"}
            </span>
            {proposalEpochs(proposal).length > 0 && (
              <span>{epochsLabel(proposalEpochs(proposal))}</span>
            )}
            {proposal.commitOid && (
              <a
                className="text-cyan-300 hover:underline"
                href={`?commit=${encodeURIComponent(proposal.commitOid)}`}
              >
                View commit {proposal.commitOid.slice(0, 10)}
              </a>
            )}
            <span title={proposal.repositoryPath}>
              {proposal.repositoryName || `Repository ${proposal.repositoryId}`}{" "}
              · {proposal.branch || "No destination branch"} ·{" "}
              {proposal.files.length} files
            </span>
            {(props.revision ||
              (current.data?.retainedRevisions.length ?? 0) > 1) && (
              <label className="inline-flex items-center gap-1">
                Revision{" "}
                <select
                  value={props.revision ?? 0}
                  className="rounded bg-slate-900 px-2 py-1 text-slate-300"
                  aria-label="Proposal revision"
                  onChange={(event) =>
                    open(proposal.id, Number(event.target.value) || undefined)
                  }
                >
                  <option value={0}>Current · r{current.data?.revision}</option>
                  {(current.data?.retainedRevisions ?? []).map((r) => (
                    <option key={r} value={r}>
                      Retained r{r}
                    </option>
                  ))}
                </select>
              </label>
            )}
            <span className="ml-auto" title={props.operator}>
              {props.canApprove
                ? "Approving as operator"
                : "Read-only approval access"}
            </span>
          </div>
        )}
      </header>
      {inbox ? (
        <ProposalInbox
          proposals={list.data?.proposals ?? []}
          isLoading={list.isLoading}
          error={list.error}
          repoId={props.repoId}
          onOpen={open}
        />
      ) : query.error ? (
        <div className="p-6 text-sm text-red-300" role="alert">
          Proposal unavailable: {query.error.message}. Retained legacy revisions
          may not exist.
        </div>
      ) : !proposal ? (
        <div className="flex flex-1 items-center justify-center text-slate-400">
          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          Loading captured review…
        </div>
      ) : isMobile ? (
        <>
          <nav
            className="grid shrink-0 grid-cols-3 border-b border-slate-800 px-3"
            aria-label="Proposal views"
          >
            {(["overview", "files", "diff"] as const).map((item) => (
              <button
                key={item}
                onClick={() => setPage(item)}
                aria-current={page === item ? "page" : undefined}
                className={`min-h-11 border-b-2 py-3 text-sm capitalize ${page === item ? "border-cyan-400 text-cyan-300" : "border-transparent text-slate-400"}`}
                data-testid={`proposal-tab-${item}`}
              >
                {item === "files" ? `Files (${proposal.files.length})` : item}
              </button>
            ))}
          </nav>
          <div className="min-h-0 flex-1 overflow-hidden">
            {page === "overview"
              ? inspector
              : page === "files"
                ? files
                : change}
          </div>
          {page !== "overview" && (
            <div className="shrink-0 border-t border-slate-800 px-4 pb-[max(12px,env(safe-area-inset-bottom))] pt-3">
              <button
                className="min-h-11 w-full rounded-lg bg-blue-600 px-4 py-3 text-sm font-medium text-white"
                onClick={() => setPage("overview")}
              >
                Review &amp; commit
              </button>
            </div>
          )}
        </>
      ) : (
        <div className="grid min-h-0 flex-1 grid-cols-[minmax(220px,260px)_minmax(260px,1fr)_minmax(280px,360px)] gap-3 p-3">
          <aside
            className="min-h-0 overflow-hidden rounded-lg border border-slate-800 bg-slate-900/30"
            aria-label="Proposed files"
          >
            {files}
          </aside>
          <main className="min-h-0 min-w-0 overflow-hidden">{change}</main>
          <aside
            className="min-h-0 overflow-hidden rounded-lg border border-slate-800 bg-slate-900/30"
            aria-label="Proposal inspector"
          >
            {inspector}
          </aside>
        </div>
      )}
    </div>
  );
}
