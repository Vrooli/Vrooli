import { useMemo, useState } from "react";

import { Dialog } from "../../components/ui/dialog";
import { Field } from "../../components/ui/field";
import { Input } from "../../components/ui/input";
import { Select } from "../../components/ui/select";
import { Button } from "../../components/ui/button";
import { SnapshotBrowser } from "../snapshot/SnapshotBrowser";
import { AuditReport } from "../audits/AuditReport";
import { useRuns } from "../../hooks/useRuns";
import { useRunAudit } from "../../hooks/useAudits";
import { useVerifyTarget, useRestoreTarget } from "../../hooks/useRestores";
import { TargetOutcomeStatus } from "../../api/runs";
import { formatAge } from "../../lib/format";
import { tsToDate } from "../../lib/proto";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

interface SnapshotChoice {
  key: string;
  targetId: string;
  destinationId: string;
  snapshotId: string;
  startedAt: Date | undefined;
}

/**
 * The shared restore/verify flow. Snapshots are sourced from successful run
 * outcomes (the contract has no list-snapshots RPC). Verify is one-click and
 * non-destructive; restore is gated behind an explicit confirmation that spells
 * out the target, snapshot, and exact destination path being written.
 */
export function RestoreFlowDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const runs = useRuns();
  const verify = useVerifyTarget();
  const restore = useRestoreTarget();
  const audit = useRunAudit();

  const choices = useMemo<SnapshotChoice[]>(() => {
    const out: SnapshotChoice[] = [];
    for (const run of runs.data ?? []) {
      for (const o of run.outcomes) {
        if (o.status === TargetOutcomeStatus.SUCCEEDED && o.snapshotId) {
          out.push({
            key: `${o.targetId}|${o.destinationId}|${o.snapshotId}`,
            targetId: o.targetId,
            destinationId: o.destinationId,
            snapshotId: o.snapshotId,
            startedAt: tsToDate(run.startedAt),
          });
        }
      }
    }
    return out;
  }, [runs.data]);

  const [selectedKey, setSelectedKey] = useState("");
  const [showBrowse, setShowBrowse] = useState(false);
  const [location, setLocation] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [touched, setTouched] = useState(false);

  const selected = choices.find((c) => c.key === selectedKey);
  const locationError = touched && !location.trim() ? t(strings.restores.locationRequired) : undefined;

  const close = () => {
    setSelectedKey("");
    setShowBrowse(false);
    setLocation("");
    setConfirming(false);
    setTouched(false);
    verify.reset();
    restore.reset();
    audit.reset();
    onClose();
  };

  const runAudit = () => {
    if (!selected) return;
    audit.mutate({
      targetId: selected.targetId,
      destinationId: selected.destinationId,
      snapshotId: selected.snapshotId,
    });
  };

  const runVerify = () => {
    if (!selected) return;
    verify.mutate(
      { targetId: selected.targetId, destinationId: selected.destinationId, snapshotId: selected.snapshotId },
      { onSuccess: close },
    );
  };

  const beginRestore = () => {
    setTouched(true);
    if (!selected || !location.trim()) return;
    setConfirming(true);
  };

  const confirmRestore = () => {
    if (!selected) return;
    restore.mutate(
      {
        targetId: selected.targetId,
        destinationId: selected.destinationId,
        snapshotId: selected.snapshotId,
        location: location.trim(),
      },
      { onSuccess: close },
    );
  };

  const busy = verify.isPending || restore.isPending;

  return (
    <Dialog
      open={open}
      onClose={close}
      title={t(strings.restores.start)}
      footer={
        <Button variant="outline" size="sm" onClick={close} disabled={busy}>
          {t(strings.common.close)}
        </Button>
      }
    >
      {confirming && selected ? (
        <div data-testid={selectors.restores.restoreConfirm} className="flex flex-col gap-3">
          <p className="text-sm text-app-foreground">
            {t(strings.restores.restoreConfirmBody, {
              target: selected.targetId,
              snapshot: selected.snapshotId,
              location: location.trim(),
            })}
          </p>
          {restore.isError && <p className="text-sm text-app-danger">{t(strings.restores.restoreError)}</p>}
          <div className="flex justify-end gap-2">
            <Button variant="outline" size="sm" onClick={() => setConfirming(false)} disabled={busy}>
              {t(strings.common.cancel)}
            </Button>
            <Button
              size="sm"
              data-testid={selectors.restores.restoreConfirmButton}
              onClick={confirmRestore}
              disabled={busy}
              className="bg-app-danger text-white hover:brightness-95"
            >
              {busy ? t(strings.common.saving) : t(strings.restores.restore)}
            </Button>
          </div>
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          <Field label={t(strings.restores.chooseSnapshot)} hint={t(strings.restores.chooseSnapshotHint)}>
            {(p) =>
              choices.length === 0 ? (
                <p className="text-xs text-app-muted-foreground">{t(strings.restores.noSnapshots)}</p>
              ) : (
                <Select
                  {...p}
                  value={selectedKey}
                  onChange={(e) => {
                    setSelectedKey(e.target.value);
                    setShowBrowse(false);
                  }}
                >
                  <option value="" disabled>
                    {t(strings.restores.chooseSnapshot)}
                  </option>
                  {choices.map((c) => (
                    <option key={c.key} value={c.key}>
                      {`${c.targetId} · ${c.snapshotId.slice(0, 12)} · ${formatAge(c.startedAt, t(strings.common.never))}`}
                    </option>
                  ))}
                </Select>
              )
            }
          </Field>

          {selected && (
            <>
              <div className="flex flex-wrap items-center gap-2">
                <Button variant="outline" size="sm" onClick={() => setShowBrowse((v) => !v)}>
                  {t(strings.restores.browse)}
                </Button>
                <Button
                  size="sm"
                  data-testid={selectors.restores.verifyButton}
                  onClick={runVerify}
                  disabled={busy}
                >
                  {verify.isPending ? t(strings.common.saving) : t(strings.restores.verify)}
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  data-testid={selectors.audits.runButton}
                  onClick={runAudit}
                  disabled={busy || audit.isPending}
                >
                  {audit.isPending ? t(strings.audits.running) : t(strings.audits.action)}
                </Button>
              </div>
              <p className="text-xs text-app-muted-foreground">{t(strings.restores.verifyHint)}</p>
              {verify.isError && <p className="text-sm text-app-danger">{t(strings.restores.verifyError)}</p>}

              {(audit.isPending || audit.isError || audit.data) && (
                <AuditReport audit={audit.data} loading={audit.isPending} error={audit.isError} />
              )}

              {showBrowse && (
                <SnapshotBrowser destinationId={selected.destinationId} snapshotId={selected.snapshotId} />
              )}

              <Field label={t(strings.restores.location)} hint={t(strings.restores.locationHint)} error={locationError}>
                {(p) => (
                  <Input
                    {...p}
                    data-testid={selectors.restores.restoreLocation}
                    value={location}
                    onChange={(e) => setLocation(e.target.value)}
                  />
                )}
              </Field>
              <Button
                variant="outline"
                size="sm"
                data-testid={selectors.restores.restoreButton}
                onClick={beginRestore}
                className="self-start"
              >
                {t(strings.restores.restore)}
              </Button>
            </>
          )}
        </div>
      )}
    </Dialog>
  );
}
