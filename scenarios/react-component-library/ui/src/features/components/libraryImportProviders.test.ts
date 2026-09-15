import { describe, expect, it, vi } from "vitest";

import type { LibraryImportResolution } from "../../api/components";
import {
  createLibraryImportDefinitionProvider,
  createLibraryImportHoverProvider,
  importAtPosition,
} from "./libraryImportProviders";

function model(source: string) {
  return {
    getLineContent: () => source,
    uri: { path: "components/Consumer/versions/1.0.0/Consumer.tsx" },
  } as never;
}

const resolved = {
  resolved: true,
  specifier: "@vrooli/react-component-library/Button/2",
  assetId: "controls.button",
  libraryId: "react-component-library:Button",
  version: "2.2.0",
  exportKind: "component",
  description: "A primary action control.",
  sourcePath: "components/Button/versions/2.2.0/Button.tsx",
  diagnostic: "",
} as LibraryImportResolution;

describe("library import Monaco providers", () => {
  it("shows rich hover metadata for a resolvable import", async () => {
    const resolve = vi.fn().mockResolvedValue(resolved);
    const provider = createLibraryImportHoverProvider(resolve);
    const hover = await provider.provideHover(
      model('import { Button } from "@vrooli/react-component-library/Button/2";'),
      { lineNumber: 1, column: 48 } as never,
      undefined as never,
    );

    expect(resolve).toHaveBeenCalledWith(
      resolved.specifier,
      "components/Consumer/versions/1.0.0/Consumer.tsx",
    );
    expect(hover?.contents[0]).toMatchObject({
      value: expect.stringContaining("controls.button"),
    });
    expect(hover?.contents[0]).toMatchObject({
      value: expect.stringContaining("2.2.0"),
    });
    expect(hover?.contents[0]).toMatchObject({
      value: expect.stringContaining("A primary action control."),
    });
  });

  it("returns a version-pinned definition in the same Monaco viewer", async () => {
    const resolve = vi.fn().mockResolvedValue(resolved);
    const loaded: string[] = [];
    const monaco = {
      Uri: { parse: (value: string) => ({ toString: () => value }) },
    } as never;
    const provider = createLibraryImportDefinitionProvider(
      monaco,
      resolve,
      async (_result, uri) => {
        loaded.push(uri.toString());
      },
    );
    const definition = await provider.provideDefinition(
      model('import { Button } from "@vrooli/react-component-library/Button/2";'),
      { lineNumber: 1, column: 48 } as never,
      undefined as never,
    );

    expect(loaded[0]).toContain("rcl://library/react-component-library%3AButton/2.2.0/");
    expect(definition && !Array.isArray(definition)).toBe(true);
    if (!definition || Array.isArray(definition)) throw new Error("expected one Monaco definition");
    expect(definition.uri.toString()).toBe(loaded[0]);
  });

  it("explains an unresolved library import instead of hiding it", async () => {
    const resolve = vi.fn().mockResolvedValue({
      ...resolved,
      resolved: false,
      diagnostic: "library import does not resolve",
    });
    const provider = createLibraryImportHoverProvider(resolve);
    const hover = await provider.provideHover(
      model('import { Missing } from "@vrooli/react-component-library/Missing";'),
      { lineNumber: 1, column: 50 } as never,
      undefined as never,
    );

    expect(hover?.contents[0]).toMatchObject({
      value: expect.stringContaining("Unresolved library import"),
    });
    expect(hover?.contents[0]).toMatchObject({
      value: expect.stringContaining("does not resolve"),
    });
  });

  it("keeps relative imports explicit and does not offer a catalog definition", async () => {
    const resolve = vi.fn().mockResolvedValue({
      ...resolved,
      resolved: false,
      specifier: "../Button",
      diagnostic: "relative import is local to the current file",
    });
    const source = model('import { Button } from "../Button";');
    expect(importAtPosition(source, { lineNumber: 1, column: 30 } as never)?.specifier).toBe(
      "../Button",
    );

    const monaco = {
      Uri: { parse: (value: string) => ({ value }) },
    } as never;
    const provider = createLibraryImportDefinitionProvider(monaco, resolve);
    await expect(
      provider.provideDefinition(
        source,
        { lineNumber: 1, column: 30 } as never,
        undefined as never,
      ),
    ).resolves.toBeUndefined();
  });
});
