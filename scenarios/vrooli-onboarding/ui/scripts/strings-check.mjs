import { existsSync, readdirSync, readFileSync } from "node:fs";
import { dirname, join, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const uiRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const sourceRoot = join(uiRoot, "src");

function productionFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return productionFiles(path);
    if (!/\.(tsx?|jsx?)$/.test(entry.name) || /\.test\.[jt]sx?$/.test(entry.name)) return [];
    if (path.endsWith("/src/i18n/index.ts")) return [];
    return [path];
  });
}

function findHardcodedStrings() {
  if (!existsSync(sourceRoot)) return [`Missing source directory: ${sourceRoot}`];
  const findings = [];
  for (const file of productionFiles(sourceRoot)) {
    const source = readFileSync(file, "utf8");
    const location = relative(uiRoot, file);
    const jsxText = />\s*([A-Za-z][^<{\n]{2,})\s*</g;
    for (const match of source.matchAll(jsxText)) {
      const value = match[1].trim();
      if (value && !/^(V|[A-Z]\s*)$/.test(value)) findings.push(`${location}: JSX text "${value}"`);
    }
    const attributes = /\b(?:aria-label|aria-description|placeholder|title|label|alt)="([^"{}]+)"/g;
    for (const match of source.matchAll(attributes)) {
      const value = match[1].trim();
      if (value && !/^[a-z][a-z0-9-]*$/.test(value)) findings.push(`${location}: attribute "${value}"`);
    }
  }
  return findings;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const findings = findHardcodedStrings();
  if (findings.length > 0) {
    console.error(`Found ${findings.length} hardcoded production UI string(s):`);
    for (const finding of findings) console.error(`- ${finding}`);
    process.exitCode = 1;
  }
}

export { findHardcodedStrings };
