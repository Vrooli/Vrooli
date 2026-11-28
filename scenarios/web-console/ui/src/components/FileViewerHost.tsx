import { FileViewerProvider, KEYS_LOCALE, type FileViewerLoaders } from "@vrooli/file-viewer";
import { useMemo, type ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { useMessagesViewStore } from "../stores/useMessagesViewStore";
import { rewriteLocalhostUrlForProxy } from "../lib/terminalFileLinks";
import { fileViewerPreferences } from "../lib/fileViewerPreferences";

// Localhost links in rendered Markdown go through Web Console's same-origin
// loopback route, so a phone never opens "localhost" on itself.
function rewriteHref(href: string): string | null {
  return typeof window === "undefined" ? null : rewriteLocalhostUrlForProxy(href, window.location.origin);
}

// The viewer bundles neither SheetJS nor Mermaid; Web Console owns both and
// hands them over here, so each loads only when a workbook or diagram opens.
const loaders: FileViewerLoaders = {
  spreadsheet: () => import("@vrooli/file-viewer/spreadsheet").then((module) => module.spreadsheetEngine),
  diagram: () => import("mermaid").then((module) => module.default),
};

/**
 * Connects the shared file viewer to Web Console: the viewer follows the app's
 * language (i18next's `cimode` test pseudo-locale becomes the viewer's `keys`
 * one), shares the reading text size with the Messages view, routes
 * localhost links through the tunnel, and supplies the spreadsheet and
 * diagram engines. Mount once above anything that renders
 * the viewer or its Markdown.
 */
export function FileViewerHost({ children }: { children: ReactNode }) {
  const { i18n } = useTranslation();
  const fontSize = useMessagesViewStore((state) => state.messagesFontSize);
  const setFontSize = useMessagesViewStore((state) => state.setMessagesFontSize);
  const readerFontSize = useMemo(() => ({ value: fontSize, onChange: setFontSize }), [fontSize, setFontSize]);
  const language = i18n.resolvedLanguage ?? i18n.language;
  return (
    <FileViewerProvider locale={language === "cimode" ? KEYS_LOCALE : language} readerFontSize={readerFontSize} rewriteHref={rewriteHref} loaders={loaders} viewPreferences={fileViewerPreferences}>
      {children}
    </FileViewerProvider>
  );
}
