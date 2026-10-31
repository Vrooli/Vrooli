import { build, type Plugin, type ResolvedConfig } from "vite";
import { resolve } from "node:path";

// Emit an inert, self-contained script. The authenticated host fetches it as
// data, then executes it only in an opaque frame. A fetch failure must stay in
// the drawer instead of triggering the application's stale-module reload guard.
export function presentationSource(): Plugin {
  const name = "virtual:presentation-runtime-url", id = `\0${name}`;
  const developmentPath = "@web-console/presentation-runtime.js";
  let config: ResolvedConfig, cached: Promise<{ code: string; files: string[] }> | null = null;
  const compile = () => cached ??= (async () => {
    const result = await build({
      configFile: false, root: config.root, publicDir: false,
      mode: config.command === "serve" ? "development" : config.mode,
      logLevel: "warn",
      // Vite library builds preserve this expression unless explicitly defined.
      define: { "process.env.NODE_ENV": JSON.stringify(config.command === "serve" ? "development" : "production") },
      build: {
        write: false, target: "es2020", minify: "esbuild",
        lib: { entry: resolve(config.root, "src/components/file-preview/presentationRuntime.ts"), name: "PresentationRuntime", formats: ["iife"] },
        rollupOptions: { output: { inlineDynamicImports: true } },
      },
    });
    if ("on" in result) throw Error("Unexpected presentation build watcher");
    const outputs = (Array.isArray(result) ? result : [result]).flatMap(output => output.output), chunk = outputs[0];
    if (outputs.length !== 1 || !chunk || chunk.type !== "chunk" || chunk.imports.length || chunk.dynamicImports.length) throw Error("Presentation runtime must be self-contained");
    return { code: chunk.code, files: Object.keys(chunk.modules) };
  })().catch((error: unknown) => { cached = null; throw error; });
  return {
    name: "web-console-presentation-source",
    configResolved(value) { config = value; },
    resolveId(value) { if (value === name) return id; },
    async load(value) {
      if (value !== id) return;
      if (config.command === "serve") return `export default ${JSON.stringify(`${config.base}${developmentPath}`)};`;
      const { code, files } = await compile(); files.forEach(file => { this.addWatchFile(file); });
      const asset = this.emitFile({ type: "asset", name: "presentationRuntime.js", source: code });
      return `export default import.meta.ROLLUP_FILE_URL_${asset};`;
    },
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        if (!request.url?.split("?")[0]?.endsWith(`/${developmentPath}`)) { next(); return; }
        void compile().then(({ code, files }) => {
          server.watcher.add(files); response.setHeader("Content-Type", "text/javascript; charset=utf-8");
          response.setHeader("Cache-Control", "no-store"); response.end(code);
        }).catch((error: unknown) => { next(error); });
      });
    },
    handleHotUpdate() { cached = null; },
    watchChange() { cached = null; },
  };
}
