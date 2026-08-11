import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const temporaryDirectory = mkdtempSync(join(tmpdir(), "vrooli-onboarding-selectors-"));

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
      "--outDir",
      temporaryDirectory,
    ],
    { stdio: "pipe" },
  );

  const moduleUrl = pathToFileURL(join(temporaryDirectory, "selectors.js")).href;
  const selectorModule = await import(moduleUrl);
  const manifestPath = "src/consts/selectors.manifest.json";
  const manifestContents = `${JSON.stringify(selectorModule.selectorsManifest, null, 2)}\n`;
  const existingContents = existsSync(manifestPath) ? readFileSync(manifestPath, "utf8") : null;
  if (existingContents === manifestContents) {
    process.stdout.write(`Selector manifest is current: ${manifestPath}\n`);
  } else {
    writeFileSync(manifestPath, manifestContents, "utf8");
    process.stdout.write(`Generated ${manifestPath}\n`);
  }
} finally {
  rmSync(temporaryDirectory, { recursive: true, force: true });
}
