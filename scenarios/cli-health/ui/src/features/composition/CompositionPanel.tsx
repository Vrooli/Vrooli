import { useState } from "react";
import { useMutation } from "@tanstack/react-query";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { searchComposition } from "../../api/composition";

export function CompositionPanel() {
  const { t } = useTranslation();
  const [query, setQuery] = useState("");
  const [scenario, setScenario] = useState("cli-health");
  const mutation = useMutation({ mutationFn: () => searchComposition(query, scenario) });

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    if (query.trim() && scenario.trim()) mutation.mutate();
  };

  return (
    <section data-testid={selectors.composition.card} className="mt-6 rounded-xl border border-white/10 bg-black/20 p-4">
      <h2 className="text-lg font-semibold">{t(strings.composition.title)}</h2>
      <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.composition.description)}</p>
      <form onSubmit={submit} className="mt-3 flex flex-col gap-2">
        <label className="text-sm" htmlFor="composition-query">{t(strings.composition.queryLabel)}</label>
        <Input id="composition-query" data-testid={selectors.composition.query} value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t(strings.composition.queryPlaceholder)} />
        <label className="text-sm" htmlFor="composition-scenario">{t(strings.composition.scenarioLabel)}</label>
        <Input id="composition-scenario" data-testid={selectors.composition.scenario} value={scenario} onChange={(event) => setScenario(event.target.value)} placeholder={t(strings.composition.scenarioPlaceholder)} />
        <Button data-testid={selectors.composition.submit} type="submit" disabled={!query.trim() || !scenario.trim()}>{t(strings.composition.submit)}</Button>
      </form>
      {mutation.isPending && <p data-testid={selectors.composition.loading} className="mt-3 text-sm text-app-muted-foreground">{t(strings.composition.loading)}</p>}
      {mutation.error && <p data-testid={selectors.composition.error} className="mt-3 text-sm text-red-400">{t(strings.composition.error, { message: mutation.error.message })}</p>}
      {mutation.data && (
        <div className="mt-4 space-y-3">
          <p className="text-sm text-app-muted-foreground">{mutation.data.scope}</p>
          {mutation.data.results.length === 0 && <p data-testid={selectors.composition.empty}>{t(strings.composition.empty)}</p>}
          {mutation.data.results.map((result) => (
            <article key={result.id} data-testid={selectors.composition.result} className="rounded-md border border-app-border bg-app-surface-muted p-3">
              <h3 className="font-semibold">{result.title}</h3>
              <p className="mt-1 text-sm">{result.text}</p>
              <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.composition.attestation, { value: result.attestation })} · {t(strings.composition.score, { score: result.score.toFixed(2) })}</p>
              <ul className="mt-2 space-y-1 text-xs font-mono text-app-muted-foreground">
                {result.evidence.map((evidence) => <li key={`${evidence.path}:${evidence.line}`}>{evidence.path}:{evidence.line} · sha256:{evidence.sha256}</li>)}
              </ul>
            </article>
          ))}
          {mutation.data.degradations.map((degradation) => <p key={degradation.code} data-testid={selectors.composition.degradation} className="text-sm text-app-muted-foreground">{t(strings.composition.degraded, { code: degradation.code, message: degradation.message })}</p>)}
        </div>
      )}
    </section>
  );
}
