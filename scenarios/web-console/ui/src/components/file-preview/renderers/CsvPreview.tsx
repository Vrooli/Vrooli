import { useDeferredValue, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { CopyIconButton } from "@vrooli/react-component-library/CopyIconButton";
import { ArrowDown, ArrowLeft, ArrowUp, ArrowUpDown, ChevronDown, ChevronLeft, ChevronRight, Code2, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { copyText } from "../../../lib/clipboard";
import { formatBytes, parseDelimitedDetailed } from "../format";
import { columnWidth, filterAndSortRows, MAX_TABLE_COLUMNS, nextTableSort, prepareTable, TABLE_PAGE_SIZE, type TableRow, type TableSort } from "../tableData";
import { usePreviewClipboard } from "../usePreviewClipboard";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, previewSelectClass } from "./ArtifactChrome";
import { CodeLinePreview } from "./TextRenderers";
import { PreviewNotice } from "./shared";

export function CsvPreview(props: PreviewRendererProps) {
  return <TableInspector key={props.model.previewId} {...props} />;
}

function TableInspector({ model, text }: PreviewRendererProps) {
  const { t, i18n } = useTranslation();
  const c = strings.artifactPreview;
  const content = text?.content ?? "";
  const { ref, rect } = useResizeObserver<HTMLDivElement>();
  const [query, setQuery] = useState("");
  const deferredQuery = useDeferredValue(query);
  const [sort, setSort] = useState<TableSort | null>(null);
  const [page, setPage] = useState(0);
  const [view, setView] = useState("auto");
  const [header, setHeader] = useState(true);
  const [source, setSource] = useState(false);
  const [selected, setSelected] = useState<TableRow | null>(null);
  const [widths, setWidths] = useState<Record<number, number>>({});
  const [activeCell, setActiveCell] = useState({ row: 0, column: 0 });
  const resize = useRef<{ column: number; x: number; width: number } | null>(null);
  const clipboard = usePreviewClipboard();
  const parsed = useMemo(() => parseDelimitedDetailed(content, model.resolvedPath.toLowerCase().endsWith(".tsv") ? "\t" : ","), [content, model.resolvedPath]);
  const data = useMemo(() => prepareTable(parsed.rows, header), [parsed, header]);
  const locale = i18n.resolvedLanguage && i18n.resolvedLanguage !== "cimode" ? i18n.resolvedLanguage : "en";
  const rows = useMemo(() => filterAndSortRows(data.rows, deferredQuery, sort, locale), [data, deferredQuery, sort, locale]);
  const pages = Math.max(1, Math.ceil(rows.length / TABLE_PAGE_SIZE));
  const currentPage = Math.min(page, pages - 1);
  const visibleRows = rows.slice(currentPage * TABLE_PAGE_SIZE, (currentPage + 1) * TABLE_PAGE_SIZE);
  const records = view === "records" || (view === "auto" && (rect?.width ?? window.innerWidth) < 560);
  const tooWide = data.columns > MAX_TABLE_COLUMNS;
  const showSource = source || parsed.malformed || tooWide;
  const columns = Array.from({ length: Math.min(MAX_TABLE_COLUMNS, data.columns) }, (_, index) => ({ index, label: data.header[index] || t(c.column, { number: index + 1 }), width: widths[index] ?? 180 }));
  const sortColumn = (index: number) => { setSort((current) => nextTableSort(current, index)); setPage(0); };
  const count = t(c.tableCount, { loaded: data.rows.length, columns: data.columns });
  const moveCell = (event: KeyboardEvent<HTMLButtonElement>, row: number, column: number) => {
    let nextRow = row, nextColumn = column;
    if (event.key === "ArrowDown") nextRow++;
    else if (event.key === "ArrowUp") nextRow--;
    else if (event.key === "ArrowRight") nextColumn += i18n.dir() === "rtl" ? -1 : 1;
    else if (event.key === "ArrowLeft") nextColumn += i18n.dir() === "rtl" ? 1 : -1;
    else if (event.key === "Home") { nextColumn = 0; if (event.ctrlKey || event.metaKey) nextRow = 0; }
    else if (event.key === "End") { nextColumn = columns.length - 1; if (event.ctrlKey || event.metaKey) nextRow = visibleRows.length - 1; }
    else return;
    event.preventDefault(); event.stopPropagation();
    nextRow = Math.max(0, Math.min(visibleRows.length - 1, nextRow));
    nextColumn = Math.max(0, Math.min(columns.length - 1, nextColumn));
    event.currentTarget.closest("table")?.querySelector<HTMLButtonElement>(`[data-cell-row="${String(nextRow)}"][data-cell-column="${String(nextColumn)}"]`)?.focus();
  };

  if (!content.trim()) return <ArtifactFrame testId="file-preview-empty"><div className="flex flex-1 items-center justify-center p-4 text-sm text-wc-text-muted">{t(strings.messagesFileViewer.emptyFile)}</div><ArtifactFooter model={model} metadata={formatBytes(model.sizeBytes)} /></ArtifactFrame>;

  return <ArtifactFrame testId="file-preview-csv">
    <div ref={ref} className="flex min-h-0 flex-1 flex-col">
      <div className="shrink-0 space-y-2 border-b border-wc-default bg-wc-surface-raised px-3 py-2">
        <div className="flex items-center gap-2">
          <label className="relative min-w-0 flex-1"><Search className="pointer-events-none absolute left-3 top-3.5 h-4 w-4 text-wc-text-muted" />
            <input type="search" aria-label={t(c.filterRows)} placeholder={t(c.filterRows)} value={query} disabled={showSource} onChange={(event) => { setQuery(event.target.value); setPage(0); setSelected(null); }}
              onKeyDown={(event) => { if (event.key === "Escape" && query) { event.stopPropagation(); setQuery(""); } }}
              className="h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input pl-9 pr-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-wc-accent disabled:opacity-40" />
          </label>
          <select aria-label={t(c.tableView)} value={view} disabled={showSource} onChange={(event) => { setView(event.target.value); setSelected(null); }} className={`${previewSelectClass} w-24`}>
            <option value="auto">{t(c.autoView)}</option><option value="table">{t(c.tableViewLabel)}</option><option value="records">{t(c.recordsView)}</option>
          </select>
          <PreviewButton label={t(source ? c.viewTable : c.viewSource)} pressed={source} onClick={() => { setSource(!source); setSelected(null); }}><Code2 /></PreviewButton>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-xs text-wc-text-muted">
          <span>{count}</span>
          <details className="group">
            <summary className="flex min-h-11 cursor-pointer list-none items-center gap-2 text-wc-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent [&::-webkit-details-marker]:hidden">{t(c.tableOptions)}<ChevronDown className="h-3.5 w-3.5 transition group-open:rotate-180" /></summary>
            <label className="flex min-h-11 items-center gap-2"><input type="checkbox" checked={header} onChange={(event) => { setHeader(event.target.checked); setSort(null); setPage(0); setSelected(null); }} />{t(c.firstRowHeader)}</label>
          </details>
        </div>
        {(data.truncated || text?.truncated) && <PreviewNotice message={t(c.loadedRowsOnly)} tone="info" />}
        {parsed.malformed && <PreviewNotice message={t(c.malformedCsv)} />}
        {tooWide && <PreviewNotice message={t(c.tooManyColumns, { count: MAX_TABLE_COLUMNS })} tone="info" />}
        {records && !showSource && !selected && <div className="flex items-center gap-1">
          <select aria-label={t(c.sortRows)} value={sort?.column ?? "none"} onChange={(event) => { setSort(event.target.value === "none" ? null : { column: Number(event.target.value), direction: "asc" }); setPage(0); }} className={`${previewSelectClass} flex-1`}>
            <option value="none">{t(c.originalOrder)}</option>{columns.map((column) => <option key={column.index} value={column.index}>{column.label}</option>)}
          </select>
          <PreviewButton label={t(sort?.direction === "desc" ? c.sortAscending : c.sortDescending)} disabled={!sort} onClick={() => { if (sort) setSort({ ...sort, direction: sort.direction === "asc" ? "desc" : "asc" }); }}>{sort?.direction === "desc" ? <ArrowDown /> : <ArrowUp />}</PreviewButton>
        </div>}
      </div>
      {showSource ? <div className="min-h-0 flex-1"><CodeLinePreview content={content} path={model.resolvedPath} highlightLine={null} truncated={text?.truncated} /></div>
        : records ? selected ? <div className="min-h-0 flex-1 overflow-auto p-3" data-testid={selectors.artifactPreview.tableRecord}>
          <button type="button" onClick={() => { setSelected(null); }} className="mb-3 inline-flex min-h-11 items-center gap-2 rounded-lg px-2 text-sm text-wc-text-secondary outline-none hover:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-wc-accent"><ArrowLeft className="h-4 w-4" />{t(c.backToRows)}</button>
          <h3 className="mb-3 text-base font-semibold">{t(c.rowNumber, { number: selected.index + 1 })}</h3>
          <dl className="divide-y divide-wc-default rounded-xl border border-wc-default bg-wc-surface-raised px-3">
            {columns.map((column) => <div key={column.index} className="flex items-start justify-between gap-2 py-3">
              <div className="min-w-0 flex-1"><dt className="mb-1 break-words text-xs text-wc-text-muted">{column.label}</dt><dd className="whitespace-pre-wrap break-words text-sm">{selected.cells[column.index] || "—"}</dd></div>
              <CopyIconButton value={selected.cells[column.index] ?? ""} writeText={copyText} aria-label={t(c.copyCell, { column: column.label, row: selected.index + 1 })} copiedLabel={t(strings.messagesFileViewer.copied)} failedLabel={t(c.copyFailed)} />
            </div>)}
          </dl>
        </div> : <div className="min-h-0 flex-1 overflow-auto p-3" data-testid={selectors.artifactPreview.tableRecords}>
          {visibleRows.length ? <ol className="space-y-2">{visibleRows.map((row) => <li key={row.index}>
            <button type="button" onClick={() => { setSelected(row); }} className="flex min-h-20 w-full items-center justify-between gap-2 rounded-xl border border-wc-default bg-wc-surface-raised p-3 text-left outline-none hover:border-wc-hover focus-visible:ring-2 focus-visible:ring-wc-accent" aria-label={t(c.openRow, { number: row.index + 1 })}>
              <span className="min-w-0 flex-1"><span className="mb-1 block text-[11px] text-wc-text-muted">{t(c.rowNumber, { number: row.index + 1 })}</span><span className="block truncate text-sm font-medium">{row.cells[0] || "—"}</span><span className="mt-1 block truncate text-xs text-wc-text-muted">{columns.slice(1, 3).map((column) => `${column.label}: ${row.cells[column.index] || "—"}`).join(" · ")}</span></span><ChevronRight className="h-4 w-4 shrink-0 text-wc-text-muted" />
            </button>
          </li>)}</ol> : <p role="status" className="p-6 text-center text-sm text-wc-text-muted">{t(c.noMatchingRows)}</p>}
        </div> : <div className="min-h-0 flex-1 overflow-auto" data-testid={selectors.artifactPreview.tableGrid}>
          <table className="table-fixed border-separate border-spacing-0 text-left text-xs" style={{ width: 48 + columns.reduce((sum, column) => sum + column.width, 0) }}>
            <caption className="sr-only">{model.basename}</caption>
            <colgroup><col style={{ width: 48 }} />{columns.map((column) => <col key={column.index} style={{ width: column.width }} />)}</colgroup>
            <thead className="sticky top-0 z-wc-chrome"><tr>
              <th scope="col" className="border-b border-r border-wc-default bg-wc-surface-raised p-2 text-center font-normal text-wc-text-muted">#</th>
              {columns.map((column) => <th key={column.index} scope="col" aria-sort={sort?.column === column.index ? sort.direction === "asc" ? "ascending" : "descending" : "none"} className="relative border-b border-r border-wc-default bg-wc-surface-raised pr-5">
                <button type="button" onClick={() => { sortColumn(column.index); }} className="flex min-h-11 w-full items-center gap-2 px-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent" title={t(c.sortColumn, { column: column.label })}><span className="min-w-0 flex-1 truncate">{column.label}</span>{sort?.column === column.index ? sort.direction === "asc" ? <ArrowUp className="h-3.5 w-3.5 shrink-0" /> : <ArrowDown className="h-3.5 w-3.5 shrink-0" /> : <ArrowUpDown className="h-3 w-3 shrink-0 text-wc-text-muted" />}</button>
                <div role="separator" aria-orientation="vertical" aria-label={t(c.resizeColumn, { column: column.label })} aria-valuemin={100} aria-valuemax={640} aria-valuenow={column.width} tabIndex={0}
                  className="absolute right-0 top-0 h-full w-5 cursor-col-resize border-r-2 border-transparent outline-none hover:border-wc-accent focus-visible:border-wc-accent" style={{ touchAction: "none" }}
                  onPointerDown={(event) => { event.preventDefault(); event.currentTarget.setPointerCapture(event.pointerId); resize.current = { column: column.index, x: event.clientX, width: column.width }; }}
                  onPointerMove={(event) => { const start = resize.current; if (start?.column === column.index) setWidths((current) => ({ ...current, [column.index]: columnWidth(start.width + event.clientX - start.x) })); }}
                  onPointerUp={() => { resize.current = null; }} onPointerCancel={() => { resize.current = null; }} onLostPointerCapture={() => { resize.current = null; }}
                  onKeyDown={(event) => { if (event.key === "ArrowLeft" || event.key === "ArrowRight") { event.preventDefault(); event.stopPropagation(); setWidths((current) => ({ ...current, [column.index]: columnWidth(column.width + (event.key === "ArrowRight" ? 16 : -16)) })); } }} />
              </th>)}
            </tr></thead>
            <tbody>{visibleRows.map((row, rowIndex) => <tr key={row.index} className="odd:bg-wc-surface-input/20 hover:bg-wc-surface-input/60">
              <th scope="row" className="border-b border-r border-wc-default px-2 text-center font-mono font-normal text-wc-text-muted">{row.index + 1}</th>
              {columns.map((column) => <td key={column.index} className="border-b border-r border-wc-default">
                <button type="button" onClick={() => { void clipboard.copy(row.cells[column.index] ?? ""); }} title={row.cells[column.index] ?? ""} aria-label={t(c.copyCell, { column: column.label, row: row.index + 1 })}
                  data-cell-row={rowIndex} data-cell-column={column.index}
                  tabIndex={rowIndex === Math.min(activeCell.row, visibleRows.length - 1) && column.index === Math.min(activeCell.column, columns.length - 1) ? 0 : -1}
                  onFocus={() => { setActiveCell({ row: rowIndex, column: column.index }); }} onKeyDown={(event) => { moveCell(event, rowIndex, column.index); }}
                  className="block min-h-11 w-full truncate px-3 text-left outline-none hover:bg-wc-accent/10 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent">{row.cells[column.index] || "—"}</button>
              </td>)}
            </tr>)}</tbody>
          </table>
          {!visibleRows.length && <p role="status" className="p-6 text-sm text-wc-text-muted">{t(c.noMatchingRows)}</p>}
        </div>}
      {!showSource && !selected && <div className="flex shrink-0 flex-wrap items-center justify-between gap-1 border-t border-wc-default px-3 text-xs text-wc-text-muted">
        <span role="status" title={!records ? t(c.cellNavigation) : undefined}>{clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : c.copyFailed) : t(c.matchingRows, { count: rows.length })}</span>
        <div className="flex items-center"><PreviewButton label={t(c.previousPage)} disabled={currentPage <= 0} onClick={() => { setPage(currentPage - 1); }}><ChevronLeft /></PreviewButton><span className="whitespace-nowrap">{t(c.tablePage, { page: currentPage + 1, total: pages })}</span><PreviewButton label={t(c.nextPage)} disabled={currentPage >= pages - 1} onClick={() => { setPage(currentPage + 1); }}><ChevronRight /></PreviewButton></div>
      </div>}
    </div>
    <ArtifactFooter model={model} metadata={`${model.resolvedPath.toLowerCase().endsWith(".tsv") ? "TSV" : "CSV"} · ${formatBytes(model.sizeBytes)}`} />
  </ArtifactFrame>;
}
