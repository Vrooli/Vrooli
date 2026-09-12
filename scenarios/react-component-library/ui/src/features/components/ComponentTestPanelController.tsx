import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";

import {
  getComponentTestReport,
  listComponentTestReports,
  runComponentTest,
} from "../../api/componentTests";
import type { ComponentExperience } from "../../api/components";
import {
  ComponentTestPanelView,
  EVIDENCE_KINDS,
  type CaptureItem,
  storyCaptureLabel,
} from "./ComponentTestPanelView";

export function ComponentTestPanel({
  componentId,
  version,
  experience,
}: {
  componentId: string;
  version: string;
  experience?: ComponentExperience;
}) {
  const [search] = useSearchParams();
  const [includeClosure, setIncludeClosure] = useState(false);
  const failedClaims = (experience?.claims ?? []).filter((claim) =>
    (experience?.evidence ?? []).some(
      (item) => item.claimId === claim.id && item.verdict === "failed",
    ),
  );
  const [selectedClaimID, setSelectedClaimID] = useState("");
  const [selectedCaptureKind, setSelectedCaptureKind] = useState("screenshot");
  const [selectedStoryID, setSelectedStoryID] = useState("");
  const reportID = search.get("testReport") || "";
  const queryClient = useQueryClient();
  const reports = useQuery({
    queryKey: ["component-tests", componentId, version],
    queryFn: () => listComponentTestReports({ componentId, version }),
    enabled: Boolean(componentId && version),
  });
  const selected = useQuery({
    queryKey: ["component-test-report", reportID],
    queryFn: () => getComponentTestReport(reportID),
    enabled: Boolean(reportID),
  });
  const run = useMutation({
    mutationFn: () => runComponentTest({ componentId, version, includeClosure }),
    onSuccess: () =>
      void queryClient.invalidateQueries({ queryKey: ["component-tests", componentId, version] }),
  });
  const latest = run.data ?? selected.data ?? reports.data?.[0];
  const capturedArtifacts = latest?.artifacts ?? [];
  const capturedStoryIDs = Array.from(
    new Set(
      capturedArtifacts
        .filter(
          (artifact) => artifact.kind === "bas-story-sheet" || artifact.kind === "bas-screenshot",
        )
        .map((artifact) => artifact.storyId)
        .filter((storyID): storyID is string => Boolean(storyID)),
    ),
  );
  const storySheets = capturedArtifacts.filter((artifact) => artifact.kind === "bas-story-sheet");
  const defaultStoryID =
    capturedStoryIDs.find((storyID) => storyID.startsWith("review-sheet:")) ??
    capturedStoryIDs[0] ??
    "";
  const activeStoryID = selectedStoryID || defaultStoryID;
  const storyChoices = capturedStoryIDs.map((storyID) => ({
    id: storyID,
    label: storyCaptureLabel(storyID),
  }));
  const activeClaimID = selectedClaimID || failedClaims[0]?.id || "";
  const activeClaim = experience?.claims.find((claim) => claim.id === activeClaimID);
  const activeEvidence = experience?.evidence.find(
    (item) => item.claimId === activeClaimID && item.verdict === "failed",
  );
  const measurement = activeEvidence?.measurement;
  const overlaySubjects = (measurement?.subjects ?? []).map((subject) => ({
    id: subject.testId || subject.elementId,
    label: subject.value || subject.testId || subject.elementId,
    x: subject.bounds?.x ?? 0,
    y: subject.bounds?.y ?? 0,
    width: subject.bounds?.width ?? 0,
    height: subject.bounds?.height ?? 0,
  }));
  const captureItems: CaptureItem[] = EVIDENCE_KINDS.map(({ id, label, aliases }) => ({
    id,
    kind: id,
    label,
    artifact: capturedArtifacts.find(
      (candidate) =>
        aliases.includes(candidate.kind) && (!activeStoryID || candidate.storyId === activeStoryID),
    ),
  })).map((item) => ({
    ...item,
    available: Boolean(item.artifact),
    status: item.artifact ? ("available" as const) : ("missing" as const),
  }));
  const hasBASArtifacts = capturedArtifacts.some((artifact) => artifact.kind.startsWith("bas-"));
  const preferredCaptureKind = capturedArtifacts.some(
    (artifact) => artifact.kind === "bas-story-sheet",
  )
    ? "story-sheet"
    : "screenshot";
  const selectedCapture =
    captureItems.find((item) => item.id === selectedCaptureKind && item.available) ??
    captureItems.find((item) => item.id === preferredCaptureKind && item.available) ??
    captureItems.find((item) => item.id === "screenshot") ??
    captureItems[0];
  const selectStory = (storyID: string) => {
    setSelectedStoryID(storyID);
    setSelectedCaptureKind(storyID.startsWith("review-sheet:") ? "story-sheet" : "screenshot");
  };

  return (
    <ComponentTestPanelView
      version={version}
      includeClosure={includeClosure}
      setIncludeClosure={setIncludeClosure}
      failedClaims={failedClaims}
      setSelectedClaimID={setSelectedClaimID}
      setSelectedCaptureKind={setSelectedCaptureKind}
      latest={latest}
      reports={reports}
      selected={selected}
      run={run}
      activeStoryID={activeStoryID}
      storySheets={storySheets}
      storyChoices={storyChoices}
      selectStory={selectStory}
      activeClaimID={activeClaimID}
      activeClaim={activeClaim}
      activeEvidence={activeEvidence}
      measurement={measurement}
      overlaySubjects={overlaySubjects}
      captureItems={captureItems}
      hasBASArtifacts={hasBASArtifacts}
      selectedCapture={selectedCapture}
    />
  );
}
