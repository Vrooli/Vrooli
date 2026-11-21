import type { Plugin } from "vite";

export interface ResolvedLibrarySpecifier {
  name: string;
  kind: string;
  version: string;
  exactSpecifier: string;
  majorSpecifier: string;
  sourcePath: string;
  libraryId: string;
  draft?: boolean;
}

export interface ResolveLibrarySpecifierOptions {
  libraryRoot: string;
  draftOverrides?: Record<string, string>;
}

export function resolveLibrarySpecifier(
  specifier: string,
  options: ResolveLibrarySpecifierOptions,
): Promise<ResolvedLibrarySpecifier | null>;

export function classifyLibrarySpecifier(specifier: string): "bare" | "major" | "exact" | null;

export function majorSpecifier(name: string, version: string | number): string;

export interface SourceLibraryResolverOptions {
  libraryRoot?: string;
  draftOverrides?: Record<string, string>;
  devOnly?: boolean;
  usageContractsInTests?: boolean;
}

export function sourceLibraryResolver(options?: SourceLibraryResolverOptions): Plugin;
