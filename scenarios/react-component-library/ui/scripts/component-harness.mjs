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
    await page.goto(url, { waitUntil: "domcontentloaded", timeout: 15_000 });
    await page.waitForFunction(
      () => Boolean(document.querySelector("#rcl-story-result")?.textContent?.trim()),
      undefined,
      { timeout: 15_000 },
    );
    const result = await page.locator("#rcl-story-result").textContent();
    if (!result) {
      throw new Error("harness completed without a story result");
    }
    process.stdout.write(result);
  } finally {
    await browser.close();
  }
}
