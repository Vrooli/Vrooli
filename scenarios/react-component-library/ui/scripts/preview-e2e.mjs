#!/usr/bin/env node
import { createRequire } from "node:module";
import process from "node:process";

const DEFAULT_COMPONENT_ID = "bc29677f-b48c-440f-9f95-0fe2901f5f5e";
const DEFAULT_CHROME = "/usr/bin/google-chrome";

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

function assertNoKnownRuntimeErrors(logs) {
  const forbidden = [
    "react/jsx-runtime' does not provide an export named 'jsx'",
    'Dynamic require of "react" is not supported',
    "preview: render failed",
  ];
  const failures = logs.filter((line) => forbidden.some((needle) => line.includes(needle)));
  if (failures.length > 0) {
    throw new Error(`preview emitted runtime errors: ${failures.join(" | ")}`);
  }
}

async function main() {
  const { chromium } = loadPlaywright();
  const componentID = process.env.RCL_PREVIEW_COMPONENT_ID || DEFAULT_COMPONENT_ID;
  const url = `${baseURL()}/components/${componentID}`;
  const logs = [];
  const responses = [];

  const browser = await chromium.launch({
    headless: true,
    executablePath: chromeExecutable(),
    args: ["--no-sandbox"],
  });

  try {
    const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });
    page.on("console", (msg) => logs.push(`console:${msg.type()}:${msg.text()}`));
    page.on("pageerror", (err) => logs.push(`pageerror:${err.stack || err.message}`));
    page.on("requestfailed", (req) => {
      logs.push(`requestfailed:${req.url()}:${req.failure()?.errorText || "unknown"}`);
    });
    page.on("response", (res) => {
      const resURL = res.url();
      if (resURL.includes("/preview/") || res.status() >= 400) {
        responses.push(`${res.status()}:${resURL}`);
      }
    });

    await page.goto(url, { waitUntil: "domcontentloaded" });
    await page.locator('[data-testid="components-editor-preview-mode-button"]').click();

    const frameElement = page.locator('[data-testid="components-editor-preview-frame"]');
    await frameElement.waitFor({ state: "attached", timeout: 10_000 });

    const frameSrc = await frameElement.getAttribute("src");
    if (!frameSrc?.includes(`/preview/${componentID}/harness.html`)) {
      throw new Error(`preview iframe did not point at harness: ${frameSrc || "<empty>"}`);
    }

    await page.waitForFunction(() => {
      const badge = document.querySelector('[data-testid="components-editor-preview-badge"]');
      const error = document.querySelector('[data-testid="components-editor-preview-error"]');
      return Boolean(badge) || Boolean(error);
    }, null, { timeout: 10_000 });

    const previewFrame = page.frames().find((frame) => frame.url().includes(`/preview/${componentID}/`));
    if (!previewFrame) {
      throw new Error("preview frame was not present after clicking Preview");
    }
    if (previewFrame.url().includes("/components/")) {
      throw new Error(`preview frame recursively loaded the app route: ${previewFrame.url()}`);
    }

    await previewFrame.locator("#root > *").first().waitFor({ state: "attached", timeout: 10_000 });
    const rootHTML = await previewFrame.locator("#root").innerHTML();
    const iframeError = await previewFrame.locator("#preview-error").innerText();
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
    if (iframeError.trim() !== "") {
      throw new Error(`iframe preview error rendered: ${iframeError}`);
    }
    if (!badge.includes("Rendered")) {
      throw new Error(`preview did not reach rendered state, badge=${badge || "<empty>"}`);
    }
    if (!rootHTML.includes("inline-flex")) {
      throw new Error(`preview root did not contain the rendered StatusBadge DOM: ${rootHTML}`);
    }

    assertNoKnownRuntimeErrors(logs);

    console.log(JSON.stringify({
      ok: true,
      componentID,
      frameSrc,
      frameURL: previewFrame.url(),
      badge,
      previewResponses: responses,
    }, null, 2));
  } catch (error) {
    fail(error instanceof Error ? error.message : String(error), {
      componentID,
      url,
      logs,
      previewResponses: responses,
    });
  } finally {
    await browser.close();
  }
}

await main();
