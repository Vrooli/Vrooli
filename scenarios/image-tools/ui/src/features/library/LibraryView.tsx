import { useCallback, useMemo, useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { NavLink } from "react-router-dom";
import { Download, Images } from "lucide-react";

import { Button } from "../../components/ui/button";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { blobUrl, fetchBlob } from "../../api/client";
import { jobsClient } from "../../api/jobs";
import { errorMessage } from "../../lib/errorMessage";
import { useTranslation } from "../../i18n";
import { operationLabelKey } from "../workspace/operationLabel";
import { imageOutputs, type OutputItem } from "./outputs";
import { useReopenOutput } from "./useReopenOutput";

const JOBS_QUERY_KEY = ["jobs"] as const;

/** Fetch an output's bytes and trigger a browser download. */
async function downloadOutput(item: OutputItem): Promise<void> {
  const blob = await fetchBlob(item.resultRef);
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = item.resultRef.split("/").pop() || `${item.jobId}.png`;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

/**
 * Library — the outputs gallery. Derived from `ListJobs` (succeeded jobs with a
 * result blob, per the Stage-4 contract decision — no separate outputs
 * backend); each item renders a thumbnail you can open back into the Workspace,
 * filter by operation, multi-select, and bulk-download. Loading / error / empty
 * / no-match states are all handled. (Output deletion needs a backend verb
 * image-tools doesn't expose yet — see PROGRESS; outputs stay user-owned and
 * removable on disk.)
 */
export function LibraryView() {
  const { t } = useTranslation();
  const reopen = useReopenOutput();
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<ReadonlySet<string>>(() => new Set<string>());

  const jobsQuery = useQuery({
    queryKey: JOBS_QUERY_KEY,
    queryFn: () => jobsClient.listJobs({ limit: 100 }),
  });

  const allItems = useMemo(() => imageOutputs(jobsQuery.data?.jobs ?? []), [jobsQuery.data]);

  const items = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) {
      return allItems;
    }
    return allItems.filter((item) => {
      const labelKey = operationLabelKey(item.operation);
      const label = labelKey ? t(labelKey) : item.operation;
      return label.toLowerCase().includes(q) || item.operation.toLowerCase().includes(q);
    });
  }, [allItems, query, t]);

  const toggle = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const allSelected = items.length > 0 && items.every((item) => selected.has(item.jobId));
  const selectAll = useCallback(() => {
    setSelected(allSelected ? new Set<string>() : new Set(items.map((item) => item.jobId)));
  }, [allSelected, items]);

  const clearSelection = useCallback(() => setSelected(new Set<string>()), []);

  const downloadSelected = useCallback(() => {
    const targets = items.filter((item) => selected.has(item.jobId));
    // Sequential (promise chain, not await-in-loop) so the browser doesn't
    // throttle a burst of simultaneous downloads.
    void targets.reduce((chain, item) => chain.then(() => downloadOutput(item)), Promise.resolve());
  }, [items, selected]);

  const selectedCount = items.filter((item) => selected.has(item.jobId)).length;

  let body: ReactNode;
  if (jobsQuery.isLoading) {
    body = (
      <p data-testid={selectors.library.loading} className="text-app-foreground">
        {t(strings.library.loading)}
      </p>
    );
  } else if (jobsQuery.error) {
    body = (
      <p data-testid={selectors.library.error} className="text-app-danger">
        {errorMessage(jobsQuery.error, t)}
      </p>
    );
  } else if (allItems.length === 0) {
    body = (
      <EmptyState
        testId={selectors.library.empty}
        Icon={Images}
        title={t(strings.library.empty)}
        action={
          <Button asChild variant="outline">
            <NavLink to="/workspace">{t(strings.library.tryWorkspace)}</NavLink>
          </Button>
        }
      />
    );
  } else if (items.length === 0) {
    body = (
      <p data-testid={selectors.library.noMatches} className="text-app-muted-foreground">
        {t(strings.library.noMatches)}
      </p>
    );
  } else {
    body = (
      <ul
        data-testid={selectors.library.grid}
        className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4"
      >
        {items.map((item, index) => {
          const labelKey = operationLabelKey(item.operation);
          const label = labelKey ? t(labelKey) : item.operation;
          return (
            <li
              key={item.jobId}
              data-testid={selectors.library.item({ index: index + 1 })}
              className="flex flex-col gap-2 rounded-panel border border-app-border bg-app-surface p-2"
            >
              <div className="relative">
                <img
                  src={blobUrl(item.resultRef)}
                  alt={t(strings.library.thumbnailAlt, { operation: label })}
                  loading="lazy"
                  className="aspect-square w-full rounded-control object-cover animate-develop"
                  onError={(e) => {
                    e.currentTarget.closest("li")?.setAttribute("hidden", "true");
                  }}
                />
                <label className="absolute left-1.5 top-1.5 flex h-6 w-6 items-center justify-center rounded-control bg-app-surface/90">
                  <span className="sr-only">{label}</span>
                  <input
                    type="checkbox"
                    data-testid={selectors.library.select({ index: index + 1 })}
                    checked={selected.has(item.jobId)}
                    onChange={() => toggle(item.jobId)}
                    className="h-4 w-4 accent-app-primary"
                  />
                </label>
              </div>
              <div className="flex items-center justify-between gap-2">
                <span className="truncate text-xs font-medium text-app-foreground">{label}</span>
                <button
                  type="button"
                  data-testid={selectors.library.open({ index: index + 1 })}
                  onClick={() => void reopen(item)}
                  className="shrink-0 text-xs font-medium text-app-primary hover:underline"
                >
                  {t(strings.library.open)}
                </button>
              </div>
            </li>
          );
        })}
      </ul>
    );
  }

  return (
    <div data-testid={selectors.library.root} className="flex flex-col gap-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-app-muted-foreground">{t(strings.library.description)}</p>
        <Input
          data-testid={selectors.library.search}
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          aria-label={t(strings.library.searchLabel)}
          placeholder={t(strings.library.searchPlaceholder)}
          className="sm:w-64"
        />
      </div>

      {allItems.length > 0 ? (
        <div className="flex flex-wrap items-center gap-3 text-xs text-app-muted-foreground">
          <span data-testid={selectors.library.count}>
            {t(strings.library.count, { count: items.length })}
          </span>
          <button
            type="button"
            data-testid={selectors.library.selectAll}
            onClick={selectAll}
            className="font-medium text-app-primary hover:underline"
          >
            {t(strings.library.selectAll)}
          </button>
          {selectedCount > 0 ? (
            <>
              <span>{t(strings.library.selected, { count: selectedCount })}</span>
              <button
                type="button"
                data-testid={selectors.library.clearSelection}
                onClick={clearSelection}
                className="font-medium text-app-primary hover:underline"
              >
                {t(strings.library.clearSelection)}
              </button>
              <Button size="sm" data-testid={selectors.library.downloadSelected} onClick={downloadSelected}>
                <Download aria-hidden="true" className="me-2 h-4 w-4" />
                {t(strings.library.downloadSelected)}
              </Button>
            </>
          ) : null}
        </div>
      ) : null}

      {body}
    </div>
  );
}
