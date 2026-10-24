import { existsSync } from "node:fs";
import { join } from "node:path";

// Agent Manager has not adopted the generated i18n catalog yet. Keep the
// canonical Makefile contract honest: if a catalog is introduced, require the
// corresponding generator/checker instead of silently treating it as current.
const catalog = join(process.cwd(), "src", "i18n", "locales", "en.json");
if (existsSync(catalog)) {
  console.error("i18n catalog found without generated-string validation; add the standard strings generator before enabling it.");
  process.exit(1);
}

console.log("No generated i18n catalog configured.");
