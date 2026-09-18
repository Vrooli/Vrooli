import playwright from "/home/matthalloran8/Vrooli/scenarios/prompt-manager/ui/node_modules/playwright-core/index.js";
import pngjs from "/home/matthalloran8/Vrooli/scenarios/prompt-manager/ui/node_modules/pngjs/lib/png.js";
import { mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import { join } from "node:path";

const root = new URL("../", import.meta.url).pathname;
const baselineRoot = join(root, "baseline");
const roomsRoot = new URL("../../config/rooms/", import.meta.url);
const timestamps = [0, 250, 1000, 2500, 5000, 10000, 14000];
const rooms = (await readdir(roomsRoot)).filter((file) => file.endsWith(".json")).map((file) => file.slice(0, -5));
const baseURL = process.env.COMMAND_CENTER_UI_URL || ("http://127.0.0.1:" + (process.env.UI_PORT || "21235"));
const accept = process.env.VISUAL_BASELINE_ACCEPT === "1";
const { chromium } = playwright;
const { PNG } = pngjs;
const browser = await chromium.launch({ executablePath: process.env.COMMAND_CENTER_CHROME || "/usr/bin/google-chrome", args: ["--no-sandbox", "--ignore-gpu-blocklist", "--use-gl=angle", "--use-angle=gl-egl"] });
const page = await browser.newPage({ viewport: { width: 1600, height: 900 }, deviceScaleFactor: 1 });
const manifest = { seed: "command-center-milestone-1", timestampsMs: timestamps, rooms, viewport: { width: 1600, height: 900 } };
const differences = [];
try {
  for (const room of rooms) {
    for (const timestamp of timestamps) {
      const name = join(baselineRoot, room, String(timestamp) + ".png");
      await page.goto(baseURL + "/" + room + "?visualDiff=1&visualDiffAt=" + timestamp + "&samples=hide&seed=" + manifest.seed, { waitUntil: "networkidle", timeout: 60000 });
      await page.waitForTimeout(250);
      const canvas = page.locator("canvas").first();
      await canvas.waitFor({ state: "visible", timeout: 60000 });
      const actual = await canvas.screenshot();
      if (accept) { await mkdir(join(baselineRoot, room), { recursive: true }); await writeFile(name, actual); continue; }
      let expected;
      try { expected = await readFile(name); } catch { differences.push(room + "/" + timestamp + ": missing baseline"); continue; }
      const a = PNG.sync.read(actual); const e = PNG.sync.read(expected);
      if (a.width !== e.width || a.height !== e.height || a.data.length !== e.data.length) { differences.push(room + "/" + timestamp + ": dimensions differ"); continue; }
      let differentPixels = 0;
      for (let i = 0; i < a.data.length; i += 4) if (a.data[i] !== e.data[i] || a.data[i + 1] !== e.data[i + 1] || a.data[i + 2] !== e.data[i + 2]) differentPixels++;
      if (differentPixels) differences.push(room + "/" + timestamp + ": " + differentPixels + " pixels differ");
    }
  }
  if (accept) await writeFile(join(baselineRoot, "baseline.json"), JSON.stringify(manifest, null, 2) + "\n");
} finally { await browser.close(); }
if (differences.length) { console.error(differences.join("\n")); process.exit(1); }
console.log((accept ? "Captured" : "Verified") + " zero-diff baseline for " + rooms.length + " rooms × " + timestamps.length + " timestamps.");
