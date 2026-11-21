#!/usr/bin/env node
// Renders a kit template with one data object: validates the data against the
// template's schema, then embeds it in the page's hub-data element.
//   node kit/tools/render.mjs <template> <data.json> [--out page.html]
//   node kit/tools/render.mjs <template> --sample [--out page.html]
// <template> is a name (usage-mode) or a path to a template .html file.
import { readFileSync, writeFileSync } from "node:fs";
import { basename } from "node:path";
import { loadSample, loadSchema, loadTemplate, render, templateNames, validate } from "./kit.mjs";

const args = process.argv.slice(2);
const outAt = args.indexOf("--out");
const out = outAt >= 0 ? args[outAt + 1] : null;
const positional = args.filter((a, i) => !a.startsWith("--") && (outAt < 0 || i !== outAt + 1));
const [templateArg, dataPath] = positional;
if (!templateArg || (!dataPath && !args.includes("--sample"))) {
  console.error(`usage: render.mjs <template> <data.json> [--out page.html]\ntemplates: ${templateNames().join(", ")}`);
  process.exit(2);
}
const name = basename(templateArg).replace(/\.html$/, "");
if (!templateNames().includes(name)) { console.error(`unknown template "${name}"; known: ${templateNames().join(", ")}`); process.exit(2); }
const data = dataPath ? JSON.parse(readFileSync(dataPath, "utf8")) : loadSample(name);
const errors = validate(loadSchema(name), data);
if (errors.length) { console.error(`data does not match schemas/${name}.schema.json:\n  ${errors.join("\n  ")}`); process.exit(1); }
const page = render(templateArg.endsWith(".html") ? readFileSync(templateArg, "utf8") : loadTemplate(name), data);
if (out) writeFileSync(out, page); else process.stdout.write(page);
