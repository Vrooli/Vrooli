import { execFileSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const temporaryDirectory = mkdtempSync(join(tmpdir(), "scenario-to-desktop-selectors-"));

try {
  execFileSync(
    "pnpm",
    [
      "exec",
      "tsc",
      "src/consts/selectors.ts",
      "--module",
      "commonjs",
      "--moduleResolution",
      "node",
      "--target",
      "es2022",
      "--skipLibCheck",
      "--outDir",
      temporaryDirectory,
    ],
    { stdio: "pipe" },
  );

  const moduleUrl = pathToFileURL(join(temporaryDirectory, "selectors.js")).href;
  const selectorModule = await import(moduleUrl);
  const manifestPath = "src/consts/selectors.manifest.json";
  writeFileSync(manifestPath, `${JSON.stringify(selectorModule.selectorsManifest, null, 2)}\n`, "utf8");
  process.stdout.write(`Generated ${manifestPath}\n`);
} finally {
  rmSync(temporaryDirectory, { recursive: true, force: true });
}
