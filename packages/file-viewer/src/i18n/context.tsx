import { createContext, useContext, useMemo, type ReactNode } from "react";

import type { FileViewerLoaders } from "../loaders.js";
import type { MessageOverrides } from "./catalog.js";
import { createTranslator, directionOf, formatLocaleOf, type Translate, type TranslateAdapter } from "./translate.js";

/** A persisted text size for the Markdown reader, owned by the host. */
export interface ReaderFontSizePreference {
  value: number;
  onChange: (size: number) => void;
}

export interface FileViewerHost {
  /** BCP 47 tag; "en", "ar" and "ja" have catalogs, others use English. `"keys"` renders keys (tests). */
  locale?: string;
  /** Text direction; defaults to the locale's. */
  dir?: "ltr" | "rtl";
  /** Per-key copy overrides, typed against the catalog. */
  overrides?: MessageOverrides;
  /** Adapter consulted before the package catalogs; return undefined to fall through. */
  translate?: TranslateAdapter;
  /** Persist the Markdown reader's text size; without it the size is internal state. */
  readerFontSize?: ReaderFontSizePreference;
  /**
   * Rewrite a link's target before it is rendered (Web Console routes localhost
   * URLs through its tunnel). Return null or undefined to keep the link as is.
   */
  rewriteHref?: (href: string) => string | null | undefined;
  /**
   * Engines for content that needs an optional peer dependency (spreadsheets,
   * Mermaid diagrams). Without one the content shows a "not available here"
   * state with a download link or its source.
   */
  loaders?: FileViewerLoaders;
  /**
   * Who shows a file's size, format and download. "bar" (default): each
   * renderer's own bar under the preview. "host": the host shows them (in
   * `renderFooter`), so the viewer draws no bar of its own: no metadata, no
   * download, and no "Open in new tab" or "Copy source" menu either.
   */
  fileInfo?: "bar" | "host";
}

export interface FileViewerContextValue {
  t: Translate;
  /** Locale for `Intl` formatting and language-sensitive logic. */
  locale: string;
  dir: "ltr" | "rtl";
  readerFontSize: ReaderFontSizePreference | undefined;
  rewriteHref: ((href: string) => string | null | undefined) | undefined;
  loaders: FileViewerLoaders;
  fileInfo: "bar" | "host";
}

const NO_LOADERS: FileViewerLoaders = {};

const DEFAULT_VALUE: FileViewerContextValue = {
  t: createTranslator("en"),
  locale: "en",
  dir: "ltr",
  readerFontSize: undefined,
  rewriteHref: undefined,
  loaders: NO_LOADERS,
  fileInfo: "bar",
};

const FileViewerContext = createContext<FileViewerContextValue>(DEFAULT_VALUE);

/**
 * Configure copy, direction and host preferences for every viewer below it.
 * Optional: without a provider the viewer renders English with internal state.
 */
export function FileViewerProvider({ children, ...host }: FileViewerHost & { children: ReactNode }) {
  const { locale = "en", dir, overrides, translate, readerFontSize, rewriteHref, loaders, fileInfo = "bar" } = host;
  // Keyed by the loader functions, not the object, so an inline `loaders={{ ... }}` does not rebuild the context.
  const spreadsheet = loaders?.spreadsheet, diagram = loaders?.diagram;
  const value = useMemo<FileViewerContextValue>(() => ({
    t: createTranslator(locale, { overrides, adapter: translate }),
    locale: formatLocaleOf(locale),
    dir: dir ?? directionOf(locale),
    readerFontSize,
    rewriteHref,
    loaders: spreadsheet || diagram ? { spreadsheet, diagram } : NO_LOADERS,
    fileInfo,
  }), [locale, dir, overrides, translate, readerFontSize, rewriteHref, spreadsheet, diagram, fileInfo]);
  return <FileViewerContext.Provider value={value}>{children}</FileViewerContext.Provider>;
}

/** Copy, locale and direction for the current viewer scope. */
export function useFileViewerText(): Pick<FileViewerContextValue, "t" | "locale" | "dir"> {
  const { t, locale, dir } = useContext(FileViewerContext);
  return { t, locale, dir };
}

/** Host preferences and hooks (reader font size, link rewriting, engine loaders, who shows file info). */
export function useFileViewerHost(): Pick<FileViewerContextValue, "readerFontSize" | "rewriteHref" | "loaders" | "fileInfo"> {
  const { readerFontSize, rewriteHref, loaders, fileInfo } = useContext(FileViewerContext);
  return { readerFontSize, rewriteHref, loaders, fileInfo };
}
