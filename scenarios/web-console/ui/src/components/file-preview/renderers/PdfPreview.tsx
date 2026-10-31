import { useEffect, useRef, useState } from "react";
import { ChevronLeft, ChevronRight, Minus, Plus, Search, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { formatBytes } from "../format";
import { usePdfReader } from "../usePdfReader";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, PreviewLoading, PreviewToolbar, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

export function PdfPreview(props: PreviewRendererProps) {
  return <PdfReader key={`${props.model.previewId}:${props.model.blobHref}`} {...props} />;
}

function PdfReader({ model }: PreviewRendererProps) {
  const { t } = useTranslation();
  const c = strings.artifactPreview;
  const reader = usePdfReader(model.blobHref);
  const [pageInput, setPageInput] = useState("1");
  const [zoom, setZoom] = useState("page-width");
  const [searchOpen, setSearchOpen] = useState(false);
  const [query, setQuery] = useState("");
  const [password, setPassword] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);
  const find = useRef(reader.find);
  find.current = reader.find;
  useEffect(() => { setPageInput(String(reader.page)); }, [reader.page]);
  useEffect(() => {
    const timer = window.setTimeout(() => { find.current(query); }, 200);
    return () => { window.clearTimeout(timer); };
  }, [query, reader.pages]);
  useEffect(() => { if (searchOpen) inputRef.current?.focus(); }, [searchOpen]);
  const ready = reader.pages > 0 && !reader.failed;
  const closeSearch = () => { setSearchOpen(false); setQuery(""); reader.find(""); };
  const changeZoom = (value: string) => { setZoom(value); reader.zoom(value); };
  const manualZoom = (factor: number) => { changeZoom(String(Math.max(0.1, Math.min(4, reader.scale * factor)))); };
  const zoomOptions = ["0.5", "0.75", "1", "1.25", "1.5", "2", "3", "4"];
  const metadata = ["PDF", reader.pages ? t(c.pageCount, { count: reader.pages }) : "", formatBytes(model.sizeBytes)].filter(Boolean).join(" · ");
  return <ArtifactFrame testId="file-preview-pdf">
    <PreviewToolbar label={t(c.pdfControls)}>
      <div className="flex min-w-0 items-center">
        <PreviewButton label={t(c.previousPage)} disabled={!ready || reader.page <= 1} onClick={() => { reader.goTo(reader.page - 1); }}><ChevronLeft /></PreviewButton>
        <form className="flex items-center gap-1 text-xs text-wc-text-secondary" onSubmit={(event) => { event.preventDefault(); reader.goTo(Number(pageInput)); setPageInput(String(Math.max(1, Math.min(reader.pages, Math.round(Number(pageInput) || reader.page))))); }}>
          <input type="text" inputMode="numeric" pattern="[0-9]*" aria-label={t(c.page)} value={pageInput} disabled={!ready}
            onChange={(event) => { setPageInput(event.target.value); }} onBlur={() => { setPageInput(String(reader.page)); }}
            className="h-11 w-9 rounded-lg border border-wc-default bg-wc-surface-input text-center font-mono text-xs outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
          <span className="whitespace-nowrap">/ {reader.pages || "–"}</span>
        </form>
        <PreviewButton label={t(c.nextPage)} disabled={!ready || reader.page >= reader.pages} onClick={() => { reader.goTo(reader.page + 1); }}><ChevronRight /></PreviewButton>
      </div>
      <select aria-label={t(c.pdfZoom)} className={`${previewSelectClass} w-24`} value={zoom} disabled={!ready} onChange={(event) => { changeZoom(event.target.value); }}>
        <option value="page-width">{t(c.fitWidth)}</option><option value="page-fit">{t(c.fitPage)}</option>
        {!zoomOptions.includes(zoom) && !zoom.startsWith("page-") && <option value={zoom}>{Math.round(Number(zoom) * 100)}%</option>}
        {zoomOptions.map((value) => <option key={value} value={value}>{Number(value) * 100}%</option>)}
      </select>
      <PreviewButton label={t(c.searchDocument)} disabled={!ready} pressed={searchOpen} onClick={() => { if (searchOpen) closeSearch(); else setSearchOpen(true); }}><Search /></PreviewButton>
    </PreviewToolbar>
    {searchOpen && <div className="shrink-0 border-b border-wc-default bg-wc-surface-raised px-3 py-1">
      <div className="flex items-center gap-1">
        <input ref={inputRef} type="search" value={query} aria-label={t(c.searchDocument)} placeholder={t(c.searchDocument)} onChange={(event) => { setQuery(event.target.value); }}
          onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); reader.find(query, event.shiftKey, true); } else if (event.key === "Escape") { event.stopPropagation(); closeSearch(); } }}
          className="h-11 min-w-0 flex-1 rounded-lg bg-wc-surface-input px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
        <PreviewButton label={t(c.previousMatch)} disabled={!query || reader.matches.total === 0} onClick={() => { reader.find(query, true, true); }}><ChevronLeft /></PreviewButton>
        <PreviewButton label={t(c.nextMatch)} disabled={!query || reader.matches.total === 0} onClick={() => { reader.find(query, false, true); }}><ChevronRight /></PreviewButton>
        <PreviewButton label={t(c.closeSearch)} onClick={closeSearch}><X /></PreviewButton>
      </div>
      <p role="status" className="pb-1 text-xs text-wc-text-muted">{query ? reader.matches.pending ? t(c.searching) : t(c.searchMatches, { current: reader.matches.current, total: reader.matches.total }) : t(c.searchHint)}</p>
    </div>}
    <div className="relative min-h-0 flex-1">
      <div ref={reader.containerRef} role="region" className="wc-pdf-scroll absolute inset-0 overflow-auto" data-testid={selectors.artifactPreview.pdfPages} tabIndex={0} aria-label={t(strings.messagesFileViewer.pdfPreview)}>
        <div ref={reader.pagesRef} className="pdfViewer" />
      </div>
      {!ready && !reader.passwordNeeded && !reader.failed && <div className="pointer-events-none absolute inset-0"><PreviewLoading label={t(c.loadingPdf)} /></div>}
      {reader.passwordNeeded && !reader.failed && <form className="absolute inset-0 flex flex-col items-center justify-center gap-3 bg-wc-surface-base p-5" onSubmit={(event) => { event.preventDefault(); reader.unlock(password); setPassword(""); }}>
        <p className="text-sm text-wc-text-secondary">{t(reader.incorrectPassword ? c.incorrectPassword : c.passwordRequired)}</p>
        <input type="password" autoComplete="off" aria-label={t(c.password)} value={password} onChange={(event) => { setPassword(event.target.value); }} className="h-11 w-full max-w-xs rounded-lg border border-wc-default bg-wc-surface-input px-3 outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
        <button type="submit" disabled={!password} className={previewTextButtonClass}>{t(c.unlockPdf)}</button>
      </form>}
      {reader.failed && <div role="alert" className="absolute inset-0 flex items-center justify-center bg-wc-surface-base p-5"><PreviewNotice message={t(c.pdfLoadError)} /></div>}
    </div>
    {ready && <div className="flex shrink-0 items-center justify-center gap-2 border-t border-wc-default text-xs text-wc-text-muted">
      <PreviewButton label={t(c.zoomOut)} disabled={reader.scale <= 0.1} onClick={() => { manualZoom(1 / 1.25); }}><Minus /></PreviewButton>
      <span className="w-12 text-center font-mono tabular-nums">{Math.round(reader.scale * 100)}%</span>
      <PreviewButton label={t(c.zoomIn)} disabled={reader.scale >= 4} onClick={() => { manualZoom(1.25); }}><Plus /></PreviewButton>
    </div>}
    <ArtifactFooter model={model} metadata={metadata} />
  </ArtifactFrame>;
}
