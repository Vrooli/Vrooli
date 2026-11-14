import { useEffect, useMemo, useRef, useState, type MouseEvent } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { ChevronLeft, ChevronRight, Copy, FileText, List, Search, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { formatBytes } from "../format";
import { useDocument } from "../useDocument";
import { sanitizeDocument } from "../sanitizeDocument";
import { useMarkdownReader } from "../useMarkdownReader";
import { usePreviewClipboard } from "../usePreviewClipboard";
import type { DocumentData } from "../documentData";
import type { PreviewModel, PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, InspectionChoices, InspectionHeader, InspectionOptions, InspectionSetting, PreviewButton, PreviewLoading, previewTextButtonClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

export function DocumentPreview(props: PreviewRendererProps) { return <DocumentReader key={props.model.previewId} {...props} />; }
function DocumentReader({ model, onReopen }: PreviewRendererProps) {
  const { t } = useTranslation(), c = strings.documentPreview;
  const [attempt, setAttempt] = useState(0), result = useDocument(model.blobHref, model.sizeBytes, attempt);
  return <ArtifactFrame testId={selectors.documentPreview.root}>
    {!result ? <PreviewLoading label={t(c.loading)} /> : result.data ? <DocumentContent key={attempt} data={result.data} model={model} /> :
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-auto px-5 py-8 text-center">
        <FileText aria-hidden="true" className="h-9 w-9 text-wc-text-muted" />
        <p role="status" className="max-w-sm text-sm leading-relaxed text-wc-text-secondary">{t(c[result.error])}</p>
        <button type="button" className={previewTextButtonClass} onClick={() => { if (result.error === "fetch" && onReopen) onReopen(); else setAttempt(value => value + 1); }}>{t(strings.messagesFileViewer.reopen)}</button>
      </div>}
    {!result?.data && <ArtifactFooter model={model} metadata={`DOCX · ${formatBytes(model.sizeBytes)}`} />}
  </ArtifactFrame>;
}
function DocumentContent({ data, model }: { data: DocumentData; model: PreviewModel }) {
  const { t } = useTranslation(), c = strings.documentPreview, reading = strings.readerPreview;
  const html = useMemo(() => sanitizeDocument(data.html, t(c.table)), [data.html, t, c.table]);
  const [font, setFont] = useState(16), [outline, setOutline] = useState(false), [search, setSearch] = useState(false), [query, setQuery] = useState("");
  const [pendingHeading, setPendingHeading] = useState<string | null>(null);
  const input = useRef<HTMLInputElement>(null), searchButton = useRef<HTMLDivElement>(null), outlineButton = useRef<HTMLDivElement>(null);
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 760;
  const reader = useMarkdownReader(html, query, false), latestReader = useRef(reader); latestReader.current = reader;
  const clipboard = usePreviewClipboard();
  useEffect(() => { if (search) input.current?.focus(); }, [search]);
  useEffect(() => {
    if (!pendingHeading) return;
    const frame = requestAnimationFrame(() => { latestReader.current.reveal(pendingHeading); setPendingHeading(null); });
    return () => { cancelAnimationFrame(frame); };
  }, [pendingHeading]);
  const closeSearch = () => { setSearch(false); setQuery(""); searchButton.current?.querySelector("button")?.focus(); };
  const closeOutline = () => { setOutline(false); outlineButton.current?.querySelector("button")?.focus(); };
  const link = (event: MouseEvent<HTMLDivElement>) => {
    const anchor = event.target instanceof Element ? event.target.closest("a") : null;
    const href = anchor?.getAttribute("href");
    if (!href?.startsWith("#")) return;
    event.preventDefault();
    let id = href.slice(1); try { id = decodeURIComponent(id); } catch { /* Keep literal. */ }
    const target = Array.from(reader.articleRef.current?.querySelectorAll<HTMLElement>("[id]") ?? []).find(node => node.id === id);
    if (target) { target.tabIndex = -1; target.scrollIntoView({ block: "center" }); target.focus({ preventScroll: true }); }
  };
  return <div ref={ref} className="flex min-h-0 min-w-0 flex-1 flex-col">
    <style>{reader.highlightStyle}</style>
    <InspectionHeader icon={<FileText />} title={t(c.title)} description={t(c.description)}>
      <div ref={outlineButton}><PreviewButton label={t(reading.contents)} pressed={outline} onClick={() => { if (outline) closeOutline(); else { setOutline(true); if (!wide) { setSearch(false); setQuery(""); } } }}><List /></PreviewButton></div>
      <div ref={searchButton}><PreviewButton label={t(reading.findDocument)} pressed={search} onClick={() => { if (search) closeSearch(); else { setSearch(true); if (!wide) setOutline(false); } }}><Search /></PreviewButton></div>
      <InspectionOptions label={t(c.options)}>{() => <>
        <InspectionSetting label={t(reading.textSize)}><InspectionChoices label={t(reading.textSize)} value={String(font)} onChange={value => { setFont(Number(value)); }} options={[14, 16, 18, 20].map(value => ({ value: String(value), label: `${String(value)} px` }))} /></InspectionSetting>
        <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.layoutNote)}</p>
      </>}</InspectionOptions>
    </InspectionHeader>
    {search && <div className="shrink-0 border-b border-wc-default px-3 py-1">
      <div className="flex items-center gap-1"><input ref={input} data-testid={selectors.documentPreview.search} type="search" aria-label={t(reading.findDocument)} placeholder={t(reading.findDocument)} value={query} onChange={event => { setQuery(event.target.value); }}
        onKeyDown={event => { if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); closeSearch(); } else if (event.key === "Enter") { event.preventDefault(); reader.next(event.shiftKey ? -1 : 1); } }}
        className="h-11 min-w-0 flex-1 rounded-lg bg-wc-surface-input px-3 text-base text-wc-text-primary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
        <PreviewButton label={t(strings.artifactPreview.previousMatch)} disabled={!reader.matches} onClick={() => { reader.next(-1); }}><ChevronLeft /></PreviewButton>
        <PreviewButton label={t(strings.artifactPreview.nextMatch)} disabled={!reader.matches} onClick={() => { reader.next(1); }}><ChevronRight /></PreviewButton>
        <PreviewButton label={t(strings.artifactPreview.closeSearch)} onClick={closeSearch}><X /></PreviewButton></div>
      <p role="status" className="pb-1 text-xs text-wc-text-muted">{query ? t(reading.searchCount, { current: reader.matches ? reader.selected + 1 : 0, total: reader.matches, cap: reader.matches === 500 ? "+" : "" }) : t(strings.artifactPreview.searchHint)}</p>
    </div>}
    {(data.warnings || data.omittedImages > 0) && <PreviewNotice message={t(data.omittedImages ? c.omittedImages : c.warnings)} tone="info" />}
    <div className="flex min-h-0 min-w-0 flex-1">
      {outline && <aside data-testid={selectors.documentPreview.outline} className={wide ? "flex w-56 shrink-0 flex-col border-e border-wc-default" : "flex min-w-0 flex-1 flex-col"}>
        <div className="flex shrink-0 items-center justify-between px-4 py-1"><p className="text-xs font-semibold text-wc-text-muted">{t(reading.contents)}</p><PreviewButton label={t(strings.inspectionPreview.closeOptions)} onClick={closeOutline}><X /></PreviewButton></div>
        <nav aria-label={t(reading.contents)} className="min-h-0 overflow-auto px-3 pb-4">{reader.headings.length ? reader.headings.map(heading => <button key={heading.id} type="button" onClick={() => { if (!wide) setOutline(false); setPendingHeading(heading.id); }}
          style={{ paddingInlineStart: 12 + Math.min(heading.level - 1, 3) * 12 }} className="block min-h-11 w-full break-words rounded-lg py-2 pe-3 text-start text-sm text-wc-text-secondary outline-none hover:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-wc-accent">{heading.label}</button>) : <p className="p-3 text-sm text-wc-text-muted">{t(reading.noHeadings)}</p>}</nav>
      </aside>}
      <div data-testid={selectors.documentPreview.pane} tabIndex={0} role="region" aria-label={t(c.content)} className={`min-h-0 min-w-0 flex-1 overflow-auto overscroll-contain outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent ${outline && !wide ? "hidden" : ""}`}>
        <div className="mx-auto max-w-[78ch] px-5 py-7 sm:px-10 sm:py-10" style={{ fontSize: font }}>
          <div ref={reader.articleRef} data-testid={selectors.documentPreview.article} onClick={link} className="wc-document-reader" dir="auto" dangerouslySetInnerHTML={{ __html: html }} />
          {!html.trim() && <p className="text-sm text-wc-text-muted">{t(strings.messagesFileViewer.emptyFile)}</p>}
        </div>
      </div>
    </div>
    <span role="status" className="sr-only">{clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : ""}</span>
    <ArtifactFooter model={model} metadata={clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : `DOCX · ${formatBytes(model.sizeBytes)}`}>
      <button type="button" className={previewTextButtonClass} onClick={() => { void clipboard.copy(reader.articleRef.current?.innerText ?? reader.articleRef.current?.textContent ?? ""); }}><Copy aria-hidden="true" className="h-4 w-4" />{t(c.copyText)}</button>
    </ArtifactFooter>
  </div>;
}
