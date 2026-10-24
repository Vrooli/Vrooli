import { useMemo } from "react";
import { Popover } from "./ui/popover";
import { Plus, CheckSquare, Square, AlertTriangle, FileText, Paperclip } from "lucide-react";
import { useEvidence, useRuns } from "../lib/hooks-evidence";
import { changeSummaryContextItem, runPhaseContextItem } from "../lib/agentContext";
import type { AgentContextItem, RepoFileStats } from "../lib/api";

interface ContextPickerProps {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  tidinessAvailable: boolean;
  auditorAvailable: boolean;
  visualCaptureAvailable: boolean;
  fileStats?: RepoFileStats;
  contextItems: AgentContextItem[];
  onAddContext: (item: AgentContextItem) => void;
  onRemoveContext: (id: string) => void;
}

export function ContextPickerPopover({ scenarioSlug, repoId, testGenieAvailable, fileStats, contextItems, onAddContext, onRemoveContext }: ContextPickerProps) {
  const runsQuery = useRuns(scenarioSlug, { limit: 1 }, testGenieAvailable, repoId);
  const evidenceQuery = useEvidence(scenarioSlug, { limit: 20, runLimit: 10 }, testGenieAvailable, repoId);
  const changeSummary = useMemo(() => fileStats ? changeSummaryContextItem(fileStats) : null, [fileStats]);
  const phaseItems = useMemo(() => {
    const run = runsQuery.data?.runs[0];
    if (!run) return [];
    const descriptors = new Map(run.descriptorSnapshot?.phases.map((descriptor) => [descriptor.phase, descriptor]) ?? []);
    return run.phases
      .filter((phase) => phase.status === "failed" || (phase.findingsSummary?.total ?? 0) > 0)
      .map((phase) => runPhaseContextItem(run, phase, descriptors.get(phase.name), scenarioSlug));
  }, [runsQuery.data?.runs, scenarioSlug]);
  const artifactItems = useMemo(() => (evidenceQuery.data?.items ?? []).flatMap((item): AgentContextItem[] => {
    if (!item.run || !item.artifact) return [];
    return [{
      kind: "screenshot",
      id: `artifact:${item.run.runId}:${item.artifact.id}`,
      label: item.artifact.label || item.artifact.kind,
      markdown: [`## Test evidence`, "", `- Run: \`${item.run.runId}\``, `- Artifact: \`${item.artifact.id}\``, `- Kind: \`${item.artifact.kind}\``, `- Producing phase: \`${item.artifact.producingPhase || "unknown"}\``].join("\n"),
    }];
  }), [evidenceQuery.data?.items]);
  const isAttached = (id: string) => contextItems.some((item) => item.id === id);
  const toggle = (item: AgentContextItem) => isAttached(item.id) ? onRemoveContext(item.id) : onAddContext(item);
  const hasAnyItems = Boolean(changeSummary || phaseItems.length || artifactItems.length);

  return <Popover direction="up" align="start" trigger={<span className="h-8 w-8 flex items-center justify-center rounded border border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600 transition-colors"><Plus className="h-4 w-4" /></span>}>
    <div className="max-h-80 overflow-y-auto p-2 w-72">
      {!hasAnyItems && <p className="text-xs text-slate-500 p-2 text-center">No context available yet</p>}
      {changeSummary && <Section icon={<FileText className="h-3 w-3" />} title="Changes"><CheckItem item={changeSummary} checked={isAttached(changeSummary.id)} onToggle={toggle} /></Section>}
      {testGenieAvailable && <Section icon={<AlertTriangle className="h-3 w-3" />} title="Phase findings">{phaseItems.length ? phaseItems.map((item) => <CheckItem key={item.id} item={item} checked={isAttached(item.id)} onToggle={toggle} />) : <p className="text-[11px] text-slate-600 px-2 py-1">No findings</p>}</Section>}
      {testGenieAvailable && <Section icon={<Paperclip className="h-3 w-3" />} title="Evidence">{artifactItems.length ? artifactItems.map((item) => <CheckItem key={item.id} item={item} checked={isAttached(item.id)} onToggle={toggle} />) : <p className="text-[11px] text-slate-600 px-2 py-1">No evidence</p>}</Section>}
    </div>
  </Popover>;
}

function Section({ icon, title, children }: { icon: React.ReactNode; title: string; children: React.ReactNode }) {
  return <div className="mb-2 last:mb-0"><div className="flex items-center gap-1.5 px-2 py-1 text-[11px] font-medium text-slate-400 uppercase tracking-wider">{icon}{title}</div>{children}</div>;
}

function CheckItem({ item, checked, onToggle }: { item: AgentContextItem; checked: boolean; onToggle: (item: AgentContextItem) => void }) {
  return <button type="button" onClick={() => onToggle(item)} className="w-full flex items-center gap-2 px-2 py-1.5 rounded text-left hover:bg-slate-800/60 transition-colors">{checked ? <CheckSquare className="h-3.5 w-3.5 text-blue-400 shrink-0" /> : <Square className="h-3.5 w-3.5 text-slate-600 shrink-0" />}<span className="text-xs text-slate-300 truncate">{item.label}</span></button>;
}
