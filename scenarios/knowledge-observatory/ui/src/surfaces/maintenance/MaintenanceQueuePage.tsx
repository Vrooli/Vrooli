import { useCallback, useEffect, useMemo, useState } from "react";
import { AlertTriangle, CheckCircle2, ClipboardList, RefreshCw, ShieldCheck } from "lucide-react";
import { PageShell } from "../../shared/components/PageShell";
import { Button } from "../../shared/ui/button";
import {
  fetchMaintenanceInventory,
  proposeMaintenanceDispositions,
  routeMaintenanceDisposition,
  type MaintenanceCandidate,
  type MaintenanceInventoryResponse,
  type MaintenanceProposal,
  type MaintenanceRouteResponse,
} from "../../shared/services/api";
import type { Route } from "../../shared/controllers/routeController";

type FilterValue = "all" | "unresolved" | "proposed" | "resolved";

function statusTone(value: string) {
  if (value === "exact_content" || value === "resolved") return "ko-tone-good";
  if (value === "unknown" || value === "unresolved") return "ko-tone-medium";
  return "ko-tone-poor";
}

function shortDigest(value: string) { return value ? `${value.slice(0, 18)}…` : "digest unavailable"; }

export function MaintenanceQueuePage({ onNavigate }: { onNavigate: (route: Route) => void }) {
  const [inventory, setInventory] = useState<MaintenanceInventoryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>();
  const [selectedPath, setSelectedPath] = useState<string>();
  const [artifactFilter, setArtifactFilter] = useState("all");
  const [standingFilter, setStandingFilter] = useState("all");
  const [portabilityFilter, setPortabilityFilter] = useState("all");
  const [queueFilter, setQueueFilter] = useState<FilterValue>("all");
  const [proposals, setProposals] = useState<Record<string, MaintenanceProposal>>({});
  const [proposalLoading, setProposalLoading] = useState(false);
  const [routeResult, setRouteResult] = useState<MaintenanceRouteResponse>();
  const [reason, setReason] = useState("");
  const [target, setTarget] = useState("");
  const [preservation, setPreservation] = useState("preserve source bytes and incoming references");
  const [evidence, setEvidence] = useState("");
  const [dryRun, setDryRun] = useState(true);
  const [ownerAuthorized, setOwnerAuthorized] = useState(false);

  const scan = useCallback(async () => {
    setLoading(true); setError(undefined); setRouteResult(undefined);
    try { setInventory(await fetchMaintenanceInventory({ roots: ["docs", "scenarios"], max_files: 100, max_bytes: 2 << 20 })); }
    catch (cause) { setError(cause instanceof Error ? cause.message : "Inventory scan failed"); }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { void scan(); }, [scan]);

  const candidates = inventory?.candidates ?? [];
  const selected = candidates.find((candidate) => candidate.path === selectedPath) ?? candidates[0];
  const selectedProposal = selected ? proposals[selected.path] : undefined;
  const artifactClasses = useMemo(() => ["all", ...Array.from(new Set(candidates.map((candidate) => candidate.artifact_class)))], [candidates]);
  const filteredCandidates = candidates.filter((candidate) => {
    const proposal = proposals[candidate.path];
    const queueState: FilterValue = proposal ? "proposed" : candidate.reference_status === "resolved" ? "resolved" : "unresolved";
    return (artifactFilter === "all" || candidate.artifact_class === artifactFilter)
      && (standingFilter === "all" || candidate.evidence_standing === standingFilter)
      && (portabilityFilter === "all" || candidate.portability_signals.includes(portabilityFilter))
      && (queueFilter === "all" || queueState === queueFilter);
  });

  async function requestProposals() {
    setProposalLoading(true); setError(undefined);
    try {
      const response = await proposeMaintenanceDispositions(filteredCandidates, "Identify evidence gaps and propose a reversible owner review disposition.");
      setProposals((previous) => Object.fromEntries([...Object.entries(previous), ...response.proposals.map((proposal) => [proposal.source, proposal])]));
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Proposal request failed"); }
    finally { setProposalLoading(false); }
  }

  async function routeProposal() {
    if (!selected || !selectedProposal || !reason.trim() || !evidence.trim() || (!dryRun && !ownerAuthorized)) return;
    setError(undefined); setRouteResult(undefined);
    try {
      setRouteResult(await routeMaintenanceDisposition({ proposal: { ...selectedProposal, uncertainty: [...selectedProposal.uncertainty, `review reason: ${reason.trim()}`, `target: ${target.trim() || "owner decision"}`, `preservation: ${preservation.trim()}`], citations: [...selectedProposal.citations, `evidence:${evidence.trim()}`] }, expected_source_sha256: selected.sha256, idempotency_key: `ko-maint-${selected.sha256.slice(-16)}`, dry_run: dryRun, authorization: ownerAuthorized ? "owner" : undefined }));
    } catch (cause) { setError(cause instanceof Error ? cause.message : "Route request failed"); }
  }

  return <PageShell>
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="ko-kicker">Bounded inventory · owner review</p><h2 className="ko-page-title">Knowledge Maintenance Queue</h2><p className="ko-muted mt-1 max-w-3xl">Find candidate artifacts, inspect their evidence standing, and prepare reversible disposition proposals. The queue never deletes or moves source data.</p></div>
      <div className="flex flex-wrap gap-2"><Button variant="secondary" size="sm" onClick={() => onNavigate("explorer")}>Back to Explorer</Button><Button size="sm" onClick={() => void scan()} disabled={loading}><RefreshCw className={loading ? "mr-2 h-4 w-4 animate-spin" : "mr-2 h-4 w-4"} />Scan again</Button></div>
    </div>

    {error && <div className="ko-card mt-5 flex items-start gap-3 border-red-700/60 p-4 text-sm text-red-200"><AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />{error}</div>}
    {inventory && <>
      <div className="mt-5 grid gap-3 sm:grid-cols-2 xl:grid-cols-4"><div className="ko-card p-4"><p className="ko-muted text-xs">Candidates</p><p className="mt-1 text-2xl font-bold">{candidates.length}</p></div><div className="ko-card p-4"><p className="ko-muted text-xs">Revision</p><p className="mt-1 truncate font-mono text-sm">{inventory.revision || "working-tree"}</p></div><div className="ko-card p-4"><p className="ko-muted text-xs">Proposed</p><p className="mt-1 text-2xl font-bold">{Object.keys(proposals).length}</p></div><div className="ko-card p-4"><p className="ko-muted text-xs">Scan status</p><p className="mt-1 text-sm font-semibold">{inventory.truncated ? "Partial / bounded" : inventory.status}</p></div></div>
      {(inventory.truncated || inventory.warnings.length > 0) && <div className="ko-card mt-4 border-amber-700/50 p-4 text-sm text-amber-200"><AlertTriangle className="mr-2 inline h-4 w-4" />This inventory is not complete. {inventory.truncated && "The candidate bound was reached. "}{inventory.warnings.join(" ")}</div>}

      <div className="ko-card mt-5 p-4"><div className="flex flex-wrap items-end gap-3"><label className="ko-field"><span>Artifact class</span><select value={artifactFilter} onChange={(event) => setArtifactFilter(event.target.value)}>{artifactClasses.map((value) => <option key={value} value={value}>{value === "all" ? "All classes" : value}</option>)}</select></label><label className="ko-field"><span>Evidence standing</span><select value={standingFilter} onChange={(event) => setStandingFilter(event.target.value)}><option value="all">All standings</option><option value="unknown">Unknown</option><option value="exact_content">Exact content</option><option value="run_file">Run overlap</option></select></label><label className="ko-field"><span>Portability</span><select value={portabilityFilter} onChange={(event) => setPortabilityFilter(event.target.value)}><option value="all">All signals</option><option value="absolute_path">Absolute path</option><option value="external_reference">External reference</option><option value="machine_specific_endpoint">Machine endpoint</option></select></label><label className="ko-field"><span>Queue</span><select value={queueFilter} onChange={(event) => setQueueFilter(event.target.value as FilterValue)}><option value="all">All states</option><option value="unresolved">Unresolved</option><option value="proposed">Proposed</option><option value="resolved">Resolved</option></select></label><Button size="sm" variant="outline" onClick={() => void requestProposals()} disabled={proposalLoading || filteredCandidates.length === 0}><ClipboardList className="mr-2 h-4 w-4" />{proposalLoading ? "Proposing…" : `Propose ${filteredCandidates.length}`}</Button></div></div>

      <div className="mt-5 grid gap-5 xl:grid-cols-[minmax(0,1.2fr)_minmax(20rem,0.8fr)]"><div className="ko-card overflow-hidden"><div className="overflow-x-auto"><table className="ko-table"><thead><tr><th>Candidate</th><th>Standing</th><th>Portability</th><th>Owner hint</th></tr></thead><tbody>{filteredCandidates.map((candidate) => <tr key={candidate.path} className={selected?.path === candidate.path ? "ko-table-row-active" : ""} onClick={() => setSelectedPath(candidate.path)}><td><button type="button" className="text-left"><span className="block max-w-[28rem] truncate font-mono text-sm">{candidate.path}</span><span className="ko-muted text-xs">{candidate.artifact_class} · {candidate.bytes.toLocaleString()} bytes</span></button></td><td><span className={`ko-status-pill ${statusTone(candidate.evidence_standing)}`}>{candidate.evidence_standing}</span></td><td><span className="ko-muted text-xs">{candidate.portability_signals.length ? candidate.portability_signals.join(", ") : "portable signal clear"}</span></td><td className="ko-muted text-xs">{candidate.owner_hint || "unresolved"}</td></tr>)}</tbody></table></div>{filteredCandidates.length === 0 && <div className="p-8 text-center text-sm ko-muted">No candidates match the current filters.</div>}</div>

        <div className="ko-card p-5"><div className="flex items-start justify-between gap-3"><div><p className="ko-kicker">Candidate detail</p><h3 className="mt-1 break-all text-lg font-semibold">{selected?.path || "Select a candidate"}</h3></div>{selected && <span className={`ko-status-pill ${statusTone(selected.evidence_standing)}`}>{selected.evidence_standing}</span>}</div>{selected ? <div className="mt-4 space-y-4 text-sm"><dl className="grid gap-2 text-xs sm:grid-cols-2"><div><dt className="ko-muted">Source hash</dt><dd className="mt-1 break-all font-mono">{selected.sha256 || "unavailable"}</dd></div><div><dt className="ko-muted">Inspect handle</dt><dd className="mt-1 break-all font-mono">{selected.inspect_handle}</dd></div><div><dt className="ko-muted">Reference status</dt><dd className="mt-1">{selected.reference_status}</dd></div><div><dt className="ko-muted">Proposal handle</dt><dd className="mt-1 break-all font-mono">{selected.proposal_handle}</dd></div></dl>{selected.excerpt && <pre className="max-h-36 overflow-auto rounded bg-slate-950/70 p-3 text-xs text-slate-300">{selected.excerpt}</pre>}{selectedProposal ? <div className="rounded border border-cyan-800/60 bg-cyan-950/20 p-3 text-xs"><p className="font-semibold text-cyan-200">Proposal: {selectedProposal.action} · confidence {Math.round(selectedProposal.confidence * 100)}%</p><p className="mt-2 text-slate-300">{selectedProposal.uncertainty.join(" · ")}</p>{selectedProposal.prompt_injection_signal && <p className="mt-2 text-red-300">Instruction-like retrieved text was treated as untrusted evidence.</p>}</div> : <p className="text-xs ko-muted">No proposal yet. Select candidates and use “Propose” to request a bounded review suggestion.</p>}
          <div className="border-t border-slate-800 pt-4"><p className="ko-kicker">Owner review handoff</p><div className="mt-3 grid gap-3"><label className="ko-field"><span>Reason <em>(required)</em></span><textarea value={reason} onChange={(event) => setReason(event.target.value)} placeholder="Why is this candidate being reviewed?" /></label><label className="ko-field"><span>Target disposition</span><input value={target} onChange={(event) => setTarget(event.target.value)} placeholder="retain, archive, or owner decision" /></label><label className="ko-field"><span>Preservation plan</span><input value={preservation} onChange={(event) => setPreservation(event.target.value)} /></label><label className="ko-field"><span>Evidence / citation <em>(required)</em></span><input value={evidence} onChange={(event) => setEvidence(event.target.value)} placeholder="receipt, ticket, or review reference" /></label><label className="flex items-center gap-2 text-xs ko-muted"><input type="checkbox" checked={dryRun} onChange={(event) => setDryRun(event.target.checked)} /> Dry run only (recommended)</label><label className="flex items-center gap-2 text-xs ko-muted"><input type="checkbox" checked={ownerAuthorized} onChange={(event) => setOwnerAuthorized(event.target.checked)} /> I have owner authorization for a non-dry-run route</label><Button size="sm" variant="secondary" onClick={() => void routeProposal()} disabled={!selectedProposal || !reason.trim() || !evidence.trim() || (!dryRun && !ownerAuthorized)}><ShieldCheck className="mr-2 h-4 w-4" />Route owner handoff</Button></div></div>
          {routeResult && <div className={`rounded border p-3 text-xs ${routeResult.status === "refused" ? "border-red-800/60 text-red-300" : "border-emerald-800/60 text-emerald-300"}`}><CheckCircle2 className="mr-2 inline h-4 w-4" />{routeResult.status}: {routeResult.reason || routeResult.receipt}</div>}
        </div> : <p className="mt-4 ko-muted">Choose a candidate from the bounded inventory.</p>}</div>
      </div>
    </>}
    {!inventory && !loading && <div className="ko-card mt-5 p-8 text-center ko-muted">No inventory response is available.</div>}
  </PageShell>;
}
