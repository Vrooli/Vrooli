import { readFile, readdir } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { auditThemeContrast } from "./theme-contrast-core.mjs";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const configDir = join(scriptDir, "../../config");
async function json(path) { return JSON.parse(await readFile(path, "utf8")); }
async function entries(dir) {
  return Promise.all((await readdir(dir)).filter((file) => file.endsWith(".json")).map((file) => json(join(dir, file))));
}

const [board, rooms, themes] = await Promise.all([
  json(join(configDir, "board.json")),
  entries(join(configDir, "rooms")),
  entries(join(configDir, "themes")),
]);
const results = auditThemeContrast({ board, rooms, themes });
for (const result of results) {
  const weakest = result.pairs.reduce((minimum, pair) => pair.ratio < minimum.ratio ? pair : minimum);
  console.log(`${result.theme}: minimum ${result.minimum}:1 (${weakest.role} on ${weakest.surface}) across ${result.pairs.length} text/surface pairs`);
  for (const pair of result.pairs) console.log(`  ${pair.passes ? "PASS" : "FAIL"} ${pair.ratio}:1 ${pair.role} on ${pair.surface}`);
}
const failures = results.flatMap((result) => result.failures.map((pair) => `${result.theme}: ${pair.ratio}:1 ${pair.role} on ${pair.surface}`));
if (failures.length) {
  console.error(`Contrast below 4.5:1:\n${failures.join("\n")}`);
  process.exitCode = 1;
} else {
  console.log(`All ${results.reduce((count, result) => count + result.pairs.length, 0)} text/surface pairs meet 4.5:1.`);
}
