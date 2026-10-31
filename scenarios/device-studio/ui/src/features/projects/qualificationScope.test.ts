import { describe, expect, it } from "vitest";
import { qualificationScopeStatus, type QualificationReviewScope, type SelectedQualificationSetup } from "./qualificationScope";

const review: QualificationReviewScope = {
  scope: "visual_setup", result: "passed", candidateRevision: 2, candidateSha256: "a".repeat(64),
  dimensionStates: ["open"], displayIds: ["front-display"], allowedMotionIds: ["rigid_body"], supportIds: ["table-feet"],
  presetId: "reviewed-preset", presetRevision: 5, presetSha256: "b".repeat(64),
  sceneId: "reviewed-scene", sceneRevision: 3, sceneSha256: "c".repeat(64),
  motionId: "reviewed-motion", motionRevision: 4, motionSha256: "d".repeat(64),
};
const exactSetup: SelectedQualificationSetup = {
  device: { id: "device", revision: 2, sha256: "a".repeat(64) },
  scene: { id: "reviewed-scene", revision: 3, sha256: "c".repeat(64) },
  motion: { id: "reviewed-motion", revision: 4, sha256: "d".repeat(64) },
  preset: { id: "reviewed-preset", revision: 5, sha256: "b".repeat(64) },
  displayId: "front-display", deviceSupportId: "table-feet", sceneSupportId: "table", cameraId: "front-camera",
};

describe("scoped qualification status", () => {
  it("labels the exact native-reviewed binding with its limited scope", () => {
    const status = qualificationScopeStatus([review], exactSetup);
    expect(status.exactMatch).toBe(true);
    expect(status.text).toContain("Reviewed visual setup binding");
    expect(status.text).toContain("preset reviewed-preset@5");
    expect(status.text).toContain("scene reviewed-scene@3");
    expect(status.text).toContain("motion reviewed-motion@4");
    expect(status.text).toContain("display front-display");
    expect(status.text).toContain("device support table-feet");
  });

  it.each([
    ["different motion", { ...exactSetup, motion: { id: "other-motion", revision: 8, sha256: "e".repeat(64) } }],
    ["different scene", { ...exactSetup, scene: { id: "other-scene", revision: 9, sha256: "f".repeat(64) } }],
  ])("shows a setup with %s as unreviewed and names both bindings", (_label, setup) => {
    const status = qualificationScopeStatus([review], setup);
    expect(status.hasPassedReview).toBe(true);
    expect(status.exactMatch).toBe(false);
    expect(status.text).toContain("Current selected setup is unreviewed.");
    expect(status.text).toContain("Review on file:");
    expect(status.text).toContain("preset reviewed-preset@5");
    expect(status.text).toContain("display front-display");
    expect(status.text).toContain("device support table-feet");
    expect(status.text).toContain("motion scope rigid_body");
    expect(status.text).toContain("scene support table");
    expect(status.text).toContain("camera front-camera");
  });
});
