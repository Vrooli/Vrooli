#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";

const scenarioRoot = path.resolve(import.meta.dirname, "../..");
const harnessRoot = path.join(scenarioRoot, "library", "preview-harnesses");
const manifestPath = path.join(harnessRoot, "manifest.json");
const manifest = JSON.parse(fs.readFileSync(manifestPath, "utf8"));
const referenceStoriesPath = path.join(harnessRoot, manifest.referenceStories ?? "");
const failures = [];
const seen = new Set();
const registrations = new Map();
const stableVersion = /^\d+\.\d+\.\d+$/;
const validIdentifier = /^[A-Za-z_$][A-Za-z0-9_$]*$/;
const forbiddenPatterns = [
  { pattern: /from\s+["'](?:\.\.\/)*components\//, rule: "component-specific import" },
  { pattern: /from\s+["'](?:\.\.\/)*primitives\//, rule: "primitive-specific import" },
  { pattern: /from\s+["'](?:\.\.\/)*hooks\//, rule: "hook-specific import" },
  { pattern: /\b(fetch|XMLHttpRequest|WebSocket)\s*\(/, rule: "network side effect" },
  { pattern: /\b(localStorage|sessionStorage|indexedDB)\b/, rule: "persistent storage access" },
  { pattern: /document\.cookie/, rule: "cookie access" },
];

function fail(message) {
  failures.push(message);
}

if (manifest.schemaVersion !== 1) fail("manifest schemaVersion must be 1");
if (manifest.kind !== "preview-harness-registry") fail("manifest kind is invalid");
if (manifest.ownership !== "preview-only") fail("manifest ownership must be preview-only");
if (!Array.isArray(manifest.families) || manifest.families.length === 0) {
  fail("manifest must declare at least one family");
}

for (const family of manifest.families ?? []) {
  const key = `${family.id}@${family.version}:${family.export}`;
  if (seen.has(key)) fail(`duplicate family registration ${key}`);
  seen.add(key);
  registrations.set(`preview.${family.id}@${family.version}:${family.export}`, family);
  if (!/^[a-z0-9-]+$/.test(family.id ?? "")) {
    fail(`family id must be a lowercase slug: ${family.id}`);
  }
  if (!stableVersion.test(family.version ?? "")) fail(`${family.id}: version must be semantic`);
  if (!validIdentifier.test(family.export ?? ""))
    fail(`${family.id}: export must be a JS identifier`);
  for (const field of ["archetypes", "subjectKinds", "requiredCapabilities", "configKeys"]) {
    if (!Array.isArray(family[field])) fail(`${family.id}: ${field} must be an array`);
  }
  if (family.archetypes?.length === 0) fail(`${family.id}: at least one archetype is required`);
  if (family.subjectKinds?.length === 0)
    fail(`${family.id}: at least one subject kind is required`);
  const sourcePath = path.join(
    harnessRoot,
    family.id,
    "versions",
    family.version,
    `${family.export}.tsx`,
  );
  if (!fs.existsSync(sourcePath)) {
    fail(
      `${family.id}: registered implementation is missing at ${path.relative(scenarioRoot, sourcePath)}`,
    );
    continue;
  }
  const source = fs.readFileSync(sourcePath, "utf8");
  for (const { pattern, rule } of forbiddenPatterns) {
    if (pattern.test(source)) fail(`${family.id}: ${rule} is forbidden`);
  }
  if (family.id !== "showcase" && !source.includes("PreviewShowcase")) {
    fail(`${family.id}: family implementation must use PreviewShowcase`);
  }
}

let referenceStories;
if (!manifest.referenceStories || !fs.existsSync(referenceStoriesPath)) {
  fail("manifest must point to an existing referenceStories file");
} else {
  try {
    referenceStories = JSON.parse(fs.readFileSync(referenceStoriesPath, "utf8"));
  } catch (error) {
    fail(`reference stories are not valid JSON (${error.message})`);
  }
}

const referenceFamilies = new Set();
for (const reference of referenceStories?.stories ?? []) {
  if (!reference.id || !reference.family || !reference.subjectKind || !reference.story) {
    fail("every reference story must declare id, family, subjectKind, and story");
    continue;
  }
  const family = registrations.get(
    `preview.${reference.family}@${manifest.families.find((candidate) => candidate.id === reference.family)?.version}:${manifest.families.find((candidate) => candidate.id === reference.family)?.export}`,
  );
  if (!family) {
    fail(`${reference.id}: reference story family is not registered: ${reference.family}`);
    continue;
  }
  if (!family.subjectKinds.includes(reference.subjectKind)) {
    fail(`${reference.id}: subject kind ${reference.subjectKind} is not allowed by ${reference.family}`);
  }
  referenceFamilies.add(reference.family);
}
for (const family of manifest.families ?? []) {
  if (!referenceFamilies.has(family.id)) fail(`${family.id}: a reference story is required`);
}

function walkStories(directory) {
  if (!fs.existsSync(directory)) return;
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      walkStories(entryPath);
      continue;
    }
    if (entry.name !== "story.json") continue;
    let contract;
    try {
      contract = JSON.parse(fs.readFileSync(entryPath, "utf8"));
    } catch (error) {
      fail(
        `${path.relative(scenarioRoot, entryPath)}: story.json is not valid JSON (${error.message})`,
      );
      continue;
    }
    for (const story of contract.stories ?? []) {
      const ref = story.composition?.harness;
      if (!ref) continue;
      if (story.composition?.specimen) {
        fail(
          `${path.relative(scenarioRoot, entryPath)}#${story.id}: a story cannot declare both specimen and composition harness`,
        );
      }
      const key = `${ref.asset}@${ref.version}:${ref.export}`;
      const family = registrations.get(key);
      if (!family) {
        fail(
          `${path.relative(scenarioRoot, entryPath)}#${story.id}: composition harness is not registered: ${key}`,
        );
        continue;
      }
      if (
        ref.config !== undefined &&
        (ref.config === null || Array.isArray(ref.config) || typeof ref.config !== "object")
      ) {
        fail(
          `${path.relative(scenarioRoot, entryPath)}#${story.id}: composition harness config must be an object`,
        );
        continue;
      }
      for (const configKey of Object.keys(ref.config ?? {})) {
        if (!family.configKeys.includes(configKey)) {
          fail(
            `${path.relative(scenarioRoot, entryPath)}#${story.id}: config key ${configKey} is not declared by ${family.id}`,
          );
        }
      }
    }
  }
}

walkStories(path.join(scenarioRoot, "library"));

if (failures.length > 0) {
  console.error("Preview harness conformance failed:");
  for (const failure of failures) console.error(`- ${failure}`);
  process.exit(1);
}

console.log(`Preview harness conformance passed (${manifest.families.length} families).`);
