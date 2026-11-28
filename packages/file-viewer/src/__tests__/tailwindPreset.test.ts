import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";
import postcss from "postcss";
import tailwindcss from "tailwindcss";
import { describe, expect, it } from "vitest";

import preset from "../../styles/tailwind-preset.js";

async function generate(content: string) {
  const result = await postcss([tailwindcss({ presets: [preset], content: [{ raw: content, extension: "html" }] })]).process("@tailwind utilities;", { from: undefined });
  return result.css;
}

function sourceFiles(directory: string): string[] {
  return readdirSync(directory).flatMap((name) => {
    const path = join(directory, name);
    if (statSync(path).isDirectory()) return name === "__tests__" ? [] : sourceFiles(path);
    return /\.tsx?$/.test(name) && !/\.test\./.test(name) ? [path] : [];
  });
}

describe("tailwind preset", () => {
  it("maps fv utilities onto the --fv-* variables, with alpha where the token allows it", async () => {
    const css = await generate("bg-fv-accent/10 text-fv-text-muted border-fv-default border-fv-accent z-fv-chrome");
    expect(css).toContain("rgb(var(--fv-accent) / 0.1)");
    expect(css).toContain("rgb(var(--fv-text-muted) / var(--tw-text-opacity");
    expect(css).toContain("rgb(var(--fv-border-default))");
    expect(css).toContain("rgb(var(--fv-accent-border))");
    expect(css).toContain("var(--fv-z-chrome)");
  });

  it("defines every fv utility the viewer's source uses", async () => {
    const used = new Set<string>();
    for (const file of sourceFiles("src")) {
      for (const match of readFileSync(file, "utf8").matchAll(/(?<![\w-])((?:bg|text|border|ring|outline|fill|accent|divide|z|stroke)-fv-[a-z-]+(?:\/\d+)?)/g)) used.add(match[1] ?? "");
    }
    const css = await generate([...used].join(" "));
    const missing = [...used].filter((name) => !new RegExp(`\\.${name.replace(/\//g, "\\\\/")}(?![\\w-])`).test(css));
    expect(missing).toEqual([]);
  });
});
