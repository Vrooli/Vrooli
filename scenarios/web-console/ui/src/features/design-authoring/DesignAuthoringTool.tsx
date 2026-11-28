import { useMemo, useRef, useState } from "react";
import { uploadFile } from "../../api/uploads";
import { OverlayActionBar } from "@vrooli/react-component-library/OverlayActionBar/1";
import { addPacketRevision, buildAtlasHtml, createDesignPacket, createSourceRecord, DESIGN_PACKET_SCHEMA, LEGACY_DESIGN_PACKET_SCHEMA, TRACEABILITY_CONTRACT_SCHEMA, parseRequirementSourceMap, parseSceneSourceMap, parseSelectedImageManifest, resolveSelectedState, sha256Hex, sourceRequirementAliases, sourceScenesForRequirement, sourceSceneIdFromObligationId, sourceSceneObligationId, validatePacket, validateSceneManifestJoins, verifyPacketIntegrity, verifySourceReimportIdentity, type CoverageKind, type DesignPacket, type OmissionDisposition, type PacketFeature, type PacketPage, type PacketRequirement, type PacketSource, type SelectedDesignAsset } from "./model";

type ImportedFile = File & { webkitRelativePath?: string };

interface DesignAuthoringToolProps {
  sessionId?: string;
  onOpenArtifact?: (path: string) => void;
  initialSource?: { path: string; text: string };
}

interface LocalZipEntry { name: string; bytes: Uint8Array; }

const ZIP_CRC32_TABLE = Uint32Array.from({ length: 256 }, (_, value) => {
  let crc = value;
  for (let bit = 0; bit < 8; bit += 1) crc = (crc >>> 1) ^ (crc & 1 ? 0xedb88320 : 0);
  return crc >>> 0;
});

function zipCrc32(bytes: Uint8Array): number {
  let crc = 0xffffffff;
  for (const byte of bytes) crc = (crc >>> 8) ^ ZIP_CRC32_TABLE[(crc ^ byte) & 0xff]!;
  return (crc ^ 0xffffffff) >>> 0;
}

export async function makeLocalZip(entries: Array<{ name: string; data: Blob | string }>): Promise<Blob> {
  // The maximum field values are sentinels for ZIP64, which this writer does
  // not emit. Keep every classic ZIP field below those sentinels.
  const zipU16Max = 0xfffe;
  const zipU32Max = 0xfffffffe;
  const encoder = new TextEncoder();
  if (entries.length > zipU16Max) throw new Error("The local export has too many files for a classic ZIP archive.");
  const prepared: Array<LocalZipEntry & { crc: number }> = [];
  const fileNames = new Set<string>();
  const directoryPrefixes = new Set<string>();
  for (const entry of entries) {
    const name = entry.name.replace(/\\/g, "/");
    const pathParts = name.split("/");
    const key = name.toLocaleLowerCase("en-US");
    const unsafe = !name || name.startsWith("/") || /^[A-Za-z]:/.test(name) || /[\u0000-\u001f\u007f]/.test(name) ||
      pathParts.some((part) => !part || part === "." || part === "..") || encoder.encode(name).length > 0xffff;
    if (unsafe) throw new Error("The local export contains an unsafe path.");
    if (fileNames.has(key) || directoryPrefixes.has(key)) {
      throw new Error("The local export contains duplicate or colliding file paths.");
    }
    let prefix = "";
    for (let index = 0; index < pathParts.length - 1; index += 1) {
      prefix = prefix ? `${prefix}/${pathParts[index]!.toLocaleLowerCase("en-US")}` : pathParts[index]!.toLocaleLowerCase("en-US");
      if (fileNames.has(prefix)) throw new Error("The local export contains duplicate or colliding file paths.");
      directoryPrefixes.add(prefix);
    }
    fileNames.add(key);
    if (typeof entry.data !== "string" && entry.data.size > zipU32Max) {
      throw new Error("The local export contains a file that exceeds the classic ZIP size limit.");
    }
    const bytes = typeof entry.data === "string" ? encoder.encode(entry.data) : new Uint8Array(await readBytes(entry.data));
    if (bytes.length > zipU32Max) throw new Error("The local export contains a file that exceeds the classic ZIP size limit.");
    prepared.push({ name, bytes, crc: zipCrc32(bytes) });
  }
  const localParts: Uint8Array[] = [];
  const centralParts: Uint8Array[] = [];
  let localOffset = 0;
  for (const entry of prepared) {
    const nameBytes = encoder.encode(entry.name);
    const local = new Uint8Array(30 + nameBytes.length);
    const localView = new DataView(local.buffer);
    localView.setUint32(0, 0x04034b50, true); localView.setUint16(4, 20, true); localView.setUint16(6, 0x0800, true);
    localView.setUint16(8, 0, true); localView.setUint16(10, 0, true); localView.setUint16(12, 0x0021, true);
    localView.setUint32(14, entry.crc, true); localView.setUint32(18, entry.bytes.length, true); localView.setUint32(22, entry.bytes.length, true);
    localView.setUint16(26, nameBytes.length, true); localView.setUint16(28, 0, true); local.set(nameBytes, 30);
    const nextLocalOffset = localOffset + local.length + entry.bytes.length;
    if (nextLocalOffset > zipU32Max) throw new Error("The local export exceeds the classic ZIP archive size limit.");
    localParts.push(local, entry.bytes);
    const central = new Uint8Array(46 + nameBytes.length);
    const centralView = new DataView(central.buffer);
    centralView.setUint32(0, 0x02014b50, true); centralView.setUint16(4, 20, true); centralView.setUint16(6, 20, true);
    centralView.setUint16(8, 0x0800, true); centralView.setUint16(10, 0, true); centralView.setUint16(12, 0, true); centralView.setUint16(14, 0x0021, true);
    centralView.setUint32(16, entry.crc, true); centralView.setUint32(20, entry.bytes.length, true); centralView.setUint32(24, entry.bytes.length, true);
    centralView.setUint16(28, nameBytes.length, true); centralView.setUint16(30, 0, true); centralView.setUint16(32, 0, true);
    centralView.setUint16(34, 0, true); centralView.setUint16(36, 0, true); centralView.setUint32(38, 0, true); centralView.setUint32(42, localOffset, true);
    central.set(nameBytes, 46); centralParts.push(central);
    localOffset = nextLocalOffset;
  }
  const centralSize = centralParts.reduce((sum, part) => sum + part.length, 0);
  if (centralSize > zipU32Max || localOffset + centralSize + 22 > zipU32Max) {
    throw new Error("The local export exceeds the classic ZIP archive size limit.");
  }
  const end = new Uint8Array(22); const endView = new DataView(end.buffer);
  endView.setUint32(0, 0x06054b50, true); endView.setUint16(4, 0, true); endView.setUint16(6, 0, true);
  endView.setUint16(8, prepared.length, true); endView.setUint16(10, prepared.length, true);
  endView.setUint32(12, centralSize, true); endView.setUint32(16, localOffset, true); endView.setUint16(20, 0, true);
  const parts = [...localParts, ...centralParts, end];
  const archive = new Uint8Array(parts.reduce((sum, part) => sum + part.length, 0));
  let cursor = 0;
  for (const part of parts) { archive.set(part, cursor); cursor += part.length; }
  return new Blob([archive.buffer], { type: "application/zip" });
}

function downloadLocalFile(data: Blob, filename: string): void {
  const url = URL.createObjectURL(data);
  const anchor = document.createElement("a"); anchor.href = url; anchor.download = filename; anchor.style.display = "none";
  document.body.append(anchor); anchor.click(); anchor.remove(); window.setTimeout(() => URL.revokeObjectURL(url), 30_000);
}

interface PreparedInputs {
  sourceMap: File;
  sourceMapText: string;
  dossier: File;
  dossierText: string;
  supportingDocs: File[];
  manifest: File;
  manifestText: string;
  images: Map<string, File>;
  originals: File[];
  bundles: File[];
  packetArchive?: { file: File; packet: DesignPacket };
  all: File[];
}

interface PacketArchiveRecovery {
  sourceId: string;
  packetId: string;
  preservedCurrentRevision: string;
  archiveSha256: string;
}

interface EditableOriginal {
  name: string;
  origin: string;
  content: string;
  sourceId: string;
  embeddedFontDataUrl?: string;
}

const CORRECTION = {
  authority: "operator" as const,
  id: "NH-DESIGN-20261025-01",
  source: "design-to-implementation/cases/notification-hub/legacy-delivery/FEEDBACK.md",
  sourceTimestamp: "2026-10-25T14:19:12Z",
  quote: "I do in fact want the bottom nav.",
};

function readBytes(file: Blob): Promise<ArrayBuffer> {
  if (typeof file.arrayBuffer === "function") return file.arrayBuffer();
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as ArrayBuffer);
    reader.onerror = () => reject(new Error("Could not read a selected source file."));
    reader.readAsArrayBuffer(file);
  });
}

function readText(file: File): Promise<string> {
  if (typeof file.text === "function") return file.text();
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ""));
    reader.onerror = () => reject(new Error("Could not read a selected text source."));
    reader.readAsText(file);
  });
}

async function createImportedEditableSourceRecord(file: File, bytes: ArrayBuffer, origin: string, archivedSources: PacketSource[] = []): Promise<PacketSource> {
  const sha256 = await sha256Hex(bytes);
  const matches = archivedSources.filter((source) => source.name === file.name && source.sha256 === sha256 && source.bytes === bytes.byteLength);
  if (matches.length > 1) throw new Error(`Retained editable source ${file.name} matches multiple archived source identities; recovery is ambiguous.`);
  const archived = matches[0];
  // A packet export can keep an amended editable source under `editable/`
  // instead of the original `source-inputs/` folder. If the operator supplies
  // that exact named source during recovery, its bytes and size bind it to the
  // archived descriptor. Missing or ambiguous bytes still fail closed.
  return createSourceRecord({
    name: archived?.name ?? file.name,
    origin: archived?.origin ?? origin,
    version: archived?.version ?? "retained editable source; exact imported bytes",
    bytes,
    mediaType: archived?.mediaType ?? (file.type || "text/plain"),
    use: archived?.use ?? "Editable original retained alongside the source-backed atlas",
    amendmentHistory: archived?.amendmentHistory,
  });
}

function packetArchiveFromFile(file: File, text: string): DesignPacket {
  let value: unknown;
  try { value = JSON.parse(text); }
  catch { throw new Error(`Retained packet archive ${file.name} is not valid JSON.`); }
  if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error(`Retained packet archive ${file.name} is not a design packet object.`);
  const packet = value as Partial<DesignPacket>;
  if ((packet.schema !== DESIGN_PACKET_SCHEMA && packet.schema !== LEGACY_DESIGN_PACKET_SCHEMA) || typeof packet.id !== "string" || typeof packet.scenario !== "string" || !Array.isArray(packet.revisions) || typeof packet.currentRevisionId !== "string") {
    throw new Error(`Retained packet archive ${file.name} does not match the supported design-packet schema.`);
  }
  return packet as DesignPacket;
}

function assertPacketArchiveMatchesInputs(archive: DesignPacket, scenario: string, requirements: PacketRequirement[], features: PacketFeature[]): void {
  if (archive.scenario !== scenario) throw new Error(`The retained packet archive is for ${archive.scenario}, but the selected source inputs resolve to ${scenario}.`);
  const v1 = archive.revisions.find((revision) => revision.id === "v1");
  const current = archive.revisions.find((revision) => revision.id === archive.currentRevisionId);
  if (!current) throw new Error("The retained packet archive has no current revision to reconcile.");
  const hubRecovery = scenario === "Notification Hub";
  const baseline = hubRecovery ? v1 : current;
  if (!baseline) throw new Error("The retained Notification Hub packet archive must include its original v1 revision.");
  if (hubRecovery) {
    const correction = current.decisionRefs.find((decision) => decision.id === CORRECTION.id);
    if (current.id !== "v2-bottom-nav" || current.parentId !== baseline.id || !correction || correction.appliesToVersion !== baseline.id || current.decisionRefs.length !== 1 || correction.authority !== CORRECTION.authority || correction.source !== CORRECTION.source || correction.sourceTimestamp !== CORRECTION.sourceTimestamp || correction.quote !== CORRECTION.quote) {
      throw new Error("The retained packet must preserve v1 and its exact v2-bottom-nav correction ancestry before identity recovery.");
    }
  }
  const unique = (values: string[]) => new Set(values).size === values.length;
  if ((hubRecovery && (baseline.requirements.length !== 28 || current.requirements.length !== 28 || requirements.length !== 28)) || baseline.requirements.length !== requirements.length || !unique(baseline.requirements.map((item) => item.id)) || !unique(requirements.map((item) => item.id))) {
    throw new Error(hubRecovery ? "The retained packet and selected source map must each contain the same 28 unique Hub requirement rows." : "The retained packet and selected source map must contain the same unique requirement rows.");
  }
  if ((hubRecovery && (baseline.features.length !== 102 || current.features.length !== 102 || features.length !== 102)) || baseline.features.length !== features.length || !unique(baseline.features.map((item) => item.id)) || !unique(features.map((item) => item.id))) {
    throw new Error(hubRecovery ? "The retained packet and selected source map must each contain the same 102 unique Hub source-obligation IDs." : "The retained packet and selected source map must contain the same unique obligation IDs.");
  }
  // Keep source-map identity checks separate from author mapping state. V1
  // mappings can predate the current scene-join rules and v2 intentionally
  // changes reviewed mappings; exact packet bytes remain archived as a source.
  const requirementSignature = (item: PacketRequirement) => JSON.stringify({ id: item.id, text: item.text, scope: item.scope, coverage: item.coverage, disposition: item.disposition });
  const archivedRequirements = new Map(baseline.requirements.map((item) => [item.id, requirementSignature(item)]));
  const mismatchedRequirements = requirements.filter((item) => archivedRequirements.get(item.id) !== requirementSignature(item));
  if (mismatchedRequirements.length) {
    const first = mismatchedRequirements[0];
    throw new Error(`The re-imported source map does not exactly reproduce the archived requirement IDs, text, scope, coverage and dispositions. First mismatch: ${first?.id ?? "unknown"}.`);
  }
  // Requirement joins are derived from current scene matching and can change
  // as mapping rules evolve. Obligation identity, source label and disposition
  // are the stable source-map facts that recovery must preserve.
  const featureSignature = (item: PacketFeature) => JSON.stringify({ id: item.id, name: item.name, disposition: item.disposition });
  const archivedFeatures = new Map(baseline.features.map((item) => [item.id, featureSignature(item)]));
  if (features.some((item) => archivedFeatures.get(item.id) !== featureSignature(item))) {
    throw new Error("The re-imported source map does not exactly reproduce the archived obligation IDs, source labels and dispositions.");
  }
  if (current.requirements.length !== baseline.requirements.length || !unique(current.requirements.map((item) => item.id)) || current.requirements.some((item) => !archivedRequirements.has(item.id)) || current.features.length !== baseline.features.length || !unique(current.features.map((item) => item.id)) || current.features.some((item) => !archivedFeatures.has(item.id))) {
    throw new Error("The corrected archive revision dropped a source requirement or obligation identity.");
  }
  if (hubRecovery && current.requirements.some((item) => archivedRequirements.get(item.id) !== requirementSignature(item))) {
    throw new Error("The archived correction revision changed a requirement mapping or disposition outside the source correction; recovery is blocked.");
  }
  const baselineFeatureSignatures = new Map(baseline.features.map((item) => [item.id, featureSignature(item)]));
  if (hubRecovery && current.features.some((item) => baselineFeatureSignatures.get(item.id) !== featureSignature(item))) {
    throw new Error("The archived bottom-navigation correction changed or dropped a source-obligation disposition.");
  }
  const currentP2 = current.requirements.filter((item) => /\bP2 future requirement\b/i.test(item.scope));
  const p2 = requirements.filter((item) => /\bP2 future requirement\b/i.test(item.scope));
  if (hubRecovery && (p2.length !== 2 || p2.some((item) => item.disposition !== "defer") || currentP2.length !== 2 || currentP2.some((item) => item.disposition !== "defer"))) throw new Error("Both source-map P2 future requirements must remain explicitly deferred during recovery.");
}

async function readZipEntries(file: Blob): Promise<Map<string, Uint8Array>> {
  const archive = new Uint8Array(await readBytes(file));
  const view = new DataView(archive.buffer, archive.byteOffset, archive.byteLength);
  const lowerBound = Math.max(0, archive.length - 65_557);
  let endRecord = -1;
  for (let offset = archive.length - 22; offset >= lowerBound; offset -= 1) {
    if (view.getUint32(offset, true) === 0x06054b50) { endRecord = offset; break; }
  }
  if (endRecord < 0) throw new Error("The selected source bundle is not a readable ZIP archive.");
  const entries = view.getUint16(endRecord + 10, true);
  const directoryOffset = view.getUint32(endRecord + 16, true);
  const output = new Map<string, Uint8Array>();
  let cursor = directoryOffset;
  for (let index = 0; index < entries; index += 1) {
    if (view.getUint32(cursor, true) !== 0x02014b50) throw new Error("The source bundle central directory is malformed.");
    const method = view.getUint16(cursor + 10, true);
    const compressedSize = view.getUint32(cursor + 20, true);
    const filenameLength = view.getUint16(cursor + 28, true);
    const extraLength = view.getUint16(cursor + 30, true);
    const commentLength = view.getUint16(cursor + 32, true);
    const localOffset = view.getUint32(cursor + 42, true);
    const filename = new TextDecoder().decode(archive.subarray(cursor + 46, cursor + 46 + filenameLength));
    cursor += 46 + filenameLength + extraLength + commentLength;
    if (!/^(prototype\.js|fonts\/SourceSans3\.ttf)$/i.test(filename)) continue;
    if (view.getUint32(localOffset, true) !== 0x04034b50) throw new Error("The source bundle entry has no local header: " + filename);
    const localNameLength = view.getUint16(localOffset + 26, true);
    const localExtraLength = view.getUint16(localOffset + 28, true);
    const dataStart = localOffset + 30 + localNameLength + localExtraLength;
    const compressed = archive.slice(dataStart, dataStart + compressedSize);
    let contents: Uint8Array;
    if (method === 0) contents = compressed;
    else if (method === 8) {
      const compressedPayload = new Uint8Array(compressed.byteLength);
      compressedPayload.set(compressed);
      const decompressor = new DecompressionStream("deflate-raw");
      const contentsPromise = new Response(decompressor.readable).arrayBuffer();
      const writer = decompressor.writable.getWriter();
      await writer.write(compressedPayload);
      await writer.close();
      contents = new Uint8Array(await contentsPromise);
    } else throw new Error("Unsupported ZIP compression for " + filename);
    output.set(filename, contents);
  }
  return output;
}

function bytesDataUrl(bytes: Uint8Array, mimeType: string): string {
  let binary = "";
  const chunkSize = 0x8000;
  for (let offset = 0; offset < bytes.length; offset += chunkSize) binary += String.fromCharCode(...bytes.subarray(offset, offset + chunkSize));
  return "data:" + mimeType + ";base64," + btoa(binary);
}

function escapeHtml(value: string): string {
  return value.replace(/[&<>"']/g, (character) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[character] ?? character);
}

async function guessInput(files: File[]): Promise<PreparedInputs> {
  const inputNames = files.map((file) => file.name.toLowerCase());
  if (new Set(inputNames).size !== inputNames.length) throw new Error("Selected source inputs contain duplicate filenames; manifest and archive joins would be ambiguous.");
  const markdown = files.filter((file) => /\.md$/i.test(file.name));
  const mapCandidates = await Promise.all(markdown.map(async (file) => {
    const text = await readText(file);
    return parseRequirementSourceMap(text).length && parseSceneSourceMap(text).length ? { file, text } : null;
  }));
  const maps = mapCandidates.filter((candidate): candidate is { file: File; text: string } => candidate !== null);
  const sourceMap = maps.length === 1 ? maps[0] : null;
  if (!sourceMap) throw new Error(maps.length > 1 ? "Select one Markdown source map with a Requirement crosswalk and Appendix A scene table." : "Select the Markdown source map with a Requirement crosswalk and Appendix A scene table.");
  const remainingMarkdown = markdown.filter((file) => file !== sourceMap.file);
  const preferredDossier = remainingMarkdown.filter((file) => /dossier|brief|design/i.test(file.name));
  const dossier = preferredDossier.length === 1 ? preferredDossier[0] : remainingMarkdown.length === 1 ? remainingMarkdown[0] : null;
  if (!dossier) throw new Error("Select one design dossier or brief in Markdown alongside the source map.");
  const supportingDocs = remainingMarkdown.filter((file) => file !== dossier);
  const dossierText = await readText(dossier);
  const manifestCandidates = await Promise.all(files.filter((file) => /\.json$/i.test(file.name)).map(async (file) => {
    try { return { file, text: await readText(file), assets: parseSelectedImageManifest(await readText(file)) }; }
    catch { return null; }
  }));
  const manifests = manifestCandidates.filter((candidate): candidate is { file: File; text: string; assets: SelectedDesignAsset[] } => candidate !== null);
  const manifest = manifests.length === 1 ? manifests[0] : null;
  if (!manifest) throw new Error(manifests.length > 1 ? "Select one image manifest with filename, sceneId, stateIds, SHA-256 and sourceRefs." : "Select the JSON image manifest with filename, sceneId, stateIds, SHA-256 and sourceRefs.");
  const packetArchiveFiles = files.filter((file) => /design-packet.*\.json$/i.test(file.name));
  if (packetArchiveFiles.length > 1) throw new Error("Select exactly one prior design-packet JSON to preserve as the recovery archive.");
  const packetArchiveFile = packetArchiveFiles[0];
  const packetArchive = packetArchiveFile ? { file: packetArchiveFile, packet: packetArchiveFromFile(packetArchiveFile, await readText(packetArchiveFile)) } : undefined;
  const candidates = files.filter((file) => /\.(png|jpe?g)$/i.test(file.name));
  const images = new Map(candidates.map((file) => [file.name, file]));
  const originals = files.filter((file) => /\.(html|css|js|ts)$/i.test(file.name));
  const bundles = files.filter((file) => /\.zip$/i.test(file.name));
  if (bundles.length > 1) throw new Error("Select one retained source ZIP; multiple bundles make the editable-source and font join ambiguous.");
  return { sourceMap: sourceMap.file, sourceMapText: sourceMap.text, dossier, dossierText, supportingDocs, manifest: manifest.file, manifestText: manifest.text, images, originals, bundles, packetArchive, all: files };
}

function slugify(value: string): string {
  return value.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "") || "scenario";
}

function acceptedSurfacePageIds(scenario: string, file: File, text: string): Set<string> {
  if (!/^notification hub$/i.test(scenario) || !/E42/i.test(file.name)) return new Set();
  return new Set(text.match(/\bNH16\.(?:4|5|6|8)\b/g) ?? []);
}

function supportingDocumentUse(scenario: string, file: File, text: string): string {
  const isE42 = /^notification hub$/i.test(scenario) && /E42/i.test(file.name);
  if (isE42 && /^\s*#\s*E42\b/im.test(text) && /^\s*-\s*Status:\s*accepted\s*$/im.test(text)) {
    return "E42 accepted Delivery detail surface record; applies only to its named NH16.4/.5/.6/.8 scenes, not whole Notification Hub acceptance";
  }
  if (isE42) return "Supporting E42 Delivery detail verification record; page links are limited to the NH16.4/.5/.6/.8 scenes it names";
  return "Supporting source document; exact bytes retained as context, not treated as product acceptance";
}

function scenarioNameFromDossier(file: File): string {
  if (/notification[-_ ]hub/i.test(file.name)) return "Notification Hub";
  const stem = file.name.replace(/\.[^.]+$/, "").replace(/(?:[-_.](?:dossier|brief|design))$/i, "");
  return stem.split(/[-_.]+/).filter(Boolean).map((word) => word[0]!.toUpperCase() + word.slice(1)).join(" ") || "Scenario";
}

function statePageMap(assets: SelectedDesignAsset[], imageSources: Map<string, string>) {
  const pages = new Map<string, PacketPage>();
  for (const asset of assets) {
    const page = pages.get(asset.sceneId) ?? { id: asset.sceneId, title: asset.sceneId, stateIds: [], sourceRefs: [], regions: [], editableSourceIds: [] };
    page.stateIds = [...new Set([...page.stateIds, ...asset.stateIds])];
    page.sourceRefs = [...new Set([...page.sourceRefs, ...(imageSources.get(asset.filename) ? [imageSources.get(asset.filename)!] : [])])];
    page.regions = [...new Set([...page.regions, asset.filename])];
    pages.set(asset.sceneId, page);
  }
  return [...pages.values()].sort((a, b) => a.id.localeCompare(b.id, undefined, { numeric: true }));
}

function classifyRegion(value: string): string {
  return value ? `Selected design frame (${value})` : "Full selected design frame";
}

function sourceFrameOptions(assets: SelectedDesignAsset[]): Array<{ key: string; asset: SelectedDesignAsset; stateId: string }> {
  return assets.flatMap((asset) => asset.stateIds.map((stateId) => ({
    key: JSON.stringify([asset.sceneId, stateId, asset.filename, asset.sha256]),
    asset,
    stateId,
  })));
}

function exactFrameKey(requirement: PacketRequirement, assets: SelectedDesignAsset[]): string {
  const filename = requirement.mapping.region?.match(/^Selected design frame \((.+)\)$/)?.[1];
  const asset = filename ? assets.find((candidate) => candidate.sceneId === requirement.mapping.pageId && candidate.filename === filename && candidate.stateIds.includes(requirement.mapping.stateId ?? "")) : null;
  return asset ? JSON.stringify([asset.sceneId, requirement.mapping.stateId, asset.filename, asset.sha256]) : "";
}

function authoringRevisionId(packet: DesignPacket, suffix: string): string {
  let ordinal = packet.revisions.length + 1;
  let candidate = `v${ordinal}-${suffix}`;
  while (packet.revisions.some((revision) => revision.id === candidate)) candidate = `v${++ordinal}-${suffix}`;
  return candidate;
}

function lastSection(value: string, heading: string): string {
  const sections = value.split(heading);
  return sections[sections.length - 1]?.trim() ?? "";
}

function selectedImageSourceIds(packet: DesignPacket, assets: SelectedDesignAsset[]): Set<string> {
  const identities = new Set(assets.map((asset) => `${asset.filename}\u0000${asset.sha256}`));
  return new Set(packet.sources.filter((source) => identities.has(`${source.name}\u0000${source.sha256}`)).map((source) => source.id));
}

export function applyBottomNavigationSource(source: string, embeddedFontDataUrl?: string): string {
  const sidebarClose = "</nav><main id=\"app\"></main></div>";
  const nav = "</nav><main id=\"app\"></main><nav class=\"bottom-nav\" aria-label=\"Primary navigation\"><button data-route=\"queue\">Needs you</button><button data-route=\"devices\">Reach</button><button data-route=\"quiet\">Quiet hours</button><button data-route=\"order\">Escalation</button><button data-route=\"integration\">Integrations</button></nav></div>";
  if (!source.includes(sidebarClose)) throw new Error("The selected editable prototype does not contain the retained five-destination shell; the recorded correction was not applied to an uncertain source.");
  let revised = source.replace(sidebarClose, nav);
  if (/url\(['"]?fonts\/SourceSans3\.ttf['"]?\)/i.test(revised)) {
    if (!embeddedFontDataUrl) throw new Error("The source uses a bundled font; include the retained ZIP so the revised source stays self-contained.");
    revised = revised.replace(/url\(['"]?fonts\/SourceSans3\.ttf['"]?\)/ig, "url('" + embeddedFontDataUrl + "')");
  }
  const mobileRule = "@media(max-width:700px){";
  if (!revised.includes(mobileRule)) throw new Error("The selected editable prototype has no known mobile breakpoint; the navigation amendment was not applied.");
  revised = revised.replace(mobileRule, ".bottom-nav{display:none}" + mobileRule);
  revised = revised.replace(mobileRule, `${mobileRule}.bottom-nav{display:grid;position:fixed;left:0;right:0;bottom:0;z-index:8;grid-template-columns:repeat(5,minmax(0,1fr));gap:2px;padding:4px;background:var(--surface);border-top:1px solid var(--line)}.bottom-nav button{min-width:0;min-height:48px;padding:4px 2px;border:0;font-size:12px;line-height:1.15}.content{padding-bottom:96px}`);
  return revised;
}

export async function makePacket(files: File[], initialSource?: { path: string; text: string }, scenarioOverride?: string): Promise<{ packet: DesignPacket; assets: SelectedDesignAsset[]; imageUrls: Map<string, string>; editableOriginal: EditableOriginal | null; packetArchiveRecovery?: PacketArchiveRecovery }> {
  const input = await guessInput(files);
  const { sourceMapText: mapText, dossierText, manifestText } = input;
  const scenarioName = scenarioOverride?.trim() || scenarioNameFromDossier(input.dossier);
  const isHub = /^notification hub$/i.test(scenarioName);
  if (input.packetArchive) {
    const archiveIntegrity = await verifyPacketIntegrity(input.packetArchive.packet);
    const nonLegacyIssues = archiveIntegrity.issues.filter((issue) => !/^sources\.[^.]+\.(?:id|identityBinding)$/.test(issue.path));
    if (nonLegacyIssues.length) throw new Error(`Retained packet archive failed model integrity outside legacy source identity fields: ${nonLegacyIssues.slice(0, 4).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
  }
  const rows = parseRequirementSourceMap(mapText);
  const sceneRows = parseSceneSourceMap(mapText);
  const assets = parseSelectedImageManifest(manifestText);
  if (!rows.length) throw new Error("No requirements were parsed from the source-map crosswalk.");
  if (!assets.length) throw new Error("The selected-image manifest is empty.");
  const sceneJoinIssues = validateSceneManifestJoins(sceneRows, assets);
  if (sceneJoinIssues.length) throw new Error(`Source map and selected-image manifest do not join exactly: ${sceneJoinIssues.slice(0, 5).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);

  const fileByName = new Map(input.all.map((file) => [file.name, file]));
  const imageSources = new Map<string, string>();
  const imageUrls = new Map<string, string>();
  const sourceRecords: PacketSource[] = [];
  const editableSourceIds: string[] = [];
  const sourceBundle = input.bundles[0];
  if (isHub && !sourceBundle) throw new Error("Notification Hub's recorded revision requires the retained editable source ZIP.");
  const zipEntries = sourceBundle ? await readZipEntries(sourceBundle) : new Map<string, Uint8Array>();
  const bundledFont = zipEntries.get("fonts/SourceSans3.ttf");
  if (isHub && !bundledFont) throw new Error("Notification Hub's retained source ZIP is missing fonts/SourceSans3.ttf.");
  const embeddedFontDataUrl = bundledFont ? bytesDataUrl(bundledFont, "font/ttf") : undefined;
  const mapRecord = await createSourceRecord({ name: input.sourceMap.name, origin: input.sourceMap.webkitRelativePath || input.sourceMap.name, version: "retained commissioning source map; hash-bound on import", bytes: await readBytes(input.sourceMap), mediaType: "text/markdown", use: "Requirement source, scene-state crosswalk and design-obligation inventory" });
  const manifestRecord = await createSourceRecord({ name: input.manifest.name, origin: input.manifest.webkitRelativePath || input.manifest.name, version: "retained selected-image-manifest.json", bytes: await readBytes(input.manifest), mediaType: "application/json", use: "Exact selected image membership, scene/state, theme, viewport and source hash index" });
  const dossierRecord = await createSourceRecord({ name: input.dossier.name, origin: input.dossier.webkitRelativePath || input.dossier.name, version: `retained ${scenarioName} brief/design source; exact imported bytes`, bytes: await readBytes(input.dossier), mediaType: "text/markdown", use: "Design brief and editable source reference; source material is not implementation acceptance" });
  sourceRecords.push(mapRecord, manifestRecord, dossierRecord);
  const supportingSourceRows = await Promise.all(input.supportingDocs.map(async (file) => {
    const bytes = await readBytes(file);
    const text = new TextDecoder().decode(bytes);
    const record = await createSourceRecord({
      name: file.name,
      origin: (file as ImportedFile).webkitRelativePath || file.name,
      version: "retained supporting source; exact imported bytes",
      bytes,
      mediaType: file.type || "text/markdown",
      use: supportingDocumentUse(scenarioName, file, text),
    });
    return { record, pageIds: acceptedSurfacePageIds(scenarioName, file, text) };
  }));
  sourceRecords.push(...supportingSourceRows.map((row) => row.record));
  const packetArchiveRecord = input.packetArchive ? await createSourceRecord({
    name: input.packetArchive.file.name,
    origin: (input.packetArchive.file as ImportedFile).webkitRelativePath || input.packetArchive.file.name,
    version: `byte-preserved prior packet ${input.packetArchive.packet.id} · ${input.packetArchive.packet.currentRevisionId}`,
    bytes: await readBytes(input.packetArchive.file),
    mediaType: input.packetArchive.file.type || "application/json",
    use: "Exact prior packet archive; fresh packet lineage is rebuilt from the selected retained source inputs",
    amendmentHistory: ["Re-imported without mutation; prior revisions remain archived byte-for-byte."],
  }) : undefined;
  if (packetArchiveRecord) sourceRecords.push(packetArchiveRecord);
  if (sourceBundle && bundledFont) {
    const fontSource = await createSourceRecord({ name: "SourceSans3.ttf", origin: `${sourceBundle.name}!/fonts/SourceSans3.ttf`, version: "font extracted in memory from the retained source ZIP", bytes: bundledFont, mediaType: "font/ttf", use: "Original font embedded into the revised standalone editable prototype" });
    sourceRecords.push(fontSource);
    editableSourceIds.push(fontSource.id);
  }

  const missing: string[] = [];
  for (const asset of assets) {
    const file = fileByName.get(asset.filename);
    if (!file) { missing.push(asset.filename); continue; }
    const bytes = await readBytes(file);
    const actualHash = await sha256Hex(bytes);
    if (actualHash !== asset.sha256 || file.size !== asset.bytes) throw new Error(`Selected original ${asset.filename} does not match its manifest hash/size.`);
    const record = await createSourceRecord({ name: file.name, origin: (file as ImportedFile).webkitRelativePath || file.name, version: `selectedRevision from manifest · ${asset.sceneId} · ${asset.stateIds.join(", ")}`, bytes, mediaType: file.type || asset.mediaType, use: "Retained visual design source; manifest hash verified; not a product acceptance claim" });
    sourceRecords.push(record);
    imageSources.set(file.name, record.id);
    imageUrls.set(file.name, `data:${file.type || "image/png"};base64,${await new Promise<string>((resolve, reject) => { const reader = new FileReader(); reader.onload = () => resolve(String(reader.result).split(",", 2)[1] ?? ""); reader.onerror = () => reject(new Error(`Could not read ${file.name}.`)); reader.readAsDataURL(file); })}`);
  }
  if (missing.length) throw new Error(`${missing.length} manifest images are missing. First missing file: ${missing[0]}`);

  let editableOriginal: EditableOriginal | null = null;
  const sourceOriginals = [...input.originals].sort((left, right) => {
    const priority = (file: File) => /^prototype\.html$/i.test(file.name) ? 0 : /(?:prototype.*v1|v1.*prototype|original.*prototype)/i.test(file.name) ? 1 : /\.html?$/i.test(file.name) && !/(?:v2|bottom-nav)/i.test(file.name) ? 2 : /\.html?$/i.test(file.name) ? 3 : 4;
    return priority(left) - priority(right) || left.name.localeCompare(right.name);
  });
  if (isHub && !sourceOriginals.some((file) => /^prototype\.html$/i.test(file.name))) {
    throw new Error("Notification Hub recovery requires the retained v1 prototype.html by its exact source filename; a later or similarly named HTML file cannot stand in for it.");
  }
  for (const file of sourceOriginals) {
    const origin = (file as ImportedFile).webkitRelativePath || (initialSource?.path && file.name === initialSource.path.split("/").pop() ? initialSource.path : file.name);
    const record = await createImportedEditableSourceRecord(file, await readBytes(file), origin, input.packetArchive?.packet.sources);
    sourceRecords.push(record);
    editableSourceIds.push(record.id);
    if (/\.html?$/i.test(file.name) && !editableOriginal) editableOriginal = { name: file.name, origin: (file as ImportedFile).webkitRelativePath || file.name, content: await readText(file), sourceId: record.id, embeddedFontDataUrl };
  }
  if (initialSource?.text && !input.originals.some((file) => file.name === initialSource.path.split("/").pop())) {
    const source = await createSourceRecord({ name: initialSource.path.split("/").pop() || "preview-source.html", origin: initialSource.path, version: "currently open Web Console preview; exact visible bytes", bytes: initialSource.text, mediaType: "text/html", use: "Editable source opened in the Web Console preview" });
    sourceRecords.push(source);
    editableSourceIds.push(source.id);
    if (/\.html$/i.test(source.name)) editableOriginal = { name: source.name, origin: source.origin, content: initialSource.text, sourceId: source.id, embeddedFontDataUrl };
  }
  for (const file of input.bundles) {
    sourceRecords.push(await createSourceRecord({ name: file.name, origin: (file as ImportedFile).webkitRelativePath || file.name, version: "retained editable source bundle; exact archive bytes", bytes: await readBytes(file), mediaType: "application/zip", use: "Editable original prototype/source bundle" }));
  }

  const pages = statePageMap(assets, imageSources);
  pages.forEach((page) => {
    const pageEvidenceSourceIds = supportingSourceRows.filter((row) => row.pageIds.has(page.id)).map((row) => row.record.id);
    page.sourceRefs = [...new Set([...page.sourceRefs, mapRecord.id, manifestRecord.id, dossierRecord.id, ...editableSourceIds, ...pageEvidenceSourceIds])];
    page.editableSourceIds = [...editableSourceIds];
  });
  const requirements: PacketRequirement[] = rows.map((row) => {
    // The crosswalk's first column carries requirement identity. NH-AC/NH-DEC
    // citations in the proof/source columns are evidence references and can be
    // reused by many requirements; treating them as requirement IDs creates
    // false many-to-many links. Resolve scene joins from explicit requirement
    // IDs and the crosswalk's exact selected state/page families instead.
    const sourceRequirementIds = sourceRequirementAliases(row);
    const archivedPacket = input.packetArchive?.packet;
    const archivedCurrent = archivedPacket ? archivedPacket.revisions.find((revision) => revision.id === archivedPacket.currentRevisionId) : undefined;
    const archivedSourceBaseline = archivedPacket?.revisions.find((revision) => revision.id === "v1") ?? archivedCurrent;
    // Use the reviewed current mapping to seed an otherwise unjoinable visual
    // requirement; it is recoverable author state, not a claim that v1 already
    // contained that mapping.
    const archivedMapping = archivedCurrent?.requirements.find((item) => item.id === row.id)?.mapping
      ?? archivedSourceBaseline?.requirements.find((item) => item.id === row.id)?.mapping;
    const archivedTrace = archivedSourceBaseline?.requirements.find((item) => item.id === row.id)?.sourceSceneTrace ?? [];
    const archivedTraceSceneIds = [...new Set(archivedTrace.map((trace) => trace.pageId))];
    const candidateScenes = archivedTraceSceneIds.length
      ? archivedTraceSceneIds.map((sceneId) => sceneRows.find((scene) => scene.sceneId === sceneId)).filter((scene): scene is NonNullable<typeof scene> => Boolean(scene))
      : sourceScenesForRequirement(row, sceneRows, archivedMapping);
    if (archivedTraceSceneIds.length && candidateScenes.length !== archivedTraceSceneIds.length) {
      throw new Error(`Retained source-scene trace for ${row.id} references a scene absent from the exact selected source map.`);
    }
    // Preserve every v1 source-scene trace above, but use an exact reviewed v2
    // page/state selection to seed the current mapping. Treating the full trace
    // set as the mapping candidate set makes every multi-scene requirement
    // appear ambiguous and discards the retained v2 frame decision on import.
    const retainedMappingScene = archivedMapping?.pageId
      ? sceneRows.find((scene) => scene.sceneId === archivedMapping.pageId)
      : undefined;
    const retainedMappingCanSeed = Boolean(
      retainedMappingScene
      && (!archivedTraceSceneIds.length || archivedTraceSceneIds.includes(retainedMappingScene.sceneId)),
    );
    const mappingScenes = retainedMappingCanSeed ? [retainedMappingScene!] : candidateScenes;
    const sourceJoin = mappingScenes.length === 1 ? mappingScenes[0] : undefined;
    const retainedStateCanSeed = Boolean(archivedMapping?.pageId && archivedMapping.stateId && sourceJoin?.sceneId === archivedMapping.pageId);
    const requestedStates = retainedStateCanSeed ? [archivedMapping!.stateId!] : row.selectedStates.length ? row.selectedStates : sourceJoin?.stateIds ?? [];
    const resolutions = requestedStates.map((reference) => resolveSelectedState(reference, assets, sourceJoin?.sceneId));
    const resolvedStates = new Map<string, { sceneId: string; stateId: string; assets: SelectedDesignAsset[]; reference: string }>();
    for (const [index, result] of resolutions.entries()) if (result.status === "resolved") resolvedStates.set(`${result.sceneId}\u0000${result.stateId}`, { ...result, reference: requestedStates[index] ?? result.stateId });
    const onlyResolvedState = resolvedStates.size === 1 && resolutions.length > 0 && resolutions.every((result) => result.status === "resolved") ? [...resolvedStates.values()][0] : undefined;
    const archivedFrameFilename = archivedMapping?.region?.match(/^Selected design frame \((.+)\)$/)?.[1];
    const archivedFrame = archivedFrameFilename && retainedStateCanSeed
      ? assets.find((asset) => asset.sceneId === archivedMapping!.pageId && asset.stateIds.includes(archivedMapping!.stateId!) && asset.filename === archivedFrameFilename)
      : undefined;
    const selected = archivedFrameFilename
      ? archivedFrame
      : mappingScenes.length <= 1 && onlyResolvedState?.assets.length === 1 ? onlyResolvedState.assets[0] : undefined;
    const resolvedSceneId = mappingScenes.length > 1 ? null : onlyResolvedState?.sceneId ?? null;
    const resolvedStateId = mappingScenes.length > 1 ? null : onlyResolvedState?.stateId ?? null;
    const page = resolvedSceneId ? pages.find((item) => item.id === resolvedSceneId) : undefined;
    const sourceSceneTrace = candidateScenes.map((scene) => {
      const tracePage = pages.find((item) => item.id === scene.sceneId);
      if (!tracePage) return null;
      const sourceImages = assets.filter((asset) => asset.sceneId === scene.sceneId).map((asset) => ({
        filename: asset.filename,
        sha256: asset.sha256,
        sourceId: imageSources.get(asset.filename) ?? "",
        stateIds: [...asset.stateIds],
      }));
      return {
        pageId: tracePage.id,
        stateIds: [...tracePage.stateIds],
        obligationId: sourceSceneObligationId(scene.sceneId),
        sourceMapSourceId: mapRecord.id,
        manifestSourceId: manifestRecord.id,
        sourceIds: [...tracePage.sourceRefs],
        editableSourceIds: [...tracePage.editableSourceIds],
        sourceImages,
      };
    }).filter((trace): trace is NonNullable<typeof trace> => Boolean(trace));
    const explicitlyDeferred = /\bP2 future requirement\b/i.test(row.scope);
    const coverage = row.coverage === "nonvisual" || (explicitlyDeferred && !resolvedSceneId) ? "nonvisual" : row.coverage === "unknown" && !resolvedSceneId ? "unknown" : "visual";
    const sourceForFrame = selected ? sourceRecords.find((source) => source.name === selected.filename && source.sha256 === selected.sha256) : undefined;
    const mappingRationale = selected
      ? `Source-map state family ${onlyResolvedState?.reference}; selected-image manifest binds ${selected.filename} (${selected.sha256}). This is a design-source mapping only.`
      : onlyResolvedState && onlyResolvedState.assets.length > 1
        ? `Source-map state ${onlyResolvedState.stateId} resolves to ${onlyResolvedState.assets.length} exact source frames; choose one frame to complete the mapping.`
        : mappingScenes.length > 1
          ? `Requirement links to ${mappingScenes.length} source scenes; choose one exact scene and frame.`
          : resolvedSceneId && resolvedStateId
            ? `Source-map state ${resolvedStateId} resolves to ${resolvedSceneId}; choose one exact selected-image frame.`
            : explicitlyDeferred
              ? "Source map marks this future requirement as not selected in the current scope; the packet retains it as deferred."
              : coverage === "nonvisual"
                ? "Source map identifies the proof surface as nonvisual; this packet does not invent a screen."
                : "No unique selected scene/state/frame was resolved; choose an exact mapping or explicitly classify the requirement as nonvisual.";
    return {
      id: row.id,
      text: row.text,
      scope: row.scope,
      coverage,
      mapping: { pageId: page?.id ?? null, stateId: resolvedStateId, region: selected ? classifyRegion(selected.filename) : null, rationale: mappingRationale, sourceId: sourceForFrame?.id ?? null },
      sourceRefs: [...new Set([mapRecord.id, manifestRecord.id, ...(sourceForFrame ? [sourceForFrame.id] : []), ...sourceSceneTrace.flatMap((trace) => trace.sourceIds), ...supportingSourceRows.filter((sourceRow) => sourceRow.pageIds.has(page?.id ?? "")).map((sourceRow) => sourceRow.record.id)])],
      evidenceRefs: row.evidenceRefs,
      disposition: explicitlyDeferred ? "defer" : coverage === "unknown" ? "unknown" : "preserve",
      sourceRequirementIds,
      sourceMapRefs: [...row.sourceMapRefs],
      sourceSceneTrace,
      sourceStateText: row.selectedStateText,
      proofBasisText: row.proofBasisText,
      behaviorProofStatus: "not-verified",
    };
  });
  const features = sceneRows.map((scene) => {
    const page = pages.find((item) => item.id === scene.sceneId);
    const isPresent = Boolean(page);
    const sourceImages = page ? assets.filter((asset) => asset.sceneId === page.id).map((asset) => ({ filename: asset.filename, sha256: asset.sha256, sourceId: imageSources.get(asset.filename) ?? "", stateIds: [...asset.stateIds] })) : [];
    const sourceRequirementIds = [...scene.requirementIds];
    const linkedRequirementIds = requirements.filter((item) => item.sourceSceneTrace?.some((trace) => trace.pageId === scene.sceneId)).map((item) => item.id);
    const unresolvedSourceRequirementIds = sourceRequirementIds.filter((id) => /^(?:NOTIFICA|OT)-P[012]-\d{3}$/.test(id) && !requirements.some((item) => (item.sourceRequirementIds ?? []).includes(id)));
    return {
      id: sourceSceneObligationId(scene.sceneId),
      name: scene.obligations || `Source-map scene ${scene.sceneId}`,
      sourceRef: mapRecord.id,
      requirementIds: linkedRequirementIds,
      sourceRequirementIds,
      unresolvedSourceRequirementIds,
      pageId: page?.id,
      stateIds: [...(page?.stateIds ?? [])],
      relatedSourceIds: [...(page?.sourceRefs ?? [mapRecord.id, manifestRecord.id])],
      sourceImages,
      disposition: isPresent ? "preserve" as const : "unknown" as const,
      rationale: isPresent ? `This source-map Appendix A obligation maps to exact page ${page!.id}, states ${page!.stateIds.join(", ")}, and ${page!.regions.length} selected images with verified source hashes. Source presence does not establish product behavior or acceptance.` : "This source-map Appendix A obligation has no selected manifest image. An author must assign preserve, remove, defer or unknown before export.",
    };
  });
  for (const page of pages) if (!sceneRows.some((scene) => scene.sceneId === page.id)) {
    features.push({
      id: sourceSceneObligationId(page.id),
      name: `Selected manifest scene ${page.id} has no Appendix A obligation row`,
      sourceRef: mapRecord.id,
      requirementIds: requirements.filter((item) => item.sourceSceneTrace?.some((trace) => trace.pageId === page.id)).map((item) => item.id),
      sourceRequirementIds: [],
      unresolvedSourceRequirementIds: [],
      pageId: page.id,
      stateIds: [...page.stateIds],
      relatedSourceIds: [...page.sourceRefs],
      sourceImages: assets.filter((asset) => asset.sceneId === page.id).map((asset) => ({ filename: asset.filename, sha256: asset.sha256, sourceId: imageSources.get(asset.filename) ?? "", stateIds: [...asset.stateIds] })),
      disposition: "unknown" as const,
      rationale: "The selected-image manifest contains a scene absent from the source-map Appendix A inventory; reconcile it before export.",
    });
  }
  if (input.packetArchive) {
    const sourceIdentity = await verifySourceReimportIdentity(
      input.packetArchive.packet.sources,
      sourceRecords.filter((source) => source !== packetArchiveRecord),
      { allowAdditionalImportedSources: input.supportingDocs.length > 0 },
    );
    if (!sourceIdentity.valid) throw new Error(`Retained packet source identity re-import failed closed: ${sourceIdentity.issues.slice(0, 5).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
    assertPacketArchiveMatchesInputs(input.packetArchive.packet, scenarioName, requirements, features);
  }
  if (packetArchiveRecord) {
    for (const page of pages) page.sourceRefs = [...new Set([...page.sourceRefs, packetArchiveRecord.id])];
    for (const requirement of requirements) requirement.sourceRefs = [...new Set([...requirement.sourceRefs, packetArchiveRecord.id])];
  }
  const boundary = isHub
    ? "Authoring boundary: source material only. Historical E42 evidence applies only to the accepted Delivery detail surface; this packet does not imply whole-product acceptance."
    : "Authoring boundary: source material only. This packet does not imply a served implementation or product acceptance.";
  const archiveNote = input.packetArchive && packetArchiveRecord
    ? `\n\nSource identity recovery archived the exact prior packet bytes as ${packetArchiveRecord.id} (${packetArchiveRecord.sha256}). This new packet is rebuilt from the retained source inputs; the archived packet and its v1/v2 JSON bytes remain unchanged.`
    : "";
  const brief = `${dossierText}\n\n${boundary}${archiveNote}`;
  const recoverySuffix = packetArchiveRecord ? `-rebound-${packetArchiveRecord.sha256.slice(0, 8)}` : "";
  const packet = await createDesignPacket({ id: `${slugify(scenarioName)}-${mapRecord.sha256.slice(0, 8)}${recoverySuffix}`, scenario: scenarioName, brief, sources: sourceRecords, pages, requirements, features, createdAt: new Date().toISOString(), traceabilitySchema: TRACEABILITY_CONTRACT_SCHEMA });
  const recovery = packetArchiveRecord && input.packetArchive ? { sourceId: packetArchiveRecord.id, packetId: input.packetArchive.packet.id, preservedCurrentRevision: input.packetArchive.packet.currentRevisionId, archiveSha256: packetArchiveRecord.sha256 } : undefined;
  return { packet, assets, imageUrls, editableOriginal, packetArchiveRecovery: recovery };
}

export function atlasPage(packet: DesignPacket, sceneId: string, assets: SelectedDesignAsset[], imageUrls: Map<string, string>): string {
  const revision = packet.revisions.find((item) => item.id === packet.currentRevisionId)!;
  const page = revision.pages.find((item) => item.id === sceneId)!;
  const mapped = revision.requirements.filter((item) => item.mapping.pageId === page.id);
  const images = assets.filter((asset) => asset.sceneId === sceneId).map((asset) => `<figure><img src="${imageUrls.get(asset.filename) ?? ""}" alt="${escapeHtml(asset.caption || asset.filename)}"><figcaption><b>${escapeHtml(asset.filename)}</b> · ${escapeHtml(asset.stateIds.join(", "))} · ${escapeHtml(asset.sha256)}<br>${escapeHtml(asset.caption)}</figcaption></figure>`).join("");
  const reqs = mapped.map((item) => `<li><b>${escapeHtml(item.id)}</b> — ${escapeHtml(item.mapping.stateId || "unmapped")} / ${escapeHtml(item.mapping.region || "no region")} · ${escapeHtml(item.disposition)}<br>${escapeHtml(item.text)}<small>Source evidence only; not behavioral proof.</small></li>`).join("");
  return `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${escapeHtml(page.title)} · ${escapeHtml(revision.id)}</title><style>body{font:16px/1.5 system-ui,sans-serif;max-width:1200px;margin:auto;padding:24px;color:#17242e}header,section,figure{border:1px solid #becbd4;border-radius:10px;padding:16px;margin:16px 0}figure{display:inline-block;vertical-align:top;width:min(390px,100%);box-sizing:border-box}img{display:block;width:100%;height:auto;background:#eaf0f4}figcaption{overflow-wrap:anywhere;font-size:13px;margin-top:10px}small{display:block;color:#455564}a{color:#075985}@media print{body{max-width:none}figure{break-inside:avoid}}</style></head><body><header><p>${escapeHtml(packet.scenario)} · ${escapeHtml(revision.id)} · page ${escapeHtml(page.id)}</p><h1>${escapeHtml(page.title)}</h1><p>Page/state: ${escapeHtml(page.stateIds.join(", "))}. Source-bound design artifact, not an implementation or acceptance record.</p></header><section><h2>Selected original screens (${assets.filter(a => a.sceneId === page.id).length})</h2>${images}</section><section><h2>Mapped requirements</h2><ul>${reqs || "<li>No crosswalk requirement resolved to this scene.</li>"}</ul></section><section><h2>Revision lineage</h2><p>Packet revision <code>${escapeHtml(revision.id)} · ${revision.contentSha256}</code></p><p>${escapeHtml(revision.changeSummary)}</p>${(revision.decisionRefs ?? []).map(d=>`<blockquote>${escapeHtml(d.quote)}<br>${escapeHtml(d.id)} · ${escapeHtml(d.sourceTimestamp)} · applies to ${escapeHtml(d.appliesToVersion)}</blockquote>`).join("")}</section></body></html>`;
}

export function buildImplementerHandoff(packet: DesignPacket): string {
  const revision = packet.revisions.find((item) => item.id === packet.currentRevisionId);
  if (!revision) throw new Error("The current packet revision is missing.");
  const cell = (value: unknown) => String(value ?? "—").replace(/\|/g, "\\|").replace(/\r?\n/g, "<br>");
  const requirementRows = revision.requirements.map((item) => {
    const traces = (item.sourceSceneTrace ?? []).map((trace) => `${trace.pageId}[${trace.stateIds.join(",")}] / ${trace.obligationId}`).join("; ") || "no visual scene; see source proof basis";
    const sourceIds = [...new Set([...(item.sourceRefs ?? []), ...(item.sourceSceneTrace ?? []).flatMap((trace) => trace.sourceIds), ...(item.sourceSceneTrace ?? []).flatMap((trace) => trace.editableSourceIds)])];
    const frame = item.mapping.sourceId ? `${item.mapping.pageId}/${item.mapping.stateId}/${item.mapping.region} (${item.mapping.sourceId})` : "no single visual frame";
    return `| ${cell(item.id)} | ${cell(item.coverage)} | ${cell(item.sourceRequirementIds?.join(", "))} | ${cell(traces)} | ${cell(frame)} | ${cell(item.disposition)} | ${cell(item.behaviorProofStatus)} | ${cell(item.proofBasisText)} | ${cell(sourceIds.join(", "))} |`;
  }).join("\n");
  const omissionRows = revision.features.map((item) => {
    const images = (item.sourceImages ?? []).map((image) => `${image.filename}#${image.sha256}@${image.sourceId}[${image.stateIds.join(",")}]`).join("; ");
    return `| ${cell(item.id)} | ${cell(item.pageId)} | ${cell(item.stateIds?.join(", "))} | ${cell(item.sourceRequirementIds?.join(", "))} | ${cell(item.requirementIds.join(", "))} | ${cell(item.relatedSourceIds?.join(", "))} | ${cell(images)} | ${cell(item.disposition)} | ${cell(item.rationale)} |`;
  }).join("\n");
  return `# Fresh implementer handoff — ${packet.scenario}\n\nPacket: ${packet.id} · revision: ${revision.id} · SHA-256: ${revision.contentSha256}. Trace contract: ${packet.traceabilitySchema ?? "legacy"}.\n\n## Scope and verification boundary\n\nUse this retained packet, its exact editable source, and linked evidence to make a separate local design prototype. Do not edit or serve the existing ${packet.scenario} product. The design workflow verified source bytes, image hashes, and lineage only. **Behavior proof is not verified for any requirement.** The source-map proof/status text below is copied as context; it is not test execution, runtime behavior, or acceptance. For each requirement, independently record expected and observed behavior only when an authorized executable proof surface is available; otherwise retain status as unverified.\n\n## Requirement-to-source trace (${revision.requirements.length})\n\n| Requirement | Coverage | Source requirement IDs | Exact source scene/page/state/obligation links | Representative page/state/frame source ID | Disposition | Behavior proof | Source-map proof basis | Exact source IDs |\n|---|---|---|---|---|---|---|---|---|\n${requirementRows}\n\n## Source-scene and omission register (${revision.features.length})\n\n| Obligation ID | Atlas page | States | Source requirement IDs | Linked packet requirements | Exact source IDs | Selected filename/hash/source ID/state | Disposition | Basis |\n|---|---|---|---|---|---|---|---|---|\n${omissionRows}\n\n## Implementer comparison record\n\nFor every requirement and scene obligation, record the revised source ID, implemented route/state or named nonvisual proof surface, expected and observed behavior, exact artifact/build identity, test or runtime evidence, result, time and rework. Keep design/source trace separate from behavior proof. Report missing proof authority as unverified; do not invent behavior or silently omit a row. V1 remains unchanged; the correction reference in ${revision.id} applies to its exact ancestor and names every affected atlas page.\n\n## Reuse path\n\nIn Web Console File Preview, open another scenario's retained brief, source map with a Requirement crosswalk and Appendix A scene table, selected-image manifest, referenced originals, and editable source. Launch **Design packet**, validate source hashes and scene joins, resolve every requirement trace and omission, version any operator-authorized correction against its exact parent, then export the same packet, atlas, PDF, trace CSVs and handoff. Only scenario inputs and version decisions change.\n`;
}

interface ToolProps extends DesignAuthoringToolProps { onClose: () => void; }

export function DesignAuthoringTool({ sessionId, onOpenArtifact, initialSource, onClose }: ToolProps) {
  const fileInput = useRef<HTMLInputElement>(null);
  const importedFiles = useRef<File[] | null>(null);
  const [files, setFiles] = useState<File[]>([]);
  const [scenarioName, setScenarioName] = useState("");
  const [packet, setPacket] = useState<DesignPacket | null>(null);
  const [packetArchiveRecovery, setPacketArchiveRecovery] = useState<PacketArchiveRecovery | null>(null);
  const [requirementDrafts, setRequirementDrafts] = useState<PacketRequirement[]>([]);
  const [featureDrafts, setFeatureDrafts] = useState<PacketFeature[]>([]);
  const [reviewSummary, setReviewSummary] = useState("Recorded source mapping and omission review.");
  const [assets, setAssets] = useState<SelectedDesignAsset[]>([]);
  const [imageUrls, setImageUrls] = useState<Map<string, string>>(new Map());
  const [editableOriginal, setEditableOriginal] = useState<EditableOriginal | null>(null);
  const [editableSource, setEditableSource] = useState("");
  const [revisedSource, setRevisedSource] = useState<string | null>(null);
  const [revisedSourceRevisionId, setRevisedSourceRevisionId] = useState("");
  const [revisedSourceId, setRevisedSourceId] = useState<string | null>(null);
  const [sourceEditsPending, setSourceEditsPending] = useState(false);
  const [revisionSummary, setRevisionSummary] = useState("Author a versioned design revision from the retained source.");
  const [artifactPaths, setArtifactPaths] = useState<{ atlas: string; original: string; revised: string; handoff: string } | null>(null);
  const [localExportReady, setLocalExportReady] = useState(false);
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [activeStep, setActiveStep] = useState(0);
  const [selectedSceneId, setSelectedSceneId] = useState("");
  const current = useMemo(() => packet?.revisions.find((revision) => revision.id === packet.currentRevisionId) ?? null, [packet]);
  const isHubPacket = packet?.scenario === "Notification Hub";
  const reviewDirty = Boolean(current && (JSON.stringify(requirementDrafts) !== JSON.stringify(current.requirements) || JSON.stringify(featureDrafts) !== JSON.stringify(current.features)));
  const draftPacket = useMemo(() => {
    if (!packet || !current || !requirementDrafts.length || !featureDrafts.length) return packet;
    return { ...packet, revisions: packet.revisions.map((revision) => revision.id === current.id ? { ...revision, requirements: requirementDrafts, features: featureDrafts } : revision) };
  }, [packet, current, requirementDrafts, featureDrafts]);
  const validation = useMemo(() => draftPacket ? validatePacket(draftPacket) : null, [draftPacket]);
  const linkedCorrection = current?.decisionRefs?.some((decision) => decision.id === CORRECTION.id) ?? false;
  const scenes = useMemo(() => current?.pages ?? [], [current]);
  const selectedScene = scenes.find((scene) => scene.id === selectedSceneId) ?? scenes[0] ?? null;
  const selectedSceneAssets = selectedScene ? assets.filter((asset) => asset.sceneId === selectedScene.id) : [];
  const frameOptions = useMemo(() => sourceFrameOptions(assets), [assets]);
  const stepNames = ["Inputs & lineage", "Mappings & omissions", "Inspect atlas", "Edit & revise", "Export & hand off"];

  const importSources = async () => {
    setBusy(true); setError(""); setStatus("");
    try {
      const result = await makePacket(files, initialSource, scenarioName || undefined);
      const integrity = await verifyPacketIntegrity(result.packet, { allowUnresolvedMappings: true });
      if (!integrity.valid) throw new Error(`Imported packet failed ${integrity.issues.length} identity check(s): ${integrity.issues.slice(0, 3).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
      setPacket(result.packet); setPacketArchiveRecovery(result.packetArchiveRecovery ?? null); setAssets(result.assets); setImageUrls(result.imageUrls);
      importedFiles.current = files;
      setRequirementDrafts(result.packet.revisions[0]!.requirements);
      setFeatureDrafts(result.packet.revisions[0]!.features);
      setEditableOriginal(result.editableOriginal); setEditableSource(result.editableOriginal?.content ?? ""); setRevisedSource(null); setRevisedSourceRevisionId(""); setRevisedSourceId(null); setSourceEditsPending(false);
      setScenarioName(result.packet.scenario);
      setArtifactPaths(null);
      setReviewSummary("Recorded source mapping and omission review.");
      setSelectedSceneId(result.packet.revisions[0]!.pages[0]?.id ?? "");
      setActiveStep(1);
      const initialRequirements = result.packet.revisions[0]!.requirements;
      const exactlyMapped = initialRequirements.filter((item) => item.coverage === "visual" && item.mapping.pageId && item.mapping.stateId && item.mapping.region || item.coverage === "nonvisual" && item.disposition !== "unknown").length;
      const unresolved = initialRequirements.length - exactlyMapped;
      const recoveryStatus = result.packetArchiveRecovery ? ` Archived prior packet ${result.packetArchiveRecovery.packetId} byte-for-byte as a source record; this new packet starts at v1 with current source identities. Apply the existing bottom-navigation correction to create its linked v2.` : "";
      setStatus(`Imported ${result.assets.length} hash-checked visual sources across ${result.packet.revisions[0]!.pages.length} scenes. ${exactlyMapped} requirements have an exact scene/frame or explicit nonvisual disposition; ${unresolved} need review before complete export.${recoveryStatus}`);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not import the source packet."); }
    finally { setBusy(false); }
  };

  const updateRequirement = (requirementId: string, update: (item: PacketRequirement) => PacketRequirement) => {
    setRequirementDrafts((items) => items.map((item) => item.id === requirementId ? update(item) : item));
  };

  const updateFeature = (featureId: string, update: (item: PacketFeature) => PacketFeature) => {
    setFeatureDrafts((items) => items.map((item) => item.id === featureId ? update(item) : item));
  };

  const setRequirementDisposition = (requirementId: string, disposition: OmissionDisposition) => updateRequirement(requirementId, (item) => ({
    ...item,
    disposition,
    mapping: disposition === item.disposition ? item.mapping : { ...item.mapping, rationale: `${item.mapping.rationale}\nDisposition rationale: ` },
  }));

  const setRequirementRationale = (requirementId: string, rationale: string) => updateRequirement(requirementId, (item) => ({ ...item, mapping: { ...item.mapping, rationale } }));

  const setFeatureDisposition = (featureId: string, disposition: OmissionDisposition) => updateFeature(featureId, (item) => ({
    ...item,
    disposition,
    rationale: disposition === item.disposition ? item.rationale : `${item.rationale}\nDisposition rationale: `,
  }));

  const setFeatureRationale = (featureId: string, rationale: string) => updateFeature(featureId, (item) => ({ ...item, rationale }));

  const selectRequirementFrame = (requirementId: string, encoded: string) => {
    if (!packet || !current || !encoded) return;
    try {
      const [sceneId, stateId, filename, sha256] = JSON.parse(encoded) as [string, string, string, string];
      const asset = assets.find((item) => item.sceneId === sceneId && item.stateIds.includes(stateId) && item.filename === filename && item.sha256 === sha256);
      const pageMatches = current.pages.filter((item) => item.id === sceneId && item.stateIds.includes(stateId));
      const sourceMatches = packet.sources.filter((item) => item.name === filename && item.sha256 === sha256);
      const source = sourceMatches[0];
      if (!asset || pageMatches.length !== 1 || sourceMatches.length !== 1 || !source || !current.sourceIds.includes(source.id) || !pageMatches[0]!.sourceRefs.includes(source.id)) {
        throw new Error("This selected frame no longer has one exact scene, state, image hash and retained source join. Re-import the matching source packet.");
      }
      const imageSourceIds = selectedImageSourceIds(packet, assets);
      updateRequirement(requirementId, (item) => ({
        ...item,
        coverage: "visual",
        mapping: {
          pageId: sceneId,
          stateId,
          region: classifyRegion(filename),
          rationale: `Author selected ${sceneId} / ${stateId}; selected-image manifest binds ${filename} (${sha256}). This is a design-source mapping only.`,
          sourceId: source.id,
        },
        sourceRefs: [...new Set([...item.sourceRefs.filter((id) => !imageSourceIds.has(id)), source.id])],
      }));
      setSelectedSceneId(sceneId);
      setError("");
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Could not resolve this source frame.");
    }
  };

  const updateRequirementCoverage = (requirementId: string, coverage: CoverageKind) => {
    const imageSourceIds = packet ? selectedImageSourceIds(packet, assets) : new Set<string>();
    updateRequirement(requirementId, (item) => ({
      ...item,
      coverage,
      sourceRefs: coverage === "visual" ? item.sourceRefs : item.sourceRefs.filter((id) => !imageSourceIds.has(id)),
      mapping: coverage === "visual"
        ? { ...item.mapping, rationale: "Select one exact page, state and manifest-bound source frame before export." }
        : { pageId: null, stateId: null, region: null, rationale: coverage === "nonvisual" ? "Author-confirmed nonvisual proof surface.\nNonvisual proof basis: " : "Unresolved coverage; choose visual mapping or confirm a nonvisual proof surface." },
      disposition: coverage === "unknown" ? "unknown" : item.disposition === "unknown" ? "preserve" : item.disposition,
    }));
  };

  const draftReviewIssues = useMemo(() => {
    if (!current) return [] as string[];
    const issues: string[] = [];
    for (const item of requirementDrafts) {
      const original = current.requirements.find((candidate) => candidate.id === item.id);
      if (!original) { issues.push(`${item.id}: source requirement ID is stale.`); continue; }
      const changed = JSON.stringify(item) !== JSON.stringify(original);
      if (changed && !item.mapping.rationale.trim()) issues.push(`${item.id}: add the source basis for this mapping or disposition change.`);
      if (item.disposition !== original.disposition && !lastSection(item.mapping.rationale, "Disposition rationale:")) issues.push(`${item.id}: explain the changed packet disposition.`);
      if (item.coverage !== original.coverage && item.coverage === "nonvisual" && !lastSection(item.mapping.rationale, "Nonvisual proof basis:")) issues.push(`${item.id}: explain why this requirement has no visual design surface.`);
    }
    for (const item of featureDrafts) {
      const original = current.features.find((candidate) => candidate.id === item.id);
      if (!original) { issues.push(`${item.id}: source obligation ID is stale.`); continue; }
      if (item.disposition !== original.disposition && !lastSection(item.rationale, "Disposition rationale:")) issues.push(`${item.id}: explain the changed obligation disposition.`);
      if (item.disposition !== "preserve" && !item.rationale.trim()) issues.push(`${item.id}: add a rationale for this omission disposition.`);
    }
    return issues;
  }, [current, requirementDrafts, featureDrafts]);
  const revisedSourceRecord = packet?.sources.find((source) => source.id === revisedSourceId) ?? null;
  const sourceInputsCurrent = importedFiles.current === files;
  const exportReady = Boolean(packet && current && sourceInputsCurrent && validation?.valid && !draftReviewIssues.length && !reviewDirty && !sourceEditsPending && editableOriginal && revisedSource && revisedSourceRecord && (!isHubPacket || linkedCorrection));

  const recordReviewRevision = async () => {
    if (!packet || !current || !reviewDirty) return;
    if (isHubPacket && !linkedCorrection) {
      setError("For Notification Hub, mapping and omission edits will be included with the existing version-bound correction. Apply that correction before creating a separate review revision.");
      return;
    }
    setBusy(true); setError("");
    try {
      if (!reviewSummary.trim()) throw new Error("Describe the mapping and omission review before creating a revision.");
      if (draftReviewIssues.length) throw new Error(`Review is incomplete: ${draftReviewIssues.slice(0, 3).join("; ")}`);
      const sourceIntegrity = await verifyPacketIntegrity(packet, { allowUnresolvedMappings: true });
      if (!sourceIntegrity.valid) throw new Error(`The retained packet has ${sourceIntegrity.issues.length} integrity issue(s): ${sourceIntegrity.issues.slice(0, 3).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
      if (!validation?.valid) throw new Error(`Review is blocked by ${validation?.issues.length ?? 0} mapping or omission issue(s): ${validation?.issues.slice(0, 3).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
      const revisionId = authoringRevisionId(packet, "review");
      const revised = await addPacketRevision(packet, { revisionId, createdAt: new Date().toISOString(), changeSummary: reviewSummary.trim(), requirements: requirementDrafts, features: featureDrafts, sourceIds: current.sourceIds });
      const revision = revised.revisions.find((item) => item.id === revisionId)!;
      setPacket(revised); setRequirementDrafts(revision.requirements); setFeatureDrafts(revision.features); setArtifactPaths(null);
      setStatus(`Recorded an immutable ${revisionId} review revision from ${current.id}. All prior mappings, obligations and correction lineage remain in packet history.`);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not record the source review revision."); }
    finally { setBusy(false); }
  };

  const applyCorrection = async () => {
    if (!packet || !current) return;
    setBusy(true); setError("");
    try {
      if (reviewDirty && !reviewSummary.trim()) throw new Error("Describe the mapping and omission changes before creating their version-bound correction revision.");
      if (draftReviewIssues.length) throw new Error(`Finish the staged mapping and omission review first: ${draftReviewIssues.slice(0, 3).join("; ")}`);
      if (!validation?.valid) throw new Error(`The correction revision is blocked by ${validation?.issues.length ?? 0} unresolved mapping or omission issue(s): ${validation?.issues.slice(0, 3).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
      const sourceIntegrity = await verifyPacketIntegrity(packet, { allowUnresolvedMappings: true });
      if (!sourceIntegrity.valid) throw new Error(`The retained packet has ${sourceIntegrity.issues.length} integrity issue(s): ${sourceIntegrity.issues.slice(0, 3).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
      const nextRequirements = requirementDrafts.map((item) => ({ ...item }));
      if (!editableOriginal) throw new Error("Import the retained editable prototype.html source before applying this source-bound correction.");
      const amendedHtml = applyBottomNavigationSource(editableSource || editableOriginal.content, editableOriginal.embeddedFontDataUrl);
      const amendment = await createSourceRecord({ name: `${editableOriginal.name.replace(/\.html$/i, "")}-v2-bottom-nav.html`, origin: editableOriginal.origin, version: `authored revision from ${current.id} · correction ${CORRECTION.id}`, bytes: amendedHtml, mediaType: "text/html", use: "Editable source amendment linked to the exact operator correction", amendmentHistory: [`Applied ${CORRECTION.id} to ${current.id}; original source remains retained.`] });
      // Recovery can include the exact v2 source as a retained input. Reuse
      // its identical identity instead of appending a duplicate source ID.
      const amendmentSource = packet.sources.find((source) => source.id === amendment.id) ?? amendment;
      const affectedPageIds = current.pages.map((page) => page.id);
      const nextPages = current.pages.map((page) => ({
        ...page,
        designRegions: [...new Set([...(page.designRegions ?? []), "persistent primary navigation: History, Delivery, Settings / Reach, and all five destinations"])],
        sourceRefs: [...new Set([...page.sourceRefs, amendmentSource.id])],
        editableSourceIds: [...new Set([...page.editableSourceIds, amendmentSource.id])],
      }));
      const nextRequirementsWithRevisionSource = nextRequirements.map((item) => ({
        ...item,
        sourceRefs: [...new Set([...item.sourceRefs, ...(item.sourceSceneTrace ?? []).flatMap((trace) => affectedPageIds.includes(trace.pageId) ? [amendmentSource.id] : [])])],
        sourceSceneTrace: (item.sourceSceneTrace ?? []).map((trace) => affectedPageIds.includes(trace.pageId) ? {
          ...trace,
          sourceIds: [...new Set([...trace.sourceIds, amendmentSource.id])],
          editableSourceIds: [...new Set([...trace.editableSourceIds, amendmentSource.id])],
        } : trace),
      }));
      const nextFeatures = featureDrafts.map((item) => ({
        ...item,
        relatedSourceIds: item.pageId && affectedPageIds.includes(item.pageId) ? [...new Set([...(item.relatedSourceIds ?? []), amendmentSource.id])] : item.relatedSourceIds,
      }));
      const nextPacket = { ...packet, sources: packet.sources.some((source) => source.id === amendmentSource.id) ? packet.sources : [...packet.sources, amendmentSource] };
      const reviewSuffix = reviewDirty ? ` Included review: ${reviewSummary.trim()}.` : "";
      const revised = await addPacketRevision(nextPacket, { revisionId: "v2-bottom-nav", createdAt: new Date().toISOString(), changeSummary: `Revision amends the editable prototype source to retain primary mobile navigation in the five destinations; all ${affectedPageIds.length} atlas pages link the exact amended source while requirement mappings and omission dispositions remain preserved. V1 source and packet remain unchanged.${reviewSuffix}`, sourceIds: [...new Set([...current.sourceIds, amendmentSource.id])], pages: nextPages, requirements: nextRequirementsWithRevisionSource, features: nextFeatures, decisionRefs: [{ ...CORRECTION, appliesToVersion: current.id, affectedPageIds, scopeDescription: `Shared editable mobile-navigation shell applied to all ${affectedPageIds.length} atlas pages; requirement dispositions and behavior acceptance are unchanged.` }] });
      setRevisedSource(amendedHtml);
      const revision = revised.revisions.find((item) => item.id === revised.currentRevisionId)!;
      setPacket(revised); setRequirementDrafts(revision.requirements); setFeatureDrafts(revision.features); setRevisedSourceRevisionId("v2"); setRevisedSourceId(amendmentSource.id); setSourceEditsPending(false); setStatus("Created immutable v2 linked to operator decision NH-DESIGN-20261025-01 and its exact v1 parent. This is a design revision, not operator acceptance of a served build.");
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not create the linked revision."); }
    finally { setBusy(false); }
  };

  const createEditableRevision = async () => {
    if (!packet || !current || !editableOriginal) return;
    setBusy(true); setError("");
    try {
      if (editableSource === (revisedSource ?? editableOriginal.content)) throw new Error("Edit the current source revision before creating a new immutable revision.");
      const changeSummary = revisionSummary.trim();
      if (!changeSummary) throw new Error("Describe the source change before creating a revision.");
      if (reviewDirty && !reviewSummary.trim()) throw new Error("Describe the mapping and omission changes before combining them with this editable-source revision.");
      if (draftReviewIssues.length) throw new Error(`Finish the staged mapping and omission review first: ${draftReviewIssues.slice(0, 3).join("; ")}`);
      if (!validation?.valid) throw new Error(`Revision is blocked by ${validation?.issues.length ?? 0} unresolved mapping or omission issue(s): ${validation?.issues.slice(0, 3).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
      // A newly reviewed revision may resolve mappings that were incomplete in
      // its immutable parent. Preserve the parent byte-for-byte and allow only
      // those historical mapping gaps through integrity verification; current
      // revision readiness below remains strict.
      const sourceIntegrity = await verifyPacketIntegrity(packet, { allowUnresolvedMappings: true });
      if (!sourceIntegrity.valid) throw new Error(`The retained packet has ${sourceIntegrity.issues.length} integrity issue(s): ${sourceIntegrity.issues.slice(0, 3).map((issue) => `${issue.path}: ${issue.message}`).join("; ")}`);
      const revisionId = `v${packet.revisions.length + 1}`;
      const source = await createSourceRecord({ name: `${editableOriginal.name.replace(/\.html?$/i, "")}-${revisionId}.html`, origin: editableOriginal.origin, version: `authored revision from ${current.id}`, bytes: editableSource, mediaType: "text/html", use: `Editable source for immutable packet revision ${revisionId}`, amendmentHistory: [`Created ${changeSummary}; parent source remains retained.`] });
      const nextPages = current.pages.map((page) => ({ ...page, sourceRefs: [...new Set([...page.sourceRefs, source.id])], editableSourceIds: [...new Set([...page.editableSourceIds, source.id])] }));
      const nextRequirements = requirementDrafts.map((item) => ({
        ...item,
        sourceRefs: [...new Set([...item.sourceRefs, ...(item.sourceSceneTrace ?? []).flatMap((trace) => trace.pageId ? [source.id] : [])])],
        sourceSceneTrace: (item.sourceSceneTrace ?? []).map((trace) => ({ ...trace, sourceIds: [...new Set([...trace.sourceIds, source.id])], editableSourceIds: [...new Set([...trace.editableSourceIds, source.id])] })),
      }));
      const nextFeatures = featureDrafts.map((item) => ({ ...item, relatedSourceIds: [...new Set([...(item.relatedSourceIds ?? []), source.id])] }));
      const nextPacket = { ...packet, sources: [...packet.sources, source] };
      const reviewSuffix = reviewDirty ? ` Mapping/omission review: ${reviewSummary.trim()}.` : "";
      const revised = await addPacketRevision(nextPacket, { revisionId, createdAt: new Date().toISOString(), changeSummary: `${changeSummary}; the exact amended editable source is linked to all ${nextPages.length} packet pages.${reviewSuffix}`, sourceIds: [...current.sourceIds, source.id], pages: nextPages, requirements: nextRequirements, features: nextFeatures });
      setRevisedSource(editableSource);
      const revision = revised.revisions.find((item) => item.id === revisionId)!;
      setPacket(revised); setRequirementDrafts(revision.requirements); setFeatureDrafts(revision.features); setRevisedSourceRevisionId(revisionId); setRevisedSourceId(source.id); setSourceEditsPending(false); setArtifactPaths(null);
      setStatus(`Created immutable ${revisionId} linked to ${current.id}. This is a design revision, not product acceptance.`);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not create the editable revision."); }
    finally { setBusy(false); }
  };

  const savePacket = async () => {
    if (!packet) return;
    setBusy(true); setError("");
    try {
      if (importedFiles.current !== files) throw new Error("The selected source inputs changed after import. Re-validate them before saving so source identities cannot become stale.");
      if (reviewDirty) throw new Error("Record the staged mapping and omission edits as an immutable revision before export.");
      if (sourceEditsPending) throw new Error("Create an immutable source revision for the current editor text before export.");
      const integrity = await verifyPacketIntegrity(packet);
      if (!integrity.valid) throw new Error(`Export is blocked by ${integrity.issues.length} packet integrity or mapping issue(s): ${integrity.issues.slice(0, 3).map((item) => `${item.path}: ${item.message}`).join("; ")}`);
      if (!editableOriginal) throw new Error("Import the editable prototype.html source before saving the versioned packet.");
      const revision = packet.revisions.find((item) => item.id === packet.currentRevisionId);
      const linkedCorrection = revision?.decisionRefs?.some((decision) => decision.id === CORRECTION.id) ?? false;
      if (!revisedSource || !revisedSourceRecord || (packet.scenario === "Notification Hub" && !linkedCorrection)) throw new Error(packet.scenario === "Notification Hub" ? "Apply the linked operator correction to create the required editable v2 before saving the Stage 1 packet." : "Create an immutable editable revision before saving this design packet.");
      const prefix = slugify(packet.scenario);
      const packetName = packetArchiveRecovery ? `${prefix}-design-packet-rebound-${packetArchiveRecovery.archiveSha256.slice(0, 8)}.json` : packet.scenario === "Notification Hub" ? "notification-hub-design-packet.json" : `${prefix}-design-packet.json`;
      if (!sessionId) {
        const sourceInputs = [...files].sort((a, b) => a.name.localeCompare(b.name));
        const ordered = [...assets].sort((a, b) => a.sceneId.localeCompare(b.sceneId, undefined, { numeric: true }));
        const sceneIds = [...new Set(ordered.map((asset) => asset.sceneId))];
        const atlasEntries = sceneIds.map((sceneId) => ({ name: `atlas/${prefix}-atlas-${sceneId}.html`, data: atlasPage(packet, sceneId, ordered, imageUrls) }));
        const atlasIndex = `${buildAtlasHtml(packet).replace("</body>", `<section><h2>Visual atlas pages</h2><ol>${sceneIds.map((id) => `<li><a href="${prefix}-atlas-${escapeHtml(id)}.html">${escapeHtml(id)}</a></li>`).join("")}</ol><p>Each linked page embeds its manifest-selected images and requirement joins; each image hash is printed beside its original.</p></section></body>`)}`;
        const csv = (rows: string[][]) => rows.map((row) => row.map((cell) => `"${cell.replace(/"/g, '""')}"`).join(",")).join("\r\n") + "\r\n";
        const requirementRows = [["requirement_id", "requirement_text", "requirement_scope", "source_requirement_ids", "source_map_refs", "source_state_scope_text", "coverage", "trace_page_ids", "trace_states", "trace_obligation_ids", "representative_page", "representative_state", "representative_frame", "representative_source_id", "trace_source_ids_by_scene", "editable_source_ids_by_scene", "selected_images_hash_and_source_ids", "disposition", "behavior_proof_status", "source_map_proof_basis", "recorded_evidence_references"], ...revision!.requirements.map((item) => [item.id, item.text, item.scope, (item.sourceRequirementIds ?? []).join(";"), (item.sourceMapRefs ?? []).join(";"), item.sourceStateText ?? "", item.coverage, (item.sourceSceneTrace ?? []).map((trace) => trace.pageId).join(";"), (item.sourceSceneTrace ?? []).map((trace) => `${trace.pageId}:${trace.stateIds.join("|")}`).join(";"), (item.sourceSceneTrace ?? []).map((trace) => trace.obligationId).join(";"), item.mapping.pageId ?? "", item.mapping.stateId ?? "", item.mapping.region ?? "", item.mapping.sourceId ?? "", JSON.stringify((item.sourceSceneTrace ?? []).map((trace) => ({ page: trace.pageId, sources: trace.sourceIds }))), JSON.stringify((item.sourceSceneTrace ?? []).map((trace) => ({ page: trace.pageId, editable: trace.editableSourceIds }))), JSON.stringify((item.sourceSceneTrace ?? []).map((trace) => ({ page: trace.pageId, images: trace.sourceImages }))), item.disposition, item.behaviorProofStatus ?? "", item.proofBasisText ?? "", item.evidenceRefs.join(";")])];
        const omissionRows = [["obligation_id", "source_obligation", "atlas_page", "state_ids", "source_requirement_ids", "linked_packet_requirement_ids", "source_ids", "selected_images_hash_source_id_state", "disposition", "basis"], ...revision!.features.map((item) => [item.id, item.name, item.pageId ?? "", (item.stateIds ?? []).join(";"), (item.sourceRequirementIds ?? []).join(";"), item.requirementIds.join(";"), (item.relatedSourceIds ?? [item.sourceRef]).join(";"), JSON.stringify(item.sourceImages ?? []), item.disposition, item.rationale])];
        const bundleEntries: Array<{ name: string; data: Blob | string }> = [
          { name: packetName, data: JSON.stringify(packet, null, 2) },
          { name: `editable/${prefix}-prototype-v1.html`, data: editableOriginal.content },
          { name: `editable/${prefix}-prototype-${revisedSourceRevisionId || revision!.id}.html`, data: revisedSource },
          { name: `atlas/${prefix}-design-atlas.html`, data: atlasIndex },
          { name: `handoff/${prefix}-fresh-implementer-handoff.md`, data: buildImplementerHandoff(packet) },
          { name: `trace/${prefix}-requirements.csv`, data: csv(requirementRows) },
          { name: `trace/${prefix}-omissions.csv`, data: csv(omissionRows) },
          ...atlasEntries,
          ...sourceInputs.map((file) => ({ name: `source-inputs/${file.name}`, data: file })),
        ];
        const archive = await makeLocalZip(bundleEntries);
        downloadLocalFile(archive, `${prefix}-authoring-export.zip`);
        setLocalExportReady(true); setArtifactPaths(null);
        setStatus(`Downloaded local authoring package (${bundleEntries.length} files): packet and source ledger, editable v1/v2, ${sceneIds.length}-page atlas, requirements and omission CSVs, handoff, and ${sourceInputs.length} retained input files. No session or upload API was used.`);
        return;
      }
      const packetPath = await uploadFile(sessionId, new Blob([JSON.stringify(packet, null, 2)], { type: "application/json" }), packetName);
      const input = await guessInput(files);
      const retainedPaths = [] as string[];
      for (const file of [...input.originals, ...input.bundles, input.sourceMap, input.dossier, input.manifest, ...(input.packetArchive ? [input.packetArchive.file] : [])]) {
        retainedPaths.push(await uploadFile(sessionId, file, `source-${file.name}`));
      }
      const originalPath = await uploadFile(sessionId, new Blob([editableOriginal.content], { type: "text/html" }), packet.scenario === "Notification Hub" ? "notification-hub-prototype-v1.html" : `${prefix}-prototype-v1.html`);
      const sourceRevisionName = revisedSourceRevisionId || revision?.id || "v2";
      const revisedPath = await uploadFile(sessionId, new Blob([revisedSource], { type: "text/html" }), packet.scenario === "Notification Hub" ? `notification-hub-prototype-${sourceRevisionName}.html` : `${prefix}-prototype-${sourceRevisionName}.html`);
      const scenePaths = new Map<string, string>();
      const ordered = [...assets].sort((a, b) => a.sceneId.localeCompare(b.sceneId, undefined, { numeric: true }));
      for (const sceneId of [...new Set(ordered.map((asset) => asset.sceneId))]) {
        const pageHtml = atlasPage(packet, sceneId, ordered, imageUrls);
        const pageName = packet.scenario === "Notification Hub" ? `hub-atlas-${sceneId}.html` : `${prefix}-atlas-${sceneId}.html`;
        const path = await uploadFile(sessionId, new Blob([pageHtml], { type: "text/html" }), pageName);
        scenePaths.set(sceneId, path);
      }
      const atlasIndex = `${buildAtlasHtml(packet).replace("</body>", `<section><h2>Visual atlas pages</h2><ol>${[...scenePaths].map(([id,path])=>`<li><a href="${escapeHtml(path)}">${escapeHtml(id)}</a></li>`).join("")}</ol><p>Each linked page embeds its manifest-selected images and requirement joins. Use Web Console preview links to move between pages; each image hash is printed beside its original.</p></section></body>`)}`;
      const atlasPath = await uploadFile(sessionId, new Blob([atlasIndex], { type: "text/html" }), packet.scenario === "Notification Hub" ? "notification-hub-design-atlas.html" : `${prefix}-design-atlas.html`);
      const handoffPath = await uploadFile(sessionId, new Blob([buildImplementerHandoff(packet)], { type: "text/markdown" }), packet.scenario === "Notification Hub" ? "notification-hub-fresh-implementer-handoff.md" : `${prefix}-fresh-implementer-handoff.md`);
      setArtifactPaths({ atlas: atlasPath, original: originalPath, revised: revisedPath, handoff: handoffPath }); setLocalExportReady(false);
      setStatus(`Saved immutable packet and ${scenePaths.size} self-contained atlas scene pages to this Web Console session. Packet: ${packetPath} · retained source inputs: ${retainedPaths.length} files · editable v1 source: ${originalPath} · editable v2 source: ${revisedPath} · Atlas index: ${atlasPath} · implementer handoff: ${handoffPath}.`);
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Could not save the packet to this Web Console session."); }
    finally { setBusy(false); }
  };

  const printAtlas = async () => {
    if (!packet) return;
    const revision = packet.revisions.find((item) => item.id === packet.currentRevisionId);
    if (!revision) { setError("The current packet revision is missing."); return; }
    if (!sourceInputsCurrent) { setError("The selected source inputs changed after import. Re-validate them before printing the atlas."); return; }
    if (!exportReady) { setError("Finish the packet mapping, linked revision and editable source before opening the atlas print view."); return; }
    const opened = window.open("about:blank", "_blank");
    if (!opened) { setError("The browser blocked the print tab. Allow this Web Console tab to open a local print preview, then retry."); return; }
    opened.opener = null;
    const integrity = await verifyPacketIntegrity(packet);
    if (!integrity.valid) { opened.close(); setError(`Atlas export is blocked by ${integrity.issues.length} packet integrity issue(s): ${integrity.issues.slice(0, 3).map((item) => `${item.path}: ${item.message}`).join("; ")}`); return; }
    const uniqueScenes = [...new Set(assets.map((asset) => asset.sceneId))];
    const visualPages = uniqueScenes.map((sceneId) => {
      const body = atlasPage(packet, sceneId, assets, imageUrls).match(/<body>([\s\S]*?)<\/body>/)?.[1] ?? "";
      return `<article class="scene" id="pdf-page-${encodeURIComponent(sceneId)}">${body}</article>`;
    }).join("");
    const requirementRows = revision.requirements.map((item) => `<tr><th>${escapeHtml(item.id)}</th><td>${escapeHtml(item.text)}</td><td>${escapeHtml(item.mapping.stateId ?? "—")}</td><td>${escapeHtml(item.mapping.pageId ?? "—")}</td><td>${escapeHtml(item.mapping.region ?? "—")}</td><td>${escapeHtml(item.disposition)}</td></tr>`).join("");
    const omissionRows = revision.features.map((item) => `<tr><th>${escapeHtml(item.id)}</th><td>${escapeHtml(item.name)}</td><td>${escapeHtml(item.disposition)}</td><td>${escapeHtml(item.rationale)}</td></tr>`).join("");
    const contents = uniqueScenes.map((id) => `<li><a href="#pdf-page-${encodeURIComponent(id)}">${escapeHtml(id)}</a></li>`).join("");
    const html = `<!doctype html><html><head><meta charset="utf-8"><title>${escapeHtml(packet.scenario)} design atlas</title><style>body{font:10pt/1.35 system-ui,sans-serif;color:#17242e;margin:18mm}h1{font-size:22pt}h2{font-size:16pt}h3{font-size:12pt}table{border-collapse:collapse;width:100%;margin:.5rem 0 1rem}th,td{text-align:left;vertical-align:top;border:1px solid #aebbc5;padding:.25rem;overflow-wrap:anywhere;font-size:8pt}figure{display:inline-block;vertical-align:top;width:45%;box-sizing:border-box;margin:1%;padding:5px;border:1px solid #bcc7ce;break-inside:avoid}img{display:block;width:100%;max-height:55vh;object-fit:contain}figcaption{font-size:7pt;overflow-wrap:anywhere}article.scene{break-before:page}article.scene>section{break-inside:auto}a{color:#075985}@media screen{body{max-width:1100px;margin:2rem auto}.scene{border-top:2px solid #64748b;padding-top:1rem;margin-top:3rem}}</style></head><body><header><h1>${escapeHtml(packet.scenario)} — design atlas</h1><p>Revision ${escapeHtml(revision.id)} · ${escapeHtml(revision.contentSha256)} · ${uniqueScenes.length} source scenes · ${assets.length} hash-verified selected image originals.</p><p>Source design evidence, not implementation or whole-product acceptance.</p></header><nav><h2>Scene index</h2><ol>${contents}</ol></nav><section><h2>Requirement/state map (${revision.requirements.length})</h2><table><thead><tr><th>Requirement</th><th>Source text</th><th>State</th><th>Scene</th><th>Region</th><th>Disposition</th></tr></thead><tbody>${requirementRows}</tbody></table></section><section><h2>Scene and omission register (${revision.features.length})</h2><table><thead><tr><th>Obligation</th><th>Source obligation</th><th>Disposition</th><th>Rationale</th></tr></thead><tbody>${omissionRows}</tbody></table></section>${visualPages}</body></html>`;
    opened.document.open(); opened.document.write(html); opened.document.close();
    setStatus(sessionId ? "Opened a local, self-contained print preview. Choose Print → Save as PDF in your browser; no file was sent outside this Web Console session." : "Opened a local, self-contained atlas preview. Choose Print → Save as PDF in your browser; no session API was used.");
  };

  return <div className="fixed inset-0 z-[100] bg-black/60 p-2 sm:p-6" role="presentation">
    <section role="dialog" aria-modal="true" aria-labelledby="design-authoring-title" className="mx-auto flex h-full max-w-6xl flex-col overflow-hidden rounded-xl border border-wc-default bg-wc-surface-raised shadow-2xl">
      <header className="flex items-start justify-between gap-4 border-b border-wc-default p-4 sm:p-6"><div><p className="text-xs font-semibold uppercase tracking-wider text-wc-text-muted">Web Console · local design packet</p><h2 id="design-authoring-title" className="mt-1 text-xl font-semibold text-wc-text-primary">Author from retained evidence</h2><p className="mt-1 max-w-3xl text-sm text-wc-text-secondary">Build and inspect a versioned design packet from retained source evidence. This does not edit or accept a product.</p></div><button type="button" onClick={onClose} aria-label="Close authoring workflow" className="min-h-11 shrink-0 rounded-md border border-wc-default px-3 text-sm">Close</button></header>
      <nav aria-label="Design packet steps" className="border-b border-wc-default px-3 py-3 sm:px-6"><ol className="grid grid-cols-2 gap-2 md:grid-cols-5">{stepNames.map((name, index) => <li key={name}><button type="button" disabled={!packet && index > 0} aria-current={activeStep === index ? "step" : undefined} onClick={() => setActiveStep(index)} className={`min-h-11 w-full rounded-md border px-3 py-2 text-left text-xs font-semibold sm:text-sm ${activeStep === index ? "border-wc-accent bg-wc-surface-input text-wc-text-primary" : "border-wc-default text-wc-text-secondary disabled:opacity-40"}`}><span className="mr-2 inline-flex h-6 w-6 items-center justify-center rounded-full bg-wc-surface-raised">{index + 1}</span>{name}</button></li>)}</ol></nav>
      <div className="min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
        {activeStep === 0 && <section className="mx-auto max-w-4xl rounded-lg border border-wc-default bg-wc-surface-input p-4 sm:p-6">
          <p className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">Step 1 · Inputs & lineage</p><h3 className="mt-1 text-lg font-semibold text-wc-text-primary">Select one retained design packet</h3>
          <p className="mt-2 text-sm text-wc-text-secondary">Choose a Markdown brief or dossier, a source map with requirement and scene tables, a JSON selected-image manifest, its referenced image files, and the editable HTML source. For Notification Hub, include its retained ZIP so the revised original keeps its bundled font. To recover source identity from an older packet, also include that prior packet JSON; the tool preserves it byte-for-byte, verifies its v1 rows against the re-import, and creates a new source-bound v1 before you apply the existing correction.</p>
          <label className="mt-5 block text-sm font-medium text-wc-text-primary" htmlFor="design-scenario-name">Scenario name</label><input id="design-scenario-name" value={scenarioName} onChange={(event) => setScenarioName(event.currentTarget.value)} placeholder="Use the brief filename if blank" className="mt-1 min-h-11 w-full rounded-md border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary" />
          <label className="mt-4 block text-sm font-medium text-wc-text-primary" htmlFor="design-input-files">Design brief, source map, supporting Markdown, manifest, selected images and editable source</label><input id="design-input-files" ref={fileInput} type="file" multiple onChange={(event) => setFiles(Array.from(event.currentTarget.files ?? []))} className="mt-2 block w-full text-sm text-wc-text-secondary file:mr-3 file:min-h-11 file:rounded-md file:border-0 file:bg-wc-accent file:px-4 file:font-semibold file:text-white" /><p className="mt-2 text-sm text-wc-text-secondary">Additional Markdown sources are retained with exact hashes. Accepted-surface records link only to the scene IDs they name; they never imply acceptance of the whole product.</p>
          <div className="mt-2 text-sm text-wc-text-secondary">{files.length} files selected</div>{packet && importedFiles.current !== files && <p role="alert" className="mt-2 rounded border border-amber-400 bg-amber-50 p-3 text-sm text-amber-900">Inputs changed after import. Re-validate to bind and compare the new source set before saving or exporting.</p>}<div className="mt-5 flex flex-wrap gap-3"><button type="button" disabled={busy || !files.length} onClick={() => void importSources()} className="min-h-11 rounded-md bg-wc-accent px-4 text-sm font-semibold text-white disabled:opacity-50">{busy ? "Validating source…" : "Validate and build packet"}</button></div>
          {packet && current && <div className="mt-4 rounded-md border border-wc-default bg-wc-surface-raised p-3"><p className="text-sm text-wc-text-secondary">Loaded <strong>{packet.scenario}</strong> · revision {current.id} · {current.sourceIds.length} linked source records · packet content {current.contentSha256}</p><details className="mt-3"><summary className="cursor-pointer text-sm font-medium text-wc-text-primary">Review source ledger ({packet.sources.length} exact source records)</summary><div className="mt-3 max-h-96 overflow-auto rounded border border-wc-default"><table className="w-full text-left text-xs"><thead className="sticky top-0 bg-wc-surface-raised"><tr><th className="p-2">Source</th><th className="p-2">Origin / version</th><th className="p-2">SHA-256 / bytes</th><th className="p-2">Use</th></tr></thead><tbody>{packet.sources.map((source) => <tr key={source.id} className="border-t border-wc-default"><td className="p-2">{source.name}</td><td className="p-2 break-all">{source.origin}<br />{source.version}</td><td className="p-2 break-all">{source.sha256}<br />{source.bytes} bytes</td><td className="p-2">{source.use}</td></tr>)}</tbody></table></div></details></div>}
        </section>}
        {activeStep === 1 && packet && current && <section>
          <p className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">Step 2 · Mappings & omissions</p>
          <h3 className="mt-1 text-lg font-semibold text-wc-text-primary">Review every requirement and source obligation</h3>
          <p className="mt-1 text-sm text-wc-text-secondary">Revision {current.id} · {current.contentSha256} · {current.pages.length} scenes · {requirementDrafts.length} requirements · {featureDrafts.length} obligations. Any unresolved or stale join blocks export.</p>
          <p className="mt-2 rounded-md bg-wc-surface-input p-3 text-xs text-wc-text-secondary">A disposition records this design packet’s scope. “Remove” does not authorize removing a product feature or accept a product change.</p>
          {reviewDirty && <p role="status" className="mt-3 rounded-md border border-wc-default p-3 text-sm text-wc-text-secondary">You have unversioned review edits. They will not be exported until recorded as an immutable revision.</p>}
          {validation && !validation.valid && <div role="alert" className="mt-3 rounded-md border border-amber-400 bg-amber-50 p-3 text-sm text-amber-900"><strong>{validation.issues.length} packet checks need attention</strong><ul className="mt-2 list-disc pl-5">{validation.issues.slice(0, 8).map((issue) => <li key={`${issue.path}:${issue.message}`}>{issue.path}: {issue.message}</li>)}</ul></div>}
          <div className="mt-4 grid gap-4 xl:grid-cols-2">
            <div className="rounded-lg border border-wc-default p-4">
              <h4 className="font-semibold text-wc-text-primary">Requirements → state → atlas region</h4>
              <ul className="mt-3 space-y-3">{requirementDrafts.map((item) => <li key={item.id} className="rounded-md border border-wc-default bg-wc-surface-input p-3">
                <strong className="text-sm text-wc-text-primary">{item.id}</strong><p className="mt-1 text-sm text-wc-text-secondary">{item.text}</p>
                <p className="mt-1 text-xs text-wc-text-muted">Current: {item.mapping.pageId ?? "No atlas page"} · {item.mapping.stateId ?? "No selected state"} · {item.mapping.region ?? "No mapped frame"}</p>
                <details className="mt-2"><summary className="cursor-pointer text-sm font-medium text-wc-text-secondary">Edit coverage, mapping and disposition</summary><div className="mt-3 grid gap-3 sm:grid-cols-2">
                  <label className="text-xs font-medium text-wc-text-primary">Coverage<select aria-label={`Coverage for ${item.id}`} value={item.coverage} onChange={(event) => updateRequirementCoverage(item.id, event.currentTarget.value as CoverageKind)} className="mt-1 min-h-11 w-full rounded-md border border-wc-default bg-wc-surface-raised px-2 text-sm"><option value="visual">Visual design surface</option><option value="nonvisual">Nonvisual proof surface</option><option value="unknown">Unknown — unresolved</option></select></label>
                  <label className="text-xs font-medium text-wc-text-primary">Packet disposition<select aria-label={`Requirement disposition for ${item.id}`} value={item.disposition} onChange={(event) => setRequirementDisposition(item.id, event.currentTarget.value as OmissionDisposition)} className="mt-1 min-h-11 w-full rounded-md border border-wc-default bg-wc-surface-raised px-2 text-sm"><option value="preserve">Preserve</option><option value="remove">Remove (requires justification)</option><option value="defer">Defer</option><option value="unknown">Unknown — unresolved</option></select></label>
                  {item.coverage === "visual" && <label className="text-xs font-medium text-wc-text-primary sm:col-span-2">Exact source frame<select aria-label={`Exact source frame for ${item.id}`} value={exactFrameKey(item, assets)} onChange={(event) => selectRequirementFrame(item.id, event.currentTarget.value)} className="mt-1 min-h-11 w-full rounded-md border border-wc-default bg-wc-surface-raised px-2 text-sm"><option value="">Choose one scene, state and hash-verified image</option>{frameOptions.filter(({ asset, stateId }) => item.sourceSceneTrace?.some((trace) => trace.pageId === asset.sceneId && trace.stateIds.includes(stateId))).map(({ key, asset, stateId }) => <option key={key} value={key}>{asset.sceneId} · {stateId} · {asset.filename}</option>)}</select></label>}
                  <label className="text-xs font-medium text-wc-text-primary sm:col-span-2">Mapping / disposition basis<textarea aria-label={`Mapping basis for ${item.id}`} value={item.mapping.rationale} onChange={(event) => setRequirementRationale(item.id, event.currentTarget.value)} className="mt-1 min-h-20 w-full rounded-md border border-wc-default bg-wc-surface-raised p-2 text-sm" /></label>
                </div>
                {item.coverage === "visual" && (!item.mapping.pageId || !item.mapping.stateId || !item.mapping.region) && <p className="mt-2 text-xs font-medium text-amber-800">Choose one exact frame. Missing or ambiguous candidates remain unresolved.</p>}
                {item.coverage === "nonvisual" && <p className="mt-2 text-xs text-wc-text-muted">Keep its named nonvisual proof source linked; do not invent a screen.</p>}
                <p className="mt-2 break-all text-xs text-wc-text-muted">Packet source IDs: {item.sourceRefs.join(" · ") || "none"}</p>
                <p className="mt-2 text-xs text-wc-text-muted">Source-map references: {item.sourceMapRefs?.join(" · ") || "none recorded"}</p>
                <p className="mt-2 text-xs text-wc-text-muted">Source state/scope: {item.sourceStateText || "missing"}</p>
                <p className="mt-2 text-xs text-wc-text-muted">Source-map proof basis (reference only; behavior not verified): {item.proofBasisText || "missing"}</p>
                <details className="mt-2"><summary className="cursor-pointer text-xs font-medium text-wc-text-secondary">All source-scene traces ({item.sourceSceneTrace?.length ?? 0})</summary><ul className="mt-2 space-y-2">{(item.sourceSceneTrace ?? []).map((trace) => { const obligation = featureDrafts.find((feature) => feature.id === trace.obligationId); return <li key={trace.obligationId} className="break-all rounded border border-wc-default p-2 text-xs"><strong>{trace.pageId}</strong> · states {trace.stateIds.join(", ")} · obligation {trace.obligationId} · disposition {obligation?.disposition ?? "missing"}<br />Editable source IDs: {trace.editableSourceIds.join(" · ")}<br />Exact source IDs: {trace.sourceIds.join(" · ")}<br />Selected images: {trace.sourceImages.map((image) => `${image.filename} · ${image.sha256} · ${image.sourceId} · ${image.stateIds.join(", ")}`).join("; ") || "none"}</li>; })}</ul></details></details>
              </li>)}</ul>
            </div>
            <div className="rounded-lg border border-wc-default p-4">
              <h4 className="font-semibold text-wc-text-primary">Source feature / omission register</h4>
              <p className="mt-1 text-xs text-wc-text-muted">Every indexed obligation keeps its stable ID and source join through revisions.</p>
              <ul className="mt-3 space-y-2">{featureDrafts.map((item) => <li key={item.id} className="rounded-md border border-wc-default bg-wc-surface-input p-3">
                <strong className="text-sm text-wc-text-primary">{item.name}</strong>
                <p className="mt-1 text-xs text-wc-text-muted">{item.id} · linked: {item.requirementIds.join(", ") || "none named"}</p>
                <p className="mt-1 break-all text-xs text-wc-text-muted">Page {item.pageId ?? "missing"} · states {item.stateIds?.join(", ") ?? "missing"} · exact source IDs {item.relatedSourceIds?.join(" · ") ?? item.sourceRef}</p>
                <p className="mt-1 text-xs text-wc-text-muted">Source requirement IDs: {item.sourceRequirementIds?.join(", ") || "none named"}{item.unresolvedSourceRequirementIds?.length ? ` · unresolved: ${item.unresolvedSourceRequirementIds.join(", ")}` : ""}</p>
                <label className="mt-2 block text-xs font-medium text-wc-text-primary">Disposition<select aria-label={`Disposition for ${item.id}`} value={item.disposition} onChange={(event) => setFeatureDisposition(item.id, event.currentTarget.value as OmissionDisposition)} className="mt-1 min-h-11 w-full rounded-md border border-wc-default bg-wc-surface-raised px-2 text-sm"><option value="preserve">Preserve</option><option value="remove">Remove (requires justification)</option><option value="defer">Defer</option><option value="unknown">Unknown — unresolved</option></select></label>
                <label className="mt-2 block text-xs font-medium text-wc-text-primary">Disposition basis<textarea aria-label={`Disposition basis for ${item.id}`} value={item.rationale} onChange={(event) => setFeatureRationale(item.id, event.currentTarget.value)} className="mt-1 min-h-20 w-full rounded-md border border-wc-default bg-wc-surface-raised p-2 text-sm" /></label>
                <p className="mt-2 text-xs text-wc-text-muted">Source join: {item.sourceRef}</p>
              </li>)}</ul>
            </div>
          </div>
          {reviewDirty && <div className="mt-4 rounded-lg border border-wc-default p-4"><label className="block text-sm font-medium text-wc-text-primary" htmlFor="design-review-summary">Revision summary</label><input id="design-review-summary" value={reviewSummary} onChange={(event) => setReviewSummary(event.currentTarget.value)} className="mt-1 min-h-11 w-full max-w-2xl rounded-md border border-wc-default bg-wc-surface-input px-3 text-sm" /><p className="mt-1 text-xs text-wc-text-muted">This explanation is stored in packet history with the new version hash.</p>{draftReviewIssues.length > 0 && <ul className="mt-2 list-disc pl-5 text-sm text-amber-800">{draftReviewIssues.slice(0, 6).map((issue) => <li key={issue}>{issue}</li>)}</ul>}{validation && !validation.valid && <ul className="mt-2 list-disc pl-5 text-xs text-amber-800">{validation.issues.slice(0, 6).map((issue) => <li key={`${issue.path}:${issue.message}`}>{issue.message}</li>)}</ul>}{isHubPacket && !linkedCorrection ? <p className="mt-2 text-sm text-wc-text-secondary">Hub review edits will be recorded with the existing bottom-navigation correction so its decision remains bound to the exact parent version.</p> : <button type="button" disabled={busy || !reviewDirty || draftReviewIssues.length > 0 || !validation?.valid} onClick={() => void recordReviewRevision()} className="mt-3 min-h-11 rounded-md bg-wc-accent px-4 text-sm font-semibold text-white disabled:opacity-50">Record mapping and omission revision</button>}</div>}
          <p className="mt-4 text-xs text-wc-text-muted">Images and source-index rows are design evidence only. Backend, privacy, delivery, and platform obligations require their named independent proof surfaces.</p>
        </section>}
        {activeStep === 2 && packet && current && <section><p className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">Step 3 · Inspect atlas</p><h3 className="mt-1 text-lg font-semibold text-wc-text-primary">Inspect the selected source frames</h3><p className="mt-1 text-sm text-wc-text-secondary">Each image is bound to the selected-image manifest and source hash. Use the scene selector to review every atlas entry.</p>{reviewDirty && <p className="mt-2 rounded-md border border-wc-default p-3 text-sm text-amber-800">This atlas pane includes unversioned review edits. Record them as a revision before export.</p>}<label className="mt-4 block text-sm font-medium text-wc-text-primary" htmlFor="design-atlas-scene">Atlas scene</label><select id="design-atlas-scene" value={selectedScene?.id ?? ""} onChange={(event) => setSelectedSceneId(event.currentTarget.value)} className="mt-1 min-h-11 w-full max-w-xl rounded-md border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary">{scenes.map((scene) => <option key={scene.id} value={scene.id}>{scene.id} · {scene.stateIds.join(", ")}</option>)}</select><div className="mt-4 grid gap-4 lg:grid-cols-[minmax(0,2fr)_minmax(16rem,1fr)]">{selectedSceneAssets.length ? <div className="grid gap-4 sm:grid-cols-2">{selectedSceneAssets.map((asset) => <figure key={asset.filename} className="overflow-hidden rounded-lg border border-wc-default bg-wc-surface-input"><img src={imageUrls.get(asset.filename) ?? ""} alt={`${asset.sceneId} · ${asset.stateIds.join(", ")}`} className="max-h-[70vh] w-full object-contain" /><figcaption className="break-all border-t border-wc-default p-3 text-xs text-wc-text-secondary"><strong>{asset.filename}</strong><br />{asset.sceneId} · {asset.stateIds.join(", ")}<br />SHA-256 {asset.sha256}</figcaption></figure>)}</div> : <p className="rounded-lg border border-wc-default p-4 text-sm text-wc-text-secondary">No selected frame is mapped to this scene. This gap is retained in the packet.</p>}<aside className="rounded-lg border border-wc-default p-4"><h4 className="font-semibold text-wc-text-primary">Linked requirements</h4><ul className="mt-2 space-y-2">{requirementDrafts.filter((item) => item.mapping.pageId === selectedScene?.id || item.sourceSceneTrace?.some((trace) => trace.pageId === selectedScene?.id)).map((item) => <li key={item.id} className="border-b border-wc-default pb-2 text-sm"><strong>{item.id}</strong><p className="text-xs text-wc-text-secondary">{item.text}</p><span className="text-xs text-wc-text-muted">{item.sourceSceneTrace?.find((trace) => trace.pageId === selectedScene?.id)?.stateIds.join(", ") ?? item.mapping.stateId ?? "No state"} · {item.mapping.region ?? "No single representative frame"} · behavior not verified</span></li>)}</ul><h4 className="mt-4 font-semibold text-wc-text-primary">Scene obligations</h4><ul className="mt-2 space-y-1">{featureDrafts.filter((item) => sourceSceneIdFromObligationId(item.id) === selectedScene?.id).map((item) => <li key={item.id} className="text-sm text-wc-text-secondary"><strong>{item.name}</strong><span className="block text-xs text-wc-text-muted">{item.id} · {item.disposition}</span></li>)}</ul></aside></div></section>}
        {activeStep === 3 && packet && current && <section className="mx-auto max-w-5xl"><p className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">Step 4 · Edit & revise</p><h3 className="mt-1 text-lg font-semibold text-wc-text-primary">Create an immutable source revision</h3><p className="mt-1 text-sm text-wc-text-secondary">V1 remains unchanged. A revision records its parent version, source hash, change reason, and—when using the retained Hub correction—the exact existing decision.</p><div className="mt-4 grid gap-4 lg:grid-cols-2"><div className="rounded-lg border border-wc-default p-4"><h4 className="font-semibold text-wc-text-primary">Original source · {editableOriginal?.name ?? "missing"}</h4><p className="mt-1 break-all text-xs text-wc-text-muted">Origin: {editableOriginal?.origin ?? "Import an editable HTML file"} · V1 source record: {editableOriginal?.sourceId ?? "not bound"}</p><label className="mt-4 block text-sm font-medium text-wc-text-primary" htmlFor="design-source-editor">Editable HTML source</label><textarea id="design-source-editor" value={editableSource} readOnly={Boolean(isHubPacket && linkedCorrection)} onChange={(event) => { setEditableSource(event.currentTarget.value); setSourceEditsPending(true); }} className="mt-2 min-h-72 w-full rounded-md border border-wc-default bg-wc-surface-input p-3 font-mono text-xs text-wc-text-primary read-only:opacity-75" spellCheck={false} /></div><div className="space-y-4"><div className="rounded-lg border border-wc-default p-4"><h4 className="font-semibold text-wc-text-primary">Existing correction</h4>{isHubPacket ? <><blockquote className="mt-2 border-l-2 border-wc-accent pl-3 text-sm text-wc-text-secondary">“I do in fact want the bottom nav.”</blockquote><p className="mt-2 text-xs text-wc-text-muted">Decision {CORRECTION.id} · {CORRECTION.source} · {CORRECTION.sourceTimestamp}</p><button type="button" disabled={busy || linkedCorrection || !editableOriginal} onClick={() => void applyCorrection()} className="mt-4 min-h-11 rounded-md bg-wc-accent px-4 text-sm font-semibold text-white disabled:opacity-50">{linkedCorrection ? "Correction linked to v1" : "Apply correction to this source"}</button>{linkedCorrection && <p className="mt-2 text-sm text-emerald-800">Revision {current.id} records this decision's exact parent version: {current.decisionRefs.find((item) => item.id === CORRECTION.id)?.appliesToVersion}.</p>}</> : <><label className="mt-2 block text-sm font-medium text-wc-text-primary" htmlFor="design-revision-summary">Change reason</label><input id="design-revision-summary" value={revisionSummary} onChange={(event) => setRevisionSummary(event.currentTarget.value)} className="mt-1 min-h-11 w-full rounded-md border border-wc-default bg-wc-surface-input px-3 text-sm text-wc-text-primary" /><button type="button" disabled={busy || !editableOriginal || !editableSource.trim()} onClick={() => void createEditableRevision()} className="mt-3 min-h-11 rounded-md border border-wc-default px-4 text-sm font-semibold text-wc-text-primary disabled:opacity-50">Create immutable revision</button></>}</div><div className="rounded-lg border border-wc-default p-4"><h4 className="font-semibold text-wc-text-primary">Revision review</h4>{revisedSource ? <><p className="mt-1 break-all text-sm text-wc-text-secondary">Packet revision {current.id} · {current.contentSha256}<br />Editable source {revisedSourceRevisionId || "unbound"} · {revisedSourceRecord ? `${revisedSourceRecord.name} · SHA-256 ${revisedSourceRecord.sha256}` : "source record missing"}.</p>{sourceEditsPending && <p className="mt-2 text-sm text-amber-800">The editor has unversioned changes. Create a new immutable revision before export.</p>}<details className="mt-3"><summary className="cursor-pointer text-sm font-medium">Inspect revised source text</summary><pre className="mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-all rounded bg-wc-surface-input p-3 text-xs">{revisedSource}</pre></details></> : <p className="mt-1 text-sm text-wc-text-secondary">No revised source yet. Finish the recorded correction or create a revision before export.</p>}</div></div></div></section>}
        {activeStep === 4 && packet && current && <section className="mx-auto max-w-4xl"><p className="text-xs font-semibold uppercase tracking-wide text-wc-text-muted">Step 5 · Export & hand off</p><h3 className="mt-1 text-lg font-semibold text-wc-text-primary">Save the packet and prepare the next implementer</h3><p className="mt-1 text-sm text-wc-text-secondary">{sessionId ? "Exports use this Web Console session's existing upload path." : "Exports are assembled locally in one ZIP; nothing is uploaded."} The atlas print view is local; choose Print → Save as PDF in the browser.</p><div className="mt-4 rounded-lg border border-wc-default p-4"><h4 className="font-semibold text-wc-text-primary">Readiness</h4><p className={`mt-2 text-sm ${exportReady ? "text-emerald-800" : "text-amber-800"}`}>{exportReady ? `Trace complete: ${current.requirements.length} requirements and ${current.features.length} source-scene obligations; behavior proof is unverified for all ${current.requirements.length}; editable ${current.id} is linked.` : `Not ready: ${validation?.issues.length ?? 0} packet issues · ${editableOriginal ? "editable source present" : "editable source missing"} · ${revisedSource ? "revision present" : "revision missing"}${sourceEditsPending ? " · editor changes need a revision" : ""}${!revisedSourceRecord && revisedSource ? " · source record missing" : ""}${!sourceInputsCurrent ? " · selected inputs changed; re-import" : ""}${isHubPacket && !linkedCorrection ? " · existing Hub correction not linked" : ""}.`}</p>{validation && !validation.valid && <ul className="mt-2 list-disc pl-5 text-xs text-wc-text-secondary">{validation.issues.slice(0, 8).map((issue) => <li key={`${issue.path}:${issue.message}`}>{issue.path}: {issue.message}</li>)}</ul>}<p className="mt-2 break-all text-xs text-wc-text-muted">Packet {packet.id} · revision {current.id} · SHA-256 {current.contentSha256}</p></div><div className="mt-4 flex flex-wrap gap-3"><button type="button" disabled={busy || !exportReady} onClick={() => void savePacket()} className="min-h-11 rounded-md bg-wc-accent px-4 text-sm font-semibold text-white disabled:opacity-50">{busy ? "Saving…" : sessionId ? "Save packet, atlas and handoff" : "Download local packet and atlas package"}</button><button type="button" disabled={!exportReady} onClick={printAtlas} className="min-h-11 rounded-md border border-wc-default px-4 text-sm font-semibold text-wc-text-primary disabled:opacity-50">Open atlas print view / PDF</button></div>{artifactPaths && onOpenArtifact && <div className="mt-4 flex flex-wrap gap-2" aria-label="Saved design artifacts"><button type="button" onClick={() => onOpenArtifact(artifactPaths.original)} className="min-h-11 rounded-md border border-wc-default px-3 text-sm">Open v1 source</button><button type="button" onClick={() => onOpenArtifact(artifactPaths.revised)} className="min-h-11 rounded-md border border-wc-default px-3 text-sm">Open {current.id} source</button><button type="button" onClick={() => onOpenArtifact(artifactPaths.atlas)} className="min-h-11 rounded-md border border-wc-default px-3 text-sm">Open atlas index</button><button type="button" onClick={() => onOpenArtifact(artifactPaths.handoff)} className="min-h-11 rounded-md border border-wc-default px-3 text-sm">Open implementer handoff</button></div>}<div className="mt-4 rounded-lg bg-wc-surface-input p-4"><h4 className="font-semibold text-wc-text-primary">Reuse this workflow</h4><p className="mt-1 text-sm text-wc-text-secondary">Open another scenario's retained brief, source map, selected-image manifest, referenced images and editable HTML in the same Web Console File Preview action. Resolve every unknown row, preserve the source/revision history, then generate that packet's atlas and handoff. This Stage 1 evidence is only for Notification Hub.</p></div></section>}
        {(!packet || !current) && activeStep > 0 && <p className="rounded-md border border-wc-default p-4 text-sm text-wc-text-secondary">Import and validate source evidence before continuing.</p>}
        {status && <p role="status" className="mt-4 rounded-md bg-emerald-50 p-3 text-sm text-emerald-900">{status}</p>}{error && <p role="alert" className="mt-4 rounded-md bg-red-50 p-3 text-sm text-red-900">{error}</p>}
      </div>
      <footer className="border-t border-wc-default px-4 py-3">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <span className="text-xs text-wc-text-muted">
            {sessionId ? "Private session storage" : "Local files only; no session API"} · no product deployment · no whole-product acceptance implied
          </span>
          <OverlayActionBar
            label="Design authoring steps"
            maxVisible={2}
            className="flex-row-reverse sm:w-auto"
            actions={
              activeStep < stepNames.length - 1
                ? [
                    {
                      id: "next",
                      label: "Next",
                      onSelect: () => setActiveStep((step) => Math.min(stepNames.length - 1, step + 1)),
                      disabled: !packet,
                      primary: true,
                    },
                    {
                      id: "back",
                      label: "Back",
                      onSelect: () => setActiveStep((step) => Math.max(0, step - 1)),
                      disabled: activeStep === 0,
                    },
                  ]
                : [
                    {
                      id: "back",
                      label: "Back",
                      onSelect: () => setActiveStep((step) => Math.max(0, step - 1)),
                      disabled: activeStep === 0,
                      primary: true,
                    },
                  ]
            }
          />
        </div>
      </footer>
    </section>
  </div>;
}
