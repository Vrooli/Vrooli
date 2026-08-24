#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

const scenarioRoot = path.resolve(import.meta.dirname, "../..");
const libraryRoot = path.join(scenarioRoot, "library");

function filesUnder(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
    const candidate = path.join(dir, entry.name);
    if (entry.isDirectory()) return filesUnder(candidate);
    return entry.isFile() ? [candidate] : [];
  });
}

function hierarchy(root) {
  return ({
    foundations: "foundation",
    hooks: "runtime-hook",
    services: "runtime-service",
    primitives: "primitive",
    components: "component",
    patterns: "pattern",
    navigation: "navigation",
    pages: "page-template",
  }[root] ?? "unknown");
}

function parseArgs(argv) {
  const options = { batchSize: 0, batchIndex: 0, stateFile: null };
  for (let index = 0; index < argv.length; index += 1) {
    const value = argv[index];
    if (value === "--batch-size") options.batchSize = Number(argv[++index]);
    else if (value === "--batch-index") options.batchIndex = Number(argv[++index]);
    else if (value === "--state") options.stateFile = argv[++index];
    else if (value === "--help" || value === "-h") {
      console.log("Usage: node preview-composition-inventory.mjs [--batch-size N --batch-index N] [--state FILE]");
      process.exit(0);
    }
  }
  if (!Number.isInteger(options.batchSize) || options.batchSize < 0) throw new Error("--batch-size must be a non-negative integer");
  if (!Number.isInteger(options.batchIndex) || options.batchIndex < 0) throw new Error("--batch-index must be a non-negative integer");
  return options;
}

function readState(file) {
  if (!file) return new Set();
  const parsed = JSON.parse(fs.readFileSync(path.resolve(file), "utf8"));
  return new Set((Array.isArray(parsed.completedStoryKeys) ? parsed.completedStoryKeys : []).filter((key) => typeof key === "string"));
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
  ].filter(([, pattern]) => pattern.test(source)).map(([code]) => ({ code, severity: "error" }));
}

function localExportNames(storyFile) {
  const sourceFile = path.join(path.dirname(storyFile), "story.tsx");
  if (!fs.existsSync(sourceFile)) return new Set();
  const source = fs.readFileSync(sourceFile, "utf8");
  return new Set([...source.matchAll(/\bexport\s+(?:async\s+)?(?:function|const|class)\s+([A-Za-z_$][\w$]*)/g)].map((match) => match[1]));
}

function componentIdentity(storyFile, parsed) {
  const versionDir = path.dirname(storyFile);
  const componentDir = path.dirname(path.dirname(versionDir));
  const manifestPath = path.join(componentDir, "component.json");
  let libraryId = null;
  if (fs.existsSync(manifestPath)) {
    try { libraryId = JSON.parse(fs.readFileSync(manifestPath, "utf8")).libraryId ?? null; } catch { /* inventory remains useful with a missing id */ }
  }
  return { libraryId, version: path.basename(versionDir), title: parsed.title ?? null };
}

function storyRecord(story, parsed, identity, storyFile, hasLocalHarness) {
  const composition = story.composition ?? parsed.composition ?? null;
  const sharedHarness = story.sharedHarness ?? composition?.harness ?? null;
  const specimen = composition?.specimen ?? null;
  const frame = story.frame ?? composition?.frame ?? parsed.frame ?? null;
  const fixture = composition?.fixture ?? frame?.fixture ?? null;
  // The core review set is the safe default. Authors only need to opt into a
  // named set when a story belongs to a narrower release or state review.
  const reviewSet = story.evidence?.reviewSet ?? "core";
  const rawChild = containsRawNode(story.args) || containsRawNode(parsed.args);
  const localHarness = Boolean(story.harness || specimen || (hasLocalHarness && !sharedHarness));
  const diagnostics = [];
  if (rawChild) diagnostics.push({ code: "raw-child-node", severity: "warning" });
  if (!story.expect?.length && !story.interactions?.length) diagnostics.push({ code: "no-meaningful-expectations", severity: "warning" });
  if (specimen && specimen.module !== "./story.tsx") diagnostics.push({ code: "invalid-specimen-module", severity: "error" });
  if (localHarness) {
    diagnostics.push(...localImportDiagnostics(storyFile));
    const requestedExport = story.harness ?? specimen?.export;
    if (requestedExport && !localExportNames(storyFile).has(requestedExport)) diagnostics.push({ code: "missing-harness-export", severity: "error" });
  }
  let disposition = "direct";
  if (rawChild) disposition = "migrate-named-specimen";
  else if (sharedHarness) disposition = "shared-harness";
  else if (localHarness) disposition = "local-harness-review";
  else if (fixture) disposition = "fixture-backed";
  else if (frame) disposition = "frame";
  else if (story.expect?.length || story.interactions?.length) disposition = "behavioral-direct";
  const blocking = diagnostics.some((diagnostic) => diagnostic.severity === "error");
  const proposedDisposition = blocking
    ? "defer-until-contract-repaired"
    : rawChild
      ? "migrate-to-named-specimen"
    : (!story.expect?.length && !story.interactions?.length)
        ? "author-review-required"
        : "ready-for-bounded-review";
  return {
    id: story.id ?? null,
    name: story.name ?? null,
    storyKey: `${identity.libraryId ?? "unknown"}@${identity.version}#${story.id ?? "unknown"}`,
    disposition,
    proposedDisposition,
    migration: sharedHarness || frame || fixture ? "adopted" : localHarness ? "requires-review" : "eligible-for-review",
    harness: story.harness ?? null,
    specimen,
    frame,
    sharedHarness,
    fixture,
    fixtureCount: Array.isArray(parsed.environment?.fixtures) ? parsed.environment.fixtures.length : 0,
    expectationCount: Array.isArray(story.expect) ? story.expect.length : 0,
    interactionCount: Array.isArray(story.interactions) ? story.interactions.length : 0,
    reviewSet,
    diagnostics,
    exception: localHarness && !sharedHarness ? {
      kind: "intentional-local-harness-review",
      reason: "Local executable composition requires behavior-equivalence review before shared-harness migration.",
      owner: "react-component-library Preview maintainers",
      revisitWhen: "A matching shared harness family has equivalent interaction, accessibility, and visual evidence.",
    } : null,
  };
}

function collectEntries() {
  return filesUnder(libraryRoot).filter((file) => path.basename(file) === "story.json").map((file) => {
    const relative = path.relative(scenarioRoot, file).split(path.sep).join("/");
    const parsed = JSON.parse(fs.readFileSync(file, "utf8"));
    const identity = componentIdentity(file, parsed);
    const hasLocalHarness = fs.existsSync(path.join(path.dirname(file), "story.tsx"));
    const stories = Array.isArray(parsed.stories) ? parsed.stories : [];
    const storyRecords = stories.map((story) => storyRecord(story, parsed, identity, file, hasLocalHarness));
    const harnessCounts = new Map();
    for (const story of storyRecords) {
      if (story.harness) harnessCounts.set(story.harness, (harnessCounts.get(story.harness) ?? 0) + 1);
    }
    for (const story of storyRecords) {
      if (story.harness && harnessCounts.get(story.harness) > 1) {
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
      disposition: dispositions.length > 1 ? "mixed-composition" : dispositions[0] ?? "direct",
      storyRecords,
      diagnostics: [...new Set(storyRecords.flatMap((story) => story.diagnostics.map((item) => item.code)))],
      migrationStatus: storyRecords.some((story) => story.proposedDisposition !== "ready-for-bounded-review") ? "requires-review" : "classified",
      frame: parsed.frame ? { asset: parsed.frame.asset, version: parsed.frame.version ?? null, region: parsed.frame.region } : null,
      localHarness: hasLocalHarness,
    };
  }).sort((a, b) => a.path.localeCompare(b.path));
}

const options = parseArgs(process.argv.slice(2));
const completedStoryKeys = readState(options.stateFile);
const entries = collectEntries();
const storyRecords = entries.flatMap((entry) => entry.storyRecords);
const pending = storyRecords.filter((story) => !completedStoryKeys.has(story.storyKey));
const batch = options.batchSize > 0 ? pending.slice(options.batchIndex * options.batchSize, (options.batchIndex + 1) * options.batchSize) : pending;
const batchKeys = new Set(batch.map((story) => story.storyKey));

process.stdout.write(`${JSON.stringify({
  schemaVersion: 2,
  generatedFrom: "library/**/story.json",
  summary: {
    contractCount: entries.length,
    storyCount: storyRecords.length,
    completedCount: storyRecords.length - pending.length,
    pendingCount: pending.length,
    diagnosticCount: storyRecords.reduce((count, story) => count + story.diagnostics.length, 0),
    unclassifiedCount: storyRecords.filter((story) => !story.proposedDisposition).length,
    exceptionCount: storyRecords.filter((story) => story.exception).length,
    readyForBoundedReviewCount: storyRecords.filter((story) => story.proposedDisposition === "ready-for-bounded-review").length,
    batchSize: options.batchSize || pending.length,
    batchIndex: options.batchSize > 0 ? options.batchIndex : 0,
    batchCount: options.batchSize > 0 ? Math.ceil(pending.length / options.batchSize) : pending.length ? 1 : 0,
  },
  resume: {
    stateFile: options.stateFile,
    completedStoryKeys: [...completedStoryKeys].sort(),
    nextStoryKey: pending[0]?.storyKey ?? null,
    nextBatchStoryKeys: batch.map((story) => story.storyKey),
  },
  entries,
  batch,
}, null, 2)}\n`);
