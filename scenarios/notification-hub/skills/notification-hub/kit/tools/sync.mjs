#!/usr/bin/env node
// Copies hub-charts.css and hub-charts.js into every template's kit slots.
//   node kit/tools/sync.mjs           rewrite templates whose kit copy is stale
//   node kit/tools/sync.mjs --check   exit 1 if any template is stale (no writes)
import { readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { TEMPLATES_DIR, kitSources, syncTemplate, templateNames } from "./kit.mjs";

const check = process.argv.includes("--check");
const kit = kitSources();
let stale = 0;
for (const name of templateNames()) {
  const file = join(TEMPLATES_DIR, `${name}.html`);
  const before = readFileSync(file, "utf8");
  const after = syncTemplate(before, kit);
  if (before === after) continue;
  stale++;
  if (check) console.error(`stale: templates/${name}.html (run node kit/tools/sync.mjs)`);
  else { writeFileSync(file, after); console.log(`synced: templates/${name}.html`); }
}
if (check && stale) process.exit(1);
if (!stale) console.log("all templates carry the current kit");
