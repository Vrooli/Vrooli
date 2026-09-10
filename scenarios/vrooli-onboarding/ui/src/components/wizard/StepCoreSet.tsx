import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchCoreSet, fetchScenarios } from "../../api/selection";
import { NavigationTree } from "@vrooli/react-component-library/NavigationTree/1";
import { i18n } from "../../i18n";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";

interface Props {
  seed: Set<string>;
  trustedBase: Set<string>;
  onChange: (seed: string[]) => void;
}

export function StepCoreSet({ seed, trustedBase, onChange }: Props) {
  const committed = Array.from(seed).sort();
  const committedKey = committed.join("\u0000");
  const [draftSeed, setDraftSeed] = useState(committed);
  useEffect(() => setDraftSeed(committedKey ? committedKey.split("\u0000") : []), [committedKey]);
  const draft = useMemo(() => new Set(draftSeed), [draftSeed]);
  const scenarios = useQuery({ queryKey: ["selection-scenarios"], queryFn: () => fetchScenarios() });
  const preview = useQuery({
    queryKey: ["selection-core-set", draftSeed],
    queryFn: () => fetchCoreSet(draftSeed),
  });
  const toggle = (name: string) => {
    const next = new Set(draft);
    if (next.has(name)) next.delete(name); else next.add(name);
    setDraftSeed(Array.from(next).sort());
  };
  const dirty = committedKey !== draftSeed.join("\u0000");
  const members = preview.data?.members ?? [];
  return <div data-testid="step-core-set">
    <h1 className="text-2xl font-semibold">{i18n.t("onboarding.core.heading")}</h1>
    <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.core.intro")}</p>
    {scenarios.isLoading && <p role="status" className="mt-5 text-muted">{i18n.t("onboarding.core.loading")}</p>}
    {scenarios.error && <p role="alert" className="mt-5 text-danger">{i18n.t("onboarding.core.error")}</p>}
    <NavigationTree title={i18n.t("onboarding.core.tree")}>
      <ul className="mt-5 grid gap-2 sm:grid-cols-2" data-rcl-navigation-tree-list>
      {(scenarios.data?.scenarios ?? []).map((scenario) => {
        const trusted = trustedBase.has(scenario.name);
        return <li key={scenario.name} data-rcl-navigation-tree-item><label className="flex min-h-11 items-center gap-3 rounded-lg border border-muted p-3">
          <Checkbox checked={draft.has(scenario.name)} disabled={trusted} onCheckedChange={() => toggle(scenario.name)} aria-label={i18n.t("onboarding.core.supervise", { name: scenario.name })} data-testid="core-set-toggle" />
          <span><span className="block font-medium">{scenario.name}</span>{trusted && <span className="block text-xs text-muted">{i18n.t("onboarding.core.trusted")}</span>}</span>
        </label></li>;
      })}
      </ul>
    </NavigationTree>
    <section className="mt-6 rounded-xl border border-muted bg-surface-muted p-4" aria-live="polite" data-testid="core-set-preview">
      <h2 className="font-semibold">{i18n.t("onboarding.core.closure")}</h2>
      {preview.isLoading && <p role="status" className="mt-2 text-sm text-muted">{i18n.t("onboarding.core.computing")}</p>}
      {preview.error && <p role="alert" className="mt-2 text-sm text-danger">{i18n.t("onboarding.core.previewError")}</p>}
      {preview.data && !preview.data.available && <p role="status" className="mt-2 text-sm text-warning">{preview.data.error || i18n.t("onboarding.core.unavailable")} Seed: {preview.data.seed.join(", ")}</p>}
      {preview.data?.available && <>
        <p className="mt-2 text-sm text-muted">{preview.data.memberCounts?.scenario ?? 0} scenarios · {preview.data.memberCounts?.resource ?? 0} resources</p>
        <ul className="mt-3 max-h-56 space-y-1 overflow-auto text-sm">
          {members.map((member) => <li key={`${member.kind}:${member.name}`}><span className="font-medium">{member.name}</span> <span className="text-muted">{member.kind} · {member.supervisionIntent}</span></li>)}
        </ul>
      </>}
    </section>
    <button type="button" disabled={!dirty || preview.isLoading || preview.isError || preview.data?.available === false} onClick={() => onChange(draftSeed)} data-testid="core-set-confirm" className="mt-4 min-h-11 rounded-lg bg-primary px-4 py-2 font-medium text-on-primary disabled:cursor-not-allowed disabled:opacity-50">{i18n.t("onboarding.core.confirm")}</button>
  </div>;
}
