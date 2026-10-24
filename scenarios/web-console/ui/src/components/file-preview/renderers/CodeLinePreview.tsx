import { useDeferredValue, useEffect, useMemo, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { ChevronLeft, ChevronRight, Copy, FileCode2, Minus, Plus, Search, WrapText } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { getLanguageFromPath, highlightCode, type HighlightedLine, type HighlightToken } from "../../../lib/codeHighlighter";
import { cn } from "../../../lib/classnames";
import { isLogPath, searchSource, selectedSource, sourceLines, SOURCE_PAGE_SIZE, type LogLevel, type SourceMatch } from "../sourceData";
import { usePreviewClipboard } from "../usePreviewClipboard";
import { InspectionHeader, InspectionOptions, InspectionSetting, InspectionJump, PreviewButton, previewSelectClass } from "./ArtifactChrome";
import { PreviewNotice } from "./shared";

// Every source embedding shares the same bounded reader. It never mutates the
// source or token stream; matches are intersected with token offsets at render.
export function CodeLinePreview({ content, path, highlightLine, truncated = false, plainText = false }: { content: string; path: string; highlightLine: number | null; truncated?: boolean; plainText?: boolean }) {
  const { t } = useTranslation(), c = strings.inspectionPreview;
  const all = useMemo(() => sourceLines(content), [content]), log = isLogPath(path), language = plainText ? null : getLanguageFromPath(path);
  const [searchOpen, setSearchOpen] = useState(false);
  const [query, setQuery] = useState(""), deferred = useDeferredValue(query), [level, setLevel] = useState<LogLevel | "all">("all");
  const [wrap, setWrap] = useState(plainText), [fontSize, setFontSize] = useState(plainText ? 14 : 13), [page, setPage] = useState(0);
  const [matchIndex, setMatchIndex] = useState(0), [lineInput, setLineInput] = useState(""), [invalid, setInvalid] = useState(false);
  const [selection, setSelection] = useState<{ anchor: number; end: number } | null>(highlightLine ? { anchor: highlightLine, end: highlightLine } : null);
  const [highlighted, setHighlighted] = useState<HighlightedLine[] | null>(null), [target, setTarget] = useState<number | null>(highlightLine);
  const [jump, setJump] = useState(0), [activeLine, setActiveLine] = useState(highlightLine ?? 1);
  const focusJump = useRef(false);
  const frame = useRef<HTMLDivElement>(null), clipboard = usePreviewClipboard();
  useEffect(() => {
    if (!searchOpen) return;
    const raf = requestAnimationFrame(() => { frame.current?.querySelector<HTMLInputElement>(`[data-testid="${selectors.inspectionPreview.sourceSearch}"]`)?.focus(); });
    return () => { cancelAnimationFrame(raf); };
  }, [searchOpen]);
  const filtered = useMemo(() => log && level !== "all" ? all.filter((line) => line.level === level) : all, [all, log, level]);
  const search = useMemo(() => searchSource(filtered, deferred), [filtered, deferred]);
  const currentMatch = search.matches[Math.min(matchIndex, search.matches.length - 1)];
  const pages = Math.max(1, Math.ceil(filtered.length / SOURCE_PAGE_SIZE)), currentPage = Math.min(page, pages - 1);
  const visible = filtered.slice(currentPage * SOURCE_PAGE_SIZE, (currentPage + 1) * SOURCE_PAGE_SIZE);
  const activeVisible = visible.some((line) => line.number === activeLine);
  const first = selection ? Math.min(selection.anchor, selection.end) : 0, last = selection ? Math.max(selection.anchor, selection.end) : 0;
  useEffect(() => {
    let active = true; setHighlighted(null);
    // Avoid unbounded synchronous grammar work on the main thread. Plain source
    // retains search/navigation and paging when the highlighter budget is exceeded.
    if (!language || log || content.length > 150000 || all.length > 3000 || all.some((line) => line.text.length > 10000)) return;
    void highlightCode(content, language).then((lines) => { if (active) setHighlighted(lines); }).catch(() => {});
    return () => { active = false; };
  }, [content, language, log, all]);
  useEffect(() => {
    if (!target) return;
    const raf = requestAnimationFrame(() => { const line = frame.current?.querySelector<HTMLElement>(`[data-source-line="${String(target)}"]`); line?.scrollIntoView({ block: "center" }); if (focusJump.current) { line?.querySelector("button")?.focus({ preventScroll: true }); focusJump.current = false; } });
    return () => { cancelAnimationFrame(raf); };
  }, [target, currentPage, highlighted, jump]);
  // Incoming line links may point beyond the first page.
  useEffect(() => { if (highlightLine && highlightLine <= all.length) { setPage(Math.floor((highlightLine - 1) / SOURCE_PAGE_SIZE)); setTarget(highlightLine); } }, [highlightLine, all.length]);
  const reveal = (number: number, lines = filtered) => { const index = lines.findIndex((line) => line.number === number); if (index >= 0) { setPage(Math.floor(index / SOURCE_PAGE_SIZE)); setTarget(number); } };
  const navigateMatch = (direction: number) => {
    if (!search.matches.length) return;
    const next = (matchIndex + direction + search.matches.length) % search.matches.length;
    setMatchIndex(next); const match = search.matches[next]; if (match) reveal(match.line);
  };
  const goLine = () => {
    const number = Number(lineInput);
    if (!Number.isInteger(number) || number < 1 || number > all.length) { setInvalid(true); return false; }
    setInvalid(false); setLevel("all"); setSelection({ anchor: number, end: number }); setActiveLine(number); focusJump.current = true; setJump((value) => value + 1); reveal(number, all); return true;
  };
  const moveLine = (event: KeyboardEvent<HTMLButtonElement>, number: number) => {
    const index = filtered.findIndex((line) => line.number === number);
    const next = event.key === "ArrowDown" ? index + 1 : event.key === "ArrowUp" ? index - 1 : event.key === "Home" ? 0 : event.key === "End" ? filtered.length - 1 : null;
    if (next === null) return;
    event.preventDefault(); event.stopPropagation();
    const line = filtered[Math.max(0, Math.min(filtered.length - 1, next))];
    if (!line) return;
    setActiveLine(line.number); focusJump.current = true; setJump((value) => value + 1); reveal(line.number);
    if (event.shiftKey) setSelection({ anchor: selection?.anchor ?? number, end: line.number });
  };
  useEffect(() => {
    setMatchIndex(0);
    const firstMatch = search.matches[0];
    if (firstMatch) { const index = filtered.findIndex((line) => line.number === firstMatch.line); setPage(Math.floor(index / SOURCE_PAGE_SIZE)); setTarget(firstMatch.line); }
  }, [search, filtered]);
  const fontSizes = [11, 12, 13, 14, 15, 16, 18];
  const adjustFont = (step: number) => { setFontSize((current) => fontSizes[Math.max(0, Math.min(fontSizes.length - 1, fontSizes.indexOf(current) + step))] ?? current); };
  const copyStatus = clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : "";
  return <div ref={frame} className="wc-artifact-preview flex h-full min-h-0 min-w-0 flex-col bg-wc-surface-base" data-testid="file-preview-code">
    <InspectionHeader icon={<FileCode2 />} title={t(strings.readerPreview.source)} description={`${log ? "LOG" : language ?? t(strings.messagesFileViewer.plaintext)} · ${t(strings.messagesFileViewer.linesSuffix, { count: filtered.length })}`}>
      <PreviewButton label={t(c.search)} testId={selectors.inspectionPreview.sourceSearchToggle} pressed={searchOpen} onClick={() => { setSearchOpen(!searchOpen); if (searchOpen) setQuery(""); }}><Search /></PreviewButton>
      <InspectionOptions label={t(c.options)}>{(close) => <>
        <InspectionSetting label={t(c.fontSize)}>
          <div role="group" aria-label={t(c.fontSize)} className="flex items-center justify-between rounded-xl border border-wc-default bg-wc-surface-base p-1">
            <PreviewButton label={t(strings.messagesFileViewer.decreaseFontSize)} disabled={fontSize <= 11} onClick={() => { adjustFont(-1); }}><Minus /></PreviewButton>
            <span className="font-mono text-sm tabular-nums text-wc-text-primary">{fontSize}px</span>
            <PreviewButton label={t(strings.messagesFileViewer.increaseFontSize)} disabled={fontSize >= 18} onClick={() => { adjustFont(1); }}><Plus /></PreviewButton>
          </div>
        </InspectionSetting>
        <button type="button" aria-label={t(wrap ? strings.messagesFileViewer.disableWordWrap : strings.messagesFileViewer.enableWordWrap)} aria-pressed={wrap} onClick={() => { setWrap(!wrap); }} className="flex min-h-12 w-full items-center gap-3 rounded-xl border border-wc-default px-3 py-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-wc-accent">
          <WrapText className="h-4 w-4 text-wc-text-muted" aria-hidden="true" /><span className="flex-1 text-start">{t(strings.messagesFileViewer.wrap)}</span><span aria-hidden="true" className={cn("wc-inspection-toggle", wrap && "wc-inspection-toggle-on")} />
        </button>
        {log && <InspectionSetting label={t(c.severity)}><select aria-label={t(c.severity)} value={level} onChange={(event) => { setLevel(event.target.value as LogLevel | "all"); setPage(0); setMatchIndex(0); }} className={`${previewSelectClass} w-full text-base`}>
          <option value="all">{t(c.allLevels)}</option>{(["error", "warn", "info", "debug", "other"] as const).map((value) => <option key={value} value={value}>{t(c[value])}</option>)}
        </select></InspectionSetting>}
        <div className="border-t border-wc-default pt-5"><InspectionJump label={t(c.goLine)} value={lineInput} invalid={invalid} placeholder="1" numeric onChange={(value) => { setLineInput(value); setInvalid(false); }} onGo={() => { if (goLine()) close(); }} />
          {invalid && <p role="alert" className="pt-2 text-xs text-wc-danger-text">{t(c.lineInvalid)}</p>}
        </div>
      </>}</InspectionOptions>
    </InspectionHeader>
    {(searchOpen || query) && <div className="shrink-0 border-b border-wc-default px-3 py-2">      <div className="flex min-w-0 items-center gap-1">
        <label className="relative min-w-0 flex-1"><Search aria-hidden="true" className="pointer-events-none absolute start-3 top-3.5 h-4 w-4 text-wc-text-muted" />
          <input data-testid={selectors.inspectionPreview.sourceSearch} type="search" aria-label={t(c.search)} placeholder={t(c.search)} value={query} onChange={(event) => { setQuery(event.target.value); setMatchIndex(0); }}
            onKeyDown={(event) => { if (event.key === "Enter") { event.preventDefault(); navigateMatch(event.shiftKey ? -1 : 1); } if (event.key === "Escape") { event.stopPropagation(); setQuery(""); setSearchOpen(false); frame.current?.querySelector<HTMLButtonElement>(`[data-testid="${selectors.inspectionPreview.sourceSearchToggle}"]`)?.focus(); } }}
            className="h-11 w-full min-w-0 rounded-lg border border-wc-default bg-wc-surface-input/40 pe-2 ps-9 text-sm outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
        </label>
        <PreviewButton label={t(c.previousMatch)} disabled={!search.matches.length} onClick={() => { navigateMatch(-1); }}><ChevronLeft /></PreviewButton>
        <PreviewButton label={t(c.nextMatch)} disabled={!search.matches.length} onClick={() => { navigateMatch(1); }}><ChevronRight /></PreviewButton>
      </div>
      {query && <p role="status" className="px-2 py-1 text-xs text-wc-text-muted" title={t(c.searchScope)}>{t(c.matches, { current: currentMatch ? Math.min(matchIndex + 1, search.matches.length) : 0, total: search.matches.length })}{search.limited ? ` · ${t(c.matchLimit)}` : ""}</p>}
</div>}
    {truncated && <PreviewNotice message={t(strings.messagesFileViewer.truncatedNotice)} tone="info" />}
    <div dir="ltr" role="region" aria-label={t(c.controls)} tabIndex={0} data-testid={selectors.inspectionPreview.sourceLines} className="min-h-0 min-w-0 flex-1 overflow-auto bg-wc-surface-base py-2 font-mono leading-relaxed" style={{ fontSize }}>
      {visible.map((line) => <div key={line.number} data-source-line={line.number} className={cn("flex min-w-full items-start", first <= line.number && last >= line.number && "bg-wc-accent/10", target === line.number && "ring-1 ring-inset ring-wc-accent/40")}>
        <button type="button" aria-label={t(c.selectLine, { line: line.number })} aria-pressed={first <= line.number && last >= line.number} tabIndex={line.number === activeLine || !activeVisible && line.number === visible[0]?.number ? 0 : -1} onFocus={() => { setActiveLine(line.number); }} onKeyDown={(event) => { moveLine(event, line.number); }} onClick={(event) => { setActiveLine(line.number); setSelection({ anchor: event.shiftKey && selection ? selection.anchor : line.number, end: line.number }); }}
          className={cn("wc-source-line-number sticky left-0 flex shrink-0 self-stretch items-start justify-end border-e border-wc-default bg-wc-surface-raised/60 px-3 text-xs tabular-nums text-wc-text-muted outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent", plainText ? "min-h-7 py-0.5 leading-6" : "min-h-11 py-2")} style={{ minWidth: `${String(String(all.length).length + 3)}ch` }}>{line.number}</button>
        <pre className={cn("wc-source-line-text m-0 min-w-0 flex-1 bg-transparent px-3 text-wc-text-primary", plainText ? "min-h-7 py-0.5 leading-6" : "min-h-11 py-2", wrap ? "whitespace-pre-wrap [overflow-wrap:anywhere]" : "whitespace-pre")}>
          {renderTokens(highlighted?.[line.number - 1]?.tokens ?? [{ content: line.text }], search.matches.filter((match) => match.line === line.number), currentMatch)}
        </pre>
      </div>)}
      {!visible.length && <p role="status" className="p-6 text-sm text-wc-text-muted">{t(c.noLines)}</p>}
    </div>
    {(selection || copyStatus || pages > 1) && <div className="flex shrink-0 flex-wrap items-center justify-between gap-x-2 border-t border-wc-default px-2 py-1 text-xs text-wc-text-muted">
      <span role="status" className="min-w-0 flex-1">{copyStatus || (selection ? t(c.selectedLines, { start: first, end: last }) : t(strings.messagesFileViewer.linesSuffix, { count: all.length }))}</span>
      {selection && <PreviewButton label={t(c.copySelection)} onClick={() => { void clipboard.copy(selectedSource(content, selection.anchor, selection.end)); }}><Copy /></PreviewButton>}
      {pages > 1 && <div className="flex items-center"><PreviewButton label={t(strings.artifactPreview.previousPage)} disabled={!currentPage} onClick={() => { setPage(currentPage - 1); }}><ChevronLeft /></PreviewButton><span>{currentPage + 1}/{pages}</span><PreviewButton label={t(strings.artifactPreview.nextPage)} disabled={currentPage >= pages - 1} onClick={() => { setPage(currentPage + 1); }}><ChevronRight /></PreviewButton></div>}
    </div>}
  </div>;
}
function renderTokens(tokens: HighlightToken[], matches: SourceMatch[], active?: SourceMatch): ReactNode[] {
  const elements: ReactNode[] = []; let offset = 0;
  for (let index = 0; index < tokens.length; index++) {
    const token = tokens[index]; if (!token) continue; const end = offset + token.content.length;
    const bounds = [offset, ...matches.flatMap((match) => [match.start, match.end]).filter((point) => point > offset && point < end), end].sort((a, b) => a - b);
    elements.push(<span key={index} style={{ color: token.color }} className={token.fontStyle === "italic" ? "italic" : token.fontStyle === "bold" ? "font-bold" : undefined}>{bounds.slice(0, -1).map((start, part) => {
      const text = token.content.slice(start - offset, (bounds[part + 1] ?? end) - offset), match = matches.find((entry) => entry.start <= start && entry.end > start);
      return match ? <mark key={part} className={cn("rounded-sm bg-wc-accent/20 text-inherit", match === active && "outline outline-1 outline-wc-accent")}>{text}</mark> : text;
    })}</span>);
    offset = end;
  }
  return elements;
}
