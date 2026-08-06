import { readFile, writeFile } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import postcss from "postcss";
import tailwindcss from "tailwindcss";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "../../../..");
const adapterRoot = join(repoRoot, "templates", "design");
const content = [
  join(repoRoot, "scenarios", "react-component-library", "library", "**", "*.{ts,tsx}"),
];
const kits = [
  "vrooli-default",
  "vrooli-command-display",
  "vrooli-conversion-landing",
];

for (const kit of kits) {
  const adapter = join(adapterRoot, kit, "adapters", "react-vite-tailwind");
  const theme = JSON.parse(await readFile(join(adapter, "tailwind.theme.json"), "utf8"));
  const result = await postcss([
    tailwindcss({
      content,
      corePlugins: { preflight: false },
      theme: { extend: theme },
      plugins: [],
    }),
  ]).process("@tailwind utilities;", { from: join(adapter, "preview-utilities.input.css") });
  await writeFile(join(adapter, "preview-utilities.css"), result.css);
  console.log(`built ${kit}/adapters/react-vite-tailwind/preview-utilities.css`);
}
