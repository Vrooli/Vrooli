// @vitest-environment node
import { existsSync } from "node:fs";
import { describe, expect, it, vi } from "vitest";
import type { Plugin, ResolvedConfig } from "vite";

import { fileViewer } from "./index.js";

const VIRTUAL = "virtual:presentation-runtime-url";

function plugin(name: string): Plugin {
  const found = fileViewer().find((entry) => entry.name === name);
  if (!found) throw new Error(`missing plugin ${name}`);
  return found;
}

// Plugin hooks are objects-or-functions in Vite's types; these are all functions.
function call<T>(hook: unknown, context: object, ...args: unknown[]): T {
  if (typeof hook !== "function") throw new Error("expected a hook function");
  return hook.call(context, ...args) as T;
}

describe("fileViewer vite plugins", () => {
  it("provides the PDF resources and the slide runtime", () => {
    expect(fileViewer().map((entry) => entry.name)).toEqual(["file-viewer-pdf-assets", "file-viewer-presentation-source"]);
  });

  it("emits every PDF.js resource set, with its licence, into the build", () => {
    const emitFile = vi.fn();
    call(plugin("file-viewer-pdf-assets").generateBundle, { emitFile });
    const names = emitFile.mock.calls.map(([file]) => (file as { fileName: string }).fileName);
    expect(names).toContain("pdf-assets/LICENSE");
    for (const group of ["cmaps", "standard_fonts", "wasm", "iccs"]) expect(names.some((name) => name.startsWith(`pdf-assets/${group}/`))).toBe(true);
  });

  it("serves a PDF resource in development and passes other requests on", () => {
    const middlewares: ((request: { url?: string }, response: { setHeader: (name: string, value: string) => void; end: (body: Buffer) => void }, next: () => void) => void)[] = [];
    call(plugin("file-viewer-pdf-assets").configureServer, {}, { middlewares: { use: (handler: (typeof middlewares)[number]) => middlewares.push(handler) } });
    const response = { setHeader: vi.fn(), end: vi.fn() };
    const next = vi.fn();
    middlewares[0]?.({ url: "/pdf-assets/standard_fonts/LiberationSans-Regular.ttf?x=1" }, response, next);
    expect(response.end).toHaveBeenCalledOnce();
    expect(next).not.toHaveBeenCalled();
    middlewares[0]?.({ url: "/unrelated.js" }, response, next);
    expect(next).toHaveBeenCalledOnce();
  });

  it("answers the slide runtime's virtual module with a dev URL while serving", async () => {
    const presentation = plugin("file-viewer-presentation-source");
    call(presentation.configResolved, {}, { command: "serve", base: "/app/", root: process.cwd(), mode: "development" } as ResolvedConfig);
    const id = call<string | undefined>(presentation.resolveId, {}, VIRTUAL);
    expect(id).toBe(`\0${VIRTUAL}`);
    expect(call<string | undefined>(presentation.resolveId, {}, "./other")).toBeUndefined();
    const source = await call<Promise<string>>(presentation.load, {}, id);
    expect(source).toBe('export default "/app/@vrooli/file-viewer/presentation-runtime.js";');
  });

  // The runtime is built from dist/, so this needs `pnpm build` first.
  it.skipIf(!existsSync(new URL("../dist/preview/presentationRuntime.js", import.meta.url)))("builds the slide runtime as one self-contained script", async () => {
    const presentation = plugin("file-viewer-presentation-source");
    call(presentation.configResolved, {}, { command: "build", base: "./", root: process.cwd(), mode: "production" } as ResolvedConfig);
    const emitFile = vi.fn().mockReturnValue("ref1");
    const addWatchFile = vi.fn();
    const source = await call<Promise<string>>(presentation.load, { emitFile, addWatchFile }, `\0${VIRTUAL}`);
    expect(source).toBe("export default import.meta.ROLLUP_FILE_URL_ref1;");
    const asset = emitFile.mock.calls[0]?.[0] as { name: string; source: string };
    expect(asset.name).toBe("presentationRuntime.js");
    expect(asset.source).toContain("fv-presentation");
    expect(asset.source).not.toMatch(/^\s*import\s/m);
  }, 60_000);
});
