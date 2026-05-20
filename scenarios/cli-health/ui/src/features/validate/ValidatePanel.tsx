import { useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { validationClient } from "../../api/clients";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  Severity,
  type ValidateScenarioResponse,
} from "@vrooli/proto-types/cli-health/v1/validation/validation_pb";

export function ValidatePanel() {
  const { t } = useTranslation();
  const [scenario, setScenario] = useState("");

  const mutation = useMutation<ValidateScenarioResponse>({
    mutationFn: async () => validationClient.validateScenario({ scenario }),
  });

  const handleSubmit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!scenario.trim()) return;
    mutation.mutate();
  };

  const severityLabel = (s: Severity): { label: string; className: string; testId: string } => {
    switch (s) {
      case Severity.ERROR:
        return {
          label: t(strings.validate.severityError),
          className: "bg-red-500/20 text-red-300",
          testId: "severity-error",
        };
      case Severity.WARNING:
        return {
          label: t(strings.validate.severityWarning),
          className: "bg-yellow-500/20 text-yellow-300",
          testId: "severity-warning",
        };
      default:
        return {
          label: t(strings.validate.severityInfo),
          className: "bg-sky-500/20 text-sky-300",
          testId: "severity-info",
        };
    }
  };

  const data = mutation.data;

  return (
    <section
      data-testid={selectors.validate.card}
      className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-lg font-semibold">{t(strings.validate.title)}</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.validate.description)}</p>
      <form onSubmit={handleSubmit} className="mt-3 flex gap-2">
        <Input
          data-testid={selectors.validate.input}
          value={scenario}
          onChange={(e) => setScenario(e.target.value)}
          placeholder={t(strings.validate.placeholder)}
        />
        <Button data-testid={selectors.validate.submit} type="submit">
          {t(strings.validate.submit)}
        </Button>
      </form>

      {mutation.isPending && (
        <p data-testid={selectors.validate.loading} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.validate.loading)}
        </p>
      )}
      {mutation.error && (
        <p data-testid={selectors.validate.error} className="mt-3 text-sm text-red-400">
          {t(strings.validate.error, { message: mutation.error.message })}
        </p>
      )}
      {data && (
        <div className="mt-3">
          <div className="flex items-center gap-2">
            {data.passed ? (
              <span
                data-testid={selectors.validate.passed}
                className="rounded-control bg-green-500/20 px-2 py-0.5 text-xs font-medium text-green-300"
              >
                {t(strings.validate.passed)}
              </span>
            ) : (
              <span
                data-testid={selectors.validate.failed}
                className="rounded-control bg-red-500/20 px-2 py-0.5 text-xs font-medium text-red-300"
              >
                {t(strings.validate.failed)}
              </span>
            )}
            <span data-testid={selectors.validate.summary} className="text-xs text-app-muted-foreground">
              {t(strings.validate.summary, {
                errors: data.summary?.errors ?? 0,
                warnings: data.summary?.warnings ?? 0,
                infos: data.summary?.infos ?? 0,
              })}
            </span>
          </div>
          {data.findings.length === 0 ? (
            <p data-testid={selectors.validate.empty} className="mt-2 text-sm">
              {t(strings.validate.noFindings)}
            </p>
          ) : (
            <ul data-testid={selectors.validate.findings} className="mt-2 space-y-2">
              {data.findings.map((f, i) => {
                const sev = severityLabel(f.severity);
                return (
                  <li
                    key={`${f.code}/${f.location}/${i}`}
                    data-testid={selectors.validate.finding}
                    className="rounded-md border border-app-border bg-app-surface-muted p-3"
                  >
                    <div className="flex items-center gap-2">
                      <span
                        className={`rounded-control px-2 py-0.5 text-xs font-mono ${sev.className}`}
                      >
                        {sev.label}
                      </span>
                      <span className="font-mono text-xs">{f.code}</span>
                    </div>
                    <p className="mt-1 text-sm">{f.message}</p>
                    {f.location && (
                      <p className="mt-1 text-xs text-app-muted-foreground font-mono">
                        {f.location}
                      </p>
                    )}
                    {f.suggestion && (
                      <p className="mt-1 text-xs text-app-muted-foreground">→ {f.suggestion}</p>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      )}
    </section>
  );
}
