#!/usr/bin/env node

import { createRequire } from "node:module";
import process from "node:process";

const require = createRequire(import.meta.url);

function loadPlaywright() {
  try {
    return require("playwright");
  } catch {
    return require("../../../../packages/api-base/node_modules/playwright");
  }
}

const url = process.argv[2];
if (!url) {
  console.error("component harness URL is required");
  process.exitCode = 2;
} else {
  const { chromium } = loadPlaywright();
  const executablePath = process.env.RCL_CHROME_BIN || "/usr/bin/google-chrome";
  const browser = await chromium.launch({
    headless: true,
    executablePath,
    args: ["--no-sandbox"],
  });

  try {
    const page = await browser.newPage();
    const consoleErrors = [];
    const pageErrors = [];
    const failedRequests = [];
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
    page.on("pageerror", (error) => {
      pageErrors.push(error && error.stack ? error.stack : String(error));
    });
    page.on("requestfailed", (request) => {
      failedRequests.push(`${request.method()} ${request.url()} - ${request.failure()?.errorText || "request failed"}`);
    });
    await page.goto(url, { waitUntil: "domcontentloaded", timeout: 15_000 });
    try {
      await page.waitForFunction(
        () => Boolean(document.querySelector("#rcl-story-result")?.textContent?.trim()),
        undefined,
        { timeout: 15_000 },
      );
    } catch (error) {
      const previewError = await page.locator("#preview-error").textContent().catch(() => "");
      const state = await page.locator("#root").getAttribute("data-experience-state").catch(() => "");
      const diagnostics = [
        `state=${state || "unknown"}`,
        previewError ? `preview-error=${previewError.trim()}` : "",
        pageErrors.length ? `page-errors=${pageErrors.join(" | ")}` : "",
        consoleErrors.length ? `console-errors=${consoleErrors.join(" | ")}` : "",
        failedRequests.length ? `failed-requests=${failedRequests.join(" | ")}` : "",
      ].filter(Boolean).join("; ");
      throw new Error(`${error.message}${diagnostics ? ` (${diagnostics})` : ""}`);
    }
    const result = await page.locator("#rcl-story-result").textContent();
    if (!result) {
      throw new Error("harness completed without a story result");
    }
    process.stdout.write(result);
  } finally {
    await browser.close();
  }
}
