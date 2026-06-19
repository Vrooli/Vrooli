import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { Timestamp } from "@bufbuild/protobuf/wkt";
import type { RecoveryEvent } from "@vrooli/proto-types/tunnel-manager/v1/recovery/recovery_pb";

import { Button } from "../../components/ui/button";
import { QueryState } from "../../components/ui/QueryState";
import { StatusBadge } from "../../components/ui/StatusBadge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { recoveryClient } from "../../api/recovery";
import {
  recoveryStatusLabel,
  recoveryStatusTone,
  eventOutcomeLabel,
  eventOutcomeTone,
} from "./labels";

const STATE_KEY = ["recovery-state"] as const;
const EVENTS_KEY = ["recovery-events"] as const;

function whenLabel(t: ReturnType<typeof useTranslation>["t"], ts?: Timestamp): string {
  if (!ts) return t(strings.common.never);
  return formatDate(timestampDate(ts), { dateStyle: "medium", timeStyle: "short" });
}

/**
 * RecoveryPanel renders the live auto-recovery state machine, the durable event
 * timeline, and the operator manual-recover action (guarded by a confirm
 * dialog, with an optional force toggle to bypass the circuit breaker).
 */
export function RecoveryPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [confirming, setConfirming] = useState(false);
  const [force, setForce] = useState(false);

  const stateQuery = useQuery({ queryKey: STATE_KEY, queryFn: () => recoveryClient.getState({}) });
  const eventsQuery = useQuery({ queryKey: EVENTS_KEY, queryFn: () => recoveryClient.listEvents({}) });

  const recoverMutation = useMutation({
    mutationFn: () => recoveryClient.recover({ force }),
    onSuccess: () => {
      setConfirming(false);
      void queryClient.invalidateQueries({ queryKey: STATE_KEY });
      void queryClient.invalidateQueries({ queryKey: EVENTS_KEY });
    },
  });

  const state = stateQuery.data?.state;
  const events = eventsQuery.data?.events ?? [];

  return (
    <div className="flex flex-col gap-6">
      <section
        data-testid={selectors.recovery.statePanel}
        className="flex flex-col gap-4 rounded-panel border border-app-border bg-app-surface p-4"
      >
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">{t(strings.recovery.stateHeading)}</h2>
          <Button
            variant="outline"
            data-testid={selectors.recovery.refreshButton}
            onClick={() => void stateQuery.refetch()}
          >
            {t(strings.common.refresh)}
          </Button>
        </div>
        <QueryState
          isLoading={stateQuery.isLoading}
          error={stateQuery.error}
          loadingLabel={t(strings.recovery.loading)}
          errorLabel={t(strings.recovery.error)}
        >
          {state && (
            <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <div className="flex flex-col gap-1">
                <dt className="text-xs uppercase text-app-muted-foreground">{t(strings.recovery.statusLabel)}</dt>
                <dd>
                  <StatusBadge tone={recoveryStatusTone(state.status)} data-testid={selectors.recovery.statusValue}>
                    {t(recoveryStatusLabel(state.status))}
                  </StatusBadge>
                </dd>
              </div>
              <div className="flex flex-col gap-1">
                <dt className="text-xs uppercase text-app-muted-foreground">{t(strings.recovery.circuitOpenLabel)}</dt>
                <dd>
                  <StatusBadge
                    tone={state.circuitOpen ? "danger" : "success"}
                    data-testid={selectors.recovery.circuitValue}
                  >
                    {state.circuitOpen ? t(strings.recovery.circuitOpen) : t(strings.recovery.circuitClosed)}
                  </StatusBadge>
                </dd>
              </div>
              <Stat label={t(strings.recovery.consecFailuresLabel)} value={state.consecFailures} />
              <Stat label={t(strings.recovery.backoffLabel)} value={state.backoffLevel} />
              <Stat label={t(strings.recovery.failedRecoveriesLabel)} value={state.failedRecoveries} />
              <Stat label={t(strings.recovery.lastCheckLabel)} value={whenLabel(t, state.lastCheck)} />
              <Stat label={t(strings.recovery.lastRecoveryLabel)} value={whenLabel(t, state.lastRecovery)} />
              <Stat label={t(strings.recovery.nextRetryLabel)} value={whenLabel(t, state.nextRetryAfter)} />
            </dl>
          )}
        </QueryState>

        <div className="flex flex-wrap items-center gap-3 border-t border-app-border pt-4">
          <Button data-testid={selectors.recovery.recoverButton} onClick={() => setConfirming(true)}>
            {t(strings.recovery.recoverButton)}
          </Button>
          <label className="flex items-center gap-2 text-sm text-app-muted-foreground">
            <input
              type="checkbox"
              data-testid={selectors.recovery.forceToggle}
              checked={force}
              onChange={(e) => setForce(e.target.checked)}
            />
            {t(strings.recovery.recoverForce)}
          </label>
        </div>
        {recoverMutation.error && (
          <p data-testid={selectors.recovery.actionError} role="alert" className="text-sm text-app-danger">
            {t(strings.recovery.recoverError)}
          </p>
        )}
      </section>

      {confirming && (
        <div
          data-testid={selectors.recovery.confirmDialog}
          role="alertdialog"
          aria-modal="true"
          aria-label={t(strings.recovery.recoverConfirmTitle)}
          className="flex flex-col gap-3 rounded-panel border border-app-warning/40 bg-app-surface p-4"
        >
          <h3 className="font-semibold">{t(strings.recovery.recoverConfirmTitle)}</h3>
          <p className="text-sm text-app-muted-foreground">{t(strings.recovery.recoverConfirmBody)}</p>
          <div className="flex gap-2">
            <Button
              data-testid={selectors.recovery.confirmButton}
              disabled={recoverMutation.isPending}
              onClick={() => recoverMutation.mutate()}
            >
              {t(strings.recovery.recoverConfirm)}
            </Button>
            <Button
              variant="outline"
              data-testid={selectors.recovery.cancelButton}
              onClick={() => setConfirming(false)}
            >
              {t(strings.recovery.recoverCancel)}
            </Button>
          </div>
        </div>
      )}

      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t(strings.recovery.timelineHeading)}</h2>
        <QueryState
          isLoading={eventsQuery.isLoading}
          error={eventsQuery.error}
          isEmpty={events.length === 0}
          errorLabel={t(strings.recovery.timelineError)}
          emptyLabel={t(strings.recovery.timelineEmpty)}
        >
          <div className="overflow-x-auto rounded-panel border border-app-border">
            <table data-testid={selectors.recovery.timeline} className="w-full text-left text-sm">
              <thead className="border-b border-app-border bg-app-surface-muted text-xs uppercase text-app-muted-foreground">
                <tr>
                  <th className="px-3 py-2">{t(strings.recovery.colWhen)}</th>
                  <th className="px-3 py-2">{t(strings.recovery.colTrigger)}</th>
                  <th className="px-3 py-2">{t(strings.recovery.colAction)}</th>
                  <th className="px-3 py-2">{t(strings.recovery.colOutcome)}</th>
                  <th className="px-3 py-2">{t(strings.recovery.colAttempt)}</th>
                </tr>
              </thead>
              <tbody>
                {events.map((event: RecoveryEvent) => (
                  <tr
                    key={event.id}
                    data-testid={selectors.recovery.timelineRow}
                    className="border-b border-app-border last:border-0"
                  >
                    <td className="px-3 py-2">{whenLabel(t, event.createdAt)}</td>
                    <td className="px-3 py-2">{event.trigger}</td>
                    <td className="px-3 py-2">{event.action}</td>
                    <td className="px-3 py-2">
                      <StatusBadge tone={eventOutcomeTone(event.outcome)}>
                        {t(eventOutcomeLabel(event.outcome))}
                      </StatusBadge>
                    </td>
                    <td className="px-3 py-2 tabular-nums">{event.attempt}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </QueryState>
      </section>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="flex flex-col gap-1">
      <dt className="text-xs uppercase text-app-muted-foreground">{label}</dt>
      <dd className="font-medium tabular-nums">{value}</dd>
    </div>
  );
}
