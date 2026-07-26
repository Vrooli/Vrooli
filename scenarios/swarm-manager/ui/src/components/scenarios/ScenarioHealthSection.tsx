import { useState } from "react";
import { Activity } from "lucide-react";
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
  return <DetailSection title="Test Genie health" icon={Activity} data-testid="scenario-health-section">
    <div className="space-y-3 text-sm">
      <p data-testid="scenario-health-state"><span className="font-medium">Evidence: </span>{health.evidenceState}{health.freshness ? ` (${health.freshness})` : ""}</p>
      {health.reason && <p className="text-slate-400" data-testid="scenario-health-reason">{health.reason}</p>}
      {health.sourceRunId && <p className="text-xs text-slate-500">Source run: {health.sourceRunId}</p>}
      {health.verdict && <p>Provider verdict: {health.verdict}</p>}
      {!actionable && <p className="text-amber-300">Remediation is unavailable until Test Genie supplies fresh canonical evidence.</p>}
      {health.phases?.map((phase) => <article key={phase.phase} className="rounded border border-slate-700 p-3">
        <div className="font-medium">{phase.label || phase.phase}</div>
        <div className="text-slate-400">{phase.currentRung || "No current rung"}{phase.nextRung ? ` → ${phase.nextRung}` : ""}</div>
        {phase.priorityCapabilityLabel && <div>Priority: {phase.priorityCapabilityLabel}</div>}
        {phase.blockingCodes?.length ? <div className="text-amber-300">Blocking: {phase.blockingCodes.join(", ")}</div> : null}
        {actionable && phase.priorityCapabilityId && <button type="button" data-testid="scenario-remediation-preview-button" className="mt-2 rounded bg-cyan-700 px-3 py-1 text-xs" onClick={() => { setError(null); scenariosService.previewRemediation(scenarioName, { scenarioName, providerPhase: phase.phase, capabilityId: phase.priorityCapabilityId! }).then(setPreview).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not preview remediation")); }}>Preview remediation</button>}
      </article>)}
      {error && <p role="alert" className="text-red-300">{error}</p>}
      {preview && <div className="rounded border border-cyan-700 bg-slate-950 p-3" data-testid="scenario-remediation-preview">
        <h3 className="font-medium">{preview.proposal.title}</h3><p className="mt-1 text-slate-300">{preview.proposal.description}</p>
        {preview.existing ? <p className="mt-2 text-amber-300">Existing remediation: {preview.existing.state}{preview.existing.workRef ? ` (${preview.existing.workRef})` : ""}</p> : <><p className="mt-2">No work has been created. Review and explicitly accept to create it.</p><button type="button" data-testid="scenario-remediation-accept-button" disabled={applying} className="mt-2 rounded bg-emerald-700 px-3 py-1 text-sm" onClick={() => { setApplying(true); scenariosService.applyRemediation(scenarioName, preview.proposal.target, preview.proposal.fingerprint).then((result) => setPreview({ ...preview, existing: { state: result.created ? "accepted" : "existing", workRef: result.workRef } })).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not apply remediation")).finally(() => setApplying(false)); }}>Accept remediation</button></>}
      </div>}
      {actionable && health.phases?.length ? <div className="border-t border-slate-700 pt-3"><p className="text-slate-300">Broad maturity campaigns are separate from phase remediation and require confirmation.</p><button type="button" data-testid="scenario-maturity-campaign-preview-button" className="mt-2 rounded border border-violet-600 px-3 py-1 text-xs text-violet-200" onClick={() => { setError(null); scenariosService.previewMaturityCampaign(scenarioName, { scenarioName, maturityTarget: "operator-selected maturity outcome", providerPhases: health.phases!.map((phase) => phase.phase) }).then(setCampaignPreview).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not preview maturity campaign")); }}>Preview maturity campaign</button></div> : null}
      {campaignPreview && <div className="rounded border border-violet-700 bg-slate-950 p-3" data-testid="scenario-maturity-campaign-preview"><h3 className="font-medium">{campaignPreview.proposal.title}</h3><p className="mt-1 text-slate-300">{campaignPreview.proposal.description}</p><p className="mt-2 text-xs text-slate-400">Workflow: {campaignPreview.proposal.declaredWorkflow}. Tracker: {campaignPreview.proposal.trackerAvailability}</p><button type="button" data-testid="scenario-maturity-campaign-confirm-button" disabled={applying} className="mt-2 rounded bg-violet-700 px-3 py-1 text-sm" onClick={() => { setApplying(true); scenariosService.applyMaturityCampaign(scenarioName, campaignPreview.proposal.target, campaignPreview.proposal.fingerprint).then((result) => setCampaignPreview({ ...campaignPreview, existingGoalRef: result.goalRef, proposal: { ...campaignPreview.proposal, trackerAvailability: result.trackerAvailability, trackerRef: result.trackerRef } })).catch((value: unknown) => setError(value instanceof Error ? value.message : "Could not create maturity campaign")).finally(() => setApplying(false)); }}>Confirm maturity campaign</button>{campaignPreview.existingGoalRef ? <p className="mt-2 text-emerald-300">Governed goal: {campaignPreview.existingGoalRef}</p> : null}{campaignPreview.proposal.trackerRef ? <p className="mt-1 text-emerald-300">Campaign tracker: {campaignPreview.proposal.trackerRef}</p> : null}</div>}
    </div>
  </DetailSection>;
}
