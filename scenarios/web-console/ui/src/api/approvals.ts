// Approvals inbox client.
//
// The inbox is web-console's single surface for every human-approval decision
// in the fleet, no matter which scenario raised it. Those decisions all ride
// notification-hub's domain-neutral ask substrate (operatordecision →
// ConversationsService).
//
// TRANSPORT: this is a REST client (a sibling of api/notifications.ts), NOT a
// Connect client. notification-hub's ConversationsService is Connect-RPC only,
// so Web Console's Go facade (api/approvals_facade.go) owns the Connect call and
// exposes thin REST routes the browser calls same-origin:
//   GET  /api/v1/approvals          — list the operator's asks
//   POST /api/v1/approvals/answer   — answer one ask (approve/reject/any option)
// Keeping the hub call in the Go facade is deliberate: that is the one place
// that injects the operator Bearer AND the Cloudflare Access assertion onto the
// outbound call so the hub attributes an answer to the verified human
// (answered_by, Phase 7). The UI never speaks Connect to the hub and never
// learns its URL.
//
// DECOUPLING (load-bearing): this imports NOTHING from agent-manager, even
// though agent-manager raises some asks (run review, authorization scope,
// effort decisions) alongside deployment-manager, swarm-manager and
// vrooli-autoheal. Each arrives as an ask keyed by `source`; the inbox never
// reaches a producer. api/__tests__/approvals.decoupling.test.ts fails the build
// if an agent-manager import (or a Connect transport to the hub) ever appears
// here. See docs/internal/SEAMS.md and the ratified mockups in
// docs/mockups/approvals/.

import { buildApiUrl } from "@vrooli/api-base";
import { deviceIdentity } from "../lib/deviceIdentity";
import { getAccessToken } from "../lib/auth";
import { API_BASE, makeApiError } from "./client";

/** The scenarios that raise asks. Drives the per-source context panel + icon. */
export type ApprovalSource =
  | "agent-manager"
  | "deployment-manager"
  | "swarm-manager"
  | "vrooli-autoheal"
  | "notification-hub"
  | "unknown";

/** An ask's lifecycle state as the UI treats it. */
export type ApprovalState = "pending" | "answered" | "expired" | "unknown";

/** One answer option in a decision's menu. */
export interface ApprovalOption {
  key: string;
  label: string;
}

/**
 * ApprovalDecision is the UI's view of a hub ask. It is the ONE content model
 * the list and detail both render (modeled once, composed twice — see the
 * mockups README). Fields map to the facade's JSON (api/approvals_facade.go
 * approvalView); the UI derives `source` and `kind` so components never parse
 * the event type themselves.
 */
export interface ApprovalDecision {
  id: string;
  question: string;
  options: ApprovalOption[];
  /** Option key the producer recommends; the detail renders it as the primary action. */
  recommended: string;
  recommendationReason: string;
  /** Applied by the hub at the deadline when nothing is answered; only ever set for reversible asks. */
  defaultAnswer: string;
  reversible: boolean;
  /** "", "low", "normal", "high", "critical". */
  urgency: string;
  /** Read-only deep link to the producer's own review surface (never an inline producer call). */
  contextUrl: string;
  /** Optional requester-authored purpose, stakes, and practical outcome. */
  bigPicture: string;
  /** Optional requester-authored mechanism, evidence, and limitations. */
  technicalDetails: string;
  /** RFC3339; empty when the ask never had a deadline. */
  deadline: string;
  state: ApprovalState;
  answer: string;
  answerLabel: string;
  note: string;
  /** The verified human who answered (Cloudflare Access subject), when known. */
  answeredBy: string;
  answeredAt: string;
  late: boolean;
  defaultAppliedAt: string;
  resolvedAt: string;
  createdAt: string;
  updatedAt: string;
  /** The raising scenario, classified for the context panel. */
  source: ApprovalSource;
  /** Raw `<source>.<kind>.decision_requested.v1` event type. */
  sourceEventType: string;
  /** The decision KIND parsed from the event type (release, run-review, …). */
  kind: string;
}

/** The facade's JSON row shape (snake_case), 1:1 with approvalView in Go. */
interface ApprovalRowJSON {
  id?: string;
  question?: string;
  options?: { key?: string; label?: string }[];
  recommended?: string;
  recommendation_reason?: string;
  default_answer?: string;
  reversible?: boolean;
  urgency?: string;
  context_url?: string;
  big_picture?: string;
  technical_details?: string;
  deadline?: string;
  state?: string;
  answer?: string;
  answer_label?: string;
  note?: string;
  answered_by?: string;
  answered_at?: string;
  late?: boolean;
  default_applied_at?: string;
  resolved_at?: string;
  created_at?: string;
  updated_at?: string;
  source?: string;
  source_event_type?: string;
  requester_ref?: string;
}

const KNOWN_SOURCES: ApprovalSource[] = [
  "agent-manager",
  "deployment-manager",
  "swarm-manager",
  "vrooli-autoheal",
  "notification-hub",
];

function classifySource(raw: string | undefined): ApprovalSource {
  const trimmed = (raw ?? "").trim();
  return (KNOWN_SOURCES as string[]).includes(trimmed) ? (trimmed as ApprovalSource) : "unknown";
}

/** Parse the KIND out of `<source>.<kind>.decision_requested.v1`. */
function kindFromEventType(eventType: string | undefined, source: string | undefined): string {
  const et = (eventType ?? "").trim();
  if (!et) return "";
  let rest = et;
  const src = (source ?? "").trim();
  if (src && rest.startsWith(src + ".")) rest = rest.slice(src.length + 1);
  rest = rest.replace(/\.decision_requested\.v\d+$/, "");
  return rest;
}

/**
 * Derive the display state. The hub carries "pending" / "answered"; a still-
 * pending ask past its deadline reads as expired so the UI can style it with
 * the error palette and disable its actions.
 */
function deriveState(row: ApprovalRowJSON): ApprovalState {
  const raw = (row.state ?? "").trim();
  if (raw === "answered" || row.answered_at) return "answered";
  if (raw === "pending") {
    if (row.deadline) {
      const dl = Date.parse(row.deadline);
      if (!Number.isNaN(dl) && dl < Date.now() && !row.answered_at) return "expired";
    }
    return "pending";
  }
  if (raw === "expired") return "expired";
  return raw ? "unknown" : "pending";
}

function toDecision(row: ApprovalRowJSON): ApprovalDecision {
  return {
    id: row.id ?? "",
    question: row.question ?? "",
    options: (row.options ?? []).map((o) => ({ key: o.key ?? "", label: o.label || (o.key ?? "") })),
    recommended: row.recommended ?? "",
    recommendationReason: row.recommendation_reason ?? "",
    defaultAnswer: row.default_answer ?? "",
    reversible: Boolean(row.reversible),
    urgency: row.urgency ?? "",
    contextUrl: row.context_url ?? "",
    bigPicture: row.big_picture ?? "",
    technicalDetails: row.technical_details ?? "",
    deadline: row.deadline ?? "",
    state: deriveState(row),
    answer: row.answer ?? "",
    answerLabel: row.answer_label ?? "",
    note: row.note ?? "",
    answeredBy: row.answered_by ?? "",
    answeredAt: row.answered_at ?? "",
    late: Boolean(row.late),
    defaultAppliedAt: row.default_applied_at ?? "",
    resolvedAt: row.resolved_at ?? "",
    createdAt: row.created_at ?? "",
    updatedAt: row.updated_at ?? "",
    source: classifySource(row.source),
    sourceEventType: row.source_event_type ?? "",
    kind: kindFromEventType(row.source_event_type, row.source),
  };
}

/** Raised when notification-hub is unreachable (facade returns 503). */
export class ApprovalsUnavailableError extends Error {
  constructor(message = "Approvals service is unavailable") {
    super(message);
    this.name = "ApprovalsUnavailableError";
  }
}

/**
 * Raised when the facade reports the decision was already resolved (409) —
 * someone else (or the deadline default) answered it first. The caller shows a
 * friendly "already resolved" message AND still refetches so the stale row
 * drops out of the list, rather than letting the operator keep hitting 409.
 */
export class ApprovalAlreadyResolvedError extends Error {
  constructor(message = "This decision was already resolved") {
    super(message);
    this.name = "ApprovalAlreadyResolvedError";
  }
}

/**
 * Raised when the operator's session is no longer valid (401). The caller shows
 * a "session expired — sign in" state wired to the real sign-in affordance
 * instead of a dead retry.
 */
export class ApprovalsUnauthorizedError extends Error {
  constructor(message = "Your session expired") {
    super(message);
    this.name = "ApprovalsUnauthorizedError";
  }
}

async function authedFetch(path: string, init?: RequestInit): Promise<Response> {
  const headers = new Headers(init?.headers);
  headers.set("X-Vrooli-Device-Id", deviceIdentity().id);
  const token = await getAccessToken();
  if (token && !headers.has("Authorization")) {
    headers.set("Authorization", `Bearer ${token}`);
  }
  const res = await fetch(buildApiUrl(path, { baseUrl: API_BASE }), {
    ...init,
    headers,
    cache: "no-store",
  });
  if (res.status === 503) {
    throw new ApprovalsUnavailableError();
  }
  // A session that lapsed mid-session: the generic error branch would render a
  // dead Retry. Surface a typed error so the inbox offers sign-in instead.
  if (res.status === 401 || res.status === 403) {
    throw new ApprovalsUnauthorizedError();
  }
  // The decision was answered by someone else (or the deadline default) between
  // this operator loading the list and acting. Typed so the caller can show a
  // friendly message and refetch, rather than echo a raw 409.
  if (res.status === 409) {
    throw new ApprovalAlreadyResolvedError();
  }
  if (!res.ok) {
    throw makeApiError("approvals_error", `approvals request failed (${String(res.status)})`, res.status);
  }
  return res;
}

/**
 * List the operator's asks. `openOnly` restricts to still-pending decisions
 * (the inbox default); pass false for the resolved/history view.
 */
export async function listApprovals(openOnly = true, limit = 100): Promise<ApprovalDecision[]> {
  const params = new URLSearchParams({ open_only: String(openOnly), limit: String(limit) });
  const res = await authedFetch(`/api/v1/approvals?${params.toString()}`, { method: "GET" });
  const body = (await res.json()) as { approvals?: ApprovalRowJSON[] };
  return Array.isArray(body.approvals) ? body.approvals.map(toDecision) : [];
}

/**
 * Read one decision by id — the deep-link landing path, where the decision may
 * be resolved or outside the current list filter.
 */
export async function getApproval(askId: string): Promise<ApprovalDecision> {
  const res = await authedFetch(`/api/v1/approvals/${encodeURIComponent(askId)}`, { method: "GET" });
  const body = (await res.json()) as { approval?: ApprovalRowJSON };
  if (!body.approval) throw new Error("decision not found");
  return toDecision(body.approval);
}

/**
 * Answer a decision with the chosen option key and an optional operator note.
 * This is the ONE mutation the inbox performs — approve, reject, and every
 * other option all route through the facade's single answer route, which calls
 * notification-hub's Answer RPC with the operator's verified identity.
 */
export async function answerApproval(askId: string, answer: string, note = ""): Promise<void> {
  await authedFetch("/api/v1/approvals/answer", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ ask_id: askId, answer, note }),
  });
}

/** Count of still-pending decisions — the badge / banner source. */
export async function countPendingApprovals(): Promise<number> {
  const open = await listApprovals(true, 100);
  return open.filter((d) => d.state === "pending").length;
}
