#!/usr/bin/env node
import { createRequire } from "node:module";
import process from "node:process";
import { createHash } from "node:crypto";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";

const DEFAULT_CHROME = "/usr/bin/google-chrome";
const COMPONENT_LINK_SELECTOR = '[data-testid^="sidebar-component-"]';

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
    .replace(/data:text\/javascript;base64,[A-Za-z0-9+/=]+/g, "data:text/javascript;base64,<bundle>")
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

async function selectResolvedTheme(page, theme) {
  for (let attempts = 0; attempts < 3; attempts += 1) {
    const resolved = await page.locator("html").getAttribute("data-resolved-theme");
    if (resolved === theme) return;
    await page.locator('[data-testid="theme-toggle"]').click();
    await page.waitForTimeout(100);
  }
  throw new Error(`app theme did not resolve to ${theme}`);
}

async function captureThemeTier(page, frameElements, componentID) {
  const captures = {};
  const outputDir = screenshotArtifactDir();
  if (outputDir) await mkdir(outputDir, { recursive: true });

  for (const theme of ["light", "dark"]) {
    await selectResolvedTheme(page, theme);
    await page.waitForTimeout(100);
    captures[theme] = [];
    const count = await frameElements.count();
    for (let index = 0; index < count; index += 1) {
      const iframe = frameElements.nth(index);
      const handle = await iframe.elementHandle();
      const frame = await handle?.contentFrame();
      if (!frame) throw new Error(`preview iframe ${index} had no content frame for ${theme}`);
      await frame.waitForFunction((expected) => document.documentElement.dataset.resolvedTheme === expected, theme, {
        timeout: 5_000,
      });
      const background = await frame.locator("body").evaluate((element) => getComputedStyle(element).backgroundColor);
      if (!background || background === "rgba(0, 0, 0, 0)" || background === "transparent") {
        throw new Error(`preview iframe ${index} has a blank ${theme} background`);
      }
      const bytes = await iframe.screenshot();
      if (bytes.length < 256) throw new Error(`preview iframe ${index} ${theme} screenshot is unexpectedly small`);
      const hash = screenshotHash(bytes);
      const example = new URL((await iframe.getAttribute("src")) || "http://invalid/").searchParams.get("example") || "default";
      const artifact = `${safeArtifactPart(componentID)}--${safeArtifactPart(example)}--${index}--${theme}.png`;
      if (outputDir) await writeFile(path.join(outputDir, artifact), bytes);
      captures[theme].push({ example, background, bytes: bytes.length, hash, artifact: outputDir ? artifact : undefined });
    }
  }

  if (captures.light.length !== captures.dark.length) throw new Error("theme screenshot frame count changed");
  for (let index = 0; index < captures.light.length; index += 1) {
    const light = captures.light[index];
    const dark = captures.dark[index];
    if (light.background === dark.background) {
      throw new Error(`preview iframe ${index} background did not change between light and dark`);
    }
    if (light.hash === dark.hash) {
      throw new Error(`preview iframe ${index} screenshot did not change between light and dark`);
    }
  }
  return captures;
}

async function componentTargetsFromSidebar(page) {
  if (process.env.RCL_PREVIEW_COMPONENT_ID) {
    return [{
      id: process.env.RCL_PREVIEW_COMPONENT_ID,
      label: process.env.RCL_PREVIEW_COMPONENT_ID,
    }];
  }
  await page.goto(`${baseURL()}/components`, { waitUntil: "domcontentloaded" });
  await page.locator('[data-testid="app-shell"]').waitFor({ state: "visible", timeout: 15_000 });
  const sidebarError = await page
    .locator('[data-testid="sidebar-component-list-error"]')
    .innerText({ timeout: 250 })
    .catch(() => "");
  if (sidebarError.trim() !== "") {
    throw new Error(`component list failed to load: ${sidebarError}`);
  }
  await page.locator(COMPONENT_LINK_SELECTOR).first().waitFor({ state: "attached", timeout: 15_000 });
  const targets = await page.locator(COMPONENT_LINK_SELECTOR).evaluateAll((links) => (
    links
      .map((link) => {
        const href = link.getAttribute("href") || "";
        const id = href.split("/components/")[1] || "";
        return {
          id: decodeURIComponent(id.split(/[?#]/)[0] || ""),
          label: (link.textContent || "").trim(),
        };
      })
      .filter((target) => target.id)
  ));
  const byID = new Map();
  for (const target of targets) {
    byID.set(target.id, target);
  }
  return [...byID.values()].sort((a, b) => a.label.localeCompare(b.label));
}

async function assertComponentPreview(page, componentID) {
  const url = `${baseURL()}/components/${componentID}`;
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
    await page.goto(url, { waitUntil: "domcontentloaded" });
    await page.locator('[data-testid="components-editor-panel"]').waitFor({ state: "visible", timeout: 15_000 });
    // Desktop shows Preview, Code, and Info together. The segmented switcher
    // is intentionally mobile-only, so only use it when it is actually shown.
    const previewModeButton = page.locator('[data-testid="components-editor-preview-mode-button"]');
    if (await previewModeButton.isVisible()) {
      await previewModeButton.click();
    }

    const frameElements = page.locator('[data-testid="components-editor-preview-frame"]');
    await frameElements.first().waitFor({ state: "attached", timeout: 10_000 });

    const frameSrc = await frameElements.first().getAttribute("src");
    if (!frameSrc?.includes(`/preview/${componentID}/harness.html`)) {
      throw new Error(`preview iframe did not point at harness: ${frameSrc || "<empty>"}`);
    }

    await page.waitForFunction(() => {
      const badge = document.querySelector('[data-testid="components-editor-preview-badge"]');
      const error = document.querySelector('[data-testid="components-editor-preview-error"]');
      return Boolean(badge) || Boolean(error);
    }, null, { timeout: 10_000 });

    const frameCount = await frameElements.count();
    const previewFrames = page.frames().filter((frame) => frame.url().includes(`/preview/${componentID}/`));
    if (previewFrames.length < frameCount) {
      throw new Error(`expected ${frameCount} preview frame(s), found ${previewFrames.length}`);
    }
    const frameResults = [];
    for (const previewFrame of previewFrames) {
      if (previewFrame.url().includes("/components/")) {
        throw new Error(`preview frame recursively loaded the app route: ${previewFrame.url()}`);
      }
      await previewFrame.locator("#root > *").first().waitFor({ state: "attached", timeout: 10_000 });
      const rootHTML = await previewFrame.locator("#root").innerHTML();
      const iframeError = await previewFrame.locator("#preview-error").innerText();
      if (iframeError.trim() !== "") {
        throw new Error(`iframe preview error rendered: ${iframeError}`);
      }
      if (rootHTML.trim() === "") {
        throw new Error("preview root was empty after rendered state");
      }
      frameResults.push({
        url: previewFrame.url(),
        example: new URL(previewFrame.url()).searchParams.get("example") || "__default__",
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

    const screenshots = await captureThemeTier(page, frameElements, componentID);
    assertNoKnownRuntimeErrors(logs);
    return { ok: true, componentID, frameSrc, frames: frameResults, screenshots, badge, previewResponses: responses };
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
    const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
    const componentTargets = await componentTargetsFromSidebar(page);
    if (componentTargets.length === 0) {
      throw new Error("component list returned no IDs");
    }

    const results = [];
    const failures = [];
    for (const target of componentTargets) {
      try {
        results.push({
          label: target.label,
          ...(await assertComponentPreview(page, target.id)),
        });
      } catch (error) {
        failures.push({
          componentID: target.id,
          label: target.label,
          message: error instanceof Error ? error.message : String(error),
          details: error instanceof PreviewFailure ? compactDetails(error.details) : undefined,
        });
      }
    }

    if (failures.length > 0) {
      throw new Error(`preview failed for ${failures.length}/${componentTargets.length} component(s): ${JSON.stringify(failures)}`);
    }

    console.log(JSON.stringify({ ok: true, checked: componentTargets.length, results }, null, 2));
  } catch (error) {
    fail(error instanceof Error ? error.message : String(error), {
      url: baseURL(),
    });
  } finally {
    await browser.close();
  }
}

await main();
