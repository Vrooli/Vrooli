import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import { describe, expect, it } from "vitest";

const sourceRoot = import.meta.dirname;
function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    return entry.isDirectory() ? sourceFiles(path) : statSync(path).isFile() && /\.[tj]sx?$/.test(entry.name) ? [path] : [];
  });
}

describe("RCL adoption boundary", () => {
  it("uses published primitives and has no private UI fork", () => {
    const source = sourceFiles(sourceRoot).filter((path) => path !== import.meta.filename).map((path) => readFileSync(path, "utf8")).join("\n");
    expect(source).not.toMatch(/components\/ui\/(button|input|textarea)/);
    expect(source).toContain("@vrooli/react-component-library/Button/2");
    expect(source).toContain("@vrooli/react-component-library/Input/1");
  });
});
