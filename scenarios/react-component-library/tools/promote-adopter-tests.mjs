import { createHash } from "node:crypto";
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { execFileSync } from "node:child_process";
import { dirname, extname, relative, resolve } from "node:path";

const repoRoot = resolve(new URL("../../..", import.meta.url).pathname);
const libraryRoot = resolve(repoRoot, "scenarios/react-component-library/library");
const packageManifest = JSON.parse(
  readFileSync(resolve(repoRoot, "packages/react-component-library/package.json"), "utf8"),
);

const testFiles = execFileSync(
  "rg",
  ["--files", "scenarios"],
  { cwd: repoRoot, maxBuffer: 20 * 1024 * 1024 },
)
  .toString()
  .trim()
  .split("\n")
  .filter(Boolean)
  .filter((file) =>
    !file.startsWith("scenarios/react-component-library/") &&
    /\.(test|spec)\.[jt]sx?$/.test(file),
  )
  .map((file) => resolve(repoRoot, file));

const directImport = /(?:^|\n)\s*import(?:[^\n]*?from\s*)?["']@vrooli\/react-component-library\/([^"'\s)]+)["']/m;
const groups = new Map();
const ignored = [];
let directCandidateCount = 0;

for (const file of testFiles) {
  const source = readFileSync(file, "utf8");
  const match = source.match(directImport);
  if (!match) continue;
  directCandidateCount += 1;
  if (file.endsWith("empty-state-dialog.test.tsx")) {
    ignored.push({ file: relative(repoRoot, file), reason: "mixed adopter-local Dialog test" });
    continue;
  }
  const normalized = source.replace(/\/\/.*$/gm, "").replace(/\s+/g, " ").trim();
  const hash = createHash("sha256").update(normalized).digest("hex");
  if (!groups.has(hash)) groups.set(hash, { hash, source: file, specifier: match[1], copies: [] });
  groups.get(hash).copies.push(file);
}

function sourceForSpecifier(specifier) {
  const entry = packageManifest.exports?.[`./${specifier}`]?.import;
  if (typeof entry !== "string") throw new Error(`No package export for ${specifier}`);
  const relativeSource = entry.replace(/^\.\/dist\//, "").replace(/\.js$/, "");
  for (const extension of [".tsx", ".ts", ".jsx", ".js"]) {
    const candidate = resolve(libraryRoot, `${relativeSource}${extension}`);
    if (existsSync(candidate)) return candidate;
  }
  throw new Error(`No library source for ${specifier}: ${relativeSource}`);
}

function rewriteSource(source, destination, componentSource) {
  const packagePath = `@vrooli/react-component-library/${source.match(directImport)[1]}`;
  const localImport = relative(dirname(destination), componentSource).replaceAll("\\", "/");
  const localSpecifier = localImport.startsWith(".") ? localImport : `./${localImport}`;
  let rewritten = source.replaceAll(packagePath, localSpecifier);
  rewritten = rewritten.replace(
    /from\s*(["'])(\.{1,2}\/[^"']*test-utils(?:\/[^"']*)?)\1/g,
    `from "../../../../../ui/src/test-utils"`,
  );
  // This test exercised the package's own mermaid cache after the import
  // migration. Keep the helper paired with the promoted library source.
  rewritten = rewritten.replace(
    /from\s*(["'])\.\/useMermaidSvg\1/g,
    `from "./useMermaidSvg"`,
  );
  return rewritten;
}

const promoted = [];
for (const group of groups.values()) {
  const componentSource = sourceForSpecifier(group.specifier);
  const versionDir = dirname(componentSource);
  const baseName = group.source.split("/").at(-1);
  const candidate = resolve(versionDir, baseName);
  const destination = existsSync(candidate)
    ? resolve(versionDir, `${baseName.replace(/\.(test|spec)\./, `.${group.hash.slice(0, 8)}.$1.`)}`)
    : candidate;
  mkdirSync(versionDir, { recursive: true });
  writeFileSync(destination, rewriteSource(readFileSync(group.source, "utf8"), destination, componentSource));
  promoted.push({
    hash: group.hash,
    source: relative(repoRoot, group.source),
    destination: relative(repoRoot, destination),
    component: relative(repoRoot, componentSource),
    specifier: group.specifier,
    copies: group.copies.map((file) => relative(repoRoot, file)),
  });
}

const evidencePath = resolve(
  repoRoot,
  "scenarios/react-component-library/docs/evidence/2026-08-25-promoted-tests.json",
);

if (process.argv.includes("--reconcile")) {
  const current = JSON.parse(readFileSync(evidencePath, "utf8"));
  const seen = new Set();
  const surviving = current.promoted.filter((item) => {
    if (!existsSync(resolve(repoRoot, item.destination))) return false;
    const source = readFileSync(resolve(repoRoot, item.destination), "utf8");
    const normalized = source.replace(/\/\/.*$/gm, "").replace(/\s+/g, " ").trim();
    const hash = createHash("sha256").update(normalized).digest("hex");
    if (seen.has(hash)) return false;
    seen.add(hash);
    return true;
  });
  writeFileSync(
    evidencePath,
    `${JSON.stringify({
      ...current,
      generatedAt: new Date().toISOString(),
      candidateCount: current.candidateCount - current.ignored.length,
      promoted: surviving,
      uniquePromotedCount: surviving.length,
    }, null, 2)}\n`,
  );
  console.log(JSON.stringify({ candidateCount: current.candidateCount - current.ignored.length, promoted: surviving.length, mode: "reconcile" }, null, 2));
  process.exit(0);
}

writeFileSync(
  evidencePath,
  `${JSON.stringify({ generatedAt: new Date().toISOString(), candidateCount: directCandidateCount, promoted, uniquePromotedCount: promoted.length, ignored }, null, 2)}\n`,
);
console.log(JSON.stringify({ candidateCount: directCandidateCount, promoted: promoted.length, ignored }, null, 2));
