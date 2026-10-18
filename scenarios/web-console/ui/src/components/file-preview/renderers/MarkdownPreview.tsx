import { useCallback, useEffect, useRef, useState, type MouseEvent } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { BookOpen, ChevronLeft, ChevronRight, Code2, Copy, List, Search, Settings2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { MarkdownRenderer } from "../../markdown";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { looksLikeFileReference } from "../../../lib/fileReferences";
import { formatBytes } from "../format";
import { useMarkdownReader } from "../useMarkdownReader";
import { usePreviewClipboard } from "../usePreviewClipboard";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, PreviewToolbar, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
import { CodeLinePreview } from "./TextRenderers";
import { PreviewNotice } from "./shared";

export function MarkdownPreview(props: PreviewRendererProps) { return <MarkdownReader key={props.model.previewId} {...props} />; }
function MarkdownReader({ model, text, onNavigate }: PreviewRendererProps) {
  const { t } = useTranslation(), c = strings.readerPreview;
  const content = text?.content ?? "";
  const [source, setSource] = useState(!!model.line), [outline, setOutline] = useState(false), [search, setSearch] = useState(false);
  const [query, setQuery] = useState(""), [font, setFont] = useState(16);
  const input = useRef<HTMLInputElement>(null), frame = useRef<HTMLDivElement>(null);
  const [pendingHeading, setPendingHeading] = useState<string | null>(null);
  const { ref, rect } = useResizeObserver<HTMLDivElement>();
  const wide = (rect?.width ?? 0) >= 760;
  const reader = useMarkdownReader(content, query, source);
  const readerRef = useRef(reader); readerRef.current = reader;
  const clipboard = usePreviewClipboard();
  const copyStatus = clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : "";
  useEffect(() => { if (search) input.current?.focus(); }, [search]);
  useEffect(() => {
    if (!pendingHeading) return;
    const frame = requestAnimationFrame(() => { readerRef.current.reveal(pendingHeading); setPendingHeading(null); });
    return () => { cancelAnimationFrame(frame); };
  }, [pendingHeading]);
  const closeSearch = () => { setSearch(false); setQuery(""); frame.current?.querySelector<HTMLButtonElement>(`[data-testid="${selectors.readerPreview.searchToggle}"]`)?.focus(); };
  const navigate = useCallback((path: string) => {
    let decoded = path;
    try { decoded = decodeURIComponent(path); } catch { /* Preserve a literal malformed escape. */ }
    if (/^(file:\/\/|\/|~\/)/.test(decoded)) onNavigate(decoded);
    else onNavigate(`${model.resolvedPath.slice(0, model.resolvedPath.lastIndexOf("/") + 1)}${decoded}`);
  }, [model.resolvedPath, onNavigate]);
  const link = useCallback((href: string, event: MouseEvent<HTMLAnchorElement>) => {
    if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    if (href.startsWith("#")) {
      event.preventDefault();
      let fragment = href.slice(1); try { fragment = decodeURIComponent(fragment); } catch { /* Use literal. */ }
      const heading = readerRef.current.headings.find((item) => item.slug === fragment || item.id === fragment);
      if (heading) readerRef.current.reveal(heading.id);
    } else if (looksLikeFileReference(href.split("#")[0] ?? href)) { event.preventDefault(); navigate(href.split("#")[0] ?? href); }
  }, [navigate]);
  const contents = <nav aria-label={t(c.contents)} className="min-h-0 overflow-auto p-3" data-testid={selectors.readerPreview.outline}>
    <p className="mb-2 px-2 text-xs font-semibold text-wc-text-muted">{t(c.contents)}</p>
    {reader.headings.length ? reader.headings.map((heading) => <button key={heading.id} type="button" onClick={() => { setOutline(false); setPendingHeading(heading.id); }}
      className="block min-h-11 w-full break-words rounded-lg py-2 pe-2 text-start text-xs text-wc-text-secondary outline-none hover:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-wc-accent" style={{ paddingInlineStart: 8 + Math.min(heading.level - 1, 3) * 12 }}>{heading.label}</button>) : <p className="p-2 text-sm text-wc-text-muted">{t(c.noHeadings)}</p>}
  </nav>;
  return <ArtifactFrame testId="file-preview-markdown" frameRef={frame}>
    <span role="status" className="sr-only">{copyStatus}</span>
    <style>{reader.highlightStyle}</style>
    <div ref={ref} className="flex min-h-0 flex-1 flex-col">
      <PreviewToolbar label={t(c.readingControls)}>
        <PreviewButton label={t(source ? c.showRead : c.showSource)} pressed={source} onClick={() => { setSource(!source); setOutline(false); }}>{source ? <BookOpen /> : <Code2 />}</PreviewButton>
        <span className="min-w-0 flex-1 truncate px-2 text-xs text-wc-text-secondary">{t(source ? c.source : c.read)}</span>
        <PreviewButton label={t(c.contents)} pressed={outline} disabled={source} onClick={() => { setOutline(!outline); if (!wide) { setSearch(false); setQuery(""); } }}><List /></PreviewButton>
        <PreviewButton label={t(c.findDocument)} pressed={search} disabled={source} testId={selectors.readerPreview.searchToggle} onClick={() => { if (search) closeSearch(); else { setSearch(true); setOutline(false); } }}><Search /></PreviewButton>
        <details className="relative" onKeyDown={(event) => { if (event.key === "Escape" && event.currentTarget.open) { event.preventDefault(); event.stopPropagation(); event.currentTarget.open = false; event.currentTarget.querySelector("summary")?.focus(); } }}>
          <summary aria-label={t(c.readingOptions)} title={t(c.readingOptions)} className="flex h-11 w-11 cursor-pointer list-none items-center justify-center rounded-lg text-wc-text-muted outline-none focus-visible:ring-2 focus-visible:ring-wc-accent [&::-webkit-details-marker]:hidden"><Settings2 className="h-4 w-4" /></summary>
          <label className="absolute end-0 top-full z-wc-chrome flex w-52 items-center justify-between gap-2 rounded-lg border border-wc-default bg-wc-surface-raised p-3 text-xs shadow-lg">{t(c.textSize)}<select aria-label={t(c.textSize)} className={previewSelectClass} value={font} onChange={(event) => { setFont(Number(event.target.value)); }}>{[14, 16, 18, 20].map((size) => <option key={size} value={size}>{size}px</option>)}</select></label>
        </details>
      </PreviewToolbar>
      {search && !source && <div className="shrink-0 border-b border-wc-default px-3 py-1">
        <div className="flex items-center gap-1"><input ref={input} type="search" aria-label={t(c.findDocument)} placeholder={t(c.findDocument)} value={query} onChange={(event) => { setQuery(event.target.value); }}
          onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); reader.next(event.shiftKey ? -1 : 1); } else if (event.key === "Escape") { event.stopPropagation(); closeSearch(); } }}
          className="h-11 min-w-0 flex-1 rounded-lg bg-wc-surface-input px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
          <PreviewButton label={t(strings.artifactPreview.previousMatch)} disabled={!reader.matches} onClick={() => { reader.next(-1); }}><ChevronLeft /></PreviewButton>
          <PreviewButton label={t(strings.artifactPreview.nextMatch)} disabled={!reader.matches} onClick={() => { reader.next(1); }}><ChevronRight /></PreviewButton>
          <PreviewButton label={t(strings.artifactPreview.closeSearch)} onClick={closeSearch}><X /></PreviewButton></div>
        <p role="status" className="pb-1 text-xs text-wc-text-muted">{query ? t(c.searchCount, { current: reader.matches ? reader.selected + 1 : 0, total: reader.matches, cap: reader.matches === 500 ? "+" : "" }) : t(strings.artifactPreview.searchHint)}</p>
      </div>}
      {text?.truncated && <PreviewNotice message={t(strings.messagesFileViewer.truncatedNotice)} tone="info" />}
      {source ? <div className="min-h-0 flex-1"><CodeLinePreview content={content} path={model.resolvedPath} highlightLine={model.line ?? null} truncated={text?.truncated} /></div> : <div className="flex min-h-0 flex-1">
        {outline && <aside className={wide ? "flex w-52 shrink-0 flex-col border-e border-wc-default" : "flex min-w-0 flex-1 flex-col"}>{contents}</aside>}
        <div className={`min-h-0 min-w-0 flex-1 overflow-auto ${outline && !wide ? "hidden" : ""}`} data-testid={selectors.readerPreview.readingPane}>
          <div ref={reader.articleRef} className="wc-markdown-reader mx-auto max-w-[72ch] px-4 py-5 sm:px-6" style={{ fontSize: font }}>
            {content.trim() ? <MarkdownRenderer content={content} onLinkClick={link} onFileReferenceClick={navigate} /> : <p data-testid="file-preview-empty" className="text-sm text-wc-text-muted">{t(strings.messagesFileViewer.emptyFile)}</p>}
          </div>
        </div>
      </div>}
    </div>
    <ArtifactFooter model={model} metadata={copyStatus || `Markdown · ${formatBytes(model.sizeBytes)}`}>
      <button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(content); }}><Copy className="h-4 w-4" />{t(c.copySource)}</button>
    </ArtifactFooter>
  </ArtifactFrame>;
}
