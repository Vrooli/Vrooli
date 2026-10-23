// [REQ:GCT-OT-P1-002]
import { act, fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { Code, ConnectError } from "@connectrpc/connect";
import { CheckStagedMessageResponseSchema, StagedMessageCollectionSchema, StagedMessageJobSchema, StagedMessageSnapshotSchema, StagedMessageSuggestionSchema } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { renderWithProviders } from "../../test-utils/renderWithProviders";
import { cancelMessageJob, checkMessage, collectMessage, getMessageJob, startMessage, waitMessageJob } from "./api";
import { useMessageSuggestions } from "./useMessageSuggestions";

vi.mock("./api", () => ({ collectMessage: vi.fn(), checkMessage: vi.fn(), startMessage: vi.fn(), getMessageJob: vi.fn(), cancelMessageJob: vi.fn(), waitMessageJob: vi.fn() }));
const snapshot = { repositoryId: "repo", digest: "immutable-selection", files: [{ path: "source.go", newBlobId: "selected" }] };
const ownerJob = (status = "queued") => create(StagedMessageJobSchema, { id: "owner-job", repositoryId: "repo", snapshotDigest: snapshot.digest, snapshot, status, applicable: true });
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
function launchedRequestKey(call = 0) {
  const key = vi.mocked(startMessage).mock.calls[call]?.[0].requestKey;
  if (!key) throw new Error("Expected an explicit launch request");
  return key;
}
function launchedRequest(call: number) {
  const request = vi.mocked(startMessage).mock.calls[call]?.[0];
  if (!request) throw new Error("Expected a recorded drafting admission");
  return request;
}
function Harness() {
  const s = useMessageSuggestions("repo", 1, false);
  return <>
    <output data-testid="admission">{s.admissionState}</output>
    <output data-testid="owner-status">{s.job?.status}</output>
    <output data-testid="cancel-requested">{String(s.job?.cancelRequested ?? false)}</output>
    <output data-testid="request-key">{s.session.requestKey}</output>
    <output data-testid="initial-message">{s.session.initialMessage}</output>
    <output data-testid="phase">{s.session.phase}</output>
    <output data-testid="error">{s.error}</output>
    <output data-testid="launch-detail">{s.session.launchError?.detail}</output>
    <output data-testid="previous-draft">{s.previousSuggestion?.title}</output>
    <output data-testid="previous-digest">{s.previousSnapshot?.digest}</output>
    <output data-testid="can-investigate">{String(s.canInvestigate)}</output>
    <button onClick={() => void s.collect("Human message")}>Collect</button>
    <button onClick={() => void s.start()}>Start</button>
    <button onClick={() => void s.investigateRemaining()}>Investigate</button>
    <button onClick={() => void s.cancelAgent()}>Cancel</button>
    <button onClick={() => void s.reconnect()}>Reconnect</button>
    <button onClick={s.discard}>Discard</button>
    <button onClick={() => s.update({ note: "Changed after launch", selectedContextIDs: ["different-source"] })}>Change intent</button>
  </>;
}
async function collect() {
  fireEvent.click(screen.getByText("Collect"));
  await waitFor(() => expect(screen.getByTestId("phase")).toHaveTextContent("collected"));
}
beforeEach(() => {
  vi.mocked(collectMessage).mockReset().mockResolvedValue(create(StagedMessageCollectionSchema, { id: "collection", snapshot, status: "no_suggestions" }));
  vi.mocked(checkMessage).mockReset().mockResolvedValue(create(CheckStagedMessageResponseSchema, { applicable: true }));
  vi.mocked(startMessage).mockReset().mockResolvedValue(ownerJob());
  vi.mocked(getMessageJob).mockReset().mockResolvedValue(ownerJob());
  vi.mocked(cancelMessageJob).mockReset().mockResolvedValue(ownerJob("cancelling"));
  vi.mocked(waitMessageJob).mockReset().mockImplementation((_repo, _id, _key, signal) => new Promise((_resolve, reject) => signal.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), { once: true })));
});
describe("draft admission and recovery", () => {
  const partialJob = () => create(StagedMessageJobSchema, { ...ownerJob("succeeded"), result: create(StagedMessageSuggestionSchema, { title: "fix: preserve the usable partial draft", coverage: { partial: true, coveredPaths: ["source.go"], uncoveredPaths: ["another.go"] } }) });
  it("does not offer a restored previous draft from another staged subject", async () => {
    localStorage.setItem("gct.messageDraft.repo", JSON.stringify({ requestKey: "continuation-key", admissionRequest: { repositoryId: "repo", collectionId: "", snapshotDigest: snapshot.digest, selectedContextIds: [], requestKey: "continuation-key", operatorNote: "", continuationOfJobId: "other-parent" } }));
    vi.mocked(getMessageJob).mockImplementation(async (_repo, id) => {
      if (id === "other-parent") return create(StagedMessageJobSchema, { ...partialJob(), snapshotDigest: "different-subject", snapshot: create(StagedMessageSnapshotSchema, { ...snapshot, digest: "different-subject" }) });
      throw new ConnectError("not found yet", Code.NotFound);
    });
    renderWithProviders(<Harness />);
    await waitFor(() => expect(getMessageJob).toHaveBeenCalledWith("repo", "other-parent"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    expect(screen.getByTestId("previous-draft")).toBeEmptyDOMElement();
    expect(screen.getByTestId("previous-digest")).toBeEmptyDOMElement();
    expect(screen.getByTestId("can-investigate")).toHaveTextContent("false");
  });
  it("does not create another continuation when the current owner's status is unrecognized", async () => {
    const unknown = create(StagedMessageJobSchema, { ...ownerJob("unknown-owner-state"), id: "expanded-job", continuationOfJobId: "owner-job" });
    vi.mocked(startMessage).mockResolvedValueOnce(partialJob()).mockResolvedValueOnce(unknown);
    vi.mocked(getMessageJob).mockImplementation(async (_repo, id) => id === "expanded-job" ? unknown : partialJob());
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("can-investigate")).toHaveTextContent("true"));
    fireEvent.click(screen.getByText("Investigate"));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("unknown-owner-state"));
    expect(screen.getByTestId("previous-draft")).toHaveTextContent("usable partial draft");
    expect(screen.getByTestId("can-investigate")).toHaveTextContent("false");
    fireEvent.click(screen.getByText("Investigate"));
    expect(startMessage).toHaveBeenCalledTimes(2);
  });

  it("restores a parent-only continuation started from a legacy job handle after a lost acknowledgement", async () => {
    localStorage.setItem("gct.messageDraft.repo", JSON.stringify({ jobID: "owner-job", requestKey: "legacy-key", initialMessage: "Human message" }));
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("lost continuation reply", Code.Unavailable));
    vi.mocked(getMessageJob).mockImplementation(async (_repo, id) => {
      if (id === "owner-job") return partialJob();
      throw new ConnectError("not created yet", Code.NotFound);
    });
    const view = renderWithProviders(<Harness />);
    await waitFor(() => expect(screen.getByTestId("can-investigate")).toHaveTextContent("true"));
    fireEvent.click(screen.getByText("Investigate"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    const original = launchedRequest(0);
    expect(original.collectionId).toBe("");
    expect(original.continuationOfJobId).toBe("owner-job");
    view.unmount();
    renderWithProviders(<Harness />);
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    await waitFor(() => expect(screen.getByTestId("previous-draft")).toHaveTextContent("usable partial draft"));
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(startMessage).toHaveBeenCalledTimes(2));
    expect(launchedRequest(1)).toEqual(original);
    expect(collectMessage).not.toHaveBeenCalled();
  });
  it("starts explicit expanded investigation with a new key and the completed immutable parent", async () => {
    const admitted = deferred<ReturnType<typeof ownerJob>>();
    vi.mocked(startMessage).mockResolvedValueOnce(partialJob()).mockReturnValueOnce(admitted.promise);
    vi.mocked(getMessageJob).mockResolvedValue(partialJob());
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("can-investigate")).toHaveTextContent("true"));
    const original = launchedRequest(0);
    fireEvent.click(screen.getByText("Change intent"));
    fireEvent.click(screen.getByText("Investigate"));
    expect(screen.getByTestId("admission")).toHaveTextContent("starting");
    await waitFor(() => expect(screen.getByTestId("previous-draft")).toHaveTextContent("usable partial draft"));
    const continuation = launchedRequest(1);
    expect(continuation).toEqual({ ...original, collectionId: "", selectedContextIds: [], operatorNote: "", requestKey: expect.any(String), continuationOfJobId: "owner-job" });
    expect(continuation.requestKey).not.toBe(original.requestKey);
    expect(screen.getByTestId("previous-digest")).toHaveTextContent(snapshot.digest);
    fireEvent.click(screen.getByText("Investigate"));
    fireEvent.click(screen.getByText("Collect"));
    expect(startMessage).toHaveBeenCalledTimes(2);
    expect(collectMessage).toHaveBeenCalledTimes(1);
    const running = create(StagedMessageJobSchema, { ...ownerJob("running"), id: "expanded-job", continuationOfJobId: "owner-job" });
    vi.mocked(getMessageJob).mockImplementation(async (_repo, id) => id === "owner-job" ? partialJob() : running);
    await act(async () => admitted.resolve(running));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("running"));
    expect(screen.getByTestId("can-investigate")).toHaveTextContent("false");
    expect(screen.getByTestId("previous-draft")).toHaveTextContent("usable partial draft");
  });

  it.each(["failed", "cancelled"])("retains the previous usable result after expanded investigation is %s", async status => {
    const continuation = create(StagedMessageJobSchema, { ...ownerJob(status), id: "expanded-job", continuationOfJobId: "owner-job" });
    vi.mocked(startMessage).mockResolvedValueOnce(partialJob()).mockResolvedValueOnce(continuation);
    vi.mocked(getMessageJob).mockImplementation(async (_repo, id) => id === "expanded-job" ? continuation : partialJob());
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("can-investigate")).toHaveTextContent("true"));
    fireEvent.click(screen.getByText("Investigate"));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent(status));
    expect(screen.getByTestId("previous-draft")).toHaveTextContent("usable partial draft");
    expect(screen.getByTestId("can-investigate")).toHaveTextContent("true");
    expect(JSON.parse(localStorage.getItem("gct.messageDraft.repo") ?? "{}").admissionRequest.continuationOfJobId).toBe("owner-job");
  });

  it("recovers a lost continuation acknowledgement after reload with the same parent, digest and key", async () => {
    vi.mocked(startMessage).mockResolvedValueOnce(partialJob()).mockRejectedValue(new ConnectError("lost continuation reply", Code.Unavailable));
    vi.mocked(getMessageJob).mockImplementation(async (_repo, id) => {
      if (id === "owner-job") return partialJob();
      throw new ConnectError("not created yet", Code.NotFound);
    });
    const view = renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("can-investigate")).toHaveTextContent("true"));
    fireEvent.click(screen.getByText("Investigate"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    const original = launchedRequest(1);
    view.unmount();
    renderWithProviders(<Harness />);
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    await waitFor(() => expect(screen.getByTestId("previous-draft")).toHaveTextContent("usable partial draft"));
    fireEvent.click(screen.getByText("Discard"));
    fireEvent.click(screen.getByText("Collect"));
    fireEvent.click(screen.getByText("Change intent"));
    fireEvent.click(screen.getByText("Investigate"));
    expect(startMessage).toHaveBeenCalledTimes(2);
    expect(collectMessage).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(startMessage).toHaveBeenCalledTimes(3));
    expect(launchedRequest(2)).toEqual(original);
  });

  it("does not repeat the original sampled request after a completed result", async () => {
    vi.mocked(startMessage).mockResolvedValue(ownerJob("succeeded"));
    vi.mocked(getMessageJob).mockResolvedValue(ownerJob("succeeded"));
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("succeeded"));
    fireEvent.click(screen.getByText("Start"));
    expect(startMessage).toHaveBeenCalledTimes(1);
  });

  it("acknowledges a delayed launch immediately and retains its key while duplicate actions are blocked", async () => {
    const admission = deferred<ReturnType<typeof ownerJob>>();
    vi.mocked(startMessage).mockReturnValue(admission.promise);
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    expect(screen.getByTestId("admission")).toHaveTextContent("starting");
    const requestKey = launchedRequestKey();
    expect(JSON.parse(localStorage.getItem("gct.messageDraft.repo") ?? "{}").requestKey).toBe(requestKey);
    fireEvent.click(screen.getByText("Start"));
    fireEvent.click(screen.getByText("Discard"));
    fireEvent.click(screen.getByText("Collect"));
    expect(startMessage).toHaveBeenCalledTimes(1);
    expect(collectMessage).toHaveBeenCalledTimes(1);
    expect(getMessageJob).not.toHaveBeenCalled();
    await act(async () => admission.resolve(ownerJob()));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("confirmed"));
    expect(screen.getByTestId("request-key")).toHaveTextContent(requestKey);
    expect(screen.getByTestId("owner-status")).toHaveTextContent("queued");
    expect(screen.getByTestId("initial-message")).toHaveTextContent("Human message");
  });

  it("recovers a lost acknowledgement without discarding the unresolved handle or dispatching again", async () => {
    const recovery = deferred<ReturnType<typeof ownerJob>>();
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("transport closed", Code.Unavailable));
    vi.mocked(getMessageJob).mockReturnValueOnce(recovery.promise);
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("recovering"));
    const requestKey = launchedRequestKey();
    expect(getMessageJob).toHaveBeenCalledWith("repo", "", requestKey);
    await act(async () => recovery.reject(new ConnectError("owner read unavailable", Code.Unavailable)));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("unknown"));
    fireEvent.click(screen.getByText("Discard"));
    fireEvent.click(screen.getByText("Collect"));
    fireEvent.click(screen.getByText("Start"));
    expect(screen.getByTestId("request-key")).toHaveTextContent(requestKey);
    expect(startMessage).toHaveBeenCalledTimes(1);
    expect(collectMessage).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("error")).toHaveTextContent("Could not confirm the drafting launch");
    vi.mocked(getMessageJob).mockResolvedValue(ownerJob("running"));
    fireEvent.click(screen.getByText("Reconnect"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("confirmed"));
    await waitFor(() => expect(screen.getByTestId("error")).toBeEmptyDOMElement());
    expect(screen.getByTestId("launch-detail")).toBeEmptyDOMElement();
    expect(startMessage).toHaveBeenCalledTimes(1);
  });

  it("confirms no admission only after typed NotFound and keeps sign-in detail separate from human copy", async () => {
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("401 Unauthorized at provider", Code.Unauthenticated));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("no matching job", Code.NotFound));
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    expect(screen.getByTestId("error")).toHaveTextContent("Sign-in is required to confirm");
    expect(screen.getByTestId("error")).not.toHaveTextContent("401");
    expect(screen.getByTestId("launch-detail")).toHaveTextContent("401 Unauthorized");
    const key = launchedRequestKey();
    fireEvent.click(screen.getByText("Discard"));
    fireEvent.click(screen.getByText("Collect"));
    expect(screen.getByTestId("request-key")).toHaveTextContent(key);
    expect(collectMessage).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByText("Change intent"));
    vi.mocked(startMessage).mockResolvedValue(ownerJob());
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(startMessage).toHaveBeenCalledTimes(2));
    expect(launchedRequestKey(1)).toBe(key);
    expect(vi.mocked(startMessage).mock.calls[1]?.[0]).toEqual(vi.mocked(startMessage).mock.calls[0]?.[0]);
  });

  it("keeps a point-in-time NotFound handle until a late original admission is recovered", async () => {
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("acknowledgement lost", Code.Unavailable));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("not created yet", Code.NotFound));
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    const key = launchedRequestKey();
    fireEvent.click(screen.getByText("Discard"));
    fireEvent.click(screen.getByText("Collect"));
    expect(JSON.parse(localStorage.getItem("gct.messageDraft.repo") ?? "{}").requestKey).toBe(key);
    expect(collectMessage).toHaveBeenCalledTimes(1);
    vi.mocked(getMessageJob).mockResolvedValue(ownerJob("running"));
    fireEvent.click(screen.getByText("Reconnect"));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("running"));
    expect(startMessage).toHaveBeenCalledTimes(1);
  });

  it("restores frozen admission intent after reload and retries only that same request", async () => {
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("lost reply", Code.Unavailable));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("not admitted yet", Code.NotFound));
    const view = renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    const original = vi.mocked(startMessage).mock.calls[0]?.[0];
    view.unmount();
    vi.mocked(startMessage).mockResolvedValue(ownerJob());
    renderWithProviders(<Harness />);
    await waitFor(() => expect(screen.getByTestId("admission")).toHaveTextContent("not_found"));
    fireEvent.click(screen.getByText("Change intent"));
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(startMessage).toHaveBeenCalledTimes(2));
    expect(vi.mocked(startMessage).mock.calls[1]?.[0]).toEqual(original);
    expect(collectMessage).toHaveBeenCalledTimes(1);
  });

  it("turns the bounded trusted-browser precondition into actionable sign-in copy", async () => {
    vi.mocked(startMessage).mockRejectedValue(new ConnectError("Sign in through a trusted browser session before starting a drafting agent", Code.FailedPrecondition));
    vi.mocked(getMessageJob).mockRejectedValue(new ConnectError("no job yet", Code.NotFound));
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("error")).toHaveTextContent("Sign-in is required"));
    expect(screen.getByTestId("launch-detail")).toHaveTextContent("trusted browser session");
  });

  it("preserves authoritative queued, running, cancellation and terminal failure states", async () => {
    const running = deferred<ReturnType<typeof ownerJob>>();
    const terminal = deferred<ReturnType<typeof ownerJob>>();
    vi.mocked(waitMessageJob).mockReturnValueOnce(running.promise).mockReturnValue(terminal.promise);
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("queued"));
    await act(async () => running.resolve(ownerJob("running")));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("running"));
    fireEvent.click(screen.getByText("Cancel"));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("cancelling"));
    // Owner disposition can be failure as well as cancelled; neither creates a
    // result or overwrites the operator's message.
    await act(async () => terminal.resolve(ownerJob("failed")));
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent("failed"));
    expect(screen.getByTestId("admission")).toHaveTextContent("confirmed");
    expect(screen.getByTestId("initial-message")).toHaveTextContent("Human message");
    expect(startMessage).toHaveBeenCalledTimes(1);
  });

  it.each(["cancelled", "cancelling"])("keeps confirmed %s when an older Wait reply resolves before React effect cleanup", async cancellationStatus => {
    const waiting = deferred<ReturnType<typeof ownerJob>>();
    const cancellation = deferred<ReturnType<typeof ownerJob>>();
    vi.mocked(waitMessageJob).mockReturnValue(waiting.promise);
    vi.mocked(cancelMessageJob).mockReturnValue(cancellation.promise);
    renderWithProviders(<Harness />);
    await collect();
    fireEvent.click(screen.getByText("Start"));
    await waitFor(() => expect(waitMessageJob).toHaveBeenCalled());
    fireEvent.click(screen.getByText("Cancel"));
    await act(async () => {
      cancellation.resolve({ ...ownerJob(cancellationStatus), cancelRequested: true });
      await Promise.resolve();
      await Promise.resolve();
      waiting.resolve(ownerJob("running"));
    });
    await waitFor(() => expect(screen.getByTestId("owner-status")).toHaveTextContent(cancellationStatus));
    expect(screen.getByTestId("cancel-requested")).toHaveTextContent("true");
    expect(startMessage).toHaveBeenCalledTimes(1);
  });
});
