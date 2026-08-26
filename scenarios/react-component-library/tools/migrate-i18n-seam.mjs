import { readdir, readFile, writeFile } from "node:fs/promises";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const libraryRoot = join(fileURLToPath(new URL("..", import.meta.url)), "library");
const callPattern = /useStrings\(\s*(["'])([^"']+)\1\s*,\s*(["'])([\s\S]*?)\3\s*,?\s*\)/g;
const stringPattern = /useStrings\(/g;

async function filesUnder(root) {
  const entries = await readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const path = join(root, entry.name);
    if (entry.isDirectory()) files.push(...await filesUnder(path));
    else if (/\.(?:ts|tsx)$/.test(entry.name)) files.push(path);
  }
  return files;
}

function slug(value) {
  return value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 48) || "value";
}

function namedKey(key, fallback, used) {
  const parts = key.split(".");
  if (/^\d+$/.test(parts.at(-1) ?? "")) {
    parts.pop();
    const role = parts.at(-1);
    if (["text", "label", "title", "description", "placeholder", "alt", "aria-label"].includes(role)) parts.pop();
    parts.push(slug(fallback));
  }
  let candidate = parts.join(".");
  if (used.has(candidate) && used.get(candidate) !== fallback) candidate = `${candidate}.${slug(fallback)}`;
  used.set(candidate, fallback);
  return candidate;
}

function versionDirectory(path) {
  const parts = relative(libraryRoot, path).split("/");
  if (parts.length < 5 || parts[2] !== "versions") return null;
  return { directory: join(libraryRoot, parts.slice(0, 4).join("/")), kind: parts[0], name: parts[1], version: parts[3] };
}

const files = await filesUnder(libraryRoot);
const byVersion = new Map();
let changedFiles = 0;
let migratedCalls = 0;
for (const file of files) {
  if (file.endsWith("/useLocale.ts")) continue;
  let source = await readFile(file, "utf8");
  const matches = [...source.matchAll(callPattern)];
  if (matches.length === 0 && !stringPattern.test(source)) continue;
  stringPattern.lastIndex = 0;
  const info = versionDirectory(file);
  if (!info) continue;
  const used = byVersion.get(info.directory)?.used ?? new Map();
  const definitions = byVersion.get(info.directory)?.definitions ?? [];
  source = source.replace(callPattern, (full, keyQuote, oldKey, fallbackQuote, fallback) => {
    const key = namedKey(oldKey, fallback, used);
    definitions.push({ key, fallback });
    migratedCalls += 1;
    return full.replace(`${keyQuote}${oldKey}${keyQuote}`, `${keyQuote}${key}${keyQuote}`);
  });
  source = source.replaceAll("translate(", "useStrings(").replace(/\btranslate\b/g, "useStrings");
  if (source !== await readFile(file, "utf8")) {
    await writeFile(file, source);
    changedFiles += 1;
  }
  byVersion.set(info.directory, { ...info, used, definitions });
}

for (const item of byVersion.values()) {
  const unique = new Map(item.definitions.map((entry) => [entry.key, entry.fallback]));
  const entries = [...unique.entries()].sort(([a], [b]) => a.localeCompare(b));
  const body = [
    `import { defineStrings } from "@vrooli/react-component-library/useLocale/1.0.1";`,
    "",
    `export const ${item.name.replace(/[^A-Za-z0-9_$]/g, "") || "component"}Strings = defineStrings(`,
    `  "react-component-library:${item.name}",`,
    "  {",
    ...entries.map(([key, fallback]) => `    ${JSON.stringify(key)}: ${JSON.stringify(fallback)},`),
    "  },",
    ");",
    "",
  ].join("\n");
  await writeFile(join(item.directory, `${item.name}.strings.ts`), body);
}

console.log(JSON.stringify({ changedFiles, migratedCalls, stringModules: byVersion.size }));
