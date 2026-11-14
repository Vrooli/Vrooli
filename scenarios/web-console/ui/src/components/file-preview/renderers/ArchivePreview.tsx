import { useEffect, useMemo, useRef } from "react";
import { Archive, ArrowUp, ChevronRight, Eye, File, Folder, Search, ShieldAlert } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";

import { strings } from "../../../consts/strings";
import { DEFAULT_ARCHIVE_VIEW, type ArchiveView } from "../../../api/filePreview";
import { archiveDisplayName, archiveRows, archiveStatusKey, type ArchiveRow } from "../archiveData";
import { formatBytes } from "../format";
import type { PreviewRendererProps } from "../types";
import { cn } from "../../../lib/classnames";
import {
  ArtifactFrame, ArtifactFooter, InspectionHeader, InspectionOptions,
  InspectionSetting, PreviewButton, PreviewLoading, previewSelectClass, previewTextButtonClass,
} from "./ArtifactChrome";

const columnClass = "grid grid-cols-[minmax(0,1fr)_5rem_9rem_1rem] items-center";

export function ArchivePreview({
  model, archive, archiveView = DEFAULT_ARCHIVE_VIEW,
  onArchiveFolder, onArchiveEntry, onArchiveViewChange, onReopen,
}: PreviewRendererProps) {
  const { t } = useTranslation(), c = strings.archivePreview, d = strings.documentPreview;
  const { ref, rect } = useResizeObserver<HTMLDivElement>();
  const wide = (rect?.width ?? 0) >= 640;
  const rows = useMemo(() => archive ? archiveRows(archive, archiveView) : [], [archive, archiveView]);
  if (!archive) return <PreviewLoading label={t(strings.messagesFileViewer.loadingPreview)} />;

  const ready = archive.status === "ready";
  const parent = archiveView.folder.split("/").slice(0, -1).join("/");
  const description = ready
    ? t(c.summary, { count: archive.entries.length, size: formatBytes(model.sizeBytes) })
    : formatBytes(model.sizeBytes);

  return (
    <ArtifactFrame testId="file-preview-archive" frameRef={ref}>
      <InspectionHeader icon={<Archive />} title={archive.format ? t(c.title, { format: archive.format }) : t(c.archive)} description={description}>
        <PreviewButton label={t(d.up)} disabled={!ready || !archiveView.folder} onClick={() => { onArchiveFolder?.(parent); }}>
          <ArrowUp className="h-4 w-4" />
        </PreviewButton>
        <ArchiveOptions view={archiveView} onChange={onArchiveViewChange} />
      </InspectionHeader>
      {ready ? (
        <>
          <div className="shrink-0 border-b border-wc-default px-3 py-2 sm:px-4">
            <label className="relative flex min-w-0 items-center">
              <Search aria-hidden="true" className="pointer-events-none absolute start-3 h-4 w-4 text-wc-text-muted" />
              <input type="search" value={archiveView.query} onChange={event => { onArchiveViewChange?.({ query: event.target.value, page: 1 }); }}
                aria-label={t(c.filter)} placeholder={t(c.filter)} data-testid="archive-filter"
                className="h-11 w-full min-w-0 rounded-lg border border-wc-default bg-wc-surface-input pe-3 ps-10 text-base text-wc-text-primary placeholder:text-wc-text-faint focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
            </label>
            {archiveView.query.trim() && <p role="status" className="mt-2 text-xs text-wc-text-muted">{t(d.results, { count: rows.length })}</p>}
          </div>
          {wide && rows.length > 0 && (
            <div aria-hidden="true" className={`${columnClass} shrink-0 gap-3 border-b border-wc-default bg-wc-surface-raised px-5 py-2 text-[11px] font-medium text-wc-text-muted`}>
              <span>{t(d.name)}</span><span className="text-end">{t(d.size)}</span><span className="text-end">{t(c.status)}</span><span />
            </div>
          )}
          <div className="min-h-0 flex-1 overflow-auto overscroll-contain">
            {rows.length === 0 ? (
              <div className="flex flex-col items-center gap-3 px-5 py-12 text-center">
                <Search aria-hidden="true" className="h-7 w-7 text-wc-text-faint" />
                <p className="text-sm text-wc-text-muted">{t(archiveView.query.trim() ? c.noResults : c.empty)}</p>
                {!archiveView.showHidden && !archiveView.query.trim() && (
                  <button type="button" className={previewTextButtonClass} onClick={() => { onArchiveViewChange?.({ showHidden: true, page: 1 }); }}>{t(d.showHidden)}</button>
                )}
              </div>
            ) : (
              <ul>{rows.slice(0, archiveView.page * 100).map(row => (
                <ArchiveEntryRow key={row.id} row={row} wide={wide}
                  onOpen={() => {
                    if (row.folder) onArchiveFolder?.(row.folder);
                    else onArchiveEntry?.({ id: row.id, name: row.fullName });
                  }} />
              ))}</ul>
            )}
            {rows.length > archiveView.page * 100 && (
              <div className="p-4"><button type="button" className={`${previewTextButtonClass} w-full`} onClick={() => { onArchiveViewChange?.({ page: archiveView.page + 1 }); }}>
                {t(strings.messagesFileViewer.directoryLoadMore)}
              </button></div>
            )}
          </div>
        </>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-4 overflow-auto px-6 py-10 text-center" role="status">
          <Archive aria-hidden="true" className="h-8 w-8 text-wc-text-faint" />
          <p className="max-w-sm text-sm leading-relaxed text-wc-text-secondary">{t(archiveStatusKey(archive.status))}</p>
          <button type="button" onClick={onReopen} className={previewTextButtonClass}>{t(strings.messagesFileViewer.reopen)}</button>
        </div>
      )}
      <ArtifactFooter model={model} metadata={t(c.readOnly)} />
    </ArtifactFrame>
  );
}

function ArchiveOptions({ view, onChange }: { view: ArchiveView; onChange?: (value: Partial<ArchiveView>) => void }) {
  const { t } = useTranslation(), c = strings.archivePreview, d = strings.documentPreview;
  return (
    <InspectionOptions label={t(c.options)}>{() => <>
      <InspectionSetting label={t(strings.messagesFileViewer.directorySortLabel)}>
        <select value={view.sort} aria-label={t(strings.messagesFileViewer.directorySortLabel)}
          onChange={event => { onChange?.({ sort: event.target.value as ArchiveView["sort"], page: 1 }); }} className={`${previewSelectClass} w-full text-base`}>
          <option value="name">{t(d.name)}</option><option value="size">{t(strings.messagesFileViewer.directorySortSize)}</option>
        </select>
      </InspectionSetting>
      <label className="flex min-h-12 cursor-pointer items-center gap-3 rounded-xl border border-wc-default px-3 py-2 text-sm">
        <Eye aria-hidden="true" className="h-4 w-4 text-wc-text-muted" /><span className="flex-1">{t(d.showHidden)}</span>
        <input type="checkbox" checked={view.showHidden} onChange={() => { onChange?.({ showHidden: !view.showHidden, page: 1 }); }} className="peer sr-only" />
        <span aria-hidden="true" className={cn("wc-inspection-toggle peer-focus-visible:ring-2 peer-focus-visible:ring-wc-accent", view.showHidden && "wc-inspection-toggle-on")} />
      </label>
      <p className="text-xs leading-relaxed text-wc-text-muted">{t(c.limits)}</p>
    </>}</InspectionOptions>
  );
}

export function ArchiveBreadcrumbs({ folder, onNavigate }: { folder: string; onNavigate?: (folder: string) => void }) {
  const { t } = useTranslation(), c = strings.archivePreview;
  const parts = folder.split("/").filter(Boolean);
  const navigation = useRef<HTMLDivElement>(null);
  useEffect(() => { navigation.current?.querySelector("[aria-current=location]")?.scrollIntoView({ block: "nearest", inline: "end" }); }, [folder]);
  const buttonClass = "min-h-11 shrink-0 rounded px-2 text-wc-text-secondary outline-none hover:text-wc-text-primary focus-visible:ring-2 focus-visible:ring-wc-accent";
  return (
    <div ref={navigation} role="group" aria-label={t(c.location)} data-testid="archive-breadcrumbs" className="flex min-h-11 min-w-0 flex-1 items-center overflow-x-auto text-xs">
      <button type="button" aria-current={parts.length === 0 ? "location" : undefined} onClick={() => { onNavigate?.(""); }} className={buttonClass}>{t(c.archive)}</button>
      {parts.map((part, index) => {
        const path = parts.slice(0, index + 1).join("/");
        return <span key={path} className="flex shrink-0 items-center">
          <ChevronRight aria-hidden="true" className="h-3 w-3 text-wc-text-faint rtl:rotate-180" />
          <button type="button" aria-current={index === parts.length - 1 ? "location" : undefined}
            onClick={() => { onNavigate?.(path); }} className={`${buttonClass} max-w-52 truncate`}><bdi>{part}</bdi></button>
        </span>;
      })}
    </div>
  );
}

function ArchiveEntryRow({ row, wide, onOpen }: { row: ArchiveRow; wide: boolean; onOpen: () => void }) {
  const { t } = useTranslation(), c = strings.archivePreview;
  const enabled = Boolean(row.folder) || row.status === "ready";
  const status = t(row.status === "unsupported" ? c.methodUnsupported : archiveStatusKey(row.status));
  const name = archiveDisplayName(row.name);
  const detail = row.collisions > 1 ? t(c.duplicate, { count: row.collisions, index: row.collisionIndex }) : status;
  const identity = (
    <span className="flex min-w-0 items-center gap-3">
      <span aria-hidden="true" className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-lg", row.folder ? "bg-wc-accent/10 text-wc-accent" : "bg-wc-surface-raised text-wc-text-muted")}>
        {row.folder ? <Folder className="h-4 w-4" /> : enabled ? <File className="h-4 w-4" /> : <ShieldAlert className="h-4 w-4" />}
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium text-wc-text-primary" title={name} dir="auto">{name}</span>
        {!wide && <span className="mt-0.5 block truncate text-xs text-wc-text-muted">
          <bdi>{row.folder ? status : formatBytes(row.sizeBytes)}</bdi>{!row.folder && <> · <bdi>{detail}</bdi></>}
        </span>}
      </span>
    </span>
  );
  const body = <>
    {identity}
    {wide && <>
      <span className="text-end text-xs tabular-nums text-wc-text-muted"><bdi>{row.folder ? "—" : formatBytes(row.sizeBytes)}</bdi></span>
      <span className="truncate text-end text-xs text-wc-text-muted" title={detail}>{detail}</span>
    </>}
    <ChevronRight aria-hidden="true" className={cn("h-4 w-4 shrink-0 text-wc-text-faint rtl:rotate-180", !enabled && "invisible")} />
  </>;
  const rowClass = cn("min-h-16 w-full gap-3 border-b border-wc-default px-4 py-3 text-start sm:px-5", wide ? columnClass : "flex items-center [&>span:first-child]:flex-1");
  return <li>{enabled ? (
    <button type="button" data-testid="archive-entry" onClick={onOpen} className={cn(rowClass, "outline-none transition hover:bg-wc-surface-raised focus-visible:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent")}>{body}</button>
  ) : <div data-testid="archive-entry-blocked" className={rowClass}>{body}</div>}</li>;
}
