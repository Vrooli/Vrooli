// @vitest-environment node
import { readdirSync, readFileSync } from "node:fs";
import { join, relative, sep } from "node:path";
import { describe, expect, it } from "vitest";

/**
 * `xlsx` and `mermaid` are optional peers: a host that cannot install them must
 * still install, bundle and run the package. That holds only while nothing a
 * host reaches from the main entry imports them.
 */
const manifest = JSON.parse(readFileSync("package.json", "utf8")) as {
  dependencies: Record<string, string>;
  peerDependencies: Record<string, string>;
  peerDependenciesMeta: Record<string, { optional?: boolean }>;
};

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return entry.name === "__tests__" || entry.name === "test-utils" ? [] : sourceFiles(path);
    return /\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name) ? [path] : [];
  });
}

function importers(specifier: string): string[] {
  const pattern = new RegExp(`(?:from\\s+|import\\s*\\(\\s*|import\\s+)["']${specifier}(?:/[^"']*)?["']`);
  const code = (file: string) => readFileSync(file, "utf8").replace(/\/\*[\s\S]*?\*\//g, "").replace(/^\s*\/\/.*$/gm, "");
  return sourceFiles("src").filter((file) => pattern.test(code(file))).map((file) => relative("src", file).split(sep).join("/"));
}

describe("optional peer dependencies", () => {
  it.each(["xlsx", "mermaid"])("declares %s as an optional peer, never a dependency", (name) => {
    expect(manifest.peerDependencies[name]).toBeDefined();
    expect(manifest.peerDependenciesMeta[name]?.optional).toBe(true);
    expect(manifest.dependencies[name]).toBeUndefined();
  });

  it("imports xlsx only from the spreadsheet subpath", () => {
    const files = importers("xlsx");
    expect(files.length).toBeGreaterThan(0);
    expect(files.filter((file) => !file.startsWith("spreadsheet/"))).toEqual([]);
  });

  it("never imports mermaid; the host's diagram loader supplies it", () => {
    expect(importers("mermaid")).toEqual([]);
  });
});
