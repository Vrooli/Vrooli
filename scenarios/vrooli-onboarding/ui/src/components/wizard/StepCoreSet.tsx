import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchV2CoreSet, fetchV2Scenarios } from "../../lib/api";

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
  const scenarios = useQuery({ queryKey: ["v2-scenarios"], queryFn: fetchV2Scenarios });
  const preview = useQuery({
    queryKey: ["v2-core-set", draftSeed],
    queryFn: () => fetchV2CoreSet(draftSeed),
  });
  const toggle = (name: string) => {
    const next = new Set(draft);
    if (next.has(name)) next.delete(name); else next.add(name);
    setDraftSeed(Array.from(next).sort());
  };
  const dirty = committedKey !== draftSeed.join("\u0000");
  const members = preview.data?.members ?? [];
  return <div data-testid="step-core-set">
    <h1 className="text-2xl font-semibold">Core supervision set</h1>
    <p className="mt-2 text-sm text-muted">Choose the seed scenarios that must remain supervised. The preview shows every scenario and resource pulled in by their declared dependency closure.</p>
    {scenarios.isLoading && <p role="status" className="mt-5 text-muted">Loading scenarios…</p>}
    {scenarios.error && <p role="alert" className="mt-5 text-danger">Core-set choices could not be loaded.</p>}
    <div className="mt-5 grid gap-2 sm:grid-cols-2">
      {(scenarios.data?.scenarios ?? []).map((scenario) => {
        const trusted = trustedBase.has(scenario.name);
        return <label key={scenario.name} className="flex min-h-11 items-center gap-3 rounded-lg border border-muted p-3">
          <input type="checkbox" checked={draft.has(scenario.name)} disabled={trusted} onChange={() => toggle(scenario.name)} aria-label={`Supervise ${scenario.name}`} data-testid="core-set-toggle" className="h-5 w-5 accent-emerald-500" />
          <span><span className="block font-medium">{scenario.name}</span>{trusted && <span className="block text-xs text-muted">Trusted-base member; cannot be removed</span>}</span>
        </label>;
      })}
    </div>
    <section className="mt-6 rounded-xl border border-muted bg-surface-muted p-4" aria-live="polite" data-testid="core-set-preview">
      <h2 className="font-semibold">Computed closure</h2>
      {preview.isLoading && <p role="status" className="mt-2 text-sm text-muted">Computing closure…</p>}
      {preview.error && <p role="alert" className="mt-2 text-sm text-danger">The closure preview could not be computed. Your current seed remains visible and authoritative.</p>}
      {preview.data && !preview.data.available && <p role="status" className="mt-2 text-sm text-warning">{preview.data.error ?? "Closure unavailable."} Seed: {preview.data.seed.join(", ")}</p>}
      {preview.data?.available && <>
        <p className="mt-2 text-sm text-muted">{preview.data.member_counts?.scenario ?? 0} scenarios · {preview.data.member_counts?.resource ?? 0} resources</p>
        <ul className="mt-3 max-h-56 space-y-1 overflow-auto text-sm">
          {members.map((member) => <li key={`${member.kind}:${member.name}`}><span className="font-medium">{member.name}</span> <span className="text-muted">{member.kind} · {member.supervision_intent}</span></li>)}
        </ul>
      </>}
    </section>
    <button type="button" disabled={!dirty || preview.isLoading || preview.isError || preview.data?.available === false} onClick={() => onChange(draftSeed)} data-testid="core-set-confirm" className="mt-4 min-h-11 rounded-lg bg-primary px-4 py-2 font-medium text-on-primary disabled:cursor-not-allowed disabled:opacity-50">Confirm supervision set</button>
  </div>;
}
