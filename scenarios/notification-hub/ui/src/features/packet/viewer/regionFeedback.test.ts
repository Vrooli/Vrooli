import { describe, expect, it } from "vitest";

import { makeAttachment } from "../mocks/askFactory";
import { isRegionFeedbackFolder, parseRegionFeedbackBundle, regionCorrectionFromPacket, regionEntryPath, validateRegionEntries } from "./regionFeedback";

const digest = (letter: string) => letter.repeat(64);

function fixture() {
  const manifest = {
    schema: "ux-region-feedback/v1",
    id: "compare-1",
    route: "/settings",
    state_id: "account-panel-open",
    data_id: "settings-fixture-7",
    before: { id: "capture-a", build_id: `sha256:${digest("a")}`, route: "/settings", state_id: "account-panel-open", data_id: "settings-fixture-7", fidelity: "captured", viewport: { width: 390, height: 844 }, image: { path: "captures/before.png", sha256: digest("1") } },
    after: { id: "capture-b", build_id: `sha256:${digest("b")}`, route: "/settings", state_id: "account-panel-open", data_id: "settings-fixture-7", fidelity: "composited", viewport: { width: 390, height: 844 }, image: { path: "captures/after.png", sha256: digest("2") } },
    regions: [{ id: "settings-nav", node_id: "outline-23", label: "Settings navigation", component: "SettingsNav", evidence_id: "crop-evidence-23", bounds: { x: 0.1, y: 0.2, width: 0.5, height: 0.2 }, crop: { path: "regions/crops/settings.png", sha256: digest("3") } }],
  };
  const folder = makeAttachment({ name: "capture-review", folder: true, entries: [
    { path: "ux-region-feedback.json", mediaType: "application/json", sha256: digest("0") },
    { path: "captures/before.png", mediaType: "image/png", sha256: digest("1"), sizeBytes: 8192 },
    { path: "captures/after.png", mediaType: "image/png", sha256: digest("2"), sizeBytes: 8192 },
    { path: "regions/crops/settings.png", mediaType: "image/png", sha256: digest("3"), sizeBytes: 2048 },
    { path: "regions/settings-nav.json", mediaType: "application/json", sha256: digest("4") },
  ] });
  return { manifest, folder };
}

describe("region feedback packet contract", () => {
  it("accepts a versioned compare, validates attached images, and provides an answerable region path", () => {
    const { manifest, folder } = fixture();
    const parsed = parseRegionFeedbackBundle(manifest);
    expect(isRegionFeedbackFolder(folder)).toBe(true);
    expect(parsed?.issues).toEqual([]);
    expect(validateRegionEntries(parsed!.bundle, folder)).toEqual([]);
    expect(regionEntryPath("settings-nav")).toBe("regions/settings-nav.json");
  });

  it("keeps identity and fidelity mismatches visible instead of claiming a direct compare", () => {
    const { manifest } = fixture();
    manifest.after.route = "/profile";
    manifest.after.viewport.width = 412;
    manifest.after.fidelity = "sketch";
    const parsed = parseRegionFeedbackBundle(manifest);
    expect(parsed?.issues.map((issue) => issue.code)).toContain("routeMismatch");
    expect(parsed?.issues.map((issue) => issue.code)).toContain("viewportMismatch");
    expect(parsed?.bundle.after.fidelity).toBe("sketch");
  });

  it("rejects unknown fidelity, duplicate region ids, and out-of-range hit targets", () => {
    const { manifest } = fixture();
    const [region] = manifest.regions;
    if (!region) throw new Error("the fixture has a region");
    expect(parseRegionFeedbackBundle({ ...manifest, before: { ...manifest.before, fidelity: "confirmed" } })).toBeNull();
    expect(parseRegionFeedbackBundle({ ...manifest, before: { ...manifest.before, image: { ...manifest.before.image, path: "../private.png" } } })).toBeNull();
    expect(parseRegionFeedbackBundle({ ...manifest, regions: [{ ...region, crop: { ...region.crop, path: "drive:private.png" } }] })).toBeNull();
    expect(parseRegionFeedbackBundle({ ...manifest, regions: [region, region] })).toBeNull();
    expect(parseRegionFeedbackBundle({ ...manifest, regions: [{ ...region, bounds: { x: 0.8, y: 0, width: 0.3, height: 0.5 } }] })).toBeNull();
  });

  it("accepts an explicit bundle kind but never mistakes a correction event for a bundle", () => {
    const { manifest } = fixture();
    expect(parseRegionFeedbackBundle({ ...manifest, kind: "bundle" })).not.toBeNull();
    expect(parseRegionFeedbackBundle({ ...manifest, kind: "correction" })).toBeNull();
  });

  it("maps the exact packet path and answer text into the shared correction contract", () => {
    const { manifest } = fixture();
    const parsed = parseRegionFeedbackBundle(manifest);
    const event = regionCorrectionFromPacket({
      packetId: "ask-17", attachmentName: "capture-review", regionPath: "regions/settings-nav.json",
      text: "Keep the selected destination visible.", ownerScenario: "notification-hub",
      corpus: "ui-health-ex1-private", correctionId: "correction-17", bundle: parsed!.bundle,
    });
    expect(event).toMatchObject({
      schema: "ux-region-feedback/v1", kind: "correction", correction_id: "correction-17",
      origin: { kind: "hub-packet", id: "ask-17", attachment: "capture-review" },
      capture: { id: "capture-b", fidelity: "composited", image: { path: "captures/after.png" } },
      region: { id: "settings-nav", node_id: "outline-23", component: "SettingsNav" },
      text: "Keep the selected destination visible.",
    });
    expect(regionCorrectionFromPacket({
      packetId: "ask-17", attachmentName: "capture-review", regionPath: "regions/other.json",
      text: "Keep the selected destination visible.", ownerScenario: "notification-hub",
      corpus: "ui-health-ex1-private", correctionId: "correction-17", bundle: parsed!.bundle,
    })).toBeNull();
  });

  it("refuses a crop whose attached bytes do not match its declared digest", () => {
    const { manifest, folder } = fixture();
    const parsed = parseRegionFeedbackBundle(manifest)!;
    const wrong = folder.entries.map((entry) => entry.path === "regions/crops/settings.png" ? { ...entry, sha256: digest("9") } : entry);
    const changedFolder = { ...folder, entries: wrong };
    expect(validateRegionEntries(parsed.bundle, changedFolder)[0]?.code).toBe("regionCropMismatch");
  });
});
