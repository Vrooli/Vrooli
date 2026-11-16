// Vite plugin for @vrooli/file-viewer. A host adds `fileViewer()` to its Vite
// plugins (see README, "Consumer setup"). It provides what the viewer's
// renderers load by URL rather than import:
//
//   - PDF.js resources (CJK cmaps, standard fonts, wasm decoders, ICC
//     profiles), served in development and emitted into the build, so PDFs
//     render offline and behind proxies;
//   - the self-contained slide runtime, built once with its dependencies and
//     exposed as the `virtual:presentation-runtime-url` module.
//
// Workers need no plugin: they are `new Worker(new URL(...))` calls Vite
// bundles itself, provided the package is excluded from dependency
// pre-bundling (`optimizeDeps.exclude`).
import { readFileSync, readdirSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { build } from "vite";

const RUNTIME_ENTRY = fileURLToPath(new URL("../dist/preview/presentationRuntime.js", import.meta.url));
const VIRTUAL_ID = "virtual:presentation-runtime-url";
const DEV_PATH = "@vrooli/file-viewer/presentation-runtime.js";

/**
 * PDF.js fetches these resources by name. Emit the complete owned sets so CJK,
 * standard fonts, color profiles and image decoders work offline and via proxies.
 * @returns {import("vite").Plugin}
 */
function pdfAssets() {
  const root = dirname(createRequire(import.meta.url).resolve("pdfjs-dist/package.json"));
  /** @type {Map<string, string>} */
  const files = new Map();
  for (const group of ["cmaps", "standard_fonts", "wasm", "iccs"]) {
    for (const name of readdirSync(join(root, group))) {
      if (/\.(bcmap|pfb|ttf|wasm|icc)$/.test(name) || name.startsWith("LICENSE")) files.set(`pdf-assets/${group}/${name}`, join(root, group, name));
    }
  }
  return {
    name: "file-viewer-pdf-assets",
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

/**
 * Emit an inert, self-contained script. The authenticated host fetches it as
 * data, then executes it only in an opaque frame. A fetch failure must stay in
 * the drawer instead of triggering the application's stale-module reload guard.
 * @returns {import("vite").Plugin}
 */
function presentationSource() {
  const id = `\0${VIRTUAL_ID}`;
  /** @type {import("vite").ResolvedConfig} */
  let config;
  /** @type {Promise<{ code: string; files: string[] }> | null} */
  let cached = null;
  const compile = () => cached ??= (async () => {
    const result = await build({
      configFile: false, root: config.root, publicDir: false,
      mode: config.command === "serve" ? "development" : config.mode,
      logLevel: "warn",
      // Vite library builds preserve this expression unless explicitly defined.
      define: { "process.env.NODE_ENV": JSON.stringify(config.command === "serve" ? "development" : "production") },
      build: {
        write: false, target: "es2020", minify: "esbuild",
        lib: { entry: RUNTIME_ENTRY, name: "PresentationRuntime", formats: ["iife"] },
        rollupOptions: { output: { inlineDynamicImports: true } },
      },
    });
    if ("on" in result) throw Error("Unexpected presentation build watcher");
    const outputs = (Array.isArray(result) ? result : [result]).flatMap((output) => output.output), chunk = outputs[0];
    if (outputs.length !== 1 || !chunk || chunk.type !== "chunk" || chunk.imports.length || chunk.dynamicImports.length) throw Error("Presentation runtime must be self-contained");
    return { code: chunk.code, files: Object.keys(chunk.modules) };
  })().catch((error) => { cached = null; throw error; });
  return {
    name: "file-viewer-presentation-source",
    configResolved(value) { config = value; },
    resolveId(value) { if (value === VIRTUAL_ID) return id; },
    async load(value) {
      if (value !== id) return;
      if (config.command === "serve") return `export default ${JSON.stringify(`${config.base}${DEV_PATH}`)};`;
      const { code, files } = await compile();
      files.forEach((file) => { this.addWatchFile(file); });
      const asset = this.emitFile({ type: "asset", name: "presentationRuntime.js", source: code });
      return `export default import.meta.ROLLUP_FILE_URL_${asset};`;
    },
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        if (!request.url?.split("?")[0]?.endsWith(`/${DEV_PATH}`)) { next(); return; }
        void compile().then(({ code, files }) => {
          server.watcher.add(files);
          response.setHeader("Content-Type", "text/javascript; charset=utf-8");
          response.setHeader("Cache-Control", "no-store");
          response.end(code);
        }).catch((error) => { next(error); });
      });
    },
    handleHotUpdate() { cached = null; },
    watchChange() { cached = null; },
  };
}

/**
 * The file viewer's Vite plugins.
 * @returns {import("vite").Plugin[]}
 */
export function fileViewer() {
  return [pdfAssets(), presentationSource()];
}

export default fileViewer;
