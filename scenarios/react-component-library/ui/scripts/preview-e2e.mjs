#!/usr/bin/env node
import { createRequire } from "node:module";
import process from "node:process";
import { createHash } from "node:crypto";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const DEFAULT_CHROME = "/usr/bin/google-chrome";
const LIST_COMPONENTS_PATH =
  "/vrooli.react_component_library.v1.components.ComponentsService/ListComponents";
const LIST_COMPONENT_STORIES_PATH =
  "/vrooli.react_component_library.v1.components.ComponentsService/ListComponentStories";

const require = createRequire(import.meta.url);

function loadPlaywright() {
  try {
    return require("playwright");
  } catch {
    return require("../../../../packages/api-base/node_modules/playwright");
  }
}

function baseURL() {
  if (process.env.RCL_PREVIEW_E2E_URL) {
    return process.env.RCL_PREVIEW_E2E_URL.replace(/\/+$/, "");
  }
  const port = process.env.UI_PORT || "21242";
  return `http://localhost:${port}`;
}

function chromeExecutable() {
  return process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || process.env.CHROME_BIN || DEFAULT_CHROME;
}

function escapeRegExp(value) {
  return String(value).replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function fail(message, details) {
  console.error(JSON.stringify({ ok: false, message, ...details }, null, 2));
  process.exitCode = 1;
}

class PreviewFailure extends Error {
  constructor(message, details) {
    super(message);
    this.name = "PreviewFailure";
    this.details = details;
  }
}

function assertNoKnownRuntimeErrors(logs) {
  const forbidden = [
    "does not provide an export named",
    "react/jsx-runtime' does not provide an export named 'jsx'",
    'Dynamic require of "react" is not supported',
    "preview: render failed",
  ];
  const failures = logs.filter((line) => forbidden.some((needle) => line.includes(needle)));
  if (failures.length > 0) {
    throw new Error(`preview emitted runtime errors: ${failures.join(" | ")}`);
  }
}

function compactLogLine(line) {
  return line
    .replace(
      /data:text\/javascript;base64,[A-Za-z0-9+/=]+/g,
      "data:text/javascript;base64,<bundle>",
    )
    .slice(0, 1_200);
}

function compactDetails(details) {
  return {
    logs: (details.logs || []).map(compactLogLine).slice(0, 12),
    previewResponses: (details.previewResponses || []).slice(0, 20),
  };
}

function screenshotArtifactDir() {
  return process.env.RCL_PREVIEW_E2E_ARTIFACT_DIR || process.env.TEST_GENIE_ARTIFACT_DIR || "";
}

function screenshotHash(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

function captureResultPath() {
  return process.env.RCL_PREVIEW_E2E_RESULT_PATH || "";
}

function safeArtifactPart(value) {
  return value.replace(/[^a-zA-Z0-9._-]+/g, "-");
}

function escapeHTML(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function storySheetGroups(stories) {
  const requested = Number(process.env.RCL_PREVIEW_SHEET_SIZE || 4);
  const size = Number.isFinite(requested) ? Math.max(1, Math.min(4, requested)) : 4;
  const groups = [];
  for (let index = 0; index < stories.length; index += size) {
    groups.push(stories.slice(index, index + size));
  }
  return groups;
}

function requestedStoryIDs(storyIDs, storyMetadata) {
  const requested = process.env.RCL_PREVIEW_STORY_IDS?.trim();
  if (requested) {
    if (requested.toLowerCase() === "all") return [...new Set(storyIDs.filter(Boolean))];
    return [...new Set(requested.split(",").map((story) => story.trim()).filter(Boolean))];
  }
  const reviewSet = process.env.RCL_PREVIEW_REVIEW_SET?.trim();
  if (reviewSet) {
    const selected = storyIDs.filter(
      (story) => (storyMetadata.get(story)?.evidence?.reviewSet || "core") === reviewSet,
    );
    if (selected.length === 0) throw new Error(`review set ${reviewSet} has no declared stories`);
    return [...new Set(selected)];
  }
  return [...new Set(storyIDs.filter(Boolean))];
}

function storyState(storyID, metadata) {
  if (metadata?.state) return metadata.state;
  if (metadata?.evidence?.states?.length === 1) return metadata.evidence.states[0];
  const normalized = String(storyID).toLowerCase();
  if (/loading|pending|async|saving|working/.test(normalized)) return "loading";
  if (/error|failed|failure|invalid/.test(normalized)) return "error";
  if (/empty|no-results|no-results/.test(normalized)) return "empty";
  if (/disabled|unavailable|locked/.test(normalized)) return "disabled";
  if (/focus|keyboard/.test(normalized)) return "focus";
  if (/long|stress|overflow/.test(normalized)) return "stress";
  if (/success|ready|resolved|recovered/.test(normalized)) return "success";
  return "default";
}

async function captureStorySheet(page, tiles, metadata) {
  const tilesHTML = tiles
    .map(
      ({ story, storyName, state, sourceArtifact, bytes }) => `
        <article data-preview-sheet-tile>
          <h2>${escapeHTML(storyName || story)}</h2>
          <p data-preview-sheet-story-id>${escapeHTML(story)} · ${escapeHTML(state || "default")}</p>
          <img alt="${escapeHTML(story)} story" src="data:image/png;base64,${Buffer.from(bytes).toString("base64")}" />
          <small>source: ${escapeHTML(sourceArtifact || "capture unavailable")}</small>
        </article>`,
    )
    .join("");
  await page.setContent(
    `<!doctype html>
    <html><head><style>
      :root { color-scheme: ${metadata.theme}; }
      * { box-sizing: border-box; }
      html, body { margin: 0; background: ${metadata.theme === "dark" ? "#111827" : "#f8fafc"}; color: ${metadata.theme === "dark" ? "#f8fafc" : "#0f172a"}; font-family: ui-sans-serif, system-ui, sans-serif; }
      [data-preview-sheet] { display: grid; gap: 16px; padding: 24px; width: min(100%, 1440px); background: inherit; }
      [data-preview-sheet-header] { display: grid; gap: 4px; }
      [data-preview-sheet-header] h1 { margin: 0; font-size: 20px; }
      [data-preview-sheet-header] p { margin: 0; color: ${metadata.theme === "dark" ? "#cbd5e1" : "#475569"}; font-size: 13px; }
      [data-preview-sheet-grid] { display: grid; grid-template-columns: repeat(${tiles.length > 1 ? 2 : 1}, minmax(0, 1fr)); gap: 16px; align-items: start; }
      [data-preview-sheet-tile] { display: grid; gap: 8px; min-width: 0; }
      [data-preview-sheet-tile] h2 { margin: 0; font-size: 13px; font-weight: 700; }
      [data-preview-sheet-story-id], [data-preview-sheet-tile] small { margin: 0; color: ${metadata.theme === "dark" ? "#cbd5e1" : "#475569"}; font-size: 11px; overflow-wrap: anywhere; }
      [data-preview-sheet-tile] img { display: block; width: 100%; height: auto; border: 1px solid ${metadata.theme === "dark" ? "#475569" : "#cbd5e1"}; border-radius: 10px; }
      [data-preview-sheet-footer] { margin: 0; color: ${metadata.theme === "dark" ? "#94a3b8" : "#64748b"}; font-size: 11px; }
    </style></head><body>
      <main data-preview-sheet="story-gallery" data-preview-capture-boundary="component-sheet">
        <header data-preview-sheet-header><h1>${escapeHTML(metadata.title)}</h1><p>${tiles.length} stories · ${escapeHTML(metadata.version || "version unknown")} · ${escapeHTML(metadata.kit || "kit unknown")} · ${escapeHTML(metadata.theme)} · ${escapeHTML(metadata.viewport)}</p></header>
        <div data-preview-sheet-grid>${tilesHTML}</div>
        <p data-preview-sheet-footer>source manifest: ${escapeHTML(metadata.manifestPath || "capture-manifest.json")}</p>
      </main>
    </body></html>`,
    { waitUntil: "load" },
  );
  const sheet = page.locator("[data-preview-sheet]");
  await sheet.waitFor({ state: "visible" });
  return sheet.screenshot({ animations: "disabled" });
}

function previewViewports() {
  const requested = process.env.RCL_PREVIEW_VIEWPORTS;
  if (!requested) return [{ name: "desktop", width: 1254, height: 720 }];
  return requested.split(",").map((entry) => {
    const [name, dimensions] = entry.trim().split(":");
    const [width, height] = (dimensions || "").split("x").map(Number);
    if (
      !name ||
      !Number.isFinite(width) ||
      !Number.isFinite(height) ||
      width < 320 ||
      height < 160
    ) {
      throw new Error(`invalid RCL_PREVIEW_VIEWPORTS entry ${entry}; expected name:WIDTHxHEIGHT`);
    }
    return { name, width, height };
  });
}

function previewKits() {
  const requested = process.env.RCL_PREVIEW_KITS;
  if (!requested) return ["vrooli-default"];
  const kits = requested
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);
  if (kits.length === 0) throw new Error("RCL_PREVIEW_KITS must contain at least one design kit");
  return [...new Set(kits)];
}

function themeVariantKits() {
  const requested = process.env.RCL_PREVIEW_THEME_VARIANT_KITS;
  if (!requested) return ["vrooli-default"];
  return requested
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);
}

function themesForKit(kit) {
  return themeVariantKits().includes(kit) ? ["light", "dark"] : ["dark"];
}

function assetTimeoutMs() {
  const requested = Number(process.env.RCL_PREVIEW_ASSET_TIMEOUT_MS);
  return Number.isFinite(requested) && requested > 0 ? Math.min(requested, 600_000) : 120_000;
}

function storyTimeoutMs() {
  const requested = Number(process.env.RCL_PREVIEW_STORY_TIMEOUT_MS);
  const fallback = Math.min(assetTimeoutMs(), 60_000);
  return Number.isFinite(requested) && requested > 0
    ? Math.min(requested, assetTimeoutMs())
    : fallback;
}

async function withTimeout(operation, timeoutMs, label) {
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => {
      reject(new Error(`preview timed out after ${timeoutMs}ms: ${label}`));
    }, timeoutMs);
  });
  try {
    return await Promise.race([operation(), timeout]);
  } finally {
    clearTimeout(timer);
  }
}

async function withAssetTimeout(operation, label) {
  return withTimeout(operation, assetTimeoutMs(), `asset ${label}`);
}

function classifyCaptureFailure(error) {
  const message = error instanceof Error ? error.message : String(error);
  if (/timed out/i.test(message)) {
    return { stage: "settle", category: "capture-infrastructure", retryable: true };
  }
  if (/404|not found|story mismatch|no indexed story/i.test(message)) {
    return { stage: "resolve", category: "resolver-contract", retryable: false };
  }
  if (/expectation|expected .*count|unsupported browser expectation/i.test(message)) {
    return { stage: "assert", category: "expectation", retryable: false };
  }
  if (/overflow|unexpectedly small|blank .*background|blank root|did not mount|preview error|root=\)/i.test(message)) {
    return { stage: "mount", category: "product-rendering", retryable: false };
  }
  if (/screenshot|component sheet|capture/i.test(message)) {
    return { stage: "capture", category: "capture-infrastructure", retryable: true };
  }
  if (/requestfailed|ERR_ABORTED|navigation|browser|context/i.test(message)) {
    return { stage: "navigate", category: "environment", retryable: true };
  }
  return { stage: "navigate", category: "capture-infrastructure", retryable: true };
}

async function selectResolvedTheme(page, theme) {
  const mode = page.locator(`[data-testid="components-theme-switcher-mode-${theme}"]`);
  if (!(await mode.isVisible())) {
    await page.locator('[data-testid="components-theme-switcher-appearance-toggle"]').click();
    await mode.waitFor({ state: "visible", timeout: 5_000 });
  }
  await mode.click();
  await page.waitForTimeout(100);
}

async function captureThemeTier(
  page,
  frameElements,
  componentID,
  storyIDs = [],
  sourceOverride = "",
  componentLabel = componentID,
  storyMetadata = new Map(),
) {
  const captures = {};
  const outputDir = screenshotArtifactDir();
  if (outputDir) await mkdir(outputDir, { recursive: true });
  // Screenshots taken from an iframe element can include the host's painted
  // surface where the child document is transparent. That made the old
  // artifact look like a component plus the editor's tool panels. Reopen the
  // exact harness URL in an isolated page so the artifact is ground truth for
  // the component document itself.
  const captureContexts = [];
  const viewports = previewViewports();
  const kits = previewKits();
  const requestedStory = process.env.RCL_PREVIEW_STORY_ID?.trim();
  const selectedStoryIDs = requestedStory
    ? [requestedStory]
    : requestedStoryIDs(storyIDs, storyMetadata);
  const stories = selectedStoryIDs.length > 0 ? selectedStoryIDs : [undefined];
  if (stories.filter(Boolean).length !== new Set(stories.filter(Boolean)).size) {
    throw new Error("capture request contains duplicate story IDs");
  }
  const sheetMode = process.env.RCL_PREVIEW_STORY_SHEET === "1" && stories.length > 1;
  const sheetGroups = sheetMode ? storySheetGroups(stories.filter(Boolean)) : [];

  try {
    const mountedSources = new Set(sourceOverride ? [sourceOverride] : []);
    const count = await frameElements.count();
    for (let index = 0; index < count; index += 1) {
      const source = await frameElements.nth(index).getAttribute("src");
      if (source?.includes("story=")) mountedSources.add(source);
    }
    // The editor may expose the new URL through its live Frame before the
    // iframe element attribute is repainted. Prefer that story-pinned URL so
    // isolated captures never reopen the empty initial harness.
    for (const frame of page.frames()) {
      if (
        frame.url().includes(`/preview/${componentID}/harness.html`) &&
        frame.url().includes("story=")
      ) {
        mountedSources.add(frame.url());
      }
    }
    if (mountedSources.size === 0)
      throw new Error("no story-pinned preview source available for capture");
    const sources = [...mountedSources];
    const parsedSourceEntries = sources.map((source, index) => ({
      source,
      index,
      story: new URL(source).searchParams.get("story") || undefined,
    }));
    // A focused run must select the matching story-pinned source rather than
    // replaying the requested story against every iframe source mounted by the
    // editor. That old fallback produced duplicate authoritative rows for a
    // one-story request and made source cardinality look like subject
    // cardinality.
    const matchingSources = parsedSourceEntries.filter(
      (entry) => !entry.story || stories.includes(entry.story),
    );
    const sourceEntries =
      stories.length === 1 && matchingSources.some((entry) => entry.story === stories[0])
        ? matchingSources.filter((entry) => entry.story === stories[0])
        : stories.length > 1
          ? matchingSources
          : [matchingSources[0] || parsedSourceEntries[0]];
    for (const kit of kits) {
      for (const viewport of viewports) {
        const captureContext = await page
          .context()
          .browser()
          .newContext({ viewport: { width: viewport.width, height: viewport.height } });
        captureContexts.push(captureContext);
        const capturePage = await captureContext.newPage();
        capturePage.setDefaultTimeout(assetTimeoutMs());
        capturePage.setDefaultNavigationTimeout(assetTimeoutMs());
        for (const theme of themesForKit(kit)) {
          await selectResolvedTheme(page, theme);
          await page.waitForTimeout(100);
          captures[theme] ||= [];
          const sheetTiles = new Map();
          for (const sourceEntry of sourceEntries) {
            const { source, index } = sourceEntry;
            const storiesForSource = sourceEntry.story
              ? stories.filter((story) => story === sourceEntry.story)
              : stories;
            for (const storyID of storiesForSource) {
              await withTimeout(async () => {
                const kitSource = new URL(source);
                kitSource.searchParams.set("kit", kit);
                // Preserve the story's declared frame. Isolated capture removes
                // the editor workspace, but it must retain the component's
                // declared composition context so the artifact proves the
                // component works inside its intended sheet.
                // A focused capture may target any declared story without
                // changing the catalog route or the selected story in the host
                // editor. This keeps visual ground truth available for states
                // such as an exiting/hidden Presence boundary.
                if (storyID) kitSource.searchParams.set("story", storyID);
                const forcedFailure = process.env.RCL_PREVIEW_FORCE_FAILURE || "";
                if (forcedFailure === "404") {
                  throw new Error(
                    `preview route returned 404 for ${baseURL()}/preview/does-not-exist/harness.html?story=${encodeURIComponent(storyID || "default")}`,
                  );
                }
                await capturePage.goto(kitSource.toString(), { waitUntil: "domcontentloaded" });
                if (forcedFailure === "timeout") {
                  await new Promise((resolve) => setTimeout(resolve, storyTimeoutMs() + 250));
                }
                if (forcedFailure === "blank-root") {
                  await capturePage.evaluate(() => document.querySelector("#root")?.replaceChildren());
                }
                if (forcedFailure === "expectation") {
                  throw new Error(`story ${storyID || "default"} forced expectation failure`);
                }
              try {
                await capturePage
                  .locator("#root > *")
                  .first()
                  .waitFor({ state: "attached", timeout: 10_000 });
              } catch (error) {
                const previewError = await capturePage
                  .locator("#preview-error")
                  .innerText()
                  .catch(() => "");
                throw new Error(
                  `isolated preview did not mount at ${capturePage.url()}${
                    previewError
                      ? `: ${previewError}`
                      : ` (root=${(
                          await capturePage
                            .locator("#root")
                            .innerHTML()
                            .catch(() => "")
                        ).slice(0, 160)})`
                  }`,
                  { cause: error },
                );
              }
              const expectedStory = kitSource.searchParams.get("story");
              if (expectedStory) {
                const renderedStory = await capturePage
                  .locator('meta[name="story-id"]')
                  .getAttribute("content");
                if (renderedStory !== expectedStory) {
                  throw new Error(
                    `isolated preview story mismatch: requested ${expectedStory}, rendered ${renderedStory || "<empty>"} at ${capturePage.url()}`,
                  );
                }
              }
              // The harness installs its theme bridge before the React module
              // resolves, but posting after the first mount makes this
              // isolated path resilient to a cold module graph as well.
              await capturePage.evaluate((resolvedTheme) => {
                window.postMessage({ type: "rcl-resolved-theme", theme: resolvedTheme }, "*");
              }, theme);
              await capturePage.waitForFunction(
                (expected) => document.documentElement.dataset.resolvedTheme === expected,
                theme,
                { timeout: 5_000 },
              );
              // Font, token, and component styles can settle one frame after the
              // resolved-theme marker flips. Measure after a bounded paint
              // window so a transient pre-layout width cannot become a false
              // responsive failure.
              await capturePage.waitForTimeout(300);
              const visual = await capturePage.locator("#root").evaluate((element) => {
                const style = getComputedStyle(element);
                const rect = element.getBoundingClientRect();
                return {
                  background: style.backgroundColor,
                  width: rect.width,
                  height: rect.height,
                  interactiveCount: element.querySelectorAll(
                    "button, input, select, textarea, a[href]",
                  ).length,
                  // Measure the rendered component surface, not hidden
                  // harness bookkeeping nodes outside #root. The screenshot
                  // artifact is the explicit component sheet, while layout
                  // validation remains rooted at the full rendered surface.
                  scrollWidth: element.scrollWidth,
                  viewportWidth: window.innerWidth,
                  overflowing: Array.from(element.querySelectorAll("*"))
                    .map((node) => {
                      const rect = node.getBoundingClientRect();
                      return {
                        tag: node.tagName.toLowerCase(),
                        className: node.className || "",
                        left: Math.round(rect.left),
                        right: Math.round(rect.right),
                        width: Math.round(rect.width),
                      };
                    })
                    .filter((item) => item.left < -1 || item.right > window.innerWidth + 1)
                    .slice(0, 8),
                };
              });
              if (
                !visual.background ||
                visual.background === "rgba(0, 0, 0, 0)" ||
                visual.background === "transparent"
              ) {
                throw new Error(
                  `preview iframe ${index} has a blank ${theme} background at ${viewport.name} kit ${kit}`,
                );
              }
              if (visual.width < 320 || visual.height < 160)
                throw new Error(
                  `preview iframe ${index} ${theme} visual surface is unexpectedly small at ${viewport.name} kit ${kit}`,
                );
              if (
                process.env.RCL_PREVIEW_STRICT_LAYOUT === "1" &&
                visual.scrollWidth > visual.viewportWidth + 1
              ) {
                throw new Error(
                  `preview iframe ${index} overflows ${viewport.name} kit ${kit}: scrollWidth=${visual.scrollWidth}, viewportWidth=${visual.viewportWidth}${visual.overflowing.length > 0 ? ` offenders=${JSON.stringify(visual.overflowing)}` : ""}`,
                );
              }
              const sheet = capturePage.locator("[data-preview-sheet]");
              const sheetCount = await sheet.count();
              if (sheetCount !== 1) {
                throw new Error(
                  `isolated preview must expose exactly one component sheet, found ${sheetCount} at ${capturePage.url()}`,
                );
              }
              await sheet.waitFor({ state: "visible", timeout: 5_000 });
              const bytes = await sheet.screenshot({ animations: "disabled" });
              if (bytes.length < 256)
                throw new Error(
                  `preview iframe ${index} ${theme} screenshot is unexpectedly small at ${viewport.name} kit ${kit}`,
                );
              const hash = screenshotHash(bytes);
              const story = kitSource.searchParams.get("story") || "default";
              const artifact = `${safeArtifactPart(componentID)}--${safeArtifactPart(story)}--${index}--${safeArtifactPart(kit)}--${safeArtifactPart(viewport.name)}--${theme}.png`;
              if (outputDir) await writeFile(path.join(outputDir, artifact), bytes);
              captures[theme].push({
                kit,
                story,
                theme,
                storyName: storyMetadata.get(story)?.name || story,
                specimen: storyMetadata.get(story)?.harness || null,
                harness: storyMetadata.get(story)?.sharedHarness || null,
                frame: storyMetadata.get(story)?.frame || null,
                fixtureIds: storyMetadata.get(story)?.fixtureIds || [],
                state: storyState(story, storyMetadata.get(story)),
                automated: {
                  consoleErrors: 0,
                  overflow: visual.scrollWidth > visual.viewportWidth + 1,
                  a11yViolations: null,
                },
                human: { status: "needs-review", notes: [] },
                index,
                viewport: viewport.name,
                viewportWidth: visual.viewportWidth,
                viewportHeight: viewport.height,
                scrollWidth: visual.scrollWidth,
                background: visual.background,
                bytes: bytes.length,
                hash,
                width: visual.width,
                height: visual.height,
                interactiveCount: visual.interactiveCount,
                captureTarget: "component-sheet",
                captureSelector: "[data-preview-sheet]",
                artifact: outputDir ? artifact : undefined,
              });
              const tileKey = sheetMode ? "all" : `${index}`;
              const tiles = sheetTiles.get(tileKey) || [];
              tiles.push({
                story,
                storyName: storyMetadata.get(story)?.name || story,
                state: storyState(story, storyMetadata.get(story)),
                bytes,
                sourceArtifact: captures[theme].at(-1)?.artifact || null,
              });
              sheetTiles.set(tileKey, tiles);
              }, storyTimeoutMs(), `story ${storyID || "default"} · ${kit} · ${viewport.name} · ${theme}`);
            }
          }
          if (sheetMode && outputDir) {
            const sheetPage = await captureContext.newPage();
            sheetPage.setDefaultTimeout(assetTimeoutMs());
            for (const [sourceIndex, tiles] of sheetTiles) {
              for (let groupIndex = 0; groupIndex < sheetGroups.length; groupIndex += 1) {
                const group = sheetGroups[groupIndex];
                const selectedTiles = group
                  .map((story) => tiles.find((tile) => tile.story === story))
                  .filter(Boolean);
                if (selectedTiles.length === 0) continue;
                if (selectedTiles.length !== group.length) {
                  throw new Error(
                    `review sheet is missing ${group.length - selectedTiles.length} source capture(s) for ${group.join(", ")}`,
                  );
                }
                const sourceArtifacts = selectedTiles
                  .map((tile) => tile.sourceArtifact)
                  .filter(Boolean);
                if (new Set(sourceArtifacts).size !== sourceArtifacts.length) {
                  throw new Error(`review sheet has duplicate source captures for ${group.join(", ")}`);
                }
                const sourceURL = sources[Number(sourceIndex)] || sources[0];
                const bytes = await captureStorySheet(sheetPage, selectedTiles, {
                  title: `${componentLabel} stories`,
                  version: new URL(sourceURL).searchParams.get("version") || "version unknown",
                  kit,
                  theme,
                  viewport: viewport.name,
                  manifestPath: "capture-manifest.json",
                });
                const storyLabel = group.map((story) => safeArtifactPart(story)).join("-");
                const artifact = `${safeArtifactPart(componentID)}--stories-${storyLabel}--${sourceIndex}--${safeArtifactPart(kit)}--${safeArtifactPart(viewport.name)}--${theme}.png`;
                await writeFile(path.join(outputDir, artifact), bytes);
                captures[theme].push({
                  stories: group,
                  sourceArtifacts,
                  theme,
                  index: Number(sourceIndex),
                  viewport: viewport.name,
                  viewportWidth: viewport.width,
                  viewportHeight: viewport.height,
                  kit,
                  bytes: bytes.length,
                  hash: screenshotHash(bytes),
                  captureTarget: "component-sheet",
                  captureSelector: "[data-preview-sheet]",
                  sheetSize: selectedTiles.length,
                  artifact,
                });
              }
            }
            await sheetPage.close();
          }
        }
      }
    }
  } finally {
    await Promise.all(captureContexts.map((context) => context.close()));
  }

  for (const light of captures.light || []) {
    if (light.stories) continue;
    const dark = (captures.dark || []).find(
      (candidate) =>
        candidate.kit === light.kit &&
        candidate.story === light.story &&
        candidate.index === light.index &&
        candidate.viewport === light.viewport,
    );
    if (!dark)
      throw new Error(
        `theme screenshot frame missing dark pair for kit ${light.kit} at ${light.viewport}`,
      );
    if (themeVariantKits().includes(light.kit)) {
      if (light.background === dark.background) {
        throw new Error(
          `preview iframe ${light.index} background did not change between light and dark for kit ${light.kit} at ${light.viewport} story ${light.story}`,
        );
      }
      if (light.hash === dark.hash) {
        throw new Error(
          `preview iframe ${light.index} screenshot did not change between light and dark for kit ${light.kit} at ${light.viewport} story ${light.story}`,
        );
      }
    }
  }
  return captures;
}

async function previewableAssetTargets() {
  if (process.env.RCL_PREVIEW_COMPONENT_ID) {
    const requested = process.env.RCL_PREVIEW_COMPONENT_ID.split(",")
      .map((value) => value.trim())
      .filter(Boolean);
    const isUUID = (value) => /^[0-9a-f-]{36}$/i.test(value);
    if (requested.every(isUUID)) return requested.map((id) => ({ id, label: id }));
    const response = await fetch(`${baseURL()}${LIST_COMPONENTS_PATH}`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json" },
      body: JSON.stringify({ limit: 500, assetKind: 1 }),
    });
    if (!response.ok)
      throw new Error(`typed catalog query failed: ${response.status} ${await response.text()}`);
    const payload = await response.json();
    const normalize = (value) =>
      String(value)
        .replace(/[^a-z0-9]/gi, "")
        .toLowerCase();
    return requested.map((value) => {
      const key = normalize(value.split(".").pop());
      const match = (payload.components || []).find(
        (component) =>
          component.id === value ||
          component.libraryId === value ||
          component.displayName === value ||
          normalize(component.displayName) === key ||
          normalize(component.slug) === key,
      );
      if (!match?.id)
        throw new Error(
          `requested preview component ${value} was not found by id, library id, display name, or catalog slug`,
        );
      return {
        id: match.id,
        label: match.displayName || match.libraryId || match.id,
        sourcePath: match.sourcePath || "",
      };
    });
  }
  const response = await fetch(`${baseURL()}${LIST_COMPONENTS_PATH}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify({ limit: 500, assetKind: 1 }),
  });
  if (!response.ok)
    throw new Error(`typed catalog query failed: ${response.status} ${await response.text()}`);
  const payload = await response.json();
  const targets = (payload.components || [])
    .filter((component) => component.id)
    .map((component) => ({
      id: component.id,
      label: component.displayName || component.libraryId || component.id,
      sourcePath: component.sourcePath || "",
    }));
  const byID = new Map();
  for (const target of targets) {
    byID.set(target.id, target);
  }
  return [...byID.values()].sort((a, b) => a.label.localeCompare(b.label));
}

async function componentStories(componentID) {
  const response = await fetch(`${baseURL()}${LIST_COMPONENT_STORIES_PATH}`, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify({ componentId: componentID, limit: 500 }),
  });
  if (!response.ok)
    throw new Error(`typed story query failed: ${response.status} ${await response.text()}`);
  const payload = await response.json();
  const stories = new Map();
  for (const contract of payload.stories || []) {
    try {
      for (const story of JSON.parse(contract.storiesJson || "[]")) {
        // Story IDs are scoped to a component version. Reusing an ID such as
        // "primary" across released contracts must not let a historical
        // expectation overwrite the version currently rendered in the frame.
        stories.set(`${contract.version}:${story.id}`, {
          ...story,
          expect: story.expect || [],
          fixtureIds: contract.environment?.fixtures || [],
        });
      }
    } catch {
      throw new Error("indexed story contract has invalid storiesJson");
    }
  }
  return stories;
}

async function assertStoryExpectations(frame, name, expectations) {
  for (const expectation of expectations) {
    if (expectation.kind === "count") {
      if (!expectation.selector) {
        throw new Error(`story ${name} has a count expectation without a selector`);
      }
      const expected = Number(expectation.value);
      if (!Number.isInteger(expected) || expected < 0) {
        throw new Error(`story ${name} has an invalid count expectation value ${JSON.stringify(expectation.value)}`);
      }
      const actual = await frame.locator(expectation.selector).count();
      if (actual !== expected) {
        throw new Error(
          `story ${name} expected ${expectation.selector} count=${expected}, got ${actual}`,
        );
      }
      continue;
    }
    if (expectation.kind === "role") {
      await frame
        .getByRole(expectation.role, { name: expectation.name, exact: true })
        .waitFor({ state: "visible", timeout: 10_000 });
      continue;
    }
    if (expectation.kind === "text") {
      await frame
        .getByText(expectation.value, { exact: false })
        .waitFor({ state: "visible", timeout: 10_000 });
      continue;
    }
    if (expectation.kind === "attribute") {
      const attribute = expectation.attribute || expectation.name;
      if (!attribute) {
        throw new Error(`story ${name} has an attribute expectation without an attribute name`);
      }
      const value = await frame
        .locator(expectation.selector)
        .first()
        .getAttribute(attribute, { timeout: 10_000 });
      const expected = expectation.value ?? "";
      if ((value ?? "") !== expected)
        throw new Error(
          `story ${name} expected ${expectation.selector}[${attribute}]=${JSON.stringify(expected)}, got ${JSON.stringify(value)}`,
        );
      continue;
    }
    if (expectation.kind === "layout") {
      const target = frame.locator(expectation.selector).first();
      await target.waitFor({ state: "visible", timeout: 10_000 });
      const metrics = await target.evaluate((element) => {
        const rect = element.getBoundingClientRect();
        return {
          width: rect.width,
          height: rect.height,
          clientWidth: element.clientWidth,
          scrollWidth: element.scrollWidth,
        };
      });
      if (expectation.minWidth !== undefined && metrics.width < Number(expectation.minWidth))
        throw new Error(
          `story ${name} expected ${expectation.selector} width >= ${expectation.minWidth}, got ${metrics.width}`,
        );
      if (expectation.minHeight !== undefined && metrics.height < Number(expectation.minHeight))
        throw new Error(
          `story ${name} expected ${expectation.selector} height >= ${expectation.minHeight}, got ${metrics.height}`,
        );
      if (expectation.maxWidth !== undefined && metrics.width > Number(expectation.maxWidth))
        throw new Error(
          `story ${name} expected ${expectation.selector} width <= ${expectation.maxWidth}, got ${metrics.width}`,
        );
      if (expectation.maxHeight !== undefined && metrics.height > Number(expectation.maxHeight))
        throw new Error(
          `story ${name} expected ${expectation.selector} height <= ${expectation.maxHeight}, got ${metrics.height}`,
        );
      if (expectation.noOverflow && metrics.scrollWidth > metrics.clientWidth + 1)
        throw new Error(
          `story ${name} expected ${expectation.selector} not to overflow horizontally: scrollWidth=${metrics.scrollWidth}, clientWidth=${metrics.clientWidth}`,
        );
      continue;
    }
    throw new Error(
      `story ${name} has unsupported browser expectation kind ${String(expectation.kind)}`,
    );
  }
}

async function assertAssetPreview(page, componentID, target = {}) {
  const assetPath = `/assets/${encodeURIComponent(componentID)}`;
  page.setDefaultTimeout(assetTimeoutMs());
  page.setDefaultNavigationTimeout(assetTimeoutMs());
  const logs = [];
  const responses = [];

  const onConsole = (msg) => logs.push(`console:${msg.type()}:${msg.text()}`);
  const onPageError = (err) => logs.push(`pageerror:${err.stack || err.message}`);
  const onRequestFailed = (req) => {
    logs.push(`requestfailed:${req.url()}:${req.failure()?.errorText || "unknown"}`);
  };
  const onResponse = (res) => {
    const resURL = res.url();
    if (resURL.includes("/preview/") || res.status() >= 400) {
      responses.push(`${res.status()}:${resURL}`);
    }
  };

  page.on("console", onConsole);
  page.on("pageerror", onPageError);
  page.on("requestfailed", onRequestFailed);
  page.on("response", onResponse);

  try {
    // Enter through the catalog and follow its current asset route. This
    // proves the catalog link contract while the server independently supports
    // refreshing the resulting detail deep link.
    await page.goto(`${baseURL()}/`, { waitUntil: "domcontentloaded" });
    // The catalog's persistent asset navigator lives alongside the main
    // workspace, not inside it. Constraining this lookup to app-main made the
    // runtime gate falsely fail for every valid catalog item.
    // The responsive shell keeps both desktop and mobile catalog navigators in
    // the DOM. The first matching link can therefore be intentionally hidden
    // at the desktop capture width; select the visible route affordance so a
    // focused component run exercises the same navigation contract as the
    // catalog-scale sweep.
    const assetLink = page.locator(`a[href="${assetPath}"]:visible`).first();
    const assetTreeItem = page
      .getByRole("treeitem", { name: new RegExp(`\\b${escapeRegExp(target.label)}\\b`) })
      .first();
    if (await assetLink.count()) {
      await assetLink.waitFor({ state: "visible", timeout: 15_000 });
      await assetLink.click();
    } else {
      await assetTreeItem.waitFor({ state: "visible", timeout: 15_000 });
      await assetTreeItem.click();
    }
    await page
      .locator('[data-testid="components-editor-panel"]')
      .waitFor({ state: "visible", timeout: 15_000 });
    // The runner uses the desktop viewport where preview is mounted with the
    // other workspace panes. Do not click the responsive mode toggle: in the
    // desktop shell it is an action toggle and can hide the active preview.

    const frameElements = page.locator('[data-testid="components-editor-preview-frame"]');
    // The catalog route can resolve the component content and its indexed
    // story contract independently. On a cold API/database cache the host
    // panel is visible before the first real specimen can be mounted; allow
    // that bounded loading state to settle instead of racing it.
    await frameElements.first().waitFor({ state: "attached", timeout: 30_000 });

    const frameSrc = await frameElements.first().getAttribute("src");
    if (!frameSrc?.includes(`/preview/${componentID}/harness.html`)) {
      throw new Error(`preview iframe did not point at harness: ${frameSrc || "<empty>"}`);
    }

    await page.waitForFunction(
      () => {
        const badge = document.querySelector('[data-testid="components-editor-preview-badge"]');
        const error = document.querySelector('[data-testid="components-editor-preview-error"]');
        return Boolean(badge) || Boolean(error);
      },
      null,
      { timeout: 10_000 },
    );

    // Wait for the editor to finish replacing its initial loading iframe
    // before taking a Frame handle. A handle captured during that navigation
    // can remain valid while pointing at the empty predecessor document.
    await page.waitForFunction(
      () =>
        Array.from(
          document.querySelectorAll('[data-testid="components-editor-preview-frame"]'),
        ).some((frame) => (frame.getAttribute("src") || "").includes("story=")),
      null,
      { timeout: 10_000 },
    );

    // Resolve the live page frame after navigation rather than deriving a
    // Frame handle from an iframe element. Playwright can retain an element
    // handle across the editor's iframe replacement while its Frame object
    // still points at the aborted loading document.
    const frameCount = await frameElements.count();
    const previewFrames = page
      .frames()
      .filter(
        (frame) =>
          frame.url().includes(`/preview/${componentID}/harness.html`) &&
          frame.url().includes("story="),
      );
    if (previewFrames.length === 0) {
      throw new Error(
        `expected a story-pinned preview frame, found ${frameCount} mounted frame(s)`,
      );
    }
    const expectations = await componentStories(componentID);
    const frameResults = [];
    for (const previewFrame of previewFrames) {
      if (previewFrame.url().includes("/assets/")) {
        throw new Error(`preview frame recursively loaded the app route: ${previewFrame.url()}`);
      }
      await previewFrame
        .locator("#root > *")
        .first()
        .waitFor({ state: "attached", timeout: 30_000 });
      const rootHTML = await previewFrame.locator("#root").innerHTML();
      const iframeError = await previewFrame.locator("#preview-error").innerText();
      if (iframeError.trim() !== "") {
        throw new Error(`iframe preview error rendered: ${iframeError}`);
      }
      if (rootHTML.trim() === "") {
        throw new Error("preview root was empty after rendered state");
      }
      const frameURL = new URL(previewFrame.url());
      const story = frameURL.searchParams.get("story") || "__default__";
      const version = frameURL.searchParams.get("version") || "__current__";
      const declared = expectations.get(`${version}:${story}`);
      if (!declared) throw new Error(`preview frame ${story} has no indexed story expectations`);
      await assertStoryExpectations(previewFrame, story, declared.expect);
      frameResults.push({
        url: previewFrame.url(),
        story,
        storyName: declared.name || story,
        specimen: declared.harness || null,
        harness: declared.sharedHarness || null,
        frame: declared.frame || null,
        fixtureIds: declared.fixtureIds || [],
        expectationCount: declared.expect.length,
      });
    }
    const hostError = await page
      .locator('[data-testid="components-editor-preview-error"]')
      .innerText({ timeout: 250 })
      .catch(() => "");
    const badge = await page
      .locator('[data-testid="components-editor-preview-badge"]')
      .innerText({ timeout: 1_000 })
      .catch(() => "");

    if (hostError.trim() !== "") {
      throw new Error(`host preview error rendered: ${hostError}`);
    }
    if (!badge.includes("Rendered")) {
      throw new Error(`preview did not reach rendered state, badge=${badge || "<empty>"}`);
    }

    const renderedVersion = frameResults[0]
      ? new URL(frameResults[0].url).searchParams.get("version") || ""
      : "";
    const storyMetadata = new Map(
      [...expectations.entries()]
        .filter(([key]) => !renderedVersion || key.startsWith(`${renderedVersion}:`))
        .map(([key, value]) => [key.slice(key.indexOf(":") + 1), value]),
    );
    const storyIDs = [...storyMetadata.keys()]
    const screenshots = await captureThemeTier(
      page,
      frameElements,
      componentID,
      storyIDs,
      frameResults[0]?.url || frameSrc,
      target.label,
      storyMetadata,
    );
    assertNoKnownRuntimeErrors(logs);
    const result = {
      ok: true,
      componentID,
      frameSrc,
      frames: frameResults,
      screenshots,
      badge,
      previewResponses: responses,
    };
    if (screenshotArtifactDir()) {
      const outputDir = screenshotArtifactDir();
      const captureRows = Object.values(screenshots)
        .flat()
        .filter((entry) => entry.artifact)
        .map((entry) => ({
          assetId: target.id,
          assetLabel: target.label,
          version: renderedVersion,
          storyId: entry.story || null,
          storyName: entry.storyName || entry.story || null,
          specimen: entry.specimen || null,
          frame: entry.frame || null,
          harness: entry.harness || null,
          fixtureIds: entry.fixtureIds || [],
          kit: entry.kit,
          theme: entry.theme || null,
          viewport: { name: entry.viewport, width: entry.viewportWidth, height: entry.viewportHeight || null },
          state: entry.state || "default",
          artifactKind: entry.stories ? "review-sheet" : "individual",
          artifact: entry.artifact,
          sourceArtifacts: entry.sourceArtifacts || [],
          hash: entry.hash,
          automated: entry.automated || {},
          human: entry.human || { status: "needs-review", notes: [] },
        }));
      await writeFile(
        path.join(outputDir, `capture-manifest-${safeArtifactPart(target.id)}.json`),
        `${JSON.stringify({ schemaVersion: 2, kind: "react-component-library-preview-capture", generatedAt: new Date().toISOString(), captures: captureRows }, null, 2)}\n`,
      );
    }
    return result;
  } catch (error) {
    throw new PreviewFailure(error instanceof Error ? error.message : String(error), {
      logs,
      previewResponses: responses,
    });
  } finally {
    page.off("console", onConsole);
    page.off("pageerror", onPageError);
    page.off("requestfailed", onRequestFailed);
    page.off("response", onResponse);
  }
}

function manifestRows(results) {
  return results.flatMap((result) =>
    Object.values(result.screenshots || {})
      .flat()
      .filter((entry) => entry.artifact)
      .map((entry) => ({
        assetId: result.componentID,
        assetLabel: result.label,
        version: entry.version || new URL(result.frames[0]?.url || "http://localhost").searchParams.get("version") || null,
        storyId: entry.story || null,
        storyIds: entry.stories || null,
        artifactKind: entry.stories ? "review-sheet" : "individual",
        storyName: entry.storyName || entry.story || null,
        specimen: entry.specimen || null,
        frame: entry.frame || null,
        harness: entry.harness || null,
        fixtureIds: entry.fixtureIds || [],
        kit: entry.kit,
        theme: entry.theme || null,
        viewport: { name: entry.viewport, width: entry.viewportWidth || null, height: entry.viewportHeight || null },
        state: entry.state || "default",
        artifact: entry.artifact,
        sourceArtifacts: entry.sourceArtifacts || [],
        hash: entry.hash,
        automated: entry.automated || {},
        human: entry.human || { status: "needs-review", notes: [] },
      })),
  );
}

function captureRequest(componentTargets) {
  return {
    assets: componentTargets.map((target) => ({ id: target.id, label: target.label })),
    stories: process.env.RCL_PREVIEW_STORY_IDS?.trim() || "all",
    reviewSet: process.env.RCL_PREVIEW_REVIEW_SET?.trim() || "core",
    kits: previewKits(),
    themes: [...new Set(previewKits().flatMap((kit) => themesForKit(kit)))],
    viewports: previewViewports(),
    storyTimeoutMs: storyTimeoutMs(),
    assetTimeoutMs: assetTimeoutMs(),
    artifactDirectory: screenshotArtifactDir() || null,
  };
}

async function writeRunManifest(results, failures = [], request = null) {
  const outputDir = screenshotArtifactDir();
  const destination = captureResultPath() || (outputDir ? path.join(outputDir, "capture-manifest.json") : "");
  if (!destination) return;
  await mkdir(path.dirname(destination), { recursive: true });
  const captures = manifestRows(results);
  const summary = {
    passed: results.length,
    failed: failures.length,
    skipped: 0,
    timedOut: failures.filter((failure) => failure.category === "capture-infrastructure" && /timed out/i.test(failure.message || "")).length,
  };
  await writeFile(
    destination,
    `${JSON.stringify({
      schemaVersion: 2,
      kind: "react-component-library-preview-capture",
      generatedAt: new Date().toISOString(),
      request,
      summary,
      captures,
      failures,
    }, null, 2)}\n`,
  );
  if (outputDir) {
    const reportLines = [
      "React Component Library Preview capture report",
      `passed=${summary.passed} failed=${summary.failed} skipped=${summary.skipped} timedOut=${summary.timedOut}`,
      `manifest=${path.basename(destination)}`,
      ...captures.map((capture) =>
        `${capture.artifactKind} ${capture.assetId} ${capture.storyId || capture.storyIds?.join(",") || "unknown-story"} ${capture.theme} ${capture.viewport.name} artifact=${capture.artifact || "none"} sources=${capture.sourceArtifacts.join(",") || "none"}`,
      ),
      ...failures.map((failure) =>
        `failure ${failure.category} stage=${failure.stage} retryable=${failure.retryable} ${failure.message}`,
      ),
    ];
    await writeFile(path.join(outputDir, "capture-report.txt"), `${reportLines.join("\n")}\n`);
  }
}

async function main() {
  const { chromium } = loadPlaywright();

  const browser = await chromium.launch({
    headless: true,
    executablePath: chromeExecutable(),
    args: ["--no-sandbox"],
  });

  try {
    const componentTargets = await previewableAssetTargets();
    if (componentTargets.length === 0) {
      throw new Error("catalog list returned no IDs");
    }

    const results = [];
    const failures = [];
    const request = captureRequest(componentTargets);
    for (const target of componentTargets) {
      // A preview route can load large dependency graphs and third-party
      // runtime modules. Reusing the host page across the whole catalog lets
      // retained iframe/document state degrade later assets, turning a valid
      // catalog item into a false missing-link or closed-context failure.
      // Keep each asset's host route isolated while still sharing the browser
      // process and the exact harness capture infrastructure.
      const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
      console.error(`[preview] start ${target.label}`);
      try {
        results.push({
          label: target.label,
          ...(await withAssetTimeout(
            () => assertAssetPreview(page, target.id, target),
            target.label,
          )),
        });
      } catch (error) {
        console.error(`[preview] failed ${target.label}`);
        const classification = classifyCaptureFailure(error);
        failures.push({
          componentID: target.id,
          label: target.label,
          stage: classification.stage,
          category: classification.category,
          retryable: classification.retryable,
          message: error instanceof Error ? error.message : String(error),
          details: error instanceof PreviewFailure ? compactDetails(error.details) : undefined,
        });
      } finally {
        console.error(`[preview] finish ${target.label}`);
        await page.close();
      }
    }

    if (failures.length > 0) {
      await writeRunManifest(results, failures, request);
      throw new Error(
        `preview failed for ${failures.length}/${componentTargets.length} catalog asset(s): ${JSON.stringify(failures)}`,
      );
    }

    await writeRunManifest(results, [], request);
    console.log(JSON.stringify({
      ok: true,
      checked: componentTargets.length,
      summary: { passed: results.length, failed: 0, skipped: 0, timedOut: 0 },
      results,
    }, null, 2));
    console.log("[REQ:SC-004] Preview sweep rendered every catalog asset story.");
  } catch (error) {
    fail(error instanceof Error ? error.message : String(error), {
      url: baseURL(),
    });
  } finally {
    await browser.close();
  }
}

await main();
