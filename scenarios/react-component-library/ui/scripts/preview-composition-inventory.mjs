#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

const scenarioRoot = path.resolve(import.meta.dirname, "../..");
const libraryRoot = path.join(scenarioRoot, "library");
const inventoryPath = path.join(scenarioRoot, "docs/evidence/preview-composition-inventory.json");
const canonicalLedgerMetadata = {
  dispositionPolicy: {
    direct:
      "Keep the story direct when the subject is the behavior and no contextual composition changes interpretation.",
    "behavioral-direct":
      "Keep the story local and behavior-focused when interactions or state transitions are the meaningful subject.",
    "fixture-backed": "Use only deterministic typed fixture data for external-state behavior.",
    "shared-harness":
      "Promote only harnesses whose family contract is reused by multiple assets without hard-coded subjects.",
    "named-specimen-review":
      "Move child structure into named specimen exports when it materially explains the component behavior.",
    "local-harness-review":
      "Retain the local executable composition until behavior-equivalence evidence supports shared-harness adoption or deferral.",
  },
  adoptionClosure: {
    reconcileScanned: 423,
    canonicalRecordsApplied: 394,
    remainingExceptions: 2,
    exceptions: [
      {
        scenario: "program-runtime",
        path: "ui/src/layout/Sidebar.tsx",
        libraryId: "local:ProgramRuntimeSidebar",
        owner: "program-runtime maintainers",
        reason:
          "Scenario-specific sidebar has no canonical RCL implementation and is intentionally excluded from adoption closure.",
        revisitWhen:
          "A reusable navigation shell contract is extracted and the scenario can adopt it without losing product-specific behavior.",
      },
      {
        scenario: "template-manager",
        path: "ui/src/layout/Sidebar.tsx",
        libraryId: "local:TemplateManagerSidebar",
        owner: "template-manager maintainers",
        reason:
          "Scenario-specific sidebar has no canonical RCL implementation and is intentionally excluded from adoption closure.",
        revisitWhen:
          "A reusable navigation shell contract is extracted and the scenario can adopt it without losing product-specific behavior.",
      },
    ],
  },
  releaseReconciliation: {
    catalogIndex:
      "209 indexed assets, zero indexing errors after EvidenceCarousel immutable-release repair.",
    publishedChanges: [
      "EvidenceCarousel@1.0.8: redesigned evidence workspace",
      "EvidenceCarousel@1.0.9: design-system content-height ramp correction",
      "Card@1.2.0: semantic child-composition and realistic state stories",
      "Dialog@1.2.0: explicit open and close behavior stories",
      "SidebarShell@1.3.0: realistic navigation and failure-state stories",
    ],
    validation: [
      "Focused component preflight passed for EvidenceCarousel@1.0.8 and @1.0.9.",
      "Fresh BAS evidence passed all stages for EvidenceCarousel@1.0.9.",
      "ComponentTestPanel UI regression suite passed 6/6.",
      "Dialog@1.2.0 component test passed declared behavior and experience evidence; persisted screenshot selected BAS primary frame.",
    ],
  },
};

function filesUnder(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const candidate = path.join(dir, entry.name);
    if (entry.isDirectory()) return filesUnder(candidate);
    return entry.isFile() ? [candidate] : [];
  });
}

function hierarchy(root) {
  return (
    {
      foundations: "foundation",
      hooks: "runtime-hook",
      services: "runtime-service",
      primitives: "primitive",
      components: "component",
      patterns: "pattern",
      navigation: "navigation",
      pages: "page-template",
    }[root] ?? "unknown"
  );
}

function parseArgs(argv) {
  const options = {
    batchSize: 50,
    batchIndex: 0,
    stateFile: "docs/evidence/preview-composition-state.json",
    markComplete: [],
  };
  for (let index = 0; index < argv.length; index += 1) {
    const value = argv[index];
    if (value === "--batch-size") options.batchSize = Number(argv[++index]);
    else if (value === "--batch-index") options.batchIndex = Number(argv[++index]);
    else if (value === "--state") options.stateFile = argv[++index];
    else if (value === "--mark-complete")
      options.markComplete.push(...String(argv[++index]).split(",").filter(Boolean));
    else if (value === "--help" || value === "-h") {
      console.log(
        "Usage: node preview-composition-inventory.mjs [--batch-size N --batch-index N] [--state FILE]",
      );
      process.exit(0);
    }
  }
  if (!Number.isInteger(options.batchSize) || options.batchSize < 0)
    throw new Error("--batch-size must be a non-negative integer");
  if (!Number.isInteger(options.batchIndex) || options.batchIndex < 0)
    throw new Error("--batch-index must be a non-negative integer");
  return options;
}

function resolvedStatePath(file) {
  return path.isAbsolute(file) ? file : path.join(scenarioRoot, file);
}

function readState(file) {
  if (!file || !fs.existsSync(resolvedStatePath(file))) {
    return { completedStoryKeys: new Set(), lastCompletedBatchIndex: null };
  }
  const parsed = JSON.parse(fs.readFileSync(resolvedStatePath(file), "utf8"));
  return {
    completedStoryKeys: new Set(
      (Array.isArray(parsed.completedStoryKeys) ? parsed.completedStoryKeys : []).filter(
        (key) => typeof key === "string",
      ),
    ),
    lastCompletedBatchIndex: Number.isInteger(parsed.lastCompletedBatchIndex)
      ? parsed.lastCompletedBatchIndex
      : null,
  };
}

function writeState(file, completedStoryKeys, batchIndex) {
  const target = resolvedStatePath(file);
  fs.mkdirSync(path.dirname(target), { recursive: true });
  fs.writeFileSync(
    target,
    `${JSON.stringify(
      {
        completedStoryKeys: [...completedStoryKeys].sort(),
        lastCompletedBatchIndex: batchIndex,
        updatedAt: new Date().toISOString(),
      },
      null,
      2,
    )}\n`,
  );
}

function readLedgerMetadata() {
  const sources = [inventoryPath];
  for (const source of sources) {
    if (!fs.existsSync(source)) continue;
    try {
      const parsed = JSON.parse(fs.readFileSync(source, "utf8"));
      const metadata = {};
      for (const key of ["dispositionPolicy", "adoptionClosure", "releaseReconciliation"]) {
        if (parsed[key] !== undefined) metadata[key] = parsed[key];
      }
      if (Object.keys(metadata).length) return metadata;
    } catch {
      // A malformed evidence sidecar must not prevent a fresh inventory.
    }
  }
  return canonicalLedgerMetadata;
}

function containsRawNode(value) {
  if (!value || typeof value !== "object") return false;
  if (Array.isArray(value)) return value.some(containsRawNode);
  if (Object.prototype.hasOwnProperty.call(value, "$node")) return true;
  return Object.values(value).some(containsRawNode);
}

function localImportDiagnostics(storyFile) {
  const sourceFile = path.join(path.dirname(storyFile), "story.tsx");
  if (!fs.existsSync(sourceFile)) return [];
  const source = fs.readFileSync(sourceFile, "utf8");
  return [
    ["network-access", /(?:fetch\s*\(|XMLHttpRequest|WebSocket)/],
    ["storage-access", /(?:localStorage|sessionStorage|indexedDB|document\.cookie)/],
    ["production-service-access", /(?:child_process|node:fs|node:http|process\.)/],
  ]
    .filter(([, pattern]) => pattern.test(source))
    .map(([code]) => ({ code, severity: "error" }));
}

function localExportNames(storyFile) {
  const sourceFile = path.join(path.dirname(storyFile), "story.tsx");
  if (!fs.existsSync(sourceFile)) return new Set();
  const source = fs.readFileSync(sourceFile, "utf8");
  const names = new Set(
    [
      ...source.matchAll(/\bexport\s+(?:async\s+)?(?:function|const|class)\s+([A-Za-z_$][\w$]*)/g),
    ].map((match) => match[1]),
  );
  for (const match of source.matchAll(/\bexport\s*\{([^}]+)\}/g)) {
    for (const item of match[1].split(",")) {
      const name = item.trim().split(/\s+as\s+/)[1] ?? item.trim();
      if (/^[A-Za-z_$][\w$]*$/.test(name)) names.add(name);
    }
  }
  return names;
}

function componentIdentity(storyFile, parsed) {
  const versionDir = path.dirname(storyFile);
  const componentDir = path.dirname(path.dirname(versionDir));
  const manifestPath = path.join(componentDir, "component.json");
  let libraryId = null;
  if (fs.existsSync(manifestPath)) {
    try {
      libraryId = JSON.parse(fs.readFileSync(manifestPath, "utf8")).libraryId ?? null;
    } catch {
      /* inventory remains useful with a missing id */
    }
  }
  return { libraryId, version: path.basename(versionDir), title: parsed.title ?? null };
}

function storyRecord(story, parsed, identity, storyFile, hasLocalHarness) {
  const composition = story.composition ?? parsed.composition ?? null;
  const compositionHarness = composition?.harness ?? null;
  const specimen = composition?.specimen ?? null;
  const frame = composition?.frame ?? null;
  const fixture = composition?.fixture ?? frame?.fixture ?? null;
  // The core review set is the safe default. Authors only need to opt into a
  // named set when a story belongs to a narrower release or state review.
  const reviewSet = story.evidence?.reviewSet ?? "core";
  const rawChild = containsRawNode(story.args) || containsRawNode(parsed.args);
  const localHarness = Boolean(specimen || (hasLocalHarness && !compositionHarness));
  const diagnostics = [];
  if (rawChild) diagnostics.push({ code: "raw-child-node", severity: "warning" });
  if (!story.expect?.length && !story.interactions?.length)
    diagnostics.push({ code: "no-meaningful-expectations", severity: "warning" });
  if (specimen && specimen.module !== "./story.tsx")
    diagnostics.push({ code: "invalid-specimen-module", severity: "error" });
  if (localHarness) {
    diagnostics.push(...localImportDiagnostics(storyFile));
    const requestedExport = specimen?.export;
    if (requestedExport && !localExportNames(storyFile).has(requestedExport))
      diagnostics.push({ code: "missing-harness-export", severity: "error" });
  }
  let disposition = "direct";
  if (rawChild) disposition = "named-specimen-review";
  else if (compositionHarness) disposition = "composition-harness";
  else if (localHarness) disposition = "local-harness-review";
  else if (fixture) disposition = "fixture-backed";
  else if (frame) disposition = "frame";
  else if (story.expect?.length || story.interactions?.length) disposition = "behavioral-direct";
  const blocking = diagnostics.some((diagnostic) => diagnostic.severity === "error");
  const proposedDisposition = blocking
    ? "defer-until-contract-repaired"
    : rawChild
      ? "named-specimen-required"
      : !story.expect?.length && !story.interactions?.length
        ? "author-review-required"
        : "ready-for-bounded-review";
  return {
    id: story.id ?? null,
    name: story.name ?? null,
    storyKey: `${identity.libraryId ?? "unknown"}@${identity.version}#${story.id ?? "unknown"}`,
    disposition,
    proposedDisposition,
    compositionStatus:
      compositionHarness || frame || fixture
        ? "adopted"
        : localHarness
          ? "requires-review"
          : "eligible-for-review",
    harness: null,
    specimen,
    frame,
    compositionHarness,
    fixture,
    fixtureCount: Array.isArray(parsed.environment?.fixtures)
      ? parsed.environment.fixtures.length
      : 0,
    expectationCount: Array.isArray(story.expect) ? story.expect.length : 0,
    interactionCount: Array.isArray(story.interactions) ? story.interactions.length : 0,
    reviewSet,
    diagnostics,
    exception:
      localHarness && !compositionHarness
        ? {
            kind: "intentional-local-harness-review",
            reason:
              "Local executable composition requires behavior-equivalence review before shared-harness adoption.",
            owner: "react-component-library Preview maintainers",
            revisitWhen:
              "A matching composition harness family has equivalent interaction, accessibility, and visual evidence.",
          }
        : null,
  };
}

function collectEntries() {
  return filesUnder(libraryRoot)
    .filter((file) => path.basename(file) === "story.json")
    .map((file) => {
      const relative = path.relative(scenarioRoot, file).split(path.sep).join("/");
      const parsed = JSON.parse(fs.readFileSync(file, "utf8"));
      const identity = componentIdentity(file, parsed);
      const hasLocalHarness = fs.existsSync(path.join(path.dirname(file), "story.tsx"));
      const stories = Array.isArray(parsed.stories) ? parsed.stories : [];
      const storyRecords = stories.map((story) =>
        storyRecord(story, parsed, identity, file, hasLocalHarness),
      );
      const harnessCounts = new Map();
      for (const story of storyRecords) {
        if (story.specimen?.export)
          harnessCounts.set(
            story.specimen.export,
            (harnessCounts.get(story.specimen.export) ?? 0) + 1,
          );
      }
      for (const story of storyRecords) {
        if (story.specimen?.export && harnessCounts.get(story.specimen.export) > 1) {
          story.diagnostics.push({ code: "repeated-harness-reference", severity: "warning" });
        }
      }
      const dispositions = [...new Set(storyRecords.map((story) => story.disposition))];
      return {
        path: relative,
        libraryId: identity.libraryId,
        version: identity.version,
        title: identity.title,
        hierarchy: hierarchy(relative.split("/")[1] ?? "unknown"),
        schemaVersion: parsed.schemaVersion ?? null,
        kind: parsed.kind ?? null,
        stories: storyRecords.map((story) => story.id).filter(Boolean),
        disposition: dispositions.length > 1 ? "mixed-composition" : (dispositions[0] ?? "direct"),
        storyRecords,
        diagnostics: [
          ...new Set(storyRecords.flatMap((story) => story.diagnostics.map((item) => item.code))),
        ],
        reviewStatus: storyRecords.some(
          (story) => story.proposedDisposition !== "ready-for-bounded-review",
        )
          ? "requires-review"
          : "classified",
        frame: null,
        localHarness: hasLocalHarness,
      };
    })
    .sort((a, b) => a.path.localeCompare(b.path));
}

const options = parseArgs(process.argv.slice(2));
const state = readState(options.stateFile);
const completedStoryKeys = state.completedStoryKeys;
const entries = collectEntries();
const storyRecords = entries.flatMap((entry) => entry.storyRecords);
const activeStoryKeys = new Set(storyRecords.map((story) => story.storyKey));
let stateReconciled = false;
for (const storyKey of completedStoryKeys) {
  if (!activeStoryKeys.has(storyKey)) {
    completedStoryKeys.delete(storyKey);
    stateReconciled = true;
  }
}
for (const storyKey of options.markComplete) {
  if (storyRecords.some((story) => story.storyKey === storyKey)) completedStoryKeys.add(storyKey);
}
if (options.markComplete.length || stateReconciled) {
  writeState(
    options.stateFile,
    completedStoryKeys,
    options.markComplete.length ? options.batchIndex : state.lastCompletedBatchIndex,
  );
}
const pending = storyRecords.filter((story) => !completedStoryKeys.has(story.storyKey));
const batch =
  options.batchSize > 0
    ? pending.slice(
        options.batchIndex * options.batchSize,
        (options.batchIndex + 1) * options.batchSize,
      )
    : pending;
const batchKeys = new Set(batch.map((story) => story.storyKey));

process.stdout.write(
  `${JSON.stringify(
    {
      schemaVersion: 2,
      generatedFrom: "library/**/story.json",
      ...readLedgerMetadata(),
      summary: {
        contractCount: entries.length,
        storyCount: storyRecords.length,
        completedCount: storyRecords.length - pending.length,
        pendingCount: pending.length,
        diagnosticCount: storyRecords.reduce((count, story) => count + story.diagnostics.length, 0),
        unclassifiedCount: storyRecords.filter((story) => !story.proposedDisposition).length,
        exceptionCount: storyRecords.filter((story) => story.exception).length,
        readyForBoundedReviewCount: storyRecords.filter(
          (story) => story.proposedDisposition === "ready-for-bounded-review",
        ).length,
        batchSize: options.batchSize,
        batchIndex: options.batchSize > 0 ? options.batchIndex : 0,
        batchCount:
          options.batchSize > 0
            ? Math.ceil(storyRecords.length / options.batchSize)
            : storyRecords.length
              ? 1
              : 0,
      },
      resume: {
        stateFile: options.stateFile,
        completedStoryKeys: [...completedStoryKeys].sort(),
        nextStoryKey: pending[0]?.storyKey ?? null,
        nextBatchStoryKeys: batch.map((story) => story.storyKey),
      },
      entries,
      batch,
    },
    null,
    2,
  )}\n`,
);
