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

function safeArtifactPart(value) {
  return value.replace(/[^a-zA-Z0-9._-]+/g, "-");
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
  return Number.isFinite(requested) && requested > 0 ? requested : 120_000;
}

async function withAssetTimeout(operation, label) {
  let timer;
  const timeout = new Promise((_, reject) => {
    timer = setTimeout(() => {
      reject(new Error(`asset preview timed out after ${assetTimeoutMs()}ms: ${label}`));
    }, assetTimeoutMs());
  });
  try {
    return await Promise.race([operation(), timeout]);
  } finally {
    clearTimeout(timer);
  }
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
  const stories = requestedStory
    ? [requestedStory]
    : storyIDs.length > 0
      ? [...new Set(storyIDs)]
      : [undefined];

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
          for (let index = 0; index < sources.length; index += 1) {
            const source = sources[index];
            for (const storyID of stories) {
              const kitSource = new URL(source);
              kitSource.searchParams.set("kit", kit);
              // The host uses the frame contract to compose catalog surfaces
              // inside the editor. Isolated screenshots need the component's
              // own specimen only; retaining that frame can leave a capture
              // waiting on a parent-owned region that does not exist here.
              kitSource.searchParams.set("frame", "off");
              // A focused capture may target any declared story without
              // changing the catalog route or the selected story in the host
              // editor. This keeps visual ground truth available for states
              // such as an exiting/hidden Presence boundary.
              if (storyID) kitSource.searchParams.set("story", storyID);
              await capturePage.goto(kitSource.toString(), { waitUntil: "domcontentloaded" });
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
                  // artifact is #root, so responsive validation must use the
                  // same boundary.
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
              const bytes = await capturePage
                .locator("#root")
                .screenshot({ animations: "disabled" });
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
                index,
                viewport: viewport.name,
                viewportWidth: visual.viewportWidth,
                scrollWidth: visual.scrollWidth,
                background: visual.background,
                bytes: bytes.length,
                hash,
                width: visual.width,
                height: visual.height,
                interactiveCount: visual.interactiveCount,
                artifact: outputDir ? artifact : undefined,
              });
            }
          }
        }
      }
    }
  } finally {
    await Promise.all(captureContexts.map((context) => context.close()));
  }

  for (const light of captures.light || []) {
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
        stories.set(`${contract.version}:${story.id}`, story.expect || []);
      }
    } catch {
      throw new Error("indexed story contract has invalid storiesJson");
    }
  }
  return stories;
}

async function assertStoryExpectations(frame, name, expectations) {
  for (const expectation of expectations) {
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
      await assertStoryExpectations(previewFrame, story, declared);
      frameResults.push({ url: previewFrame.url(), story, expectationCount: declared.length });
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
    const storyIDs = [...expectations.keys()]
      .filter((key) => !renderedVersion || key.startsWith(`${renderedVersion}:`))
      .map((key) => key.slice(key.indexOf(":") + 1));
    const screenshots = await captureThemeTier(
      page,
      frameElements,
      componentID,
      storyIDs,
      frameResults[0]?.url || frameSrc,
    );
    assertNoKnownRuntimeErrors(logs);
    return {
      ok: true,
      componentID,
      frameSrc,
      frames: frameResults,
      screenshots,
      badge,
      previewResponses: responses,
    };
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
        failures.push({
          componentID: target.id,
          label: target.label,
          message: error instanceof Error ? error.message : String(error),
          details: error instanceof PreviewFailure ? compactDetails(error.details) : undefined,
        });
      } finally {
        console.error(`[preview] finish ${target.label}`);
        await page.close();
      }
    }

    if (failures.length > 0) {
      throw new Error(
        `preview failed for ${failures.length}/${componentTargets.length} catalog asset(s): ${JSON.stringify(failures)}`,
      );
    }

    console.log(JSON.stringify({ ok: true, checked: componentTargets.length, results }, null, 2));
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
