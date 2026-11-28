import { describe, expect, it } from "vitest";
import { applyCaptureOutlinePatch, CaptureOutlinePatchError, validateCaptureOutlinePatch, type CaptureOutlinePatch } from "./captureOutlinePatch";
import type { DesignDocument } from "./design-types";
import type { DesignPatchOperation } from "./studio/proposal";

const digest = (value: string) => value.repeat(64);

function patch(): CaptureOutlinePatch {
  return {
    schema: "rcl.capture-outline-patch/v1",
    patch_id: "patch-17",
    source: {
      id: "capture-17",
      scenario: "notification-hub",
      build_id: `sha256:${digest("a")}`,
      route: "/settings",
      state_id: "navigation-open",
      data_id: "fixture-account-3",
      viewport: { width: 390, height: 844 },
      image: { path: "captures/before.png", sha256: digest("b") },
      outline: { path: "outline/settings.json", sha256: digest("c") },
    },
    preview: {
      id: "render-17",
      scenario: "notification-hub",
      build_id: `sha256:${digest("d")}`,
      route: "/settings",
      state_id: "navigation-open",
      data_id: "fixture-account-3",
      viewport: { width: 390, height: 844 },
      fidelity: "rendered",
      image: { path: "captures/after.png", sha256: digest("e") },
    },
    target: {
      region_id: "settings-primary-action",
      outline_node_id: "node-24",
      design_frame_id: "frame-phone",
      design_node_ids: ["node-24", "node-24.label"],
      component_identity: "Button/2.2.14",
      bounds: { x: 0.08, y: 0.72, width: 0.84, height: 0.12 },
    },
    composition: {
      design_id: "design-17",
      base_revision: 4,
      operation_id: "operation-8",
      summary: "Keep the primary action on one row.",
      operations: [{ kind: "setProp", nodeId: "node-24", key: "density", value: "compact" }],
    },
  };
}

describe("capture outline patch", () => {
  it("accepts a bounded patch tied to one capture and selected region", () => {
    expect(() => validateCaptureOutlinePatch(patch())).not.toThrow();
  });

  it("rejects a stale or mismatched schema before import", () => {
    const candidate = patch();
    Object.assign(candidate, { schema: "rcl.capture-outline-patch/v0" });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(CaptureOutlinePatchError);
  });

  it("rejects fields that the versioned schema does not define", () => {
    const candidate = patch();
    Object.assign(candidate.target, { neighboringRegionIds: ["other-region"] });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/target contains unsupported fields/);
  });

  it("rejects query strings and escaping evidence paths", () => {
    const candidate = patch();
    Object.assign(candidate.source, { route: "/settings?account=private" });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/query-free/);
    Object.assign(candidate.source, { route: "/settings" });
    Object.assign(candidate.source.outline, { path: "../outline.json" });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/safe relative path/);
  });

  it("rejects a region selection whose geometry escapes the captured image", () => {
    const candidate = patch();
    Object.assign(candidate.target, { bounds: { x: 0.8, y: 0.8, width: 0.3, height: 0.3 } });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/fit inside/);
  });

  it("requires the preview to match route, state, data, and viewport at component-backed fidelity", () => {
    const candidate = patch();
    Object.assign(candidate.preview, { data_id: "different-account" });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/exact source scenario, state, and data/);
    Object.assign(candidate.preview, { data_id: "fixture-account-3", fidelity: "sketch" });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/fidelity must be composited or rendered/);
    Object.assign(candidate.preview, { fidelity: "rendered", viewport: { width: 1280, height: 800 } });
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/exact source viewport/);
  });

  it("rejects edits that change a different node or the page-wide theme", () => {
    const candidate = patch();
    candidate.composition.operations = [{ kind: "setText", nodeId: "unrelated", text: "Replace" }];
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/escapes the selected region/);
    candidate.composition.operations = [{ kind: "setTheme", key: "accent", value: "#000000" }];
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/not a region-scoped edit/);
  });

  it("caps operation count and requires a node map for a patch target", () => {
    const candidate = patch();
    candidate.composition.operations = Array.from({ length: 9 }, (_, index): DesignPatchOperation => ({ kind: "setProp", nodeId: "node-24", key: `p${index}`, value: "x" }));
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/1–8/);
    candidate.composition.operations = [{ kind: "removeLayer", nodeId: "node-24" }];
    candidate.target.design_node_ids = [];
    expect(() => validateCaptureOutlinePatch(candidate)).toThrow(/unique non-empty mapped node set/);
  });

  it("applies against the recorded composition revision without mutating the source document", () => {
    const candidate = patch();
    candidate.composition.operations = [{ kind: "setText", nodeId: "node-24", text: "Continue" }];
    const doc: DesignDocument = {
      id: "design-17", name: "Settings", updatedAt: "2026-11-22T00:00:00Z", revision: 4,
      frames: [{ id: "frame-phone", name: "Phone", device: "phone", spec: {
        id: "page-root", type: "page", props: {}, children: [{ id: "node-24", type: "button", props: { text: "Next" }, children: [{ id: "node-24.label", type: "text", props: { text: "Next" } }] }],
      } }],
      theme: { bg: "#ffffff", surface: "#ffffff", surfaceAlt: "#eeeeee", ink: "#111111", muted: "#666666", line: "#dddddd", accent: "#334455", accentInk: "#ffffff", accentSoft: "#ddeeff", secondary: "#667788" },
    };
    const result = applyCaptureOutlinePatch(doc, candidate);

    expect(result.changed).toEqual(["node-24"]);
    expect(result.doc.frames[0]?.spec.children?.[0]?.props.text).toBe("Continue");
    expect(doc.frames[0]?.spec.children?.[0]?.props.text).toBe("Next");
  });

  it("requires every mapped node to belong to the exact target frame", () => {
    const candidate = patch();
    candidate.target.design_frame_id = "frame-tablet";
    const doc: DesignDocument = {
      id: "design-17", name: "Settings", updatedAt: "2026-11-22T00:00:00Z", revision: 4,
      frames: [{ id: "frame-phone", name: "Phone", device: "phone", spec: {
        id: "page-root", type: "page", props: {}, children: [{ id: "node-24", type: "button", props: { text: "Next" } }],
      } }],
      theme: { bg: "#ffffff", surface: "#ffffff", surfaceAlt: "#eeeeee", ink: "#111111", muted: "#666666", line: "#dddddd", accent: "#334455", accentInk: "#ffffff", accentSoft: "#ddeeff", secondary: "#667788" },
    };
    expect(() => applyCaptureOutlinePatch(doc, candidate)).toThrow(/target design frame/);
    candidate.target.design_frame_id = "frame-phone";
    candidate.target.design_node_ids = ["another-frame-node"];
    candidate.composition.operations = [{ kind: "setText", nodeId: "another-frame-node", text: "Outside" }];
    expect(() => applyCaptureOutlinePatch(doc, candidate)).toThrow(/belong to the named frame/);
  });

  it("refuses to apply after the design revision has advanced", () => {
    const candidate = patch();
    candidate.composition.operations = [{ kind: "setText", nodeId: "node-24", text: "Continue" }];
    const doc: DesignDocument = {
      id: "design-17", name: "Settings", updatedAt: "2026-11-22T00:00:00Z", revision: 5,
      frames: [{ id: "frame-phone", name: "Phone", device: "phone", spec: { id: "node-24", type: "button", props: { text: "Next" } } }],
      theme: { bg: "#ffffff", surface: "#ffffff", surfaceAlt: "#eeeeee", ink: "#111111", muted: "#666666", line: "#dddddd", accent: "#334455", accentInk: "#ffffff", accentSoft: "#ddeeff", secondary: "#667788" },
    };
    expect(() => applyCaptureOutlinePatch(doc, candidate)).toThrow(/composition changed/);
  });
});
