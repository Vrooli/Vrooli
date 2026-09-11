import { useEffect, useRef, useState } from "react";
import { clone, create } from "@bufbuild/protobuf";
import { ConnectError } from "@connectrpc/connect";
import { DevelopmentGuidanceSchema, DevelopmentOutcomeSchema, PreviewDevelopmentRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import type { DevelopmentGuidance, PreviewDevelopmentRequest, PreviewDevelopmentResponse } from "@vrooli/proto-types/swarm-manager/v1/api/transition_pb";
import type { DevelopmentClient } from "../../services/development-service";
import { Slider } from "@vrooli/react-component-library/Slider/1.2.4";
import { Drawer } from "../ui/drawer";
import { Button } from "../ui/button";

const preferencesKey = "swarm.development-guidance.v1";
const efforts = ["focused", "balanced", "thorough"] as const;
const startingStates = ["unknown", "prototype", "established", "fragile"] as const;
const validations = ["targeted", "balanced", "certification"] as const;
type Preferences = { effort: string; startingState: string; validation: string; repairRelatedCode: boolean };
const defaults: Preferences = { effort: "balanced", startingState: "unknown", validation: "targeted", repairRelatedCode: false };

function readPreferences(): Preferences {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(preferencesKey) ?? "null");
    if (!value || typeof value !== "object") return defaults;
    const p = value as Record<string, unknown>;
    return {
      effort: efforts.find((v) => v === p.effort) ?? defaults.effort,
      startingState: startingStates.find((v) => v === p.startingState) ?? defaults.startingState,
      validation: validations.find((v) => v === p.validation) ?? defaults.validation,
      repairRelatedCode: p.repairRelatedCode === true,
    };
  } catch { return defaults; }
}

type Props = {
  initial: PreviewDevelopmentRequest;
  client: DevelopmentClient;
  onClose: () => void;
  onReviewed: (proposal: PreviewDevelopmentRequest, review: PreviewDevelopmentResponse) => void;
};

/** Configures a review, not an approval. All prompt text comes from the owner API. */
export function DevelopmentGoalDrawer({ initial, client, onClose, onReviewed }: Props) {
  const [proposal, setProposal] = useState(() => {
    const value = clone(PreviewDevelopmentRequestSchema, initial);
    value.budgetPolicy ||= "metered-cancellation";
    value.guidance ??= create(DevelopmentGuidanceSchema, readPreferences());
    return value;
  });
  const [review, setReview] = useState<PreviewDevelopmentResponse>();
  const [remember, setRemember] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const version = useRef(0);
  useEffect(() => () => { version.current++; }, []);
  const guidance = proposal.guidance ?? create(DevelopmentGuidanceSchema, defaults);

  function change(update: (draft: PreviewDevelopmentRequest) => void) {
    version.current++;
    const value = clone(PreviewDevelopmentRequestSchema, proposal);
    update(value);
    setProposal(value);
    setReview(undefined);
    setError("");
    setBusy(false);
  }
  function changeGuidance(update: (value: DevelopmentGuidance) => void) {
    change((draft) => { update(draft.guidance ??= create(DevelopmentGuidanceSchema, defaults)); });
  }
  async function preview() {
    const current = ++version.current;
    setBusy(true);
    setReview(undefined);
    setError("");
    try {
      const submitted = clone(PreviewDevelopmentRequestSchema, proposal);
      // Preserve newlines during editing; normalize only at the review boundary.
      for (const field of ["artifactPaths", "acceptanceAllow", "acceptanceDeny", "allowedEffects"] as const) {
        submitted[field] = submitted[field].map((value) => value.trim()).filter(Boolean);
      }
      setProposal(submitted);
      const result = await client.previewDevelopment(submitted);
      if (version.current === current) setReview(result);
    } catch (failure) {
      if (version.current === current) setError(ConnectError.from(failure).message);
    } finally { if (version.current === current) setBusy(false); }
  }
  function useReview() {
    if (!review) return;
    if (remember) {
      // No target, budget, path, custom instruction, or approval is persisted here.
      try { localStorage.setItem(preferencesKey, JSON.stringify({ effort: guidance.effort, startingState: guidance.startingState, validation: guidance.validation, repairRelatedCode: guidance.repairRelatedCode })); }
      catch { setError("Browser preferences could not be saved. Your reviewed configuration can still be used."); }
    }
    onReviewed(proposal, review);
  }
  const lines = (text: string) => text.split("\n");
  const inputClass = "mt-1 w-full rounded border border-slate-700 bg-slate-950 p-2";
  return <Drawer isOpen onClose={onClose} title="Configure adaptive plan strategy" description="Review the plan target, discretion, and limits before approval. Nothing here launches an agent."
    className="md:w-[680px]" footer={<div className="flex flex-wrap gap-2">
      <Button disabled={busy} onClick={preview}>{busy ? "Preparing preview…" : "Preview goal"}</Button>
      <Button variant="outline" disabled={!review || busy} onClick={useReview}>Use reviewed configuration</Button>
    </div>}>
    <div className="space-y-5 p-4 text-sm text-slate-200">
      <p>Remembered preferences are suggestions, not grants. Scope, effects, budgets and the exact target still require explicit review.</p>
      <label className="block">Scenario<input className={inputClass} value={proposal.scenario} onChange={(e) => change((p) => { p.scenario = e.target.value; })} /></label>
      <label className="block">Desired outcome<textarea className={inputClass} rows={4} value={proposal.objective} onChange={(e) => change((p) => { p.objective = e.target.value; })} /></label>
      <label className="block">Additional target artifacts, one path per line<textarea className={inputClass} rows={3} value={proposal.artifactPaths.join("\n")} onChange={(e) => change((p) => { p.artifactPaths = lines(e.target.value); })} /></label>
      <p className="text-xs text-slate-400">The owner also discovers required scenario contracts, skills and programs. The preview lists the exact retained artifact set.</p>
      <fieldset className="space-y-3"><legend className="font-semibold">Protected outcome evidence</legend>
        <p>Define observable completion, including the cohort and acceptance bands. A passing process or an agent's completion statement does not replace this evidence.</p>
        {proposal.outcomes.map((outcome, index) => <div key={index} className="space-y-2 rounded border border-slate-700 p-3">
          <label className="block">Outcome {index + 1} ID<input className={inputClass} value={outcome.id} onChange={(e) => change((p) => { const row = p.outcomes[index]; if (row) row.id = e.target.value; })} /></label>
          <label className="block">Outcome {index + 1} criterion<textarea className={inputClass} rows={3} value={outcome.criterion} onChange={(e) => change((p) => { const row = p.outcomes[index]; if (row) row.criterion = e.target.value; })} /></label>
          <label className="block">Outcome {index + 1} evidence owner<input className={inputClass} value={outcome.evidenceSource} onChange={(e) => change((p) => { const row = p.outcomes[index]; if (row) row.evidenceSource = e.target.value; })} /></label>
          <Button variant="outline" size="sm" onClick={() => change((p) => { p.outcomes.splice(index, 1); })}>Remove outcome {index + 1}</Button>
        </div>)}
        <Button variant="outline" size="sm" disabled={proposal.outcomes.length >= 32} onClick={() => change((p) => { p.outcomes.push(create(DevelopmentOutcomeSchema)); })}>Add protected outcome</Button>
        <p className="text-xs text-amber-200">Naming an evidence owner does not configure or qualify its receipt adapter. The runtime must establish that separately.</p>
      </fieldset>
      <fieldset className="space-y-3"><legend className="font-semibold">Engineering guidance</legend>
        <Slider
          label="Investigation effort"
          description="Focused → balanced → thorough. Guides investigation; does not change the model or budget."
          aria-label="Investigation effort"
          min={0}
          max={2}
          step={1}
          value={Math.max(0, efforts.findIndex((v) => v === guidance.effort))}
          formatValue={(value) => efforts[value] ?? "balanced"}
          onChange={(value) => changeGuidance((g) => { g.effort = efforts[value] ?? "balanced"; })}
        />
        <label className="block">Starting code state<select className={inputClass} value={guidance.startingState} onChange={(e) => changeGuidance((g) => { g.startingState = e.target.value; })}>
          {startingStates.map((value) => <option key={value} value={value}>{value}</option>)}
        </select></label>
        <p className="text-xs text-slate-400">Your assessment, not a measured maturity grade. The agent must verify it.</p>
        <label className="block">Validation strategy<select className={inputClass} value={guidance.validation} onChange={(e) => changeGuidance((g) => { g.validation = e.target.value; })}>
          {validations.map((value) => <option key={value} value={value}>{value}</option>)}
        </select></label>
        <p className="text-xs text-slate-400">Targeted first minimizes full suites and baselines. Required outcome evidence always remains required.</p>
      </fieldset>
      <fieldset className="space-y-3"><legend className="font-semibold">Proposed authority</legend>
        <label className="flex gap-2"><input type="checkbox" checked={guidance.repairRelatedCode} onChange={(e) => changeGuidance((g) => { g.repairRelatedCode = e.target.checked; })} />Permit necessary repairs in related code within the listed paths</label>
        <p className="text-xs text-slate-400">This checkbox never adds paths. List the shared packages and dependency scenarios you intend to permit. Prohibited paths still win.</p>
        <label className="block">Permitted paths, one per line<textarea className={inputClass} rows={3} value={proposal.acceptanceAllow.join("\n")} onChange={(e) => change((p) => { p.acceptanceAllow = lines(e.target.value); })} /></label>
        <label className="block">Prohibited paths, one per line<textarea className={inputClass} rows={2} value={proposal.acceptanceDeny.join("\n")} onChange={(e) => change((p) => { p.acceptanceDeny = lines(e.target.value); })} /></label>
        <label className="block">Proposed effects, one per line<textarea className={inputClass} rows={3} value={proposal.allowedEffects.join("\n")} onChange={(e) => change((p) => { p.allowedEffects = lines(e.target.value); })} /></label>
        <p className="text-xs text-amber-200">Prose effects are not enforcement. Runtime readiness must verify the owning capabilities before launch.</p>
        <label className="block">Token budget policy<select className={inputClass} value={proposal.budgetPolicy} onChange={(e) => change((p) => { p.budgetPolicy = e.target.value; })}>
          <option value="metered-cancellation">Metered cancellation (recommended)</option>
          <option value="hard-ceiling">Hard ceiling — requires qualified runner support</option>
        </select></label>
        <p className="text-xs text-amber-200">Metered cancellation can overshoot while provider usage reports and cancellation are in flight. All usage remains charged. Hard mode refuses unsupported execution paths. This policy is reviewed authority and is not remembered.</p>
        <label className="block">Aggregate token limit<input type="number" min="1" step="1" className={inputClass} value={proposal.maxTokens.toString()} onChange={(e) => { if (/^\d*$/.test(e.target.value)) change((p) => { p.maxTokens = BigInt(e.target.value || "0"); }); }} /></label>
        <label className="block">Aggregate wall-time limit (seconds)<input type="number" min="1" step="1" className={inputClass} value={proposal.maxWallSeconds.toString()} onChange={(e) => { if (/^\d*$/.test(e.target.value)) change((p) => { p.maxWallSeconds = BigInt(e.target.value || "0"); }); }} /></label>
      </fieldset>
      <label className="block">Additional instructions<textarea className={inputClass} rows={5} maxLength={8192} value={guidance.additionalInstructions} onChange={(e) => changeGuidance((g) => { g.additionalInstructions = e.target.value; })} /></label>
      <p className="text-xs text-slate-400">Retained with the reviewed goal. Cannot override protected outcomes, prohibitions, limits, or evidence obligations.</p>
      <label className="flex gap-2"><input type="checkbox" checked={remember} onChange={(e) => setRemember(e.target.checked)} />Remember guidance preferences on this browser</label>
      {error && <p role="alert" className="text-red-300">{error}</p>}
      {review && <section aria-label="Goal preview" className="space-y-3">
        <p className="break-all">Review fingerprint: {review.proposalDigest}</p>
        <ul>{review.findings.map((f, i) => <li key={i}>{f.code}: {f.detail}</li>)}</ul>
        <details><summary>Resolved target artifacts ({review.artifacts.length})</summary><ul>{review.artifacts.map((artifact) => <li className="break-all" key={artifact.path}>{artifact.path} · sha256:{artifact.sha256}</li>)}</ul></details>
        <pre className="whitespace-pre-wrap rounded bg-slate-950 p-3 text-xs">{review.goalMessage}</pre>
        {!!review.launchBlockers.length && <div role="status"><p>Launch unavailable</p><ul>{review.launchBlockers.map((b) => <li key={b}>{b}</li>)}</ul></div>}
      </section>}
    </div>
  </Drawer>;
}
