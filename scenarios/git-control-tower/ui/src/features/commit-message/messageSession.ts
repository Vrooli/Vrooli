import type { QueryClient } from "@tanstack/react-query";
import type { StagedMessageCollection, StagedMessageJob, startMessage } from "./api";

type AdmissionRequest = Parameters<typeof startMessage>[0];
export interface DraftSession {
  generation: number;
  phase: "idle" | "collecting" | "collected" | "cancelled";
  collection?: StagedMessageCollection;
  jobID?: string;
  initialMessage: string;
  selectedContextIDs: string[];
  note: string;
  requestKey?: string;
  admissionRequest?: AdmissionRequest;
  error?: string;
  launchError?: { message: string; detail: string };
  staleReason?: string;
  applied?: boolean;
}

export const storageKey = (repo: string) => `gct.messageDraft.${repo || "default"}`;
function restoredAdmission(value: unknown, repo: string, requestKey?: string): AdmissionRequest | undefined {
  if (!value || typeof value !== "object") return;
  const request = value as Partial<AdmissionRequest>;
  const continuation = typeof request.continuationOfJobId === "string" && Boolean(request.continuationOfJobId) && request.continuationOfJobId.length <= 200;
  if (request.repositoryId !== repo || !requestKey || request.requestKey !== requestKey ||
    typeof request.collectionId !== "string" || !request.collectionId && !continuation || request.collectionId.length > 200 ||
    typeof request.snapshotDigest !== "string" || !request.snapshotDigest || request.snapshotDigest.length > 200 ||
    typeof request.operatorNote !== "string" || request.operatorNote.length > 4000 ||
    !Array.isArray(request.selectedContextIds) || request.selectedContextIds.length > 20 ||
    !request.selectedContextIds.every(id => typeof id === "string" && id.length <= 300) ||
    request.continuationOfJobId !== undefined && (typeof request.continuationOfJobId !== "string" || request.continuationOfJobId.length > 200)) return;
  return { repositoryId: repo, collectionId: request.collectionId, snapshotDigest: request.snapshotDigest,
    operatorNote: request.operatorNote, selectedContextIds: [...request.selectedContextIds], requestKey,
    ...(request.continuationOfJobId ? { continuationOfJobId: request.continuationOfJobId } : {}) };
}
export function initialSession(repo: string): DraftSession {
  const empty: DraftSession = { generation: 0, phase: "idle", initialMessage: "", selectedContextIDs: [], note: "" };
  try {
    const saved: unknown = JSON.parse(localStorage.getItem(storageKey(repo)) ?? "null");
    if (saved && typeof saved === "object") {
      const jobID = "jobID" in saved && typeof saved.jobID === "string" ? saved.jobID : undefined;
      const requestKey = "requestKey" in saved && typeof saved.requestKey === "string" ? saved.requestKey : undefined;
      const admissionRequest = restoredAdmission("admissionRequest" in saved ? saved.admissionRequest : undefined, repo, requestKey);
      if (jobID || requestKey) return { ...empty, jobID, requestKey, admissionRequest, initialMessage: "initialMessage" in saved && typeof saved.initialMessage === "string" ? saved.initialMessage : "" };
    }
  } catch { /* A storage failure must never lose the operator's message. */ }
  return empty;
}

export const messageSessionKey = (repo: string) => ["commit-message", "session", repo];

/** Retire browser presentation only. Durable jobs remain at their owner. */
export function resetMessageSession(client: QueryClient, repo: string, expected: DraftSession) {
  const key = messageSessionKey(repo);
  const current = client.getQueryData<DraftSession>(key);
  if (!current || current.generation !== expected.generation) return;
  client.setQueryData<DraftSession>(key, {
    generation: current.generation + 1, phase: "idle", initialMessage: "", selectedContextIDs: [], note: "",
  });
  try { localStorage.removeItem(storageKey(repo)); } catch { /* In-memory reset still succeeds. */ }
}

/** Capture before committing so a newer staging investigation is never cleared. */
export function captureMessageSession(client: QueryClient, repo: string) {
  const key = messageSessionKey(repo);
  const session = client.getQueryData<DraftSession>(key) ?? initialSession(repo);
  if (!client.getQueryData(key)) client.setQueryData(key, session);
  return session;
}

export function retireCommittedMessageSession(client: QueryClient, repo: string, expected: DraftSession) {
  const current = client.getQueryData<DraftSession>(messageSessionKey(repo));
  if (!current || current.generation !== expected.generation) return;
  const handle = current.jobID || current.requestKey;
  if (handle) {
    const job = client.getQueryData<StagedMessageJob>(["commit-message", "job", repo, handle]);
    // A commit does not settle a running/unknown launch. Keep its only handle
    // available for explicit cancellation or reconnect, even when it is stale.
    if (!job || !["succeeded", "failed", "cancelled"].includes(job.status)) return;
  }
  resetMessageSession(client, repo, current);
}
