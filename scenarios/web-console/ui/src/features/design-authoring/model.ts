export const DESIGN_PACKET_SCHEMA = "web-console.design-packet/v2" as const;
export const LEGACY_DESIGN_PACKET_SCHEMA = "web-console.design-packet/v1" as const;
export const TRACEABILITY_CONTRACT_SCHEMA = "web-console.design-trace/v1" as const;

export type OmissionDisposition = "preserve" | "remove" | "defer" | "unknown";
export type CoverageKind = "visual" | "nonvisual" | "unknown";

export interface PacketSource {
  id: string;
  name: string;
  origin: string;
  version: string;
  sha256: string;
  bytes: number;
  mediaType: string;
  use: string;
  amendmentHistory: string[];
  /** Exact descriptors bound by id for synchronous stale-origin/version detection. */
  identityBinding: string;
}

export interface SourceReimportMatch {
  archivedSourceId: string;
  reimportedSourceId: string;
  identityBinding: string;
}

export interface SourceReimportResult {
  valid: boolean;
  matches: SourceReimportMatch[];
  issues: PacketValidationIssue[];
}

export interface RequirementMapping {
  pageId: string | null;
  stateId: string | null;
  region: string | null;
  rationale: string;
  sourceId?: string | null;
}

export interface RequirementSceneTrace {
  pageId: string;
  stateIds: string[];
  obligationId: string;
  sourceMapSourceId: string;
  manifestSourceId: string;
  sourceIds: string[];
  editableSourceIds: string[];
  sourceImages: Array<{ filename: string; sha256: string; sourceId: string; stateIds: string[] }>;
}

export interface PacketRequirement {
  id: string;
  text: string;
  scope: string;
  coverage: CoverageKind;
  mapping: RequirementMapping;
  sourceRefs: string[];
  evidenceRefs: string[];
  disposition: OmissionDisposition;
  /** Exact source-map scene/state and image membership joins for every related Appendix A row. */
  sourceSceneTrace?: RequirementSceneTrace[];
  /** Verbatim source-map selected-state and proof-surface/verification descriptions. */
  sourceStateText?: string;
  proofBasisText?: string;
  /** The required behavior was not verified by authoring or design evidence. */
  behaviorProofStatus?: "not-verified";
  /** Exact source IDs from the crosswalk cell, retained even when they do not resolve to a packet row. */
  sourceRequirementIds?: string[];
  sourceMapRefs?: string[];
}

export interface PacketFeature {
  id: string;
  name: string;
  sourceRef: string;
  requirementIds: string[];
  sourceRequirementIds?: string[];
  unresolvedSourceRequirementIds?: string[];
  pageId?: string;
  stateIds?: string[];
  relatedSourceIds?: string[];
  sourceImages?: Array<{ filename: string; sha256: string; sourceId: string; stateIds: string[] }>;
  disposition: OmissionDisposition;
  rationale: string;
}

export interface PacketPage {
  id: string;
  title: string;
  stateIds: string[];
  sourceRefs: string[];
  regions: string[];
  /** Source-authored layout regions; distinct from manifest filenames in `regions`. */
  designRegions?: string[];
  editableSourceIds: string[];
}

export interface OperatorDecisionReference {
  authority: "operator";
  id: string;
  source: string;
  sourceTimestamp: string;
  quote: string;
  appliesToVersion: string;
  affectedPageIds?: string[];
  scopeDescription?: string;
}

export interface PacketRevision {
  id: string;
  parentId: string | null;
  createdAt: string;
  title: string;
  sourceIds: string[];
  pages: PacketPage[];
  requirements: PacketRequirement[];
  features: PacketFeature[];
  decisionRefs: OperatorDecisionReference[];
  changeSummary: string;
  contentSha256: string;
}

export interface DesignPacket {
  schema: typeof DESIGN_PACKET_SCHEMA | typeof LEGACY_DESIGN_PACKET_SCHEMA;
  traceabilitySchema?: typeof TRACEABILITY_CONTRACT_SCHEMA;
  id: string;
  scenario: string;
  brief: string;
  sources: PacketSource[];
  revisions: PacketRevision[];
  currentRevisionId: string;
}

export interface RequirementSourceRow {
  id: string;
  text: string;
  scope: string;
  selectedStates: string[];
  selectedStateText: string;
  proofBasisText: string;
  sourceRefs: string[];
  sourceMapRefs: string[];
  evidenceRefs: string[];
  coverage: CoverageKind;
}

export interface SelectedDesignAsset {
  filename: string;
  sceneId: string;
  stateIds: string[];
  sha256: string;
  bytes: number;
  sourceRefs: string[];
  caption: string;
  mediaType: string;
}

export interface SceneSourceRow {
  sceneId: string;
  stateIds: string[];
  filenames: string[];
  sourceHeadings: string[];
  obligations: string;
  requirementIds: string[];
}

export interface PacketValidationIssue {
  path: string;
  message: string;
}

export interface PacketValidation {
  valid: boolean;
  issues: PacketValidationIssue[];
  mappedRequirementCount: number;
  explicitlyDisposedFeatureCount: number;
}

export interface NewPacketInput {
  id: string;
  scenario: string;
  brief: string;
  sources: PacketSource[];
  pages: PacketPage[];
  requirements: PacketRequirement[];
  features: PacketFeature[];
  createdAt: string;
  traceabilitySchema?: typeof TRACEABILITY_CONTRACT_SCHEMA;
}

export interface NewRevisionInput {
  revisionId: string;
  createdAt: string;
  changeSummary: string;
  pages?: PacketPage[];
  requirements?: PacketRequirement[];
  features?: PacketFeature[];
  sourceIds?: string[];
  decisionRefs?: OperatorDecisionReference[];
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

/** Stable JSON is used for packet identities and portable manifests. */
export function canonicalJson(value: unknown): string {
  if (Array.isArray(value)) return `[${value.map(canonicalJson).join(",")}]`;
  if (value && typeof value === "object") {
    const record = value as Record<string, unknown>;
    return `{${Object.keys(record).sort().map((key) => `${JSON.stringify(key)}:${canonicalJson(record[key])}`).join(",")}}`;
  }
  return JSON.stringify(value) ?? "null";
}

function sourceIdentityDescriptor(source: Pick<PacketSource, "name" | "origin" | "version" | "sha256" | "bytes" | "mediaType">): string {
  return canonicalJson({ name: source.name, origin: source.origin, version: source.version, sha256: source.sha256, bytes: source.bytes, mediaType: source.mediaType });
}

async function sourceIdentityId(source: Pick<PacketSource, "sha256">, identityBinding: string): Promise<string> {
  const metadataHash = await sha256Hex(identityBinding);
  return `src-${source.sha256.slice(0, 16)}-${metadataHash.slice(0, 12)}`;
}

/**
 * Reconcile a legacy packet's source rows against freshly imported byte-bound
 * records. Legacy rows may lack identityBinding, but their recorded descriptor
 * must match exactly one verified current record. Neither input is mutated.
 */
export async function verifySourceReimportIdentity(
  archivedSources: PacketSource[],
  reimportedSources: PacketSource[],
  options: { allowAdditionalImportedSources?: boolean } = {},
): Promise<SourceReimportResult> {
  const issues: PacketValidationIssue[] = [];
  const matches: SourceReimportMatch[] = [];
  const archivedIds = new Set<string>();
  const importedIds = new Set<string>();
  const archivedDescriptors = new Set<string>();
  const importedDescriptors = new Set<string>();
  if (!archivedSources.length || !reimportedSources.length) issues.push({ path: "sources", message: "Source re-import requires non-empty archived and current identity sets." });

  const inspect = async (source: PacketSource, path: string, isImported: boolean) => {
    if (!source || typeof source !== "object") {
      issues.push({ path, message: "Source identity row is missing or malformed." });
      return null;
    }
    if (typeof source.id !== "string" || typeof source.name !== "string" || typeof source.origin !== "string" || typeof source.version !== "string" || typeof source.sha256 !== "string" || typeof source.mediaType !== "string" || typeof source.bytes !== "number" ||
      !/^[a-f0-9]{64}$/i.test(source.sha256) || !Number.isSafeInteger(source.bytes) || source.bytes < 0 || !source.name.trim() || !source.origin.trim() || !source.version.trim() || !source.mediaType.trim()) {
      issues.push({ path, message: "Source identity descriptor is incomplete or malformed." });
      return null;
    }
    const descriptor = sourceIdentityDescriptor(source);
    if (source.identityBinding && source.identityBinding !== descriptor) {
      issues.push({ path: `${path}.identityBinding`, message: "Source identity binding is stale or does not match its descriptor." });
    }
    if (isImported) {
      if (source.identityBinding !== descriptor) issues.push({ path: `${path}.identityBinding`, message: "Re-imported source must carry its exact identity binding." });
      if (source.id !== await sourceIdentityId(source, descriptor)) issues.push({ path: `${path}.id`, message: "Re-imported source ID does not bind its exact identity descriptor." });
    } else if (source.identityBinding) {
      if (source.id !== await sourceIdentityId(source, descriptor)) issues.push({ path: `${path}.id`, message: "Archived source ID does not bind its current identity descriptor." });
    } else if (source.id !== `src-${source.sha256.slice(0, 16)}`) {
      issues.push({ path: `${path}.id`, message: "Unbound legacy source ID is not the recognized content-only form." });
    }
    return descriptor;
  };

  const archivedRows = await Promise.all(archivedSources.map(async (source, index) => ({ source, index, descriptor: await inspect(source, `archivedSources[${index}]`, false) })));
  const importedRows = await Promise.all(reimportedSources.map(async (source, index) => ({ source, index, descriptor: await inspect(source, `reimportedSources[${index}]`, true) })));
  for (const { source, index, descriptor } of archivedRows) {
    if (!source || typeof source !== "object") continue;
    if (archivedIds.has(source.id)) issues.push({ path: `archivedSources[${index}].id`, message: "Duplicate archived source ID makes source lineage ambiguous." });
    archivedIds.add(source.id);
    if (descriptor && archivedDescriptors.has(descriptor)) issues.push({ path: `archivedSources[${index}]`, message: "Duplicate archived source descriptor makes re-import ambiguous." });
    if (descriptor) archivedDescriptors.add(descriptor);
  }
  for (const { source, index, descriptor } of importedRows) {
    if (!source || typeof source !== "object") continue;
    if (importedIds.has(source.id)) issues.push({ path: `reimportedSources[${index}].id`, message: "Duplicate re-imported source ID makes source lineage ambiguous." });
    importedIds.add(source.id);
    if (descriptor && importedDescriptors.has(descriptor)) issues.push({ path: `reimportedSources[${index}]`, message: "Duplicate re-imported source descriptor makes source lineage ambiguous." });
    if (descriptor) importedDescriptors.add(descriptor);
  }

  if (options.allowAdditionalImportedSources ? archivedRows.length > importedRows.length : archivedRows.length !== importedRows.length) issues.push({ path: "sources", message: "Re-import must account for every archived source exactly once." });
  const claimed = new Set<number>();
  for (const { source, index, descriptor } of archivedRows) {
    if (!source || typeof source !== "object" || !descriptor) continue;
    const candidates = importedRows.filter((row) => row.descriptor === descriptor);
    if (candidates.length !== 1) {
      issues.push({ path: `archivedSources[${index}]`, message: candidates.length ? "Archived source matches multiple re-imported identities." : "No re-imported source matches the archived name, origin, version, bytes and media type." });
      continue;
    }
    const candidate = candidates[0]!;
    if (claimed.has(candidate.index)) {
      issues.push({ path: `reimportedSources[${candidate.index}]`, message: "Re-imported source was joined to more than one archived identity." });
      continue;
    }
    claimed.add(candidate.index);
    matches.push({ archivedSourceId: source.id, reimportedSourceId: candidate.source.id, identityBinding: descriptor });
  }
  const additionalRows = importedRows.filter((row) => !claimed.has(row.index));
  if (options.allowAdditionalImportedSources) {
    for (const { source, index } of additionalRows) {
      if (archivedRows.some((row) => row.source.name === source.name || row.source.origin === source.origin)) {
        issues.push({ path: `reimportedSources[${index}]`, message: "Additional source reuses an archived name or origin without matching its exact identity." });
      }
    }
  }
  return { valid: issues.length === 0 && matches.length === archivedRows.length && (options.allowAdditionalImportedSources || matches.length === importedRows.length), matches, issues };
}

/** Stable packet feature key for one Appendix A source-scene obligation. */
export function sourceSceneObligationId(sceneId: string): string {
  const normalized = sceneId.trim();
  if (normalized !== sceneId || !/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(normalized)) throw new Error("Source scene ID must be a non-empty stable token.");
  return `source-scene-${normalized}`;
}

/** Resolve a source-scene feature key without guessing or splitting on scene punctuation. */
export function sourceSceneIdFromObligationId(obligationId: string): string | null {
  if (!obligationId.startsWith("source-scene-")) return null;
  const sceneId = obligationId.slice("source-scene-".length);
  return /^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(sceneId) ? sceneId : null;
}

export type SelectedStateResolution =
  | { status: "resolved"; sceneId: string; stateId: string; assets: SelectedDesignAsset[] }
  | { status: "missing"; reference: string }
  | { status: "ambiguous"; reference: string; candidates: Array<{ sceneId: string; stateId: string }> };

/** Resolve a source-state reference only when it identifies one logical scene/state. */
export function resolveSelectedState(reference: string, assets: SelectedDesignAsset[], sceneId?: string): SelectedStateResolution {
  const matches = assets.filter((asset) => (!sceneId || asset.sceneId === sceneId) && matchingAssetStates(reference, [asset]).length > 0);
  const logicalStates = new Map<string, { sceneId: string; stateId: string; assets: SelectedDesignAsset[] }>();
  for (const asset of matches) {
    for (const stateId of asset.stateIds) {
      if (!(reference.endsWith("*") ? stateId.startsWith(reference.slice(0, -1)) : stateId === reference)) continue;
      const key = `${asset.sceneId}\u0000${stateId}`;
      const entry = logicalStates.get(key) ?? { sceneId: asset.sceneId, stateId, assets: [] };
      entry.assets.push(asset);
      logicalStates.set(key, entry);
    }
  }
  const candidates = [...logicalStates.values()];
  if (!candidates.length) return { status: "missing", reference };
  if (candidates.length > 1) return { status: "ambiguous", reference, candidates: candidates.map(({ sceneId: candidateScene, stateId }) => ({ sceneId: candidateScene, stateId })) };
  const resolved = candidates[0]!;
  return { status: "resolved", sceneId: resolved.sceneId, stateId: resolved.stateId, assets: resolved.assets };
}

export async function sha256Hex(bytes: ArrayBuffer | Uint8Array | string): Promise<string> {
  const data = typeof bytes === "string"
    ? new TextEncoder().encode(bytes)
    : bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  // Copy to a concrete ArrayBuffer for TS's newer BufferSource typing (the
  // input may be backed by SharedArrayBuffer when supplied by a caller).
  const digest = await globalThis.crypto.subtle.digest("SHA-256", data.slice().buffer as ArrayBuffer);
  return Array.from(new Uint8Array(digest), (part) => part.toString(16).padStart(2, "0")).join("");
}

export async function createSourceRecord(input: {
  name: string;
  origin: string;
  version: string;
  bytes: ArrayBuffer | Uint8Array | string;
  mediaType: string;
  use: string;
  amendmentHistory?: string[];
}): Promise<PacketSource> {
  const byteLength = typeof input.bytes === "string"
    ? new TextEncoder().encode(input.bytes).byteLength
    : input.bytes.byteLength;
  const sha256 = await sha256Hex(input.bytes);
  const identityBinding = canonicalJson({ name: input.name, origin: input.origin, version: input.version, sha256, bytes: byteLength, mediaType: input.mediaType });
  const metadataHash = await sha256Hex(identityBinding);
  return {
    id: `src-${sha256.slice(0, 16)}-${metadataHash.slice(0, 12)}`,
    name: input.name,
    origin: input.origin,
    version: input.version,
    sha256,
    bytes: byteLength,
    mediaType: input.mediaType,
    use: input.use,
    amendmentHistory: [...(input.amendmentHistory ?? [])],
    identityBinding,
  };
}

function revisionContent(revision: Omit<PacketRevision, "contentSha256">): string {
  return canonicalJson(revision);
}

export async function createDesignPacket(input: NewPacketInput): Promise<DesignPacket> {
  const seed: Omit<PacketRevision, "contentSha256"> = {
    id: "v1",
    parentId: null,
    createdAt: input.createdAt,
    title: `${input.scenario} design packet`,
    sourceIds: input.sources.map((source) => source.id),
    pages: clone(input.pages),
    requirements: clone(input.requirements),
    features: clone(input.features),
    decisionRefs: [],
    changeSummary: "Initial imported design packet; source decisions and product acceptance remain separate.",
  };
  const contentSha256 = await sha256Hex(revisionContent(seed));
  return {
    schema: DESIGN_PACKET_SCHEMA,
    ...(input.traceabilitySchema ? { traceabilitySchema: input.traceabilitySchema } : {}),
    id: input.id,
    scenario: input.scenario,
    brief: input.brief,
    sources: clone(input.sources),
    revisions: [{ ...seed, contentSha256 }],
    currentRevisionId: seed.id,
  };
}

/** Add a new immutable snapshot; the previous version and its decisions remain unchanged. */
export async function addPacketRevision(packet: DesignPacket, input: NewRevisionInput): Promise<DesignPacket> {
  const parent = packet.revisions.find((revision) => revision.id === packet.currentRevisionId);
  if (!parent) throw new Error("The current packet revision is missing.");
  if (packet.revisions.some((revision) => revision.id === input.revisionId)) throw new Error("Revision ids must be unique.");

  const nextSourceIds = input.sourceIds ?? parent.sourceIds;
  const nextRequirementIds = (input.requirements ?? parent.requirements).map((item) => item.id);
  const nextFeatureIds = (input.features ?? parent.features).map((item) => item.id);
  if (new Set(nextSourceIds).size !== nextSourceIds.length) throw new Error("Revision source IDs must be unique.");
  if (new Set(nextRequirementIds).size !== nextRequirementIds.length) throw new Error("Revision requirement IDs must be unique.");
  if (new Set(nextFeatureIds).size !== nextFeatureIds.length) throw new Error("Revision obligation IDs must be unique.");
  const availableSourceIds = new Set(packet.sources.map((source) => source.id));
  if (!nextSourceIds.every((id) => availableSourceIds.has(id))) throw new Error("Revision references an unknown source identity.");
  if (!parent.sourceIds.every((id) => nextSourceIds.includes(id))) throw new Error("A revision cannot discard a source lineage record.");
  if (!parent.requirements.every((item) => nextRequirementIds.includes(item.id))) throw new Error("A revision cannot discard a source requirement ID; record its disposition instead.");
  if (!parent.features.every((item) => nextFeatureIds.includes(item.id))) throw new Error("A revision cannot discard a source-scene obligation ID; record its disposition instead.");

  const revisionSeed: Omit<PacketRevision, "contentSha256"> = {
    id: input.revisionId,
    parentId: parent.id,
    createdAt: input.createdAt,
    title: parent.title,
    sourceIds: [...nextSourceIds],
    pages: clone(input.pages ?? parent.pages),
    requirements: clone(input.requirements ?? parent.requirements),
    features: clone(input.features ?? parent.features),
    decisionRefs: clone(input.decisionRefs ?? parent.decisionRefs),
    changeSummary: input.changeSummary,
  };
  const allowedDecisionVersions = new Set<string>();
  let ancestorId: string | null = parent.id;
  while (ancestorId && !allowedDecisionVersions.has(ancestorId)) {
    allowedDecisionVersions.add(ancestorId);
    ancestorId = packet.revisions.find((revision) => revision.id === ancestorId)?.parentId ?? null;
  }
  for (const decision of revisionSeed.decisionRefs) {
    if (!allowedDecisionVersions.has(decision.appliesToVersion)) throw new Error(`Operator decision ${decision.id} must be bound to an ancestor packet revision.`);
  }
  const contentSha256 = await sha256Hex(revisionContent(revisionSeed));
  return {
    ...clone(packet),
    revisions: [...packet.revisions, { ...revisionSeed, contentSha256 }],
    currentRevisionId: revisionSeed.id,
  };
}

export function parseRequirementSourceMap(markdown: string): RequirementSourceRow[] {
  const lines = markdown.split(/\r?\n/);
  const heading = lines.findIndex((line) => /^\s*##\s+Requirement crosswalk\s*$/i.test(line));
  if (heading < 0) return [];
  const rows: RequirementSourceRow[] = [];
  let inTable = false;

  for (const line of lines.slice(heading + 1)) {
    if (/^\s*##\s+/.test(line)) break;
    if (!line.includes("|") || /^\s*\|\s*:?-{2,}/.test(line)) continue;
    const cells = splitMarkdownRow(line);
    if (!inTable) {
      if (cells[0]?.toLowerCase().includes("requirement")) inTable = true;
      continue;
    }
    if (cells.length < 5) continue;
    const ids = requirementIdMatches(cells[0] ?? "");
    if (ids.length === 0) continue;
    const selectedStateText = stripMarkup(cells[2] ?? "");
    const selectedStates = parseStateReferences(selectedStateText);
    const sourceRefs = uniqueMatches(cells[3] ?? "", /\b(?:NH[ -]?(?:AC|DEC)[ -]?\d{2,3}|NH\s*\d{2}\s+—\s+[^;<]+|Current contracts\s*\/\s*[^;<]+|Experience\s*\/\s*[^;<]+|Behavior and migration\s*\/\s*[^;<]+|No [^;<]+)\b/g)
      .map((ref) => ref.replace(/\bNH[ -]+(AC|DEC)[ -]+(\d{2,3})\b/i, (_match, type: string, id: string) => `NH-${type.toUpperCase()}-${id}`));
    const evidenceRefs = uniqueMatches(cells[4] ?? "", /(?:\bapi\/[^\s<;]+|\.vrooli\/[^\s<;]+|\bTest[A-Za-z0-9_]+)/g);
    const noScene = /\bno\s+(?:dedicated\s+|selected\s+|specific\s+|visual\s+|UI\s+)*(?:[\w-]+\s+){0,3}(?:screen|scene|state)(?:\s*\/\s*(?:scene|state))?\b/i.test(cells[2] ?? "");
    rows.push({
      id: ids.join(" / "),
      text: stripMarkup(cells[0] ?? ""),
      scope: stripMarkup(cells[1] ?? ""),
      selectedStates,
      selectedStateText,
      proofBasisText: stripMarkup(cells[4] ?? ""),
      sourceRefs: sourceRefs.map(stripMarkup),
      sourceMapRefs: (cells[3] ?? "").split(/<br\s*\/?\s*>/i).map(stripMarkup).filter(Boolean),
      evidenceRefs: evidenceRefs.map(stripMarkup),
      coverage: noScene ? "nonvisual" : selectedStates.length > 0 ? "visual" : "unknown",
    });
  }
  assertUniqueRequirementRows(rows);
  return rows;
}

/** Requirement identity comes from the crosswalk's first cell; proof/source citations remain references. */
export function sourceRequirementAliases(row: RequirementSourceRow): string[] {
  return [...new Set(row.id.split(/\s*\/\s*/).map((id) => id.trim()).filter(Boolean))];
}

/** Join requirements to Appendix A only through explicit requirement aliases, selected states, or an exact retained mapping. */
export function sourceScenesForRequirement(row: RequirementSourceRow, scenes: SceneSourceRow[], mapped?: { pageId: string | null; stateId: string | null }): SceneSourceRow[] {
  const aliases = new Set(sourceRequirementAliases(row));
  const direct = scenes.filter((scene) => scene.requirementIds.some((id) => aliases.has(id)));
  const selected = scenes.filter((scene) => row.selectedStates.some((reference) => {
    const matches = (candidate: string) => reference.endsWith("*") ? candidate.startsWith(reference.slice(0, -1)) : candidate === reference;
    return matches(scene.sceneId) || scene.stateIds.some(matches);
  }));
  const exactMapping = mapped?.pageId && mapped.stateId
    ? scenes.filter((scene) => scene.sceneId === mapped.pageId && scene.stateIds.includes(mapped.stateId!))
    : [];
  return [...new Map([...direct, ...selected, ...exactMapping].map((scene) => [scene.sceneId, scene])).values()];
}

/** Read the checked-in selected-image index without treating its claims as acceptance. */
export function parseSelectedImageManifest(json: string): SelectedDesignAsset[] {
  const value: unknown = JSON.parse(json);
  if (!Array.isArray(value)) throw new Error("Selected image manifest must be a JSON array.");
  const assets = value.map((row, index) => {
    if (!row || typeof row !== "object") throw new Error(`Manifest row ${index + 1} is not an object.`);
    const item = row as Record<string, unknown>;
    const required = ["filename", "sceneId", "sha256"] as const;
    for (const key of required) if (typeof item[key] !== "string" || !item[key].trim()) throw new Error(`Manifest row ${index + 1} is missing ${key}.`);
    if (!Array.isArray(item.stateIds) || !item.stateIds.length || !item.stateIds.every((id) => typeof id === "string" && Boolean(id.trim()))) throw new Error(`Manifest row ${index + 1} has invalid stateIds.`);
    if (new Set(item.stateIds as string[]).size !== (item.stateIds as string[]).length) throw new Error(`Manifest row ${index + 1} has duplicate state IDs.`);
    if (!Array.isArray(item.sourceRefs) || !item.sourceRefs.every((ref) => typeof ref === "string")) throw new Error(`Manifest row ${index + 1} has invalid sourceRefs.`);
    if (!/^[a-f0-9]{64}$/i.test(item.sha256 as string)) throw new Error(`Manifest row ${index + 1} has an invalid SHA-256.`);
    if (!/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(item.sceneId as string)) throw new Error(`Manifest row ${index + 1} has an invalid sceneId.`);
    if (typeof item.bytes !== "number" || !Number.isSafeInteger(item.bytes) || item.bytes <= 0) throw new Error(`Manifest row ${index + 1} has invalid byte count.`);
    return {
      filename: item.filename as string,
      sceneId: item.sceneId as string,
      stateIds: item.stateIds as string[],
      sha256: (item.sha256 as string).toLowerCase(),
      bytes: item.bytes,
      sourceRefs: item.sourceRefs as string[],
      caption: typeof item.caption === "string" ? item.caption : "",
      mediaType: "image/png",
    };
  });
  const filenames = new Set<string>();
  for (const asset of assets) {
    const key = asset.filename.trim().toLowerCase();
    if (filenames.has(key)) throw new Error(`Selected image manifest has duplicate filename: ${asset.filename}`);
    filenames.add(key);
  }
  return assets;
}

/** Expand a source-map state family against the exact selected manifest. */
export function matchingAssetStates(reference: string, assets: SelectedDesignAsset[]): SelectedDesignAsset[] {
  const pattern = reference.endsWith("*") ? reference.slice(0, -1) : reference;
  return assets.filter((asset) => asset.stateIds.some((state) => reference.endsWith("*") ? state.startsWith(pattern) : state === reference));
}

/** Parse Appendix A's explicit scene obligation row, retaining each selected scene. */
export function parseSceneSourceMap(markdown: string): SceneSourceRow[] {
  const lines = markdown.split(/\r?\n/);
  const heading = lines.findIndex((line) => /^\s*##\s+Appendix A\b/i.test(line));
  if (heading < 0) return [];
  const rows: SceneSourceRow[] = [];
  let inTable = false;
  for (const line of lines.slice(heading + 1)) {
    if (/^\s*##\s+/.test(line)) break;
    if (!line.includes("|") || /^\s*\|\s*:?-{2,}/.test(line)) continue;
    const cells = splitMarkdownRow(line);
    if (!inTable) {
      if (/scene family/i.test(cells[0] ?? "")) inTable = true;
      continue;
    }
    if (cells.length < 5 || !cells[0]) continue;
    const sceneId = stripMarkup(cells[0]).replace(/^`|`$/g, "");
    if (!sceneId || /^scene family$/i.test(sceneId)) continue;
    const obligations = stripMarkup(cells[4] ?? "");
    const requirementIds = new Set(requirementIdMatches(obligations));
    for (const match of obligations.matchAll(/\b(NH-AC|NH-DEC)-(\d{2,3}(?:[-–]\d{2,3})?(?:\/\d{2,3}(?:[-–]\d{2,3})?)*)/g)) {
      const requirementType = match[1];
      const references = match[2];
      if (!requirementType || !references) continue;
      for (const part of references.split("/")) {
        const range = part.match(/^(\d{2,3})[-–](\d{2,3})$/);
        if (range) {
          const startToken = range[1]; const endToken = range[2];
          if (!startToken || !endToken) continue;
          const start = Number(startToken); const end = Number(endToken);
          if (end >= start && end - start <= 100) for (let id = start; id <= end; id += 1) requirementIds.add(`${requirementType}-${String(id).padStart(startToken.length, "0")}`);
        } else if (/^\d{2,3}$/.test(part)) requirementIds.add(`${requirementType}-${part}`);
      }
    }
    for (const match of obligations.matchAll(/\b(NOTIFICA|OT)-P([01])-(\d{3})(?:\/P([01])-(\d{3}))?/g)) {
      requirementIds.add(`${match[1]}-P${match[2]}-${match[3]}`);
      if (match[4] && match[5]) requirementIds.add(`${match[1]}-P${match[4]}-${match[5]}`);
    }
    rows.push({
      sceneId,
      stateIds: parseStateReferences(cells[1] ?? ""),
      filenames: uniqueMatches(cells[2] ?? "", /[A-Za-z0-9_.-]+\.png/g),
      sourceHeadings: (cells[3] ?? "").split(/<br\s*\/?\s*>/i).map(stripMarkup).filter(Boolean),
      obligations,
      requirementIds: [...requirementIds],
    });
  }
  const ids = new Set<string>();
  const filenames = new Set<string>();
  for (const row of rows) {
    if (!/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(row.sceneId)) throw new Error(`Appendix A has an invalid scene ID: ${row.sceneId}`);
    if (ids.has(row.sceneId)) throw new Error(`Appendix A has duplicate scene ID: ${row.sceneId}`);
    ids.add(row.sceneId);
    if (new Set(row.stateIds).size !== row.stateIds.length) throw new Error(`Appendix A scene ${row.sceneId} has duplicate state IDs.`);
    if (new Set(row.filenames).size !== row.filenames.length) throw new Error(`Appendix A scene ${row.sceneId} has duplicate manifest filenames.`);
    for (const filename of row.filenames) {
      const key = filename.toLowerCase();
      if (filenames.has(key)) throw new Error(`Appendix A has ambiguous manifest filename: ${filename}`);
      filenames.add(key);
    }
  }
  return rows;
}

/** Fail closed when source-map scene membership and selected-image rows disagree. */
export function validateSceneManifestJoins(scenes: SceneSourceRow[], assets: SelectedDesignAsset[]): PacketValidationIssue[] {
  const issues: PacketValidationIssue[] = [];
  const manifestByName = new Map<string, SelectedDesignAsset>();
  for (const asset of assets) {
    const key = asset.filename.toLowerCase();
    if (manifestByName.has(key)) issues.push({ path: `manifest.${asset.filename}.filename`, message: "Selected-image manifest has a duplicate or ambiguous filename." });
    manifestByName.set(key, asset);
  }
  const sceneByName = new Map<string, SceneSourceRow>();
  for (const scene of scenes) {
    for (const filename of scene.filenames) {
      const key = filename.toLowerCase();
      if (sceneByName.has(key)) issues.push({ path: `scenes.${scene.sceneId}.filenames`, message: `Filename ${filename} is claimed by more than one source scene.` });
      sceneByName.set(key, scene);
      const asset = manifestByName.get(key);
      if (!asset) {
        issues.push({ path: `scenes.${scene.sceneId}.filenames.${filename}`, message: "Source-map filename is missing from the selected-image manifest." });
        continue;
      }
      if (asset.sceneId !== scene.sceneId) issues.push({ path: `manifest.${filename}.sceneId`, message: "Selected-image scene ID does not match its source-map row." });
      if (!scene.stateIds.length || asset.stateIds.some((stateId) => !scene.stateIds.some((reference) => reference.endsWith("*") ? stateId.startsWith(reference.slice(0, -1)) : reference === stateId))) {
        issues.push({ path: `manifest.${filename}.stateIds`, message: "Selected-image state IDs do not resolve uniquely through their source-map scene row." });
      }
    }
  }
  for (const asset of assets) {
    const expectedScene = sceneByName.get(asset.filename.toLowerCase());
    const scene = scenes.find((candidate) => candidate.sceneId === asset.sceneId);
    if (expectedScene && expectedScene.sceneId !== asset.sceneId) continue;
    if (!expectedScene && scene) issues.push({ path: `manifest.${asset.filename}`, message: "Selected-image file is assigned to a scene that does not declare the filename." });
  }
  return issues;
}

function splitMarkdownRow(line: string): string[] {
  return line.trim().replace(/^\|/, "").replace(/\|$/, "").split(/(?<!\\)\|/).map((cell) => cell.trim().replace(/\\\|/g, "|"));
}

function uniqueMatches(value: string, pattern: RegExp): string[] {
  return [...new Set([...value.matchAll(pattern)].map(([match]) => match.trim()).filter(Boolean))];
}

/** Accept stable requirement ids from other scenario source-map packets. */
function requirementIdMatches(value: string): string[] {
  return uniqueMatches(value, /\b(?:NOTIFICA|OT)-P[01]-\d{3}\b|\bNH-(?:AC|DEC)-\d{2,3}\b|\b[A-Z][A-Z0-9]*(?:[-_][A-Z0-9]+)+\b/g);
}

function assertUniqueRequirementRows(rows: RequirementSourceRow[]): void {
  const seen = new Set<string>();
  for (const row of rows) {
    const aliases = row.id.split(/\s*\/\s*/).map((id) => id.trim()).filter(Boolean);
    for (const id of aliases) {
      if (seen.has(id)) throw new Error(`Requirement crosswalk has duplicate or ambiguous ID: ${id}`);
      seen.add(id);
    }
  }
}

function parseStateReferences(value: string): string[] {
  const expanded = value.replace(/\bNH(\d{2})\.(\d+)\s*[–-]\s*NH(\d{2})\.(\d+)\b/gi, (full, startChapter: string, startPart: string, endChapter: string, endPart: string) => {
    if (startChapter !== endChapter) return full;
    const start = Number(startPart);
    const end = Number(endPart);
    if (end < start || end - start > 100) return full;
    return Array.from({ length: end - start + 1 }, (_, index) => `NH${startChapter}.${start + index}`).join(", ");
  });
  return uniqueMatches(expanded, /\b(?:NH\d{2}[-][A-Z0-9*-]+|NH\d{2}(?:\.\d+)?|(?:reach|quiet|integration|settings|preferences|order)-[a-z0-9*-]+|[A-Za-z][A-Za-z0-9]*(?:[-_][A-Za-z0-9*-]+)+)(?=$|[^A-Za-z0-9])/gi);
}

function stripMarkup(value: string): string {
  return value.replace(/<br\s*\/?>/gi, "; ").replace(/`/g, "").replace(/<[^>]+>/g, "").replace(/\s+/g, " ").trim();
}

function sameSorted(left: string[], right: string[]): boolean {
  if (left.length !== right.length) return false;
  const a = [...left].sort();
  const b = [...right].sort();
  return a.every((value, index) => value === b[index]);
}

export function validatePacket(packet: DesignPacket): PacketValidation {
  const issues: PacketValidationIssue[] = [];
  const sourceIds = new Set<string>();
  packet.sources.forEach((source, index) => {
    if (sourceIds.has(source.id)) issues.push({ path: `sources[${index}].id`, message: `Duplicate source ID ${source.id}.` });
    sourceIds.add(source.id);
  });
  const revisionIds = new Set(packet.revisions.map((revision) => revision.id));
  if (revisionIds.size !== packet.revisions.length) issues.push({ path: "revisions", message: "Revision IDs must be unique." });
  const current = packet.revisions.find((revision) => revision.id === packet.currentRevisionId);
  if (packet.schema !== DESIGN_PACKET_SCHEMA && packet.schema !== LEGACY_DESIGN_PACKET_SCHEMA) issues.push({ path: "schema", message: "Unsupported packet schema." });
  if (!packet.scenario.trim()) issues.push({ path: "scenario", message: "Scenario name is required." });
  if (!current) issues.push({ path: "currentRevisionId", message: "Current revision does not exist." });

  for (const source of packet.sources) {
    if (!/^[a-f0-9]{64}$/.test(source.sha256)) issues.push({ path: `sources.${source.id}.sha256`, message: "Source SHA-256 is missing or malformed." });
    if (!new RegExp(`^src-${source.sha256.slice(0, 16)}-[a-f0-9]{12}$`).test(source.id)) issues.push({ path: `sources.${source.id}.id`, message: "Source ID must bind its content and origin/version identity." });
    if (!source.origin.trim()) issues.push({ path: `sources.${source.id}.origin`, message: "Source origin is required." });
    if (!source.version.trim()) issues.push({ path: `sources.${source.id}.version`, message: "Source version is required." });
    if (!source.use.trim()) issues.push({ path: `sources.${source.id}.use`, message: "Source use is required." });
    if (!Number.isSafeInteger(source.bytes) || source.bytes < 0) issues.push({ path: `sources.${source.id}.bytes`, message: "Source byte count must be a non-negative integer." });
    if (source.identityBinding) {
      const expectedBinding = canonicalJson({ name: source.name, origin: source.origin, version: source.version, sha256: source.sha256, bytes: source.bytes, mediaType: source.mediaType });
      if (source.identityBinding !== expectedBinding) issues.push({ path: `sources.${source.id}.identityBinding`, message: "Source origin/version changed without a new source identity." });
    } else issues.push({ path: `sources.${source.id}.identityBinding`, message: "Source origin/version binding is required." });
  }

  const revisionIndex = new Map(packet.revisions.map((revision, index) => [revision.id, index]));
  for (const [index, revision] of packet.revisions.entries()) {
    if (!revision.id.trim()) issues.push({ path: `revisions[${index}].id`, message: "Revision ID is required." });
    if (!/^[a-f0-9]{64}$/i.test(revision.contentSha256)) issues.push({ path: `revisions.${revision.id}.contentSha256`, message: "Revision content SHA-256 is missing or malformed." });
    if (index > 0 && !revision.parentId) issues.push({ path: `revisions.${revision.id}.parentId`, message: "Every revision after v1 must have a parent." });
    if (revision.parentId && !revisionIds.has(revision.parentId)) issues.push({ path: `revisions.${revision.id}.parentId`, message: "Parent revision is missing." });
    if (revision.parentId && (revisionIndex.get(revision.parentId) ?? Infinity) >= index) issues.push({ path: `revisions.${revision.id}.parentId`, message: "Parent revision must precede its child." });
    if (new Set(revision.sourceIds).size !== revision.sourceIds.length) issues.push({ path: `revisions.${revision.id}.sourceIds`, message: "Revision source IDs must be unique." });
    if (!revision.sourceIds.every((id) => sourceIds.has(id))) issues.push({ path: `revisions.${revision.id}.sourceIds`, message: "Revision references an unknown source." });
    const revisionSourceIds = new Set(revision.sourceIds);
    const pageIds = new Set<string>();
    const requirementIds = new Set<string>();
    const featureIds = new Set<string>();
    for (const page of revision.pages) {
      if (!page.id.trim()) issues.push({ path: `revisions.${revision.id}.pages[${pageIds.size}].id`, message: "Page ID is required." });
      if (!/^[A-Za-z0-9][A-Za-z0-9._-]*$/.test(page.id)) issues.push({ path: `revisions.${revision.id}.pages.${page.id}.id`, message: "Page ID must be a stable token." });
      if (pageIds.has(page.id)) issues.push({ path: `revisions.${revision.id}.pages.${page.id}`, message: "Page IDs must be unique." });
      pageIds.add(page.id);
      if (new Set(page.stateIds).size !== page.stateIds.length || page.stateIds.some((stateId) => !stateId.trim())) issues.push({ path: `revisions.${revision.id}.pages.${page.id}.stateIds`, message: "Page state IDs must be present and unique." });
      if (!page.sourceRefs.every((id) => sourceIds.has(id) && revisionSourceIds.has(id))) issues.push({ path: `revisions.${revision.id}.pages.${page.id}.sourceRefs`, message: "Page source join is missing or stale for this revision." });
      if (!page.editableSourceIds.every((id) => sourceIds.has(id) && revisionSourceIds.has(id))) issues.push({ path: `revisions.${revision.id}.pages.${page.id}.editableSourceIds`, message: "Editable source join is missing or stale for this revision." });
    }
    for (const requirement of revision.requirements) {
      if (!requirement.id.trim()) issues.push({ path: `revisions.${revision.id}.requirements`, message: "Requirement ID is required." });
      if (requirementIds.has(requirement.id)) issues.push({ path: `revisions.${revision.id}.requirements.${requirement.id}`, message: "Requirement IDs must be unique." });
      requirementIds.add(requirement.id);
      if (!requirement.sourceRefs.every((id) => sourceIds.has(id) && revisionSourceIds.has(id))) issues.push({ path: `revisions.${revision.id}.requirements.${requirement.id}.sourceRefs`, message: "Requirement source join is missing or stale for this revision." });
    }
    for (const feature of revision.features) {
      if (featureIds.has(feature.id)) issues.push({ path: `revisions.${revision.id}.features.${feature.id}`, message: "Obligation IDs must be unique." });
      featureIds.add(feature.id);
      if (!feature.id.trim()) issues.push({ path: `revisions.${revision.id}.features`, message: "Obligation ID is required." });
      const sceneId = sourceSceneIdFromObligationId(feature.id);
      if (feature.id.startsWith("source-scene-") && !sceneId) issues.push({ path: `revisions.${revision.id}.features.${feature.id}.id`, message: "Source-scene obligation ID is malformed." });
      if (!sourceIds.has(feature.sourceRef) || !revisionSourceIds.has(feature.sourceRef)) issues.push({ path: `revisions.${revision.id}.features.${feature.id}.sourceRef`, message: "Obligation source join is missing or stale for this revision." });
    }
    if (revision.features.some((feature) => feature.id.startsWith("source-scene-"))) {
      for (const page of revision.pages) {
        if (!featureIds.has(`source-scene-${page.id}`)) issues.push({ path: `revisions.${revision.id}.pages.${page.id}.obligation`, message: "Page is missing a stable source-scene obligation disposition." });
      }
    }
    const decisionIds = new Set<string>();
    for (const decision of revision.decisionRefs) {
      if (decisionIds.has(decision.id)) issues.push({ path: `revisions.${revision.id}.decisionRefs.${decision.id}`, message: "Decision IDs must be unique within a revision." });
      decisionIds.add(decision.id);
      const ancestors = new Set<string>();
      let cursor = revision.parentId;
      while (cursor && !ancestors.has(cursor)) {
        ancestors.add(cursor);
        cursor = packet.revisions.find((candidate) => candidate.id === cursor)?.parentId ?? null;
      }
      if (!revisionIds.has(decision.appliesToVersion) || !ancestors.has(decision.appliesToVersion)) issues.push({ path: `revisions.${revision.id}.decisionRefs.${decision.id}`, message: "Operator decision must bind to an ancestor packet version." });
      if (!decision.quote.trim() || !decision.source.trim() || !decision.sourceTimestamp.trim()) issues.push({ path: `revisions.${revision.id}.decisionRefs.${decision.id}`, message: "Decision quote, source and timestamp are all required." });
    }
    if (revision.parentId) {
      const parent = packet.revisions.find((candidate) => candidate.id === revision.parentId);
      if (parent) {
        const childReqs = new Set(revision.requirements.map((item) => item.id));
        const childFeatures = new Set(revision.features.map((item) => item.id));
        const childSources = new Set(revision.sourceIds);
        if (parent.requirements.some((item) => !childReqs.has(item.id))) issues.push({ path: `revisions.${revision.id}.requirements`, message: "Revision dropped a source requirement ID; preserve it and record its disposition." });
        if (parent.features.some((item) => !childFeatures.has(item.id))) issues.push({ path: `revisions.${revision.id}.features`, message: "Revision dropped a source-scene obligation ID; preserve it and record its disposition." });
        if (parent.sourceIds.some((id) => !childSources.has(id))) issues.push({ path: `revisions.${revision.id}.sourceIds`, message: "Revision dropped a source lineage record." });
      }
    }
  }

  if (packet.traceabilitySchema !== undefined) {
    if (packet.traceabilitySchema !== TRACEABILITY_CONTRACT_SCHEMA) {
      issues.push({ path: "traceabilitySchema", message: "Unsupported source-trace contract version." });
    } else {
      // The trace contract applies to the current exported revision. Earlier
      // revisions remain immutable, content-hash-verified history and may
      // predate this contract; rewriting them would invalidate their lineage.
      for (const revision of packet.revisions.filter((item) => item.id === packet.currentRevisionId)) {
        const revisionPath = `revisions.${revision.id}`;
        const sourceById = new Map(packet.sources.map((source) => [source.id, source]));
        const pageById = new Map(revision.pages.map((page) => [page.id, page]));
        const featureById = new Map(revision.features.map((feature) => [feature.id, feature]));
        const requirementById = new Map(revision.requirements.map((requirement) => [requirement.id, requirement]));
        for (const feature of revision.features) {
          const sceneId = sourceSceneIdFromObligationId(feature.id);
          if (!sceneId) continue;
          const path = `${revisionPath}.features.${feature.id}`;
          const page = pageById.get(sceneId);
          if (feature.pageId !== sceneId || !page) issues.push({ path: `${path}.pageId`, message: "Source-scene obligation must link to its exact atlas page." });
          if (!feature.stateIds?.length || !page || !sameSorted(feature.stateIds, page.stateIds)) issues.push({ path: `${path}.stateIds`, message: "Source-scene obligation must retain every exact state on its atlas page." });
          if (!feature.relatedSourceIds?.length || !feature.relatedSourceIds.every((id) => sourceById.has(id) && revision.sourceIds.includes(id) && page?.sourceRefs.includes(id))) issues.push({ path: `${path}.relatedSourceIds`, message: "Source-scene obligation must link its exact page, map, manifest and image source IDs." });
          if (!feature.sourceImages || !page || !sameSorted(feature.sourceImages.map((image) => image.filename), page.regions)) issues.push({ path: `${path}.sourceImages`, message: "Source-scene obligation must retain every selected image filename for its atlas page." });
          if (feature.unresolvedSourceRequirementIds?.length) issues.push({ path: `${path}.unresolvedSourceRequirementIds`, message: `Unresolved source requirement IDs remain: ${feature.unresolvedSourceRequirementIds.join(", ")}.` });
          for (const image of feature.sourceImages ?? []) {
            const source = sourceById.get(image.sourceId);
            if (!page?.sourceRefs.includes(image.sourceId) || !revision.sourceIds.includes(image.sourceId) || !source || source.name !== image.filename || source.sha256 !== image.sha256 || !image.stateIds.length || image.stateIds.some((stateId) => !page.stateIds.includes(stateId))) {
              issues.push({ path: `${path}.sourceImages.${image.filename}`, message: "Selected image filename, hash, state and packet source ID do not join exactly." });
            }
          }
          for (const id of feature.requirementIds) if (!requirementById.has(id)) issues.push({ path: `${path}.requirementIds`, message: `Linked packet requirement ${id} is missing.` });
        }
        for (const requirement of revision.requirements) {
          const path = `${revisionPath}.requirements.${requirement.id}`;
          if (!requirement.sourceRequirementIds?.length) issues.push({ path: `${path}.sourceRequirementIds`, message: "Exact source-map requirement aliases are required." });
          if (!requirement.sourceMapRefs?.length) issues.push({ path: `${path}.sourceMapRefs`, message: "Exact source-map heading/capability references are required." });
          if (!requirement.sourceStateText?.trim()) issues.push({ path: `${path}.sourceStateText`, message: "Verbatim source-map state/scope description is required." });
          if (!requirement.proofBasisText?.trim()) issues.push({ path: `${path}.proofBasisText`, message: "Verbatim source-map proof or implementation status is required." });
          if (requirement.behaviorProofStatus !== "not-verified") issues.push({ path: `${path}.behaviorProofStatus`, message: "Design-source evidence must remain explicitly separate from verified behavior." });
          if (requirement.coverage === "unknown") issues.push({ path: `${path}.coverage`, message: "Unknown visual/nonvisual coverage blocks the trace-complete export." });
          const expectedFeatures = revision.features.filter((feature) => feature.requirementIds.includes(requirement.id)).map((feature) => feature.id).sort();
          const trace = requirement.sourceSceneTrace ?? [];
          const traceIds = trace.map((entry) => entry.obligationId).sort();
          if (!sameSorted(traceIds, expectedFeatures)) issues.push({ path: `${path}.sourceSceneTrace`, message: "Requirement must link every and only source-scene obligation that names it." });
          if (requirement.coverage === "visual" && !trace.length) issues.push({ path: `${path}.sourceSceneTrace`, message: "Visual requirement needs at least one exact source-scene/page/state trace." });
          for (const entry of trace) {
            const tracePath = `${path}.sourceSceneTrace.${entry.obligationId}`;
            const page = pageById.get(entry.pageId);
            const feature = featureById.get(entry.obligationId);
            if (!page || !feature || entry.pageId !== feature.pageId || !sameSorted(entry.stateIds, feature.stateIds ?? [])) issues.push({ path: tracePath, message: "Requirement source-scene page, state and obligation links do not agree." });
            if (!entry.sourceMapSourceId || !entry.manifestSourceId || !entry.sourceIds.includes(entry.sourceMapSourceId) || !entry.sourceIds.includes(entry.manifestSourceId) || !entry.sourceIds.every((id) => sourceById.has(id) && revision.sourceIds.includes(id) && page?.sourceRefs.includes(id))) issues.push({ path: `${tracePath}.sourceIds`, message: "Requirement trace source IDs must join to the exact retained map, manifest, page and image records." });
            if (!sameSorted(entry.editableSourceIds, page?.editableSourceIds ?? []) || !entry.editableSourceIds.length) issues.push({ path: `${tracePath}.editableSourceIds`, message: "Requirement trace must link every editable packet source for its atlas page." });
            if (!feature?.sourceImages || canonicalJson(entry.sourceImages) !== canonicalJson(feature.sourceImages)) issues.push({ path: `${tracePath}.sourceImages`, message: "Requirement trace must retain the exact selected filename, hash and source ID set." });
          }
          if (requirement.coverage === "visual") {
            const page = requirement.mapping.pageId ? pageById.get(requirement.mapping.pageId) : undefined;
            const filename = requirement.mapping.region?.match(/^Selected design frame \((.+)\)$/)?.[1];
            const source = requirement.mapping.sourceId ? sourceById.get(requirement.mapping.sourceId) : undefined;
            const mappingRequiredNow = revision.id === packet.currentRevisionId;
            const hasCompleteHistoricalMapping = Boolean(requirement.mapping.pageId && requirement.mapping.stateId && requirement.mapping.region && requirement.mapping.sourceId);
            if (mappingRequiredNow || hasCompleteHistoricalMapping) {
              if (!page || !requirement.mapping.stateId || !filename || !source || !revision.sourceIds.includes(source.id) || source.name !== filename || !page.regions.includes(filename) || !page.stateIds.includes(requirement.mapping.stateId) || !page.sourceRefs.includes(source.id)) {
                issues.push({ path: `${path}.mapping`, message: "Visual requirement needs an exact page, state, selected frame, hash-bound source ID and scene-trace link." });
              }
              if (!trace.some((entry) => entry.pageId === requirement.mapping.pageId && entry.stateIds.includes(requirement.mapping.stateId ?? ""))) issues.push({ path: `${path}.mapping`, message: "Selected requirement frame must belong to one of its source-mapped scenes/states." });
            }
          } else if (requirement.coverage === "nonvisual") {
            const explicitNoVisual = /\b(no|nonvisual|non-visual|not selected as a visual|future requirement)\b/i.test(requirement.sourceStateText ?? "");
            if (!explicitNoVisual) issues.push({ path: `${path}.sourceStateText`, message: "Nonvisual proof scope must be explicitly named by the retained source map." });
          }
        }
        for (const decision of revision.decisionRefs) {
          if (!decision.scopeDescription?.trim() || !decision.affectedPageIds?.length || decision.affectedPageIds.some((pageId) => !pageById.has(pageId))) {
            issues.push({ path: `${revisionPath}.decisionRefs.${decision.id}.affectedPageIds`, message: "Version-bound correction must name its full affected-page scope." });
          }
        }
      }
    }
  }

  const mappedRequirementCount = current?.requirements.filter((requirement) => {
    if (requirement.coverage === "nonvisual") return requirement.disposition !== "unknown";
    return Boolean(requirement.mapping.pageId && requirement.mapping.stateId && requirement.mapping.region);
  }).length ?? 0;
  current?.requirements.forEach((requirement) => {
    if (requirement.coverage === "visual" && (!requirement.mapping.pageId || !requirement.mapping.stateId || !requirement.mapping.region)) {
      issues.push({ path: `requirements.${requirement.id}.mapping`, message: "Visual requirements need a page, state and region, or an explicit nonvisual disposition." });
    } else if (requirement.coverage === "unknown" && requirement.disposition === "unknown") {
      issues.push({ path: `requirements.${requirement.id}.disposition`, message: "Unmapped requirement needs an explicit preserve, remove, defer or unknown disposition." });
    }
    if (requirement.coverage === "visual" && requirement.mapping.pageId && requirement.mapping.stateId) {
      const page = current.pages.find((candidate) => candidate.id === requirement.mapping.pageId);
      if (!page) issues.push({ path: `requirements.${requirement.id}.mapping.pageId`, message: "Requirement references a missing or stale page ID." });
      else if (!page.stateIds.includes(requirement.mapping.stateId)) issues.push({ path: `requirements.${requirement.id}.mapping.stateId`, message: "Requirement state is not present on the selected page." });
      const selectedFrame = requirement.mapping.region?.match(/^Selected design frame \((.+)\)$/);
      if (selectedFrame) {
        const filename = selectedFrame[1]!;
        const sourceHash = requirement.mapping.rationale.match(new RegExp(`selected-image manifest binds ${escapeRegExp(filename)} \\(([a-f0-9]{64})\\)`, "i"))?.[1]?.toLowerCase();
        const source = packet.sources.find((candidate) => candidate.name === filename && candidate.sha256 === sourceHash);
        if (!page?.regions.includes(filename) || !source || !source.id || !current.sourceIds.includes(source.id) || !page.sourceRefs.includes(source.id)) {
          issues.push({ path: `requirements.${requirement.id}.mapping.region`, message: "Selected-frame mapping is missing an exact source filename/hash join in this revision." });
        }
      }
    }
    if (requirement.coverage === "unknown") issues.push({ path: `requirements.${requirement.id}.coverage`, message: "Unknown visual/nonvisual coverage must be resolved before complete packet export." });
  });

  const explicitlyDisposedFeatureCount = current?.features.filter((feature) => feature.disposition !== "unknown" || feature.rationale.trim().length > 0).length ?? 0;
  current?.features.forEach((feature) => {
    if (feature.disposition === "unknown") issues.push({ path: `features.${feature.id}.disposition`, message: "Unknown feature disposition must be resolved before complete packet export." });
  });
  return { valid: issues.length === 0, issues, mappedRequirementCount, explicitlyDisposedFeatureCount };
}

/** Verify immutable revision hashes and source IDs that bind content plus origin/version metadata. */
export async function verifyPacketIntegrity(packet: DesignPacket, options: { allowUnresolvedMappings?: boolean } = {}): Promise<PacketValidation> {
  const structural = validatePacket(packet);
  const issues = structural.issues.filter((issue) => !(options.allowUnresolvedMappings && (issue.path.startsWith("requirements.") || issue.path.includes(".requirements.")) && issue.path.endsWith(".mapping") && (
    issue.message.startsWith("Visual requirements need a page, state and region") ||
    issue.message.startsWith("Visual requirement needs an exact page, state, selected frame") ||
    issue.message.startsWith("Selected requirement frame must belong to one of its source-mapped scenes/states")
  )));
  for (const source of packet.sources) {
    const identityBinding = canonicalJson({ name: source.name, origin: source.origin, version: source.version, sha256: source.sha256, bytes: source.bytes, mediaType: source.mediaType });
    if (source.identityBinding && source.identityBinding !== identityBinding) issues.push({ path: `sources.${source.id}.identityBinding`, message: "Source ID does not bind its recorded origin, version and content identity." });
    const metadataHash = await sha256Hex(identityBinding);
    const currentId = `src-${source.sha256.slice(0, 16)}-${metadataHash.slice(0, 12)}`;
    if (source.id !== currentId || source.identityBinding !== identityBinding) issues.push({ path: `sources.${source.id}.id`, message: "Source ID does not bind its recorded origin, version and content identity." });
  }
  for (const revision of packet.revisions) {
    const { contentSha256, ...seed } = revision;
    const expected = await sha256Hex(revisionContent(seed));
    if (contentSha256 !== expected) issues.push({ path: `revisions.${revision.id}.contentSha256`, message: "Revision content changed without creating a new immutable revision." });
  }
  return { ...structural, valid: issues.length === 0, issues };
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character] ?? character);
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function buildAtlasHtml(packet: DesignPacket): string {
  const revision = packet.revisions.find((item) => item.id === packet.currentRevisionId);
  if (!revision) throw new Error("The current packet revision is missing.");
  const pageLinks = revision.pages.map((page) => `<li><a href="#page-${encodeURIComponent(page.id)}">${escapeHtml(page.title)}</a></li>`).join("");
  const requirementTraceRows = revision.requirements.map((item) => {
    const traces = (item.sourceSceneTrace ?? []).map((trace) => `${trace.pageId} [${trace.stateIds.join(", ")}] · ${trace.obligationId} · source IDs ${trace.sourceIds.join(", ")} · editable ${trace.editableSourceIds.join(", ")}`).join("<br>") || "No visual page; see exact nonvisual source/proof description.";
    return `<tr><th scope="row">${escapeHtml(item.id)}</th><td>${escapeHtml(item.text)} · ${escapeHtml(item.scope)}</td><td>${escapeHtml(item.sourceRequirementIds?.join(", ") ?? "")}</td><td>${escapeHtml(item.coverage)} · ${escapeHtml(item.sourceStateText ?? "")}</td><td>${traces}</td><td>${escapeHtml(item.mapping.pageId ?? "—")} / ${escapeHtml(item.mapping.stateId ?? "—")} / ${escapeHtml(item.mapping.region ?? "—")} · ${escapeHtml(item.mapping.sourceId ?? "—")}</td><td>${escapeHtml(item.disposition)}</td><td>Not verified</td><td>${escapeHtml(item.proofBasisText ?? "")}</td></tr>`;
  }).join("");
  const pages = revision.pages.map((page) => {
    const requirements = revision.requirements.filter((item) => item.mapping.pageId === page.id || item.sourceSceneTrace?.some((trace) => trace.pageId === page.id));
    const rows = requirements.map((item) => {
      const trace = item.sourceSceneTrace?.find((entry) => entry.pageId === page.id);
      return `<tr><th scope="row">${escapeHtml(item.id)}</th><td>${escapeHtml(item.text)}</td><td>${escapeHtml(trace?.stateIds.join(", ") || item.mapping.stateId || "Unmapped")}</td><td>${escapeHtml(item.mapping.region ?? "No single representative frame")}</td><td>${escapeHtml(trace?.obligationId ?? "—")}</td><td>${escapeHtml(trace?.sourceIds.join(", ") ?? item.sourceRefs.join(", "))}</td><td>${escapeHtml(item.disposition)} · behavior not verified</td></tr>`;
    }).join("");
    const obligation = revision.features.find((feature) => feature.id === `source-scene-${page.id}`);
    return `<section id="page-${encodeURIComponent(page.id)}"><h2>${escapeHtml(page.title)}</h2><p>Page <code>${escapeHtml(page.id)}</code> · states: ${page.stateIds.map(escapeHtml).join(", ") || "none recorded"}</p><p>Source refs: ${page.sourceRefs.map(escapeHtml).join(", ") || "none recorded"}</p><p>Editable sources: ${page.editableSourceIds.map(escapeHtml).join(", ") || "none recorded"}</p><p>Layout regions: ${(page.designRegions ?? []).map(escapeHtml).join(", ") || "none recorded"}</p><p>Omission disposition: ${escapeHtml(obligation?.disposition ?? "missing")} · ${escapeHtml(obligation?.rationale ?? "No matching scene obligation.")}</p><h3>Linked requirements and state trace</h3><table><thead><tr><th>Requirement</th><th>Source text</th><th>Trace states</th><th>Representative frame</th><th>Scene obligation</th><th>Exact source IDs</th><th>Disposition / behavior proof</th></tr></thead><tbody>${rows || "<tr><td colspan=\"7\">No source requirement links this page.</td></tr>"}</tbody></table></section>`;
  }).join("");
  const omissions = revision.features.map((feature) => `<tr><th scope="row"><code>${escapeHtml(feature.id)}</code></th><td>${escapeHtml(feature.name)}</td><td>${escapeHtml(feature.pageId ?? "—")}</td><td>${escapeHtml(feature.stateIds?.join(", ") ?? "—")}</td><td>${escapeHtml(feature.sourceRequirementIds?.join(", ") ?? "—")}</td><td>${escapeHtml(feature.requirementIds.join(", ") || "none named")}</td><td>${escapeHtml(feature.relatedSourceIds?.join(", ") ?? feature.sourceRef)}</td><td>${escapeHtml((feature.sourceImages ?? []).map((image) => `${image.filename} · ${image.sha256} · ${image.sourceId} · ${image.stateIds.join(", ")}`).join("<br>") || "none")}</td><td>${escapeHtml(feature.disposition)}</td><td>${escapeHtml(feature.rationale)}</td></tr>`).join("");
  const sources = packet.sources.map((source) => `<tr><th scope="row"><code>${escapeHtml(source.id)}</code></th><td>${escapeHtml(source.name)}</td><td>${escapeHtml(source.origin)}</td><td>${escapeHtml(source.version)}</td><td><code>${escapeHtml(source.sha256)}</code></td><td>${escapeHtml(source.use)}</td><td>${escapeHtml(source.amendmentHistory.join("; "))}</td></tr>`).join("");
  const history = packet.revisions.map((item) => `<li><a href="#revision-${encodeURIComponent(item.id)}">${escapeHtml(item.id)}</a> — ${escapeHtml(item.changeSummary)} <code>${escapeHtml(item.contentSha256)}</code></li>`).join("");
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${escapeHtml(packet.scenario)} design atlas</title><style>
    :root{font-family:system-ui,sans-serif;color:#17242e;background:#f3f6f8}body{max-width:72rem;margin:auto;padding:2rem;line-height:1.55}header,section,nav{background:#fff;border:1px solid #c7d2db;border-radius:.75rem;padding:1.25rem;margin:1rem 0}h1,h2,h3{line-height:1.2}a{color:#075985}table{border-collapse:collapse;width:100%;margin:1rem 0}th,td{text-align:start;vertical-align:top;border:1px solid #c7d2db;padding:.5rem}code{overflow-wrap:anywhere}.badge{display:inline-block;padding:.15rem .45rem;border-radius:999px;background:#e7eef3}@media print{body{max-width:none;padding:0;background:#fff}header,section,nav{break-inside:avoid;border-color:#9ca3af}a{color:inherit;text-decoration:none}nav{break-after:page}section{break-before:page}}
    </style></head><body><header><p class="badge">Design packet ${escapeHtml(revision.id)} · ${escapeHtml(revision.contentSha256)}</p><h1>${escapeHtml(packet.scenario)} — design atlas</h1><p>${escapeHtml(packet.brief)}</p><p>Packet ${escapeHtml(packet.id)}; this is source/design evidence, not product implementation or whole-product acceptance. All requirement behavior proof remains unverified.</p></header><nav aria-label="Atlas contents"><h2>Pages</h2><ol>${pageLinks}</ol></nav><section><h2>Source lineage</h2><table><thead><tr><th>Source ID</th><th>Source</th><th>Origin</th><th>Version</th><th>SHA-256</th><th>Use</th><th>Amendment history</th></tr></thead><tbody>${sources}</tbody></table></section><section><h2>All requirement source traces (${revision.requirements.length})</h2><table><thead><tr><th>Requirement</th><th>Source requirement</th><th>Source-map IDs</th><th>Coverage and source scope</th><th>Every source page/state/obligation</th><th>Representative frame/source ID</th><th>Disposition</th><th>Behavior proof</th><th>Source-map proof basis</th></tr></thead><tbody>${requirementTraceRows}</tbody></table></section>${pages}<section><h2>Omission and preservation register (${revision.features.length})</h2><table><thead><tr><th>Obligation ID</th><th>Source obligation</th><th>Page</th><th>States</th><th>Source requirement IDs</th><th>Linked packet requirements</th><th>Exact source IDs</th><th>Selected images</th><th>Disposition</th><th>Rationale</th></tr></thead><tbody>${omissions || "<tr><td colspan=\"10\">No omitted features recorded.</td></tr>"}</tbody></table></section><section><h2>Revision history</h2><ol>${history}</ol>${packet.revisions.map((item) => `<div id="revision-${encodeURIComponent(item.id)}"><h3>${escapeHtml(item.id)} — ${escapeHtml(item.changeSummary)}</h3>${item.decisionRefs.map((decision) => `<blockquote>${escapeHtml(decision.quote)}<p>Operator decision ${escapeHtml(decision.id)} · ${escapeHtml(decision.sourceTimestamp)} · source ${escapeHtml(decision.source)} · linked to ${escapeHtml(decision.appliesToVersion)} · affected pages ${escapeHtml(decision.affectedPageIds?.join(", ") ?? "unspecified")} · ${escapeHtml(decision.scopeDescription ?? "")}</p></blockquote>`).join("")}</div>`).join("")}</section></body></html>`;
}
