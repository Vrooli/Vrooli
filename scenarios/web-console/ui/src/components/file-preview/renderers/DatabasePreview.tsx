import { useEffect, useId, useRef, useState } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { ArrowDown, ArrowLeft, ArrowUp, ChevronLeft, ChevronRight, Database, Download, KeyRound, RefreshCw, Search } from "lucide-react";
import { useTranslation } from "react-i18next";
import { strings } from "../../../consts/strings";
import { selectors } from "../../../consts/selectors";
import { DATABASE_MAX_PAGE, DATABASE_PAGE_SIZE, databaseCellText, databaseCsv, databaseDetail, type DatabaseCell, type DatabaseQuery, type DatabaseSchema } from "../databaseData";
import { useDatabasePreview } from "../useDatabasePreview";
import { usePreviewClipboard } from "../usePreviewClipboard";
import { formatBytes } from "../format";
import type { PreviewRendererProps } from "../types";
import { ArtifactFooter, ArtifactFrame, InspectionChoices, InspectionHeader, InspectionOptions, InspectionSetting, PreviewButton, PreviewLoading, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
const initialQuery: DatabaseQuery = { table: "", page: 0, filterColumn: "", filter: "", sortColumn: "", descending: false };
const inputClass = "h-11 min-w-0 flex-1 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-base text-wc-text-primary outline-none placeholder:text-wc-text-muted focus-visible:ring-2 focus-visible:ring-wc-accent";
export function DatabasePreview(props: PreviewRendererProps) {
  return <DatabaseBrowser key={props.model.previewId} {...props} />;
}
function DatabaseBrowser({ model, onReopen }: PreviewRendererProps) {
  const { t, i18n } = useTranslation(), c = strings.databasePreview;
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 640;
  const [attempt, setAttempt] = useState(0), [mode, setMode] = useState("data"), [tableSearch, setTableSearch] = useState("");
  const [query, setQuery] = useState<DatabaseQuery>(initialQuery), [record, setRecord] = useState<number | null>(null);
  const [selected, setSelected] = useState<{
    cell: DatabaseCell;
    column: string;
  } | null>(null);
  const reader = useDatabasePreview(model.blobHref, model.sizeBytes, attempt), clipboard = usePreviewClipboard();
  const backRef = useRef<HTMLButtonElement>(null), tableSelect = useRef<HTMLSelectElement>(null), focusReturn = useRef<HTMLButtonElement | null>(null);
  const host = useRef<HTMLDivElement>(null), download = useRef<{
    url: string;
    timer: ReturnType<typeof setTimeout>;
  } | null>(null);
  const filterId = useId();
  const browse = reader.browse;
  const page = reader.page && JSON.stringify(reader.page.query) === JSON.stringify(query) ? reader.page : null;
  const schema = reader.page?.query.table === query.table ? reader.page.schema : undefined, rows = page?.rows ?? [], info = reader.info;
  const tables = info?.tables.filter(name => name.toLocaleLowerCase().includes(tableSearch.toLocaleLowerCase())) ?? [];
  useEffect(() => {
    if (info)
      setQuery({ ...initialQuery, table: info.tables[0] ?? "" });
  }, [info]);
  useEffect(() => {
    if (!info || !query.table)
      return;
    const timer = setTimeout(() => {
      browse(query);
    }, 200);
    return () => {
      clearTimeout(timer);
    };
  }, [info, query, browse]);
  useEffect(() => {
    setRecord(null);
    setSelected(null);
    host.current?.scrollTo(0, 0);
  }, [query, mode]);
  useEffect(() => {
    if (record !== null)
      backRef.current?.focus({ preventScroll: true });
  }, [record]);
  useEffect(() => () => {
    if (download.current) {
      clearTimeout(download.current.timer);
      URL.revokeObjectURL(download.current.url);
    }
  }, []);
  const update = (next: Partial<DatabaseQuery>) => {
    setQuery(value => ({ ...value, ...next, page: next.page ?? 0 }));
  };
  const refresh = () => {
    if (onReopen)
      onReopen();
    else
      setAttempt(n => n + 1);
  };
  const inspect = (cell: DatabaseCell, column: string, trigger: HTMLButtonElement) => {
    focusReturn.current = trigger;
    setSelected({ cell, column });
  };
  const closeRecord = () => {
    const index = record;
    setRecord(null);
    requestAnimationFrame(() => {
      const trigger = host.current?.querySelector<HTMLButtonElement>(`[data-database-record="${String(index)}"]`);
      (trigger ?? tableSelect.current)?.focus({ preventScroll: true });
    });
  };
  const exportPage = () => {
    if (!page)
      return;
    if (download.current) {
      clearTimeout(download.current.timer);
      URL.revokeObjectURL(download.current.url);
    }
    const url = URL.createObjectURL(new Blob([databaseCsv(page)], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = `${model.basename.replace(/\.[^.]+$/, "")}-${query.table.replace(/[^\p{L}\p{N}_-]/gu, "_").slice(0, 60)}-${String(query.page + 1)}.csv`;
    document.body.append(link);
    link.click();
    link.remove();
    download.current = {
      url, timer: setTimeout(() => {
        URL.revokeObjectURL(url);
        download.current = null;
      }, 1000)
    };
  };
  const snapshotTime = info ? new Date(info.snapshotTime) : null;
  const snapshotLabel = snapshotTime && Number.isFinite(snapshotTime.getTime()) ? snapshotTime.toLocaleTimeString(i18n.resolvedLanguage === "cimode" ? "en" : i18n.resolvedLanguage, { hour: "2-digit", minute: "2-digit" }) : "—";
  const cellLabel = (cell: DatabaseCell) => cell.type === "null" ? t(c.nullValue) : cell.type === "blob" ? t(c.binary, { size: formatBytes(cell.bytes) }) : cell.value === "" ? t(c.emptyValue) : databaseCellText(cell);
  const pending = Boolean(info?.tables.length && !page && !reader.queryError);
  return <ArtifactFrame testId={selectors.databasePreview.root} frameRef={ref}>
    <InspectionHeader icon={<Database />} title={t(c.title)} description={info ? t(c.snapshot, { time: snapshotLabel }) : t(c.description)}>
      <PreviewButton label={t(c.refresh)} onClick={refresh}>
        <RefreshCw />
      </PreviewButton>
      <InspectionOptions label={t(c.details)}>{close => <>
        <InspectionSetting label={t(c.tables)}>
          <input type="search" aria-label={t(c.searchTables)} value={tableSearch} maxLength={256} onChange={e => {
            setTableSearch(e.target.value);
          }} className={`${inputClass} w-full`} />
        </InspectionSetting>
        <div className="max-h-48 overflow-y-auto overscroll-contain" role="group" aria-label={t(c.tables)}>{tables.length ? tables.map(name => <button key={name} className="block min-h-11 w-full truncate rounded-lg px-3 text-start text-sm text-wc-text-secondary outline-none hover:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-wc-accent aria-pressed:text-wc-accent" aria-pressed={query.table === name} title={name} onClick={() => {
          setQuery({ ...initialQuery, table: name });
          setTableSearch("");
          close();
        }}>{name}</button>) : <p role="status" className="text-xs text-wc-text-muted">{t(c.noTableMatch)}</p>}</div>
        <InspectionSetting label={t(c.sort)}>
          <select className={`${previewSelectClass} w-full`} aria-label={t(c.sort)} value={query.sortColumn} onChange={e => {
            update({ sortColumn: e.target.value });
          }}>
            <option value="">{t(c.natural)}</option>{schema?.columns.map(col => <option key={col.name} value={col.name}>{col.name}</option>)}</select>
        </InspectionSetting>
        <InspectionChoices label={t(c.sort)} value={query.descending ? "desc" : "asc"} options={[{ value: "asc", label: t(c.ascending) }, { value: "desc", label: t(c.descending) }]} onChange={value => {
          update({ descending: value === "desc", sortColumn: query.sortColumn || schema?.columns[0]?.name || "" });
        }} />
        <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.ordinaryOnly)} · {formatBytes(info?.bytes ?? model.sizeBytes)}</p>
        <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.exportHint)}</p>
      </>}</InspectionOptions>
    </InspectionHeader>
    {reader.error ? <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-y-auto p-6 text-center">
      <Database aria-hidden="true" className="h-9 w-9 text-wc-text-muted" />
      <p role="status" className="max-w-md text-sm leading-relaxed text-wc-text-secondary">{t(c[reader.error])}</p>
      <button className={previewTextButtonClass} onClick={refresh}>{t(strings.messagesFileViewer.reopen)}</button>
    </div>
      : !info ? <div className="min-h-0 flex-1">
        <PreviewLoading label={t(c.loading)} />
      </div>
        : !info.tables.length ? <div className="flex min-h-0 flex-1 items-center justify-center overflow-auto p-6 text-center text-sm leading-relaxed text-wc-text-muted">{t(c.noTables)}</div>
          : <>
            <div className="flex shrink-0 items-center justify-between gap-2 border-b border-wc-default px-3 py-2">
              <select ref={tableSelect} data-testid={selectors.databasePreview.tables} className={`${previewSelectClass} max-w-[55%] flex-1 font-medium`} value={query.table} aria-label={t(c.tables)} onChange={e => {
                setQuery({ ...initialQuery, table: e.target.value });
              }}>
                {info.tables.map(name => <option key={name} value={name}>{name}</option>)}
              </select>
              <div className="flex shrink-0 gap-1" role="group" aria-label={t(c.mode)}>{["data", "schema"].map(value => <button key={value} className={`${previewTextButtonClass} border-transparent px-3 aria-pressed:bg-wc-surface-input aria-pressed:text-wc-text-primary`} aria-pressed={mode === value} onClick={() => {
                setMode(value);
              }}>{t(value === "data" ? c.data : c.schema)}</button>)}</div>
            </div>
            
            {mode === "data" && record === null && <div className="flex shrink-0 gap-2 border-b border-wc-default px-3 py-2">
              <select aria-label={t(c.filterColumn)} className={`${previewSelectClass} w-28 shrink-0 sm:w-36`} value={query.filterColumn} onChange={e => {
                update({ filterColumn: e.target.value });
              }} disabled={!schema && !query.filterColumn}>
                <option value="">{t(c.column)}</option>{schema?.columns.map(col => <option key={col.name} value={col.name}>{col.name}</option>)}{!schema && query.filterColumn && <option value={query.filterColumn}>{query.filterColumn}</option>}
              </select>
              <label className="relative flex min-w-0 flex-1 items-center">
                <Search aria-hidden="true" className="pointer-events-none absolute start-3 h-4 w-4 text-wc-text-muted" />
                <span className="sr-only">{t(c.filter)}</span>
                <input id={filterId} type="search" maxLength={256} disabled={!query.filterColumn} value={query.filter} onChange={e => {
                  update({ filter: e.target.value });
                }} placeholder={t(c.filter)} title={t(c.filterHint)} className={`${inputClass} w-full ps-9 disabled:opacity-50`} />
              </label>
            </div>}
            <div ref={host} className="min-h-0 min-w-0 flex-1 overflow-auto overscroll-contain" aria-busy={pending || reader.busy}>
              {reader.queryError ? <div className="flex flex-col items-center gap-4 p-6 text-center">
                <p role="status" className="text-sm text-wc-text-secondary">{t(c[reader.queryError])}</p>{query.filter && <button className={previewTextButtonClass} onClick={() => {
                  update({ filter: "" });
                }}>{t(c.clear)}</button>}<button className={previewTextButtonClass} onClick={() => {
                  reader.browse(query);
                }}>{t(c.retry)}</button>
              </div>
                : pending ? <PreviewLoading label={t(c.loading)} />
                  : schema && mode === "schema" ? <SchemaView schema={schema} wide={wide} />
                    : page && rows.length === 0 ? <div className="space-y-2 p-6 text-center">
                      <p className="text-sm font-medium">{t(c.empty)}</p>
                      <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.emptyHint)}</p>{query.filter && <button className={previewTextButtonClass} onClick={() => {
                        update({ filter: "" });
                      }}>{t(c.clear)}</button>}</div>
                      : page && schema && (wide && record === null ? <table data-testid={selectors.databasePreview.grid} className="w-full border-collapse text-start text-xs" dir="ltr">
                        <caption className="sr-only">{query.table}</caption>
                        <thead className="sticky top-0 z-[1] bg-wc-surface-raised">
                          <tr>{schema.columns.map(col => <th scope="col" key={col.name} aria-sort={query.sortColumn === col.name ? (query.descending ? "descending" : "ascending") : "none"} className="max-w-64 border-b border-wc-default px-1 text-start font-medium">
                            <button className="flex min-h-11 w-full items-center gap-2 px-3 text-wc-text-secondary outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" onClick={() => {
                              update({ sortColumn: col.name, descending: query.sortColumn === col.name ? !query.descending : false });
                            }}>
                              <span className="truncate" title={col.name}>{col.name}</span>{col.primaryKey > 0 && <KeyRound aria-label={t(c.primaryKey)} className="h-3 w-3 shrink-0 text-wc-accent" />}{query.sortColumn === col.name && (query.descending ? <ArrowDown aria-hidden="true" className="h-3 w-3 shrink-0" /> : <ArrowUp aria-hidden="true" className="h-3 w-3 shrink-0" />)}</button>
                          </th>)}</tr>
                        </thead>
                        <tbody>{rows.map((row, index) => <tr key={index} className="border-b border-wc-default hover:bg-wc-surface-input">{row.map((cell, column) => <td key={column} className="max-w-72 p-0">
                          <button className={`block min-h-11 w-full truncate px-4 py-3 text-start outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent ${cell.type === "null" || cell.type === "blob" ? "text-wc-text-muted" : "text-wc-text-secondary"}`} aria-label={`${schema.columns[column]?.name ?? ""}: ${cellLabel(cell).slice(0, 160)}`} title={cellLabel(cell).slice(0, 160)} onClick={event => {
                            inspect(cell, schema.columns[column]?.name ?? "", event.currentTarget);
                          }}>{cellLabel(cell).slice(0, 160)}</button>
                        </td>)}</tr>)}</tbody>
                      </table>
                        : record === null ? <div data-testid={selectors.databasePreview.records}>{rows.map((row, index) => {
                          const preferred = schema.columns.findIndex(col => /^(title|name|label|subject)$/i.test(col.name));
                          const titleIndex = preferred >= 0 ? preferred : 0;
                          return <button key={index} data-database-record={index} className="flex min-h-20 w-full items-center gap-3 border-b border-wc-default px-4 py-4 text-start outline-none hover:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent" onClick={() => {
                            setRecord(index);
                          }}>
                            <span className="min-w-0 flex-1">
                              <span className="block truncate text-sm font-medium">{row[titleIndex] ? cellLabel(row[titleIndex]).slice(0, 160) : t(c.record, { number: query.page * DATABASE_PAGE_SIZE + index + 1 })}</span>
                              <span className="mt-1 block truncate text-xs text-wc-text-muted">{row.map((cell, i) => ({ cell, i })).filter(({ i }) => i !== titleIndex).slice(0, 3).map(({ cell }) => cellLabel(cell).slice(0, 60)).join(" · ")}</span>
                            </span>
                            <ChevronRight aria-hidden="true" className="h-4 w-4 shrink-0 text-wc-text-muted rtl:rotate-180" />
                          </button>;
                        })}</div>
                          : <div className="p-3 sm:p-4">
                            <button ref={backRef} className={`${previewTextButtonClass} mb-4 border-transparent px-1`} onClick={closeRecord}>
                              <ArrowLeft aria-hidden="true" className="h-4 w-4 rtl:rotate-180" />{t(c.back)}</button>
                            <p className="mb-4 text-xs text-wc-text-muted">{t(c.record, { number: query.page * DATABASE_PAGE_SIZE + record + 1 })}</p>
                            <dl>{schema.columns.map((col, index) => {
                              const cell = rows[record]?.[index];
                              return cell && <div key={col.name} className="border-b border-wc-default py-3">
                                <dt className="mb-1 break-words text-xs text-wc-text-muted">{col.name}</dt>
                                <dd>
                                  <button className="min-h-11 w-full whitespace-pre-wrap break-words text-start font-mono text-sm text-wc-text-secondary outline-none [overflow-wrap:anywhere] focus-visible:ring-2 focus-visible:ring-wc-accent" onClick={event => {
                                    inspect(cell, col.name, event.currentTarget);
                                  }}>{cellLabel(cell).slice(0, 240)}{cell.value.length > 240 ? "…" : ""}</button>
                                </dd>
                              </div>;
                            })}</dl>
                          </div>)}
            </div>
            {mode === "data" && record === null && <div data-testid={selectors.databasePreview.navigation} className="flex shrink-0 items-center justify-between gap-1 border-t border-wc-default px-2 py-1">
              <PreviewButton label={t(c.export)} disabled={!page || !rows.length} onClick={exportPage}>
                <Download />
              </PreviewButton>
              <span role="status" className="min-w-0 truncate px-1 text-[11px] tabular-nums text-wc-text-muted">{rows.length ? t(c.range, { start: query.page * DATABASE_PAGE_SIZE + 1, end: query.page * DATABASE_PAGE_SIZE + rows.length }) : t(c.page, { page: query.page + 1 })}</span>
              <div className="flex shrink-0 items-center">
                <PreviewButton label={t(c.previous)} disabled={reader.busy || query.page === 0} onClick={() => {
                  update({ page: query.page - 1 });
                }}>
                  <ChevronLeft className="rtl:rotate-180" />
                </PreviewButton>
                <PreviewButton label={t(c.next)} disabled={reader.busy || !page?.hasMore || query.page >= DATABASE_MAX_PAGE} onClick={() => {
                  update({ page: query.page + 1 });
                }}>
                  <ChevronRight className="rtl:rotate-180" />
                </PreviewButton>
              </div>
            </div>}
            {query.page >= DATABASE_MAX_PAGE && page?.hasMore && <p role="status" className="shrink-0 px-4 py-2 text-xs text-wc-text-muted">{t(c.pageLimit)}</p>}
          </>}
    <ArtifactFooter model={model} metadata={`SQLite · ${formatBytes(info?.bytes ?? model.sizeBytes)}`} />
    <ResponsiveDialog open={selected !== null} onClose={() => {
      setSelected(null);
    }} returnFocusRef={focusReturn} avoidKeyboard dismissAffordance="close" size="md" title={t(c.cell)} ariaLabel={t(c.cell)} closeLabel={t(strings.inspectionPreview.closeOptions)} testId="database-cell-dialog">
      {selected && <div className="space-y-4 p-4 sm:p-5">
        <div>
          <p className="break-words text-sm font-semibold [overflow-wrap:anywhere]">{selected.column}</p>
          <p className="mt-1 text-xs text-wc-text-muted">{selected.cell.type.toUpperCase()} · {formatBytes(selected.cell.bytes)}</p>
        </div>{selected.cell.truncated && <p role="status" className="text-xs text-wc-text-muted">{t(c.truncated)}</p>}<pre dir="auto" className="max-h-[50dvh] overflow-auto whitespace-pre-wrap break-words font-mono text-xs leading-relaxed text-wc-text-secondary [overflow-wrap:anywhere]">{databaseDetail(selected.cell)}</pre>{selected.cell.type !== "blob" && selected.cell.type !== "null" && <button className={previewTextButtonClass} onClick={() => {
          void clipboard.copy(selected.cell.value);
        }}>{t(c.copy)}</button>}<p role="status" className="text-xs text-wc-text-muted">{clipboard.status ? t(clipboard.status === "copied" ? c.copied : c.copyFailed) : ""}</p>
      </div>}
    </ResponsiveDialog>
  </ArtifactFrame>;
}
function SchemaView({ schema, wide }: {
  schema: DatabaseSchema;
  wide: boolean;
}) {
  const { t } = useTranslation(), c = strings.databasePreview;
  return <div data-testid={selectors.databasePreview.schema} className={`${wide ? "grid grid-cols-[minmax(0,2fr)_minmax(0,1fr)] gap-8" : "space-y-8"} p-4 sm:p-6`}>
    <section>
      <h3 className="mb-3 text-xs font-semibold text-wc-text-muted">{t(c.columns)}</h3>
      <dl>{schema.columns.map(col => <div key={col.name} className="border-b border-wc-default py-3">
        <dt className="flex min-w-0 items-center gap-2">
          <span className="break-words font-mono text-sm [overflow-wrap:anywhere]">{col.name}</span>{col.primaryKey > 0 && <KeyRound className="h-3.5 w-3.5 shrink-0 text-wc-accent" aria-label={t(c.primaryKey)} />}</dt>
        <dd className="mt-1 break-words text-xs leading-relaxed text-wc-text-muted [overflow-wrap:anywhere]">{col.type || "—"} · {t(col.primaryKey > 0 ? c.primaryKey : col.required ? c.required : c.optional)}{col.generated ? ` · ${t(c.generated)}` : ""}{col.defaultValue !== null && <span className="mt-1 block font-mono">{t(c.default)}: {col.defaultValue}</span>}</dd>
      </div>)}</dl>
    </section>
    <div className="min-w-0 space-y-8"><section>
      <h3 className="mb-3 text-xs font-semibold text-wc-text-muted">{t(c.indexes)}</h3>{!schema.indexes.length ? <p className="text-xs text-wc-text-muted">{t(c.none)}</p> : <ul className="space-y-4">{schema.indexes.map(index => <li key={index.name} className="break-words [overflow-wrap:anywhere]">
        <p className="font-mono text-xs">{index.name}</p>
        <p className="mt-1 text-xs text-wc-text-muted">{index.columns.join(", ")}{index.unique ? ` · ${t(c.unique)}` : ""}{index.partial ? ` · ${t(c.partial)}` : ""}</p>
      </li>)}</ul>}</section>
    <section>
      <h3 className="mb-3 text-xs font-semibold text-wc-text-muted">{t(c.foreignKeys)}</h3>{!schema.foreignKeys.length ? <p className="text-xs text-wc-text-muted">{t(c.none)}</p> : <ul className="space-y-3">{schema.foreignKeys.map((key, index) => <li key={index} className="break-words font-mono text-xs text-wc-text-secondary [overflow-wrap:anywhere]">{key.from} → {key.table}{key.to ? `.${key.to}` : ""}</li>)}</ul>}</section>
    </div>
  </div>;
}
