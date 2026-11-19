import { apiUrl, decodeApiError } from "./client";

export interface ReleaseText { title: string; subtitle?: string; description?: string }
export interface ReleaseTrack { order: number; track_sha256: string; title_override?: string; languages?: string[] }
export interface ReleaseRecording { track_sha256: string; take_id: string; receipt_ref: string; operator_track_approval: string }
export interface ReleaseTrackRights { track_sha256: string; source_license?: string; usage_scope: "unknown" | "personal_only" | "commercial" | "blocked"; decision_status: "unknown" | "review_required" | "operator_confirmed" | "blocked"; decision_id?: string; evidence_ref?: string; note?: string }
export interface ReleaseRightsDecision { decision_id: string; release_id: string; revision: number; track_sha256: string; source_license: string; usage_scope: "personal_only" | "commercial" | "blocked"; decision: "confirmed" | "denied"; evidence_ref: string; recording_provenance_sha256: string; reviewer_source: string; note: string; created_at: string }
export interface ReleaseEdition {
  release_id: string;
  revision: number;
  kind: "single" | "ep" | "album";
  status: "draft";
  title: string;
  default_locale: string;
  artist_owner?: string;
  artist_subject_id?: string;
  artist_subject_version?: string;
  localized_metadata: Record<string, ReleaseText>;
  tracks: ReleaseTrack[];
  recording_provenance?: ReleaseRecording[];
  credits: Array<Record<string, unknown>>;
  assets: Array<Record<string, unknown>>;
  track_rights: ReleaseTrackRights[];
  rights_decisions?: ReleaseRightsDecision[];
  derivations: Array<Record<string, unknown>>;
  created_at: string;
}
export interface EligibilityCheck {
  property: string;
  status: "supported" | "missing" | "blocked";
  evidence?: string[];
  missing?: string[];
}
export interface ReleaseAssessment {
  edition: ReleaseEdition;
  reviews: Array<{ review_id: string; revision: number; decision: string; edition_sha256: string; heard_all_tracks: boolean; reviewed_track_sha256s: string[]; note: string; created_at: string }>;
  eligibility: { ready_for_operator_judgment: boolean; showcase_ready: boolean; checks: EligibilityCheck[] };
}
export interface ReleasePage {
  items: Array<{ release_id: string; revision: number; kind: string; status: string; title: string; created_at: string; track_count: number }>;
  next_cursor?: string;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(apiUrl(path), { cache: "no-store", ...init });
  if (!response.ok) throw await decodeApiError(response);
  return response.json() as Promise<T>;
}

export function saveReleaseDraft(edition: Omit<ReleaseEdition, "created_at">): Promise<{ edition: ReleaseEdition }> {
  return request("/releases", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(edition) });
}
export function listReleases(cursor = ""): Promise<ReleasePage> {
  const query = new URLSearchParams({ limit: "20" });
  if (cursor) query.set("cursor", cursor);
  return request("/releases?" + query.toString());
}
export function getRelease(releaseID: string, revision?: number): Promise<ReleaseAssessment> {
  const query = revision ? "?revision=" + revision : "";
  return request("/releases/" + encodeURIComponent(releaseID) + query);
}
export function recordReleaseReview(releaseID: string, review: { review_id: string; revision: number; decision: "approved" | "changes_requested"; heard_all_tracks: boolean; reviewed_track_sha256s: string[]; note: string }) {
  return request("/releases/" + encodeURIComponent(releaseID) + "/reviews", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(review) });
}
export function recordReleaseRightsDecision(releaseID: string, decision: { decision_id: string; revision: number; track_sha256: string; source_license: string; usage_scope: "personal_only" | "commercial" | "blocked"; decision: "confirmed" | "denied"; evidence_ref: string; note: string }) {
  return request("/releases/" + encodeURIComponent(releaseID) + "/rights-decisions", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(decision) });
}
