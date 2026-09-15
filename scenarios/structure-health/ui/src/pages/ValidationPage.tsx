import { useMutation } from "@tanstack/react-query";
import { useState } from "react";

import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import {
  TARGET_KINDS,
  type TargetKindValue,
  ValidationTargetKind,
  validationClient,
} from "../api/validation";
import { useTranslation } from "../i18n";
import { errorMessage } from "../lib/errorMessage";

const statusLabels: Record<number, string> = {
  1: "passed",
  2: "failed",
  3: "degraded",
  4: "error",
  5: "skipped",
};

export function ValidationPage() {
  const { t } = useTranslation();
  const [kind, setKind] = useState<TargetKindValue>(ValidationTargetKind.SCENARIO);
  const [id, setId] = useState("");
  const [root, setRoot] = useState("");
  const [path, setPath] = useState("");
  const mutation = useMutation({
    mutationFn: () => validationClient.validateTarget({ kind, id, root, path }),
  });

  return (
    <section
      data-testid={selectors.pages.validation}
      aria-labelledby="validation-heading"
      className="flex flex-col gap-4"
    >
      <div>
        <h2 id="validation-heading" className="text-2xl font-semibold">
          {t(strings.pages.validation.title)}
        </h2>
        <p className="mt-2 text-app-muted-foreground">{t(strings.pages.validation.description)}</p>
      </div>

      <form
        data-testid={selectors.validation.view}
        className="grid gap-3 rounded-panel border border-app-border bg-app-surface p-4 md:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault();
          if (id.trim()) void mutation.mutateAsync();
        }}
      >
        <label className="flex flex-col gap-1 text-sm">
          <span>{t(strings.validation.kind)}</span>
          <select
            data-testid={selectors.validation.kind}
            value={kind}
            onChange={(event) => setKind(Number(event.target.value) as TargetKindValue)}
            className="h-10 rounded-md border border-white/20 bg-white/5 px-3 text-white"
          >
            {TARGET_KINDS.map((targetKind) => (
              <option key={targetKind.value} value={targetKind.value} className="bg-slate-900">
                {targetKind.label}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span>{t(strings.validation.id)}</span>
          <Input
            data-testid={selectors.validation.id}
            value={id}
            onChange={(event) => setId(event.target.value)}
            placeholder={t(strings.validation.idPlaceholder)}
            required
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span>{t(strings.validation.root)}</span>
          <Input
            data-testid={selectors.validation.root}
            value={root}
            onChange={(event) => setRoot(event.target.value)}
            placeholder={t(strings.validation.rootPlaceholder)}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span>{t(strings.validation.path)}</span>
          <Input
            data-testid={selectors.validation.path}
            value={path}
            onChange={(event) => setPath(event.target.value)}
            placeholder={t(strings.validation.pathPlaceholder)}
          />
        </label>
        <div className="md:col-span-2">
          <Button data-testid={selectors.validation.submit} type="submit" disabled={mutation.isPending}>
            {mutation.isPending ? t(strings.validation.loading) : t(strings.validation.submit)}
          </Button>
        </div>
      </form>

      {mutation.error && (
        <p data-testid={selectors.validation.error} className="text-app-danger">
          {errorMessage(mutation.error, t)}
        </p>
      )}

      {mutation.data && <ValidationResult response={mutation.data} />}
    </section>
  );
}

function ValidationResult({ response }: { response: Awaited<ReturnType<typeof validationClient.validateTarget>> }) {
  const { t } = useTranslation();
  const assessment = response.assessment;
  const findings = assessment?.findings ?? [];
  return (
    <section
      data-testid={selectors.validation.result}
      aria-label={t(strings.validation.resultTitle)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h3 className="text-lg font-semibold">{t(strings.validation.resultTitle)}</h3>
        <span className="rounded-control border border-app-border px-2 py-1 text-xs uppercase">
          {statusLabels[response.status] ?? t(strings.validation.unknownStatus)}
        </span>
      </div>
      <p className="mt-2 text-sm text-app-muted-foreground">
        {response.target?.id ?? "—"} · {response.target?.root || t(strings.validation.noRoot)}
      </p>
      {assessment?.local && (
        <p className="mt-3 text-sm">
          {t(strings.validation.maturity, {
            current: assessment.local.currentLevel,
            next: assessment.local.nextLevel,
          })}
        </p>
      )}
      <div data-testid={selectors.validation.findings} className="mt-4">
        <h4 className="text-sm font-medium text-app-muted-foreground">
          {t(strings.validation.findingsTitle)}
        </h4>
        {findings.length === 0 ? (
          <p data-testid={selectors.validation.emptyFindings} className="mt-2 text-sm text-app-success">
            {t(strings.validation.noFindings)}
          </p>
        ) : (
          <ul className="mt-2 flex flex-col gap-2 text-sm">
            {findings.map((finding) => (
              <li key={`${finding.code}:${finding.location}`} className="rounded-control border border-app-border p-2">
                <span className="font-medium">{finding.code}</span>
                <span className="ms-2 text-xs uppercase text-app-muted-foreground">{finding.severity}</span>
                <p className="mt-1 text-app-muted-foreground">{finding.message}</p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}
