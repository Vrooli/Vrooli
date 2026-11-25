import { webcrypto } from "node:crypto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";
import { DesignAuthoringTool, makePacket } from "./DesignAuthoringTool";
import { canonicalJson, sha256Hex, verifyPacketIntegrity, type DesignPacket, type PacketSource } from "./model";

beforeAll(() => {
  if (!globalThis.crypto?.subtle) Object.defineProperty(globalThis, "crypto", { configurable: true, value: webcrypto });
});

function fileText(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result ?? ""));
    reader.onerror = () => reject(new Error("Could not read test file."));
    reader.readAsText(file);
  });
}

describe("source mapping and omission review", () => {
  it("preserves a prior packet archive and rebinds its exact bytes through the model source record", async () => {
    const imageBytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
    const image = new File([imageBytes], "sign-in.png", { type: "image/png" });
    const sourceMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| ADMIN-P0-001 | Sign-in journey | auth-ready | Access / Sign in | ui/auth.test.ts |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready | sign-in.png | Access / Sign in | ADMIN-P0-001 |
`;
    const manifest = JSON.stringify([{ filename: image.name, sceneId: "access-sign-in", stateIds: ["auth-ready"], sha256: await sha256Hex(imageBytes), bytes: image.size, sourceRefs: ["Access / Sign in"] }]);
    const baseFiles = [
      new File(["# Atlas Example"], "atlas-example-brief.md", { type: "text/markdown" }),
      new File([sourceMap], "atlas-example-source-map.md", { type: "text/markdown" }),
      new File([manifest], "atlas-example-selected-image-manifest.json", { type: "application/json" }),
      image,
      new File(["<!doctype html><main>version one</main>"], "notification-prototype-v1.html", { type: "text/html" }),
      new File(["<!doctype html><main>stale later source</main>"], "notification-prototype-v2.html", { type: "text/html" }),
    ];
    const oldPacket = await makePacket(baseFiles, undefined, "Atlas Example");
    const originalPacketText = JSON.stringify(oldPacket.packet, null, 2);
    const legacyPacket = JSON.parse(originalPacketText) as Omit<DesignPacket, "sources"> & { sources: Array<Omit<PacketSource, "identityBinding"> & { identityBinding?: string }> };
    Reflect.deleteProperty(legacyPacket, "traceabilitySchema");
    legacyPacket.sources.forEach((source) => { source.id = `src-${source.sha256.slice(0, 16)}`; delete source.identityBinding; });
    const reboundSourceId = new Map(legacyPacket.sources.map((source, index) => [oldPacket.packet.sources[index]!.id, source.id]));
    for (const revision of legacyPacket.revisions) {
      revision.sourceIds = revision.sourceIds.map((id) => reboundSourceId.get(id) ?? id);
      for (const page of revision.pages) {
        page.sourceRefs = page.sourceRefs.map((id) => reboundSourceId.get(id) ?? id);
        page.editableSourceIds = page.editableSourceIds.map((id) => reboundSourceId.get(id) ?? id);
      }
      for (const requirement of revision.requirements) requirement.sourceRefs = requirement.sourceRefs.map((id) => reboundSourceId.get(id) ?? id);
      for (const feature of revision.features) feature.sourceRef = reboundSourceId.get(feature.sourceRef) ?? feature.sourceRef;
      const { contentSha256: _oldContentSha256, ...revisionBody } = revision;
      revision.contentSha256 = await sha256Hex(canonicalJson(revisionBody));
    }
    const archiveText = JSON.stringify(legacyPacket, null, 2);
    const archive = new File([archiveText], "atlas-example-design-packet.json", { type: "application/json" });
    const recovered = await makePacket([...baseFiles, archive], undefined, "Atlas Example");
    const archiveSource = recovered.packet.sources.find((source) => source.name === archive.name);

    expect(recovered.packet.id).toContain("-rebound-");
    expect(recovered.packetArchiveRecovery).toMatchObject({ packetId: oldPacket.packet.id, preservedCurrentRevision: "v1" });
    expect(recovered.editableOriginal?.name).toBe("notification-prototype-v1.html");
    expect(archiveSource).toBeDefined();
    expect(archiveSource?.sha256).toBe(await sha256Hex(archiveText));
    expect(archiveSource?.bytes).toBe(new TextEncoder().encode(archiveText).byteLength);
    expect(archiveSource?.mediaType).toBe("application/json");
    expect(archiveSource?.identityBinding).toBe(canonicalJson({ name: archiveSource?.name, origin: archiveSource?.origin, version: archiveSource?.version, sha256: archiveSource?.sha256, bytes: archiveSource?.bytes, mediaType: archiveSource?.mediaType }));
    expect(recovered.packet.revisions[0]?.sourceIds).toContain(archiveSource?.id);
    expect(recovered.packet.revisions[0]?.requirements[0]?.sourceRefs).toContain(archiveSource?.id);
    expect(await fileText(archive)).toBe(archiveText);
    expect((await verifyPacketIntegrity(recovered.packet)).valid).toBe(true);
    expect(JSON.stringify(oldPacket.packet, null, 2)).toBe(originalPacketText);

    const stalePacket = JSON.parse(archiveText) as DesignPacket;
    const staleRevision = stalePacket.revisions[0]!;
    staleRevision.requirements[0]!.mapping.pageId = "stale-scene";
    const staleSeed = JSON.parse(JSON.stringify(staleRevision));
    delete staleSeed.contentSha256;
    staleRevision.contentSha256 = await sha256Hex(canonicalJson(staleSeed));
    const staleArchive = new File([JSON.stringify(stalePacket)], "atlas-example-design-packet.json", { type: "application/json" });
    await expect(makePacket([...baseFiles, staleArchive], undefined, "Atlas Example")).rejects.toThrow(/failed model integrity|does not exactly reproduce the archived/);
    await expect(makePacket([...baseFiles, new File([imageBytes], image.name, { type: "image/png" })], undefined, "Atlas Example")).rejects.toThrow(/duplicate filenames/);
    await expect(makePacket([...baseFiles, new File(["zip a"], "source-a.zip"), new File(["zip b"], "source-b.zip")], undefined, "Atlas Example")).rejects.toThrow(/one retained source ZIP/);
  });

  it("fails closed before packet creation when a source-map filename is absent from the exact manifest", async () => {
    const bytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
    const image = new File([bytes], "actual-frame.png", { type: "image/png" });
    const sourceMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| ADMIN-P0-001 | Sign-in path | auth-ready | Access / Sign in | ui/auth.test.ts |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready | missing-frame.png | Access / Sign in | ADMIN-P0-001 |
`;
    const manifest = JSON.stringify([{ filename: image.name, sceneId: "access-sign-in", stateIds: ["auth-ready"], sha256: await sha256Hex(bytes), bytes: image.size, sourceRefs: ["Access / Sign in"] }]);
    await expect(makePacket([
      new File(["# Atlas Example"], "atlas-example-brief.md", { type: "text/markdown" }),
      new File([sourceMap], "atlas-example-source-map.md", { type: "text/markdown" }),
      new File([manifest], "atlas-example-images.json", { type: "application/json" }),
      image,
      new File(["<!doctype html><main>editable</main>"], "prototype.html", { type: "text/html" }),
    ], undefined, "Atlas Example")).rejects.toThrow(/Source map and selected-image manifest do not join exactly/);
  });

  it("resolves an ambiguous exact frame, records dispositions in an immutable revision, and gates export until the source revision exists", async () => {
    const originalBytes = new Uint8Array([137, 80, 78, 71, 13, 10, 26, 10]);
    const alternateBytes = new Uint8Array([137, 80, 78, 71, 0, 1, 2, 3]);
    const original = new File([originalBytes], "sign-in.png", { type: "image/png" });
    const alternate = new File([alternateBytes], "sign-in-alt.png", { type: "image/png" });
    const sourceMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| ADMIN-P0-001 | Sign-in journey | auth-ready | Access / Sign in | ui/auth.test.ts |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready | sign-in.png<br/>sign-in-alt.png | Access / Sign in | ADMIN-P0-001 |
`;
    const manifest = JSON.stringify([
      { filename: original.name, sceneId: "access-sign-in", stateIds: ["auth-ready"], sha256: await sha256Hex(originalBytes), bytes: original.size, sourceRefs: ["Access / Sign in"] },
      { filename: alternate.name, sceneId: "access-sign-in", stateIds: ["auth-ready"], sha256: await sha256Hex(alternateBytes), bytes: alternate.size, sourceRefs: ["Access / Sign in"] },
    ]);
    render(<DesignAuthoringTool sessionId="session-test" onOpenArtifact={() => {}} onClose={() => {}} />);

    fireEvent.change(screen.getByLabelText("Scenario name"), { target: { value: "Atlas Example" } });
    fireEvent.change(screen.getByLabelText("Design brief, source map, supporting Markdown, manifest, selected images and editable source"), { target: { files: [
      new File(["# Atlas Example"], "atlas-example-brief.md", { type: "text/markdown" }),
      new File([sourceMap], "atlas-example-source-map.md", { type: "text/markdown" }),
      new File([manifest], "atlas-example-images.json", { type: "application/json" }),
      original,
      alternate,
      new File(["<!doctype html><main>editable source v1</main>"], "prototype.html", { type: "text/html" }),
    ] } });
    fireEvent.click(screen.getByRole("button", { name: "Validate and build packet" }));
    expect(await screen.findByText(/Imported 2 hash-checked visual sources/)).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Mappings & omissions/ }));
    fireEvent.click(screen.getByText("Edit coverage, mapping and disposition"));
    const framePicker = screen.getByLabelText("Exact source frame for ADMIN-P0-001") as HTMLSelectElement;
    expect(framePicker).toHaveValue("");
    const alternateHash = await sha256Hex(alternateBytes);
    fireEvent.change(framePicker, { target: { value: JSON.stringify(["access-sign-in", "auth-ready", alternate.name, alternateHash]) } });
    expect(screen.getByText(/Current: access-sign-in · auth-ready · Selected design frame \(sign-in-alt.png\)/)).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Requirement disposition for ADMIN-P0-001"), { target: { value: "defer" } });
    const requirementBasis = screen.getByLabelText("Mapping basis for ADMIN-P0-001") as HTMLTextAreaElement;
    fireEvent.change(requirementBasis, { target: { value: `${requirementBasis.value}\nDisposition rationale: The retained packet marks this route as deferred for this authoring scope.` } });
    const obligationId = "source-scene-access-sign-in";
    fireEvent.change(screen.getByLabelText(`Disposition for ${obligationId}`), { target: { value: "defer" } });
    const obligationBasis = screen.getByLabelText(`Disposition basis for ${obligationId}`) as HTMLTextAreaElement;
    fireEvent.change(obligationBasis, { target: { value: `${obligationBasis.value}\nDisposition rationale: Preserve this source obligation as an explicit handoff deferral.` } });
    expect(screen.getByRole("button", { name: "Record mapping and omission revision" })).toBeEnabled();
    fireEvent.click(screen.getByRole("button", { name: "Record mapping and omission revision" }));
    await waitFor(() => expect(screen.getByText(/Recorded an immutable v2-review review revision from v1/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: /Edit & revise/ }));
    fireEvent.change(screen.getByLabelText("Editable HTML source"), { target: { value: "<!doctype html><main>editable source v2</main>" } });
    fireEvent.click(screen.getByRole("button", { name: /Export & hand off/ }));
    expect(screen.getByText(/editor changes need a revision/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save packet, atlas and handoff" })).toBeDisabled();
    fireEvent.click(screen.getByRole("button", { name: /Edit & revise/ }));
    fireEvent.click(screen.getByRole("button", { name: "Create immutable revision" }));
    await waitFor(() => expect(screen.getByText(/Created immutable v3 linked to v2-review/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole("button", { name: /Export & hand off/ }));
    expect(screen.getByText(/Trace complete: 1 requirements and 1 source-scene obligations; behavior proof is unverified for all 1; editable v3 is linked/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save packet, atlas and handoff" })).toBeEnabled();
  });
});
