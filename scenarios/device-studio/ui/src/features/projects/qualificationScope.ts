type Ref = { id: string; revision: number; sha256: string };

export type QualificationReviewScope = {
  scope: string;
  result: string;
  candidateRevision: number;
  candidateSha256: string;
  dimensionStates: string[];
  displayIds: string[];
  allowedMotionIds: string[];
  supportIds: string[];
  presetId: string;
  presetRevision: number;
  presetSha256: string;
  sceneId: string;
  sceneRevision: number;
  sceneSha256: string;
  motionId: string;
  motionRevision: number;
  motionSha256: string;
};

export type SelectedQualificationSetup = {
  device: Ref;
  scene: Ref;
  motion: Ref;
  preset: Ref;
  displayId: string;
  deviceSupportId: string;
  sceneSupportId: string;
  cameraId: string;
};

function refText(ref: Ref): string {
  return `${ref.id}@${ref.revision} sha256=${ref.sha256}`;
}

export function matchesQualificationScope(review: QualificationReviewScope, selected: SelectedQualificationSetup): boolean {
  return review.result === "passed" &&
    review.candidateRevision === selected.device.revision && review.candidateSha256 === selected.device.sha256 &&
    review.presetId === selected.preset.id && review.presetRevision === selected.preset.revision && review.presetSha256 === selected.preset.sha256 &&
    review.sceneId === selected.scene.id && review.sceneRevision === selected.scene.revision && review.sceneSha256 === selected.scene.sha256 &&
    review.motionId === selected.motion.id && review.motionRevision === selected.motion.revision && review.motionSha256 === selected.motion.sha256 &&
    review.displayIds.includes(selected.displayId) && review.supportIds.includes(selected.deviceSupportId);
}

export function reviewedScopeText(review: QualificationReviewScope): string {
  const label = review.scope === "visual_setup" ? "visual setup" : "physical geometry";
  return `Reviewed ${label} binding: parent device revision ${review.candidateRevision} sha256=${review.candidateSha256}; preset ${review.presetId}@${review.presetRevision} sha256=${review.presetSha256}; scene ${review.sceneId}@${review.sceneRevision} sha256=${review.sceneSha256}; motion ${review.motionId}@${review.motionRevision} sha256=${review.motionSha256}; display ${review.displayIds.join(", ") || "none"}; device support ${review.supportIds.join(", ") || "none"}; motion scope ${review.allowedMotionIds.join(", ") || "none"}; dimensions ${review.dimensionStates.join(", ") || "none"}.`;
}

export function selectedSetupText(selected: SelectedQualificationSetup): string {
  return `Current selected setup: preset ${refText(selected.preset)}; device ${refText(selected.device)}; scene ${refText(selected.scene)}; motion ${refText(selected.motion)}; display ${selected.displayId}; device support ${selected.deviceSupportId}; scene support ${selected.sceneSupportId}; camera ${selected.cameraId}.`;
}

export function qualificationScopeStatus(reviews: QualificationReviewScope[], selected?: SelectedQualificationSetup): { hasPassedReview: boolean; exactMatch: boolean; text: string } {
  const passed = reviews.filter((review) => review.result === "passed");
  if (passed.length === 0) return { hasPassedReview: false, exactMatch: false, text: "No accepted scoped review is available; treat this selection as unreviewed." };
  const exact = selected ? passed.find((review) => matchesQualificationScope(review, selected)) : undefined;
  if (exact) return { hasPassedReview: true, exactMatch: true, text: reviewedScopeText(exact) };
  const summaries = passed.map(reviewedScopeText).join(" ");
  const current = selected ? selectedSetupText(selected) : "No complete selected setup is available.";
  return { hasPassedReview: true, exactMatch: false, text: `Current selected setup is unreviewed. ${current} Review on file: ${summaries}` };
}
