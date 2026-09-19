import { useState } from "react";
import { ChevronUp, File, Folder } from "lucide-react";

import { AsyncSection } from "../../components/AsyncSection";
import { EmptyState } from "../../components/EmptyState";
import { Button } from "../../components/ui/button";
import { useSnapshotEntries } from "../../hooks/useRuns";
import { formatBytes } from "../../lib/format";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const basename = (path: string) => path.split("/").filter(Boolean).pop() ?? path;
const parentOf = (path: string) => path.split("/").filter(Boolean).slice(0, -1).join("/");

/**
 * Read-only snapshot content browser. Drills into directories one path at a
 * time via BrowseSnapshot (lazy, not an eager recursive tree) so it stays
 * responsive on large snapshots.
 */
export function SnapshotBrowser({
  destinationId,
  snapshotId,
}: {
  destinationId: string;
  snapshotId: string;
}) {
  const { t } = useTranslation();
  const [path, setPath] = useState("");
  const { data, isLoading, isError, refetch } = useSnapshotEntries(destinationId, snapshotId, path);
  const entries = data ?? [];

  return (
    <div data-testid={selectors.snapshot.browser} className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          data-testid={selectors.snapshot.up}
          disabled={path === ""}
          onClick={() => setPath(parentOf(path))}
        >
          <ChevronUp aria-hidden="true" className="me-1 h-4 w-4" />
          {t(strings.snapshot.up)}
        </Button>
        <span className="truncate font-mono text-xs text-app-muted-foreground">/{path}</span>
      </div>

      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={entries.length === 0}
        onRetry={() => void refetch()}
        skeletonRows={3}
        emptyState={<EmptyState title={t(strings.snapshot.empty)} />}
      >
        <ul className="max-h-60 overflow-y-auto rounded-control border border-app-border">
          {entries.map((e) => (
            <li key={e.path} className="border-b border-app-border last:border-b-0">
              {e.isDir ? (
                <button
                  type="button"
                  onClick={() => setPath(e.path)}
                  className="flex w-full items-center gap-2 px-3 py-2 text-start text-sm hover:bg-app-surface-muted"
                >
                  <Folder aria-hidden="true" className="h-4 w-4 shrink-0 text-app-info" />
                  <span className="truncate font-mono">{basename(e.path)}</span>
                </button>
              ) : (
                <div className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
                  <span className="flex min-w-0 items-center gap-2">
                    <File aria-hidden="true" className="h-4 w-4 shrink-0 text-app-muted-foreground" />
                    <span className="truncate font-mono">{basename(e.path)}</span>
                  </span>
                  <span className="shrink-0 text-xs text-app-muted-foreground">{formatBytes(e.sizeBytes)}</span>
                </div>
              )}
            </li>
          ))}
        </ul>
      </AsyncSection>
    </div>
  );
}
