import { useState } from "react";
import { act, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { create, fromJson } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import {
  CheckStagedMessageResponseSchema,
  OmissionSchema,
  StagedMessageFileSchema,
  StagedMessageCollectionSchema,
  StagedMessageCoverageSchema,
  StagedMessageScope,
  StagedMessageSnapshotSchema,
  StagedMessageJobSchema,
  StagedMessageSuggestionSchema,
  type CheckStagedMessageResponse,
  type StagedMessageJob,
} from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { CommitMessageComposer } from "./CommitMessageComposer";
import { mergeContinuationMessage, parseMessage } from "./message";
import { CommitPanel } from "../../components/CommitPanel";
import { useCommit } from "../../lib/hooks-core";
import { createCommit } from "../../lib/api";
import { CreateCommitResponseSchema } from "@vrooli/proto-types/git-control-tower/v1/repo/repo_pb";
import { cancelMessageJob, checkMessage, collectMessage, getMessageJob, startMessage, waitMessageJob } from "./api";

vi.mock("../../lib/api", async importOriginal => ({ ...await importOriginal<typeof import("../../lib/api")>(), createCommit: vi.fn() }));

vi.mock("./api", () => ({ collectMessage: vi.fn(), checkMessage: vi.fn(), startMessage: vi.fn(), getMessageJob: vi.fn(), cancelMessageJob: vi.fn(), waitMessageJob: vi.fn() }));

const snapshot = { digest: "snapshot-1", repositoryId: "repo-1", baseHead: "head-1", capturedAt: "2026-10-21T12:00:00Z", files: [{ path: "api/source.go", newBlobId: "blob-1" }, { path: "ui/view.tsx", newBlobId: "blob-2" }] };
const suggestion = () => create(StagedMessageSuggestionSchema, { id: "suggestion-1", title: "fix: retain staged source", body: "Read the selected version and related updates.", coverage: { coveredPaths: ["api/source.go"], uncoveredPaths: ["ui/view.tsx"], matchedEdits: 1, totalEdits: 3, partial: true }, trailers: [{ key: "Vrooli-Proposal", value: "saved-reference" }] });
const collection = (withMessage = true) => create(StagedMessageCollectionSchema, { id: "collection-1", snapshot, status: withMessage ? "partial" : "no_suggestions", suggestions: withMessage ? [suggestion()] : [], contexts: [{ id: "context-1", title: "Matching saved work", kind: "proposal", detail: "Retained content matches api/source.go", matchedPaths: ["api/source.go"], status: "available" }] });
const job = (status = "running") => create(StagedMessageJobSchema, { id: "job-1", executionId: "owner-1", repositoryId: "repo-1", snapshotDigest: snapshot.digest, snapshot, status, applicable: true, createdAt: new Date().toISOString(), phase: "Reading staged changes", ...(status === "succeeded" ? { result: suggestion() } : {}) });

function Harness({ initial = "", version = 1, count = 2, pending = false, repo = "repo-1", continuation }: { initial?: string; version?: number; count?: number; pending?: boolean; repo?: string; continuation?: string }) {
  const [message, setMessage] = useState(initial);
  return <><CommitMessageComposer key={repo} value={message} onChange={setMessage} repoId={repo} stagedCount={count} statusVersion={version} updatingIndex={pending} disabled={false} amend={false} />{continuation && <button onClick={() => setMessage(current => mergeContinuationMessage(current, continuation))}>Continue from history</button>}<output data-testid="serialized-message">{message}</output></>;
}
function reviewEvidence() {
  const summary = screen.getByText("Review evidence");
  if (!summary.closest("details")?.open) fireEvent.click(summary);
}
function suggest() { fireEvent.click(screen.getByRole("button", { name: "Suggest commit title and description" })); }
async function gathered() { suggest(); await screen.findByText("Partial reusable suggestion"); }

function CommitHarness({ onCommit }: { onCommit: (message: string) => void }) {
  const [message, setMessage] = useState("Operator title\n\nX-Custom: keep");
  return <CommitPanel repoId="repo-1" stagedCount={2} commitMessage={message} onCommitMessageChange={setMessage} onCommit={onCommit} isCommitting={false} />;
}

function CommitLifecycleHarness({ repo = "repo-1", showComposer = true }: { repo?: string; showComposer?: boolean }) {
  const [message, setMessage] = useState("");
  const commit = useCommit(repo);
  return <>
    {showComposer && <CommitMessageComposer key={repo} value={message} onChange={setMessage} repoId={repo} stagedCount={2} statusVersion={1} updatingIndex={false} disabled={false} amend={false} />}
    <button onClick={() => commit.mutate({ message })}>Commit selection</button>
  </>;
}

beforeEach(() => {
  vi.mocked(createCommit).mockReset().mockResolvedValue(create(CreateCommitResponseSchema, { success: true, hash: "new-commit" }));
  vi.mocked(collectMessage).mockReset().mockResolvedValue(collection());
  vi.mocked(checkMessage).mockReset().mockResolvedValue(create(CheckStagedMessageResponseSchema, { applicable: true, currentSnapshotDigest: snapshot.digest }));
  vi.mocked(startMessage).mockReset().mockResolvedValue(job());
  vi.mocked(getMessageJob).mockReset().mockResolvedValue(job());
  vi.mocked(cancelMessageJob).mockReset().mockResolvedValue(job("cancelling"));
  vi.mocked(waitMessageJob).mockReset().mockImplementation((_repo, _id, _key, signal) => new Promise((_resolve, reject) => signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true })));
});

describe("staged commit message composer", () => {
  it.each(["history first", "AI first"])("keeps the AI message and history references in either order (%s)", async order => {
    const continuation = "Earlier work p2\n\nVrooli-Effort: effort:work\nVrooli-Continues: abc123";
    renderWithProviders(<Harness continuation={continuation} />);
    const continueFromHistory = () => fireEvent.click(screen.getByRole("button", { name: "Continue from history" }));
    if (order === "history first") continueFromHistory();
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    if (order === "AI first") continueFromHistory();
    continueFromHistory();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source");
    expect(screen.getByTestId("commit-message-description")).toHaveValue("Read the selected version and related updates.");
    const fields = parseMessage(screen.getByTestId("serialized-message").textContent ?? "");
    expect(fields.trailers.split("\n").sort()).toEqual([
      "Vrooli-Continues: abc123", "Vrooli-Effort: effort:work", "Vrooli-Proposal: saved-reference",
    ].sort());
    expect(startMessage).not.toHaveBeenCalled();
  });

  it.each(["navigation", "reload"])("retires an accepted agent draft after a confirmed commit and collects fresh evidence after %s", async mode => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    vi.mocked(getMessageJob).mockResolvedValue(job("succeeded"));
    vi.mocked(createCommit).mockResolvedValue(create(CreateCommitResponseSchema, { success: true, hash: "new-commit" }));
    const view = renderWithProviders(<CommitLifecycleHarness />);
    await screen.findByText("Agent suggestion ready");
    suggest();
    fireEvent.click(await screen.findByRole("button", { name: "Use partial suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(screen.getByText("Suggestion applied · editable")).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Commit selection", hidden: true }));
    await waitFor(() => expect(screen.queryByTestId("commit-suggestion-standing")).not.toBeInTheDocument());
    expect(localStorage.getItem("gct.messageDraft.repo-1")).toBeNull();
    view.unmount();
    renderWithProviders(<CommitLifecycleHarness />, mode === "navigation" ? { queryClient: view.queryClient } : {});
    expect(screen.queryByTestId("commit-suggestion-standing")).not.toBeInTheDocument();
    await gathered();
    expect(collectMessage).toHaveBeenCalledTimes(1);
    expect(startMessage).not.toHaveBeenCalled();
  });


  it.each(["refused", "transport error", "missing hash"])("preserves the accepted draft when a commit has %s", async failure => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    vi.mocked(getMessageJob).mockResolvedValue(job("succeeded"));
    if (failure === "transport error") vi.mocked(createCommit).mockRejectedValue(new Error("connection lost"));
    else vi.mocked(createCommit).mockResolvedValue(create(CreateCommitResponseSchema, failure === "refused" ? { success: false, error: "precommit failed" } : { success: true }));
    renderWithProviders(<CommitLifecycleHarness />);
    await screen.findByText("Agent suggestion ready");
    suggest();
    fireEvent.click(await screen.findByRole("button", { name: "Use partial suggestion" }));
    await screen.findByText("Suggestion applied · editable");
    fireEvent.click(screen.getByRole("button", { name: "Commit selection", hidden: true }));
    await waitFor(() => expect(createCommit).toHaveBeenCalledTimes(1));
    await act(async () => {});
    expect(screen.getByText("Suggestion applied · editable")).toBeVisible();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source");
    expect(localStorage.getItem("gct.messageDraft.repo-1")).toContain("job-1");
  });

  it("clears a completed draft while its composer is unmounted, without clearing another repository", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    localStorage.setItem("gct.messageDraft.other-repo", JSON.stringify({ jobID: "other-job" }));
    vi.mocked(getMessageJob).mockResolvedValue(job("succeeded"));
    const view = renderWithProviders(<CommitLifecycleHarness />);
    await screen.findByText("Agent suggestion ready");
    view.rerender(<CommitLifecycleHarness showComposer={false} />);
    fireEvent.click(screen.getByRole("button", { name: "Commit selection", hidden: true }));
    await waitFor(() => expect(localStorage.getItem("gct.messageDraft.repo-1")).toBeNull());
    expect(localStorage.getItem("gct.messageDraft.other-repo")).toContain("other-job");
    view.rerender(<CommitLifecycleHarness />);
    expect(screen.queryByTestId("commit-suggestion-standing")).not.toBeInTheDocument();
  });

  it("retires the committed repository after a switch while preserving the newly viewed repository", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    localStorage.setItem("gct.messageDraft.other-repo", JSON.stringify({ jobID: "other-job" }));
    vi.mocked(getMessageJob).mockImplementation(async repo => ({ ...job("succeeded"), repositoryId: repo, id: repo === "repo-1" ? "job-1" : "other-job" }));
    let finish: (value: Awaited<ReturnType<typeof createCommit>>) => void = () => { throw new Error("commit not started"); };
    vi.mocked(createCommit).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    const view = renderWithProviders(<CommitLifecycleHarness />);
    const invalidate = vi.spyOn(view.queryClient, "invalidateQueries");
    await screen.findByText("Agent suggestion ready");
    fireEvent.click(screen.getByRole("button", { name: "Commit selection" }));
    await waitFor(() => expect(createCommit).toHaveBeenCalledTimes(1));
    view.rerender(<CommitLifecycleHarness repo="other-repo" />);
    await screen.findByText("Agent suggestion ready");
    await act(async () => finish(create(CreateCommitResponseSchema, { success: true, hash: "earlier-commit" })));
    expect(localStorage.getItem("gct.messageDraft.repo-1")).toBeNull();
    expect(localStorage.getItem("gct.messageDraft.other-repo")).toContain("other-job");
    expect(screen.getByText("Agent suggestion ready")).toBeVisible();
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["repo", "history", "repo-1"] });
    expect(invalidate).not.toHaveBeenCalledWith({ queryKey: ["repo", "history", "other-repo"] });
  });

  it("ignores a late evidence response after a successful commit retires its collection", async () => {
    let finish: (value: ReturnType<typeof collection>) => void = () => { throw new Error("collection not started"); };
    vi.mocked(collectMessage).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    renderWithProviders(<CommitLifecycleHarness />);
    suggest();
    await screen.findByText("Checking existing evidence…");
    fireEvent.click(screen.getByRole("button", { name: "Commit selection", hidden: true }));
    await waitFor(() => expect(screen.queryByTestId("commit-suggestion-standing")).not.toBeInTheDocument());
    await act(async () => finish(collection()));
    expect(screen.queryByTestId("commit-suggestion-standing")).not.toBeInTheDocument();
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("keeps a newer collection when an earlier commit finishes", async () => {
    let finish: (value: Awaited<ReturnType<typeof createCommit>>) => void = () => { throw new Error("commit not started"); };
    vi.mocked(createCommit).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    renderWithProviders(<CommitLifecycleHarness />);
    fireEvent.click(screen.getByRole("button", { name: "Commit selection", hidden: true }));
    await waitFor(() => expect(createCommit).toHaveBeenCalledTimes(1));
    await gathered();
    await act(async () => finish(create(CreateCommitResponseSchema, { success: true, hash: "earlier-commit" })));
    expect(screen.getByText("Partial reusable suggestion")).toBeVisible();
    expect(screen.getByTestId("commit-suggestion-standing")).toBeVisible();
  });

  it("ignores a late applicability check instead of adopting a committed draft", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    vi.mocked(getMessageJob).mockResolvedValue(job("succeeded"));
    renderWithProviders(<CommitLifecycleHarness />);
    await screen.findByText("Agent suggestion ready");
    suggest();
    let finish: (value: CheckStagedMessageResponse) => void = () => { throw new Error("verification not started"); };
    vi.mocked(checkMessage).mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    fireEvent.click(await screen.findByRole("button", { name: "Use partial suggestion" }));
    fireEvent.click(screen.getByRole("button", { name: "Commit selection", hidden: true }));
    await waitFor(() => expect(localStorage.getItem("gct.messageDraft.repo-1")).toBeNull());
    await act(async () => finish(create(CheckStagedMessageResponseSchema, { applicable: true })));
    expect(screen.getByTestId("commit-message-input")).toHaveValue("");
    expect(screen.queryByTestId("commit-suggestion-standing")).not.toBeInTheDocument();
  });

  it.each(["queued", "running", "unknown"])("retains %s owner handles for cancellation and reconnect after a commit", async status => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    vi.mocked(getMessageJob).mockResolvedValue(job(status));
    renderWithProviders(<CommitLifecycleHarness />);
    await waitFor(() => expect(getMessageJob).toHaveBeenCalledTimes(1));
    fireEvent.click(screen.getByRole("button", { name: "Commit selection", hidden: true }));
    await waitFor(() => expect(createCommit).toHaveBeenCalledTimes(1));
    await act(async () => {});
    expect(localStorage.getItem("gct.messageDraft.repo-1")).toContain("job-1");
    expect(screen.getByTestId("commit-suggestion-standing")).toBeVisible();
    expect(startMessage).not.toHaveBeenCalled();
    expect(cancelMessageJob).not.toHaveBeenCalled();
  });

  it.each([
    ["", "No failure explanation is available for this saved request."],
    ["Drafting stopped because the workflow token budget was exhausted.", "Drafting stopped because the workflow token budget was exhausted."],
  ])("explains a failed investigation before its preserved previous draft (cause: %s)", async (error, expectedReason) => {
    const prior = { ...suggestion(), body: "Previous draft paragraph. ".repeat(80), messageScope: StagedMessageScope.LIMITED, messageScopeExplanation: "Deployment behavior remains unresolved." };
    vi.mocked(startMessage).mockResolvedValueOnce({ ...job("succeeded"), result: prior }).mockResolvedValueOnce({ ...job("failed"), id: "failed-investigation", continuationOfJobId: "job-1", error });
    renderWithProviders(<Harness initial="Keep my message" />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Limited draft");
    fireEvent.click(screen.getByRole("button", { name: "Investigate unresolved themes" }));
    await screen.findByRole("heading", { name: "Drafting failed" });
    const failure = screen.getByRole("alert");
    expect(within(failure).getByText(expectedReason)).toBeVisible();
    expect(screen.getByText("The investigation failed. This earlier draft is still available.")).toBeVisible();
    expect(screen.queryByText("This earlier draft is still available. The new draft may leave unresolved themes.")).not.toBeInTheDocument();
    const previous = screen.getByTestId("commit-message-previous-draft");
    if (error) {
      const technical = screen.getByText("Technical details");
      expect(technical.compareDocumentPosition(previous) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    }
    expect(screen.getByRole("button", { name: "Use previous draft" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Investigate unresolved themes" })).toBeEnabled();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep my message");
    expect(startMessage).toHaveBeenCalledTimes(2);
  });

  it.each(["succeeded", "queued", "running"])("keeps the 285-file selection neutral after an evidence-denied investigation (%s)", async status => {
    const paths = Array.from({ length: 285 }, (_, index) => `src/file-${index}.go`);
    const selected = create(StagedMessageCollectionSchema, { ...collection(), snapshot: create(StagedMessageSnapshotSchema, { ...snapshot, files: paths.map(path => create(StagedMessageFileSchema, { path, newBlobId: "retained-blob" })) }) });
    const partial = create(StagedMessageSuggestionSchema, { ...suggestion(), coverage: create(StagedMessageCoverageSchema, { coveredPaths: paths.slice(0, 26), uncoveredPaths: paths.slice(26), partial: true, matchedEdits: 26, totalEdits: 285 }) });
    const explanation = "Investigation made no progress: evidence access requires the owning verified workflow child. No new evidence was read; the previous draft is retained.";
    const limited = { ...partial, messageScope: StagedMessageScope.LIMITED, messageScopeExplanation: explanation };
    vi.mocked(collectMessage).mockResolvedValue(selected);
    vi.mocked(startMessage).mockResolvedValueOnce({ ...job("succeeded"), snapshot: selected.snapshot, result: partial }).mockResolvedValueOnce({ ...job(status), id: "blocked-continuation", snapshot: selected.snapshot, continuationOfJobId: "job-1", ...(status === "succeeded" ? { result: limited } : {}) });
    renderWithProviders(<Harness count={285} />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    if (status === "succeeded") {
      await screen.findByText("Limited draft");
      expect(screen.getByRole("heading", { name: "Why this draft is limited" })).toBeVisible();
      expect(screen.getByText(explanation)).toBeVisible();
      expect(screen.getByText("Detailed evidence inspected: 26 of 285 staged files.")).toBeVisible();
      expect(screen.getByRole("button", { name: "Use previous draft" })).toBeVisible();
    } else await screen.findByText(status === "queued" ? "Drafting agent queued" : "Drafting agent is running");
    expect(screen.getByText("Staged selection: 285 files")).toBeVisible();
    expect(screen.queryByText("For all 285 staged files")).not.toBeInTheDocument();
    expect(screen.queryByText("Summarizes the whole staged selection.")).not.toBeInTheDocument();
    expect(startMessage).toHaveBeenCalledTimes(2);
  });

  it("keeps an unrecognized future scope explicit instead of inferring whole selection", async () => {
    const unknown = fromJson(StagedMessageSuggestionSchema, { id: "future-scope", title: "fix: retain the message", messageScope: 99, coverage: { coveredPaths: ["api/source.go"], uncoveredPaths: ["ui/view.tsx"], partial: false } });
    vi.mocked(startMessage).mockResolvedValue({ ...job("succeeded"), result: unknown });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Draft scope unavailable");
    expect(screen.queryByText("Summarizes the whole staged selection.")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Investigate unresolved themes" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use suggestion" })).toBeVisible();
  });

  it("shows whole-selection synthesis independently of sampled detailed inspection", async () => {
    const whole = { ...suggestion(), messageScope: StagedMessageScope.WHOLE_SELECTION, messageScopeExplanation: "The complete staged inventory and selected key changes support this summary.", unknowns: ["Deployment behavior still needs confirmation"] };
    vi.mocked(startMessage).mockResolvedValue({ ...job("succeeded"), result: whole });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Whole-selection draft");
    expect(screen.getByText("Summarizes the whole staged selection.")).toBeVisible();
    expect(screen.getByText("Detailed evidence inspected: 1 of 2 staged files.")).toBeVisible();
    expect(screen.getByText("Deployment behavior still needs confirmation")).toBeVisible();
    expect(screen.queryByTestId("commit-message-partial-coverage")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use suggestion" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Investigate remaining changes" })).not.toBeInTheDocument();
    expect(startMessage).toHaveBeenCalledTimes(1);
    reviewEvidence();
    expect(screen.getByRole("list", { name: "Files without detailed inspection" })).toHaveTextContent("ui/view.tsx");
    expect(screen.queryByRole("list", { name: "Uncovered staged files" })).not.toBeInTheDocument();
  });

  it("offers further synthesis for substantive gaps even when every file has detailed evidence", async () => {
    const limited = { ...suggestion(), messageScope: StagedMessageScope.LIMITED, messageScopeExplanation: "No new evidence resolved the uncertain deployment behavior. Investigate that theme before relying on this draft.", coverage: create(StagedMessageCoverageSchema, { coveredPaths: ["api/source.go", "ui/view.tsx"], uncoveredPaths: [], partial: false, matchedEdits: 3, totalEdits: 3 }) };
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded")).mockResolvedValueOnce({ ...job("succeeded"), id: "limited-continuation", continuationOfJobId: "job-1", result: limited });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    await screen.findByText("Limited draft");
    expect(screen.getByText(limited.messageScopeExplanation)).toBeVisible();
    expect(screen.getByText("Detailed evidence inspected: 2 of 2 staged files.")).toBeVisible();
    expect(screen.getByRole("button", { name: "Use limited suggestion" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Investigate unresolved themes" })).toBeVisible();
    expect(screen.getByRole("heading", { name: "Previous draft kept" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Use previous draft" })).toBeVisible();
    expect(startMessage).toHaveBeenCalledTimes(2);
  });

  it("lets a whole-selection continuation supersede the prior partial draft without requiring every patch", async () => {
    const whole = { ...suggestion(), id: "whole-draft", title: "fix: summarize all affected themes", messageScope: StagedMessageScope.WHOLE_SELECTION, messageScopeExplanation: "Read the full inventory and selected important changes." };
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded")).mockResolvedValueOnce({ ...job("succeeded"), id: "continuation-1", continuationOfJobId: "job-1", result: whole });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    await screen.findByText("Whole-selection draft");
    expect(screen.queryByRole("heading", { name: "Previous draft kept" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Use suggestion" })).toBeVisible();
    expect(startMessage).toHaveBeenCalledTimes(2);
  });

  it("keeps the earlier draft available when further investigation still returns a partial message", async () => {
    const newer = { ...suggestion(), id: "newer-partial", title: "fix: describe additional changes" };
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded")).mockResolvedValueOnce({ ...job("succeeded"), id: "continuation-1", continuationOfJobId: "job-1", result: newer });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    await screen.findByRole("heading", { name: newer.title });
    expect(screen.getByRole("heading", { name: "Previous draft kept" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Use partial suggestion" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Investigate remaining changes" })).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Use previous draft" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(startMessage).toHaveBeenCalledTimes(2);
  });

  it("distinguishes initial sampled detail from completed investigation receipts", async () => {
    const initial = collection();
    const file = initial.snapshot?.files[0];
    if (!file) throw new Error("Fixture needs an initial staged file");
    file.patch = "+initial detail";
    vi.mocked(collectMessage).mockResolvedValue(initial);
    vi.mocked(startMessage).mockResolvedValue({ ...job("succeeded"), snapshot: initial.snapshot, continuationOfJobId: "previous-job", evidenceFilesRead: 2, evidenceFilesTotal: 2 });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    reviewEvidence();
    expect(screen.getByText("Additional detailed files read: 2")).toBeVisible();
    expect(screen.getByText("Initial patch details: 1 file. Initial evidence limits: 0 files.")).toBeVisible();
    expect(screen.getByText(/These counts describe the initial evidence only/)).toBeVisible();
    expect(screen.queryByText(/Agent input uses bounded summaries/)).not.toBeInTheDocument();
  });

  it("offers explicit further investigation beside using an agent partial draft", async () => {
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded"));
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    const use = screen.getByRole("button", { name: "Use partial suggestion" });
    const investigate = screen.getByRole("button", { name: "Investigate remaining changes" });
    expect(within(screen.getByTestId("commit-message-actions")).getByRole("button", { name: "Use partial suggestion" })).toBe(use);
    expect(within(screen.getByTestId("commit-message-actions")).getByRole("button", { name: "Investigate remaining changes" })).toBe(investigate);
    expect(use.querySelector("svg")).toHaveAttribute("aria-hidden", "true");
    expect(investigate.querySelector("svg")).toHaveAttribute("aria-hidden", "true");
    expect(startMessage).toHaveBeenCalledTimes(1);
  });

  it.each(["failed", "cancelled"])("keeps a usable previous draft if further investigation is %s", async status => {
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded")).mockResolvedValueOnce({ ...job(status), id: "continuation-1", error: status === "failed" ? "Investigation could not finish" : "" });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    await screen.findByRole("heading", { name: "Previous draft kept" });
    const previous = screen.getByRole("button", { name: "Use previous draft" });
    expect(previous).toBeEnabled();
    fireEvent.click(previous);
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(startMessage).toHaveBeenCalledTimes(2);
    const [first, second] = vi.mocked(startMessage).mock.calls.map(args => args[0]);
    expect(second?.requestKey).not.toBe(first?.requestKey);
  });

  it("keeps the previous draft visible but unapplied during uncertain continuation admission", async () => {
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded")).mockRejectedValueOnce(new ConnectError("Reply lost", Code.Unavailable));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("Status unavailable", Code.Unavailable));
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    await screen.findByRole("heading", { name: "Could not confirm launch" });
    expect(screen.getByRole("heading", { name: "Previous draft kept" })).toBeVisible();
    expect(screen.getByRole("button", { name: "Reconnect to job" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Use previous draft" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Use partial suggestion" })).not.toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("");
    expect(startMessage).toHaveBeenCalledTimes(2);
  });

  it("shows investigation evidence-read progress separately from previous message coverage", async () => {
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded")).mockResolvedValueOnce({ ...job(), id: "continuation-1", continuationOfJobId: "job-1", evidenceFilesRead: 80, evidenceFilesTotal: 285 });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    await screen.findByText("Additional detailed files read: 80");
    expect(screen.getByText("Fully describes 1 of 2 staged files.")).toBeVisible();
    expect(screen.queryByRole("button", { name: "Use previous draft" })).not.toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("");
  });

  it("compares human edits before using a previous draft after investigation fails", async () => {
    vi.mocked(startMessage).mockResolvedValueOnce(job("succeeded")).mockResolvedValueOnce({ ...job("failed"), id: "continuation-1", error: "Investigation could not finish" });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.change(screen.getByTestId("commit-message-input"), { target: { value: "My own message" } });
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Agent suggestion");
    fireEvent.click(screen.getByRole("button", { name: "Investigate remaining changes" }));
    await screen.findByRole("heading", { name: "Previous draft kept" });
    fireEvent.click(screen.getByRole("button", { name: "Use previous draft" }));
    await screen.findByRole("heading", { name: "Keep your edits in view" });
    expect(screen.getByText("Agent suggestion")).toBeVisible();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("My own message");
    expect(screen.getByRole("button", { name: "Use replacement" })).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "Keep mine" }));
    expect(screen.getByTestId("commit-message-input")).toHaveValue("My own message");
  });

  it("distinguishes absent related context from captured staged evidence", async () => {
    const captured = collection(false);
    captured.contexts = [];
    const capturedFile = captured.snapshot?.files[0];
    if (!capturedFile) throw new Error("Fixture needs a captured staged file");
    capturedFile.patch = "diff --git a/api/source.go b/api/source.go\n+read staged evidence";
    vi.mocked(collectMessage).mockResolvedValue(captured);
    renderWithProviders(<Harness />);
    suggest();
    await screen.findByRole("heading", { name: "No reusable message found" });
    expect(screen.getByText(/An agent can draft for the whole selection using staged-diff evidence/)).toBeVisible();
    reviewEvidence();
    expect(screen.getByText(/No related work context is available/)).toBeVisible();
    expect(screen.getByText("Initial patch details: 1 · 0 sources")).toBeVisible();
    expect(startMessage).not.toHaveBeenCalled();
  });

  it.each([
    ["failed_precondition: Sign in through a trusted browser session before starting a drafting agent", "Sign in through a trusted browser session"],
    ["unauthenticated: 401 Unauthorized", "Drafting could not verify your sign-in"],
  ])("surfaces actionable access recovery while launch remains unconfirmed (%s)", async (error, guidance) => {
    vi.mocked(startMessage).mockResolvedValue({ ...job(), executionId: "", error });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByRole("heading", { name: "Confirming agent launch" });
    expect(screen.getByRole("alert")).toHaveTextContent(guidance);
    expect(screen.getByRole("button", { name: "Reconnect to job" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Start drafting agent" })).not.toBeInTheDocument();
  });

  it("surfaces trusted-browser sign-in guidance from a durable failed job", async () => {
    vi.mocked(startMessage).mockResolvedValue({ ...job("failed"), executionId: "", error: "failed_precondition: Sign in through a trusted browser session before starting a drafting agent" });
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByRole("heading", { name: "Drafting request failed" });
    expect(screen.getByRole("alert")).toHaveTextContent("Sign in through a trusted browser session before starting a drafting agent");
    expect(screen.getByText("Sign in through a trusted browser session before starting a drafting agent.")).toBeVisible();
  });

  it("prioritizes same-request recovery over refreshing stale evidence after an uncertain launch", async () => {
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("lost reply", Code.Unavailable));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("not admitted yet", Code.NotFound));
    const view = renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByRole("heading", { name: "No job found yet" });
    vi.mocked(checkMessage).mockResolvedValue(create(CheckStagedMessageResponseSchema, { applicable: false, reason: "Staged selection changed" }));
    view.rerender(<Harness version={2} />);
    await screen.findByTestId("commit-message-stale");
    expect(screen.getByTestId("commit-message-primary")).toHaveTextContent("Retry same request");
    expect(collectMessage).toHaveBeenCalledTimes(1);
  });

  it.each([true, false])("keeps reload recovery reachable after a job-not-found response (frozen intent: %s)", async frozen => {
    const requestKey = "pending-admission";
    const admissionRequest = { repositoryId: "repo-1", collectionId: "collection-1", snapshotDigest: snapshot.digest, selectedContextIds: ["context-1"], operatorNote: "Original intent", requestKey };
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ requestKey, ...(frozen ? { admissionRequest } : {}) }));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("not admitted yet", Code.NotFound));
    renderWithProviders(<Harness initial="Keep my message" />);
    await waitFor(() => expect(getMessageJob).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    await screen.findByRole("heading", { name: "No job found yet" });
    const primary = screen.getByTestId("commit-message-primary");
    expect(primary).toHaveTextContent(frozen ? "Retry same request" : "Reconnect to job");
    fireEvent.click(primary);
    if (frozen) await waitFor(() => expect(startMessage).toHaveBeenCalledWith(admissionRequest));
    else {
      await waitFor(() => expect(getMessageJob).toHaveBeenCalledTimes(2));
      expect(startMessage).not.toHaveBeenCalled();
    }
    expect(collectMessage).not.toHaveBeenCalled();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep my message");
  });

  it("disables secondary agent launch while the index is changing", async () => {
    const view = renderWithProviders(<Harness />);
    await gathered();
    view.rerender(<Harness pending />);
    const launch = screen.getByRole("button", { name: "Ask agent to complete" });
    expect(launch).toBeDisabled();
    fireEvent.click(launch);
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("shows stopping immediately while the cancellation request is unresolved", async () => {
    vi.mocked(cancelMessageJob).mockImplementation(() => new Promise(() => undefined));
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Drafting agent is running");
    fireEvent.click(within(screen.getByRole("dialog", { name: "Commit message suggestions" })).getByRole("button", { name: "Cancel agent" }));
    expect(screen.getByRole("heading", { name: "Requesting cancellation…" })).toBeVisible();
    expect(within(screen.getByRole("dialog", { name: "Commit message suggestions" })).getByRole("button", { name: "Cancel agent" })).toBeDisabled();
  });
  it("distinguishes a saved request from an agent launch when the owner handle is unknown", async () => {
    vi.mocked(startMessage).mockResolvedValue({ ...job(), executionId: "", error: "Owner acknowledgement unavailable" });
    renderWithProviders(<Harness initial="Keep my message" />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByRole("heading", { name: "Confirming agent launch" });
    expect(screen.queryByText("Drafting agent is running")).not.toBeInTheDocument();
    expect(screen.getByText("Your request is saved. Confirming whether the agent has started.")).toBeVisible();
    expect(screen.getByText("elapsed")).toBeVisible();
    expect(screen.getByRole("button", { name: "Reconnect to job" })).toBeVisible();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep my message");
  });
  it("recovers an uncertain launch by reconnecting to the same request without redispatch", async () => {
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("Owner unavailable", Code.Unavailable));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("Owner unavailable", Code.Unavailable));
    renderWithProviders(<Harness initial="Keep the operator text" />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByRole("heading", { name: "Could not confirm launch" });
    expect(screen.getByTestId("commit-message-primary")).toHaveTextContent("Reconnect to job");
    expect(screen.queryByRole("button", { name: "Start drafting agent" })).not.toBeInTheDocument();
    vi.mocked(getMessageJob).mockResolvedValue(job());
    fireEvent.click(screen.getByRole("button", { name: "Reconnect to job" }));
    await screen.findByText("Drafting agent is running");
    expect(startMessage).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep the operator text");
  });
  it("shows immediate launch feedback above evidence while admission is unresolved", async () => {
    vi.mocked(startMessage).mockImplementation(() => new Promise(() => undefined));
    renderWithProviders(<Harness initial="Keep my message" />);
    await gathered();
    reviewEvidence();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    expect(screen.getByRole("heading", { name: "Starting drafting agent…" })).toBeVisible();
    expect(screen.getByTestId("commit-drafting-progress")).toHaveTextContent("Confirming your drafting request.");
    expect(screen.getByTestId("commit-drafting-progress")).toHaveTextContent("Your message stays unchanged");
    expect(screen.getByRole("button", { name: "Starting agent…" })).toBeDisabled();
    expect(screen.getByRole("checkbox", { name: /Matching saved work/ })).toBeDisabled();
    expect(screen.queryByText("fix: retain staged source")).not.toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep my message");
  });

  it("keeps failed admission status unconfirmed when no execution handle was returned", async () => {
    vi.mocked(startMessage).mockResolvedValue(create(StagedMessageJobSchema, { ...job("failed"), executionId: "", error: "start message workflow: unauthenticated: 401 Unauthorized" }));
    renderWithProviders(<Harness initial="Keep my message" />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByRole("heading", { name: "Drafting request failed" });
    expect(screen.getByRole("alert")).toHaveTextContent("We could not confirm whether the agent started");
    expect(screen.getByRole("button", { name: "Check evidence again" })).toBeVisible();
    expect(screen.queryByRole("button", { name: "Reconnect to job" })).not.toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep my message");
    expect(startMessage).toHaveBeenCalledTimes(1);
  });

  it("keeps all 4096 staged identities while previewing bounded evidence and uncovered paths", async () => {
    const large = collection();
    if (!large.snapshot || !large.suggestions[0]?.coverage) throw new Error("fixture lacks snapshot coverage");
    large.snapshot.files = Array.from({ length: 4096 }, (_, index) => create(StagedMessageFileSchema, { path: `scenarios/work-${index % 4}/file-${index}.go`, newBlobId: `blob-${index}`, patch: index < 64 ? "@@ selected patch" : "", omissionReason: index < 64 ? "" : "detail budget exhausted" }));
    const omitted = large.snapshot.files.slice(64);
    large.omissions = omitted.map(file => create(OmissionSchema, { path: file.path, reason: file.omissionReason }));
    large.suggestions[0].coverage.coveredPaths = large.snapshot.files.slice(0, 64).map(file => file.path);
    large.suggestions[0].coverage.uncoveredPaths = omitted.map(file => file.path);
    large.suggestions[0].coverage.totalEdits = 4096;
    large.suggestions[0].coverage.matchedEdits = 64;
    vi.mocked(collectMessage).mockResolvedValue(large);
    renderWithProviders(<Harness count={4096} />);
    await gathered();
    const dialog = screen.getByRole("dialog", { name: "Commit message suggestions" });
    expect(dialog).toHaveTextContent("Staged selection: 4,096 files");
    expect(dialog).not.toHaveTextContent("Initial patch details: 64 files");
    reviewEvidence();
    expect(dialog).toHaveTextContent("Initial patch details: 64 files");
    expect(dialog).toHaveTextContent("Fully describes 64 of 4,096 staged files");
    expect(dialog).toHaveTextContent("Initial evidence limits: 4,032 files");
    expect(dialog).toHaveTextContent("Evidence limits do not change which files will be committed");
    expect(dialog.querySelectorAll("li").length).toBeLessThan(100);
    const paths = screen.getByRole("list", { name: "Uncovered staged files" });
    expect(within(paths).getAllByRole("listitem")).toHaveLength(20);
    fireEvent.click(screen.getByRole("button", { name: "Show 20 more uncovered staged files" }));
    expect(within(paths).getAllByRole("listitem")).toHaveLength(40);
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(checkMessage).toHaveBeenCalledWith("repo-1", snapshot.digest);
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("renders related sources deliberately in small batches and preserves checked intent when collapsed", async () => {
    const large = collection();
    const original = large.contexts[0];
    if (!original) throw new Error("fixture lacks source");
    large.contexts = Array.from({ length: 100 }, (_, index) => ({ ...original, id: `context-${index}`, title: `Related work ${index}`, matchedPaths: Array.from({ length: 4096 }, (_, file) => `src/file-${file}.go`) }));
    vi.mocked(collectMessage).mockResolvedValue(large);
    renderWithProviders(<Harness count={4096} />);
    await gathered();
    const dialog = screen.getByRole("dialog", { name: "Commit message suggestions" });
    expect(dialog.querySelectorAll("input[type=checkbox]")).toHaveLength(0);
    const summary = screen.getByText("Review evidence");
    fireEvent.click(summary);
    expect(screen.getAllByRole("checkbox")).toHaveLength(10);
    expect(dialog.querySelectorAll("li").length).toBeLessThan(250);
    fireEvent.click(screen.getByRole("checkbox", { name: /Related work 0/ }));
    fireEvent.click(summary);
    fireEvent.click(summary);
    await waitFor(() => expect(screen.getByRole("checkbox", { name: /Related work 0/ })).toBeChecked());
    fireEvent.click(screen.getByRole("button", { name: "Show 10 more related sources" }));
    expect(screen.getAllByRole("checkbox")).toHaveLength(20);
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("does not treat omitted patch details as uncovered retained-message coverage", async () => {
    const full = collection();
    if (!full.snapshot || !full.suggestions[0]?.coverage) throw new Error("fixture lacks snapshot coverage");
    full.snapshot.files = Array.from({ length: 4096 }, (_, index) => create(StagedMessageFileSchema, { path: `src/file-${index}.go`, newBlobId: `blob-${index}`, omissionReason: "detail budget exhausted" }));
    full.omissions = full.snapshot.files.map(file => create(OmissionSchema, { path: file.path, reason: file.omissionReason }));
    full.suggestions[0].coverage.partial = false;
    full.suggestions[0].coverage.coveredPaths = full.snapshot.files.map(file => file.path);
    full.suggestions[0].coverage.uncoveredPaths = [];
    full.suggestions[0].coverage.totalEdits = 4096;
    full.suggestions[0].coverage.matchedEdits = 4096;
    vi.mocked(collectMessage).mockResolvedValue(full);
    renderWithProviders(<Harness count={4096} />);
    suggest();
    await screen.findByText("Reusable suggestion", { selector: "span" });
    expect(screen.getByRole("button", { name: "Use suggestion" })).toBeEnabled();
    expect(screen.getByRole("dialog", { name: "Commit message suggestions" })).toHaveTextContent("Fully describes 4,096 of 4,096 staged files");
    expect(screen.queryByTestId("commit-message-partial-coverage")).not.toBeInTheDocument();
    reviewEvidence();
    expect(screen.getByRole("dialog", { name: "Commit message suggestions" })).toHaveTextContent("Initial evidence limits: 4,096 files");
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("splits existing multiline messages and preserves unknown folded trailers while editing", () => {
    renderWithProviders(<Harness initial={"fix: old title\n\nOperator description.\n\nX-Custom: keep\n  folded value"} />);
    expect(screen.getByRole("textbox", { name: "Title" })).toHaveValue("fix: old title");
    expect(screen.getByRole("textbox", { name: /Description/ })).toHaveValue("Operator description.");
    fireEvent.change(screen.getByTestId("commit-message-input"), { target: { value: "fix: new title" } });
    expect(screen.getByTestId("serialized-message").textContent).toBe("fix: new title\n\nOperator description.\n\nX-Custom: keep\n  folded value");
    fireEvent.change(screen.getByTestId("commit-message-description"), { target: { value: "First paragraph.\n\n" } });
    expect(screen.getByTestId("commit-message-description")).toHaveValue("First paragraph.\n\n");
    expect(collectMessage).not.toHaveBeenCalled();
  });

  it("accepts a partial reusable suggestion without starting an agent and keeps unknown references", async () => {
    renderWithProviders(<Harness initial={"\n\nX-Custom: operator-value"} />);
    await gathered();
    expect(screen.getByTestId("commit-message-partial-coverage")).toHaveTextContent("Review them before using this draft");
    reviewEvidence();
    expect(screen.getByRole("list", { name: "Uncovered staged files" })).toHaveTextContent("ui/view.tsx");
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(screen.getByTestId("commit-message-trailers")).toHaveValue("X-Custom: operator-value\nVrooli-Proposal: saved-reference");
    expect(startMessage).not.toHaveBeenCalled();
    expect(checkMessage).toHaveBeenCalledWith("repo-1", snapshot.digest);
  });

  it("applying a suggestion only edits the message; the separate Commit action receives all fields", async () => {
    const commit = vi.fn();
    renderWithProviders(<CommitHarness onCommit={commit} />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(commit).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("commit-button"));
    expect(commit).toHaveBeenCalledWith("fix: retain staged source\n\nRead the selected version and related updates.\n\nX-Custom: keep\nVrooli-Proposal: saved-reference", expect.objectContaining({ amend: false, conventional: false }));
  });

  it("offers explicit Start or Cancel when context exists without a reusable message", async () => {
    vi.mocked(collectMessage).mockResolvedValue(collection(false));
    renderWithProviders(<Harness />);
    suggest();
    await screen.findByText("No reusable message found", { selector: "h3" });
    expect(screen.queryByRole("button", { name: "Use suggestion" })).not.toBeInTheDocument();
    expect(startMessage).not.toHaveBeenCalled();
    fireEvent.click(within(screen.getByRole("dialog", { name: "Commit message suggestions" })).getByRole("button", { name: "Hide suggestions" }));
    expect(startMessage).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    fireEvent.click(screen.getByRole("button", { name: "Start drafting agent" }));
    await waitFor(() => expect(startMessage).toHaveBeenCalledTimes(1));
    await screen.findByText("Drafting agent is running");
    expect(screen.getByTestId("commit-message-input")).toHaveValue("");
  });

  it("offers a full retained message with exact coverage without invoking an agent", async () => {
    const full = collection();
    const first = full.suggestions[0];
    if (!first?.coverage) throw new Error("fixture lacks coverage");
    first.coverage.partial = false;
    first.coverage.matchedEdits = 3;
    first.coverage.uncoveredPaths = [];
    full.status = "ready";
    vi.mocked(collectMessage).mockResolvedValue(full);
    renderWithProviders(<Harness />);
    suggest();
    await screen.findByText("Reusable suggestion", { selector: "span" });
    fireEvent.click(screen.getByRole("button", { name: "Use suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue(first.title));
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("binds selected related context and the operator note only on explicit agent start", async () => {
    renderWithProviders(<Harness />);
    await gathered();
    reviewEvidence();
    fireEvent.click(screen.getByText(/Add context/));
    fireEvent.click(screen.getByRole("checkbox", { name: /Matching saved work/ }));
    fireEvent.change(screen.getByRole("textbox", { name: /Why were these changes needed/ }), { target: { value: "Avoid restoring unrelated staged files." } });
    expect(startMessage).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await waitFor(() => expect(startMessage).toHaveBeenCalledWith(expect.objectContaining({ repositoryId: "repo-1", collectionId: "collection-1", snapshotDigest: snapshot.digest, selectedContextIds: ["context-1"], operatorNote: "Avoid restoring unrelated staged files.", requestKey: expect.any(String) })));
  });

  it("preselects only verified retained intent, permits unchecking all, and infers no authorship trailers", async () => {
    const captured = collection();
    const first = captured.suggestions[0];
    const retained = captured.contexts[0];
    if (!first || !retained) throw new Error("fixture requires a retained suggestion");
    first.sourceIds = ["context-1", "speculative", "private"];
    first.trailers = [];
    retained.fullMatch = true;
    retained.status = "retained_edits_included";
    retained.body = "Explains the retained change. Co-Authored-By: not an attribution instruction.";
    captured.contexts.push({ ...retained, id: "speculative", kind: "session_event", title: "Possibly related session", fullMatch: false, status: "possibly_related" }, { ...retained, id: "private", title: "Private source", status: "private" });
    vi.mocked(collectMessage).mockResolvedValue(captured);
    renderWithProviders(<Harness />);
    await gathered();
    reviewEvidence();
    expect(screen.getByRole("checkbox", { name: /Matching saved work/ })).toBeChecked();
    expect(screen.getByRole("checkbox", { name: /Possibly related session/ })).not.toBeChecked();
    expect(screen.getByRole("checkbox", { name: /Private source/ })).not.toBeChecked();
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue(first.title));
    expect(screen.getByTestId("commit-message-trailers")).toHaveValue("");
    expect(startMessage).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    const disclosure = screen.getByText("Review evidence");
    if (!disclosure.closest("details")?.open) fireEvent.click(disclosure);
    fireEvent.click(screen.getByRole("checkbox", { name: /Matching saved work/ }));
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await waitFor(() => expect(startMessage).toHaveBeenCalledWith(expect.objectContaining({ selectedContextIds: [] })));
  });

  it("cancels deterministic collection and ignores its late result", async () => {
    let finish: ((value: ReturnType<typeof collection>) => void) | undefined;
    vi.mocked(collectMessage).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    renderWithProviders(<Harness initial="Keep my title" />);
    suggest();
    expect(screen.getByText("Gathering evidence")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await act(async () => finish?.(collection()));
    expect(screen.queryByText("Partial reusable suggestion")).not.toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep my title");
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("requires comparison when the operator edits while suggestions are gathering", async () => {
    let finish: ((value: ReturnType<typeof collection>) => void) | undefined;
    vi.mocked(collectMessage).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    renderWithProviders(<Harness initial="Original title" />);
    suggest();
    fireEvent.click(within(screen.getByRole("dialog", { name: "Commit message suggestions" })).getByRole("button", { name: "Hide suggestions" }));
    fireEvent.change(screen.getByTestId("commit-message-input"), { target: { value: "Operator's revised title" } });
    await act(async () => finish?.(collection()));
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await screen.findByText("Keep your edits in view");
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Operator's revised title");
    fireEvent.click(screen.getByRole("button", { name: "Keep mine" }));
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Operator's revised title");
  });

  it("checks exact staged applicability after worktree refreshes and before applying", async () => {
    const view = renderWithProviders(<Harness initial="Operator title" />);
    await gathered();
    view.rerender(<Harness initial="Operator title" version={2} />);
    await waitFor(() => expect(checkMessage).toHaveBeenCalledTimes(2));
    expect(screen.queryByTestId("commit-message-stale")).not.toBeInTheDocument();
    vi.mocked(checkMessage).mockResolvedValue(create(CheckStagedMessageResponseSchema, { applicable: false, reason: "Index blob changed", currentSnapshotDigest: "snapshot-2" }));
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await screen.findByTestId("commit-message-stale");
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Operator title");
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("marks same-path staging drift stale while preserving the draft and editor", async () => {
    const view = renderWithProviders(<Harness initial="Existing text" />);
    await gathered();
    vi.mocked(checkMessage).mockResolvedValue(create(CheckStagedMessageResponseSchema, { applicable: false, reason: "Staged contents changed" }));
    view.rerender(<Harness initial="Existing text" version={2} />);
    await screen.findByTestId("commit-message-stale");
    expect(screen.getByText("fix: retain staged source")).toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Existing text");
    expect(screen.getByRole("button", { name: "Refresh evidence" })).toBeInTheDocument();
  });

  it("keeps the owner job while hidden and cancellation pending until owner confirmation", async () => {
    renderWithProviders(<Harness />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Drafting agent is running");
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: "Hide progress" }));
    expect(cancelMessageJob).not.toHaveBeenCalled();
    fireEvent.change(screen.getByTestId("commit-message-input"), { target: { value: "Human text during run" } });
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    fireEvent.click(within(screen.getByRole("dialog", { name: "Commit message suggestions" })).getByRole("button", { name: "Cancel agent" }));
    await waitFor(() => expect(cancelMessageJob).toHaveBeenCalledWith("repo-1", "job-1", expect.any(String)));
    await screen.findByText("Stopping agent…");
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Human text during run");
    expect(screen.getAllByRole("button", { name: "Cancel agent" }).every(button => button.hasAttribute("disabled"))).toBe(true);
  });

  it("reattaches after reload using the persisted job handle without another Start", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1", requestKey: "request-1", initialMessage: "Before reload" }));
    renderWithProviders(<Harness initial="Saved human text" />);
    await waitFor(() => expect(getMessageJob).toHaveBeenCalledWith("repo-1", "job-1", "request-1"));
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalled());
    expect(startMessage).not.toHaveBeenCalled();
    expect(collectMessage).not.toHaveBeenCalled();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Saved human text");
  });

  it("recovers a lost admission acknowledgement by request key, without dispatching", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ requestKey: "pending-admission", initialMessage: "" }));
    renderWithProviders(<Harness />);
    await waitFor(() => expect(getMessageJob).toHaveBeenCalledWith("repo-1", "", "pending-admission"));
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("bounds observer reattachment when an owner returns unchanged active state immediately", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    vi.mocked(waitMessageJob).mockResolvedValue(job());
    const view = renderWithProviders(<Harness />);
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalled());
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 120)); });
    expect(vi.mocked(waitMessageJob).mock.calls.length).toBeLessThanOrEqual(2);
    view.unmount();
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("stops following a transient owner error and reconnects only on request", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    vi.mocked(waitMessageJob).mockResolvedValueOnce({ ...job(), error: "Agent Manager unavailable" }).mockResolvedValueOnce(job("succeeded"));
    renderWithProviders(<Harness initial="Keep human text" />);
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalledTimes(1));
    await act(async () => { await new Promise(resolve => setTimeout(resolve, 120)); });
    expect(waitMessageJob).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    expect(screen.getByRole("alert")).toHaveTextContent("Status is temporarily unavailable");
    fireEvent.click(screen.getByRole("button", { name: "Reconnect to job" }));
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalledTimes(2));
    await screen.findByText("Agent suggestion");
    expect(startMessage).not.toHaveBeenCalled();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep human text");
  });

  it("shows completed agent output as an unapplied suggestion and preserves edits", async () => {
    let finish: ((value: StagedMessageJob) => void) | undefined;
    vi.mocked(waitMessageJob).mockImplementation((_repo, _id, _key, signal) => new Promise((resolve, reject) => { finish = resolve; signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true }); }));
    renderWithProviders(<Harness initial="Original" />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Drafting agent is running");
    await waitFor(() => expect(finish).toBeTypeOf("function"));
    fireEvent.click(screen.getByRole("button", { name: "Hide progress" }));
    fireEvent.change(screen.getByTestId("commit-message-input"), { target: { value: "Edited while running" } });
    await act(async () => finish?.(job("succeeded")));
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Edited while running");
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    expect(screen.getByText("Agent suggestion")).toBeInTheDocument();
    expect(screen.getByTestId("commit-message-partial-coverage")).toHaveTextContent("Some changes are not described by this result");
    expect(screen.queryByRole("button", { name: "Ask agent to complete" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Check evidence again" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await screen.findByText("Keep your edits in view");
    fireEvent.click(screen.getByRole("button", { name: "Use replacement" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(startMessage).toHaveBeenCalledTimes(1);
  });

  it("retains human edits made while a final applicability check is in flight", async () => {
    let finish: ((value: CheckStagedMessageResponse) => void) | undefined;
    renderWithProviders(<Harness initial="Original" />);
    await gathered();
    await waitFor(() => expect(checkMessage).toHaveBeenCalled());
    vi.mocked(checkMessage).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    fireEvent.click(within(screen.getByRole("dialog", { name: "Commit message suggestions" })).getByRole("button", { name: "Hide suggestions" }));
    fireEvent.change(screen.getByTestId("commit-message-input"), { target: { value: "Last second human edit" } });
    await act(async () => finish?.(create(CheckStagedMessageResponseSchema, { applicable: true })));
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Last second human edit");
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    await screen.findByText("Keep your edits in view");
  });

  it("cannot apply a late suggestion check to another repository's editor", async () => {
    let finish: ((value: CheckStagedMessageResponse) => void) | undefined;
    const view = renderWithProviders(<Harness initial="Original repository text" />);
    await gathered();
    await waitFor(() => expect(checkMessage).toHaveBeenCalled());
    vi.mocked(checkMessage).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    view.rerender(<Harness initial="Original repository text" repo="repo-2" />);
    fireEvent.change(screen.getByTestId("commit-message-input"), { target: { value: "New repository message" } });
    await act(async () => finish?.(create(CheckStagedMessageResponseSchema, { applicable: true })));
    expect(screen.getByTestId("commit-message-input")).toHaveValue("New repository message");
    expect(screen.getByTestId("serialized-message")).toHaveTextContent("New repository message");
  });

  it("rejects an older applicable reply after a complete staging cycle reports a newer index", async () => {
    let finish: ((value: CheckStagedMessageResponse) => void) | undefined;
    const view = renderWithProviders(<Harness initial="Keep this operator message" />);
    await gathered();
    await waitFor(() => expect(checkMessage).toHaveBeenCalled());
    vi.mocked(checkMessage)
      .mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }))
      .mockResolvedValue(create(CheckStagedMessageResponseSchema, { applicable: false, currentSnapshotDigest: "new-index", reason: "New staged content" }));
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    view.rerender(<Harness initial="Keep this operator message" version={2} pending />);
    view.rerender(<Harness initial="Keep this operator message" version={3} />);
    await screen.findByTestId("commit-message-stale");
    await act(async () => finish?.(create(CheckStagedMessageResponseSchema, { applicable: true, currentSnapshotDigest: snapshot.digest })));
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep this operator message");
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("rechecks a newer worktree-only status refresh and still accepts the unchanged staged subject", async () => {
    let finish: ((value: CheckStagedMessageResponse) => void) | undefined;
    const view = renderWithProviders(<Harness initial="Operator message" />);
    await gathered();
    await waitFor(() => expect(checkMessage).toHaveBeenCalled());
    vi.mocked(checkMessage).mockImplementationOnce(() => new Promise(resolve => { finish = resolve; }));
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    view.rerender(<Harness initial="Operator message" version={2} />);
    await waitFor(() => expect(checkMessage).toHaveBeenCalledTimes(3));
    await act(async () => finish?.(create(CheckStagedMessageResponseSchema, { applicable: true, currentSnapshotDigest: snapshot.digest })));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(checkMessage).toHaveBeenCalledTimes(4);
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("ignores a late successful wait response after confirmed cancellation", async () => {
    let finish: ((value: StagedMessageJob) => void) | undefined;
    vi.mocked(waitMessageJob).mockImplementation(() => new Promise(resolve => { finish = resolve; }));
    vi.mocked(cancelMessageJob).mockResolvedValue(job("cancelled"));
    renderWithProviders(<Harness initial="Keep the operator message" />);
    await gathered();
    fireEvent.click(screen.getByRole("button", { name: "Ask agent to complete" }));
    await screen.findByText("Drafting agent is running");
    await waitFor(() => expect(finish).toBeTypeOf("function"));
    fireEvent.click(within(screen.getByRole("dialog", { name: "Commit message suggestions" })).getByRole("button", { name: "Cancel agent" }));
    await screen.findByText("Cancelled", { selector: "h3" });
    await act(async () => finish?.(job("succeeded")));
    expect(screen.queryByText("Agent suggestion")).not.toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep the operator message");
  });

  it("shows failures with retry evidence collection and never restarts an agent automatically", async () => {
    vi.mocked(collectMessage).mockRejectedValue(new Error("Context provider unavailable"));
    renderWithProviders(<Harness initial="Keep text" />);
    suggest();
    await screen.findByRole("alert");
    expect(screen.getByRole("button", { name: "Check evidence again" })).toBeInTheDocument();
    expect(screen.getByTestId("commit-message-input")).toHaveValue("Keep text");
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("exposes failed and unknown owner states without inventing a suggestion", async () => {
    localStorage.setItem("gct.messageDraft.repo-1", JSON.stringify({ jobID: "job-1" }));
    vi.mocked(getMessageJob).mockResolvedValue(job("unsupported-owner-status"));
    renderWithProviders(<Harness initial="Keep me" />);
    await waitFor(() => expect(getMessageJob).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: "View" }));
    await screen.findByText("Execution status unavailable", { selector: "h3" });
    expect(screen.queryByRole("button", { name: "Use suggestion" })).not.toBeInTheDocument();
    expect(startMessage).not.toHaveBeenCalled();
  });

  it("keeps suggestions disabled for empty staging and queued staging operations", () => {
    const view = renderWithProviders(<Harness count={0} />);
    expect(screen.getByRole("button", { name: "Suggest commit title and description" })).toBeDisabled();
    expect(screen.getByText("Stage files to enable suggestions.")).toBeInTheDocument();
    view.rerender(<Harness count={2} pending />);
    expect(screen.getByRole("button", { name: "Suggest commit title and description" })).toBeDisabled();
    expect(collectMessage).not.toHaveBeenCalled();
  });

  it("uses the same keyboard-aware responsive dialog on a mobile viewport", async () => {
    Object.defineProperty(window, "matchMedia", { configurable: true, value: vi.fn().mockImplementation((query: string) => ({ matches: query.includes("max-width"), media: query, addEventListener: vi.fn(), removeEventListener: vi.fn(), addListener: vi.fn(), removeListener: vi.fn() })) });
    renderWithProviders(<Harness />);
    await gathered();
    const dialog = screen.getByRole("dialog", { name: "Commit message suggestions" });
    expect(dialog).toBeInTheDocument();
    expect(dialog.closest("[data-rcl-responsive-dialog]")).toHaveAttribute("data-avoid-keyboard", "true");
    expect(dialog.closest("[data-rcl-responsive-dialog]")).toHaveAttribute("data-presentation", "sheet");
    fireEvent.click(screen.getByRole("button", { name: "Use partial suggestion" }));
    await waitFor(() => expect(screen.getByTestId("commit-message-input")).toHaveValue("fix: retain staged source"));
    expect(startMessage).not.toHaveBeenCalled();
  });
});
