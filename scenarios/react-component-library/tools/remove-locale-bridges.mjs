import { readdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";

const root = join(process.cwd(), "scenarios");
const marker = /\n?\/\/ vrooli:library-locale-bridge start[\s\S]*?\/\/ vrooli:library-locale-bridge end\n?/g;

async function walk(path) {
  const entries = await readdir(path, { withFileTypes: true });
  for (const entry of entries) {
    const child = join(path, entry.name);
    if (entry.isDirectory()) await walk(child);
    else if (entry.name === "index.ts" && child.includes("/ui/src/i18n/")) {
      const source = await readFile(child, "utf8");
      const updated = source.replace(marker, "\n");
      if (updated !== source) await writeFile(child, updated);
    }
  }
}

await walk(root);
