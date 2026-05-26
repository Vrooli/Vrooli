import { useState } from "react";
import { Plus, RotateCcw } from "lucide-react";

import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/EmptyState";
import { AsyncSection } from "../components/AsyncSection";
import { Button } from "../components/ui/button";
import { StatusChip } from "../components/ui/status-chip";
import { RestoreFlowDialog } from "../features/restores/RestoreFlowDialog";
import { useRestores } from "../hooks/useRestores";
import { RestoreMode } from "../api/restores";
import { restoreStatusMeta } from "../lib/status";
import { RESTORE_STATUS_STRINGS } from "../consts/statusStrings";
import { formatAge } from "../lib/format";
import { tsToDate } from "../lib/proto";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * Restores surface — verify and restore history plus the launcher for a new
 * verify/restore. Verify is the encouraged, non-destructive proof of
 * restorability; restore is confirmation-gated inside the flow.
 */
export function RestoresPage() {
  const { t } = useTranslation();
  const { data, isLoading, isError, refetch } = useRestores();
  const restores = data ?? [];
  const [flowOpen, setFlowOpen] = useState(false);
  const never = t(strings.common.never);

  return (
    <section data-testid={selectors.pages.restores} aria-labelledby="restores-heading" className="flex flex-col gap-6">
      <div id="restores-heading">
        <PageHeader
          title={t(strings.layout.nav.restores)}
          subtitle={t(strings.restores.subtitle)}
          actions={
            <Button size="sm" data-testid={selectors.restores.startButton} onClick={() => setFlowOpen(true)}>
              <Plus aria-hidden="true" className="me-1.5 h-4 w-4" />
              {t(strings.restores.start)}
            </Button>
          }
        />
      </div>

      <AsyncSection
        isLoading={isLoading}
        isError={isError}
        isEmpty={restores.length === 0}
        onRetry={() => void refetch()}
        emptyState={
          <EmptyState
            icon={RotateCcw}
            title={t(strings.restores.empty)}
            description={t(strings.restores.emptyHint)}
            action={
              <Button size="sm" onClick={() => setFlowOpen(true)}>
                {t(strings.restores.start)}
              </Button>
            }
          />
        }
      >
        <ul data-testid={selectors.restores.list} className="flex flex-col divide-y divide-app-border rounded-panel border border-app-border bg-app-surface">
          {restores.map((r) => {
            const meta = restoreStatusMeta(r.status);
            const isVerify = r.mode === RestoreMode.VERIFY;
            return (
              <li
                key={r.id}
                data-testid={selectors.restores.row({ id: r.id })}
                className="flex flex-wrap items-center gap-x-3 gap-y-1 p-3"
              >
                <span className="min-w-0 flex-1 truncate font-mono text-sm text-app-foreground">{r.targetId}</span>
                <span className="text-xs text-app-muted-foreground">
                  {isVerify ? t(strings.restores.modeVerify) : t(strings.restores.modeRestore)}
                </span>
                <StatusChip tone={meta.tone} labelKey={RESTORE_STATUS_STRINGS[meta.slug]} />
                {r.checksum && (
                  <span className="hidden max-w-[10rem] truncate font-mono text-xs text-app-muted-foreground sm:inline">
                    {r.checksum}
                  </span>
                )}
                <span className="text-xs text-app-muted-foreground">{formatAge(tsToDate(r.requestedAt), never)}</span>
              </li>
            );
          })}
        </ul>
      </AsyncSection>

      <RestoreFlowDialog open={flowOpen} onClose={() => setFlowOpen(false)} />
    </section>
  );
}
