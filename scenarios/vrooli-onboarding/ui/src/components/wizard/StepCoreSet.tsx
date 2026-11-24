import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchCoreSet, fetchScenarios } from "../../api/selection";
import { readCoreAuthority } from "../../api/coreAuthority";
import { i18n } from "../../i18n";
import { CardShell } from "@vrooli/react-component-library/CardShell/1";

interface Props {
  target?: string;
  seed: Set<string>;
  trustedBase: Set<string>;
  selectedScenarios?: Set<string>;
  onPendingChangesChange?: (pending: boolean) => void;
  onChange: (seed: string[], important: { kind: "scenario" | "resource"; name: string }[], members: { kind: "scenario" | "resource"; name: string }[], expectedRevision?: string) => void;
}

export function StepCoreSet({ seed, trustedBase, selectedScenarios = new Set(), onPendingChangesChange, onChange, target = "local" }: Props) {
  const scenarios = useQuery({ queryKey: ["selection-scenarios", target], queryFn: () => fetchScenarios(target) });
  const authority = useQuery({ queryKey: ["core-authority", target], queryFn: () => readCoreAuthority(target) });
  const requiredNames = useMemo(() => new Set(
    (scenarios.data?.scenarios ?? [])
      .map((scenario) => scenario.name)
      .filter((name) => ["agent-manager", "prompt-manager", "web-console", "vrooli-autoheal"].includes(name)),
  ), [scenarios.data]);
  const scenarioChoices = useMemo(() => [...(scenarios.data?.scenarios ?? [])].sort((a, b) => {
    const aProtected = trustedBase.has(a.name) || requiredNames.has(a.name);
    const bProtected = trustedBase.has(b.name) || requiredNames.has(b.name);
    if (aProtected !== bProtected) return aProtected ? -1 : 1;
    return a.name.localeCompare(b.name);
  }), [requiredNames, scenarios.data, trustedBase]);
  const committed = Array.from(new Set([...seed, ...requiredNames])).sort();
  const committedKey = committed.join("\u0000");
  const [draftSeed, setDraftSeed] = useState(committed);
  useEffect(() => setDraftSeed(committedKey ? committedKey.split("\u0000") : []), [committedKey]);
  const [draftImportant, setDraftImportant] = useState<{ kind: "scenario" | "resource"; name: string }[]>([]);
  useEffect(() => setDraftImportant(authority.data?.authority.important ?? []), [authority.data]);
  const draft = useMemo(() => new Set(draftSeed), [draftSeed]);
  const preview = useQuery({
    queryKey: ["selection-core-set", target, draftSeed],
    queryFn: () => fetchCoreSet(draftSeed, target),
  });
  const toggle = (name: string) => {
    const next = new Set(draft);
    if (next.has(name)) next.delete(name); else next.add(name);
    setDraftSeed(Array.from(next).sort());
  };
  const dirty = committedKey !== draftSeed.join("\u0000");
  const importantSet = new Set(draftImportant.map((member) => `${member.kind}:${member.name}`));
  const toggleImportant = (name: string) => {
    const key = `scenario:${name}`;
    if (importantSet.has(key)) importantSet.delete(key); else importantSet.add(key);
    setDraftImportant(Array.from(importantSet).map((value) => {
      const [kind, ...parts] = value.split(":");
      return { kind: kind as "scenario" | "resource", name: parts.join(":") };
    }));
  };
  const importantDirty = JSON.stringify([...draftImportant].sort((a,b) => `${a.kind}:${a.name}`.localeCompare(`${b.kind}:${b.name}`))) !== JSON.stringify([...(authority.data?.authority.important ?? [])].sort((a,b) => `${a.kind}:${a.name}`.localeCompare(`${b.kind}:${b.name}`)));
  const hasPendingChanges = dirty || importantDirty;
  useEffect(() => {
    onPendingChangesChange?.(hasPendingChanges);
    return () => onPendingChangesChange?.(false);
  }, [hasPendingChanges, onPendingChangesChange]);
  const members = preview.data?.members ?? [];
  return <div data-testid="step-core-set">
    <h1 className="text-2xl font-semibold">{i18n.t("onboarding.core.heading")}</h1>
    <p className="mt-2 text-sm text-muted">{i18n.t("onboarding.core.intro")}</p>
    {hasPendingChanges && <p role="status" className="core-set-confirm-hint mt-3">{i18n.t("onboarding.core.confirmBeforeContinue")}</p>}
    {scenarios.isLoading && <p role="status" className="mt-5 text-muted">{i18n.t("onboarding.core.loading")}</p>}
    {scenarios.error && <p role="alert" className="mt-5 text-danger">{i18n.t("onboarding.core.error")}</p>}
    <div className="core-set-layout mt-5">
      <section className="core-set-preview rounded-xl border border-muted bg-surface-muted p-4" aria-live="polite" data-testid="core-set-preview" aria-labelledby="core-set-preview-heading">
        <h2 id="core-set-preview-heading" className="font-semibold">{i18n.t("onboarding.core.closure")}</h2>
        {preview.isLoading && <p role="status" className="mt-2 text-sm text-muted">{i18n.t("onboarding.core.computing")}</p>}
        {preview.error && <p role="alert" className="mt-2 text-sm text-danger">{i18n.t("onboarding.core.previewError")}</p>}
        {preview.data && !preview.data.available && <p role="status" className="mt-2 text-sm text-warning">{preview.data.error || i18n.t("onboarding.core.unavailable")} Seed: {preview.data.seed.join(", ")}</p>}
        {preview.data?.available && <>
          <p className="mt-2 text-sm text-muted">{preview.data.memberCounts?.scenario ?? 0} scenarios · {preview.data.memberCounts?.resource ?? 0} resources</p>
          <details className="mt-3">
            <summary className="core-set-preview__details">{i18n.t("onboarding.core.reviewClosure", { count: members.length })}</summary>
            <ul className="mt-2 space-y-1 text-sm">
              {members.map((member) => <li key={`${member.kind}:${member.name}`}><span className="font-medium">{member.name}</span> <span className="text-muted">{member.kind} · {member.supervisionIntent}</span></li>)}
            </ul>
          </details>
        </>}
      </section>
      <section className="core-set-seeds" aria-labelledby="core-set-seeds-heading">
        <h2 id="core-set-seeds-heading" className="font-semibold">{i18n.t("onboarding.core.tree")}</h2>
        <p className="mt-1 text-sm text-muted">{i18n.t("onboarding.core.seedInstructions")}</p>
        <ul className="mt-3 grid gap-2 sm:grid-cols-2">
        {scenarioChoices.map((scenario) => {
          const trusted = trustedBase.has(scenario.name) || requiredNames.has(scenario.name);
          const eligible = selectedScenarios.has(scenario.name) || seed.has(scenario.name);
          return <li key={scenario.name}>
          <CardShell
            className={`scenario-choice-card scenario-choice-card--compact core-set-option-card${trusted ? " core-set-option-card--protected" : ""}`}
            testId="core-set-toggle"
            interactive={!trusted}
            selectLabel={i18n.t("onboarding.core.supervise", { name: scenario.name })}
            selection={trusted ? undefined : {
              selectionMode: true,
              selected: draft.has(scenario.name),
              disabled: !eligible,
              onToggleSelect: () => toggle(scenario.name),
            }}
          >
            <div className="scenario-choice-card__body">
              <span className="scenario-choice-card__name">{scenario.name}</span>
              {trusted && <span className="core-set-option-card__protected-label" role="note">{i18n.t("onboarding.core.trusted")}</span>}
              {!trusted && !eligible && <span className="core-set-option-card__dependency-label" role="note">{i18n.t("onboarding.core.selectBeforeTier")}</span>}
            </div>
          </CardShell>
          {!trusted && !draft.has(scenario.name) && (eligible || importantSet.has(`scenario:${scenario.name}`)) && <label className="core-set-important">
            <input type="checkbox" aria-label={i18n.t("onboarding.core.importantScenario", { name: scenario.name })} checked={importantSet.has(`scenario:${scenario.name}`)} disabled={seed.has(scenario.name)} onChange={() => toggleImportant(scenario.name)} />
            {i18n.t("onboarding.core.importantScenarioLabel")}
          </label>}
          </li>;
      })}
        </ul>
      </section>
    </div>
    <button type="button" disabled={(!dirty && !importantDirty) || preview.isLoading || preview.isError || preview.data?.available === false || authority.isLoading || !authority.data?.revision} onClick={() => onChange(draftSeed, draftImportant, authority.data?.authority.members ?? [], authority.data!.revision)} data-testid="core-set-confirm" className="mt-4 min-h-11 rounded-lg bg-primary px-4 py-2 font-medium text-on-primary disabled:cursor-not-allowed disabled:opacity-50">{i18n.t("onboarding.core.confirm")}</button>
  </div>;
}
