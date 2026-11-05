import { useState } from "react";
import { Activity, AlertTriangle, ArrowRight, CheckCircle2 } from "lucide-react";
import { DetailSection } from "../detail/DetailSection";
import type { ScenarioHealthSnapshot } from "../../types";
import { scenariosService, type ScenarioMaturityCampaignPreview, type ScenarioRemediationPreview } from "../../services";

export function ScenarioHealthSection({ scenarioName, health }: { scenarioName: string; health?: ScenarioHealthSnapshot }) {
  const [preview, setPreview] = useState<ScenarioRemediationPreview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [applying, setApplying] = useState(false);
  const [campaignPreview, setCampaignPreview] = useState<ScenarioMaturityCampaignPreview | null>(null);
  if (!health) return null;
  const actionable = health.evidenceState === "fresh";
  const phases = health.phases ?? [];
  return <DetailSection title="Test Genie health" icon={Activity} data-testid="scenario-health-section">
    <div className="space-y-3 text-sm">
      <div className={`rounded-xl border p-3 ${actionable ? "border-emerald-500/25 bg-emerald-500/5" : "border-amber-500/25 bg-amber-500/5"}`}>
        <div className="flex items-start gap-2">
          {actionable ? <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-emerald-400" /> : <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-300" />}
          <div className="min-w-0">
            <p className="font-medium text-slate-200" data-testid="scenario-health-state">
              Evidence is {health.evidenceState}{health.freshness ? ` · ${health.freshness}` : ""}
            </p>
            {health.reason && <p className="mt-1 text-xs leading-5 text-slate-400" data-testid="scenario-health-reason">{health.reason}</p>}
            {!actionable && <p className="mt-2 text-xs text-amber-200">Remediation becomes available when Test Genie has fresh canonical evidence.</p>}
          </div>
        </div>
        {(health.sourceRunId || health.verdict) && <div className="mt-3 flex flex-wrap gap-x-3 gap-y-1 border-t border-slate-800/70 pt-2 text-[11px] text-slate-500">
          {health.sourceRunId && <span>Run {health.sourceRunId}</span>}
          {health.verdict && <span>Verdict: {health.verdict}</span>}
        </div>}
      </div>
      {phases.map((phase) => <article key={phase.phase} className="rounded-xl border border-slate-800 bg-slate-900/40 p-3">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="font-medium text-slate-100">{phase.label || phase.phase}</div>
            <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-slate-400">
              <span>{phase.currentRung || "No current rung"}</span>
              {phase.nextRung && <><ArrowRight className="h-3.5 w-3.5 text-slate-600" /><span>{phase.nextRung}</span></>}
            </div>
          </div>
          {phase.blockingCodes?.length ? <span className="rounded-full bg-amber-500/10 px-2 py-0.5 text-[11px] text-amber-200">{phase.blockingCodes.length} blocked</span> : <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] text-emerald-300">Clear</span>}
        </div>
        {phase.priorityCapabilityLabel && <p className="mt-3 border-l-2 border-cyan-500/50 pl-2 text-xs text-slate-300">Next capability: {phase.priorityCapabilityLabel}</p>}
        {phase.blockingCodes?.length ? <p className="mt-2 text-xs text-amber-200">Blocking: {phase.blockingCodes.join(", ")}</p> : null}
        {actionable && phase.priorityCapabilityId && <button type="button" data-testid="scenario-remediation-preview-button" className="mt-3 rounded-md border border-cyan-500/40 bg-cyan-500/10 px-3 py-1.5 text-xs font-medium text-cyan-200 transition-colors hover:bg-cyan-500/20" onClick={() => { const capabilityId = phase.priorityCapabilityId; if (!capabilityId) return; setError(null); scenariosService.previewRemediation(scenarioName, { scenarioName, providerPhase: phase.phase, capabilityId }).then(setPreview).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not preview remediation")); }}>Preview remediation</button>}
      </article>)}
      {error && <p role="alert" className="text-red-300">{error}</p>}
      {preview && <div className="rounded border border-cyan-700 bg-slate-950 p-3" data-testid="scenario-remediation-preview">
        <h3 className="font-medium">{preview.proposal.title}</h3><p className="mt-1 text-slate-300">{preview.proposal.description}</p>
        {preview.existing ? <p className="mt-2 text-amber-300">Existing remediation: {preview.existing.state}{preview.existing.workRef ? ` (${preview.existing.workRef})` : ""}</p> : <><p className="mt-2">No work has been created. Review and explicitly accept to create it.</p><button type="button" data-testid="scenario-remediation-accept-button" disabled={applying} className="mt-2 rounded bg-emerald-700 px-3 py-1 text-sm" onClick={() => { setApplying(true); scenariosService.applyRemediation(scenarioName, preview.proposal.target, preview.proposal.fingerprint).then((result) => setPreview({ ...preview, existing: { state: result.created ? "accepted" : "existing", workRef: result.workRef } })).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not apply remediation")).finally(() => setApplying(false)); }}>Accept remediation</button></>}
      </div>}
      {actionable && phases.length ? <div className="border-t border-slate-700 pt-3"><p className="text-slate-300">Broad maturity campaigns are separate from phase remediation and require confirmation.</p><button type="button" data-testid="scenario-maturity-campaign-preview-button" className="mt-2 rounded border border-violet-600 px-3 py-1 text-xs text-violet-200" onClick={() => { setError(null); scenariosService.previewMaturityCampaign(scenarioName, { scenarioName, maturityTarget: "operator-selected maturity outcome", providerPhases: phases.map((phase) => phase.phase) }).then(setCampaignPreview).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not preview maturity campaign")); }}>Preview maturity campaign</button></div> : null}
      {campaignPreview && <div className="rounded border border-violet-700 bg-slate-950 p-3" data-testid="scenario-maturity-campaign-preview"><h3 className="font-medium">{campaignPreview.proposal.title}</h3><p className="mt-1 text-slate-300">{campaignPreview.proposal.description}</p><p className="mt-2 text-xs text-slate-400">Workflow: {campaignPreview.proposal.declaredWorkflow}. Tracker: {campaignPreview.proposal.trackerAvailability}</p><button type="button" data-testid="scenario-maturity-campaign-confirm-button" disabled={applying} className="mt-2 rounded bg-violet-700 px-3 py-1 text-sm" onClick={() => { setApplying(true); scenariosService.applyMaturityCampaign(scenarioName, campaignPreview.proposal.target, campaignPreview.proposal.fingerprint).then((result) => setCampaignPreview({ ...campaignPreview, existingGoalRef: result.goalRef, proposal: { ...campaignPreview.proposal, trackerAvailability: result.trackerAvailability, trackerRef: result.trackerRef } })).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not create maturity campaign")).finally(() => setApplying(false)); }}>Confirm maturity campaign</button>{campaignPreview.existingGoalRef ? <p className="mt-2 text-emerald-300">Governed goal: {campaignPreview.existingGoalRef}</p> : null}{campaignPreview.proposal.trackerRef ? <p className="mt-1 text-emerald-300">Campaign tracker: {campaignPreview.proposal.trackerRef}</p> : null}</div>}
    </div>
  </DetailSection>;
}
