import { useCallback, useEffect, useId, useMemo, useRef, useState, type MouseEvent } from "react";
import { BookOpen, ChevronLeft, ChevronRight, Copy, Download, List, Search, X } from "lucide-react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { useTranslation } from "react-i18next";
import { useEbook } from "../useEbook";
import { renderEbookChapter, type EbookData } from "../epubData";
import { useMarkdownReader } from "../useMarkdownReader";
import { usePreviewClipboard } from "../usePreviewClipboard";
import { selectors } from "../../../consts/selectors";
import { strings } from "../../../consts/strings";
import type { PreviewModel, PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, InspectionChoices, InspectionHeader, InspectionOptions, InspectionSetting, PreviewButton, PreviewLoading, previewTextButtonClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

export function EbookPreview(props: PreviewRendererProps) { return <EbookReader key={props.model.previewId} {...props} />; }
function EbookReader({ model, onReopen }: PreviewRendererProps) {
  const { t } = useTranslation(), c = strings.ebookPreview, [attempt, setAttempt] = useState(0), result = useEbook(model.blobHref, model.sizeBytes, attempt);
  return <ArtifactFrame testId={selectors.ebookPreview.root}>
    {!result ? <PreviewLoading label={t(c.loading)} /> : "data" in result ? <EbookContent key={attempt} book={result.data} model={model} /> : <>
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-auto px-5 py-8 text-center"><BookOpen aria-hidden="true" className="h-9 w-9 text-wc-text-muted" /><p role="status" className="max-w-sm text-sm leading-relaxed text-wc-text-secondary">{t(c[result.error])}</p><button type="button" className={previewTextButtonClass} onClick={() => { if (result.error === "fetch" && onReopen) onReopen(); else setAttempt(value => value + 1); }}>{t(strings.messagesFileViewer.reopen)}</button></div>
      <ArtifactFooter model={model} metadata={model.basename} />
    </>}
  </ArtifactFrame>;
}
function EbookContent({ book, model }: { book: EbookData; model: PreviewModel }) {
  const { t } = useTranslation(), c = strings.ebookPreview;
  const [chapter, setChapter] = useState(0), [contents, setContents] = useState(false), [search, setSearch] = useState(false), [query, setQuery] = useState(""), [font, setFont] = useState(18), [measure, setMeasure] = useState("comfortable"), [progress, setProgress] = useState(0);
  const [rendered, setRendered] = useState<{ html: string; omitted: boolean; error?: "invalid" | "limit" }>({ html: "", omitted: false });
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 760;
  const reader = useMarkdownReader(rendered.html, query, false, true), clipboard = usePreviewClipboard(), prefix = `epub-${useId().replace(/[^a-z0-9]/gi, "")}-${String(chapter)}-`;
  const pane = useRef<HTMLDivElement>(null), input = useRef<HTMLInputElement>(null), contentsButton = useRef<HTMLDivElement>(null), searchButton = useRef<HTMLDivElement>(null), fragment = useRef<string | null>(null), pendingFocus = useRef(false);
  useEffect(() => {
    try {
      const value = renderEbookChapter(book, chapter, prefix, t(c.table)); setRendered(value);
      return value.dispose;
    } catch (error) { setRendered({ html: "", omitted: false, error: error instanceof RangeError ? "limit" : "invalid" }); }
  }, [book, chapter, prefix, t, c.table]);
  const weights = useMemo(() => book.chapters.map(item => (book.parts[item.path]?.length ?? 0)), [book]), total = weights.reduce((sum, value) => sum + value, 0);
  const updateProgress = useCallback(() => {
    const node = pane.current; if (!node) return;
    const extent = node.scrollHeight - node.clientHeight, fraction = extent > 0 ? Math.max(0, Math.min(1, node.scrollTop / extent)) : 1;
    setProgress(Math.round((weights.slice(0, chapter).reduce((sum, value) => sum + value, 0) + (weights[chapter] ?? 0) * fraction) / total * 100));
  }, [chapter, weights, total]);
  const { articleRef } = reader;
  useEffect(() => {
    if (!rendered.html) return;
    const frame = requestAnimationFrame(() => {
      const destination = fragment.current;
      const target = destination ? Array.from(articleRef.current?.querySelectorAll<HTMLElement>("[id]") ?? []).find(node => node.id === `${prefix}${encodeURIComponent(destination)}`) : null;
      if (target) { target.tabIndex = -1; target.scrollIntoView({ block: "start" }); target.focus({ preventScroll: true }); }
      else if (pendingFocus.current) { if (pane.current) pane.current.scrollTop = 0; pane.current?.focus({ preventScroll: true }); }
      fragment.current = null; pendingFocus.current = false; updateProgress();
    });
    return () => { cancelAnimationFrame(frame); };
  }, [rendered.html, contents, wide, prefix, articleRef, updateProgress]);
  useEffect(() => { if (search) input.current?.focus(); }, [search]);
  const closeContents = () => { setContents(false); contentsButton.current?.querySelector("button")?.focus(); };
  const closeSearch = () => { setSearch(false); setQuery(""); searchButton.current?.querySelector("button")?.focus(); };
  const navigate = (index: number, target = "") => {
    if (index < 0 || index >= book.chapters.length) return;
    fragment.current = target || null; pendingFocus.current = true; setChapter(index); if (!wide) setContents(false); setQuery("");
    if (index === chapter) {
      const node = target ? Array.from(reader.articleRef.current?.querySelectorAll<HTMLElement>("[id]") ?? []).find(element => element.id === `${prefix}${encodeURIComponent(target)}`) : pane.current;
      if (node) { node.tabIndex = -1; node.scrollIntoView({ block: "start" }); node.focus({ preventScroll: true }); }
    }
  };
  const link = (event: MouseEvent<HTMLDivElement>) => {
    const anchor = event.target instanceof Element ? event.target.closest("a") : null, href = anchor?.getAttribute("href");
    if (!href?.startsWith("#wc-epub=")) return;
    event.preventDefault();
    try { const target = JSON.parse(decodeURIComponent(href.slice(9))) as { chapter: number; fragment: string }; if (Number.isInteger(target.chapter) && typeof target.fragment === "string") navigate(target.chapter, target.fragment); } catch { /* Invalid internal link stays inert. */ }
  };
  return <div ref={ref} className="flex min-h-0 min-w-0 flex-1 flex-col">
    <style>{reader.highlightStyle}</style>
    <InspectionHeader icon={<BookOpen />} title={book.title || model.basename} description={t(c.summary, { creator: book.creator || t(c.unknownCreator), count: book.chapters.length })}>
      <div ref={contentsButton}><PreviewButton label={t(c.contents)} pressed={contents} onClick={() => { if (contents) closeContents(); else { setContents(true); if (!wide) { setSearch(false); setQuery(""); } } }}><List /></PreviewButton></div>
      <div ref={searchButton}><PreviewButton label={t(c.find)} pressed={search} onClick={() => { if (search) closeSearch(); else { setSearch(true); if (!wide) setContents(false); } }}><Search /></PreviewButton></div>
      <InspectionOptions label={t(c.options)}>{() => <>
        <InspectionSetting label={t(c.textSize)}><InspectionChoices label={t(c.textSize)} value={String(font)} options={[16, 18, 20, 22].map(value => ({ value: String(value), label: `${String(value)} px` }))} onChange={value => { setFont(Number(value)); }} /></InspectionSetting>
        <InspectionSetting label={t(c.width)}><InspectionChoices label={t(c.width)} value={measure} options={[{ value: "comfortable", label: t(c.comfortable) }, { value: "wide", label: t(c.wide) }]} onChange={setMeasure} /></InspectionSetting>
        <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.fidelity)}</p>
        <button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(reader.articleRef.current?.innerText ?? reader.articleRef.current?.textContent ?? ""); }}><Copy aria-hidden="true" className="h-4 w-4" />{t(c.copy)}</button>
        {model.canDownload && <a href={model.blobHref} download={model.basename} className={previewTextButtonClass}><Download aria-hidden="true" className="h-4 w-4" />{t(strings.messagesFileViewer.download)}</a>}
        <p role="status" className="text-xs text-wc-text-muted">{clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : ""}</p>
      </>}</InspectionOptions>
    </InspectionHeader>
    {book.fixed && <PreviewNotice message={t(c.fixed)} tone="info" />}
    {rendered.omitted && <PreviewNotice message={t(c.omittedImages)} tone="info" />}
    {search && <div className="shrink-0 border-b border-wc-default px-3 py-1"><div className="flex items-center gap-1"><input ref={input} type="search" aria-label={t(c.find)} placeholder={t(c.find)} value={query} onChange={event => { setQuery(event.target.value); }} onKeyDown={event => { if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); closeSearch(); } else if (event.key === "Enter") { event.preventDefault(); reader.next(event.shiftKey ? -1 : 1); } }} className="h-11 min-w-0 flex-1 rounded-lg bg-wc-surface-input px-3 text-base text-wc-text-primary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
      <PreviewButton label={t(strings.artifactPreview.previousMatch)} disabled={!reader.matches} onClick={() => { reader.next(-1); }}><ChevronLeft /></PreviewButton><PreviewButton label={t(strings.artifactPreview.nextMatch)} disabled={!reader.matches} onClick={() => { reader.next(1); }}><ChevronRight /></PreviewButton><PreviewButton label={t(strings.artifactPreview.closeSearch)} onClick={closeSearch}><X /></PreviewButton></div><p role="status" className="pb-1 text-xs text-wc-text-muted">{query ? t(strings.readerPreview.searchCount, { current: reader.matches ? reader.selected + 1 : 0, total: reader.matches, cap: reader.matches === 500 ? "+" : "" }) : t(strings.artifactPreview.searchHint)}</p></div>}
    <div className="flex min-h-0 min-w-0 flex-1">
      {contents && <aside data-testid={selectors.ebookPreview.contents} className={wide ? "flex w-60 shrink-0 flex-col border-e border-wc-default" : "flex min-w-0 flex-1 flex-col"}><div className="flex items-center justify-between px-4 py-1"><p className="text-xs font-semibold text-wc-text-muted">{t(c.contents)}</p><PreviewButton label={t(strings.inspectionPreview.closeOptions)} onClick={closeContents}><X /></PreviewButton></div>
        <nav className="min-h-0 overflow-auto px-3 pb-4" aria-label={t(c.contents)} onKeyDown={event => { if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); closeContents(); } }}>{book.contents.map((item, i) => <button type="button" key={i} aria-current={chapter === item.chapter ? "location" : undefined} onClick={() => { navigate(item.chapter, item.fragment); }} style={{ paddingInlineStart: 12 + item.depth * 12 }} className={`block min-h-11 w-full break-words rounded-lg py-2 pe-3 text-start text-sm outline-none hover:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-wc-accent ${chapter === item.chapter ? "bg-wc-accent/10 text-wc-accent" : "text-wc-text-secondary"}`}>{item.title}</button>)}</nav>
      </aside>}
      <div ref={pane} data-testid={selectors.ebookPreview.pane} tabIndex={0} role="region" aria-label={t(c.content)} onScroll={updateProgress} onLoadCapture={updateProgress} className={`min-h-0 min-w-0 flex-1 overflow-auto overscroll-contain outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent ${contents && !wide ? "hidden" : ""}`}>
        <div className={`mx-auto px-5 py-7 sm:px-10 sm:py-10 ${measure === "wide" ? "max-w-[95ch]" : "max-w-[70ch]"}`} style={{ fontSize: font }}>
          {rendered.error ? <p role="status" className="text-sm text-wc-text-muted">{t(c[rendered.error])}</p> : <div ref={reader.articleRef} data-testid={selectors.ebookPreview.article} onClick={link} className="wc-document-reader" dir="auto" dangerouslySetInnerHTML={{ __html: rendered.html }} />}
        </div>
      </div>
    </div>
    <div className="h-0.5 shrink-0 bg-wc-surface-input" aria-hidden="true"><div className="h-full bg-wc-accent/60" style={{ width: `${String(progress)}%` }} /></div>
    <div role="group" aria-label={t(c.navigation)} className="flex shrink-0 items-center justify-between gap-1 border-t border-wc-default px-3 py-1"><p className="min-w-0 truncate text-xs text-wc-text-muted" title={t(c.counter, { current: chapter + 1, total: book.chapters.length })}>{chapter + 1} / {book.chapters.length}</p><div className="flex items-center gap-1"><PreviewButton label={t(c.previous)} disabled={!chapter} onClick={() => { navigate(chapter - 1); }}><ChevronLeft className="rtl:rotate-180" /></PreviewButton><PreviewButton label={t(c.next)} disabled={chapter >= book.chapters.length - 1} onClick={() => { navigate(chapter + 1); }}><ChevronRight className="rtl:rotate-180" /></PreviewButton></div><p role="status" aria-label={t(c.progress, { percent: progress })} className="text-xs tabular-nums text-wc-text-muted">{progress}%</p></div>
  </div>;
}
