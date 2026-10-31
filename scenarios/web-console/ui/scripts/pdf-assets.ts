import { readFileSync, readdirSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import type { Plugin } from "vite";

// PDF.js fetches these resources by name. Emit the complete owned sets so CJK,
// standard fonts, color profiles and image decoders work offline and via proxies.
export function pdfAssets(): Plugin {
  const root = dirname(createRequire(import.meta.url).resolve("pdfjs-dist/package.json"));
  const files = new Map<string, string>();
  for (const group of ["cmaps", "standard_fonts", "wasm", "iccs"]) {
    for (const name of readdirSync(join(root, group))) {
      if (/\.(bcmap|pfb|ttf|wasm|icc)$/.test(name) || name.startsWith("LICENSE")) files.set(`pdf-assets/${group}/${name}`, join(root, group, name));
    }
  }
  return {
    name: "web-console-pdf-assets",
    generateBundle() {
      for (const [fileName, path] of files) this.emitFile({ type: "asset", fileName, source: readFileSync(path) });
      this.emitFile({ type: "asset", fileName: "pdf-assets/LICENSE", source: readFileSync(join(root, "LICENSE")) });
    },
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        const name = request.url?.split("?")[0]?.replace(/^\//, "");
        const path = name ? files.get(name) : undefined;
        if (!path) { next(); return; }
        response.setHeader("Content-Type", path.endsWith(".wasm") ? "application/wasm" : "application/octet-stream");
        response.end(readFileSync(path));
      });
    },
  };
}
