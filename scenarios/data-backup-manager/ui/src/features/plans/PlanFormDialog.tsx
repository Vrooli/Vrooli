import { useState } from "react";
import { Code, ConnectError } from "@connectrpc/connect";

import { Dialog } from "../../components/ui/dialog";
import { Field } from "../../components/ui/field";
import { Input } from "../../components/ui/input";
import { Button } from "../../components/ui/button";
import { useCreatePlan, useUpdatePlan } from "../../hooks/usePlans";
import { useTargets } from "../../hooks/useTargets";
import { useDestinations } from "../../hooks/useDestinations";
import type { Plan } from "../../api/plans";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

interface CheckOption {
  id: string;
  label: string;
}

function CheckList({
  options,
  selected,
  onToggle,
  emptyLabel,
  "data-testid": testId,
}: {
  options: CheckOption[];
  selected: string[];
  onToggle: (id: string) => void;
  emptyLabel: string;
  "data-testid"?: string;
}) {
  if (options.length === 0) {
    return <p className="text-xs text-app-muted-foreground">{emptyLabel}</p>;
  }
  return (
    <div
      data-testid={testId}
      className="flex max-h-40 flex-col gap-1 overflow-y-auto rounded-control border border-app-border p-2"
    >
      {options.map((o) => (
        <label key={o.id} className="flex items-center gap-2 text-sm text-app-foreground">
          <input
            type="checkbox"
            className="h-4 w-4"
            checked={selected.includes(o.id)}
            onChange={() => onToggle(o.id)}
          />
          <span className="truncate">{o.label}</span>
        </label>
      ))}
    </div>
  );
}

/**
 * Plan binding builder: pick targets × destinations, set a schedule and
 * retention, and toggle enabled, with a live human summary. Enforces the API's
 * "at least one target and one destination" rule inline before submit.
 */
export function PlanFormDialog({
  open,
  onClose,
  plan,
}: {
  open: boolean;
  onClose: () => void;
  plan?: Plan;
}) {
  const { t } = useTranslation();
  const isEdit = Boolean(plan);
  const create = useCreatePlan();
  const update = useUpdatePlan();
  const targets = useTargets();
  const destinations = useDestinations();

  const [name, setName] = useState(plan?.name ?? "");
  const [targetIds, setTargetIds] = useState<string[]>(plan?.targetIds ?? []);
  const [destinationIds, setDestinationIds] = useState<string[]>(plan?.destinationIds ?? []);
  const [schedule, setSchedule] = useState(plan?.schedule ?? "0 2 * * *");
  const [keepLatest, setKeepLatest] = useState(String(plan?.retention?.keepLatest ?? 7));
  const [enabled, setEnabled] = useState(plan?.enabled ?? true);
  const [touched, setTouched] = useState(false);

  const mutation = isEdit ? update : create;
  const nameError = touched && !name.trim() ? t(strings.plans.nameRequired) : undefined;
  const targetError = touched && targetIds.length === 0 ? t(strings.plans.needTarget) : undefined;
  const destError = touched && destinationIds.length === 0 ? t(strings.plans.needDestination) : undefined;

  const toggle = (setter: React.Dispatch<React.SetStateAction<string[]>>) => (id: string) =>
    setter((prev) => (prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]));

  const close = () => {
    create.reset();
    update.reset();
    setTouched(false);
    onClose();
  };

  // The API rejects a plan with FAILED_PRECONDITION when non-sensitive
  // recommended targets remain unregistered. We surface that as a distinct
  // warning with an explicit "proceed" path that resubmits with
  // allowIncompleteCoverage — incomplete coverage is never silently allowed.
  const errorCode = (err: unknown) =>
    err ? ConnectError.from(err).code : undefined;
  const isCoverageError =
    errorCode(create.error) === Code.FailedPrecondition ||
    errorCode(update.error) === Code.FailedPrecondition;

  const runMutation = (allowIncompleteCoverage: boolean) => {
    const input = {
      name: name.trim(),
      targetIds,
      destinationIds,
      schedule: schedule.trim(),
      keepLatest: Number(keepLatest) || 0,
      enabled,
      allowIncompleteCoverage,
    };
    if (isEdit && plan) {
      update.mutate({ id: plan.id, input }, { onSuccess: close });
    } else {
      create.mutate(input, { onSuccess: close });
    }
  };

  const submit = () => {
    setTouched(true);
    if (!name.trim() || targetIds.length === 0 || destinationIds.length === 0) return;
    runMutation(false);
  };

  return (
    <Dialog
      open={open}
      onClose={close}
      title={isEdit ? t(strings.plans.editTitle) : t(strings.plans.createTitle)}
      data-testid={selectors.plans.form}
      footer={
        <>
          <Button variant="outline" size="sm" onClick={close} disabled={mutation.isPending}>
            {t(strings.common.cancel)}
          </Button>
          <Button size="sm" onClick={submit} disabled={mutation.isPending} data-testid={selectors.plans.formSubmit}>
            {mutation.isPending ? t(strings.common.saving) : isEdit ? t(strings.common.save) : t(strings.common.create)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label={t(strings.plans.name)} error={nameError}>
          {(p) => (
            <Input {...p} data-testid={selectors.plans.formName} value={name} onChange={(e) => setName(e.target.value)} />
          )}
        </Field>

        <Field label={t(strings.plans.targets)} error={targetError}>
          {() => (
            <CheckList
              data-testid={selectors.plans.targetPicker}
              options={(targets.data ?? []).map((t2) => ({ id: t2.id, label: `${t2.owner}/${t2.name}` }))}
              selected={targetIds}
              onToggle={toggle(setTargetIds)}
              emptyLabel={t(strings.plans.noTargets)}
            />
          )}
        </Field>

        <Field label={t(strings.plans.destinations)} error={destError}>
          {() => (
            <CheckList
              data-testid={selectors.plans.destinationPicker}
              options={(destinations.data ?? []).map((d) => ({ id: d.id, label: d.name }))}
              selected={destinationIds}
              onToggle={toggle(setDestinationIds)}
              emptyLabel={t(strings.plans.noDestinations)}
            />
          )}
        </Field>

        <Field label={t(strings.plans.schedule)} hint={t(strings.plans.scheduleHint)}>
          {(p) => (
            <Input
              {...p}
              data-testid={selectors.plans.formSchedule}
              value={schedule}
              onChange={(e) => setSchedule(e.target.value)}
            />
          )}
        </Field>

        <Field label={t(strings.plans.keepLatest)} hint={t(strings.plans.keepLatestHint)}>
          {(p) => (
            <Input
              {...p}
              type="number"
              min={0}
              data-testid={selectors.plans.formKeepLatest}
              value={keepLatest}
              onChange={(e) => setKeepLatest(e.target.value)}
            />
          )}
        </Field>

        <label className="flex items-center gap-2 text-sm text-app-foreground">
          <input
            type="checkbox"
            data-testid={selectors.plans.formEnabled}
            className="h-4 w-4"
            checked={enabled}
            onChange={(e) => setEnabled(e.target.checked)}
          />
          {t(strings.plans.enabled)}
        </label>

        <p data-testid={selectors.plans.summary} className="rounded-control bg-app-surface-muted px-3 py-2 text-xs text-app-muted-foreground">
          {t(strings.plans.summary, {
            targets: targetIds.length,
            destinations: destinationIds.length,
            keep: Number(keepLatest) || 0,
          })}
        </p>

        {isCoverageError ? (
          <div
            data-testid={selectors.plans.coverageWarning}
            className="flex flex-col gap-2 rounded-panel border border-app-warning/40 bg-app-warning/10 p-3"
          >
            <p className="text-sm text-app-foreground">{t(strings.coverage.incompleteRejected)}</p>
            <div>
              <Button
                size="sm"
                variant="outline"
                data-testid={selectors.plans.proceedIncompleteCoverage}
                disabled={mutation.isPending}
                onClick={() => runMutation(true)}
              >
                {t(strings.coverage.proceedIncomplete)}
              </Button>
            </div>
          </div>
        ) : (
          (create.isError || update.isError) && (
            <p className="text-sm text-app-danger">
              {isEdit ? t(strings.plans.updateError) : t(strings.plans.createError)}
            </p>
          )
        )}
      </div>
    </Dialog>
  );
}
