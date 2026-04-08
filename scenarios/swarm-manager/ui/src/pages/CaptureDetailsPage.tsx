/**
 * CaptureDetailsPage — Full detail overlay for a capture.
 *
 * Shows the raw capture text, attachments at full size with lightbox,
 * classification triage, and metadata. Opened from sidebar CaptureCard
 * click or via deep-link (?detail=capture&id=...).
 */

import { useState, useCallback, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { Loader2, RefreshCw, Trash2, MessageSquare } from "lucide-react";
import { DetailPageLayout } from "../components/detail/DetailPageLayout";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { CaptureTriage } from "../components/capture/capture-triage";
import { ConfirmDialog } from "../components/ui/confirm-dialog";
import { Button } from "../components/ui/button";
import { ErrorState } from "../components/ui/error-state";
import { PageLoadingState } from "../components/ui/loading-states";
import { captureService } from "../services/capture-service";
import { NoteEditor } from "../components/ui/note-editor";
import { useCaptureStore } from "../stores/capture-store";
import { useDetailSelectionStore } from "../stores/detail-selection-store";
import { useDetailNavigation } from "../hooks/useDetailNavigation";
import { useRuntimeConfig } from "../hooks/useRuntimeConfig";
import { formatRelativeTime } from "../lib";
import type { Capture } from "../types";
import type { BacklogFormValues } from "../types";
import { BacklogFormDialog } from "../components/backlog/backlog-form-dialog";
import { backlogService } from "../services/backlog-service";
import { useBacklogStore } from "../stores";

export function CaptureDetailsPage() {
  const selection = useDetailSelectionStore((s) => s.selection);
  const { closeDetail } = useDetailNavigation();
  const captureId = selection?.identifier;

  // Try to get from store first (instant for sidebar click-through)
  const storeCapture = useCaptureStore((s) =>
    s.captures.find((c) => c.id === captureId),
  );

  // Fallback: fetch from API for deep-links
  const { data: fetchedCapture, isLoading, error } = useQuery({
    queryKey: ["capture", captureId],
    queryFn: () => captureService.get(captureId ?? ""),
    enabled: !!captureId && !storeCapture,
  });

  const capture: Capture | undefined = storeCapture ?? fetchedCapture;

  const [isRetrying, setIsRetrying] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const removeCapture = useCaptureStore((s) => s.removeCapture);
  const updateCapture = useCaptureStore((s) => s.updateCapture);
  const upsertBacklogItem = useBacklogStore((s) => s.upsertItem);

  // Delete confirmation state
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);
  const { getDeleteConfirmLevel } = useRuntimeConfig();

  // Edit dialog state
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [editPrefill, setEditPrefill] = useState<BacklogFormValues | undefined>();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  // Lightbox state
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  const handleRetry = useCallback(async () => {
    if (!captureId) return;
    setIsRetrying(true);
    try {
      await captureService.classify(captureId);
      updateCapture(captureId, { status: "classifying", classification: null });
    } catch {
      // Keep failed state.
    } finally {
      setIsRetrying(false);
    }
  }, [captureId, updateCapture]);

  const performDelete = useCallback(async () => {
    if (!captureId) return;
    setIsDeleting(true);
    setShowDeleteDialog(false);
    try {
      await captureService.remove(captureId);
      removeCapture(captureId);
      closeDetail();
    } catch {
      setIsDeleting(false);
    }
  }, [captureId, removeCapture, closeDetail]);

  const handleDeleteClick = useCallback(() => {
    if (getDeleteConfirmLevel("capture") === "none") {
      performDelete();
    } else {
      setShowDeleteDialog(true);
    }
  }, [getDeleteConfirmLevel, performDelete]);

  const handleEditItem = useCallback((prefill: BacklogFormValues) => {
    setEditPrefill(prefill);
    setSubmitError(null);
    setShowEditDialog(true);
  }, []);

  const handleEditSubmit = useCallback(async (values: BacklogFormValues) => {
    setIsSubmitting(true);
    setSubmitError(null);
    try {
      const created = await backlogService.create({ ...values, suggestedSkills: [] });
      upsertBacklogItem(created);
      setShowEditDialog(false);
      setEditPrefill(undefined);
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : "Failed to create backlog item");
    } finally {
      setIsSubmitting(false);
    }
  }, [upsertBacklogItem]);

  const handleCaptureResolved = useCallback(() => {
    closeDetail();
  }, [closeDetail]);

  // Close lightbox on Escape
  useEffect(() => {
    if (!lightboxSrc) return;
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setLightboxSrc(null);
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [lightboxSrc]);

  if (!captureId) {
    return (
      <DetailPageLayout header={<DetailPageHeader entityType="Capture" title="Not found" nodeId={null} lenses={[]} />}>
        <ErrorState message="No capture selected." onRetry={closeDetail} />
      </DetailPageLayout>
    );
  }

  if (isLoading && !capture) {
    return <PageLoadingState label="Loading capture..." />;
  }

  if (error && !capture) {
    return (
      <DetailPageLayout header={<DetailPageHeader entityType="Capture" title="Error" nodeId={null} lenses={[]} />}>
        <ErrorState message="Capture not found." onRetry={closeDetail} />
      </DetailPageLayout>
    );
  }

  if (!capture) {
    return (
      <DetailPageLayout header={<DetailPageHeader entityType="Capture" title="Not found" nodeId={null} lenses={[]} />}>
        <ErrorState message="Capture not found." onRetry={closeDetail} />
      </DetailPageLayout>
    );
  }

  const items = capture.classification?.items ?? [];
  const statusLabel =
    capture.status === "classifying" ? "Classifying" :
    capture.status === "failed" ? "Failed" :
    capture.status === "classified" && items.length > 0 ? "Classified" :
    "No action";

  const headerActions = (
    <div className="flex items-center gap-2">
      {capture.status === "failed" && (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleRetry}
          disabled={isRetrying}
        >
          <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${isRetrying ? "animate-spin" : ""}`} />
          Retry
        </Button>
      )}
      <Button
        variant="ghost"
        size="sm"
        onClick={handleDeleteClick}
        disabled={isDeleting}
        className="text-red-400 hover:text-red-300"
      >
        {isDeleting ? (
          <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
        ) : (
          <Trash2 className="mr-1.5 h-3.5 w-3.5" />
        )}
        Delete
      </Button>
    </div>
  );

  return (
    <DetailPageLayout
      header={
        <DetailPageHeader
          entityType="Capture"
          entityIcon={MessageSquare}
          title={capture.text.length > 80 ? capture.text.slice(0, 80) + "..." : capture.text}
          subtitle={formatRelativeTime(capture.created)}
          status={statusLabel}
          nodeId={null}
          lenses={[]}
          actions={headerActions}
        />
      }
    >
      <div className="mx-auto max-w-3xl space-y-6">
        {/* Full capture text */}
        <section>
          <h2 className="mb-2 text-sm font-medium uppercase tracking-wider text-slate-500">Capture Text</h2>
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
            <p className="whitespace-pre-wrap text-slate-200">{capture.text}</p>
          </div>
        </section>

        {/* Personal note */}
        <section>
          <NoteEditor
            note={capture.note ?? ""}
            onSave={async (note) => {
              await captureService.updateNote(capture.id, note);
              updateCapture(capture.id, { note });
            }}
          />
        </section>

        {/* Attachments */}
        {capture.attachments.length > 0 && (
          <section>
            <h2 className="mb-2 text-sm font-medium uppercase tracking-wider text-slate-500">
              Attachments ({capture.attachments.length})
            </h2>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {capture.attachments.map((url, i) => (
                <button
                  key={i}
                  type="button"
                  onClick={() => setLightboxSrc(url)}
                  className="group overflow-hidden rounded-lg border border-slate-800 bg-slate-900/50 transition-colors hover:border-slate-600"
                >
                  <img
                    src={url}
                    alt={`Attachment ${i + 1}`}
                    className="w-full object-contain transition-transform group-hover:scale-[1.02]"
                  />
                </button>
              ))}
            </div>
          </section>
        )}

        {/* Classification status */}
        {capture.status === "classifying" && (
          <section>
            <div className="flex items-center gap-2 rounded-lg border border-cyan-500/20 bg-cyan-500/5 p-4">
              <Loader2 className="h-4 w-4 animate-spin text-cyan-400" />
              <span className="text-sm text-cyan-400">Classification in progress...</span>
            </div>
          </section>
        )}

        {capture.status === "failed" && (
          <section>
            <div className="flex items-center justify-between rounded-lg border border-red-500/20 bg-red-500/5 p-4">
              <span className="text-sm text-red-400">Classification failed</span>
              <Button variant="ghost" size="sm" onClick={handleRetry} disabled={isRetrying}>
                <RefreshCw className={`mr-1.5 h-3.5 w-3.5 ${isRetrying ? "animate-spin" : ""}`} />
                Retry
              </Button>
            </div>
          </section>
        )}

        {capture.status === "classified" && items.length === 0 && (
          <section>
            <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
              <span className="text-sm italic text-slate-500">Nothing actionable detected</span>
            </div>
          </section>
        )}

        {/* Triage suggestions */}
        {capture.status === "classified" && items.length > 0 && (
          <section>
            <h2 className="mb-2 text-sm font-medium uppercase tracking-wider text-slate-500">
              Suggested Items
            </h2>
            <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
              <CaptureTriage
                capture={capture}
                onEditItem={handleEditItem}
                onCaptureResolved={handleCaptureResolved}
              />
            </div>
          </section>
        )}

        {/* Metadata */}
        <section>
          <h2 className="mb-2 text-sm font-medium uppercase tracking-wider text-slate-500">Metadata</h2>
          <div className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
            <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
              <dt className="text-slate-500">Capture ID</dt>
              <dd className="font-mono text-xs text-slate-300">{capture.id}</dd>
              <dt className="text-slate-500">Created</dt>
              <dd className="text-slate-300">{new Date(capture.created).toLocaleString()}</dd>
              <dt className="text-slate-500">Status</dt>
              <dd className="text-slate-300">{capture.status}</dd>
              {capture.classification?.classifiedAt && (
                <>
                  <dt className="text-slate-500">Classified at</dt>
                  <dd className="text-slate-300">{new Date(capture.classification.classifiedAt).toLocaleString()}</dd>
                </>
              )}
            </dl>
          </div>
        </section>
      </div>

      {/* Lightbox overlay */}
      {lightboxSrc && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
          onClick={() => setLightboxSrc(null)}
        >
          <button
            type="button"
            className="absolute right-4 top-4 rounded-full bg-slate-800/80 p-2 text-slate-300 transition-colors hover:bg-slate-700 hover:text-white"
            onClick={() => setLightboxSrc(null)}
            aria-label="Close lightbox"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
          <img
            src={lightboxSrc}
            alt="Full resolution attachment"
            className="max-h-[90vh] max-w-[90vw] rounded-lg object-contain"
            onClick={(e) => e.stopPropagation()}
          />
        </div>
      )}

      {/* Delete confirmation dialog */}
      {(() => {
        const deleteLevel = getDeleteConfirmLevel("capture");
        return deleteLevel !== "none" ? (
          <ConfirmDialog
            isOpen={showDeleteDialog}
            onClose={() => setShowDeleteDialog(false)}
            onConfirm={performDelete}
            title="Delete Capture"
            description="Are you sure you want to delete this capture? This action cannot be undone."
            confirmationText={deleteLevel === "strong" ? capture.id : undefined}
            confirmLabel="Delete"
            isLoading={isDeleting}
          />
        ) : null;
      })()}

      {/* Edit before adding dialog */}
      <BacklogFormDialog
        isOpen={showEditDialog}
        mode="create"
        initialValues={editPrefill}
        isSubmitting={isSubmitting}
        submitError={submitError}
        onClose={() => {
          setShowEditDialog(false);
          setEditPrefill(undefined);
        }}
        onSubmit={handleEditSubmit}
      />
    </DetailPageLayout>
  );
}
