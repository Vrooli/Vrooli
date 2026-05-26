import { useState } from "react";

import { Dialog } from "../../components/ui/dialog";
import { Field } from "../../components/ui/field";
import { Input } from "../../components/ui/input";
import { Select } from "../../components/ui/select";
import { Button } from "../../components/ui/button";
import { useCreateDestination, useUpdateDestination } from "../../hooks/useDestinations";
import { BackendKind, CapPolicy, type Destination } from "../../api/destinations";
import { backendSlug, capPolicySlug } from "../../lib/status";
import { BACKEND_STRINGS, CAP_POLICY_STRINGS } from "../../consts/statusStrings";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

const BYTES_PER_GB = 1024 ** 3;
const gbToBytes = (gb: string): bigint => BigInt(Math.max(0, Math.round((parseFloat(gb) || 0) * BYTES_PER_GB)));
const bytesToGb = (bytes: bigint): string => {
  const gb = Number(bytes) / BYTES_PER_GB;
  return gb === 0 ? "0" : String(Number(gb.toFixed(2)));
};

const BACKEND_OPTIONS = [BackendKind.FILESYSTEM, BackendKind.S3];
const POLICY_OPTIONS = [CapPolicy.ALERT_BLOCK, CapPolicy.ALERT_ONLY];

/**
 * Create / edit a destination. On create the operator picks the backend and
 * location; on edit those are locked (the repository already exists) and only
 * the cap and policy are mutable — matching the API's cap-only Update. There is
 * no passphrase field: kopia + vault own secrets.
 */
export function DestinationFormDialog({
  open,
  onClose,
  destination,
}: {
  open: boolean;
  onClose: () => void;
  destination?: Destination;
}) {
  const { t } = useTranslation();
  const isEdit = Boolean(destination);
  const create = useCreateDestination();
  const update = useUpdateDestination();

  const [name, setName] = useState(destination?.name ?? "");
  const [backend, setBackend] = useState<BackendKind>(destination?.backendKind ?? BackendKind.FILESYSTEM);
  const [location, setLocation] = useState(destination?.location ?? "");
  const [capGb, setCapGb] = useState(destination ? bytesToGb(destination.capBytes) : "0");
  const [policy, setPolicy] = useState<CapPolicy>(destination?.capPolicy ?? CapPolicy.ALERT_BLOCK);
  const [touched, setTouched] = useState(false);

  const mutation = isEdit ? update : create;
  const nameError = !isEdit && touched && !name.trim() ? t(strings.destinations.nameRequired) : undefined;
  const locationError = !isEdit && touched && !location.trim() ? t(strings.destinations.locationRequired) : undefined;

  const close = () => {
    create.reset();
    update.reset();
    setTouched(false);
    onClose();
  };

  const submit = () => {
    setTouched(true);
    if (isEdit && destination) {
      update.mutate(
        { id: destination.id, capBytes: gbToBytes(capGb), capPolicy: policy },
        { onSuccess: close },
      );
      return;
    }
    if (!name.trim() || !location.trim()) return;
    create.mutate(
      {
        name: name.trim(),
        backendKind: backend,
        location: location.trim(),
        capBytes: gbToBytes(capGb),
        capPolicy: policy,
      },
      { onSuccess: close },
    );
  };

  return (
    <Dialog
      open={open}
      onClose={close}
      title={isEdit ? t(strings.destinations.editTitle) : t(strings.destinations.createTitle)}
      data-testid={selectors.destinations.form}
      footer={
        <>
          <Button variant="outline" size="sm" onClick={close} disabled={mutation.isPending}>
            {t(strings.common.cancel)}
          </Button>
          <Button
            size="sm"
            onClick={submit}
            disabled={mutation.isPending}
            data-testid={selectors.destinations.formSubmit}
          >
            {mutation.isPending
              ? t(strings.common.saving)
              : isEdit
                ? t(strings.common.save)
                : t(strings.common.create)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        {isEdit && <p className="text-xs text-app-muted-foreground">{t(strings.destinations.editLockedHint)}</p>}
        <Field label={t(strings.destinations.name)} hint={isEdit ? undefined : t(strings.destinations.nameHint)} error={nameError}>
          {(p) => (
            <Input
              {...p}
              data-testid={selectors.destinations.formName}
              value={name}
              disabled={isEdit}
              onChange={(e) => setName(e.target.value)}
            />
          )}
        </Field>
        <Field label={t(strings.destinations.backend)}>
          {(p) => (
            <Select
              {...p}
              data-testid={selectors.destinations.formBackend}
              value={backend}
              disabled={isEdit}
              onChange={(e) => setBackend(Number(e.target.value))}
            >
              {BACKEND_OPTIONS.map((b) => (
                <option key={b} value={b}>
                  {t(BACKEND_STRINGS[backendSlug(b)])}
                </option>
              ))}
            </Select>
          )}
        </Field>
        <Field label={t(strings.destinations.location)} hint={isEdit ? undefined : t(strings.destinations.locationHint)} error={locationError}>
          {(p) => (
            <Input
              {...p}
              data-testid={selectors.destinations.formLocation}
              value={location}
              disabled={isEdit}
              onChange={(e) => setLocation(e.target.value)}
            />
          )}
        </Field>
        <Field label={t(strings.destinations.cap)} hint={t(strings.destinations.capHint)}>
          {(p) => (
            <Input
              {...p}
              type="number"
              min={0}
              step="0.1"
              data-testid={selectors.destinations.formCap}
              value={capGb}
              onChange={(e) => setCapGb(e.target.value)}
            />
          )}
        </Field>
        <Field label={t(strings.destinations.policy)}>
          {(p) => (
            <Select
              {...p}
              data-testid={selectors.destinations.formPolicy}
              value={policy}
              onChange={(e) => setPolicy(Number(e.target.value))}
            >
              {POLICY_OPTIONS.map((pol) => (
                <option key={pol} value={pol}>
                  {t(CAP_POLICY_STRINGS[capPolicySlug(pol)])}
                </option>
              ))}
            </Select>
          )}
        </Field>
        {(create.isError || update.isError) && (
          <p className="text-sm text-app-danger">
            {isEdit ? t(strings.destinations.updateError) : t(strings.destinations.createError)}
          </p>
        )}
      </div>
    </Dialog>
  );
}
