import { useMemo, useState } from "react";
import {
  AlertTriangle,
  ChevronRight,
  Eye,
  EyeOff,
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
  Search,
  Table,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { strings } from "../../../consts/strings";
import { cn } from "../../../lib/classnames";
import { joinPath } from "../../../lib/paths";
import { formatBytes } from "../format";
import { PreviewNotice } from "./shared";
import type { DirectoryEntry, DirectorySort, PreviewKind } from "../../../api/filePreview";
import type { PreviewRendererProps } from "../types";

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
export function DirectoryPreview({
  model,
  listing,
  onNavigate,
  onLoadMore,
  onListOptionsChange,
  loadingMore,
}: PreviewRendererProps) {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");

  // Depend on listing?.entries rather than a defaulted local: `?? []` builds a
  // fresh array each render, which would make the memo recompute every time.
  const entries = useMemo(() => listing?.entries ?? [], [listing?.entries]);
  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return entries;
    return entries.filter((e) => e.name.toLowerCase().includes(needle));
  }, [entries, query]);

  if (!listing) {
    return (
      <div className="flex h-full items-center justify-center gap-2 text-wc-text-muted">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>{t(strings.messagesFileViewer.loadingPreview)}</span>
      </div>
    );
  }

  const sortDowngraded = listing.effectiveSort !== listing.sort;

  return (
    <div className="flex h-full flex-col" data-testid="file-preview-directory">
      {/* Toolbar. Scrolls horizontally rather than wrapping, so a narrow
          phone viewport never pushes the list below the fold. */}
      <div className="shrink-0 border-b border-wc-default px-3 py-2">
        <div className="flex items-center gap-2 overflow-x-auto pb-0.5">
          <label className="relative flex min-w-[9rem] flex-1 items-center">
            <Search className="pointer-events-none absolute left-2 h-3.5 w-3.5 text-wc-text-faint" />
            <input
              type="search"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder={t(strings.messagesFileViewer.directoryFilter)}
              data-testid="directory-filter"
              className="w-full rounded-lg border border-wc-default bg-wc-surface-input py-1.5 pl-7 pr-2 text-xs text-wc-text-primary placeholder:text-wc-text-faint focus:border-wc-accent focus:outline-none"
            />
          </label>

          <select
            value={listing.sort}
            onChange={(e) => onListOptionsChange({ sort: e.target.value as DirectorySort })}
            aria-label={t(strings.messagesFileViewer.directorySortLabel)}
            data-testid="directory-sort"
            className="shrink-0 rounded-lg border border-wc-default bg-wc-surface-input px-2 py-1.5 text-xs text-wc-text-secondary focus:border-wc-accent focus:outline-none"
          >
            {SORT_OPTIONS.map((o) => (
              <option key={o.value} value={o.value}>
                {t(o.label)}
              </option>
            ))}
          </select>

          <button
            type="button"
            onClick={() => onListOptionsChange({ showHidden: !listing.showHidden })}
            aria-pressed={listing.showHidden}
            data-testid="directory-toggle-hidden"
            title={t(
              listing.showHidden
                ? strings.messagesFileViewer.directoryHideHidden
                : strings.messagesFileViewer.directoryShowHidden,
            )}
            className={cn(
              "inline-flex shrink-0 items-center gap-1.5 rounded-lg border px-2 py-1.5 text-xs font-medium transition",
              listing.showHidden
                ? "border-wc-accent bg-wc-accent/10 text-wc-text-primary"
                : "border-wc-default bg-wc-surface-input text-wc-text-secondary hover:text-wc-text-primary",
            )}
          >
            {listing.showHidden ? <Eye className="h-3.5 w-3.5" /> : <EyeOff className="h-3.5 w-3.5" />}
          </button>
        </div>

        <div className="mt-1.5 flex items-center gap-2 text-[11px] text-wc-text-faint">
          <span data-testid="directory-count" className="tabular-nums">
            {t(strings.messagesFileViewer.directoryEntryCount, {
              loaded: entries.length,
              total: listing.totalEntries,
            })}
          </span>
        </div>
      </div>

      {sortDowngraded && (
        <div className="px-3 pt-2">
          <PreviewNotice message={t(strings.messagesFileViewer.directorySortDowngraded)} tone="info" />
        </div>
      )}
      {listing.truncated && (
        <div className="px-3 pt-2">
          <PreviewNotice message={t(strings.messagesFileViewer.directoryTruncated)} tone="info" />
        </div>
      )}

      <div className="min-h-0 flex-1 overflow-auto">
        {entries.length === 0 ? (
          <EmptyState
            showHidden={listing.showHidden}
            onShowHidden={() => onListOptionsChange({ showHidden: true })}
          />
        ) : filtered.length === 0 ? (
          <p className="px-4 py-8 text-center text-sm text-wc-text-muted" data-testid="directory-filter-empty">
            {t(strings.messagesFileViewer.directoryFilterEmpty, { query: query.trim() })}
          </p>
        ) : (
          <ul className="divide-y divide-wc-default/60">
            {filtered.map((entry) => (
              <EntryRow
                key={entry.name}
                entry={entry}
                // Join from the path the server actually listed, not the
                // model's, so a row can never address a different directory
                // than the one whose entries are on screen.
                onOpen={() => onNavigate(joinPath(listing.resolvedPath || model.resolvedPath, entry.name))}
              />
            ))}
          </ul>
        )}

        {listing.nextPageToken && !query.trim() && (
          <div className="px-3 py-3">
            <button
              type="button"
              onClick={onLoadMore}
              disabled={loadingMore}
              data-testid="directory-load-more"
              className="inline-flex w-full items-center justify-center gap-2 rounded-lg border border-wc-default bg-wc-surface-input px-3 py-2 text-xs font-medium text-wc-text-secondary transition hover:text-wc-text-primary disabled:opacity-60"
            >
              {loadingMore && <Loader2 className="h-3.5 w-3.5 animate-spin" />}
              {t(
                loadingMore
                  ? strings.messagesFileViewer.directoryLoadingMore
                  : strings.messagesFileViewer.directoryLoadMore,
              )}
            </button>
          </div>
        )}
      </div>
    </div>
  );
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
          className="inline-flex items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-2.5 py-1.5 text-xs font-medium text-wc-text-secondary transition hover:text-wc-text-primary"
        >
          <Eye className="h-3.5 w-3.5" />
          {t(strings.messagesFileViewer.directoryShowHidden)}
        </button>
      )}
    </div>
  );
}

function EntryRow({ entry, onOpen }: { entry: DirectoryEntry; onOpen: () => void }) {
  const { t } = useTranslation();
  const isDir = entry.entryType === "directory" || entry.kind === "directory";
  const described = entryDetail(entry);
  const detail = "literal" in described ? described.literal : t(described.key, described.params ?? {});

  const body = (
    <>
      <EntryGlyph entry={entry} />
      <span className="min-w-0 flex-1">
        <span
          className={cn(
            "block truncate font-mono text-[13px]",
            isDir && "font-semibold text-wc-accent",
            entry.entryType === "symlink" && !entry.symlinkBroken && "text-wc-text-secondary",
            entry.symlinkBroken && "text-amber-400 line-through decoration-amber-400/60",
            !entry.canPreview && !entry.symlinkBroken && "text-wc-text-faint",
          )}
        >
          {entry.name}
          {isDir && "/"}
        </span>
        {entry.entryType === "symlink" && entry.symlinkTarget && (
          <span className="block truncate font-mono text-[11px] text-wc-text-faint">
            → {entry.symlinkTarget}
          </span>
        )}
      </span>
      <span className="shrink-0 text-right font-mono text-[11px] tabular-nums text-wc-text-faint">
        {detail}
      </span>
      {entry.canPreview && <ChevronRight className="h-3.5 w-3.5 shrink-0 text-wc-text-faint" />}
    </>
  );

  // Rows are at least 44px tall so they stay a comfortable touch target on a
  // phone, where this drawer is a bottom sheet.
  const rowClass = "flex w-full items-center gap-3 px-3 py-2.5 text-left min-h-[44px]";

  if (!entry.canPreview) {
    return (
      <li>
        <div className={cn(rowClass, "cursor-default opacity-70")} title={detail} data-testid="directory-entry">
          {body}
        </div>
      </li>
    );
  }

  return (
    <li>
      <button type="button" onClick={onOpen} data-testid="directory-entry" className={cn(rowClass, "transition hover:bg-wc-surface-raised")}>
        {body}
      </button>
    </li>
  );
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
    case "markdown":
    case "text":
      return <FileText className={className} />;
    case "code":
      return <FileCode className={className} />;
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
