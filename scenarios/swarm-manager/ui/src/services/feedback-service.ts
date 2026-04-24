/**
 * Feedback Service — data access for initiative feedback rounds.
 *
 * Thin layer over the HTTP surface at `/initiatives/{name}/feedback/*`.
 * Mirrors review-service / capture-service conventions so consumers can
 * substitute the client for tests via `createFeedbackService(mockClient)`.
 *
 * Responsibilities:
 *   - Translate a `(name, round)` + payload tuple into the correct endpoint
 *   - Handle multipart vs JSON dispatch for the start/continue endpoints
 *   - Surface the 409 lock-conflict shape as a typed error so the override
 *     dialog can render the current holder without a second round trip
 */

import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { ApiError } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
  FeedbackDecideResponse,
  FeedbackDecisionKind,
  FeedbackRound,
  FeedbackRoundType,
  ItemActivity,
  LockHolder,
  LockStatusResponse,
} from "../types";

// ---------------------------------------------------------------------------
// Request shapes
// ---------------------------------------------------------------------------

export interface StartFeedbackArgs {
  type: FeedbackRoundType;
  text: string;
  files?: File[];
  /** Optional short slug hint; otherwise derived from text. */
  slug?: string;
  /** Pre-empt an active lock on the initiative. */
  override?: boolean;
  /** Identifier of the user for audit — falls back to "user" on server. */
  decidedBy?: string;
}

export interface ContinueFeedbackArgs {
  text: string;
  files?: File[];
  decidedBy?: string;
}

export interface DecideFeedbackArgs {
  kind: FeedbackDecisionKind;
  acceptedMutationIds?: string[];
  rationale?: string;
  decidedBy?: string;
}

export interface DismissFeedbackArgs {
  rationale?: string;
  decidedBy?: string;
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

/**
 * Thrown when Start fails with 409 Conflict because another agent or round
 * holds the initiative lock (another feedback round, an initiative review).
 * The `holder` field is the current lock state — the UI renders this in
 * the override warning dialog.
 */
export class FeedbackLockConflictError extends Error {
  readonly holder?: LockHolder;

  constructor(message: string, holder?: LockHolder) {
    super(message);
    this.name = "FeedbackLockConflictError";
    this.holder = holder;
  }
}

/**
 * Thrown when Start fails with 409 Conflict because one or more member
 * backlog items currently have an in-flight agent run (workshop, execute,
 * etc). Distinct from `FeedbackLockConflictError` because the user sees a
 * per-item breakdown — "research/foo is workshopping" — rather than a
 * generic "initiative is locked" message. Override path cancels the listed
 * runs before starting the new feedback round.
 */
export class FeedbackBusyError extends Error {
  readonly activities: ItemActivity[];

  constructor(message: string, activities: ItemActivity[]) {
    super(message);
    this.name = "FeedbackBusyError";
    this.activities = activities;
  }
}

// ---------------------------------------------------------------------------
// Service contract
// ---------------------------------------------------------------------------

export interface IFeedbackService {
  list(name: string): Promise<FeedbackRound[]>;
  get(name: string, round: number): Promise<FeedbackRound>;
  start(name: string, args: StartFeedbackArgs): Promise<FeedbackRound>;
  continue_(name: string, round: number, args: ContinueFeedbackArgs): Promise<FeedbackRound>;
  decide(name: string, round: number, args: DecideFeedbackArgs): Promise<FeedbackDecideResponse>;
  dismiss(name: string, round: number, args?: DismissFeedbackArgs): Promise<FeedbackRound>;
  lockStatus(name: string): Promise<LockStatusResponse>;
  attachmentUrl(name: string, round: number, attachmentId: string): string;
}

// ---------------------------------------------------------------------------
// Factory
// ---------------------------------------------------------------------------

export function createFeedbackService(
  apiClient: IApiClient = defaultApiClient,
): IFeedbackService {
  return {
    async list(name) {
      const resp = await apiClient.get<{ rounds?: FeedbackRound[]; count?: number }>(
        API_ENDPOINTS.initiativeFeedback(name),
      );
      return resp.rounds ?? [];
    },

    async get(name, round) {
      return apiClient.get<FeedbackRound>(
        API_ENDPOINTS.initiativeFeedbackRound(name, round),
      );
    },

    async start(name, args) {
      // Multipart when there are files; JSON otherwise. The server picks
      // up either shape on the same endpoint.
      const hasFiles = (args.files?.length ?? 0) > 0;
      const body = hasFiles ? buildStartFormData(args) : buildStartJSON(args);
      try {
        return await apiClient.post<FeedbackRound>(
          API_ENDPOINTS.initiativeFeedback(name),
          body,
        );
      } catch (err) {
        throw mapLockConflict(err);
      }
    },

    async continue_(name, round, args) {
      const hasFiles = (args.files?.length ?? 0) > 0;
      const body = hasFiles ? buildContinueFormData(args) : {
        text: args.text,
        decided_by: args.decidedBy,
      };
      return apiClient.post<FeedbackRound>(
        API_ENDPOINTS.initiativeFeedbackContinue(name, round),
        body,
      );
    },

    async decide(name, round, args) {
      return apiClient.post<FeedbackDecideResponse>(
        API_ENDPOINTS.initiativeFeedbackDecide(name, round),
        {
          kind: args.kind,
          accepted_mutation_ids: args.acceptedMutationIds,
          rationale: args.rationale,
          decided_by: args.decidedBy,
        },
      );
    },

    async dismiss(name, round, args) {
      return apiClient.post<FeedbackRound>(
        API_ENDPOINTS.initiativeFeedbackDismiss(name, round),
        {
          rationale: args?.rationale,
          decided_by: args?.decidedBy,
        },
      );
    },

    async lockStatus(name) {
      return apiClient.get<LockStatusResponse>(
        API_ENDPOINTS.initiativeFeedbackLock(name),
      );
    },

    attachmentUrl(name, round, attachmentId) {
      return API_ENDPOINTS.initiativeFeedbackAttachment(name, round, attachmentId);
    },
  };
}

export const feedbackService = createFeedbackService();

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function buildStartFormData(args: StartFeedbackArgs): FormData {
  const fd = new FormData();
  fd.append("type", args.type);
  fd.append("text", args.text);
  if (args.slug) fd.append("slug", args.slug);
  if (args.override) fd.append("override", "true");
  if (args.decidedBy) fd.append("decided_by", args.decidedBy);
  for (const file of args.files ?? []) {
    fd.append("files", file, file.name);
  }
  return fd;
}

function buildStartJSON(args: StartFeedbackArgs): Record<string, unknown> {
  return {
    type: args.type,
    text: args.text,
    slug: args.slug,
    override: args.override ?? false,
    decided_by: args.decidedBy,
  };
}

function buildContinueFormData(args: ContinueFeedbackArgs): FormData {
  const fd = new FormData();
  fd.append("text", args.text);
  if (args.decidedBy) fd.append("decided_by", args.decidedBy);
  for (const file of args.files ?? []) {
    fd.append("files", file, file.name);
  }
  return fd;
}

/**
 * Map a 409 Conflict into the appropriate typed error so the UI can
 * render the right warning without a second round-trip.
 *
 * Server 409 payloads come in two shapes, keyed by body:
 *   - `{ error, holder }`      → another feedback round / review holds the
 *                                initiative lock. Surface as
 *                                FeedbackLockConflictError so the dialog
 *                                shows the holder + override path.
 *   - `{ error, activities }`  → one or more member backlog items have
 *                                in-flight agent runs. Surface as
 *                                FeedbackBusyError so the dialog lists
 *                                specific blockers.
 *
 * A 409 with neither shape (or a non-JSON body) falls back to a generic
 * FeedbackLockConflictError with no holder — the dialog still renders the
 * override path, just without extra context. Non-409s pass through.
 */
function mapLockConflict(err: unknown): unknown {
  if (!(err instanceof ApiError) || err.type !== "http" || err.status !== 409) {
    return err;
  }
  let holder: LockHolder | undefined;
  let activities: ItemActivity[] | undefined;
  try {
    const parsed = JSON.parse(err.message) as {
      holder?: LockHolder;
      activities?: ItemActivity[];
      error?: string;
    };
    holder = parsed.holder;
    activities = parsed.activities;
  } catch {
    /* tolerate non-JSON bodies — treat as generic lock conflict below. */
  }
  if (activities && activities.length > 0) {
    return new FeedbackBusyError(
      "initiative has active item-level agent runs",
      activities,
    );
  }
  return new FeedbackLockConflictError("initiative is locked", holder);
}
