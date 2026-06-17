import { useMemo, useState } from "react";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { Copy, Download, FileText, Image as ImageIcon, LayoutGrid, List as ListIcon, Trash2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { formatBytes } from "../../lib/formatBytes";
import { errorMessage } from "../../lib/errorMessage";
import { useSession } from "../session/SessionProvider";
import { downloadItem, ItemKind, Retention, type Item } from "../../api/transfer";
import { useDeleteItemMutation, useItemsQuery } from "./queries";
import { ItemThumbnail } from "./ItemThumbnail";

type SortKey = "newest" | "name" | "size";
type KindFilter = "all" | "file" | "text";
type ViewMode = "cards" | "list";

const RETENTION_LABEL = {
  [Retention.UNSPECIFIED]: strings.transfer.retention.held,
  [Retention.LIVE]: strings.transfer.retention.live,
  [Retention.HELD]: strings.transfer.retention.held,
  [Retention.PINNED]: strings.transfer.retention.pinned,
} as const satisfies Record<Retention, string>;

/**
 * The Receive half (top). Lists items the device may pull with client-side
 * search / sort / kind-filter and a card↔list view toggle. Files download via
 * the device-token-aware fetch→save path; text snippets render inline with a
 * copy button; owner-origin items expose remove.
 */
export function ReceivePanel() {
  const { t } = useTranslation();
  const { isPaired, session } = useSession();
  const itemsQuery = useItemsQuery(isPaired);
  const deleteItem = useDeleteItemMutation();

  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<SortKey>("newest");
  const [filter, setFilter] = useState<KindFilter>("all");
  const [view, setView] = useState<ViewMode>("cards");
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const ownDeviceId = session.device?.id ?? "";

  const items = useMemo(() => {
    const all = itemsQuery.data ?? [];
    const needle = search.trim().toLowerCase();
    const filtered = all.filter((item) => {
      if (filter === "file" && item.kind !== ItemKind.FILE) return false;
      if (filter === "text" && item.kind !== ItemKind.TEXT) return false;
      if (!needle) return true;
      return (
        item.name.toLowerCase().includes(needle) ||
        item.text.toLowerCase().includes(needle)
      );
    });
    const sorted = [...filtered];
    sorted.sort((a, b) => {
      if (sort === "name") return a.name.localeCompare(b.name);
      if (sort === "size") return Number(b.sizeBytes - a.sizeBytes);
      const at = a.createdAt ? timestampDate(a.createdAt).getTime() : 0;
      const bt = b.createdAt ? timestampDate(b.createdAt).getTime() : 0;
      return bt - at;
    });
    return sorted;
  }, [itemsQuery.data, search, filter, sort]);

  const handleCopy = async (item: Item) => {
    try {
      await navigator.clipboard.writeText(item.text);
      setCopiedId(item.id);
      window.setTimeout(() => setCopiedId((id) => (id === item.id ? null : id)), 1500);
    } catch {
      // Clipboard can be blocked; the snippet is still visible to select manually.
    }
  };

  return (
    <section
      data-testid={selectors.receive.panel}
      aria-labelledby="receive-heading"
      className="flex h-full min-h-0 flex-col gap-3 border-b-4 border-app-primary bg-app-surface p-4"
    >
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 id="receive-heading" className="text-lg font-semibold text-app-primary">
            {t(strings.transfer.receive.heading)}
          </h2>
          <p className="text-xs text-app-muted-foreground">
            {t(strings.transfer.receive.description)}
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Input
            data-testid={selectors.receive.search}
            type="search"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            aria-label={t(strings.transfer.receive.searchLabel)}
            placeholder={t(strings.transfer.receive.searchPlaceholder)}
            className="h-9 w-44"
          />
          <label className="sr-only" htmlFor="receive-sort">
            {t(strings.transfer.receive.sortLabel)}
          </label>
          <select
            id="receive-sort"
            data-testid={selectors.receive.sort}
            value={sort}
            onChange={(e) => setSort(e.target.value as SortKey)}
            aria-label={t(strings.transfer.receive.sortLabel)}
            className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
          >
            <option value="newest">{t(strings.transfer.receive.sortNewest)}</option>
            <option value="name">{t(strings.transfer.receive.sortName)}</option>
            <option value="size">{t(strings.transfer.receive.sortSize)}</option>
          </select>
          <label className="sr-only" htmlFor="receive-filter">
            {t(strings.transfer.receive.filterLabel)}
          </label>
          <select
            id="receive-filter"
            data-testid={selectors.receive.filter}
            value={filter}
            onChange={(e) => setFilter(e.target.value as KindFilter)}
            aria-label={t(strings.transfer.receive.filterLabel)}
            className="h-9 rounded-control border border-app-border bg-app-surface px-2 text-sm text-app-foreground"
          >
            <option value="all">{t(strings.transfer.receive.filterAll)}</option>
            <option value="file">{t(strings.transfer.receive.filterFile)}</option>
            <option value="text">{t(strings.transfer.receive.filterText)}</option>
          </select>
          <Button
            data-testid={selectors.receive.viewToggle}
            variant="outline"
            size="sm"
            onClick={() => setView((v) => (v === "cards" ? "list" : "cards"))}
            aria-pressed={view === "list"}
            aria-label={view === "cards" ? t(strings.transfer.receive.viewList) : t(strings.transfer.receive.viewCards)}
            title={view === "cards" ? t(strings.transfer.receive.viewList) : t(strings.transfer.receive.viewCards)}
          >
            {view === "cards" ? (
              <ListIcon aria-hidden="true" className="h-4 w-4" />
            ) : (
              <LayoutGrid aria-hidden="true" className="h-4 w-4" />
            )}
          </Button>
        </div>
      </header>

      <div className="min-h-0 flex-1 overflow-auto">
        {itemsQuery.isLoading && (
          <p data-testid={selectors.receive.loading} className="text-sm text-app-muted-foreground">
            {t(strings.transfer.receive.loading)}
          </p>
        )}
        {itemsQuery.error && (
          <p data-testid={selectors.receive.error} className="text-sm text-app-danger">
            {errorMessage(itemsQuery.error, t)}
          </p>
        )}
        {itemsQuery.data && items.length === 0 && (
          <p data-testid={selectors.receive.empty} className="text-sm text-app-muted-foreground">
            {t(strings.transfer.receive.empty)}
          </p>
        )}
        {items.length > 0 && (
          <ul
            data-testid={selectors.receive.list}
            className={
              view === "cards"
                ? "grid gap-3 sm:grid-cols-2 lg:grid-cols-3"
                : "flex flex-col gap-2"
            }
          >
            {items.map((item) => {
              const isImage = item.kind === ItemKind.FILE && item.hasThumbnail;
              const canRemove = item.originDeviceId === ownDeviceId;
              return (
                <li
                  key={item.id}
                  data-testid={selectors.receive.item({ id: item.id })}
                  className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-background p-3"
                >
                  <div className="flex items-start gap-3">
                    {isImage ? (
                      <ItemThumbnail itemId={item.id} alt={item.name} />
                    ) : (
                      <span className="grid h-10 w-10 shrink-0 place-items-center rounded-control bg-app-surface-muted text-app-muted-foreground">
                        {item.kind === ItemKind.TEXT ? (
                          <FileText aria-hidden="true" className="h-5 w-5" />
                        ) : (
                          <ImageIcon aria-hidden="true" className="h-5 w-5" />
                        )}
                      </span>
                    )}
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-app-foreground" title={item.name}>
                        {item.name || (item.kind === ItemKind.TEXT ? item.text.slice(0, 40) : item.id)}
                      </p>
                      <p className="text-xs text-app-muted-foreground">
                        {item.kind === ItemKind.FILE ? formatBytes(item.sizeBytes) : null}{" "}
                        <span className="rounded-control bg-app-surface-muted px-1.5 py-0.5">
                          {t(RETENTION_LABEL[item.retention])}
                        </span>
                      </p>
                      {item.expiresAt && (
                        <p className="text-xs text-app-muted-foreground">
                          {t(strings.transfer.receive.expiresLabel, {
                            when: formatDate(timestampDate(item.expiresAt), {
                              dateStyle: "short",
                              timeStyle: "short",
                            }),
                          })}
                        </p>
                      )}
                    </div>
                  </div>

                  {item.kind === ItemKind.TEXT && (
                    <pre className="max-h-24 overflow-auto whitespace-pre-wrap break-words rounded-control bg-app-surface-muted p-2 text-xs text-app-foreground">
                      {item.text}
                    </pre>
                  )}

                  <div className="flex flex-wrap items-center gap-2">
                    {item.kind === ItemKind.FILE && (
                      <Button
                        data-testid={selectors.receive.download({ id: item.id })}
                        variant="outline"
                        size="sm"
                        onClick={() => void downloadItem(item.id, item.name)}
                      >
                        <Download aria-hidden="true" className="me-1.5 h-4 w-4" />
                        {t(strings.transfer.receive.download)}
                      </Button>
                    )}
                    {item.kind === ItemKind.TEXT && (
                      <Button
                        data-testid={selectors.receive.copy({ id: item.id })}
                        variant="outline"
                        size="sm"
                        onClick={() => void handleCopy(item)}
                      >
                        <Copy aria-hidden="true" className="me-1.5 h-4 w-4" />
                        {copiedId === item.id
                          ? t(strings.transfer.receive.copied)
                          : t(strings.transfer.receive.copy)}
                      </Button>
                    )}
                    {canRemove && (
                      <Button
                        data-testid={selectors.receive.remove({ id: item.id })}
                        variant="outline"
                        size="sm"
                        onClick={() => deleteItem.mutate(item.id)}
                        disabled={deleteItem.isPending}
                      >
                        <Trash2 aria-hidden="true" className="me-1.5 h-4 w-4" />
                        {t(strings.transfer.receive.remove)}
                      </Button>
                    )}
                  </div>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
  );
}
