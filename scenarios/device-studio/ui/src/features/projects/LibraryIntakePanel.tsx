import { useRef, useState } from "react";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import type { IntakeCandidateRevision } from "@vrooli/proto-types/device-studio/v1/library/library_pb";
import { downloadLocalGeometryCandidate, importLocalGeometryCandidate, listLocalGeometryCandidates, MAX_LOCAL_GEOMETRY_BYTES } from "../../api/library-intake";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

function newCandidateID() {
  const suffix = globalThis.crypto?.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
  return `intake-${suffix}`;
}

export function LibraryIntakePanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const fileInput = useRef<HTMLInputElement>(null);
  const [file, setFile] = useState<File>();
  const [candidateId, setCandidateId] = useState(newCandidateID);
  const [displayName, setDisplayName] = useState("");
  const [sourceLabel, setSourceLabel] = useState("");
  const [licenseClaim, setLicenseClaim] = useState("");
  const [attribution, setAttribution] = useState("");
  const [downloadError, setDownloadError] = useState("");
  const candidates = useInfiniteQuery({
    queryKey: ["local-geometry-intake"],
    queryFn: ({ pageParam }) => listLocalGeometryCandidates(pageParam),
    initialPageParam: "",
    getNextPageParam: (lastPage) => lastPage.nextAfterCandidateId || undefined,
    retry: false,
  });
  const upload = useMutation({
    mutationFn: () => {
      if (!file) throw new Error("Choose a GLB file first.");
      return importLocalGeometryCandidate(file, { candidateId, displayName, sourceLabel, licenseClaim, attribution });
    },
    onSuccess: async () => {
      setFile(undefined);
      if (fileInput.current) fileInput.current.value = "";
      setCandidateId(newCandidateID());
      setDisplayName("");
      setSourceLabel("");
      setLicenseClaim("");
      setAttribution("");
      await queryClient.invalidateQueries({ queryKey: ["local-geometry-intake"] });
    },
  });
  const importReady = Boolean(file && file.size > 0 && file.size <= MAX_LOCAL_GEOMETRY_BYTES && displayName.trim() && sourceLabel.trim() && candidateId.trim());
  const download = async (candidate: IntakeCandidateRevision) => {
    setDownloadError("");
    try {
      const blob = await downloadLocalGeometryCandidate(candidate);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = candidate.originalFilename;
      anchor.click();
      window.setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (error) {
      setDownloadError(error instanceof Error ? error.message : t(strings.physical.intakeFailure));
    }
  };

  return <Card className="physical-shot__intake" data-testid="physical-library-intake">
    <CardHeader><CardTitle as="h3">{t(strings.physical.intakeTitle)}</CardTitle><p>{t(strings.physical.intakeHelp)}</p></CardHeader>
    <CardContent className="physical-shot__fields">
      <label>{t(strings.physical.intakeFile)}<input ref={fileInput} className="physical-shot__input" type="file" accept=".glb,model/gltf-binary" onChange={(event) => setFile(event.currentTarget.files?.[0])} /></label>
      {file && <p>{file.name} · {file.size.toLocaleString()} bytes</p>}
      <label>{t(strings.physical.intakeDisplayName)}<input className="physical-shot__input" value={displayName} maxLength={120} onChange={(event) => setDisplayName(event.target.value)} /></label>
      <label>{t(strings.physical.intakeSource)}<input className="physical-shot__input" value={sourceLabel} maxLength={200} onChange={(event) => setSourceLabel(event.target.value)} /></label>
      <label>{t(strings.physical.intakeLicense)}<input className="physical-shot__input" value={licenseClaim} maxLength={200} onChange={(event) => setLicenseClaim(event.target.value)} /></label>
      <label>{t(strings.physical.intakeAttribution)}<input className="physical-shot__input" value={attribution} maxLength={200} onChange={(event) => setAttribution(event.target.value)} /></label>
      <p className="physical-shot__candidate">{t(strings.physical.intakeUnreviewed)}</p>
      <Button type="button" className="studio-primary min-h-11" disabled={!importReady || upload.isPending} onClick={() => upload.mutate()}>{upload.isPending ? t(strings.physical.intakeUploading) : t(strings.physical.intakeImport)}</Button>
      <details>
        <summary>{t(strings.physical.intakeTechnicalDetails)}</summary>
        <label>{t(strings.physical.intakeCandidateId)}<input className="physical-shot__input" value={candidateId} maxLength={128} onChange={(event) => setCandidateId(event.target.value)} /></label>
      </details>
      {upload.error && <p role="alert">{upload.error.message}</p>}
      {candidates.error && <p role="alert">{t(strings.physical.intakeListUnavailable)}</p>}
      {downloadError && <p role="alert">{downloadError}</p>}
      {candidates.data && <div><h4>{t(strings.physical.intakeCandidates)}</h4>{candidates.data.pages.flatMap((page) => page.candidates).length === 0 ? <p>{t(strings.physical.intakeEmpty)}</p> : <ul>{candidates.data.pages.flatMap((page) => page.candidates).map((candidate) => <li key={`${candidate.id}@${candidate.revision}`}>
        <strong>{candidate.displayName}</strong> · {candidate.originalFilename} · {candidate.sizeBytes.toString()} bytes<br />
        {t(strings.physical.intakeStatus, { review: candidate.reviewStatus, render: candidate.renderStatus })} · SHA-256 {candidate.originalSha256.slice(0, 12)}…
        {candidate.inspection && <span> · {candidate.inspection.supported ? t(strings.physical.intakeSupported) : t(strings.physical.intakeUnsupported)}{candidate.inspection.finding ? ` · ${candidate.inspection.finding}` : ""}</span>}
        <Button type="button" variant="outline" className="min-h-11" onClick={() => void download(candidate)}>{t(strings.physical.intakeDownload)}</Button>
      </li>)}</ul>}{candidates.hasNextPage && <Button type="button" variant="outline" className="min-h-11" disabled={candidates.isFetchingNextPage} onClick={() => void candidates.fetchNextPage()}>{t(strings.physical.intakeLoadMore)}</Button>}</div>}
    </CardContent>
  </Card>;
}
