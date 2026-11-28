import type { AskAttachment } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";
import { AskAttachment_Kind } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

export const REGION_FEEDBACK_SCHEMA = "ux-region-feedback/v1";
export const REGION_FEEDBACK_MANIFEST = "ux-region-feedback.json";
export const REGION_NOTE_DIRECTORY = "regions";

export type FidelityLabel = "captured" | "composited" | "rendered" | "sketch";

export interface CaptureRef {
  id: string;
  build_id: string;
  route: string;
  state_id: string;
  data_id: string;
  fidelity: FidelityLabel;
  viewport: { width: number; height: number };
  image: { path: string; sha256: string };
}

export interface FeedbackRegion {
  id: string;
  node_id: string;
  label: string;
  component: string;
  evidence_id: string;
  crop: { path: string; sha256: string };
  bounds: { x: number; y: number; width: number; height: number };
}

export interface RegionFeedbackBundle {
  schema: typeof REGION_FEEDBACK_SCHEMA;
  id: string;
  route: string;
  state_id: string;
  data_id: string;
  before: CaptureRef;
  after: CaptureRef;
  regions: FeedbackRegion[];
}

export interface RegionCorrectionEvent {
  schema: typeof REGION_FEEDBACK_SCHEMA;
  kind: "correction";
  correction_id: string;
  origin: { kind: "hub-packet"; id: string; attachment: string };
  owner_scenario: string;
  corpus: string;
  storage: "local";
  private: true;
  text: string;
  capture: CaptureRef;
  region: FeedbackRegion;
}

export interface ParsedRegionFeedback {
  bundle: RegionFeedbackBundle;
  issues: RegionFeedbackIssue[];
}

export type RegionFeedbackIssue =
  | { code: "routeMismatch" | "stateMismatch" | "dataMismatch" | "viewportMismatch" | "sameBuild" }
  | { code: "imageMismatch"; fidelity: FidelityLabel }
  | { code: "regionEntryMissing" | "regionCropMismatch"; regionId: string };

const record = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

const text = (value: unknown): value is string => typeof value === "string" && value.trim().length > 0;
const fidelity = (value: unknown): value is FidelityLabel =>
  value === "captured" || value === "composited" || value === "rendered" || value === "sketch";
const bundlePath = (value: unknown): value is string => typeof value === "string" && /^[A-Za-z0-9._~!$&'()*+,;=@%/-]+$/.test(value) && value.split("/").every((part) => part !== "" && part !== "." && part !== "..");
const buildIdentity = (value: unknown): value is string => typeof value === "string" && /^sha256:[0-9a-f]{64}$/.test(value);
const artifactDigest = (value: unknown): value is string => typeof value === "string" && /^[0-9a-f]{64}$/.test(value);

function capture(value: unknown): CaptureRef | null {
  if (!record(value) || !record(value.viewport) || !record(value.image)) return null;
  const viewport = value.viewport;
  const image = value.image;
  if (!text(value.id) || !buildIdentity(value.build_id) || !text(value.route) || !text(value.state_id) || !text(value.data_id)) return null;
  if (!fidelity(value.fidelity) || !Number.isInteger(viewport.width) || !Number.isInteger(viewport.height) || Number(viewport.width) <= 0 || Number(viewport.height) <= 0) return null;
  if (!bundlePath(image.path) || !artifactDigest(image.sha256)) return null;
  return {
    id: value.id,
    build_id: value.build_id,
    route: value.route,
    state_id: value.state_id,
    data_id: value.data_id,
    fidelity: value.fidelity,
    viewport: { width: Number(viewport.width), height: Number(viewport.height) },
    image: { path: image.path, sha256: image.sha256 },
  };
}

function region(value: unknown): FeedbackRegion | null {
  if (!record(value) || !record(value.bounds) || !record(value.crop)) return null;
  const bounds = value.bounds;
  const crop = value.crop;
  const numbers = [bounds.x, bounds.y, bounds.width, bounds.height];
  if (numbers.some((item) => typeof item !== "number" || !Number.isFinite(item))) return null;
  const x = bounds.x as number;
  const y = bounds.y as number;
  const width = bounds.width as number;
  const height = bounds.height as number;
  if (x < 0 || y < 0 || width <= 0 || height <= 0 || x + width > 1 || y + height > 1) return null;
  if (!text(value.id) || !text(value.node_id) || !text(value.label) || !text(value.component) || !text(value.evidence_id) || !bundlePath(crop.path) || !artifactDigest(crop.sha256)) return null;
  return { id: value.id, node_id: value.node_id, label: value.label, component: value.component, evidence_id: value.evidence_id, crop: { path: crop.path, sha256: crop.sha256 }, bounds: { x, y, width, height } };
}

/** Validate bundle shape and mark fidelity mismatches without upgrading weaker evidence. */
export function parseRegionFeedbackBundle(raw: unknown): ParsedRegionFeedback | null {
  if (!record(raw) || raw.schema !== REGION_FEEDBACK_SCHEMA || (raw.kind !== undefined && raw.kind !== "bundle") || !text(raw.id) || !text(raw.route) || !text(raw.state_id) || !text(raw.data_id)) return null;
  const before = capture(raw.before);
  const after = capture(raw.after);
  if (!before || !after || !Array.isArray(raw.regions)) return null;
  const regions = raw.regions.map(region);
  if (regions.some((item) => item === null)) return null;
  const cleanRegions = regions as FeedbackRegion[];
  const ids = new Set(cleanRegions.map((item) => item.id));
  if (ids.size !== cleanRegions.length) return null;
  const bundle: RegionFeedbackBundle = { schema: REGION_FEEDBACK_SCHEMA, id: raw.id, route: raw.route, state_id: raw.state_id, data_id: raw.data_id, before, after, regions: cleanRegions };
  const issues: RegionFeedbackIssue[] = [];
  if (before.route !== after.route || before.route !== bundle.route) issues.push({ code: "routeMismatch" });
  if (before.state_id !== after.state_id || before.state_id !== bundle.state_id) issues.push({ code: "stateMismatch" });
  if (before.data_id !== after.data_id || before.data_id !== bundle.data_id) issues.push({ code: "dataMismatch" });
  if (before.viewport.width !== after.viewport.width || before.viewport.height !== after.viewport.height) issues.push({ code: "viewportMismatch" });
  if (before.build_id === after.build_id) issues.push({ code: "sameBuild" });
  return { bundle, issues };
}

/** Region feedback uses an attached folder so an answer can store the real entry path. */
export function isRegionFeedbackFolder(attachment: AskAttachment): boolean {
  return attachment.kind === AskAttachment_Kind.FOLDER && attachment.entries.some((entry) => entry.path === REGION_FEEDBACK_MANIFEST && entry.mediaType === "application/json");
}

export function regionNotePath(regionId: string): string {
  return `${REGION_NOTE_DIRECTORY}/${encodeURIComponent(regionId)}.json`;
}

/** The folder must carry one exact placeholder entry per selectable region for Hub note validation. */
export function regionEntryPath(regionId: string): string {
  return `${REGION_NOTE_DIRECTORY}/${encodeURIComponent(regionId)}.json`;
}

export function validateRegionEntries(bundle: RegionFeedbackBundle, attachment: AskAttachment): RegionFeedbackIssue[] {
  const paths = new Set(attachment.entries.map((entry) => entry.path));
  const issues: RegionFeedbackIssue[] = [];
  for (const capture of [bundle.before, bundle.after]) {
    const entry = attachment.entries.find((item) => item.path === capture.image.path);
    if (!entry || !["image/png", "image/jpeg", "image/webp"].includes(entry.mediaType) || entry.sha256 !== capture.image.sha256 || entry.sizeBytes > 20n * 1024n * 1024n) issues.push({ code: "imageMismatch", fidelity: capture.fidelity });
  }
  for (const region of bundle.regions) {
    if (!paths.has(regionEntryPath(region.id))) issues.push({ code: "regionEntryMissing", regionId: region.id });
    const crop = attachment.entries.find((item) => item.path === region.crop.path);
    if (!crop || !["image/png", "image/jpeg", "image/webp"].includes(crop.mediaType) || crop.sha256 !== region.crop.sha256 || crop.sizeBytes > 5n * 1024n * 1024n) issues.push({ code: "regionCropMismatch", regionId: region.id });
  }
  return issues;
}

/** Convert one exact Hub attachment note into the shared local correction contract. */
export function regionCorrectionFromPacket(input: {
  packetId: string;
  attachmentName: string;
  regionPath: string;
  text: string;
  ownerScenario: string;
  corpus: string;
  correctionId: string;
  bundle: RegionFeedbackBundle;
}): RegionCorrectionEvent | null {
  const stable = (value: string) => /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(value);
  if (![input.packetId, input.attachmentName, input.ownerScenario, input.corpus, input.correctionId].every(stable)) return null;
  const text = input.text.trim();
  if (!text || text.length > 500 || /[\r\n]/.test(text)) return null;
  if (!input.regionPath.startsWith(`${REGION_NOTE_DIRECTORY}/`) || !input.regionPath.endsWith(".json")) return null;
  let encodedRegionId = input.regionPath.slice(REGION_NOTE_DIRECTORY.length + 1, -5);
  try { encodedRegionId = decodeURIComponent(encodedRegionId); } catch { return null; }
  const region = input.bundle.regions.find((item) => item.id === encodedRegionId);
  if (!region || input.regionPath !== regionEntryPath(region.id)) return null;
  return {
    schema: REGION_FEEDBACK_SCHEMA,
    kind: "correction",
    correction_id: input.correctionId,
    origin: { kind: "hub-packet", id: input.packetId, attachment: input.attachmentName },
    owner_scenario: input.ownerScenario,
    corpus: input.corpus,
    storage: "local",
    private: true,
    text,
    capture: input.bundle.after,
    region,
  };
}
