import { useDeferredValue, useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { ArrowLeft, ChevronLeft, ChevronRight, Copy, Hash, Search, Sheet, Sigma } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { cn } from "../../../lib/classnames";
import { formatBytes } from "../format";
import { cellCoordinates, columnName, workbookRows, WORKBOOK_PAGE_SIZE, type WorkbookCell, type WorkbookData } from "../workbookData";
import { useWorkbook } from "../useWorkbook";
import { usePreviewClipboard } from "../usePreviewClipboard";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, PreviewButton, PreviewLoading, InspectionHeader, InspectionOptions, InspectionSetting, InspectionChoices, InspectionJump, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";

export function SpreadsheetPreview(props: PreviewRendererProps) { return <WorkbookReader key={props.model.previewId} {...props} />; }
function WorkbookReader({ model }: PreviewRendererProps) {
  const { t } = useTranslation(), c = strings.inspectionPreview;
  const [attempt, setAttempt] = useState(0), result = useWorkbook(model.blobHref, model.sizeBytes, attempt);
  return <ArtifactFrame testId={selectors.inspectionPreview.workbook}>
    {!result ? <PreviewLoading label={t(c.loadingWorkbook)} /> : result.data ? <WorkbookInspector key={attempt} data={result.data} name={model.basename} />
      : <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-auto px-5 py-8 text-center">
        <Sheet className="h-9 w-9 text-wc-text-muted" aria-hidden="true" />
        <p role="status" className="max-w-sm text-sm leading-relaxed text-wc-text-secondary">{t(result.error === "encrypted" ? c.workbookEncrypted : result.error === "limit" ? c.workbookTooLarge : result.error === "unavailable" ? c.workbookUnavailable : result.error === "fetch" ? c.workbookFetch : c.workbookError)}</p>
        <button type="button" className={previewTextButtonClass} onClick={() => { setAttempt((current) => current + 1); }}>{t(c.retry)}</button>
      </div>}
    <ArtifactFooter model={model} metadata={`${model.basename.split(".").pop()?.toUpperCase() ?? "Excel"} · ${formatBytes(model.sizeBytes)}`} />
  </ArtifactFrame>;
}
function WorkbookInspector({ data, name }: { data: WorkbookData; name: string }) {
  const { t } = useTranslation(), c = strings.inspectionPreview;
  const [sheetIndex, setSheetIndex] = useState(() => Math.max(0, data.sheets.findIndex((sheet) => sheet.visibility === 0)));
  const sheet = data.sheets[sheetIndex] ?? data.sheets[0];
  if (!sheet) throw new Error("Workbook has no sheets");
  const [query, setQuery] = useState(""), [searchOpen, setSearchOpen] = useState(false), [address, setAddress] = useState(""), [invalid, setInvalid] = useState(false);
  const [page, setPage] = useState(0), [view, setView] = useState("auto"), [selected, setSelected] = useState<WorkbookCell | null>(() => sheet.cells[0] ?? null), [selectedRow, setSelectedRow] = useState<number | null>(null), [mobileCell, setMobileCell] = useState(false);
  const [active, setActive] = useState({ row: 0, column: 0 });
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), frame = useRef<HTMLDivElement>(null), focusFrame = useRef(0);
  const width = rect?.width ?? 0, records = view === "records" || view === "auto" && width < 560, wide = width >= 900;
  const deferredQuery = useDeferredValue(query), rows = useMemo(() => workbookRows(sheet, deferredQuery), [sheet, deferredQuery]);
  const cellMap = useMemo(() => new Map(sheet.cells.map((cell) => [cell.address, cell])), [sheet]);
  const pages = Math.max(1, Math.ceil(rows.length / WORKBOOK_PAGE_SIZE)), currentPage = Math.min(page, pages - 1), visible = rows.slice(currentPage * WORKBOOK_PAGE_SIZE, (currentPage + 1) * WORKBOOK_PAGE_SIZE);
  const columns = Array.from({ length: sheet.columns }, (_, index) => sheet.startColumn + index);
  const clipboard = usePreviewClipboard();
  const cellAt = (row: number, column: number): WorkbookCell => {
    const cellAddress = `${columnName(column)}${String(row + 1)}`;
    return cellMap.get(cellAddress) ?? { address: cellAddress, row, column, display: "", raw: "", type: "z", missingValue: true };
  };
  useEffect(() => () => { cancelAnimationFrame(focusFrame.current); }, []);
  useEffect(() => {
    frame.current?.querySelectorAll<HTMLElement>(`[data-testid="${selectors.inspectionPreview.workbookGrid}"], [data-testid="${selectors.inspectionPreview.workbookRows}"]`).forEach((surface) => { surface.scrollTop = 0; surface.scrollLeft = 0; });
  }, [sheetIndex, currentPage, deferredQuery, records]);
  useEffect(() => { if (records && selectedRow !== null) frame.current?.querySelector<HTMLButtonElement>("[data-workbook-back]")?.focus({ preventScroll: true }); }, [records, selectedRow, mobileCell]);
  const restore = (selector: string) => { cancelAnimationFrame(focusFrame.current); focusFrame.current = requestAnimationFrame(() => { const target = frame.current?.querySelector<HTMLButtonElement>(selector); target?.focus({ preventScroll: true }); target?.scrollIntoView({ block: "nearest", inline: "nearest" }); }); };
  const resetSheet = (index: number) => { setSheetIndex(index); setQuery(""); setPage(0); setSelected(data.sheets[index]?.cells[0] ?? null); setSelectedRow(null); setMobileCell(false); setAddress(""); setInvalid(false); setActive({ row: 0, column: 0 }); };
  const inspect = (cell: WorkbookCell) => { setSelected(cell); setMobileCell(true); if (records) setSelectedRow(cell.row); };
  const go = () => {
    const target = cellCoordinates(address);
    if (!target || target.row < sheet.startRow || target.row >= sheet.startRow + sheet.rows || target.column < sheet.startColumn || target.column >= sheet.startColumn + sheet.columns) { setInvalid(true); return false; }
    setInvalid(false); setQuery(""); setPage(Math.floor((target.row - sheet.startRow) / WORKBOOK_PAGE_SIZE)); setSelected(cellAt(target.row, target.column));
    setActive({ row: (target.row - sheet.startRow) % WORKBOOK_PAGE_SIZE, column: target.column - sheet.startColumn });
    if (records) { setSelectedRow(target.row); setMobileCell(true); }
    else { setMobileCell(true); restore(`[data-workbook-cell="${columnName(target.column)}${String(target.row + 1)}"]`); }
    return true;
  };
  const keyDown = (event: KeyboardEvent<HTMLButtonElement>, row: number, column: number) => {
    let r = row, col = column;
    if (event.key === "ArrowDown") r++; else if (event.key === "ArrowUp") r--; else if (event.key === "ArrowRight") col++; else if (event.key === "ArrowLeft") col--;
    else if (event.key === "Home") { col = 0; if (event.ctrlKey || event.metaKey) r = 0; }
    else if (event.key === "End") { col = columns.length - 1; if (event.ctrlKey || event.metaKey) r = visible.length - 1; }
    else return;
    event.preventDefault(); event.stopPropagation();
    r = Math.max(0, Math.min(visible.length - 1, r)); col = Math.max(0, Math.min(columns.length - 1, col));
    const cellAddress = `${columnName(columns[col] ?? sheet.startColumn)}${String((visible[r] ?? sheet.startRow) + 1)}`;
    frame.current?.querySelector<HTMLButtonElement>(`[data-workbook-cell="${cellAddress}"]`)?.focus();
  };
  const range = sheet.rows && sheet.columns ? `${columnName(sheet.startColumn)}${String(sheet.startRow + 1)}:${columnName(sheet.startColumn + sheet.columns - 1)}${String(sheet.startRow + sheet.rows)}` : "—";
  const detailVisible = !records && selected || records && mobileCell && selected;
  return <div ref={frame} className="flex min-h-0 min-w-0 flex-1 flex-col">
    <span role="status" className="sr-only">{clipboard.status ? t(clipboard.status === "copied" ? strings.messagesFileViewer.copied : strings.artifactPreview.copyFailed) : ""}</span>
    <div ref={ref} className="flex min-h-0 min-w-0 flex-1 flex-col">
      <InspectionHeader icon={<Sheet />} title={t(c.workbook)} description={`${t(c.sheetCount, { loaded: data.sheets.length, total: data.totalSheets })} · ${t(c.readOnly)}`}>
        <PreviewButton label={t(c.findCells)} pressed={searchOpen} onClick={() => { setSearchOpen(!searchOpen); setSelectedRow(null); setMobileCell(false); if (searchOpen) setQuery(""); else restore("[data-workbook-search]"); }}><Search /></PreviewButton>
        <InspectionOptions label={t(c.workbookOptions)}>{(close) => <>
          <InspectionSetting label={t(c.workbookView)}><InspectionChoices label={t(c.workbookView)} value={view} onChange={(value) => { setView(value); setSelectedRow(null); setMobileCell(false); }} options={[{ value: "auto", label: t(strings.artifactPreview.autoView) }, { value: "grid", label: t(strings.artifactPreview.tableViewLabel) }, { value: "records", label: t(strings.artifactPreview.recordsView) }]} /></InspectionSetting>
          <div className="border-t border-wc-default pt-5"><InspectionJump label={t(c.goCell)} value={address} invalid={invalid} placeholder="A1" onChange={(value) => { setAddress(value); setInvalid(false); }} onGo={() => { if (go()) close(); }} />
            {invalid && <p role="alert" className="pt-2 text-xs text-wc-danger-text">{t(c.cellInvalid)}</p>}
          </div>
        </>}</InspectionOptions>
      </InspectionHeader>
      {width < 560 && <div className="shrink-0 border-b border-wc-default px-3 py-1.5"><select aria-label={t(c.sheet)} value={sheetIndex} className={`${previewSelectClass} wc-inspection-picker w-full font-medium`} onChange={(event) => { resetSheet(Number(event.target.value)); }}>
        {data.sheets.map((entry, index) => <option key={index} value={index}>{entry.name}{entry.visibility ? ` · ${t(entry.visibility === 2 ? c.veryHiddenSheet : c.hiddenSheet)}` : ""}</option>)}
      </select></div>}
      {searchOpen && <div className="shrink-0 border-b border-wc-default px-3 py-1"><input data-workbook-search type="search" value={query} onChange={(event) => { setQuery(event.target.value); setPage(0); setSelectedRow(null); setMobileCell(false); }} aria-label={t(c.findCells)} placeholder={t(c.findCells)}
        onKeyDown={(event) => { if (event.key === "Escape") { event.stopPropagation(); setQuery(""); setSearchOpen(false); frame.current?.querySelector<HTMLButtonElement>(`button[aria-label="${CSS.escape(t(c.findCells))}"]`)?.focus(); } }}
        className="h-11 w-full rounded-lg bg-wc-surface-input px-3 text-sm outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" /></div>}
      {data.limited && <details className="shrink-0 border-b border-wc-default px-3 py-1 text-xs text-wc-text-secondary"><summary className="flex min-h-11 cursor-pointer items-center">{t(c.partialPreview)} · {t(c.loadedRange, { range })}</summary><p className="pb-2 leading-relaxed">{t(c.workbookLimits)}</p></details>}
      <div className={cn("flex min-h-0 min-w-0 flex-1", !wide && !records && "flex-col")}>
        {records && selectedRow !== null ? <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          <div className="flex shrink-0 items-center gap-2 border-b border-wc-default px-2"><button type="button" data-workbook-back className={`${previewTextButtonClass} my-1 border-transparent`} onClick={() => {
            if (mobileCell) { setMobileCell(false); restore(`[data-workbook-row-cell="${selected?.address ?? ""}"]`); }
            else { const previous = selectedRow; setSelectedRow(null); restore(`[data-workbook-row="${String(previous)}"]`); }
          }}><ArrowLeft className="h-4 w-4 shrink-0" aria-hidden="true" />{t(mobileCell ? c.rowDetails : c.backToSheet, { number: selectedRow + 1 })}</button></div>
          {detailVisible ? <CellInspector cell={selected} onCopy={(value) => { void clipboard.copy(value); }} /> : <div className="min-h-0 flex-1 overflow-auto p-3" data-testid={selectors.inspectionPreview.workbookDetail}>
            <h3 className="mb-3 text-sm font-semibold">{t(c.rowDetails, { number: selectedRow + 1 })}</h3>
            <ol className="divide-y divide-wc-default rounded-xl border border-wc-default bg-wc-surface-raised px-3">{columns.map((column) => { const cell = cellAt(selectedRow, column); return <li key={column}><button type="button" data-workbook-row-cell={cell.address} onClick={() => { inspect(cell); }} className="flex min-h-16 w-full items-center gap-3 rounded-lg py-3 text-start outline-none focus-visible:ring-2 focus-visible:ring-wc-accent">
              <span dir="ltr" className="w-12 shrink-0 font-mono text-xs text-wc-text-muted">{cell.address}</span><span className="min-w-0 flex-1 whitespace-pre-wrap text-sm [overflow-wrap:anywhere]">{cellLabel(cell, t(c.blankCell), t(c.noSavedValue), t(c.omittedCell))}</span><ChevronRight className="h-4 w-4 shrink-0 text-wc-text-muted" aria-hidden="true" /></button></li>; })}</ol>
          </div>}
        </div> : records ? <div className="min-h-0 flex-1 overflow-auto p-3" data-testid={selectors.inspectionPreview.workbookRows}>
          <ol className="space-y-2">{visible.map((row) => { const populated = sheet.cells.filter((cell) => cell.row === row).slice(0, 3); return <li key={row}><button type="button" data-workbook-row={row} onClick={() => { setSelectedRow(row); setMobileCell(false); }} className="flex min-h-20 w-full items-center gap-3 rounded-xl border border-wc-default bg-wc-surface-raised p-3 text-start outline-none hover:border-wc-hover focus-visible:ring-2 focus-visible:ring-wc-accent">
            <span className="min-w-0 flex-1"><span className="mb-1 block text-[11px] text-wc-text-muted">{t(c.rowDetails, { number: row + 1 })}</span><span className="block truncate text-sm font-medium">{populated[0] ? cellLabel(populated[0], t(c.blankCell), t(c.noSavedValue), t(c.omittedCell)) : "—"}</span><span className="mt-1 block truncate text-xs text-wc-text-muted">{populated.slice(1).map((cell) => `${columnName(cell.column)}: ${cellLabel(cell, t(c.blankCell), t(c.noSavedValue), t(c.omittedCell))}`).join(" · ")}</span></span><ChevronRight className="h-4 w-4 shrink-0 text-wc-text-muted" aria-hidden="true" />
          </button></li>; })}</ol>
          {!visible.length && <p role="status" className="p-5 text-center text-sm text-wc-text-muted">{t(query ? strings.artifactPreview.noMatchingRows : c.emptySheet)}</p>}
        </div> : <div dir="ltr" role="region" aria-label={t(c.sheet)} tabIndex={0} className="m-3 min-h-0 min-w-0 flex-1 overflow-auto rounded-xl border border-wc-default bg-wc-surface-base shadow-sm" data-testid={selectors.inspectionPreview.workbookGrid}>
          <table dir="ltr" className="table-fixed border-separate border-spacing-0 text-xs" style={{ width: 48 + sheet.columns * 160 }}>
            <caption className="sr-only">{name} · {sheet.name}</caption><colgroup><col style={{ width: 48 }} />{columns.map((column) => <col key={column} style={{ width: 160 }} />)}</colgroup>
            <thead className="sticky top-0 z-20"><tr><th scope="col" className="sticky left-0 z-20 border-b border-e border-wc-default bg-wc-surface-raised p-3 font-normal text-wc-text-muted">#</th>{columns.map((column) => <th key={column} scope="col" className="border-b border-e border-wc-default bg-wc-surface-raised p-3 font-mono font-normal text-wc-text-muted">{columnName(column)}</th>)}</tr></thead>
            <tbody>{visible.map((row, r) => <tr key={row} className="odd:bg-wc-surface-input/20"><th scope="row" className="sticky left-0 z-10 border-b border-e border-wc-default bg-wc-surface-raised px-2 font-mono font-normal text-wc-text-muted">{row + 1}</th>{columns.map((column, col) => {
              const cell = cellAt(row, column);
              return <td key={column} className="border-b border-e border-wc-default"><button type="button" data-workbook-cell={cell.address} aria-label={t(c.openCell, { address: cell.address })} aria-pressed={selected?.address === cell.address}
                tabIndex={r === Math.min(active.row, visible.length - 1) && col === Math.min(active.column, columns.length - 1) ? 0 : -1}
                onFocus={() => { setActive({ row: r, column: col }); setSelected(cell); }} onKeyDown={(event) => { keyDown(event, r, col); }} onClick={() => { inspect(cell); }} title={cell.display}
                className={cn("block min-h-11 w-full truncate px-3 outline-none hover:bg-wc-accent/10 focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent", cell.type === "n" ? "text-end tabular-nums" : "text-start", selected?.address === cell.address && "bg-wc-accent/10")}>
                {cell.omitted ? "…" : cell.missingValue ? cell.formula ? "ƒ" : "\u00a0" : cell.display || '""'}
              </button></td>;
            })}</tr>)}</tbody>
          </table>
          {!visible.length && <p role="status" className="p-5 text-sm text-wc-text-muted">{t(query ? strings.artifactPreview.noMatchingRows : c.emptySheet)}</p>}
        </div>}
        {!records && (wide || mobileCell) && <aside aria-label={t(c.cellDetails)} className={cn("min-h-0 min-w-0 overflow-auto border-wc-default bg-wc-surface-raised", wide ? "my-3 me-3 w-72 shrink-0 rounded-xl border" : "mx-3 mb-3 max-h-[45%] shrink-0 rounded-xl border")}>
          {!wide && <button type="button" className={`${previewTextButtonClass} m-2 border-transparent`} onClick={() => { setMobileCell(false); if (selected) restore(`[data-workbook-cell="${selected.address}"]`); }}><ArrowLeft className="h-4 w-4" aria-hidden="true" />{t(c.backToSheet)}</button>}
          {selected ? <CellInspector cell={selected} onCopy={(value) => { void clipboard.copy(value); }} /> : <div className="p-5 text-sm leading-relaxed text-wc-text-muted">{t(c.chooseCell)}</div>}
        </aside>}
      </div>
      {(!records || selectedRow === null) && <div className="flex shrink-0 flex-wrap items-center justify-between gap-1 border-t border-wc-default px-3 text-xs text-wc-text-muted">
        {width >= 560 ? <nav aria-label={t(c.sheet)} className="min-w-0 flex-1 overflow-x-auto"><div className="flex w-max items-center gap-1 py-1">{data.sheets.map((entry, index) => <button key={index} type="button" aria-pressed={index === sheetIndex} onClick={() => { resetSheet(index); }} className={cn("inline-flex min-h-11 max-w-52 items-center gap-2 rounded-lg px-3 text-xs outline-none transition focus-visible:ring-2 focus-visible:ring-wc-accent", index === sheetIndex ? "bg-wc-accent/10 font-semibold text-wc-accent" : "text-wc-text-muted hover:bg-wc-surface-input hover:text-wc-text-primary")}><Sheet className="h-3.5 w-3.5 shrink-0" aria-hidden="true" /><span className="truncate">{entry.name}</span>{entry.visibility > 0 && <span className="sr-only">{t(entry.visibility === 2 ? c.veryHiddenSheet : c.hiddenSheet)}</span>}</button>)}</div></nav> : <span role="status">{t(strings.artifactPreview.matchingRows, { count: rows.length })}</span>}<div className="flex items-center"><PreviewButton label={t(strings.artifactPreview.previousPage)} disabled={!currentPage} onClick={() => { setPage(currentPage - 1); }}><ChevronLeft /></PreviewButton><span>{t(strings.artifactPreview.tablePage, { page: currentPage + 1, total: pages })}</span><PreviewButton label={t(strings.artifactPreview.nextPage)} disabled={currentPage >= pages - 1} onClick={() => { setPage(currentPage + 1); }}><ChevronRight /></PreviewButton></div>
      </div>}
      <div className="shrink-0 border-t border-wc-default px-3 py-2 text-[11px] leading-relaxed text-wc-text-muted"><span title={t(c.sheetCount, { loaded: data.sheets.length, total: data.totalSheets })}>{sheet.range || "—"}</span><span className="mx-2">·</span>{t(c.formulaNotice)}</div>
    </div>
  </div>;
}
function cellLabel(cell: WorkbookCell, blank: string, missing: string, omitted: string) { return cell.omitted ? omitted : cell.missingValue ? cell.formula ? missing : blank : cell.display || '""'; }
function CellInspector({ cell, onCopy }: { cell: WorkbookCell; onCopy: (value: string) => void }) {
  const { t } = useTranslation(), c = strings.inspectionPreview;
  const type = cell.omitted ? c.omittedType : cell.type === "n" ? c.numberType : cell.type === "b" ? c.booleanType : cell.type === "e" ? c.errorType : cell.type === "z" ? c.blankType : c.stringType;
  return <div className="h-full min-h-0 overflow-auto p-3" data-testid={selectors.inspectionPreview.workbookDetail}>
    <div className="mb-3 flex items-center justify-between gap-2"><h3 dir="ltr" className="font-mono text-sm font-semibold">{cell.address}</h3><span className="rounded-md border border-wc-default bg-wc-surface-input/50 px-2 py-1 text-[10px] text-wc-text-muted">{t(type)}</span></div>
    <div className="mb-4 flex items-center gap-1 border-b border-wc-default pb-3"><PreviewButton label={t(c.copyValue)} disabled={cell.missingValue} onClick={() => { onCopy(cell.raw); }}><Copy /></PreviewButton><PreviewButton label={t(c.copyAddress)} onClick={() => { onCopy(cell.address); }}><Hash /></PreviewButton>{cell.formula && <PreviewButton label={t(c.copyFormula)} onClick={() => { if (cell.formula) onCopy(cell.formula); }}><Sigma /></PreviewButton>}</div>
    <dl className="space-y-4 text-sm"><div><dt className="mb-1 text-xs text-wc-text-muted">{t(c.savedValue)}</dt><dd className="whitespace-pre-wrap text-lg font-semibold leading-relaxed tracking-tight [overflow-wrap:anywhere]">{cellLabel(cell, t(c.blankCell), t(c.noSavedValue), t(c.omittedCell))}</dd></div>
      {!cell.missingValue && cell.raw !== cell.display && <div><dt className="mb-1 text-xs text-wc-text-muted">{t(c.rawValue)}</dt><dd dir="ltr" className="whitespace-pre-wrap font-mono text-xs [overflow-wrap:anywhere]">{cell.raw || '""'}</dd></div>}
      {cell.formula && <div><dt className="mb-1 text-xs text-wc-text-muted">{t(c.formula)}</dt><dd dir="ltr" className="whitespace-pre-wrap rounded-lg border border-wc-default bg-wc-surface-input p-3 font-mono text-xs [overflow-wrap:anywhere]">{cell.formula}</dd></div>}
      <div><dt className="mb-1 text-xs text-wc-text-muted">{t(c.cellType)}</dt><dd>{t(type)}</dd></div>
      {cell.merge && <div><dt className="mb-1 text-xs text-wc-text-muted">{t(c.mergedRange)}</dt><dd dir="ltr" className="font-mono text-xs">{cell.merge}</dd></div>}
    </dl>
  </div>;
}
