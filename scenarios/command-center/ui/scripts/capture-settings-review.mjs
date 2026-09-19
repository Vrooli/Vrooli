import playwright from "/home/matthalloran8/Vrooli/scenarios/prompt-manager/ui/node_modules/playwright-core/index.js";
import { mkdir } from "node:fs/promises";
const { chromium } = playwright;
const baseURL = process.env.COMMAND_CENTER_UI_URL || "http://127.0.0.1:24993";
const output = "/home/matthalloran8/.vrooli/plan-artifacts/command-center-settings-redesign";
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ executablePath: process.env.COMMAND_CENTER_CHROME || "/usr/bin/google-chrome", args: ["--no-sandbox"] });
try {
  for (const width of [390, 768]) {
    const page = await browser.newPage({ viewport: { width, height: 900 }, deviceScaleFactor: 1 });
    await page.goto(baseURL + "/settings", { waitUntil: "networkidle", timeout: 60000 });
    await page.screenshot({ path: output + "/settings-" + width + ".png", fullPage: true });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
    if (overflow) throw new Error("horizontal overflow at " + width + "px");
    await page.close();
  }
} finally { await browser.close(); }
console.log("Captured settings review screenshots at 390px and 768px with no horizontal overflow.");
