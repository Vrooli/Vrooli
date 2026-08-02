import { execSync } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

const tempDir = mkdtempSync(join(tmpdir(), "storage-manager-selectors-"));
try {
  execSync(
    `pnpm exec tsc src/consts/selectors.ts --module commonjs --moduleResolution node --target es2022 --skipLibCheck --outDir ${tempDir}`,
    { stdio: "pipe" },
  );
  const selectorsModule = await import(pathToFileURL(join(tempDir, "consts", "selectors.js")).href);
  writeFileSync("src/consts/selectors.manifest.json", `${JSON.stringify(selectorsModule.selectorsManifest, null, 2)}\n`, "utf8");
} finally {
  rmSync(tempDir, { recursive: true, force: true });
}
