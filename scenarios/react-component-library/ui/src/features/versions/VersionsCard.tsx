/** @vrooliComponentSource data-display.data-table */
import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { Button } from "../../components/Button";
import { EmptyState } from "../../components/EmptyState";
import { Select } from "../../components/Select";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { versionsClient, type DiffVersionsResponse, type Version } from "../../api/versions";
import { errorMessage } from "../../lib/errorMessage";
import { VersionDiffViewer } from "./VersionDiffViewer";

interface VersionsCardProps {
  componentId: string;
  selectedVersion?: string;
  onSelectVersion?: (version: string | undefined) => void;
  /** Lets the enclosing code workspace own comparison rendering. */
  onCompare?: (diff: DiffVersionsResponse) => void;
}

const EMPTY_VERSIONS: Version[] = [];

/**
 * VersionsCard renders the version-history list for one component and
 * a built-in diff viewer that picks two versions (or one version and
 * an `adoption:<id>` ref) and renders aligned left/right rows.
 *
 * Surface for req 11 (VR-001..003).
 */
export function VersionsCard({
  componentId,
  selectedVersion,
  onSelectVersion,
  onCompare,
}: VersionsCardProps) {
  const { t } = useTranslation();
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");

  const versionsQuery = useQuery({
    queryKey: ["versions", componentId],
    queryFn: () => versionsClient.listVersions({ componentId, limit: 0 }),
  });

  const diffMutation = useMutation({
    mutationFn: () => versionsClient.diffVersions({ componentId, from, to }),
    onSuccess: (diff) => onCompare?.(diff),
  });

  const versions: Version[] = versionsQuery.data?.versions ?? EMPTY_VERSIONS;
  const diff = diffMutation.data;

  const versionOptions = useMemo(
    () => versions.map((v) => v.version).filter((v) => v.length > 0),
    [versions],
  );

  return (
    <section
      data-testid={selectors.versions.card}
      aria-label={t(strings.versions.title)}
      className="mt-space-sm rounded-xl border border-app-border bg-app-surface p-space-sm backdrop-blur-sm"
    >
      <header className="flex items-center justify-between gap-space-xs">
        <h2 className="text-sm font-medium text-app-foreground">{t(strings.versions.title)}</h2>
        {versions.length > 0 && (
          <span className="text-xs text-app-muted-foreground">
            {t(strings.versions.summary, { count: versions.length })}
          </span>
        )}
      </header>

      {versionsQuery.isLoading && (
        <p data-testid={selectors.versions.loading} className="mt-space-xs text-app-foreground">
          {t(strings.versions.loading)}
        </p>
      )}
      {versionsQuery.error && (
        <p data-testid={selectors.versions.error} className="mt-space-xs text-app-danger">
          {errorMessage(versionsQuery.error, t)}
        </p>
      )}
      {!versionsQuery.isLoading && versions.length === 0 && (
        <div data-testid={selectors.versions.empty}>
          <EmptyState
            title={t(strings.versions.empty)}
            className="mt-space-xs border-app-border bg-app-surface text-app-foreground"
          />
        </div>
      )}

      {versions.length > 0 && (
        <>
          <Button
            type="button"
            variant={selectedVersion ? "secondary" : "primary"}
            className="mt-space-xs h-control-tight px-space-xs text-xs"
            onClick={() => onSelectVersion?.(undefined)}
          >
            {t(strings.versions.currentSource)}
          </Button>
          <ul
            data-testid={selectors.versions.list}
            className="mt-space-xs space-y-space-2xs text-sm text-app-foreground"
          >
            {versions.map((v) => (
              <li
                key={v.id}
                data-testid={selectors.versions.item}
                className="rounded-lg border border-app-border p-space-xs"
              >
                <div className="flex items-baseline justify-between gap-space-xs">
                  <span data-testid={selectors.versions.itemVersion} className="font-medium">
                    {v.version
                      ? t(strings.versions.versionLabel, { version: v.version })
                      : "(no @version)"}
                  </span>
                  <span
                    data-testid={selectors.versions.itemSha}
                    className="font-mono text-xs text-app-muted-foreground"
                  >
                    {t(strings.versions.shaLabel, { sha: v.contentSha256.slice(0, 12) })}
                  </span>
                </div>
                <div className="mt-space-3xs flex flex-wrap items-center gap-x-space-xs gap-y-space-3xs text-xs text-app-muted-foreground">
                  <span data-testid={selectors.versions.itemRecordedAt}>
                    {t(strings.versions.createdAt, {
                      when: v.createdAt?.seconds
                        ? new Date(Number(v.createdAt.seconds) * 1000).toLocaleString()
                        : "",
                    })}
                  </span>
                  {v.changelogMd && (
                    <span data-testid={selectors.versions.itemChangelog}>{v.changelogMd}</span>
                  )}
                </div>
                <span data-testid={selectors.versions.itemId} className="sr-only">
                  {v.id}
                </span>
                <Button
                  type="button"
                  variant={selectedVersion === v.version ? "primary" : "secondary"}
                  className="mt-space-2xs h-control-compact px-space-2xs text-xs"
                  onClick={() => onSelectVersion?.(v.version)}
                >
                  {selectedVersion === v.version
                    ? t(strings.versions.viewingVersion)
                    : t(strings.versions.viewVersion)}
                </Button>
              </li>
            ))}
          </ul>
        </>
      )}

      <div
        data-testid={selectors.versions.diff.card}
        className="mt-space-sm rounded-lg border border-app-border bg-app-surface-muted p-space-xs"
      >
        <h3 className="text-xs font-medium text-app-muted-foreground">
          {t(strings.versions.diff.title)}
        </h3>
        <div className="mt-space-2xs flex flex-wrap items-center gap-space-2xs text-xs text-app-muted-foreground">
          <label className="flex items-center gap-space-2xs">
            <span>{t(strings.versions.diff.fromLabel)}</span>
            <Select
              data-testid={selectors.versions.diff.fromSelect}
              value={from}
              onChange={(e) => setFrom(e.target.value)}
              className="min-h-control-tight w-field-short border-app-border bg-app-surface px-space-2xs py-space-3xs text-sm text-app-foreground"
              options={versionOptions.map((version) => ({ value: version, label: version }))}
              placeholder="—"
            />
          </label>
          <label className="flex items-center gap-space-2xs">
            <span>{t(strings.versions.diff.toLabel)}</span>
            <Select
              data-testid={selectors.versions.diff.toSelect}
              value={to}
              onChange={(e) => setTo(e.target.value)}
              className="min-h-control-tight w-field-short border-app-border bg-app-surface px-space-2xs py-space-3xs text-sm text-app-foreground"
              options={versionOptions.map((version) => ({ value: version, label: version }))}
              placeholder="—"
            />
          </label>
          <Button
            data-testid={selectors.versions.diff.runButton}
            onClick={() => diffMutation.mutate()}
            disabled={!from || !to || diffMutation.isPending}
            className="h-control-compact px-space-xs text-xs"
          >
            {diffMutation.isPending
              ? t(strings.versions.diff.running)
              : t(strings.versions.diff.runAction)}
          </Button>
        </div>

        {diffMutation.error && (
          <p
            data-testid={selectors.versions.diff.error}
            className="mt-space-2xs text-xs text-app-danger"
          >
            {errorMessage(diffMutation.error, t)}
          </p>
        )}

        {!diff && !diffMutation.isPending && !diffMutation.error && (
          <p
            data-testid={selectors.versions.diff.empty}
            className="mt-space-2xs text-xs text-app-muted-foreground"
          >
            {t(strings.versions.diff.empty)}
          </p>
        )}

        {diff && !onCompare && (
          <>
            <p
              data-testid={selectors.versions.diff.summary}
              className="mt-space-2xs text-xs text-app-muted-foreground"
            >
              {t(strings.versions.diff.summary, {
                from: diff.fromLabel,
                to: diff.toLabel,
                additions: diff.additions,
                removals: diff.removals,
                rows: diff.rows.length,
              })}
            </p>
            <div data-testid={selectors.versions.diff.table}>
              <VersionDiffViewer rows={diff.rows} />
            </div>
          </>
        )}
      </div>
    </section>
  );
}
