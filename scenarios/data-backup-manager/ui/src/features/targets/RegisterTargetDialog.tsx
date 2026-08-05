import { useState } from "react";

import { Dialog } from "../../components/ui/dialog";
import { Field } from "../../components/ui/field";
import { Input } from "../../components/ui/input";
import { Select } from "../../components/ui/select";
import { Button } from "../../components/ui/button";
import { useRegisterTarget } from "../../hooks/useTargets";
import { SourceKind } from "../../api/targets";
import { SOURCE_KIND_OPTIONS, sourceKindSlug } from "../../lib/status";
import { SOURCE_KIND_STRINGS } from "../../consts/statusStrings";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

/**
 * Register-target form. Co-equal with scenario self-registration: an operator
 * supplies owner + name + source kind + locator, and the API upserts on
 * owner+name. Validation is inline and the form-level API error is shown near
 * the actions; input is preserved on failure.
 */
export function RegisterTargetDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const register = useRegisterTarget();

  const [owner, setOwner] = useState("");
  const [name, setName] = useState("");
  const [kind, setKind] = useState<SourceKind>(SourceKind.FILESYSTEM);
  const [locator, setLocator] = useState("");
  const [critical, setCritical] = useState(false);
  const [touched, setTouched] = useState(false);

  const ownerError = touched && !owner.trim() ? t(strings.targets.ownerRequired) : undefined;
  const nameError = touched && !name.trim() ? t(strings.targets.nameRequired) : undefined;
  const locatorError = touched && !locator.trim() ? t(strings.targets.locatorRequired) : undefined;

  const reset = () => {
    setOwner("");
    setName("");
    setKind(SourceKind.FILESYSTEM);
    setLocator("");
    setCritical(false);
    setTouched(false);
    register.reset();
  };

  const close = () => {
    reset();
    onClose();
  };

  const submit = () => {
    setTouched(true);
    if (!owner.trim() || !name.trim() || !locator.trim()) return;
    register.mutate(
      {
        owner: owner.trim(),
        name: name.trim(),
        sourceKind: kind,
        locator: locator.trim(),
        critical,
      },
      { onSuccess: close },
    );
  };

  return (
    <Dialog
      open={open}
      onClose={close}
      title={t(strings.targets.register)}
      data-testid={selectors.targets.form}
      footer={
        <>
          <Button variant="outline" size="sm" onClick={close} disabled={register.isPending}>
            {t(strings.common.cancel)}
          </Button>
          <Button
            size="sm"
            onClick={submit}
            disabled={register.isPending}
            data-testid={selectors.targets.formSubmit}
          >
            {register.isPending ? t(strings.common.saving) : t(strings.common.create)}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-4">
        <Field label={t(strings.targets.owner)} hint={t(strings.targets.ownerHint)} error={ownerError}>
          {(p) => (
            <Input
              {...p}
              data-testid={selectors.targets.formOwner}
              value={owner}
              onChange={(e) => setOwner(e.target.value)}
            />
          )}
        </Field>
        <Field label={t(strings.targets.name)} hint={t(strings.targets.nameHint)} error={nameError}>
          {(p) => (
            <Input
              {...p}
              data-testid={selectors.targets.formName}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          )}
        </Field>
        <Field label={t(strings.targets.kind)}>
          {(p) => (
            <Select
              {...p}
              data-testid={selectors.targets.formKind}
              value={kind}
              onChange={(e) => setKind(Number(e.target.value))}
            >
              {SOURCE_KIND_OPTIONS.map((opt) => (
                <option key={opt.slug} value={opt.kind}>
                  {t(SOURCE_KIND_STRINGS[sourceKindSlug(opt.kind)])}
                </option>
              ))}
            </Select>
          )}
        </Field>
        <Field label={t(strings.targets.locator)} hint={t(strings.targets.locatorHint)} error={locatorError}>
          {(p) => (
            <Input
              {...p}
              data-testid={selectors.targets.formLocator}
              value={locator}
              onChange={(e) => setLocator(e.target.value)}
            />
          )}
        </Field>
        <Field label={t(strings.targets.critical)} hint={t(strings.targets.criticalHint)}>
          {(p) => (
            <Input
              {...p}
              type="checkbox"
              data-testid={selectors.targets.formCritical}
              checked={critical}
              onChange={(e) => setCritical(e.target.checked)}
              className="h-4 w-4 self-start p-0 accent-app-primary"
            />
          )}
        </Field>
        {register.isError && <p className="text-sm text-app-danger">{t(strings.targets.registerError)}</p>}
      </div>
    </Dialog>
  );
}
