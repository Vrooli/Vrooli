// Bundle src/index.ts -> dist/index.js as a single ESM file targeting Node 20.
// ts-morph is bundled in (no external deps at runtime besides Node builtins).

import { build } from "esbuild";
import { fileURLToPath } from "node:url";
import * as path from "node:path";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(__dirname, "..");

await build({
  entryPoints: [path.join(root, "src/index.ts")],
  outfile: path.join(root, "dist/index.js"),
  bundle: true,
  format: "esm",
  platform: "node",
  target: "node20",
  sourcemap: false,
  minify: false,
  logLevel: "info",
  // ts-morph ships a large dep (typescript); bundle everything so the
  // produced dist/index.js can be invoked without node_modules. This
  // matches the "single bundled file" requirement.
  external: [],
  // ESM bundle needs a banner for `import.meta`-style affordances + to
  // expose require for CJS deps that ts-morph (or its deps) may need.
  banner: {
    js:
      "import { createRequire as __cR } from 'module';" +
      "const require = __cR(import.meta.url);" +
      "import { fileURLToPath as __fU } from 'url';" +
      "import { dirname as __dN } from 'path';" +
      "const __filename = __fU(import.meta.url);" +
      "const __dirname = __dN(__filename);",
  },
});

console.error("[build] dist/index.js written");
