import { useState } from "react";
import { HardDrive, Pencil, Plus, Trash2 } from "lucide-react";

import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/EmptyState";
import { AsyncSection } from "../components/AsyncSection";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { Button } from "../components/ui/button";
import { StatusChip } from "../components/ui/status-chip";
import { UsageBar } from "../components/ui/usage-bar";
import { DestinationFormDialog } from "../features/destinations/DestinationFormDialog";
import { useDeleteDestination, useDestinations } from "../hooks/useDestinations";
import type { Destination } from "../api/destinations";
import { backendSlug, capPolicySlug, usageMeta } from "../lib/status";
import { BACKEND_STRINGS, CAP_POLICY_STRINGS, USAGE_STRINGS } from "../consts/statusStrings";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Destinations surface — create / edit / delete encrypted repositories and
 * watch storage usage approach each cap. Encryption algorithm and secret
 * reference are shown read-only (reassurance that storage is encrypted), never
 * editable. Delete optionally drops the underlying repository, behind a
 * confirmation.
 */
export function DestinationsPage() {
  const { t } = useTranslation();
  const { data, isLoading, isError, refetch } = useDestinations();
  const del = useDeleteDestination();
  const destinations = data ?? [];

  const [createOpen, setCreateOpen] = useState(false);
  const [editing, setEditing] = useState<Destination | null>(null);
  const [deleting, setDeleting] = useState<Destination | null>(null);
  const [deleteRepo, setDeleteRepo] = useState(false);

  const confirmDelete = () => {
    if (!deleting) return;
    del.mutate(
      { id: deleting.id, deleteRepository: deleteRepo },
      {
        onSuccess: () => {
          setDeleting(null);
          setDeleteRepo(false);
        },
      },
    );
  };

  return (
    <section
      data-testid={selectors.pages.destinations}
      aria-labelledby="destinations-heading"
      className="flex flex-col gap-6"
    >
      <div id="destinations-heading">
        <PageHeader
          title={t(strings.layout.nav.destinations)}
          subtitle={t(strings.destinations.subtitle)}
          actions={
            <Button size="sm" data-testid={selectors.destinations.createButton} onClick={() => setCreateOpen(true)}>
              <Plus aria-hidden="true" className="me-1.5 h-4 w-4" />
              {t(strings.destinations.create)}
            </Button>
          }
        />
      </div>

      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={destinations.length === 0}
        onRetry={() => void refetch()}
        emptyState={
          <EmptyState
            icon={HardDrive}
            title={t(strings.destinations.empty)}
            description={t(strings.destinations.emptyHint)}
            action={
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                {t(strings.destinations.create)}
              </Button>
            }
          />
        }
      >
        <ul data-testid={selectors.destinations.list} className="grid gap-3 sm:grid-cols-2">
          {destinations.map((d) => {
            const usage = usageMeta(d.usageState);
            return (
              <li
                key={d.id}
                data-testid={selectors.destinations.row({ id: d.id })}
                className="flex flex-col gap-3 rounded-panel border border-app-border bg-app-surface p-4"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium text-app-foreground">{d.name}</p>
                    <p className="truncate text-xs text-app-muted-foreground">
                      {t(BACKEND_STRINGS[backendSlug(d.backendKind)])}
                    </p>
                  </div>
                  <StatusChip tone={usage.tone} labelKey={USAGE_STRINGS[usage.slug]} />
                </div>

                <p className="truncate font-mono text-xs text-app-muted-foreground">{d.location}</p>
                <UsageBar usageBytes={d.usageBytes} capBytes={d.capBytes} usageState={d.usageState} />

                <dl className="grid grid-cols-[auto,1fr] gap-x-3 gap-y-1 text-xs">
                  {d.repositoryLocation && d.repositoryLocation !== d.location && (
                    <>
                      <dt className="text-app-muted-foreground">{t(strings.destinations.repositoryPath)}</dt>
                      <dd className="truncate font-mono text-app-foreground">{d.repositoryLocation}</dd>
                    </>
                  )}
                  <dt className="text-app-muted-foreground">{t(strings.destinations.policy)}</dt>
                  <dd className="text-app-foreground">{t(CAP_POLICY_STRINGS[capPolicySlug(d.capPolicy)])}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.destinations.encryption)}</dt>
                  <dd className="truncate font-mono text-app-foreground">{d.encryptionAlgorithm || "—"}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.destinations.secretRef)}</dt>
                  <dd className="truncate font-mono text-app-foreground">{d.secretRef || "—"}</dd>
                </dl>

                <div className="flex justify-end gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    aria-label={t(strings.destinations.editTitle)}
                    data-testid={selectors.destinations.editButton}
                    onClick={() => setEditing(d)}
                  >
                    <Pencil aria-hidden="true" className="h-4 w-4" />
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    aria-label={t(strings.destinations.delete)}
                    data-testid={selectors.destinations.deleteButton}
                    onClick={() => setDeleting(d)}
                  >
                    <Trash2 aria-hidden="true" className="h-4 w-4" />
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      </AsyncSection>

      <DestinationFormDialog open={createOpen} onClose={() => setCreateOpen(false)} />
      {editing && (
        <DestinationFormDialog key={editing.id} open destination={editing} onClose={() => setEditing(null)} />
      )}

      <ConfirmDialog
        open={deleting !== null}
        onClose={() => {
          setDeleting(null);
          setDeleteRepo(false);
        }}
        onConfirm={confirmDelete}
        title={t(strings.destinations.deleteTitle)}
        body={t(strings.destinations.deleteBody, { name: deleting?.name ?? "" })}
        confirmLabel={t(strings.destinations.delete)}
        danger
        busy={del.isPending}
        confirmTestId={selectors.destinations.deleteConfirm}
      >
        <label className="mt-3 flex items-start gap-2 text-sm text-app-foreground">
          <input
            type="checkbox"
            data-testid={selectors.destinations.deleteRepoToggle}
            checked={deleteRepo}
            onChange={(e) => setDeleteRepo(e.target.checked)}
            className="mt-0.5 h-4 w-4"
          />
          <span>
            {t(strings.destinations.deleteRepo)}
            <span className="block text-xs text-app-muted-foreground">{t(strings.destinations.deleteRepoHint)}</span>
          </span>
        </label>
      </ConfirmDialog>
    </section>
  );
}
