import { webcrypto } from "node:crypto";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { applyBottomNavigationSource, buildImplementerHandoff, DesignAuthoringTool, makeLocalZip, makePacket } from "./DesignAuthoringTool";
import { addPacketRevision, canonicalJson, createSourceRecord, sha256Hex, validatePacket } from "./model";

beforeAll(() => {
  if (!globalThis.crypto?.subtle) Object.defineProperty(globalThis, "crypto", { configurable: true, value: webcrypto });
});

function fileBytes(file: Blob): Promise<ArrayBuffer> {
  if (typeof file.arrayBuffer === "function") return file.arrayBuffer();
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as ArrayBuffer);
    reader.onerror = () => reject(new Error("Could not read local ZIP bytes."));
    reader.readAsArrayBuffer(file);
  });
}

function fileText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ""));
    reader.onerror = () => reject(new Error("Could not read test file."));
    reader.readAsText(file);
  });
}

async function readStoredZip(blob: Blob): Promise<Map<string, string>> {
  const rawBytes = await fileBytes(blob);
  const bytes = new Uint8Array(rawBytes);
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let endRecord = -1;
  for (let offset = bytes.length - 22; offset >= Math.max(0, bytes.length - 65_557); offset -= 1) {
    if (view.getUint32(offset, true) === 0x06054b50) { endRecord = offset; break; }
  }
  if (endRecord < 0) throw new Error("Downloaded authoring package has no ZIP directory.");
  const fileCount = view.getUint16(endRecord + 10, true);
  const directoryOffset = view.getUint32(endRecord + 16, true);
  const decoder = new TextDecoder();
  const files = new Map<string, string>();
  let cursor = directoryOffset;
  for (let index = 0; index < fileCount; index += 1) {
    if (view.getUint32(cursor, true) !== 0x02014b50) throw new Error("Downloaded authoring package has a malformed ZIP directory.");
    const method = view.getUint16(cursor + 10, true);
    if (method !== 0) throw new Error(`Unsupported ZIP compression method ${method}.`);
    const size = view.getUint32(cursor + 24, true);
    const nameLength = view.getUint16(cursor + 28, true);
    const extraLength = view.getUint16(cursor + 30, true);
    const commentLength = view.getUint16(cursor + 32, true);
    const localOffset = view.getUint32(cursor + 42, true);
    const name = decoder.decode(bytes.subarray(cursor + 46, cursor + 46 + nameLength));
    const localNameLength = view.getUint16(localOffset + 26, true);
    const localExtraLength = view.getUint16(localOffset + 28, true);
    const start = localOffset + 30 + localNameLength + localExtraLength;
    files.set(name, decoder.decode(bytes.subarray(start, start + size)));
    cursor += 46 + nameLength + extraLength + commentLength;
  }
  return files;
}

const originalPrototype = `<style>@media(max-width:700px){.shell{display:block}}</style><div class="shell"><nav><button data-route="queue">Needs you</button></nav><main id="app"></main></div>`;

describe("offline ZIP package paths", () => {
  it.each(["../escape.html", "nested/../../escape.html", "/absolute.html", "C:\\absolute.html", "nested//empty.html", "nested/./dot.html", "bad\u0000name.html"])("rejects unsafe archive path %s", async (name) => {
    await expect(makeLocalZip([{ name, data: "source" }])).rejects.toThrow(/unsafe path/);
  });

  it("rejects duplicate paths and file-versus-directory collisions before writing", async () => {
    await expect(makeLocalZip([
      { name: "source/Prototype.html", data: "first" },
      { name: "source/prototype.html", data: "second" },
    ])).rejects.toThrow(/duplicate or colliding/);
    await expect(makeLocalZip([
      { name: "atlas", data: "file" },
      { name: "atlas/page.html", data: "nested" },
    ])).rejects.toThrow(/duplicate or colliding/);
    await expect(makeLocalZip([
      { name: "atlas/page.html", data: "nested" },
      { name: "atlas", data: "file" },
    ])).rejects.toThrow(/duplicate or colliding/);
  });

  it("rejects classic ZIP64 sentinel limits before reading or writing entries", async () => {
    const arrayBuffer = vi.fn(async () => new ArrayBuffer(0));
    const oversizedBlob = { size: 0xffffffff, arrayBuffer } as unknown as Blob;
    await expect(makeLocalZip([{ name: "source/large.bin", data: oversizedBlob }])).rejects.toThrow(/classic ZIP size limit/);
    expect(arrayBuffer).not.toHaveBeenCalled();

    const tooManyFiles = Array.from({ length: 0xffff }, (_, index) => ({ name: `source/${index}.txt`, data: "" }));
    await expect(makeLocalZip(tooManyFiles)).rejects.toThrow(/too many files/);
  });

  it("reads local ZIP Blob entries through FileReader when arrayBuffer is unavailable", async () => {
    const source = new Blob(["retained local source bytes"]);
    Object.defineProperty(source, "arrayBuffer", { configurable: true, value: undefined });
    const files = await readStoredZip(await makeLocalZip([{ name: "source-inputs/brief.md", data: source }]));
    expect(files.get("source-inputs/brief.md")).toBe("retained local source bytes");
  });

  it("writes the standard ZIP CRC32 value for local entries", async () => {
    const archive = await makeLocalZip([{ name: "source-inputs/crc.txt", data: "123456789" }]);
    const bytes = new Uint8Array(await fileBytes(archive));
    const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
    expect(view.getUint32(14, true)).toBe(0xcbf43926);
  });
});

describe("retained prototype feedback revision", () => {
  it("adds the complete five-destination mobile navigation without mutating v1", () => {
    const original = originalPrototype;
    const revised = applyBottomNavigationSource(original);

    expect(original).toBe(originalPrototype);
    expect(revised).toContain(`aria-label="Primary navigation"`);
    expect(revised).toContain(`data-route="queue">Needs you`);
    expect(revised).toContain(`data-route="devices">Reach`);
    expect(revised).toContain(`data-route="quiet">Quiet hours`);
    expect(revised).toContain(`data-route="order">Escalation`);
    expect(revised).toContain(`data-route="integration">Integrations`);
    expect(revised).toContain("position:fixed");
    expect(revised).toContain("@media(max-width:700px){.bottom-nav");
  });

  it("refuses to amend an unfamiliar prototype shell", () => {
    expect(() => applyBottomNavigationSource("<main>different source</main>")).toThrow(/five-destination shell/);
  });

  it("keeps a bundled font local and self-contained in the revised source", () => {
    const source = originalPrototype.replace("<style>", "<style>@font-face{font-family:Hub;src:url('fonts/SourceSans3.ttf')}");
    expect(() => applyBottomNavigationSource(source)).toThrow(/include the retained ZIP/);
    const revised = applyBottomNavigationSource(source, "data:font/ttf;base64,AA==");
    expect(revised).toContain("url('data:font/ttf;base64,AA==')");
    expect(revised).not.toContain("fonts/SourceSans3.ttf");
  });
});

describe("generic scenario packet intake", () => {
  it("reimports a legacy packet through exact source identity joins and preserves its v1/v2 archive", async () => {
    const imageBytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
    const image = new File([imageBytes], "sign-in.png", { type: "image/png" });
    const sourceMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| IDENTITY-P0-001 | Include sign-in | auth-ready | Access / Sign in | ui/auth.test.ts |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready | sign-in.png | Access / Sign in | IDENTITY-P0-001 |
`;
    const manifest = JSON.stringify([{ filename: image.name, sceneId: "access-sign-in", stateIds: ["auth-ready"], sha256: await sha256Hex(imageBytes), bytes: image.size, sourceRefs: ["Access / Sign in"] }]);
    const retainedInputs = [
      new File(["# Identity test brief\n\nRetained source only."], "identity-test-brief.md", { type: "text/markdown" }),
      new File([sourceMap], "identity-test-source-map.md", { type: "text/markdown" }),
      new File([manifest], "identity-test-images.json", { type: "application/json" }),
      image,
      new File(["<!doctype html><main>Editable source</main>"], "prototype.html", { type: "text/html" }),
      new File(["# Accepted surface\n\nE42 context."], "E42.md", { type: "text/markdown" }),
    ];
    const original = await makePacket(retainedInputs, undefined, "Identity Test");
    const v2Content = "<!doctype html><main>Editable v2 source</main>";
    const v2File = new File([v2Content], "prototype-v2-bottom-nav.html", { type: "text/html" });
    const v2Record = await createSourceRecord({
      name: v2File.name,
      origin: "prototype.html",
      version: "authored revision from v1 · correction SYNTHETIC-REIMPORT-01",
      bytes: v2Content,
      mediaType: "text/html",
      use: "Editable source amendment linked to the exact retained correction",
      amendmentHistory: ["Applied SYNTHETIC-REIMPORT-01 to v1; original source remains retained."],
    });
    const packetWithV2Source = { ...original.packet, sources: [...original.packet.sources, v2Record] };
    const archived = await addPacketRevision(packetWithV2Source, {
      revisionId: "v2-reviewed",
      createdAt: "2026-11-21T00:05:00Z",
      changeSummary: "Retained a version-bound source review.",
      sourceIds: [...original.packet.revisions[0]!.sourceIds, v2Record.id],
      decisionRefs: [{ authority: "operator", id: "SYNTHETIC-REIMPORT-01", source: "synthetic/decision.md", sourceTimestamp: "2026-11-21T00:04:00Z", quote: "Keep the original revision.", appliesToVersion: "v1" }],
    });
    const legacyArchive = structuredClone(archived);
    Reflect.deleteProperty(legacyArchive, "traceabilitySchema");
    const oldIdByCurrentId = new Map<string, string>();
    for (const source of legacyArchive.sources) {
      const currentId = source.id;
      const legacyId = `src-${source.sha256.slice(0, 16)}`;
      if (legacyArchive.sources.some((candidate) => candidate !== source && candidate.sha256.slice(0, 16) === source.sha256.slice(0, 16))) throw new Error("Fixture source hashes must stay unique under the legacy ID width.");
      oldIdByCurrentId.set(currentId, legacyId);
      source.id = legacyId;
      delete (source as Partial<typeof source>).identityBinding;
    }
    for (const revision of legacyArchive.revisions) {
      revision.sourceIds = revision.sourceIds.map((id) => oldIdByCurrentId.get(id)!);
      for (const page of revision.pages) {
        page.sourceRefs = page.sourceRefs.map((id) => oldIdByCurrentId.get(id) ?? id);
        page.editableSourceIds = page.editableSourceIds.map((id) => oldIdByCurrentId.get(id) ?? id);
      }
      for (const requirement of revision.requirements) requirement.sourceRefs = requirement.sourceRefs.map((id) => oldIdByCurrentId.get(id) ?? id);
      for (const feature of revision.features) feature.sourceRef = oldIdByCurrentId.get(feature.sourceRef) ?? feature.sourceRef;
      const { contentSha256, ...revisionBody } = revision;
      revision.contentSha256 = await sha256Hex(canonicalJson(revisionBody));
    }
    const archiveText = JSON.stringify(legacyArchive);
    const archiveFile = new File([archiveText], "identity-test-design-packet.json", { type: "application/json" });
    const extraReference = new File(["# Additional source\n\nRetained with exact bytes."], "additional-review.md", { type: "text/markdown" });

    const rebound = await makePacket([...retainedInputs, v2File, archiveFile, extraReference], undefined, "Identity Test");
    expect(rebound.packetArchiveRecovery).toMatchObject({ packetId: archived.id, preservedCurrentRevision: "v2-reviewed", archiveSha256: await sha256Hex(archiveText) });
    expect(rebound.packet.revisions[0]?.id).toBe("v1");
    expect(rebound.packet.sources.find((source) => source.name === archiveFile.name)).toMatchObject({ sha256: await sha256Hex(archiveText), identityBinding: expect.any(String) });
    expect(rebound.packet.sources.find((source) => source.name === extraReference.name)).toMatchObject({ sha256: await sha256Hex(await fileText(extraReference)), bytes: extraReference.size, use: "Supporting source document; exact bytes retained as context, not treated as product acceptance" });
    expect(rebound.packet.sources.find((source) => source.name === v2File.name)).toMatchObject({ id: v2Record.id, origin: "prototype.html", version: v2Record.version, sha256: v2Record.sha256, amendmentHistory: v2Record.amendmentHistory });
    expect(await fileText(archiveFile)).toBe(archiveText);
    expect(legacyArchive.revisions.map((revision) => [revision.id, revision.parentId])).toEqual([["v1", null], ["v2-reviewed", "v1"]]);
    expect(legacyArchive.revisions[1]!.decisionRefs[0]).toMatchObject({ id: "SYNTHETIC-REIMPORT-01", appliesToVersion: "v1" });

    const staleArchive = structuredClone(legacyArchive);
    staleArchive.sources[0]!.version = "stale-version-with-same-content-id";
    const staleArchiveFile = new File([JSON.stringify(staleArchive)], archiveFile.name, { type: "application/json" });
    await expect(makePacket([...retainedInputs, staleArchiveFile], undefined, "Identity Test")).rejects.toThrow(/source identity re-import failed closed.*No re-imported source matches/i);
  });

  it("joins generic IDs and blocks export for source-map/manifest scene mismatches", async () => {
    const signInBytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
    const orphanBytes = new Uint8Array([1, 2, 3]);
    const signIn = new File([signInBytes], "sign-in.png", { type: "image/png" });
    const orphan = new File([orphanBytes], "unlisted.png", { type: "image/png" });
    const map = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| ADMIN-P0-001 | Sign-in flow | auth-ready | Access / Sign in | ui/auth.test.ts |
| ADMIN-P0-002 | Recovery flow | auth-recovery | Access / Recovery | ui/recovery.test.ts |
| PLATFORM-SECURE-02 | No UI screen/state; server policy | No UI screen/state; API configuration | Access / Session policy | api/session_test.go |
| NOTIFICA-P2-002 / OT-P2-002 | P2 future requirement; not selected as a current visual/product surface. | UNMAPPED: no SMS scene/state. | No selected source heading for SMS | no validation in this packet |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready, auth-recovery | sign-in.png | Access / Sign in<br>Access / Recovery | ADMIN-P0-001, ADMIN-P0-002 |
| settings-session | settings-session | no selected image | Access / Session policy | PLATFORM-SECURE-02 |
`;
    const manifest = JSON.stringify([
      { filename: signIn.name, sceneId: "access-sign-in", stateIds: ["auth-ready", "auth-recovery"], sha256: await sha256Hex(signInBytes), bytes: signIn.size, sourceRefs: ["Access / Sign in", "Access / Recovery"] },
      { filename: orphan.name, sceneId: "unlisted-scene", stateIds: ["orphan-summary"], sha256: await sha256Hex(orphanBytes), bytes: orphan.size, sourceRefs: ["Dashboard / Summary"] },
    ]);
    const packet = await makePacket([
      new File(["# Generic brief\n\nAuthoring-only source."], "generic-scenario-brief.md", { type: "text/markdown" }),
      new File([map], "generic-source-map.md", { type: "text/markdown" }),
      new File([manifest], "generic-images.json", { type: "application/json" }),
      signIn,
      orphan,
      new File(["<!doctype html><main>editable source</main>"], "prototype.html", { type: "text/html" }),
    ], undefined, "Atlas Example");
    const revision = packet.packet.revisions[0]!;

    expect(packet.packet.scenario).toBe("Atlas Example");
    expect(packet.packet.sources.find((source) => source.name === "generic-scenario-brief.md")?.version).toBe("retained Atlas Example brief/design source; exact imported bytes");
    expect(revision.requirements.find((item) => item.id === "ADMIN-P0-001")?.mapping).toMatchObject({ pageId: "access-sign-in", stateId: "auth-ready" });
    expect(revision.requirements.find((item) => item.id === "ADMIN-P0-002")?.mapping).toMatchObject({ pageId: "access-sign-in", stateId: "auth-recovery" });
    expect(revision.requirements.find((item) => item.id === "PLATFORM-SECURE-02")?.coverage).toBe("nonvisual");
    expect(revision.requirements.find((item) => item.id === "NOTIFICA-P2-002 / OT-P2-002")).toMatchObject({ coverage: "nonvisual", disposition: "defer" });
    expect(revision.features.find((item) => item.id === "source-scene-settings-session")?.disposition).toBe("unknown");
    expect(revision.features.find((item) => item.id === "source-scene-unlisted-scene")?.disposition).toBe("unknown");
    const handoff = buildImplementerHandoff(packet.packet);
    expect(handoff).toContain("| Obligation ID | Atlas page | States | Source requirement IDs | Linked packet requirements | Exact source IDs | Selected filename/hash/source ID/state | Disposition | Basis |");
    const handoffObligationIDs = handoff.split("\n").flatMap((line) => {
      const match = line.match(/^\| (source-scene-[^|]+) \|/);
      return match ? [match[1]] : [];
    });
    expect(handoffObligationIDs).toEqual(revision.features.map((item) => item.id));
    const validation = validatePacket(packet.packet);
    expect(validation.issues.map((issue) => issue.path)).toEqual(expect.arrayContaining([
      "features.source-scene-settings-session.disposition",
      "features.source-scene-unlisted-scene.disposition",
    ]));
    expect(validation.valid).toBe(false);
  });
});

describe("complete authoring journey", () => {
  it("records the existing Hub correction together with staged source mappings", async () => {
    const imageABytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10, 1]);
    const imageBBytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10, 2]);
    const imageA = new File([imageABytes], "ask-review.png", { type: "image/png" });
    const imageB = new File([imageBBytes], "ask-recorded.png", { type: "image/png" });
    const sourceMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| NH-AC-20 | Preserve the answer journey | NH14.2, NH14.3 | Answer / Review | ui/answers.test.ts |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| ask-review | NH14.2 | ask-review.png | Answer / Review | NH-AC-20 |
| ask-recorded | NH14.3 | ask-recorded.png | Answer / Recorded | NH-AC-20 |
`;
    const manifest = JSON.stringify([
      { filename: imageA.name, sceneId: "ask-review", stateIds: ["NH14.2"], sha256: await sha256Hex(imageABytes), bytes: imageA.size, sourceRefs: ["Answer / Review"] },
      { filename: imageB.name, sceneId: "ask-recorded", stateIds: ["NH14.3"], sha256: await sha256Hex(imageBBytes), bytes: imageB.size, sourceRefs: ["Answer / Recorded"] },
    ]);
    const sourceBundle = await makeLocalZip([
      { name: "prototype.html", data: originalPrototype },
      { name: "fonts/SourceSans3.ttf", data: "test-font-bytes" },
    ]);
    render(<DesignAuthoringTool onClose={() => {}} />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "Notification Hub" } });
    fireEvent.change(screen.getByLabelText("Design brief, source map, supporting Markdown, manifest, selected images and editable source"), { target: { files: [
      new File(["# Notification Hub"], "notification-hub-brief.md", { type: "text/markdown" }),
      new File([sourceMap], "notification-hub-source-map.md", { type: "text/markdown" }),
      new File([manifest], "notification-hub-images.json", { type: "application/json" }),
      imageA, imageB,
      new File([sourceBundle], "notification-hub-source.zip", { type: "application/zip" }),
      new File([originalPrototype], "prototype.html", { type: "text/html" }),
    ] } });
    fireEvent.click(screen.getByRole("button", { name: "Validate and build packet" }));
    await screen.findByRole("status");
    fireEvent.click(screen.getByRole("button", { name: /Mappings & omissions/ }));
    const row = screen.getByLabelText("Exact source frame for NH-AC-20").closest("details")!;
    fireEvent.click(row.querySelector("summary")!);
    const select = screen.getByLabelText("Exact source frame for NH-AC-20") as HTMLSelectElement;
    const selected = [...select.options].find((option) => option.textContent?.includes("ask-review.png"));
    expect(selected).toBeTruthy();
    fireEvent.change(select, { target: { value: selected!.value } });
    fireEvent.click(screen.getByRole("button", { name: /Edit & revise/ }));
    fireEvent.click(screen.getByRole("button", { name: "Apply correction to this source" }));

    await waitFor(() => expect(screen.getByText(/Created immutable v2 linked to operator decision NH-DESIGN-20261025-01/)).toBeInTheDocument());
    expect(screen.getByText(/Created immutable v2 linked to operator decision NH-DESIGN-20261025-01 and its exact v1 parent/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /Mappings & omissions/ }));
    expect(screen.getByText(/Current: ask-review · NH14\.2/)).toBeInTheDocument();
  });

  it("moves from source intake through mapping, atlas review, revision, and export readiness", async () => {
    const imageBytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
    const image = new File([imageBytes], "sign-in.png", { type: "image/png" });
    const sourceMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| ADMIN-P0-001 | Sign-in path | auth-ready | Access / Sign in | ui/auth.test.ts |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready | sign-in.png | Access / Sign in | ADMIN-P0-001 |
`;
    const manifest = JSON.stringify([{ filename: image.name, sceneId: "access-sign-in", stateIds: ["auth-ready"], sha256: await sha256Hex(imageBytes), bytes: image.size, sourceRefs: ["Access / Sign in"] }]);
    const onOpenArtifact = () => {};
    render(<DesignAuthoringTool sessionId="session-test" onOpenArtifact={onOpenArtifact} onClose={() => {}} />);
    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "Atlas Example" } });
    fireEvent.change(screen.getByLabelText("Design brief, source map, supporting Markdown, manifest, selected images and editable source"), { target: { files: [
      new File(["# Atlas Example"], "atlas-example-brief.md", { type: "text/markdown" }),
      new File([sourceMap], "atlas-example-source-map.md", { type: "text/markdown" }),
      new File([manifest], "atlas-example-images.json", { type: "application/json" }),
      image,
      new File(["<!doctype html><main>editable source v1</main>"], "prototype.html", { type: "text/html" }),
    ] } });

    fireEvent.click(screen.getByRole("button", { name: "Validate and build packet" }));
    await screen.findByRole("status");
    expect(screen.getByRole("button", { name: /Mappings & omissions/ })).toHaveAttribute("aria-current", "step");
    expect(screen.getByRole("button", { name: /Inspect atlas/ })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: /Mappings & omissions/ }));
    expect(screen.getAllByText(/ADMIN-P0-001/).length).toBeGreaterThan(0);
    expect(screen.getByText(/Selected design frame \(sign-in.png\)/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Inspect atlas/ }));
    expect(await screen.findByRole("img", { name: "access-sign-in · auth-ready" })).toBeInTheDocument();
    expect(screen.getByText(/SHA-256/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Edit & revise/ }));
    fireEvent.change(screen.getByLabelText("Editable HTML source"), { target: { value: "<!doctype html><main>editable source v2</main>" } });
    fireEvent.click(screen.getByRole("button", { name: "Create immutable revision" }));
    await waitFor(() => expect(screen.getByText(/Created immutable v2 linked to v1/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: /Export & hand off/ }));
    expect(screen.getByText(/Trace complete: 1 requirements and 1 source-scene obligations; behavior proof is unverified for all 1/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save packet, atlas and handoff" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Open atlas print view \/ PDF" })).toBeEnabled();
  });

  it("downloads an offline Hub packet with its version lineage, atlas, mappings and omission register", async () => {
    const imageBytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10, 11]);
    const image = new File([imageBytes], "ask-review.png", { type: "image/png" });
    const sourceMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| NH-AC-20 | Preserve the answer journey | NH14.2 | Answer / Review | ui/answers.test.ts |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| ask-review | NH14.2 | ask-review.png | Answer / Review | NH-AC-20 |
`;
    const manifest = JSON.stringify([{ filename: image.name, sceneId: "ask-review", stateIds: ["NH14.2"], sha256: await sha256Hex(imageBytes), bytes: image.size, sourceRefs: ["Answer / Review"] }]);
    const sourceZip = await makeLocalZip([
      { name: "prototype.html", data: originalPrototype },
      { name: "fonts/SourceSans3.ttf", data: "synthetic-notification-hub-test-font" },
    ]);
    let downloaded: Blob | undefined;
    let filename = "";
    const createUrlDescriptor = Object.getOwnPropertyDescriptor(URL, "createObjectURL");
    const revokeUrlDescriptor = Object.getOwnPropertyDescriptor(URL, "revokeObjectURL");
    const originalSetTimeout = window.setTimeout;
    let timeoutMock: { mockRestore: () => void } | undefined;
    Object.defineProperty(URL, "createObjectURL", { configurable: true, value: vi.fn((blob: Blob) => { downloaded = blob; return "blob:local-authoring-package"; }) });
    Object.defineProperty(URL, "revokeObjectURL", { configurable: true, value: vi.fn() });
    const click = vi.spyOn(HTMLAnchorElement.prototype, "click").mockImplementation(function (this: HTMLAnchorElement) { filename = this.download; });

    try {
      render(<DesignAuthoringTool onClose={() => {}} />);
      expect(screen.getByText(/Local files only; no session API/)).toBeInTheDocument();
      fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "Notification Hub" } });
      fireEvent.change(screen.getByLabelText("Design brief, source map, supporting Markdown, manifest, selected images and editable source"), { target: { files: [
        new File(["# Notification Hub"], "notification-hub-brief.md", { type: "text/markdown" }),
        new File([sourceMap], "notification-hub-source-map.md", { type: "text/markdown" }),
        new File([manifest], "notification-hub-images.json", { type: "application/json" }),
        image,
        new File([sourceZip], "notification-hub-source.zip", { type: "application/zip" }),
        new File([originalPrototype], "prototype.html", { type: "text/html" }),
      ] } });
      fireEvent.click(screen.getByRole("button", { name: "Validate and build packet" }));
      await screen.findByText(/Imported 1 hash-checked visual source/);

      fireEvent.click(screen.getByRole("button", { name: /Mappings & omissions/ }));
      const mappingDetails = screen.getByLabelText("Exact source frame for NH-AC-20").closest("details")!;
      fireEvent.click(mappingDetails.querySelector("summary")!);
      const frame = screen.getByLabelText("Exact source frame for NH-AC-20") as HTMLSelectElement;
      fireEvent.change(frame, { target: { value: JSON.stringify(["ask-review", "NH14.2", image.name, await sha256Hex(imageBytes)]) } });

      fireEvent.click(screen.getByRole("button", { name: /Edit & revise/ }));
      fireEvent.click(screen.getByRole("button", { name: "Apply correction to this source" }));
      await screen.findByText(/Created immutable .* linked to operator decision NH-DESIGN-20261025-01/);
      fireEvent.click(screen.getByRole("button", { name: /Export & hand off/ }));
      expect(screen.getByRole("button", { name: "Download local packet and atlas package" })).toBeEnabled();

      timeoutMock = vi.spyOn(window, "setTimeout").mockImplementation((handler: TimerHandler, delay = 0, ...args: unknown[]) => {
        if (delay === 30_000 && typeof handler === "function") { (handler as (...values: unknown[]) => unknown)(...args); return 0 as unknown as ReturnType<typeof window.setTimeout>; }
        return Reflect.apply(originalSetTimeout, window, [handler, delay, ...args]) as ReturnType<typeof window.setTimeout>;
      });
      fireEvent.click(screen.getByRole("button", { name: "Download local packet and atlas package" }));
      await waitFor(() => expect(downloaded).toBeInstanceOf(Blob));
      expect(filename).toBe("notification-hub-authoring-export.zip");
      expect(click).toHaveBeenCalledOnce();
      expect(screen.getByText(/No session or upload API was used/)).toBeInTheDocument();

      const files = await readStoredZip(downloaded!);
      const packet = JSON.parse(files.get("notification-hub-design-packet.json")!);
      const revision = packet.revisions.find((row: { id: string }) => row.id === packet.currentRevisionId);
      expect(revision.decisionRefs[0]).toMatchObject({ id: "NH-DESIGN-20261025-01", appliesToVersion: "v1" });
      expect(files.has("editable/notification-hub-prototype-v1.html")).toBe(true);
      expect([...files.keys()].some((path) => /^editable\/notification-hub-prototype-v2.*\.html$/.test(path))).toBe(true);
      expect(files.get("atlas/notification-hub-design-atlas.html")).toContain("Visual atlas pages");
      expect([...files.keys()].some((path) => path.startsWith("atlas/notification-hub-atlas-ask-review.html"))).toBe(true);
      expect(files.get("trace/notification-hub-requirements.csv")).toContain("NH-AC-20");
      expect(files.get("trace/notification-hub-omissions.csv")).toContain("source-scene-ask-review");
      expect(files.get("handoff/notification-hub-fresh-implementer-handoff.md")).toContain("source-scene-ask-review");
      expect(files.get("source-inputs/notification-hub-source-map.md")).toContain("Appendix A");
    } finally {
      timeoutMock?.mockRestore();
      click.mockRestore();
      if (createUrlDescriptor) Object.defineProperty(URL, "createObjectURL", createUrlDescriptor);
      else Reflect.deleteProperty(URL, "createObjectURL");
      if (revokeUrlDescriptor) Object.defineProperty(URL, "revokeObjectURL", revokeUrlDescriptor);
      else Reflect.deleteProperty(URL, "revokeObjectURL");
    }
  });
});

