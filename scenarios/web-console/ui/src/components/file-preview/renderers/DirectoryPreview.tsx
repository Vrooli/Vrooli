import { useMemo, useState } from "react";
import { useResizeObserver } from "@vrooli/react-component-library/useResizeObserver";
import { ArtifactFrame, InspectionHeader, InspectionOptions, InspectionSetting, PreviewButton, PreviewLoading, previewSelectClass, previewTextButtonClass } from "./ArtifactChrome";
import {
  AlertTriangle,
  Boxes,
  Check,
  ArrowUp,
  BookOpen,
  ChevronRight,
  Eye,
  ScanEye,
  File as FileIcon,
  FileCode,
  FileImage,
  FileText,
  FileVideo,
  Folder,
  Link2,
  Link2Off,
  Loader2,
  Music,
  Presentation,
  Search,
  Table,
  Type,
  Database,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import { joinPath } from "../../../lib/paths";
import { formatBytes } from "../format";
import { PreviewNotice } from "./shared";
import { canThumbnail, previewThumbnailHref, type DirectoryEntry, type DirectorySort, type PreviewKind } from "../../../api/filePreview";
import type { DirectorySelection, PreviewRendererProps } from "../types";

// SORT_OPTIONS is the ordering control. The two name sorts always work; the
// size and date sorts may be downgraded by the server on a large directory,
// which the toolbar reports rather than silently misrepresenting the order.
// `as const satisfies` keeps each label's literal key type — the string
// registry is a typed union, so widening to `string` would break t().
const SORT_OPTIONS = [
  { value: "dirs_first_name", label: strings.messagesFileViewer.directorySortDirsFirst },
  { value: "name", label: strings.messagesFileViewer.directorySortName },
  { value: "size_desc", label: strings.messagesFileViewer.directorySortSize },
  { value: "mtime_desc", label: strings.messagesFileViewer.directorySortMtime },
] as const satisfies ReadonlyArray<{ value: DirectorySort; label: string }>;

// DirectoryPreview renders a directory as a navigable list. Every clickable
// row calls the same onNavigate — files and directories alike — so the row has
// no branch: the controller resolves the path and the resulting kind decides
// which renderer appears next.
export function DirectoryPreview(props: PreviewRendererProps) { return <FolderBrowser key={props.model.previewId} {...props} />; }
function FolderBrowser({ model, listing, onNavigate, onLoadMore, onListOptionsChange, loadingMore, listingRefreshing = false, selection, imageSessionId }: PreviewRendererProps) {
  const { t, i18n } = useTranslation(), c = strings.documentPreview;
  const [query, setQuery] = useState("");
  const { ref, rect } = useResizeObserver<HTMLDivElement>(), wide = (rect?.width ?? 0) >= 640;
  const entries = useMemo(() => listing?.entries ?? [], [listing?.entries]);
  const filtered = useMemo(() => {
    const needle = query.trim().toLocaleLowerCase();
    return needle ? entries.filter(entry => entry.name.toLocaleLowerCase().includes(needle)) : entries;
  }, [entries, query]);
  if (!listing) return <PreviewLoading label={t(strings.messagesFileViewer.loadingPreview)} />;
  const sortDowngraded = listing.effectiveSort !== listing.sort;
  const modified = (value: number) => value > 0 && Number.isFinite(new Date(value).getTime()) ? new Intl.DateTimeFormat(i18n.resolvedLanguage === "cimode" ? "en" : i18n.resolvedLanguage, { dateStyle: "medium" }).format(value) : "—";
  return <ArtifactFrame testId="file-preview-directory" frameRef={ref}>
    <InspectionHeader icon={<Folder />} title={t(c.folderTitle)} description={<span data-testid="directory-count">{t(strings.messagesFileViewer.directoryEntryCount, { loaded: entries.length, total: listing.totalEntries })}</span>}>
      <PreviewButton label={t(c.up)} disabled={!listing.parentPath || listing.parentPath === listing.resolvedPath} onClick={() => { if (listing.parentPath) onNavigate(listing.parentPath); }}><ArrowUp className="h-4 w-4" /></PreviewButton>
      <InspectionOptions label={t(c.folderOptions)}>{() => <>
        <InspectionSetting label={t(strings.messagesFileViewer.directorySortLabel)}><select aria-disabled={listingRefreshing} value={listing.sort} onChange={event => { if (!listingRefreshing) onListOptionsChange({ sort: event.target.value as DirectorySort }); }} aria-label={t(strings.messagesFileViewer.directorySortLabel)} data-testid="directory-sort" className={`${previewSelectClass} w-full text-base`}>
          {SORT_OPTIONS.map(option => <option key={option.value} value={option.value}>{t(option.label)}</option>)}
        </select></InspectionSetting>
        <label className="flex min-h-12 cursor-pointer items-center gap-3 rounded-xl border border-wc-default px-3 py-2 text-sm"><Eye aria-hidden="true" className="h-4 w-4 text-wc-text-muted" /><span className="flex-1">{t(c.showHidden)}</span><input type="checkbox" aria-disabled={listingRefreshing} checked={listing.showHidden} onChange={() => { if (!listingRefreshing) onListOptionsChange({ showHidden: !listing.showHidden }); }} data-testid="directory-toggle-hidden" className="peer sr-only" /><span aria-hidden="true" className={cn("wc-inspection-toggle peer-focus-visible:ring-2 peer-focus-visible:ring-wc-accent", listing.showHidden && "wc-inspection-toggle-on")} /></label>
      </>}</InspectionOptions>
    </InspectionHeader>
    <div className="shrink-0 border-b border-wc-default px-3 py-2 sm:px-4">
      <label className="relative flex min-w-0 items-center"><Search aria-hidden="true" className="pointer-events-none absolute start-3 h-4 w-4 text-wc-text-muted" />
        <input type="search" value={query} onChange={event => { setQuery(event.target.value); }} aria-label={t(strings.messagesFileViewer.directoryFilter)} placeholder={t(strings.messagesFileViewer.directoryFilter)} data-testid="directory-filter"
          className="h-11 w-full min-w-0 rounded-lg border border-wc-default bg-wc-surface-input pe-3 ps-10 text-base text-wc-text-primary placeholder:text-wc-text-faint focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-wc-accent" />
      </label>
      {query.trim() && <p role="status" className="mt-2 text-xs text-wc-text-muted">{t(c.results, { count: filtered.length })}</p>}
      {listing.nextPageToken && <p className="mt-2 text-xs leading-relaxed text-wc-text-muted">{t(c.loadedOnly, { count: entries.length })}</p>}
    </div>
    {sortDowngraded && <PreviewNotice message={t(strings.messagesFileViewer.directorySortDowngraded)} tone="info" />}
    {listing.truncated && <PreviewNotice message={t(strings.messagesFileViewer.directoryTruncated)} tone="info" />}
    {wide && entries.length > 0 && <div aria-hidden="true" className={cn("wc-folder-columns shrink-0 border-b border-wc-default bg-wc-surface-raised px-5 py-2 text-[11px] font-medium text-wc-text-muted", selection?.onPreview && "pe-[4.25rem]")}><span>{t(c.name)}</span><span className="text-end">{t(c.size)}</span><span className="text-end">{t(c.modified)}</span><span /></div>}
    <div className="min-h-0 flex-1 overflow-auto overscroll-contain" aria-busy={listingRefreshing}>
      {entries.length === 0 ? <EmptyState showHidden={listing.showHidden} onShowHidden={() => { onListOptionsChange({ showHidden: true }); }} /> : filtered.length === 0 ?
        <div className="flex flex-col items-center gap-3 px-5 py-12 text-center"><Search aria-hidden="true" className="h-7 w-7 text-wc-text-faint" /><p className="max-w-sm break-all text-sm text-wc-text-muted" data-testid="directory-filter-empty">{t(strings.messagesFileViewer.directoryFilterEmpty, { query: query.trim() })}</p></div> :
        <ul>{filtered.map(entry => { const path = joinPath(listing.resolvedPath || model.resolvedPath, entry.name);
          // The directory's own preview id is the capability for its children's
          // thumbnails, so a row costs one small image request and no resolve.
          const thumbnail = imageSessionId && entry.canPreview && !entry.symlinkBroken && entry.entryType !== "directory" && canThumbnail(entry.name)
            ? previewThumbnailHref(imageSessionId, model.previewId, { entry: entry.name, size: THUMBNAIL_PX }) : undefined;
          return <EntryRow key={entry.name} entry={entry} disabled={listingRefreshing} wide={wide} modified={modified(entry.mtimeMs)} onOpen={() => { onNavigate(path); }} selection={selection} path={path} thumbnailHref={thumbnail} />; })}</ul>}
      {listing.nextPageToken && <div className="px-4 py-4"><button type="button" onClick={onLoadMore} disabled={loadingMore || listingRefreshing} data-testid="directory-load-more" className={`${previewTextButtonClass} w-full`}>
        {loadingMore && <Loader2 aria-hidden="true" className="h-4 w-4 animate-spin" />}{t(loadingMore ? strings.messagesFileViewer.directoryLoadingMore : strings.messagesFileViewer.directoryLoadMore)}
      </button></div>}
    </div>
  </ArtifactFrame>;
}

// EmptyState distinguishes a genuinely empty directory from one that only
// looks empty because the hidden filter is on — otherwise the pad reads as
// broken when it is merely filtered.
function EmptyState({ showHidden, onShowHidden }: { showHidden: boolean; onShowHidden: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="flex h-full flex-col items-center justify-center gap-3 px-4 py-10 text-center">
      <p className="text-sm text-wc-text-muted" data-testid="directory-empty">
        {t(strings.messagesFileViewer.directoryEmpty)}
      </p>
      {!showHidden && (
        <button
          type="button"
          onClick={onShowHidden}
          data-testid="directory-empty-show-hidden"
          className={previewTextButtonClass}
        >
          <Eye className="h-3.5 w-3.5" />
          {t(strings.messagesFileViewer.directoryShowHidden)}
        </button>
      )}
    </div>
  );
}

function EntryRow({ entry, onOpen, wide, modified, disabled, selection, path, thumbnailHref }: { entry: DirectoryEntry; onOpen: () => void; wide: boolean; modified: string; disabled: boolean; selection?: DirectorySelection; path: string; thumbnailHref?: string }) {
  const { t } = useTranslation();
  const isDir = entry.entryType === "directory" || entry.kind === "directory";
  const described = entryDetail(entry), detail = "literal" in described ? described.literal : t(described.key, described.params ?? {});
  const type = isDir ? t(strings.documentPreview.folder) : entry.name.includes(".") ? entry.name.split(".").at(-1)?.toUpperCase() : t(strings.documentPreview.file);
  const identity = <><EntryTile entry={entry} isDir={isDir} thumbnailHref={thumbnailHref} />
    <span className="min-w-0 flex-1"><span className={cn("block truncate text-sm font-medium text-wc-text-primary", entry.symlinkBroken && "text-wc-error-text line-through")} dir="auto" title={entry.name}>{entry.name}</span>
      {entry.entryType === "symlink" && entry.symlinkTarget ? <span className="block truncate text-xs text-wc-text-muted" dir="auto">→ {entry.symlinkTarget}</span> :
        !wide && <span className="mt-0.5 block truncate text-xs text-wc-text-muted">{[type, detail, modified !== "—" ? modified : ""].filter(Boolean).map((value, index) => <span key={index}>{index > 0 && <span aria-hidden="true"> · </span>}<bdi>{value}</bdi></span>)}</span>}
      {!wide && entry.entryType === "symlink" && <span className="block truncate text-xs text-wc-text-muted"><bdi>{detail}</bdi></span>}
    </span></>;
  // lead puts a pick-mode checkbox inside the name column, so a row keeps the
  // folder-columns grid shape the header is laid out against.
  const contentWith = (lead?: React.ReactNode) => wide ? <><span className="flex min-w-0 items-center gap-3">{lead}{identity}</span><span className="text-end text-xs tabular-nums text-wc-text-muted"><bdi>{detail}</bdi></span><span className="text-end text-xs tabular-nums text-wc-text-muted"><bdi>{modified}</bdi></span></> : <>{lead}{identity}</>;
  const body = <>
    {contentWith()}
    <ChevronRight aria-hidden="true" className={cn("h-4 w-4 shrink-0 text-wc-text-faint rtl:rotate-180", !entry.canPreview && "invisible")} />
  </>;
  const rowClass = cn("min-h-16 w-full gap-3 border-b border-wc-default px-4 py-3 text-start sm:px-5", wide ? "wc-folder-columns" : "flex items-center");
  if (selection) return <SelectableEntryRow entry={entry} isDir={isDir} body={body} contentWith={contentWith} wide={wide} rowClass={rowClass} onOpen={onOpen} disabled={disabled} selection={selection} path={path} />;
  return <li>{entry.canPreview ? <button type="button" disabled={disabled} onClick={onOpen} data-testid="directory-entry" className={cn(rowClass, "outline-none transition hover:bg-wc-surface-raised focus-visible:bg-wc-surface-input focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent")}>{body}</button> :
    <div data-testid="directory-entry" title={detail} className={cn(rowClass, "opacity-70")}>{body}</div>}</li>;
}

// SelectableEntryRow is a row in pick mode. Only readable entries can be
// picked: an unreadable or broken one would hand the agent a path it cannot
// open. A folder keeps its open-on-tap row beside the checkbox; a file row is
// the checkbox, with a separate Preview button so the operator can check what
// a file is before picking it.
function SelectableEntryRow({ entry, isDir, body, contentWith, wide, rowClass, onOpen, disabled, selection, path }: { entry: DirectoryEntry; isDir: boolean; body: React.JSX.Element; contentWith: (lead?: React.ReactNode) => React.JSX.Element; wide: boolean; rowClass: string; onOpen: () => void; disabled: boolean; selection: DirectorySelection; path: string }) {
  const selectable = entry.canPreview && !entry.symlinkBroken && entry.entryType !== "other";
  const checked = selectable && selection.isSelected(path);
  const toggle = () => { selection.onToggle(path, entry); };
  const box = <span aria-hidden="true" className={cn("flex h-5 w-5 shrink-0 items-center justify-center rounded border", checked ? "border-wc-accent bg-wc-accent text-white" : "border-wc-text-muted bg-wc-surface-input", !selectable && "opacity-40")}>{checked && <Check className="h-3.5 w-3.5" />}</span>;
  const focusRing = "outline-none transition focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-wc-accent";
  const selectedTone = checked && "bg-wc-accent/10";
  const { onPreview, previewLabel } = selection;
  if (!isDir) return <li className={cn("flex items-stretch border-b border-wc-default", selectedTone)}>
    <button type="button" role="checkbox" aria-checked={checked} aria-label={selection.toggleLabel(entry.name)} disabled={disabled || !selectable} onClick={toggle} data-testid="directory-entry" data-selected={checked ? "true" : "false"}
      className={cn(rowClass, "min-w-0 flex-1 border-b-0 ps-3 sm:ps-3 hover:bg-wc-surface-raised disabled:cursor-default", focusRing)}>{contentWith(box)}{wide && <span />}</button>
    {onPreview && (selectable ? <button type="button" disabled={disabled} onClick={() => { onPreview(path); }} aria-label={previewLabel?.(entry.name) ?? entry.name} title={previewLabel?.(entry.name)} data-testid="directory-entry-preview"
      className={cn("flex w-12 shrink-0 items-center justify-center text-wc-text-muted hover:bg-wc-surface-raised hover:text-wc-text-primary", focusRing)}><ScanEye aria-hidden="true" className="h-5 w-5" /></button> : <span aria-hidden="true" className="w-12 shrink-0" />)}
  </li>;
  return <li className={cn("flex items-stretch border-b border-wc-default", selectedTone)}>
    <button type="button" role="checkbox" aria-checked={checked} aria-label={selection.toggleLabel(entry.name)} disabled={disabled || !selectable} onClick={toggle} data-testid="directory-entry-select" className={cn("flex min-h-16 w-12 shrink-0 items-center justify-center hover:bg-wc-surface-raised", focusRing)}>{box}</button>
    <button type="button" disabled={disabled || !entry.canPreview} onClick={onOpen} data-testid="directory-entry" className={cn(rowClass, "min-w-0 flex-1 border-b-0 ps-1 sm:ps-1 hover:bg-wc-surface-raised", focusRing)}>{body}</button>
    {onPreview && <span aria-hidden="true" className="w-12 shrink-0" />}
  </li>;
}

// EntryDetail describes the right-hand column: the fact that matters most for
// this entry type, rather than a byte count that means nothing for a
// directory. It is returned as a key plus params (never a translated string)
// so the helper stays pure and independent of the i18n runtime.
type EntryDetailKey =
  | typeof strings.messagesFileViewer.directoryBrokenLink
  | typeof strings.messagesFileViewer.directorySpecialFile
  | typeof strings.messagesFileViewer.directoryUnreadable
  | typeof strings.messagesFileViewer.directoryChildCount;

type EntryDetail =
  | { literal: string }
  | { key: EntryDetailKey; params?: Record<string, unknown> };

function entryDetail(entry: DirectoryEntry): EntryDetail {
  if (entry.symlinkBroken) return { key: strings.messagesFileViewer.directoryBrokenLink };
  if (entry.entryType === "other") return { key: strings.messagesFileViewer.directorySpecialFile };
  if (!entry.canPreview) return { key: strings.messagesFileViewer.directoryUnreadable };
  if (entry.entryType === "directory") {
    return entry.childCount === null
      ? { literal: "" }
      : { key: strings.messagesFileViewer.directoryChildCount, params: { count: entry.childCount } };
  }
  return { literal: formatBytes(entry.sizeBytes) };
}

// THUMBNAIL_PX is the rendition edge asked of the host for a row's 36px tile:
// enough for a 2.5x screen without paying for detail nobody can see.
const THUMBNAIL_PX = 96;

// EntryTile is the row's 36px identity tile. An image the host can render
// shows its thumbnail, loaded lazily as the row scrolls near; until it loads,
// and if it fails, the kind glyph stays.
function EntryTile({ entry, isDir, thumbnailHref }: { entry: DirectoryEntry; isDir: boolean; thumbnailHref?: string }) {
  const [state, setState] = useState<"loading" | "loaded" | "failed">("loading");
  const showThumb = Boolean(thumbnailHref) && state !== "failed";
  return <span aria-hidden="true" data-testid={showThumb ? "directory-entry-thumbnail" : undefined} className={cn("relative flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-lg", isDir ? "bg-wc-accent/10 text-wc-accent" : "bg-wc-surface-raised text-wc-text-muted")}>
    {state !== "loaded" && <EntryGlyph entry={entry} />}
    {showThumb && <img src={thumbnailHref} alt="" loading="lazy" decoding="async" draggable={false} onLoad={() => { setState("loaded"); }} onError={() => { setState("failed"); }}
      className={cn("absolute inset-0 h-full w-full object-cover", state !== "loaded" && "opacity-0")} />}
  </span>;
}

function EntryGlyph({ entry }: { entry: DirectoryEntry }) {
  const className = "h-4 w-4 shrink-0";
  if (entry.symlinkBroken) return <Link2Off className={cn(className, "text-amber-400")} />;
  if (entry.entryType === "symlink") return <Link2 className={cn(className, "text-wc-text-muted")} />;
  if (entry.entryType === "directory" || entry.kind === "directory") {
    return <Folder className={cn(className, "text-wc-accent")} />;
  }
  if (entry.entryType === "other") return <AlertTriangle className={cn(className, "text-wc-text-faint")} />;
  return <KindGlyph kind={entry.kind} className={cn(className, "text-wc-text-muted")} />;
}

// KindGlyph maps a classified kind onto an icon. A null kind is the honest
// "determined on open" case and gets the neutral file glyph.
function KindGlyph({ kind, className }: { kind: PreviewKind | null; className: string }) {
  switch (kind) {
    case "database":
      return <Database className={className} />;
    case "font":
      return <Type className={className} />;
    case "model3d":
      return <Boxes className={className} />;
    case "presentation":
      return <Presentation className={className} />;
    case "ebook":
      return <BookOpen className={className} />;
    case "document":
    case "markdown":
    case "text":
      return <FileText className={className} />;
    case "code":
      return <FileCode className={className} />;
    case "spreadsheet":
    case "csv":
      return <Table className={className} />;
    case "diff":
      return <FileCode className={className} />;
    case "image":
    case "svg":
      return <FileImage className={className} />;
    case "video":
      return <FileVideo className={className} />;
    case "audio":
      return <Music className={className} />;
    case "pdf":
      return <FileText className={className} />;
    default:
      return <FileIcon className={className} />;
  }
}
