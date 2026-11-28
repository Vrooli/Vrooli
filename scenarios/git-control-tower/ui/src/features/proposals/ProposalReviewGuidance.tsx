import { ProposalGuidance, type GuidanceAction } from "./ProposalGuidance";

export type ReviewBlocker = {
  kind:
    | "operator"
    | "comparison"
    | "closed"
    | "legacy"
    | "diff"
    | "draft"
    | "prior"
    | "staging"
    | "base"
    | "status";
  message: string;
};
export type ProposalGuide = "blocked" | "version" | "update";

/** One ordered blocker drives both the button explanation and its next step. */
export function reviewBlocker(input: {
  canApprove: boolean;
  operatorReason?: string;
  comparing: boolean;
  open: boolean;
  historical: boolean;
  captured: boolean;
  diffReason?: string;
  dirty: boolean;
  needsAcknowledgment: boolean;
  differentStaged: boolean;
  baseChanged: boolean;
  reviewable: boolean;
  foreignStaged: boolean;
}): ReviewBlocker | undefined {
  if (!input.canApprove)
    return {
      kind: "operator",
      message:
        input.operatorReason || "Only the operator can approve proposals",
    };
  if (!input.open)
    return {
      kind: "closed",
      message: input.historical
        ? "Viewing a retained revision"
        : "This proposal is closed",
    };
  if (input.comparing)
    return {
      kind: "comparison",
      message: "Show the proposal version before committing",
    };
  if (!input.captured)
    return {
      kind: "legacy",
      message: "Choose the latest working files for this older proposal",
    };
  if (input.diffReason) return { kind: "diff", message: input.diffReason };
  if (input.dirty) return { kind: "draft", message: "Save your edits first" };
  if (input.needsAcknowledgment)
    return {
      kind: "prior",
      message: "Review and acknowledge the earlier or unattributed work",
    };
  if (input.differentStaged)
    return { kind: "staging", message: "Different file versions are staged" };
  if (input.baseChanged)
    return { kind: "base", message: "New commits changed proposal files" };
  if (!input.reviewable)
    return {
      kind: "status",
      message: "Check the proposal status before committing",
    };
  if (input.foreignStaged)
    return { kind: "staging", message: "Unstage the other staged files first" };
}

export function ProposalReviewGuidance({
  guide,
  blocker,
  busy,
  error,
  captured,
  open,
  priorPaths,
  stagedPaths,
  onClose,
  onUpdateGuide,
  onUpdate,
  onSave,
  onAcknowledge,
  onCompare,
  onReturn,
  onWorkingChanges,
  onRetry,
}: {
  guide: ProposalGuide;
  blocker?: ReviewBlocker;
  busy: boolean;
  error?: string;
  captured: boolean;
  open: boolean;
  priorPaths: string[];
  stagedPaths: string[];
  onClose: () => void;
  onUpdateGuide: () => void;
  onUpdate: () => void;
  onSave?: () => void;
  onAcknowledge: () => void;
  onCompare?: () => void;
  onReturn?: () => void;
  onWorkingChanges?: () => void;
  onRetry?: () => void;
}) {
  const closeAfter = (action: () => void) => () => {
    action();
    onClose();
  };
  const action = (label: string, callback?: () => void): GuidanceAction[] =>
    callback ? [{ label, onClick: closeAfter(callback) }] : [];
  let actions: GuidanceAction[] = [];
  if (guide === "update")
    actions = [{ label: "Update proposal", onClick: onUpdate }];
  else if (guide === "version")
    actions = [
      ...(captured ? action("Compare later edits", onCompare) : []),
      ...(open
        ? [{ label: "Use latest working files", onClick: onUpdateGuide }]
        : []),
    ];
  else if (!busy) {
    switch (blocker?.kind) {
      case "comparison":
        actions = action("Show proposal version", onReturn);
        break;
      case "legacy":
      case "base":
        actions = [
          { label: "Use latest working files", onClick: onUpdateGuide },
        ];
        break;
      case "draft":
        actions = onSave ? [{ label: "Save message", onClick: onSave }] : [];
        break;
      case "prior":
        actions = action("Include these whole files", onAcknowledge);
        break;
      case "staging":
        actions = action("Review staged changes", onWorkingChanges);
        break;
      case "diff":
      case "status":
        actions = action("Check status again", onRetry);
        break;
      case "closed":
        actions = action("Return to current review", onRetry);
        break;
      case "operator":
        break;
      default:
        actions = action("Return to confirmation", () => {});
    }
  }
  const title =
    guide === "version"
      ? "Which file version gets committed?"
      : guide === "update"
        ? "Update proposal from working files?"
        : busy
          ? "An action is running"
          : "Before you create this commit";
  const descriptions: Record<ReviewBlocker["kind"], string> = {
    operator:
      "Approval requires an operator session. You can continue reviewing the saved files here.",
    comparison:
      "You are viewing edits made after the proposal. Show the proposal version to review exactly what this commit will include.",
    closed:
      "This historical record stays available to review. Choose the current proposal or return to the inbox to continue.",
    legacy:
      "The original contents of this older proposal were not saved. Use the latest working files to create a reviewable proposal version.",
    diff: "The saved diff must load before approval. Select a file to review, or check status again to retry the request.",
    draft:
      "Save your message edits before requesting approval. Your unsaved text stays here. The subject must not be blank.",
    prior:
      "These files also contain earlier or unattributed work. Committing a whole file includes all of its reviewed contents. Confirm that you want to include them.",
    staging:
      "Review staging in Working changes. Unstage the files you want to leave out, then return here. Unstaging keeps code in your working files; it does not delete it. Different staged versions are kept intact until you decide what to do with them.",
    base: "Changes were already committed to these files after the proposal was made. Update and review the proposal to avoid replacing that committed work.",
    status:
      "The current branch or proposal status could not be confirmed. Check status again before requesting approval.",
  };
  const paths =
    blocker?.kind === "prior"
      ? priorPaths
      : blocker?.kind === "staging"
        ? stagedPaths
        : [];
  return (
    <ProposalGuidance
      error={error}
      title={title}
      onClose={onClose}
      pending={busy}
      actions={actions}
    >
      {guide === "version" ? (
        <>
          <p>
            {captured
              ? "A proposal saves the file versions you are reviewing. Creating the commit records exactly those versions and the message above."
              : "This older proposal names files but has no saved file contents. Choose the latest working files, then review the new proposal version before committing."}
          </p>
          <p>
            Edits made afterward stay in your working files, uncommitted.
            Nothing is deleted or reverted.
          </p>
          <p>
            If you want the latest edits in this commit, choose{" "}
            <strong>Use latest working files</strong>. That updates the proposal
            and opens a new version for review.
          </p>
        </>
      ) : guide === "update" ? (
        <>
          <p>
            This saves the latest working file contents as a new proposal
            version. It updates the review record; it does not change your code,
            staged changes, or Git history.
          </p>
          <p>
            Review the new diff before committing. The previous proposal version
            stays in its history.
          </p>
        </>
      ) : busy ? (
        <p>
          The current save, check, or commit must finish before another action
          can start. Your review text stays here.
        </p>
      ) : (
        <>
          <p className="font-medium text-amber-200">
            {blocker?.message || "Confirm the preview above to continue."}
          </p>
          {blocker && <p>{descriptions[blocker.kind]}</p>}
          {paths.length > 0 && (
            <ul className="max-h-40 overflow-auto break-all text-xs">
              {paths.map((path) => (
                <li key={path}>{path}</li>
              ))}
            </ul>
          )}
        </>
      )}
    </ProposalGuidance>
  );
}
