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

function storyDisposition(story, hasLocalHarness) {
  if (hasLocalHarness) return "local-harness";
  if (story.sharedHarness) return "shared-harness";
  if (story.frame) return "frame";
  if (story.expect?.length || story.interactions?.length) return "behavioral-direct";
  return "direct";
}

const entries = filesUnder(libraryRoot)
  .filter((file) => path.basename(file) === "story.json")
  .map((file) => {
    const relative = path.relative(scenarioRoot, file).split(path.sep).join("/");
    const parsed = JSON.parse(fs.readFileSync(file, "utf8"));
    const storyDir = path.dirname(file);
    const hasLocalHarness = fs.existsSync(path.join(storyDir, "story.tsx"));
    const stories = Array.isArray(parsed.stories) ? parsed.stories : [];
    const shared = stories.find((story) => story.sharedHarness)?.sharedHarness ?? null;
    const frame = parsed.frame ?? stories.find((story) => story.frame)?.frame ?? null;
    const storyRecords = stories.map((story) => ({
      id: story.id ?? null,
      name: story.name ?? null,
      disposition: storyDisposition(story, hasLocalHarness),
      migration: hasLocalHarness
        ? "requires-review"
        : story.sharedHarness || story.frame
          ? "adopted"
          : "eligible-for-representative-review",
      harness: story.harness ?? null,
      frame: story.frame ?? parsed.frame ?? null,
      sharedHarness: story.sharedHarness ?? null,
      fixtureCount: Array.isArray(parsed.environment?.fixtures)
        ? parsed.environment.fixtures.length
        : 0,
      expectationCount: Array.isArray(story.expect) ? story.expect.length : 0,
      interactionCount: Array.isArray(story.interactions) ? story.interactions.length : 0,
      exception: hasLocalHarness
        ? {
            kind: "intentional-local-harness-review",
            reason:
              "Local executable composition requires behavior-equivalence review before shared-harness migration.",
            owner: "react-component-library Preview maintainers",
            revisitWhen:
              "A matching shared harness family has equivalent interaction, accessibility, and visual evidence.",
          }
        : null,
    }));
    const disposition = hasLocalHarness
      ? "local-harness"
      : shared
        ? "shared-harness"
        : frame
          ? "frame"
          : stories.some((story) => story.expect?.length || story.interactions?.length)
            ? "behavioral-direct"
            : "direct";
    const root = relative.split("/")[1] ?? "unknown";
    return {
      path: relative,
      hierarchy: hierarchy(root),
      schemaVersion: parsed.schemaVersion ?? null,
      kind: parsed.kind ?? null,
      stories: stories.map((story) => story.id).filter(Boolean),
      disposition,
      storyRecords,
      migrationStatus: hasLocalHarness ? "requires-review" : "classified",
      frame: frame
        ? { asset: frame.asset, version: frame.version ?? null, region: frame.region }
        : null,
      sharedHarness: shared
        ? { asset: shared.asset, version: shared.version, export: shared.export }
        : null,
      localHarness: hasLocalHarness,
    };
  })
  .sort((a, b) => a.path.localeCompare(b.path));

process.stdout.write(
  `${JSON.stringify({ schemaVersion: 1, generatedFrom: "library/**/story.json", entries }, null, 2)}\n`,
);
