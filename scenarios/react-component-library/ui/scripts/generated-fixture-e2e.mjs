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

function isPainted(value) {
  return Boolean(value) && value !== "transparent" && value !== "rgba(0, 0, 0, 0)";
}

function fail(message, details = {}) {
  console.error(JSON.stringify({ ok: false, message, ...details }, null, 2));
  process.exitCode = 1;
}

const url = (process.argv[2] || "").replace(/\/+$/, "");
if (!url) {
  fail("generated fixture URL is required");
} else {
  const { chromium } = loadPlaywright();
  const browser = await chromium.launch({
    headless: true,
    executablePath: process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE || "/usr/bin/google-chrome",
    args: ["--no-sandbox"],
  });
  try {
    const page = await browser.newPage({ viewport: { width: 1254, height: 720 } });
    page.setDefaultTimeout(30_000);
    page.setDefaultNavigationTimeout(90_000);
    await page.goto(url, { waitUntil: "networkidle" });
    const control = page.locator('[data-rcl-control="true"]').first();
    await control.waitFor({ state: "visible" });
    const computed = await control.evaluate((element) => {
      const style = getComputedStyle(element);
      return {
        background: style.backgroundColor,
        border: `${style.borderTopWidth} ${style.borderTopStyle} ${style.borderTopColor}`,
        borderColor: style.borderTopColor,
        borderStyle: style.borderTopStyle,
        borderWidth: style.borderTopWidth,
        foreground: style.color,
        tag: element.tagName.toLowerCase(),
      };
    });
    if (
      !isPainted(computed.background) ||
      !isPainted(computed.borderColor) ||
      computed.borderStyle === "none" ||
      computed.borderWidth === "0px" ||
      !isPainted(computed.foreground)
    ) {
      fail("adopted control did not resolve its required visual styles", { computed });
    } else {
      console.log(JSON.stringify({ ok: true, computed }, null, 2));
    }
  } catch (error) {
    fail(error instanceof Error ? error.message : String(error));
  } finally {
    await browser.close();
  }
}
