import { useEffect, useMemo, useRef, useState } from "react";
import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Code, ConnectError } from "@connectrpc/connect";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { PresentationAppearance, PresentationKind, RenderJobStatus, RenderMode, RenderOutputProfile, type RenderJob, type RenderOutput, type ShotRevision, type VideoSampleBinding } from "@vrooli/proto-types/device-studio/v1/shots/shots_pb";
import { rendersClient, getRenderOutputContent, getVideoSampleBinding, isRenderActive, isRenderCancellable, renderOutputContentURL, verifyRenderOutputContent } from "../../api/renders";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

type Props = {
  projectId: string;
  shotId: string;
  revision?: ShotRevision;
  historical: boolean;
  hasUnsavedChanges: boolean;
  showOutput: boolean;
  onShowOutputChange: (show: boolean) => void;
  onFinalOutputChange?: (output?: FinalRenderOutput) => void;
  onAcceptVideoSample?: (binding: VideoSampleBinding) => Promise<void> | void;
};

export type FinalRenderOutput = { url: string; renderId: string; revision: bigint; metadata: RenderOutput; isVideo: boolean; isSample: boolean };

function retryKey(projectId: string, shotId: string, revision: bigint, mode: RenderMode): string {
  const suffix = mode === RenderMode.STILL ? "" : `:${mode}`;
  const slot = `device-studio:render:${projectId}:${shotId}:${revision}${suffix}`;
  try {
    const existing = globalThis.sessionStorage.getItem(slot);
    if (existing) return existing;
    const runtimeCrypto = Reflect.get(globalThis, "crypto") as { randomUUID?: () => string } | undefined;
    const random = runtimeCrypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`;
    globalThis.sessionStorage.setItem(slot, random);
    return random;
  } catch {
    return `${projectId}:${shotId}:${revision}`;
  }
}

function clearRetryKey(projectId: string, shotId: string, revision: bigint, mode: RenderMode) {
  const suffix = mode === RenderMode.STILL ? "" : `:${mode}`;
  try { globalThis.sessionStorage.removeItem(`device-studio:render:${projectId}:${shotId}:${revision}${suffix}`); } catch { /* Storage can be disabled. */ }
}

type StatusKey = typeof strings.shot.renderQueued | typeof strings.shot.renderObserved | typeof strings.shot.renderReconcile | typeof strings.shot.renderCancelRequested | typeof strings.shot.renderSucceeded | typeof strings.shot.renderFailed | typeof strings.shot.renderCancelled | typeof strings.shot.renderUnknown;

function statusLabel(status: RenderJobStatus, t: (key: StatusKey) => string): string {
  const labels: Partial<Record<RenderJobStatus, string>> = {
    [RenderJobStatus.QUEUED]: t(strings.shot.renderQueued),
    [RenderJobStatus.OBSERVED]: t(strings.shot.renderObserved),
    [RenderJobStatus.NEEDS_RECONCILIATION]: t(strings.shot.renderReconcile),
    [RenderJobStatus.CANCEL_REQUESTED]: t(strings.shot.renderCancelRequested),
    [RenderJobStatus.SUCCEEDED]: t(strings.shot.renderSucceeded),
    [RenderJobStatus.FAILED]: t(strings.shot.renderFailed),
    [RenderJobStatus.CANCELLED]: t(strings.shot.renderCancelled),
  };
  return labels[status] ?? t(strings.shot.renderUnknown);
}

export function RenderControls({ projectId, shotId, revision, historical, hasUnsavedChanges, showOutput, onShowOutputChange, onFinalOutputChange, onAcceptVideoSample }: Props) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [selectedJobId, setSelectedJobId] = useState("");
  const [submittedJob, setSubmittedJob] = useState<RenderJob>();
  const [pendingPhysicalPreviewId, setPendingPhysicalPreviewId] = useState("");
  const [outputUrlState, setOutputUrlState] = useState<{ key: string; url: string }>();
  const [viewedSampleIds, setViewedSampleIds] = useState<string[]>([]);
  const [acceptError, setAcceptError] = useState("");
  const [outputLinkError, setOutputLinkError] = useState("");

  const jobs = useInfiniteQuery({
    queryKey: ["renders", projectId],
    initialPageParam: "",
    queryFn: ({ pageParam }) => rendersClient.listRenders({ projectId, pageSize: 50, pageToken: pageParam }),
    getNextPageParam: (page) => page.nextPageToken || undefined,
    refetchInterval: (query) => {
      const currentRevision = revision?.revision;
      const canFollowPreview = Boolean(currentRevision && !historical && !hasUnsavedChanges);
      const listedPreview = query.state.data?.pages.flatMap((page) => page.jobs).find((job) => job.id === pendingPhysicalPreviewId);
      const pendingPreview = listedPreview ?? (submittedJob?.id === pendingPhysicalPreviewId ? submittedJob : undefined);
      const previewActive = canFollowPreview && pendingPreview?.shotId === shotId && pendingPreview.shotRevision === currentRevision && pendingPreview.renderMode === RenderMode.PHYSICAL_PREVIEW_PNG && isRenderActive(pendingPreview.status);
      if (previewActive) return 250;
      return query.state.data?.pages.some((page) => page.jobs.some((job) => isRenderActive(job.status))) ? 2000 : false;
    },
    refetchIntervalInBackground: false,
    refetchOnWindowFocus: true,
  });

  const submit = useMutation({
    mutationFn: async (input: { projectId: string; shotId: string; revision: bigint; idempotencyKey: string; mode: RenderMode; profile: RenderOutputProfile }) => {
      return rendersClient.submitRender({
        projectId: input.projectId, shotId: input.shotId, revision: input.revision,
        idempotencyKey: input.idempotencyKey,
        outputProfile: input.profile, renderMode: input.mode, sampleOutputIndex: 0,
      });
    },
    onSuccess: ({ job }, input) => {
      if (!job) return;
      clearRetryKey(input.projectId, input.shotId, input.revision, input.mode);
      if (input.projectId === projectId && input.shotId === shotId && input.revision === revision?.revision) {
        setSubmittedJob(job);
        setSelectedJobId(job.id);
        setPendingPhysicalPreviewId(input.mode === RenderMode.PHYSICAL_PREVIEW_PNG ? job.id : "");
        onShowOutputChange(false);
        setOutputUrlState(undefined);
      }
      void queryClient.invalidateQueries({ queryKey: ["renders", input.projectId] });
    },
    onError: () => { /* Keep the same session idempotency key for an uncertain retry. */ },
  });

  const cancel = useMutation({
    mutationFn: (job: RenderJob) => rendersClient.cancelRender({ projectId, renderId: job.id, expectedVersion: job.version }),
    onSuccess: ({ job }) => {
      if (job) setSubmittedJob(job);
      void queryClient.invalidateQueries({ queryKey: ["renders", projectId] });
    },
    onError: () => { void queryClient.invalidateQueries({ queryKey: ["renders", projectId] }); },
  });
  const acceptSample = useMutation({
    mutationFn: (input: { projectId: string; shotId: string; revision: bigint; renderId: string }) => getVideoSampleBinding(input.projectId, input.shotId, input.renderId),
    onSuccess: async (binding, input) => {
      if (input.projectId === projectId && input.shotId === shotId && input.revision === revision?.revision) await onAcceptVideoSample?.(binding);
      setAcceptError("");
    },
    onError: (error) => setAcceptError(errorMessage(error, t)),
  });

  const allJobs = useMemo(() => {
    const rows = jobs.data?.pages.flatMap((page) => page.jobs) ?? [];
    if (submittedJob && !rows.some((job) => job.id === submittedJob.id)) rows.unshift(submittedJob);
    return rows.filter((job) => job.shotId === shotId);
  }, [jobs.data, shotId, submittedJob]);
  const selectedJob = allJobs.find((job) => job.id === selectedJobId) ?? allJobs[0];
  useEffect(() => {
    if (!pendingPhysicalPreviewId) return;
    if (!revision || historical || hasUnsavedChanges) {
      setPendingPhysicalPreviewId("");
      return;
    }
    const pending = allJobs.find((job) => job.id === pendingPhysicalPreviewId);
    if (!pending || pending.shotRevision !== revision.revision || pending.renderMode !== RenderMode.PHYSICAL_PREVIEW_PNG) {
      if (pending) setPendingPhysicalPreviewId("");
      return;
    }
    if (pending.status === RenderJobStatus.FAILED || pending.status === RenderJobStatus.CANCELLED || pending.status === RenderJobStatus.SUCCEEDED) {
      setPendingPhysicalPreviewId("");
      if (pending.status === RenderJobStatus.SUCCEEDED && pending.output) {
        setSelectedJobId(pending.id);
        onShowOutputChange(true);
      }
    }
  }, [allJobs, hasUnsavedChanges, historical, onShowOutputChange, pendingPhysicalPreviewId, revision]);
  const outputKey = selectedJob?.output ? `${selectedJob.id}:${selectedJob.output.sha256}` : "";
  const outputIsVideo = selectedJob?.output?.mediaType.toLowerCase() === "video/mp4";
  const outputIdentity = selectedJob?.output ? [selectedJob.id, selectedJob.shotRevision.toString(), selectedJob.output.sha256, selectedJob.output.contentUrl, selectedJob.output.mediaType, selectedJob.output.sizeBytes.toString(), selectedJob.output.widthPixels, selectedJob.output.heightPixels].join(":") : "";
  const selectedJobRef = useRef(selectedJob);
  selectedJobRef.current = selectedJob;
  const outputQuery = useQuery({
    queryKey: ["render-output", projectId, selectedJob?.id, selectedJob?.output?.sha256],
    enabled: showOutput && !outputIsVideo && selectedJob?.status === RenderJobStatus.SUCCEEDED && !!selectedJob.output,
    queryFn: async () => {
      if (!selectedJob?.output) throw new Error("The completed render has no output metadata.");
      const blob = await getRenderOutputContent(projectId, selectedJob.id, selectedJob.output.contentUrl);
      return verifyRenderOutputContent(blob, selectedJob.output);
    },
    staleTime: 0,
    gcTime: 0,
    retry: false,
  });

  useEffect(() => {
    if (!showOutput || !outputKey) { setOutputUrlState(undefined); setOutputLinkError(""); return; }
    const currentJob = selectedJobRef.current;
    if (!currentJob?.output) { setOutputUrlState(undefined); return; }
    if (currentJob.output.mediaType.toLowerCase() === "video/mp4") {
      try {
        const url = renderOutputContentURL(projectId, currentJob.id, currentJob.output.contentUrl);
        setOutputLinkError("");
        setOutputUrlState({ key: outputKey, url });
        onFinalOutputChange?.({ url, renderId: currentJob.id, revision: currentJob.shotRevision, metadata: currentJob.output, isVideo: true, isSample: currentJob.renderMode === RenderMode.VIDEO_SAMPLE_PNG });
      } catch (error) {
        setOutputUrlState(undefined);
        setOutputLinkError(errorMessage(error, t));
        onFinalOutputChange?.(undefined);
      }
      return () => { onFinalOutputChange?.(undefined); };
    }
    if (!outputQuery.data) { setOutputUrlState(undefined); return; }
    const url = URL.createObjectURL(outputQuery.data);
    setOutputUrlState({ key: outputKey, url });
    onFinalOutputChange?.({ url, renderId: currentJob.id, revision: currentJob.shotRevision, metadata: currentJob.output, isVideo: false, isSample: currentJob.renderMode === RenderMode.VIDEO_SAMPLE_PNG });
    return () => {
      URL.revokeObjectURL(url);
      onFinalOutputChange?.(undefined);
    };
  }, [projectId, outputIsVideo, outputQuery.data, outputKey, outputIdentity, showOutput, onFinalOutputChange, t]);

  const conflict = cancel.error instanceof ConnectError && cancel.error.code === Code.Aborted;
  const submitError = submit.error ? errorMessage(submit.error, t) : "";
  const outputError = outputQuery.error ? errorMessage(outputQuery.error, t) : "";
  const acceptErrorIsCurrent = acceptSample.variables?.projectId === projectId && acceptSample.variables.shotId === shotId && acceptSample.variables.revision === revision?.revision;
  const visibleAcceptError = acceptErrorIsCurrent ? acceptError : "";
  const revisionNumber = revision?.revision.toString() ?? "";
  const outputUrl = showOutput && outputUrlState?.key === outputKey ? outputUrlState.url : "";
  const selectedIsVideoSample = selectedJob?.renderMode === RenderMode.VIDEO_SAMPLE_PNG;
  const visibleJobId = selectedJob?.id ?? "";
  useEffect(() => {
    if (!showOutput || !outputUrl || !visibleJobId || !selectedIsVideoSample || outputIsVideo) return;
    setViewedSampleIds((current) => current.includes(visibleJobId) ? current : [...current, visibleJobId]);
  }, [showOutput, outputUrl, visibleJobId, selectedIsVideoSample, outputIsVideo]);
  const submitIsCurrent = submit.variables?.projectId === projectId && submit.variables.shotId === shotId && submit.variables.revision === revision?.revision;
  const submitErrorIsCurrent = submit.isError && submitIsCurrent;
  const submitBusyForCurrent = submit.isPending && submitIsCurrent;
  const videoTiming = revision?.snapshot?.videoTiming;
  const physicalBinding = revision?.snapshot?.physicalBinding;
  const physicalMotion = Boolean(physicalBinding?.motionOutput);
  const physicalShot = Boolean(physicalBinding);
  const physicalRecording = Boolean(physicalShot && videoTiming);
  const isVideo = Boolean(videoTiming || physicalMotion);
  const autoPhone = revision?.snapshot?.presentation?.kind === PresentationKind.PHONE_SYSTEM && revision.snapshot.presentation.phoneSystemChrome?.appearance === PresentationAppearance.AUTO;
  const sampleMode = physicalMotion ? RenderMode.PHYSICAL_VIDEO_FINAL_MP4 : videoTiming ? RenderMode.VIDEO_SAMPLE_PNG : physicalShot ? RenderMode.PHYSICAL_STILL_PNG : RenderMode.STILL;
  const finalMode = physicalMotion ? RenderMode.PHYSICAL_VIDEO_FINAL_MP4 : physicalShot ? RenderMode.PHYSICAL_STILL_PNG : isVideo ? RenderMode.VIDEO_FINAL_MP4 : RenderMode.STILL;
  const hasViewedSample = allJobs.some((job) => job.shotRevision === revision?.revision && job.renderMode === RenderMode.VIDEO_SAMPLE_PNG && job.status === RenderJobStatus.SUCCEEDED && viewedSampleIds.includes(job.id));
  const canFinalVideo = Boolean(isVideo && revision?.renderEligible && (physicalMotion || (!autoPhone ? hasViewedSample : videoTiming?.sampleBinding)));
  const submitInput = revision && revision.renderEligible ? {
    projectId, shotId, revision: revision.revision,
    idempotencyKey: retryKey(projectId, shotId, revision.revision, sampleMode), mode: sampleMode,
    profile: physicalMotion ? RenderOutputProfile.MP4_H264_SDR_V1 : RenderOutputProfile.PNG_V1,
  } : undefined;
  const previewInput = revision && revision.renderEligible && physicalShot ? {
    projectId, shotId, revision: revision.revision,
    idempotencyKey: retryKey(projectId, shotId, revision.revision, RenderMode.PHYSICAL_PREVIEW_PNG),
    mode: RenderMode.PHYSICAL_PREVIEW_PNG,
    profile: RenderOutputProfile.PHYSICAL_PREVIEW_PNG_V1,
  } : undefined;
  const finalInput = revision && canFinalVideo && videoTiming ? {
    projectId, shotId, revision: revision.revision,
    idempotencyKey: retryKey(projectId, shotId, revision.revision, finalMode), mode: finalMode,
    profile: RenderOutputProfile.MP4_H264_SDR_V1,
  } : undefined;
  const retrySubmit = () => {
    const submitted = submit.variables;
    if (submitted) submit.mutate(submitted);
  };

  return <Card className="shot-render-controls">
    <CardHeader><CardTitle as="h3">{t(strings.shot.renderTitle)}</CardTitle></CardHeader>
    <CardContent className="shot-fields">
      {!revision ? <p role="status">{t(strings.shot.renderRevisionLoading)}</p> : <>
        <p className="shot-help">{historical ? t(strings.shot.renderHistorical, { revision: revisionNumber }) : hasUnsavedChanges ? t(strings.shot.renderUnsaved, { revision: revisionNumber }) : t(strings.shot.renderSaved, { revision: revisionNumber })}</p>
        {!revision.renderEligible && <div role="alert"><p>{t(strings.shot.renderBlocked)}</p>{revision.issues.filter((issue) => issue.blocksRender).map((issue, index) => <p key={`${issue.fieldPath}-${index}`}>{issue.summary}{issue.repair ? ` ${issue.repair}` : ""}</p>)}</div>}
        {submitErrorIsCurrent && <div role="alert"><p>{submitError || t(strings.shot.renderSubmitError)}</p><Button variant="outline" className="min-h-11" disabled={submitBusyForCurrent} onClick={retrySubmit}>{t(strings.shot.renderRetrySubmit)}</Button></div>}
        {physicalShot && !physicalRecording && <div className="flex flex-wrap items-center gap-space-sm">
          <Button variant="outline" className="min-h-11" disabled={!previewInput || submitBusyForCurrent} onClick={() => previewInput && submit.mutate(previewInput)}>{submitBusyForCurrent ? t(strings.shot.renderSubmitting) : t(strings.physical.renderPreview)}</Button>
          <span className="shot-help">{t(strings.physical.previewDraft)}</span>
        </div>}
        {!isVideo && <Button className="studio-primary min-h-11" disabled={!submitInput || submitBusyForCurrent} onClick={() => submitInput && submit.mutate(submitInput)}>{submitBusyForCurrent ? t(strings.shot.renderSubmitting) : physicalShot ? t(strings.physical.renderStill) : t(strings.shot.renderRevisionAction, { revision: revisionNumber })}</Button>}
        {physicalMotion && <Button className="studio-primary min-h-11" disabled={!submitInput || submitBusyForCurrent} onClick={() => submitInput && submit.mutate(submitInput)}>{submitBusyForCurrent ? t(strings.shot.renderSubmitting) : t(strings.physical.renderMotion)}</Button>}
        {isVideo && !physicalMotion && <div className="flex flex-wrap gap-space-sm">
          <Button className="studio-primary min-h-11" disabled={!submitInput || submitBusyForCurrent} onClick={() => submitInput && submit.mutate(submitInput)}>{submitBusyForCurrent ? t(strings.shot.renderSubmitting) : t(strings.shot.renderVideoSample)}</Button>
          <Button variant="outline" className="min-h-11" disabled={!finalInput || submitBusyForCurrent} onClick={() => finalInput && submit.mutate(finalInput)}>{t(strings.shot.renderVideoFinal, { revision: revisionNumber })}</Button>
        </div>}
      </>}

      <h4>{t(strings.shot.renderJobs)}</h4>
      {jobs.error && <p role="alert">{errorMessage(jobs.error, t)}</p>}
      {jobs.isLoading && <p role="status">{t(strings.shot.renderLoadingJobs)}</p>}
      {!jobs.isLoading && allJobs.length === 0 && <p>{t(strings.shot.renderNoJobs)}</p>}
      {allJobs.map((job) => <div className="shot-render-job" key={job.id}>
        <button type="button" aria-pressed={selectedJob?.id === job.id} onClick={() => { setSelectedJobId(job.id); onShowOutputChange(false); }}>
          <span>{t(strings.shot.revision, { revision: job.shotRevision.toString() })}</span><span role="status">{statusLabel(job.status, t)}</span>
        </button>
        {job.issue?.summary && <p role="alert">{job.issue.summary}</p>}
        {job.summary && <p>{job.summary}</p>}
        {isRenderCancellable(job.status) && <Button variant="outline" className="min-h-11" disabled={cancel.isPending} onClick={() => cancel.mutate(job)}>{cancel.isPending ? t(strings.shot.renderCancelPending) : t(strings.shot.renderCancel)}</Button>}
        {cancel.error && selectedJob?.id === job.id && <p role="alert">{conflict ? t(strings.shot.renderCancelConflict) : errorMessage(cancel.error, t)}</p>}
      </div>)}
      {selectedJob?.status === RenderJobStatus.SUCCEEDED && selectedJob.output && <section className="shot-render-result" aria-label={t(strings.shot.renderOutputDetails)}>
        {(outputError || outputLinkError) && <p role="alert">{outputError || outputLinkError}</p>}
        {outputQuery.isFetching && <p role="status">{t(strings.shot.renderVerifyingOutput)}</p>}
        <div className="shot-actions">
          <Button variant="outline" className="min-h-11" onClick={() => { onShowOutputChange(!showOutput); setOutputUrlState(undefined); }}>{showOutput ? t(strings.shot.renderHideOutput) : t(selectedIsVideoSample ? strings.shot.renderViewSample : outputIsVideo ? strings.shot.renderViewFinalMP4 : strings.shot.renderViewOutput)}</Button>
          {outputUrl && <a className="shot-download min-h-11" href={outputUrl} download={`device-studio-${selectedJob.id}.${outputIsVideo ? "mp4" : "png"}`}>{t(outputIsVideo ? strings.shot.renderDownloadMP4 : strings.shot.renderDownload)}</a>}
        </div>
        {selectedIsVideoSample && autoPhone && outputUrl && showOutput && <div><Button data-testid={selectors.shot.videoSampleAccept} variant="outline" className="min-h-11" disabled={acceptSample.isPending || !onAcceptVideoSample} onClick={() => { setAcceptError(""); acceptSample.mutate({ projectId, shotId, revision: selectedJob.shotRevision, renderId: selectedJob.id }); }}>{acceptSample.isPending ? t(strings.shot.renderVerifyingOutput) : t(strings.shot.renderUseSample)}</Button>{visibleAcceptError && <p role="alert">{visibleAcceptError}</p>}</div>}
        {showOutput && outputUrl && <details><summary>{t(strings.shot.renderOutputDetails)}</summary><p>{t(strings.shot.renderOutputMetadata, { width: selectedJob.output.widthPixels, height: selectedJob.output.heightPixels, mediaType: selectedJob.output.mediaType, bytes: selectedJob.output.sizeBytes.toString() })}</p><code>{selectedJob.output.sha256}</code></details>}
      </section>}
      {jobs.hasNextPage && <Button variant="outline" className="min-h-11" disabled={jobs.isFetchingNextPage} onClick={() => void jobs.fetchNextPage()}>{jobs.isFetchingNextPage ? t(strings.shot.renderLoadingJobs) : t(strings.shot.renderLoadMore)}</Button>}
    </CardContent>
  </Card>;
}
