import { buildApiUrl } from "@vrooli/api-base";
import { API_BASE, decodeApiError } from "./client";

const root = "/api/v1/channel-manager";
async function request<T>(path: string, body?: unknown): Promise<T> {
  const response = await fetch(buildApiUrl(`${root}${path}`, { baseUrl: API_BASE }), {
    method: body === undefined ? "GET" : "POST",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  if (!response.ok) throw await decodeApiError(response);
  return response.json() as Promise<T>;
}
export type Identity = { id: string; platform_id: string; handle?: string; display_label?: string; purpose: string; persona_ref?: string; goals?: string[]; notes?: string; owner_ref?: string; lifecycle?: string; environment_ref: string; vault_ref?: string; status: string; d009_acceptance_ref?: string; automation_mode?: string; lane_grants?: string[] };
export type Action = { id: string; identity_id: string; kind: string; window: string; status: string; rolled_count: number; deferred?: boolean; execution_id?: string; execution_error?: string; failure_class?: string; attempt_count?: number; next_attempt_at?: string };
export type Program = { id: string; platform_id: string; provenance?: { source_kind: string; confidence: string; revisit_trigger: string } };
export type Flag = { identity_id?: string; metric?: string; message?: string };
export type Release = { id: string; draft_id: string; action_id: string; status: string; platform_post_id: string; published_url: string; first_comment_status: string; delivery_status: string; delivery_error: string };
export type MetricSample = { id: string; release_id: string; draft_id: string; metric: string; value: number; delivery_status: string };
export type PortfolioPolicy = { minimum_post_gap_minutes: number; window_minutes: number; max_posts_per_window: number };
export type EnvironmentCheck = { identity_id: string; expected_region: string; observed_region: string; status: string; reason: string; checked_at: string };
export type ActivityEvent = { id: string; event_type: string; occurred_at: string; identity_id: string; action_id?: string; release_id?: string; execution_id?: string; actor_type: string; executor_type?: string; artifact_refs?: string[]; details?: Record<string, string> };
export type Platform = { id: string; caption_limit?: number; title_limit?: number; disclosure_required?: boolean; post_types?: { id: string; format_kind: string }[] };
export type Overview = { identities: Record<string, Identity>; actions: Record<string, Action>; platforms?: Record<string, Platform>; programs?: Record<string, Program>; program_support?: Record<string, number>; flags?: Record<string, Flag[]>; releases?: Record<string, Release>; metric_samples?: Record<string, MetricSample>; portfolio?: PortfolioPolicy; environment_checks?: Record<string, EnvironmentCheck>; activity_events?: ActivityEvent[] };
export type ReleasePreview = { caption: string; title?: string; caption_truncated: boolean; title_truncated?: boolean; media_presentation?: string; disclosure_required: boolean; release_allowed: boolean; blocking_errors: string[]; first_comment: string };
export const overview = () => request<Overview>("/overview");
export const createIdentity = (identity: Identity & { attestations: Record<string, boolean> }) => request<Identity>("/identities", identity);
export const updateIdentity = (id: string, identity: Identity) => fetch(buildApiUrl(`${root}/identities/${id}`, { baseUrl: API_BASE }), { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify(identity) }).then(async response => { if (!response.ok) throw await decodeApiError(response); return response.json() as Promise<Identity>; });
export const retireIdentity = (id: string) => request<{ status: string }>(`/identities/${id}/retire`, {});
export const startProgram = (id: string, program_id: string) => request<{ status: string }>(`/identities/${id}/start`, { program_id });
export const enqueueAction = (identity_id: string, kind: string) => request<Action>("/actions", { identity_id, kind, seed: Date.now() });
export const completeAction = (id: string, evidence: string) => request<{ status: string }>(`/actions/${id}/complete`, { evidence });
export const recordObservation = (id: string, value: number) => request<{ flag: unknown }>(`/identities/${id}/observations`, { metric: "reach", value });
export const previewRelease = (input: { platform_id: string; caption: string; title?: string; post_type?: string; format_kind: string; media_width: number; media_height: number; disclosure_visible: boolean; disclosure_in_visible_region?: boolean; first_comment: string }) => request<ReleasePreview>("/releases/preview", input);
export const assignAutomation = (identityID: string, input: { consumer_profile_key: string; session_profile_ref: string; workflow_ref: string; enabled_action_kinds: string[]; operator_note: string }) => request<{ identity_id: string }>(`/identities/${identityID}/automation`, input);
export const dispatchBrowserAction = (actionID: string) => request<{ execution_id: string }>(`/actions/${actionID}/dispatch-browser`, {});
export const configurePortfolio = (policy: PortfolioPolicy) => request<PortfolioPolicy>("/portfolio", policy);
