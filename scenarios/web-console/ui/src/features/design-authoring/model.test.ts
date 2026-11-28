import { webcrypto } from "node:crypto";
import { beforeAll, describe, expect, it } from "vitest";

import {
  addPacketRevision,
  buildAtlasHtml,
  createDesignPacket,
  createSourceRecord,
  parseRequirementSourceMap,
  parseSceneSourceMap,
  parseSelectedImageManifest,
  sha256Hex,
  matchingAssetStates,
  resolveSelectedState,
  sourceSceneIdFromObligationId,
  sourceSceneObligationId,
  sourceRequirementAliases,
  sourceScenesForRequirement,
  verifySourceReimportIdentity,
  validateSceneManifestJoins,
  verifyPacketIntegrity,
  validatePacket,
  TRACEABILITY_CONTRACT_SCHEMA,
  type DesignPacket,
  type PacketRequirement,
} from "./model";

beforeAll(() => {
  if (!globalThis.crypto?.subtle) Object.defineProperty(globalThis, "crypto", { configurable: true, value: webcrypto });
});

const sourceMap = `# Source map

## Requirement crosswalk

| Requirement / PRD | Scope disposition | Selected scene/state families | Dossier source headings / NH-AC | Existing validation or unimplemented proof |
|---|---|---|---|---|
| \`NOTIFICA-P0-001\` / \`OT-P0-001\` | In selected scope: device delivery is included. | reach-enroll-*; NH16.4–NH16.6 | NH-AC-27, NH09 — Reach: device lifecycle | test: \`api/internal/hub/service_test.go::TestSend\` |
| \`NOTIFICA-P0-009\` / \`OT-P0-009\` | Preserve the service portability rule. | No UI scene/state; \`.vrooli/service.json\` is the proof surface. | No selected UI source; no visual scene expected. | manual: \`.vrooli/service.json\` |

## Scene appendix
`;

const appendix = `## Appendix A — Asset membership and source resolution

| Scene family | State IDs | Manifest filenames (all selected items) | Exact source headings | Related registry requirement(s) / explicit dossier design obligation |
|---|---|---|---|---|
| \`NH13.1\` | \`NH13.1\` | \`NH13.1-dark-phone-p1-r2.png\`<br>\`NH13.1-light-phone-p1-r2.png\` | Experience / Information architecture | NOTIFICA-P0-007; NH-AC-34/36 (derived correspondence). |
`;

const source = {
  id: "src-aaaaaaaaaaaaaaaa-023ea9b55f4b",
  name: "source-map.md",
  origin: "/private/hub/source-map.md",
  version: "receipt-207",
  sha256: "a".repeat(64),
  bytes: 8,
  mediaType: "text/markdown",
  use: "Requirement source and initial mappings",
  amendmentHistory: [],
  identityBinding: `{"bytes":8,"mediaType":"text/markdown","name":"source-map.md","origin":"/private/hub/source-map.md","sha256":"${"a".repeat(64)}","version":"receipt-207"}`,
};

function visualRequirement(overrides: Partial<PacketRequirement> = {}): PacketRequirement {
  return {
    id: "NOTIFICA-P0-001 / OT-P0-001",
    text: "Device delivery",
    scope: "In selected scope",
    coverage: "visual",
    mapping: { pageId: "reach", stateId: "reach-enroll-ready", region: "enrollment form", rationale: "Source-mapped selected state." },
    sourceRefs: [source.id],
    evidenceRefs: [],
    disposition: "preserve",
    ...overrides,
  };
}

describe("design authoring packet lineage", () => {
  it("blocks trace-contract exports until the current requirement retains source IDs and behavior-proof status", async () => {
    const packet = await createDesignPacket({
      id: "packet-trace-gate",
      scenario: "Notification Hub",
      brief: "A trace-contract gate fixture.",
      sources: [source],
      pages: [{ id: "reach", title: "Reach", stateIds: ["reach-ready"], sourceRefs: [source.id], regions: ["reach.png"], editableSourceIds: [] }],
      requirements: [visualRequirement()],
      features: [{ id: "feature-reach", name: "Reach surface", sourceRef: source.id, requirementIds: [], disposition: "preserve", rationale: "Fixture." }],
      createdAt: "2026-11-22T00:00:00Z",
      traceabilitySchema: TRACEABILITY_CONTRACT_SCHEMA,
    });
    const messages = validatePacket(packet).issues.map((issue) => `${issue.path}: ${issue.message}`);
    expect(messages).toContain("revisions.v1.requirements.NOTIFICA-P0-001 / OT-P0-001.sourceRequirementIds: Exact source-map requirement aliases are required.");
    expect(messages).toContain("revisions.v1.requirements.NOTIFICA-P0-001 / OT-P0-001.behaviorProofStatus: Design-source evidence must remain explicitly separate from verified behavior.");
    expect(messages).toContain("revisions.v1.requirements.NOTIFICA-P0-001 / OT-P0-001.sourceSceneTrace: Visual requirement needs at least one exact source-scene/page/state trace.");
  });

  it("hashes exact source bytes and records their purpose", async () => {
    const record = await createSourceRecord({
      name: "brief.md",
      origin: "/private/brief.md",
      version: "v1",
      bytes: "hello",
      mediaType: "text/markdown",
      use: "Authoring input",
    });
    expect(record.sha256).toBe("2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824");
    expect(record.bytes).toBe(5);
    expect(record.id).toMatch(new RegExp(`^src-${record.sha256.slice(0, 16)}-[a-f0-9]{12}$`));
  });

  it("rebinds a legacy source only through an exact, unique import while preserving archived v1/v2", async () => {
    const currentSource = await createSourceRecord({
      name: "source-map.md",
      origin: "retained/source-map.md",
      version: "receipt-207",
      bytes: sourceMap,
      mediaType: "text/markdown",
      use: "Requirement source and initial mappings",
    });
    const { identityBinding: _identityBinding, ...unboundFields } = currentSource;
    const legacySource = { ...unboundFields, id: `src-${currentSource.sha256.slice(0, 16)}` } as typeof currentSource;
    const packet = await createDesignPacket({
      id: "packet-reimport",
      scenario: "Notification Hub",
      brief: "Legacy archive stays read-only during source identity recovery.",
      sources: [currentSource],
      pages: [{ id: "history", title: "History", stateIds: ["history-list"], sourceRefs: [currentSource.id], regions: ["main"], editableSourceIds: [] }],
      requirements: [visualRequirement({ sourceRefs: [currentSource.id], mapping: { pageId: "history", stateId: "history-list", region: "main", rationale: "Retained source." } })],
      features: [{ id: sourceSceneObligationId("history"), name: "History", sourceRef: currentSource.id, requirementIds: [], disposition: "preserve", rationale: "Retained source obligation." }],
      createdAt: "2026-11-21T00:00:00Z",
    });
    const revised = await addPacketRevision(packet, {
      revisionId: "v2-bottom-nav",
      createdAt: "2026-11-21T00:05:00Z",
      changeSummary: "Preserve the recorded mobile navigation correction.",
      decisionRefs: [{ authority: "operator", id: "NH-DESIGN-20261025-01", source: "retained-feedback.md", sourceTimestamp: "2026-10-25T14:19:12Z", quote: "I do in fact want the bottom nav.", appliesToVersion: "v1" }],
    });
    const archiveBefore = JSON.stringify(revised);
    const joined = await verifySourceReimportIdentity([legacySource], [currentSource]);
    expect(joined).toMatchObject({ valid: true, matches: [{ archivedSourceId: legacySource.id, reimportedSourceId: currentSource.id }] });
    const supportingSource = await createSourceRecord({ name: "E42.md", origin: "retained/E42.md", version: "accepted surface", bytes: "accepted Delivery scene", mediaType: "text/markdown", use: "Additional versioned source" });
    const extended = await verifySourceReimportIdentity([legacySource], [currentSource, supportingSource], { allowAdditionalImportedSources: true });
    expect(extended).toMatchObject({ valid: true, matches: [{ archivedSourceId: legacySource.id, reimportedSourceId: currentSource.id }] });
    expect((await verifySourceReimportIdentity([legacySource], [currentSource, supportingSource])).valid).toBe(false);
    expect(joined.issues).toEqual([]);
    expect(JSON.stringify(revised)).toBe(archiveBefore);
    expect(revised.revisions.map((revision) => [revision.id, revision.parentId])).toEqual([["v1", null], ["v2-bottom-nav", "v1"]]);
    expect(revised.revisions[1]!.decisionRefs[0]!.appliesToVersion).toBe("v1");

    const staleSource = await createSourceRecord({ name: currentSource.name, origin: currentSource.origin, version: "stale-receipt", bytes: sourceMap, mediaType: currentSource.mediaType, use: currentSource.use });
    expect((await verifySourceReimportIdentity([legacySource], [staleSource])).issues.map((issue) => issue.message)).toContain("No re-imported source matches the archived name, origin, version, bytes and media type.");
    expect((await verifySourceReimportIdentity([legacySource], [])).issues.map((issue) => issue.message)).toContain("Re-import must account for every archived source exactly once.");
    expect((await verifySourceReimportIdentity([legacySource, legacySource], [currentSource])).issues.map((issue) => issue.message)).toContain("Duplicate archived source ID makes source lineage ambiguous.");

    const ambiguousImport = { ...currentSource, id: `src-${currentSource.sha256.slice(0, 16)}-000000000000` };
    const ambiguous = await verifySourceReimportIdentity([legacySource], [currentSource, ambiguousImport]);
    expect(ambiguous.valid).toBe(false);
    expect(ambiguous.issues.map((issue) => issue.message)).toContain("Duplicate re-imported source descriptor makes source lineage ambiguous.");
    expect(ambiguous.issues.map((issue) => issue.message)).toContain("Archived source matches multiple re-imported identities.");

    const unboundImport = { ...unboundFields, id: currentSource.id } as typeof currentSource;
    expect((await verifySourceReimportIdentity([legacySource], [unboundImport])).issues.map((issue) => issue.message)).toContain("Re-imported source must carry its exact identity binding.");
    expect((await verifySourceReimportIdentity([{ ...currentSource, id: `src-${currentSource.sha256.slice(0, 16)}-000000000000` }], [currentSource])).issues.map((issue) => issue.message)).toContain("Archived source ID does not bind its current identity descriptor.");
    expect((await verifySourceReimportIdentity([{ ...legacySource, identityBinding: "stale" }], [currentSource])).issues.map((issue) => issue.message)).toContain("Source identity binding is stale or does not match its descriptor.");
    expect((await verifySourceReimportIdentity([], [])).issues.map((issue) => issue.message)).toContain("Source re-import requires non-empty archived and current identity sets.");
  });

  it("parses requirement, selected-state, source and proof links from the Hub source-map table", () => {
    const rows = parseRequirementSourceMap(sourceMap);
    expect(rows).toHaveLength(2);
    expect(sourceRequirementAliases(rows[0]!)).toEqual(["NOTIFICA-P0-001", "OT-P0-001"]);
    expect(rows[0]!.sourceRefs).toContain("NH-AC-27");
    expect(sourceRequirementAliases(rows[0]!)).not.toContain("NH-AC-27");
    expect(sourceScenesForRequirement(rows[0]!, [
      { sceneId: "direct", stateIds: ["state-direct"], filenames: [], sourceHeadings: [], obligations: "", requirementIds: ["NOTIFICA-P0-001"] },
      { sceneId: "citation-only", stateIds: ["state-citation"], filenames: [], sourceHeadings: [], obligations: "", requirementIds: ["NH-AC-27"] },
    ]).map((scene) => scene.sceneId)).toEqual(["direct"]);
    expect(rows[0]).toMatchObject({
      id: "NOTIFICA-P0-001 / OT-P0-001",
      coverage: "visual",
      selectedStates: ["reach-enroll-*", "NH16.4", "NH16.5", "NH16.6"],
      evidenceRefs: ["api/internal/hub/service_test.go::TestSend"],
    });
    expect(rows[1]).toMatchObject({
      id: "NOTIFICA-P0-009 / OT-P0-009",
      coverage: "nonvisual",
      evidenceRefs: [".vrooli/service.json"],
    });
  });

  it("parses Appendix A obligations and rejects incomplete selected-image identity", () => {
    const scenes = parseSceneSourceMap(appendix);
    expect(scenes).toEqual([{ sceneId: "NH13.1", stateIds: ["NH13.1"], filenames: ["NH13.1-dark-phone-p1-r2.png", "NH13.1-light-phone-p1-r2.png"], sourceHeadings: ["Experience / Information architecture"], obligations: "NOTIFICA-P0-007; NH-AC-34/36 (derived correspondence).", requirementIds: ["NOTIFICA-P0-007", "NH-AC-34", "NH-AC-36"] }]);
    const assets = parseSelectedImageManifest(JSON.stringify([{ filename: "frame.png", sceneId: "scene-a", stateIds: ["reach-enroll-ready"], sha256: "a".repeat(64), bytes: 4, sourceRefs: ["Dossier / Reach"], caption: "frame" }]));
    expect(matchingAssetStates("reach-enroll-*", assets)).toHaveLength(1);
    expect(() => parseSelectedImageManifest(JSON.stringify([{ filename: "bad.png", sceneId: "x", stateIds: ["ready"], sha256: "missing", bytes: 4, sourceRefs: [] }]))).toThrow(/invalid SHA/);
    expect(sourceSceneObligationId("NH16-SET-05-denied")).toBe("source-scene-NH16-SET-05-denied");
    expect(sourceSceneIdFromObligationId("source-scene-NH16-SET-05-denied")).toBe("NH16-SET-05-denied");
    expect(sourceSceneIdFromObligationId("source-scene-")).toBeNull();
    expect(resolveSelectedState("reach-enroll-ready", [...assets, { ...assets[0]!, filename: "frame-light.png" }])).toMatchObject({ status: "resolved", sceneId: "scene-a", stateId: "reach-enroll-ready" });
    expect(resolveSelectedState("reach-enroll-*", [
      { ...assets[0]!, sceneId: "scene-a", stateIds: ["reach-enroll-ready"] },
      { ...assets[0]!, filename: "frame2.png", sceneId: "scene-a", stateIds: ["reach-enroll-pending"] },
    ]).status).toBe("ambiguous");
    expect(resolveSelectedState("not-present", assets).status).toBe("missing");
    expect(() => parseSelectedImageManifest(JSON.stringify([{ filename: "no-state.png", sceneId: "scene-a", stateIds: [], sha256: "a".repeat(64), bytes: 4, sourceRefs: [] }]))).toThrow(/invalid stateIds/);
  });

  it("preserves mixed-case source-map state IDs when joining the selected manifest", () => {
    const markdown = `## Appendix A\n\n| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |\n|---|---|---|---|---|\n| integration-access-denied | NH16-SET-05-denied | access-denied.png | Settings / Integration | NOTIFICA-P1-003 |\n`;
    const scenes = parseSceneSourceMap(markdown);
    const asset = { filename: "access-denied.png", sceneId: "integration-access-denied", stateIds: ["NH16-SET-05-denied"], sha256: "a".repeat(64), bytes: 1, sourceRefs: [], caption: "", mediaType: "image/png" };
    expect(scenes[0]?.stateIds).toEqual(["NH16-SET-05-denied"]);
    expect(validateSceneManifestJoins(scenes, [asset])).toEqual([]);
  });

  it("fails closed on duplicate requirement, scene, and selected-image identities", () => {
    const repeatedRequirement = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| ADMIN-P0-001 | First path | auth-ready | Access / Sign in | ui/auth.test.ts |
| ADMIN-P0-001 | Repeated path | auth-ready | Access / Sign in | ui/auth.test.ts |
`;
    const repeatedScene = `## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready | sign-in.png | Access / Sign in | ADMIN-P0-001 |
| access-sign-in | auth-recovery | recovery.png | Access / Recovery | ADMIN-P0-002 |
`;
    expect(() => parseRequirementSourceMap(repeatedRequirement)).toThrow(/duplicate or ambiguous ID/);
    expect(() => parseSceneSourceMap(repeatedScene)).toThrow(/duplicate scene ID/);
    const repeatedAsset = JSON.stringify([
      { filename: "same.png", sceneId: "access-sign-in", stateIds: ["auth-ready"], sha256: "a".repeat(64), bytes: 4, sourceRefs: [] },
      { filename: "same.png", sceneId: "recovery", stateIds: ["auth-recovery"], sha256: "b".repeat(64), bytes: 4, sourceRefs: [] },
    ]);
    expect(() => parseSelectedImageManifest(repeatedAsset)).toThrow(/duplicate filename/);
  });

  it("accepts the same trace contract for a different scenario without Hub identifiers", () => {
    const genericMap = `## Requirement crosswalk

| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |
|---|---|---|---|---|
| ADMIN-P0-001 | Sign-in path | auth-ready | Access / Sign in | ui/auth.test.ts |
| PLATFORM-SECURE-02 | Session handling | No UI screen; server session policy | Access / Session policy | api/session_test.go |

## Appendix A

| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |
|---|---|---|---|---|
| access-sign-in | auth-ready | sign-in.png | Access / Sign in | ADMIN-P0-001 |
`;
    expect(parseRequirementSourceMap(genericMap)).toMatchObject([
      { id: "ADMIN-P0-001", coverage: "visual", selectedStates: ["auth-ready"] },
      { id: "PLATFORM-SECURE-02", coverage: "nonvisual" },
    ]);
    expect(parseSceneSourceMap(genericMap)[0]).toMatchObject({ sceneId: "access-sign-in", stateIds: ["auth-ready"], filenames: ["sign-in.png"], requirementIds: ["ADMIN-P0-001"] });
  });

  it("requires a page/state/region or an explicit nonvisual/omission disposition", async () => {
    const packet = await createDesignPacket({
      id: "packet-hub",
      scenario: "Notification Hub",
      brief: "Private Stage 1 authoring input",
      sources: [source],
      pages: [{ id: "reach", title: "Reach", stateIds: ["reach-enroll-ready"], sourceRefs: [source.id], regions: ["enrollment form"], editableSourceIds: [] }],
      requirements: [
        visualRequirement({ mapping: { pageId: null, stateId: null, region: null, rationale: "" } }),
        visualRequirement({ id: "NOTIFICA-P0-009", coverage: "nonvisual", mapping: { pageId: null, stateId: null, region: null, rationale: "No UI scene is required; service configuration is the proof surface." }, disposition: "preserve" }),
      ],
      features: [{ id: "legacy-adapter", name: "Existing adapter", sourceRef: source.id, requirementIds: [], disposition: "unknown", rationale: "" }],
      createdAt: "2026-11-21T00:00:00Z",
    });
    const result = validatePacket(packet);
    expect(result.valid).toBe(false);
    expect(result.issues.map((issue) => issue.path)).toContain("requirements.NOTIFICA-P0-001 / OT-P0-001.mapping");
    expect(result.issues.map((issue) => issue.path)).toContain("features.legacy-adapter.disposition");
  });

  it("creates a new source-bound revision without mutating the original or inventing acceptance", async () => {
    const first = await createDesignPacket({
      id: "packet-hub",
      scenario: "Notification Hub",
      brief: "Retained design packet",
      sources: [source],
      pages: [{ id: "history", title: "History", stateIds: ["history-list"], sourceRefs: [source.id], regions: ["main content"], editableSourceIds: [] }],
      requirements: [visualRequirement({ sourceRefs: [source.id], mapping: { pageId: "history", stateId: "history-list", region: "main content", rationale: "Mapped to retained history source." } })],
      features: [{ id: sourceSceneObligationId("history"), name: "History source obligation", sourceRef: source.id, requirementIds: [], disposition: "preserve", rationale: "Retain the history surface." }],
      createdAt: "2026-11-21T00:00:00Z",
    });
    const second = await addPacketRevision(first, {
      revisionId: "v2-bottom-nav",
      createdAt: "2026-11-21T00:05:00Z",
      changeSummary: "Applied the recorded primary mobile navigation correction to the design source.",
      pages: [{ ...first.revisions[0]!.pages[0]!, regions: ["main content", "persistent bottom navigation"] }],
      decisionRefs: [{
        authority: "operator",
        id: "NH-DESIGN-20261025-01",
        source: "cases/notification-hub/legacy-delivery/FEEDBACK.md",
        sourceTimestamp: "2026-10-25T14:19:12Z",
        quote: "I do in fact want the bottom nav.",
        appliesToVersion: first.revisions[0]!.id,
      }],
    });
    expect(first.revisions).toHaveLength(1);
    expect(first.currentRevisionId).toBe("v1");
    expect(second.revisions).toHaveLength(2);
    expect(second.revisions[1]?.parentId).toBe("v1");
    expect(second.revisions[1]?.decisionRefs[0]?.appliesToVersion).toBe("v1");
    expect(second.revisions[0]?.pages[0]?.regions).toEqual(["main content"]);
    expect(validatePacket(second).issues).toEqual([]);
    expect((await verifyPacketIntegrity(second)).issues).toEqual([]);
    const staleDecision = structuredClone(second);
    staleDecision.revisions[1]!.decisionRefs[0]!.appliesToVersion = "missing-version";
    expect(validatePacket(staleDecision).issues.map((issue) => issue.message)).toContain("Operator decision must bind to an ancestor packet version.");
    const staleState = structuredClone(second);
    staleState.revisions[1]!.requirements[0]!.mapping.stateId = "state-from-another-scene";
    expect(validatePacket(staleState).issues.map((issue) => issue.message)).toContain("Requirement state is not present on the selected page.");
    await expect(addPacketRevision(first, {
      revisionId: "v2-dropped-obligation",
      createdAt: "2026-11-21T00:06:00Z",
      changeSummary: "attempted omission",
      features: [],
    })).rejects.toThrow(/cannot discard a source-scene obligation ID/);
  });

  it("generates a navigable escaped atlas with source, mapping and omission sections", async () => {
    const packet = await createDesignPacket({
      id: "packet-hub",
      scenario: "Notification Hub <Hub>",
      brief: "Source-backed packet & lineage",
      sources: [source],
      pages: [{ id: "reach", title: "Reach", stateIds: ["reach-ready"], sourceRefs: [source.id], regions: ["form"], editableSourceIds: [] }],
      requirements: [visualRequirement({ sourceRefs: ["NH-AC-27"] })],
      features: [{ id: "nonvisual", name: "Nonvisual behavior", sourceRef: source.id, requirementIds: [], disposition: "preserve", rationale: "Maintain the backend contract." }],
      createdAt: "2026-11-21T00:00:00Z",
    });
    const html = buildAtlasHtml(packet);
    expect(html).toContain("Notification Hub &lt;Hub&gt; — design atlas");
    expect(html).toContain("href=\"#page-reach\"");
    expect(html).toContain("Source lineage");
    expect(html).toContain("Omission and preservation register");
    expect(html).toContain("Obligation ID");
    expect(html).toContain("<code>nonvisual</code>");
    expect(html).toContain("NH-AC");
    expect(html).not.toContain("<Hub>");
    expect(html).toContain("@media print");
  });

  it("round-trips every requirement and source-scene ID through selected-state mapping, revision, atlas, and integrity checks", async () => {
    const requirementRows = Array.from({ length: 28 }, (_, index) => {
      const ordinal = index + 1;
      const priority = ordinal <= 26 ? "P1" : "P2";
      const id = ordinal <= 26
        ? `NOTIFICA-${priority}-${String(ordinal).padStart(3, "0")} / OT-${priority}-${String(ordinal).padStart(3, "0")}`
        : `NOTIFICA-P2-${String(ordinal - 26 + 1).padStart(3, "0")} / OT-P2-${String(ordinal - 26 + 1).padStart(3, "0")}`;
      const selected = ordinal <= 26 ? `scene-${String(ordinal).padStart(3, "0")}-ready` : "No UI screen/state; future P2 requirement";
      const scope = ordinal <= 26 ? "In current selected scope." : "P2 future requirement; not selected in current scope.";
      return { id, selected, scope };
    });
    const sceneRows = Array.from({ length: 102 }, (_, index) => {
      const ordinal = index + 1;
      const sceneId = `scene-${String(ordinal).padStart(3, "0")}`;
      const stateId = `${sceneId}-ready`;
      const filename = `${sceneId}.png`;
      const linked = ordinal <= 26 ? `NOTIFICA-P1-${String(ordinal).padStart(3, "0")}; OT-P1-${String(ordinal).padStart(3, "0")}` : `Preserved source obligation ${ordinal}`;
      return { sceneId, stateId, filename, linked };
    });
    const mapMarkdown = `## Requirement crosswalk\n\n| Requirement | Scope disposition | Selected scene/state families | Source headings | Existing validation |\n|---|---|---|---|---|\n${requirementRows.map((row) => `| ${row.id} | ${row.scope} | ${row.selected} | Experience / Requirement ${row.id} | test/contract.test.ts |`).join("\n")}\n\n## Appendix A\n\n| Scene family | State IDs | Manifest filenames | Exact source headings | Related registry requirements / explicit design obligations |\n|---|---|---|---|---|\n${sceneRows.map((row) => `| ${row.sceneId} | ${row.stateId} | ${row.filename} | Experience / ${row.sceneId} | ${row.linked} |`).join("\n")}\n`;
    const imagePayloads = sceneRows.map((_, index) => `synthetic-image-${index + 1}`);
    const imageHashes = await Promise.all(imagePayloads.map((bytes) => sha256Hex(bytes)));
    const manifest = sceneRows.map((row, index) => ({
      filename: row.filename,
      sceneId: row.sceneId,
      stateIds: [row.stateId],
      sha256: imageHashes[index],
      bytes: new TextEncoder().encode(imagePayloads[index]!).byteLength,
      sourceRefs: [`Experience / ${row.sceneId}`],
      caption: row.sceneId,
    }));
    const parsedRequirements = parseRequirementSourceMap(mapMarkdown);
    const parsedScenes = parseSceneSourceMap(mapMarkdown);
    const selectedAssets = parseSelectedImageManifest(JSON.stringify(manifest));
    expect(parsedRequirements).toHaveLength(28);
    expect(parsedScenes).toHaveLength(102);
    expect(selectedAssets).toHaveLength(102);
    expect(validateSceneManifestJoins(parsedScenes, selectedAssets)).toEqual([]);
    expect(validateSceneManifestJoins(parsedScenes, selectedAssets.map((asset, index) => index === 0 ? { ...asset, sceneId: "different-scene" } : asset)).map((issue) => issue.message)).toContain("Selected-image scene ID does not match its source-map row.");

    const mapRecord = await createSourceRecord({ name: "source-map.md", origin: "synthetic/source-map.md", version: "fixture-v1", bytes: mapMarkdown, mediaType: "text/markdown", use: "Synthetic requirement and obligation source" });
    const manifestText = JSON.stringify(manifest);
    const manifestRecord = await createSourceRecord({ name: "selected-image-manifest.json", origin: "synthetic/selected-image-manifest.json", version: "fixture-v1", bytes: manifestText, mediaType: "application/json", use: "Synthetic selected-image identity source" });
    const imageSources = await Promise.all(sceneRows.map((row, index) => createSourceRecord({ name: row.filename, origin: `synthetic/images/${row.filename}`, version: `fixture-v1:${row.stateId}`, bytes: imagePayloads[index]!, mediaType: "image/png", use: "Synthetic source scene frame" })));
    const sources = [mapRecord, manifestRecord, ...imageSources];
    const pages = sceneRows.map((row, index) => ({ id: row.sceneId, title: row.sceneId, stateIds: [row.stateId], sourceRefs: [mapRecord.id, manifestRecord.id, imageSources[index]!.id], regions: [row.filename], editableSourceIds: [] }));
    const requirements: PacketRequirement[] = parsedRequirements.map((row, index) => {
      const visual = index < 26;
      const selected = visual ? selectedAssets[index]! : undefined;
      return {
        id: row.id,
        text: row.text,
        scope: row.scope,
        coverage: row.coverage,
        mapping: selected ? {
          pageId: selected.sceneId,
          stateId: resolveSelectedState(row.selectedStates[0]!, selectedAssets, selected.sceneId).status === "resolved" ? row.selectedStates[0]! : null,
          region: `Selected design frame (${selected.filename})`,
          rationale: `Source-map state family ${row.selectedStates[0]}; selected-image manifest binds ${selected.filename} (${selected.sha256}). This is a synthetic trace fixture.`,
        } : { pageId: null, stateId: null, region: null, rationale: "No UI screen/state; explicit future requirement disposition." },
        sourceRefs: [mapRecord.id, manifestRecord.id],
        evidenceRefs: row.evidenceRefs,
        disposition: visual ? "preserve" : "defer",
      };
    });
    const features = parsedScenes.map((row) => ({
      id: sourceSceneObligationId(row.sceneId),
      name: row.obligations,
      sourceRef: mapRecord.id,
      requirementIds: row.requirementIds,
      disposition: "preserve" as const,
      rationale: `Stable source scene ${row.sceneId}; manifest and source bytes are joined by exact identity.`,
    }));
    const packet = await createDesignPacket({ id: "synthetic-complete-packet", scenario: "Synthetic Transfer", brief: "Synthetic data only.", sources, pages, requirements, features, createdAt: "2026-11-21T00:00:00Z" });
    expect(validatePacket(packet).issues).toEqual([]);
    expect((await verifyPacketIntegrity(packet)).issues).toEqual([]);

    const v2 = await addPacketRevision(packet, {
      revisionId: "v2-reviewed-disposition",
      createdAt: "2026-11-21T00:05:00Z",
      changeSummary: "Preserve all source row identities while recording one versioned disposition change.",
      features: features.map((feature, index) => index === 0 ? { ...feature, disposition: "defer", rationale: "Explicitly deferred in the synthetic revision." } : feature),
      decisionRefs: [{ authority: "operator", id: "SYNTHETIC-DECISION-01", source: "synthetic/decision.md", sourceTimestamp: "2026-11-21T00:04:00Z", quote: "Keep the requirement visible.", appliesToVersion: "v1" }],
    });
    const reloaded = JSON.parse(JSON.stringify(v2)) as DesignPacket;
    const result = await verifyPacketIntegrity(reloaded);
    expect(result.issues).toEqual([]);
    const current = reloaded.revisions.find((revision) => revision.id === reloaded.currentRevisionId)!;
    expect(current.requirements).toHaveLength(28);
    expect(current.features).toHaveLength(102);
    expect(current.requirements.filter((row) => row.disposition === "defer").map((row) => row.id)).toEqual([
      "NOTIFICA-P2-002 / OT-P2-002",
      "NOTIFICA-P2-003 / OT-P2-003",
    ]);
    expect(current.features.map((feature) => feature.id)).toEqual(sceneRows.map((row) => sourceSceneObligationId(row.sceneId)));
    expect(current.features[0]?.disposition).toBe("defer");
    const atlas = buildAtlasHtml(reloaded);
    expect(atlas).toContain("source-scene-scene-001");
    expect(atlas).toContain("source-scene-scene-102");

    const staleSource = structuredClone(reloaded);
    staleSource.sources[0]!.version = "mutated-without-new-source-id";
    expect(validatePacket(staleSource).issues.map((issue) => issue.message)).toContain("Source origin/version changed without a new source identity.");
    expect((await verifyPacketIntegrity(staleSource)).issues.map((issue) => issue.message)).toContain("Source ID does not bind its recorded origin, version and content identity.");
    const staleRevision = structuredClone(reloaded);
    staleRevision.revisions[1]!.features[0]!.name = "silently mutated source obligation";
    expect((await verifyPacketIntegrity(staleRevision)).issues.map((issue) => issue.message)).toContain("Revision content changed without creating a new immutable revision.");
    const duplicateObligation = structuredClone(reloaded);
    duplicateObligation.revisions[1]!.features[1]!.id = duplicateObligation.revisions[1]!.features[0]!.id;
    expect(validatePacket(duplicateObligation).issues.map((issue) => issue.message)).toContain("Obligation IDs must be unique.");
    const missingObligation = structuredClone(reloaded);
    missingObligation.revisions[1]!.features = missingObligation.revisions[1]!.features.filter((feature) => feature.id !== sourceSceneObligationId("scene-001"));
    expect(validatePacket(missingObligation).issues.map((issue) => issue.message)).toContain("Page is missing a stable source-scene obligation disposition.");
    const staleSelectedImage = structuredClone(reloaded);
    staleSelectedImage.revisions[1]!.requirements[0]!.mapping.rationale = staleSelectedImage.revisions[1]!.requirements[0]!.mapping.rationale.replace(/[a-f0-9]{64}/, "0".repeat(64));
    expect(validatePacket(staleSelectedImage).issues.map((issue) => issue.message)).toContain("Selected-frame mapping is missing an exact source filename/hash join in this revision.");
  });
});
