import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const uiRoot = resolve(scriptDir, "..");

describe("sync-catalog accessors", () => {
  it("does not treat a localized string method as part of its catalog key", () => {
    const output = execFileSync(process.execPath, ["scripts/sync-catalog.mjs", "--check"], {
      cwd: uiRoot,
      encoding: "utf8",
    });

    expect(output).toContain("keys, up to date");
  });
});
