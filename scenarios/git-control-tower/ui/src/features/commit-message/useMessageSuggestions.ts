import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { isLimitedDraft } from "./message";
import {
  cancelMessageJob, checkMessage, collectMessage, getMessageJob, startMessage, waitMessageJob,
  type StagedMessageJob,
} from "./api";
import { initialSession, messageSessionKey, resetMessageSession, storageKey, type DraftSession } from "./messageSession";
type AdmissionRequest = Parameters<typeof startMessage>[0];

const activeStates = new Set(["queued", "running", "cancelling"]);
const autoReconnectLimit = 8;
export function isMessageJobActive(status?: string) { return activeStates.has(status ?? ""); }
function retainOwnerProgress(current: StagedMessageJob | undefined, next: StagedMessageJob) {
  if (!current || current.id !== next.id) return next;
  if (["succeeded", "failed", "cancelled"].includes(current.status) && current.status !== next.status) return current;
  if ((current.cancelRequested || current.status === "cancelling") && isMessageJobActive(next.status) && !next.cancelRequested) return current;
  return next;
}
const errorText = (value: unknown) => value instanceof Error ? value.message : "The suggestion request could not finish.";
function launchError(value: unknown) {
  const failure = ConnectError.from(value);
  const signIn = failure.code === Code.Unauthenticated || failure.code === Code.FailedPrecondition && failure.rawMessage === "Sign in through a trusted browser session before starting a drafting agent";
  const message = signIn ? "Sign-in is required to confirm the drafting launch."
    : failure.code === Code.PermissionDenied ? "Your current access could not confirm the drafting launch."
    : "Could not confirm the drafting launch. Reconnect to check whether a job was created.";
  return { message, detail: errorText(value) };
}
function pauseObserver(milliseconds: number, signal: AbortSignal) {
  if (milliseconds <= 0 || signal.aborted) return Promise.resolve();
  return new Promise<void>(resolve => {
    const done = () => { clearTimeout(timer); signal.removeEventListener("abort", done); resolve(); };
    const timer = setTimeout(done, milliseconds);
    signal.addEventListener("abort", done, { once: true });
  });
}

/** Job ownership is server-side. Query cache preserves navigation, and the
 * repository-keyed handle reattaches after reload without another Start. */
export function useMessageSuggestions(repo: string, statusVersion: number, updatingIndex: boolean) {
  const client = useQueryClient();
  const key = useMemo(() => messageSessionKey(repo), [repo]);
  const { data: session } = useQuery({ queryKey: key, queryFn: () => initialSession(repo), initialData: () => initialSession(repo), enabled: false, gcTime: Infinity });
  const isCurrentSession = useCallback((generation = session.generation) =>
    client.getQueryData<DraftSession>(key)?.generation === generation, [client, key, session.generation]);
  const update = useCallback((patch: Partial<DraftSession>, generation = session.generation) => {
    client.setQueryData<DraftSession>(key, current => current?.generation === generation ? { ...current, ...patch } : current);
  }, [client, key, session.generation]);
  const controller = useRef<AbortController>();
  const requestSequence = useRef(0);
  const invalidateCollection = useCallback(() => {
    ++requestSequence.current;
    controller.current?.abort();
  }, []);
  const actionPending = useRef(false);
  const mounted = useRef(true);
  const [action, setAction] = useState<"start" | "cancel" | "check" | null>(null);
  const [observerAttempt, setObserverAttempt] = useState(0);
  const [open, setOpen] = useState(false);
  const job = useQuery({
    queryKey: ["commit-message", "job", repo, session.jobID || session.requestKey],
    queryFn: async () => {
      const result = await getMessageJob(repo, session.jobID ?? "", session.requestKey);
      return retainOwnerProgress(client.getQueryData(["commit-message", "job", repo, session.jobID || session.requestKey]), result);
    },
    enabled: Boolean(session.jobID || session.requestKey) && action !== "start",
    retry: false,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });
  const hasHandle = Boolean(session.jobID || session.requestKey);
  const admissionState = action === "start" ? "starting" : job.data ? "confirmed"
    : !hasHandle ? "idle" : job.isFetching ? "recovering"
    : job.error && ConnectError.from(job.error).code === Code.NotFound ? "not_found" : "unknown";
  const unresolvedAdmission = hasHandle && !job.data;
  // The prior result stays at its durable owner. Browser storage retains only
  // the frozen admission, never a second potentially multi-megabyte snapshot.
  const previousJobID = job.data?.continuationOfJobId || session.admissionRequest?.continuationOfJobId;
  const previousJob = useQuery({
    queryKey: ["commit-message", "job", repo, previousJobID],
    queryFn: () => getMessageJob(repo, previousJobID ?? ""),
    enabled: Boolean(previousJobID), retry: false, staleTime: Infinity, refetchOnWindowFocus: false,
  });
  const pinnedDigest = job.data?.snapshotDigest || session.admissionRequest?.snapshotDigest;
  const previousMatches = previousJob.data?.repositoryId === repo && Boolean(pinnedDigest) && previousJob.data.snapshotDigest === pinnedDigest && previousJob.data.snapshot?.digest === pinnedDigest;
  const previousSuggestion = previousMatches && previousJob.data?.status === "succeeded" ? previousJob.data.result : undefined;
  useEffect(() => {
    if (job.data && session.launchError) update({ error: undefined, launchError: undefined });
  }, [job.data, session.launchError, update]);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
      invalidateCollection();
      const current = client.getQueryData<DraftSession>(key);
      if (current?.phase === "collecting") client.setQueryData(key, { ...current, phase: "cancelled" });
    };
  }, [client, key, invalidateCollection]);
  const observedGeneration = useRef(session.generation);
  useEffect(() => {
    if (observedGeneration.current === session.generation) return;
    observedGeneration.current = session.generation;
    if (session.phase !== "idle") return;
    invalidateCollection();
    verifiedEpoch.current = undefined;
    setOpen(false);
    setAction(null);
  }, [session.generation, session.phase, invalidateCollection]);
  useEffect(() => {
    if (!isMessageJobActive(job.data?.status) || job.data?.error) return;
    const wait = new AbortController();
    const handle = session.jobID || session.requestKey;
    const follow = async () => {
      let unchanged = 0;
      let previous = "";
      try {
        while (!wait.signal.aborted) {
          const began = Date.now();
          const result = await waitMessageJob(repo, session.jobID ?? "", session.requestKey ?? "", wait.signal);
          if (wait.signal.aborted || !isCurrentSession()) return;
          const observed = client.setQueryData<StagedMessageJob>(["commit-message", "job", repo, handle], current => retainOwnerProgress(current, result));
          if (!observed || !isMessageJobActive(observed.status)) return;
          if (observed.error) {
            update({ error: `Execution status is temporarily unavailable: ${observed.error}` });
            return;
          }
          const fingerprint = `${observed.status}\u0000${observed.phase}\u0000${observed.cancelRequested}`;
          unchanged = fingerprint === previous ? unchanged + 1 : 0;
          previous = fingerprint;
          // A healthy Wait blocks at the owner. Bound reattachment if an
          // unavailable/misbehaving provider returns unchanged state early.
          await pauseObserver(Math.min(8000, 1000 * 2 ** Math.min(unchanged, 3)) - (Date.now() - began), wait.signal);
        }
      } catch (error) {
        if (!wait.signal.aborted) update({ error: `Could not follow the existing drafting job: ${errorText(error)}` });
      }
    };
    void follow();
    // Cancels only the observer request. AM continues to own execution.
    return () => wait.abort();
  }, [client, repo, session.jobID, session.requestKey, job.data?.status, job.data?.error, update, observerAttempt, isCurrentSession]);
  const snapshot = job.data?.snapshot ?? session.collection?.snapshot ?? (previousMatches ? previousJob.data?.snapshot : undefined);
  const digest = job.data?.snapshotDigest || snapshot?.digest || "";
  const standing = useQuery({
    queryKey: ["commit-message", "standing", repo, digest, statusVersion],
    queryFn: () => checkMessage(repo, digest),
    enabled: Boolean(digest) && !updatingIndex,
    retry: false,
    staleTime: 0,
  });
  // A true reply is only usable for the observation generation that requested
  // it. A complete stage/unstage cycle can occur while that reply is in flight.
  // A newer rejection invalidates an in-flight acceptance. A successful
  // background observation adds no new invalidation: its notification can
  // arrive during the direct check for the same staged subject.
  const standingRejection = useRef(0);
  if (standing.data?.applicable === false) standingRejection.current = standing.dataUpdatedAt;
  const standingVersion = standingRejection.current;
  const verificationGeneration = useRef({ repo, digest, statusVersion, standingVersion, updatingIndex, epoch: 0 });
  const verifiedEpoch = useRef<number>();
  const previous = verificationGeneration.current;
  if (previous.repo !== repo || previous.digest !== digest || previous.statusVersion !== statusVersion || previous.standingVersion !== standingVersion || previous.updatingIndex !== updatingIndex) {
    verificationGeneration.current = { repo, digest, statusVersion, standingVersion, updatingIndex, epoch: previous.epoch + 1 };
  }
  const staleReason = session.staleReason || (standing.data && !standing.data.applicable ? standing.data.reason || "The staged selection has changed." : "") || (job.data && !job.data.applicable ? job.data.applicabilityReason || "The staged selection has changed." : "");

  const collect = async (message: string) => {
    if (updatingIndex || isMessageJobActive(job.data?.status) || actionPending.current || unresolvedAdmission) return;
    controller.current?.abort();
    const sequence = ++requestSequence.current;
    const abort = new AbortController();
    controller.current = abort;
    try { localStorage.removeItem(storageKey(repo)); } catch { /* Old terminal handle is no longer active. */ }
    const generation = (client.getQueryData<DraftSession>(key)?.generation ?? session.generation) + 1;
    client.setQueryData<DraftSession>(key, { generation, phase: "idle", initialMessage: "", selectedContextIDs: [], note: "" });
    setOpen(true);
    update({ phase: "collecting", initialMessage: message, error: undefined, launchError: undefined, staleReason: undefined, jobID: undefined, collection: undefined, requestKey: undefined, admissionRequest: undefined, selectedContextIDs: [], applied: false }, generation);
    try {
      const collection = await collectMessage(repo, abort.signal);
      if (sequence === requestSequence.current) {
        const retained = new Set(collection.suggestions[0]?.sourceIds ?? []);
        const selectedContextIDs = collection.contexts.filter(context =>
          retained.has(context.id) && context.kind === "proposal" && context.fullMatch && context.status === "retained_edits_included",
        ).slice(0, 20).map(context => context.id);
        update({ phase: "collected", collection, selectedContextIDs }, generation);
      }
    } catch (error) {
      if (sequence === requestSequence.current && !abort.signal.aborted) update({ phase: "idle", error: errorText(error) }, generation);
    }
  };
  const cancelCollection = () => {
    invalidateCollection();
    update({ phase: "cancelled" });
  };
  const verify = async () => {
    if (!digest || updatingIndex) return false;
    verifiedEpoch.current = undefined;
    setAction("check");
    try {
      for (let attempt = 0; attempt < 2; attempt++) {
        const generation = verificationGeneration.current;
        if (!isCurrentSession() || generation.updatingIndex || generation.repo !== repo || generation.digest !== digest) return false;
        const result = await checkMessage(repo, digest);
        if (!mounted.current || !isCurrentSession()) return false;
        if (generation.epoch !== verificationGeneration.current.epoch) {
          // A status refresh may describe only working-tree changes. Recheck
          // the actual staged digest once instead of permanently staling it.
          continue;
        }
        verifiedEpoch.current = result.applicable ? generation.epoch : undefined;
        update({ staleReason: result.applicable ? undefined : result.reason || "The staged selection has changed.", error: undefined });
        return result.applicable;
      }
      update({ error: "Repository status changed while checking the staged selection. Review it and try again; your message is unchanged." });
      return false;
    } catch (error) {
      if (mounted.current) update({ error: `Could not confirm the staged selection: ${errorText(error)}` });
      return false;
    } finally { if (mounted.current && isCurrentSession()) setAction(null); }
  };
  const dispatch = async (request: AdmissionRequest, selected: DraftSession) => {
    actionPending.current = true;
    setAction("start");
    const requestKey = request.requestKey;
    update({ requestKey, admissionRequest: request, jobID: undefined, error: undefined, launchError: undefined, applied: false });
    try { localStorage.setItem(storageKey(repo), JSON.stringify({ requestKey, admissionRequest: request, initialMessage: selected.initialMessage })); } catch { /* Admission remains server-owned. */ }
    try {
      const created = await startMessage(request);
      if (!isCurrentSession()) return;
      client.setQueryData(["commit-message", "job", repo, created.id], created);
      update({ jobID: created.id });
      try { localStorage.setItem(storageKey(repo), JSON.stringify({ jobID: created.id, requestKey, admissionRequest: request, initialMessage: selected.initialMessage })); } catch { /* Server job remains authoritative. */ }
    } catch (error) {
      const failure = launchError(error);
      update({ error: failure.message, launchError: failure });
    }
    finally { actionPending.current = false; if (isCurrentSession()) setAction(null); }
  };
  const start = async () => {
    // React Query batches notifications. Read the authoritative local selection
    // when the operator clicks so the final keystroke/checkbox is included.
    const selected = client.getQueryData<DraftSession>(key) ?? session;
    if ((!selected.collection?.snapshot && !selected.admissionRequest) || updatingIndex || actionPending.current || job.data || unresolvedAdmission && admissionState !== "not_found") return;
    // A NotFound read is a point-in-time observation, not cancellation proof.
    // Reuse the exact frozen intent even if the original admission arrives late.
    if (selected.requestKey && !selected.admissionRequest) {
      update({ error: "Reconnect to check the original drafting request. Its launch details are unavailable, so a new request has not been sent." });
      return;
    }
    const request = selected.admissionRequest ?? (selected.collection?.snapshot ? {
      repositoryId: repo, collectionId: selected.collection.id, snapshotDigest: selected.collection.snapshot.digest,
      selectedContextIds: [...selected.selectedContextIDs], operatorNote: selected.note, requestKey: crypto.randomUUID(),
    } : undefined);
    if (request) await dispatch(request, selected);
  };
  const continuationSource = job.data?.status === "succeeded" && job.data.result ? job.data : previousSuggestion ? previousJob.data : undefined;
  const canInvestigate = Boolean(isLimitedDraft(continuationSource?.result) && job.data && ["succeeded", "failed", "cancelled"].includes(job.data.status) && !unresolvedAdmission);
  const investigateRemaining = async () => {
    if (!canInvestigate || !continuationSource || updatingIndex || actionPending.current || staleReason) return;
    const selected = client.getQueryData<DraftSession>(key) ?? session;
    // The owner derives expanded evidence from this completed immutable job.
    // Never recollect the index or resubmit the original representative sample.
    await dispatch({ repositoryId: repo, collectionId: "", snapshotDigest: continuationSource.snapshotDigest,
      selectedContextIds: [], operatorNote: "", requestKey: crypto.randomUUID(), continuationOfJobId: continuationSource.id }, selected);
  };
  const cancelAgent = async () => {
    if ((!session.jobID && !session.requestKey) || actionPending.current) return;
    actionPending.current = true;
    setAction("cancel");
    try {
      const cancelled = await cancelMessageJob(repo, session.jobID ?? "", session.requestKey);
      client.setQueryData<StagedMessageJob>(["commit-message", "job", repo, session.jobID || session.requestKey], current => retainOwnerProgress(current, cancelled));
      update({ error: undefined });
    } catch (error) { update({ error: errorText(error) }); }
    finally { actionPending.current = false; if (isCurrentSession()) setAction(null); }
  };
  const discard = () => {
    if (isMessageJobActive(job.data?.status) || actionPending.current || unresolvedAdmission) return;
    resetMessageSession(client, repo, session);
    setOpen(false);
  };
  const refetchJob = job.refetch;
  // After the operator narrows the index back to the suggestion's files, the
  // owner re-derives applicability; a retained stale verdict must not stick.
  const recheckApplicability = useCallback(async () => {
    update({ staleReason: undefined, error: undefined });
    if (hasHandle) await refetchJob();
  }, [update, refetchJob, hasHandle]);
  const reconnect = useCallback(async () => {
    update({ error: undefined });
    await refetchJob();
    setObserverAttempt(value => value + 1);
  }, [update, refetchJob]);
  // Reads are side-effect free, so a lost connection to an existing job is
  // retried automatically: on a bounded backoff, and immediately when the
  // operator returns to the page. The manual control remains a fallback.
  const lostConnection = hasHandle && action === null && !job.isFetching &&
    (admissionState === "unknown" || admissionState === "not_found" || isMessageJobActive(job.data?.status) && Boolean(session.error || job.data?.error));
  const connected = !hasHandle || Boolean(job.data && !session.error && !job.data.error);
  const reconnectAttempt = useRef(0);
  if (connected) reconnectAttempt.current = 0;
  useEffect(() => {
    if (!lostConnection || reconnectAttempt.current >= autoReconnectLimit) return;
    const timer = setTimeout(() => { reconnectAttempt.current++; void reconnect(); }, Math.min(30_000, 2000 * 2 ** reconnectAttempt.current));
    return () => clearTimeout(timer);
  }, [lostConnection, reconnect, observerAttempt]);
  const following = isMessageJobActive(job.data?.status);
  useEffect(() => {
    if (!lostConnection && !following) return;
    const resume = () => {
      if (document.visibilityState !== "visible") return;
      reconnectAttempt.current = 0;
      // A suspended page can leave its long poll silently stalled. Reattach
      // the observer even when no error was reported.
      if (lostConnection) void reconnect();
      else setObserverAttempt(value => value + 1);
    };
    document.addEventListener("visibilitychange", resume);
    window.addEventListener("online", resume);
    return () => {
      document.removeEventListener("visibilitychange", resume);
      window.removeEventListener("online", resume);
    };
  }, [lostConnection, following, reconnect]);
  const canAdoptVerified = () => isCurrentSession() && mounted.current && !verificationGeneration.current.updatingIndex && !unresolvedAdmission && !isMessageJobActive(job.data?.status) && action !== "start" && action !== "cancel" && verifiedEpoch.current === verificationGeneration.current.epoch;
  return { session, job: job.data, snapshot, previousSuggestion, previousSnapshot: previousMatches ? previousJob.data?.snapshot : undefined, canInvestigate, investigateRemaining, staleReason, error: session.error || job.data?.error || job.error?.message || standing.error?.message, action, admissionState, canRetryAdmission: admissionState === "not_found" && Boolean(session.admissionRequest), open, setOpen, update, collect, cancelCollection, start, cancelAgent, verify, canAdoptVerified, discard, reconnect, recheckApplicability, reconnecting: hasHandle && job.isPending };
}
