import type { DesignPatchOperation } from "./studio/proposal";
import type { DesignDocument } from "./design-types";
import { applyPatch, type AppliedPatch } from "./studio/patch";
import type { TemplateNode } from "./templates/spec";

/** One portable, versioned change set from a captured outline into an RCL composition. */
export const CAPTURE_OUTLINE_PATCH_SCHEMA = "rcl.capture-outline-patch/v1" as const;

export interface PatchViewport {
  width: number;
  height: number;
}

export interface PatchEvidenceRef {
  path: string;
  sha256: string;
}

export interface PatchCaptureIdentity {
  id: string;
  scenario: string;
  build_id: string;
  route: string;
  state_id: string;
  data_id: string;
  viewport: PatchViewport;
  image: PatchEvidenceRef;
  outline: PatchEvidenceRef;
}

export interface PatchTargetRegion {
  region_id: string;
  outline_node_id: string;
  /** Exact display/frame in the RCL document that owns the mapped nodes. */
  design_frame_id: string;
  /** RCL node ids that the outline mapper proved belong to this region. */
  design_node_ids: string[];
  component_identity?: string;
  bounds: { x: number; y: number; width: number; height: number };
}

export interface PatchPreviewIdentity {
  id: string;
  scenario: string;
  build_id: string;
  route: string;
  state_id: string;
  data_id: string;
  viewport: PatchViewport;
  fidelity: "composited" | "rendered";
  image: PatchEvidenceRef;
}

export interface CaptureOutlinePatch {
  schema: typeof CAPTURE_OUTLINE_PATCH_SCHEMA;
  patch_id: string;
  source: PatchCaptureIdentity;
  target: PatchTargetRegion;
  preview: PatchPreviewIdentity;
  composition: {
    design_id: string;
    base_revision: number;
    operation_id: string;
    summary: string;
    operations: DesignPatchOperation[];
  };
}

const ID = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,199}$/;
const COMPONENT_ID = /^[A-Za-z0-9][A-Za-z0-9._:@/-]{0,199}$/;
const SHA256 = /^[a-f0-9]{64}$/;
const RELATIVE_PATH = /^[A-Za-z0-9._~!$&'()*+,;=@%/-]+$/;
const ROUTE = /^\/[A-Za-z0-9._~!$&'()*+,;=:@%/-]*$/;
const MAX_OPERATIONS = 8;

export class CaptureOutlinePatchError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "CaptureOutlinePatchError";
  }
}

function record(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function exactKeys(value: Record<string, unknown>, keys: string[], field: string): void {
  const extras = Object.keys(value).filter((key) => !keys.includes(key));
  if (extras.length) throw new CaptureOutlinePatchError(`${field} contains unsupported fields: ${extras.join(", ")}`);
}

function id(value: unknown, field: string): asserts value is string {
  if (typeof value !== "string" || !ID.test(value)) {
    throw new CaptureOutlinePatchError(`${field} must be a bounded stable identity`);
  }
}

function evidence(value: unknown, field: string): asserts value is PatchEvidenceRef {
  if (!record(value)) throw new CaptureOutlinePatchError(`${field} needs a safe relative path and SHA-256`);
  exactKeys(value, ["path", "sha256"], field);
  if (typeof value.path !== "string" || !RELATIVE_PATH.test(value.path) ||
      value.path.split("/").some((part) => part === "" || part === "." || part === "..") ||
      typeof value.sha256 !== "string" || !SHA256.test(value.sha256)) {
    throw new CaptureOutlinePatchError(`${field} needs a safe relative path and SHA-256`);
  }
}

function viewport(value: unknown, field: string): asserts value is PatchViewport {
  if (!record(value)) throw new CaptureOutlinePatchError(`${field} needs positive bounded dimensions`);
  exactKeys(value, ["width", "height"], field);
  if (!Number.isSafeInteger(value.width) || !Number.isSafeInteger(value.height) ||
      Number(value.width) < 1 || Number(value.height) < 1 || Number(value.width) > 10000 || Number(value.height) > 10000) {
    throw new CaptureOutlinePatchError(`${field} needs positive bounded dimensions`);
  }
}

/**
 * Validate an exported outline patch before it is copied into an implementer
 * packet. The capture evidence is immutable; the composition patch may only
 * touch RCL nodes proven to belong to the selected outline region.
 */
export function validateCaptureOutlinePatch(value: unknown): asserts value is CaptureOutlinePatch {
  if (!record(value) || value.schema !== CAPTURE_OUTLINE_PATCH_SCHEMA) {
    throw new CaptureOutlinePatchError(`schema must be ${CAPTURE_OUTLINE_PATCH_SCHEMA}`);
  }
  exactKeys(value, ["schema", "patch_id", "source", "target", "preview", "composition"], "patch");
  id(value.patch_id, "patch_id");
  if (!record(value.source)) throw new CaptureOutlinePatchError("source capture identity is required");
  exactKeys(value.source, ["id", "scenario", "build_id", "route", "state_id", "data_id", "viewport", "image", "outline"], "source");
  for (const field of ["id", "scenario", "build_id", "state_id", "data_id"] as const) id(value.source[field], `source.${field}`);
  if (typeof value.source.route !== "string" || !ROUTE.test(value.source.route)) {
    throw new CaptureOutlinePatchError("source.route must be a query-free route path");
  }
  viewport(value.source.viewport, "source.viewport");
  evidence(value.source.image, "source.image");
  evidence(value.source.outline, "source.outline");
  if (!record(value.preview)) throw new CaptureOutlinePatchError("a component-backed before/after preview is required");
  exactKeys(value.preview, ["id", "scenario", "build_id", "route", "state_id", "data_id", "viewport", "fidelity", "image"], "preview");
  for (const field of ["id", "scenario", "build_id", "state_id", "data_id"] as const) id(value.preview[field], `preview.${field}`);
  if (value.preview.scenario !== value.source.scenario || value.preview.state_id !== value.source.state_id || value.preview.data_id !== value.source.data_id) {
    throw new CaptureOutlinePatchError("preview must preserve the exact source scenario, state, and data identity");
  }
  if (typeof value.preview.route !== "string" || !ROUTE.test(value.preview.route) || value.preview.route !== value.source.route) {
    throw new CaptureOutlinePatchError("preview must preserve the exact source route");
  }
  if (value.preview.fidelity !== "composited" && value.preview.fidelity !== "rendered") {
    throw new CaptureOutlinePatchError("preview fidelity must be composited or rendered");
  }
  viewport(value.preview.viewport, "preview.viewport");
  if (value.preview.viewport.width !== value.source.viewport.width || value.preview.viewport.height !== value.source.viewport.height) {
    throw new CaptureOutlinePatchError("preview must preserve the exact source viewport");
  }
  evidence(value.preview.image, "preview.image");

  if (!record(value.target)) throw new CaptureOutlinePatchError("target region is required");
  exactKeys(value.target, ["region_id", "outline_node_id", "design_frame_id", "design_node_ids", "component_identity", "bounds"], "target");
  id(value.target.region_id, "target.region_id");
  id(value.target.outline_node_id, "target.outline_node_id");
  if (!Array.isArray(value.target.design_node_ids) || value.target.design_node_ids.length === 0 ||
      !value.target.design_node_ids.every((nodeId) => typeof nodeId === "string" && ID.test(nodeId)) ||
      new Set(value.target.design_node_ids).size !== value.target.design_node_ids.length) {
    throw new CaptureOutlinePatchError("target.design_node_ids must be a unique non-empty mapped node set");
  }
  if (value.target.component_identity !== undefined &&
      (typeof value.target.component_identity !== "string" || !COMPONENT_ID.test(value.target.component_identity) ||
       value.target.component_identity.split("/").some((part) => part === "" || part === "." || part === ".."))) {
    throw new CaptureOutlinePatchError("target.component_identity must be a stable component reference");
  }
  const bounds = value.target.bounds;
  if (!record(bounds)) {
    throw new CaptureOutlinePatchError("target.bounds must contain finite normalized coordinates");
  }
  exactKeys(bounds, ["x", "y", "width", "height"], "target.bounds");
  const x = bounds.x;
  const y = bounds.y;
  const width = bounds.width;
  const height = bounds.height;
  if ([x, y, width, height].some((coordinate) => typeof coordinate !== "number" || !Number.isFinite(coordinate))) {
    throw new CaptureOutlinePatchError("target.bounds must contain finite normalized coordinates");
  }
  if (typeof x !== "number" || typeof y !== "number" || typeof width !== "number" || typeof height !== "number") {
    throw new CaptureOutlinePatchError("target.bounds must contain finite normalized coordinates");
  }
  if (x < 0 || y < 0 || width <= 0 || height <= 0 || x + width > 1 || y + height > 1) {
    throw new CaptureOutlinePatchError("target.bounds must fit inside the captured image");
  }

  if (!record(value.composition)) throw new CaptureOutlinePatchError("composition revision is required");
  exactKeys(value.composition, ["design_id", "base_revision", "operation_id", "summary", "operations"], "composition");
  id(value.composition.design_id, "composition.design_id");
  id(value.composition.operation_id, "composition.operation_id");
  if (!Number.isSafeInteger(value.composition.base_revision) || Number(value.composition.base_revision) < 0) {
    throw new CaptureOutlinePatchError("composition.base_revision must be a non-negative integer");
  }
  if (typeof value.composition.summary !== "string" || !value.composition.summary.trim() || value.composition.summary.length > 500) {
    throw new CaptureOutlinePatchError("composition.summary must be 1–500 characters");
  }
  const operations = value.composition.operations;
  if (!Array.isArray(operations) || operations.length === 0 || operations.length > MAX_OPERATIONS) {
    throw new CaptureOutlinePatchError(`composition.operations must contain 1–${MAX_OPERATIONS} edits`);
  }
  id(value.target.design_frame_id, "target.design_frame_id");
  const targetIds = new Set(value.target.design_node_ids);
  for (const [index, operation] of operations.entries()) {
    if (!record(operation) || !["setText", "setProp", "addLayer", "removeLayer", "moveLayer"].includes(String(operation.kind))) {
      throw new CaptureOutlinePatchError(`composition.operations[${index}] is not a region-scoped edit`);
    }
    const allowedByKind: Record<string, string[]> = {
      setText: ["kind", "nodeId", "text"],
      setProp: ["kind", "nodeId", "key", "value"],
      addLayer: ["kind", "parentId", "node"],
      removeLayer: ["kind", "nodeId"],
      moveLayer: ["kind", "nodeId", "parentId", "index"],
    };
    exactKeys(operation, allowedByKind[String(operation.kind)] ?? [], `composition.operations[${index}]`);
    const touchedIds = [operation.nodeId, operation.parentId].filter((nodeId): nodeId is string => typeof nodeId === "string");
    if (touchedIds.length === 0 || touchedIds.some((nodeId) => !targetIds.has(nodeId))) {
      throw new CaptureOutlinePatchError(`composition.operations[${index}] escapes the selected region`);
    }
  }
}

/** Apply only when the exact composition revision exported with the patch is still open. */
export function applyCaptureOutlinePatch(doc: DesignDocument, value: unknown): AppliedPatch {
  validateCaptureOutlinePatch(value);
  if (value.composition.design_id !== doc.id || value.composition.base_revision !== doc.revision) {
    throw new CaptureOutlinePatchError("composition changed since this patch was created");
  }
  const frame = doc.frames.find((candidate) => candidate.id === value.target.design_frame_id);
  if (!frame) throw new CaptureOutlinePatchError("target design frame is not present in the open composition");
  const frameNodeIds = new Set<string>();
  const collect = (node: TemplateNode) => {
    frameNodeIds.add(node.id);
    for (const child of node.children ?? []) collect(child);
  };
  collect(frame.spec);
  if (value.target.design_node_ids.some((nodeId) => !frameNodeIds.has(nodeId))) {
    throw new CaptureOutlinePatchError("mapped design nodes must belong to the named frame");
  }
  const otherFrameNodeIds = new Set<string>();
  for (const candidate of doc.frames) {
    if (candidate.id === frame.id) continue;
    const collectOther = (node: TemplateNode) => {
      otherFrameNodeIds.add(node.id);
      for (const child of node.children ?? []) collectOther(child);
    };
    collectOther(candidate.spec);
  }
  if (value.target.design_node_ids.some((nodeId) => otherFrameNodeIds.has(nodeId))) {
    throw new CaptureOutlinePatchError("mapped design nodes must not also resolve in another frame");
  }
  return applyPatch(doc, value.composition.operations);
}
