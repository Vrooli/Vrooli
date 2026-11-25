import { readFileSync } from "node:fs";
import { inflateRawSync } from "node:zlib";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

import { webcrypto } from "node:crypto";
import { DesignAuthoringTool } from "./DesignAuthoringTool";

const PACKET_ZIP = process.env.DTI2_PACKET_ZIP;

interface ZipEntry {
  name: string;
  bytes: Buffer;
}

function zipEntries(archive: Buffer): ZipEntry[] {
  let endRecord = -1;
  for (let offset = archive.length - 22; offset >= Math.max(0, archive.length - 65_557); offset -= 1) {
    if (archive.readUInt32LE(offset) === 0x06054b50) {
      endRecord = offset;
      break;
    }
  }
  if (endRecord < 0) throw new Error("Packet ZIP has no central directory end record.");
  const entryCount = archive.readUInt16LE(endRecord + 10);
  const directoryOffset = archive.readUInt32LE(endRecord + 16);
  const entries: ZipEntry[] = [];
  let cursor = directoryOffset;
  for (let index = 0; index < entryCount; index += 1) {
    if (archive.readUInt32LE(cursor) !== 0x02014b50) throw new Error("Packet ZIP central directory entry is malformed.");
    const method = archive.readUInt16LE(cursor + 10);
    const compressedSize = archive.readUInt32LE(cursor + 20);
    const nameLength = archive.readUInt16LE(cursor + 28);
    const extraLength = archive.readUInt16LE(cursor + 30);
    const commentLength = archive.readUInt16LE(cursor + 32);
    const localOffset = archive.readUInt32LE(cursor + 42);
    const name = archive.subarray(cursor + 46, cursor + 46 + nameLength).toString("utf8");
    cursor += 46 + nameLength + extraLength + commentLength;
    if (!name.startsWith("source-inputs/")) continue;
    if (archive.readUInt32LE(localOffset) !== 0x04034b50) throw new Error(`Missing local header for ${name}.`);
    const localNameLength = archive.readUInt16LE(localOffset + 26);
    const localExtraLength = archive.readUInt16LE(localOffset + 28);
    const dataStart = localOffset + 30 + localNameLength + localExtraLength;
    const compressed = archive.subarray(dataStart, dataStart + compressedSize);
    const bytes = method === 0 ? Buffer.from(compressed) : method === 8 ? inflateRawSync(compressed) : null;
    if (!bytes) throw new Error(`Unsupported ZIP compression method ${method} for ${name}.`);
    entries.push({ name: name.slice("source-inputs/".length), bytes });
  }
  return entries;
}

function fileType(name: string): string {
  if (/\.png$/i.test(name)) return "image/png";
  if (/\.json$/i.test(name)) return "application/json";
  if (/\.zip$/i.test(name)) return "application/zip";
  if (/\.html?$/i.test(name)) return "text/html";
  if (/\.md$/i.test(name)) return "text/markdown";
  return "application/octet-stream";
}

async function blobBytes(blob: Blob): Promise<ArrayBuffer> {
  if (typeof blob.arrayBuffer === "function") return blob.arrayBuffer();
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as ArrayBuffer);
    reader.onerror = () => reject(new Error("Could not read exported local packet bytes."));
    reader.readAsArrayBuffer(blob);
  });
}

async function readStoredZip(blob: Blob): Promise<Map<string, Uint8Array>> {
  const bytes = new Uint8Array(await blobBytes(blob));
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let endRecord = -1;
  for (let offset = bytes.length - 22; offset >= Math.max(0, bytes.length - 65_557); offset -= 1) {
    if (view.getUint32(offset, true) === 0x06054b50) { endRecord = offset; break; }
  }
  if (endRecord < 0) throw new Error("Exported packet has no ZIP directory.");
  const fileCount = view.getUint16(endRecord + 10, true);
  const directoryOffset = view.getUint32(endRecord + 16, true);
  const decoder = new TextDecoder();
  const files = new Map<string, Uint8Array>();
  let cursor = directoryOffset;
  for (let index = 0; index < fileCount; index += 1) {
    if (view.getUint32(cursor, true) !== 0x02014b50) throw new Error("Exported packet ZIP directory is malformed.");
    const method = view.getUint16(cursor + 10, true);
    if (method !== 0) throw new Error(`Unexpected ZIP compression method ${method}.`);
    const size = view.getUint32(cursor + 24, true);
    const nameLength = view.getUint16(cursor + 28, true);
    const extraLength = view.getUint16(cursor + 30, true);
    const commentLength = view.getUint16(cursor + 32, true);
    const localOffset = view.getUint32(cursor + 42, true);
    const name = decoder.decode(bytes.subarray(cursor + 46, cursor + 46 + nameLength));
    const localNameLength = view.getUint16(localOffset + 26, true);
    const localExtraLength = view.getUint16(localOffset + 28, true);
    const start = localOffset + 30 + localNameLength + localExtraLength;
    files.set(name, bytes.slice(start, start + size));
    cursor += 46 + nameLength + extraLength + commentLength;
  }
  return files;
}

function exportedText(files: Map<string, Uint8Array>, name: string): string {
  const bytes = files.get(name);
  if (!bytes) throw new Error(`Export is missing ${name}.`);
  return new TextDecoder().decode(bytes);
}

function parseCsv(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let quoted = false;
  for (let index = 0; index < text.length; index += 1) {
    const character = text[index]!;
    if (quoted && character === '"' && text[index + 1] === '"') { cell += '"'; index += 1; }
    else if (character === '"') quoted = !quoted;
    else if (!quoted && character === ",") { row.push(cell); cell = ""; }
    else if (!quoted && (character === "\n" || character === "\r")) {
      if (character === "\r" && text[index + 1] === "\n") index += 1;
      row.push(cell); rows.push(row); row = []; cell = "";
    } else cell += character;
  }
  if (cell || row.length) { row.push(cell); rows.push(row); }
  return rows;
}

beforeAll(() => {
  if (!globalThis.crypto?.subtle) Object.defineProperty(globalThis, "crypto", { configurable: true, value: webcrypto });
});

describe("exact retained Notification Hub packet through the existing authoring component", () => {
  it.skipIf(!PACKET_ZIP)("imports all selected real sources, verifies the row census, and authors the version-bound navigation correction locally", async () => {
    const entries = zipEntries(readFileSync(PACKET_ZIP!));
    const basenames = entries.map(({ name }) => name.split("/").at(-1)!);
    expect(entries).toHaveLength(249);
    expect(new Set(basenames).size).toBe(entries.length);
    expect(entries.filter(({ name }) => /\.png$/i.test(name))).toHaveLength(240);
    const files = entries.map(({ name, bytes }) => {
      const browserBytes = new Uint8Array(bytes.byteLength);
      browserBytes.set(bytes);
      return new File([browserBytes], name.split("/").at(-1)!, { type: fileType(name) });
    });
    const priorPacketFile = entries.find(({ name }) => name === "notification-hub-design-packet.json");
    expect(priorPacketFile).toBeTruthy();
    const priorPacket = JSON.parse(priorPacketFile!.bytes.toString("utf8")) as { currentRevisionId: string; revisions: Array<{ id: string; decisionRefs: Array<{ id: string; appliesToVersion: string }> }> };
    expect(priorPacket.currentRevisionId).toBe("v2-bottom-nav");

    render(<DesignAuthoringTool onClose={() => {}} />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "Notification Hub" } });
    fireEvent.change(screen.getByLabelText("Design brief, source map, supporting Markdown, manifest, selected images and editable source"), { target: { files } });
    fireEvent.click(screen.getByRole("button", { name: "Validate and build packet" }));

    await screen.findByText(/Imported 240 hash-checked visual sources/);
    expect(screen.getByText(/Archived prior packet notification-hub-ebce0298/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Mappings & omissions/ })).toBeEnabled();

    fireEvent.click(screen.getByRole("button", { name: /Edit & revise/ }));
    fireEvent.click(screen.getByRole("button", { name: "Apply correction to this source" }));
    await waitFor(() => expect(screen.queryByRole("alert") || screen.queryByText(/Created immutable v2 linked to operator decision NH-DESIGN-20261025-01 and its exact v1 parent/)).toBeTruthy());
    const correctionError = screen.queryByRole("alert")?.textContent;
    if (correctionError) throw new Error(`Exact packet correction failed: ${correctionError}`);
    await screen.findByText(/Created immutable v2 linked to operator decision NH-DESIGN-20261025-01 and its exact v1 parent/);
    expect(screen.getByText(/Revision v2-bottom-nav records this decision's exact parent version: v1/)).toBeInTheDocument();

    const revisedSourceDetails = screen.getByText("Inspect revised source text").closest("details");
    expect(revisedSourceDetails).toBeTruthy();
    fireEvent.click(revisedSourceDetails!.querySelector("summary")!);
    const revisedSource = screen.getByText((_, element) => element?.tagName === "PRE" && element.textContent?.includes('<nav class="bottom-nav"') === true);
    for (const destination of ["Needs you", "Reach", "Quiet hours", "Escalation", "Integrations"]) {
      expect(revisedSource).toHaveTextContent(destination);
    }
    expect(revisedSource).toHaveTextContent(/position:fixed[^}]*bottom:0/);

    fireEvent.click(screen.getByRole("button", { name: /Export & hand off/ }));
    expect(screen.getByText(/Trace complete: 28 requirements and 102 source-scene obligations; behavior proof is unverified for all 28; editable v2-bottom-nav is linked\./)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Download local packet and atlas package" })).toBeEnabled();
    expect(screen.getByText(/Local files only; no session API/)).toBeInTheDocument();

    let downloaded: Blob | undefined;
    let downloadedName = "";
    const createUrlDescriptor = Object.getOwnPropertyDescriptor(URL, "createObjectURL");
    const revokeUrlDescriptor = Object.getOwnPropertyDescriptor(URL, "revokeObjectURL");
    const originalSetTimeout = window.setTimeout;
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (this: HTMLAnchorElement) { downloadedName = this.download; });
    Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn((blob: Blob) => { downloaded = blob; return "blob:exact-packet-authoring"; }) });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() });
    const timeout = vi.spyOn(window, "setTimeout").mockImplementation((handler: TimerHandler, delay = 0, ...args: unknown[]) => {
      if (delay === 30_000 && typeof handler === "function") { (handler as (...values: unknown[]) => unknown)(...args); return 0 as unknown as ReturnType<typeof window.setTimeout>; }
      return Reflect.apply(originalSetTimeout, window, [handler, delay, ...args]) as ReturnType<typeof window.setTimeout>;
    });
    try {
      fireEvent.click(screen.getByRole("button", { name: "Download local packet and atlas package" }));
      await waitFor(() => expect(downloaded).toBeInstanceOf(Blob));
      expect(downloadedName).toBe("notification-hub-authoring-export.zip");
      expect(screen.getByText(/Downloaded local authoring package \(358 files\).*102-page atlas.*249 retained input files\. No session or upload API was used\./)).toBeInTheDocument();
      const exported = await readStoredZip(downloaded!);
      expect(exported.size).toBe(358);
      expect(exported.has("notification-hub-design-packet-rebound-9282eb88.json")).toBe(true);
      const retainedInputs = [...exported.keys()].filter((name) => name.startsWith("source-inputs/"));
      expect(retainedInputs).toHaveLength(entries.length);
      for (const entry of entries) {
        const retained = exported.get(`source-inputs/${entry.name}`);
        expect(retained, `export must preserve ${entry.name}`).toBeDefined();
        expect(Buffer.from(retained!).equals(entry.bytes), `export bytes must match ${entry.name}`).toBe(true);
      }

      const packet = JSON.parse(exportedText(exported, "notification-hub-design-packet-rebound-9282eb88.json"));
      expect(packet.currentRevisionId).toBe("v2-bottom-nav");
      const originalRevision = packet.revisions.find((revision: { id: string }) => revision.id === "v1");
      const revised = packet.revisions.find((revision: { id: string }) => revision.id === "v2-bottom-nav");
      expect(revised.parentId).toBe("v1");
      expect(revised.decisionRefs[0]).toMatchObject({ id: "NH-DESIGN-20261025-01", appliesToVersion: "v1" });
      expect(originalRevision.requirements).toHaveLength(28);
      expect(revised.requirements).toHaveLength(28);
      expect(revised.features).toHaveLength(102);
      expect(revised.requirements.every((row: { behaviorProofStatus: string }) => row.behaviorProofStatus === "not-verified")).toBe(true);
      expect(exportedText(exported, "editable/notification-hub-prototype-v1.html")).toBe(entries.find(({ name }) => name === "prototype.html")!.bytes.toString("utf8"));
      expect(exportedText(exported, "editable/notification-hub-prototype-v2.html")).toContain('<nav class="bottom-nav"');
      for (const destination of ["Needs you", "Reach", "Quiet hours", "Escalation", "Integrations"]) {
        expect(exportedText(exported, "editable/notification-hub-prototype-v2.html")).toContain(destination);
      }
      expect(exportedText(exported, "editable/notification-hub-prototype-v2.html")).toMatch(/position:fixed[^}]*bottom:0/);

      const requirementRows = parseCsv(exportedText(exported, "trace/notification-hub-requirements.csv"));
      const obligationRows = parseCsv(exportedText(exported, "trace/notification-hub-omissions.csv"));
      expect(requirementRows).toHaveLength(29);
      expect(requirementRows[0]).toHaveLength(21);
      const expectedRequirementRows = revised.requirements.map((item: any) => [
        item.id, item.text, item.scope, (item.sourceRequirementIds ?? []).join(";"), (item.sourceMapRefs ?? []).join(";"),
        item.sourceStateText ?? "", item.coverage, (item.sourceSceneTrace ?? []).map((trace: any) => trace.pageId).join(";"),
        (item.sourceSceneTrace ?? []).map((trace: any) => `${trace.pageId}:${trace.stateIds.join("|")}`).join(";"),
        (item.sourceSceneTrace ?? []).map((trace: any) => trace.obligationId).join(";"), item.mapping.pageId ?? "",
        item.mapping.stateId ?? "", item.mapping.region ?? "", item.mapping.sourceId ?? "",
        JSON.stringify((item.sourceSceneTrace ?? []).map((trace: any) => ({ page: trace.pageId, sources: trace.sourceIds }))),
        JSON.stringify((item.sourceSceneTrace ?? []).map((trace: any) => ({ page: trace.pageId, editable: trace.editableSourceIds }))),
        JSON.stringify((item.sourceSceneTrace ?? []).map((trace: any) => ({ page: trace.pageId, images: trace.sourceImages }))),
        item.disposition, item.behaviorProofStatus ?? "", item.proofBasisText ?? "", item.evidenceRefs.join(";"),
      ]);
      expect(requirementRows.slice(1)).toEqual(expectedRequirementRows);
      expect(requirementRows.slice(1).every((row) => row[18] === "not-verified")).toBe(true);
      expect(obligationRows).toHaveLength(103);
      expect(obligationRows[0]).toHaveLength(10);
      const expectedObligationRows = revised.features.map((item: any) => [
        item.id, item.name, item.pageId ?? "", (item.stateIds ?? []).join(";"), (item.sourceRequirementIds ?? []).join(";"),
        item.requirementIds.join(";"), (item.relatedSourceIds ?? [item.sourceRef]).join(";"),
        JSON.stringify(item.sourceImages ?? []), item.disposition, item.rationale,
      ]);
      expect(obligationRows.slice(1)).toEqual(expectedObligationRows);
      expect(exportedText(exported, "atlas/notification-hub-design-atlas.html")).toContain("Visual atlas pages");
      expect([...exported.keys()].filter((name) => /^atlas\/notification-hub-atlas-.+\.html$/.test(name))).toHaveLength(102);
      expect(exportedText(exported, "handoff/notification-hub-fresh-implementer-handoff.md")).toContain("Behavior proof is not verified for any requirement.");
      expect(exportedText(exported, "source-inputs/notification-hub-design-packet.json")).toBe(priorPacketFile!.bytes.toString("utf8"));
    } finally {
      timeout.mockRestore();
      click.mockRestore();
      if (createUrlDescriptor) Object.defineProperty(URL, "createObjectURL", createUrlDescriptor);
      else Reflect.deleteProperty(URL, "createObjectURL");
      if (revokeUrlDescriptor) Object.defineProperty(URL, "revokeObjectURL", revokeUrlDescriptor);
      else Reflect.deleteProperty(URL, "revokeObjectURL");
    }
  }, 120_000);
});
