import { useState, useEffect } from "react";

interface ValidationRecord {
  id: string;
  profile_id: string;
  smoke_test_id: string;
  status: string;
  video_url: string;
  video_size_bytes: number;
  video_duration_ms: number;
  platform: string;
  git_commit_hash: string;
  review_decision: string;
  reviewed_by: string;
  review_notes: string;
  created_at: string;
  completed_at: string | null;
  reviewed_at: string | null;
}

interface ReviewResponse {
  status: string;
  decision: string;
  approval_id?: string;
  approval_status?: string;
}

interface VideoReviewPanelProps {
  validationId: string;
  apiBase?: string;
}

interface ApprovalStatusRecord {
  id: string;
  status: string;
  platform: string;
  validation_id?: string;
}

function isValidationRecord(value: unknown): value is ValidationRecord {
  return Boolean(value && typeof value === "object" && "id" in value);
}

function isApprovalStatusArray(value: unknown): value is ApprovalStatusRecord[] {
  return Array.isArray(value);
}

async function fetchJson(url: string): Promise<unknown> {
  const response = await fetch(url);
  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`);
  }
  return response.json();
}

export function VideoReviewPanel({ validationId, apiBase = "" }: VideoReviewPanelProps) {
  const [validation, setValidation] = useState<ValidationRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [reviewNotes, setReviewNotes] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [approvalStatus, setApprovalStatus] = useState<{ id: string; status: string } | null>(null);

  useEffect(() => {
    fetchJson(`${apiBase}/api/v1/validations/${validationId}`)
      .then((payload) => {
        if (!isValidationRecord(payload)) {
          throw new Error("Invalid validation payload");
        }
        setValidation(payload);
      })
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Failed to load validation"))
      .finally(() => setLoading(false));
  }, [validationId, apiBase]);

  // Fetch approval status on mount if already reviewed and has commit hash.
  useEffect(() => {
    if (!validation?.review_decision || !validation?.git_commit_hash || !validation?.profile_id) return;
    fetch(`${apiBase}/api/v1/profiles/${validation.profile_id}/approvals?commit=${validation.git_commit_hash}`)
      .then((res) => {
        if (!res.ok) return [];
        return res.json();
      })
      .then((approvals: unknown) => {
        if (!isApprovalStatusArray(approvals)) {
          return;
        }
        const match = approvals.find((a) => a.validation_id === validationId);
        if (match) {
          setApprovalStatus({ id: match.id, status: match.status });
        }
      })
      .catch(() => {});
  }, [validation?.review_decision, validation?.git_commit_hash, validation?.profile_id, validationId, apiBase]);

  const submitReview = async (decision: "approved" | "rejected") => {
    setSubmitting(true);
    try {
      const res = await fetch(`${apiBase}/api/v1/validations/${validationId}/review`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ decision, notes: reviewNotes }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const reviewResp = (await res.json()) as ReviewResponse;

      // Capture approval status from enriched response.
      if (reviewResp.approval_id) {
        setApprovalStatus({ id: reviewResp.approval_id, status: reviewResp.approval_status || decision });
      }

      // Refresh validation data.
      const updated = await fetchJson(`${apiBase}/api/v1/validations/${validationId}`);
      if (!isValidationRecord(updated)) {
        throw new Error("Invalid validation payload");
      }
      setValidation(updated);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Review submission failed");
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) return <div className="p-4">Loading validation...</div>;
  if (error) return <div className="p-4 text-red-600">Error: {error}</div>;
  if (!validation) return <div className="p-4">Validation not found</div>;

  const isReviewed = !!validation.review_decision;
  const hasVideo = !!validation.video_url;

  return (
    <div className="flex flex-col gap-4 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold">Visual Validation</h2>
        <div className="flex items-center gap-2">
          {approvalStatus && (
            <span
              className={`rounded px-2 py-1 text-xs font-medium ${
                approvalStatus.status === "approved"
                  ? "bg-blue-100 text-blue-800"
                  : approvalStatus.status === "rejected"
                    ? "bg-red-100 text-red-800"
                    : "bg-gray-100 text-gray-800"
              }`}
            >
              Approval: {approvalStatus.status}
            </span>
          )}
          <span
            className={`rounded px-2 py-1 text-sm font-medium ${
              validation.status === "passed" || validation.review_decision === "approved"
                ? "bg-green-100 text-green-800"
                : validation.status === "failed" || validation.review_decision === "rejected"
                  ? "bg-red-100 text-red-800"
                  : "bg-yellow-100 text-yellow-800"
            }`}
          >
            {validation.review_decision || validation.status}
          </span>
        </div>
      </div>

      {/* Metadata */}
      <div className="grid grid-cols-2 gap-2 text-sm">
        <div>
          <span className="font-medium">Platform:</span> {validation.platform || "unknown"}
        </div>
        <div>
          <span className="font-medium">Created:</span> {new Date(validation.created_at).toLocaleString()}
        </div>
        {validation.git_commit_hash && (
          <div>
            <span className="font-medium">Commit:</span> {validation.git_commit_hash.substring(0, 12)}
          </div>
        )}
        {validation.video_duration_ms > 0 && (
          <div>
            <span className="font-medium">Duration:</span> {(validation.video_duration_ms / 1000).toFixed(1)}s
          </div>
        )}
        {validation.video_size_bytes > 0 && (
          <div>
            <span className="font-medium">Size:</span> {(validation.video_size_bytes / 1024 / 1024).toFixed(1)} MB
          </div>
        )}
      </div>

      {/* Video player */}
      {hasVideo && (
        <video
          controls
          className="w-full rounded border"
          src={`${apiBase}/api/v1/validations/${validationId}/video`}
        >
          Your browser does not support the video tag.
        </video>
      )}

      {!hasVideo && <div className="rounded border border-dashed p-8 text-center text-gray-500">No video recording available</div>}

      {/* Review form */}
      {!isReviewed && (
        <div className="flex flex-col gap-2">
          <textarea
            className="rounded border p-2"
            rows={3}
            placeholder="Review notes (optional)"
            value={reviewNotes}
            onChange={(e) => setReviewNotes(e.target.value)}
          />
          <div className="flex gap-2">
            <button
              className="rounded bg-green-600 px-4 py-2 text-white hover:bg-green-700 disabled:opacity-50"
              disabled={submitting}
              onClick={() => submitReview("approved")}
            >
              Approve
            </button>
            <button
              className="rounded bg-red-600 px-4 py-2 text-white hover:bg-red-700 disabled:opacity-50"
              disabled={submitting}
              onClick={() => submitReview("rejected")}
            >
              Reject
            </button>
          </div>
        </div>
      )}

      {/* Review history */}
      {isReviewed && (
        <div className="rounded border bg-gray-50 p-3">
          <h3 className="font-medium">Review</h3>
          <div className="mt-1 text-sm">
            <div>
              <span className="font-medium">Decision:</span> {validation.review_decision}
            </div>
            {validation.reviewed_by && (
              <div>
                <span className="font-medium">By:</span> {validation.reviewed_by}
              </div>
            )}
            {validation.review_notes && (
              <div>
                <span className="font-medium">Notes:</span> {validation.review_notes}
              </div>
            )}
            {validation.reviewed_at && (
              <div>
                <span className="font-medium">At:</span> {new Date(validation.reviewed_at).toLocaleString()}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
