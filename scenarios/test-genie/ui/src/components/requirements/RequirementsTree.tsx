import { useState } from "react";
import { ChevronDown, ChevronRight, Check, Circle, X } from "lucide-react";
import { cn } from "../../lib/utils";
import { selectors } from "../../consts/selectors";
import type { ModuleSnapshot, RequirementItem, ValidationItem } from "../../lib/api";

interface RequirementsTreeProps { modules: ModuleSnapshot[]; filter?: "all" | "passed" | "failed" | "not_run"; searchQuery?: string }

export function RequirementsTree({ modules, filter = "all", searchQuery = "" }: RequirementsTreeProps) {
  const [expandedModules, setExpandedModules] = useState(new Set<string>());
  const [expandedRequirements, setExpandedRequirements] = useState(new Set<string>());
  const visible = modules.filter((module) => !searchQuery || module.name.toLowerCase().includes(searchQuery.toLowerCase()) || module.requirements?.some((requirement) => `${requirement.id} ${requirement.title}`.toLowerCase().includes(searchQuery.toLowerCase()))).sort((a, b) => a.name.localeCompare(b.name));
  if (visible.length === 0) return <div className="rounded-xl border border-white/10 bg-white/[0.02] p-6 text-center text-slate-400">{modules.length === 0 ? "No requirements found. Run tests to establish requirement evidence." : "No requirements match your search."}</div>;
  return <div className="space-y-2" data-testid={selectors.requirements.tree}>{visible.map((module) => {
    const expanded = expandedModules.has(module.name);
    const requirements = (module.requirements ?? []).filter((requirement) => matches(requirement, filter, searchQuery));
    return <div key={module.name} className="overflow-hidden rounded-xl border border-white/10 bg-white/[0.02]"><button type="button" onClick={() => setExpandedModules((current) => toggle(current, module.name))} className="flex w-full items-center gap-3 px-4 py-3 text-left hover:bg-white/[0.02]">{expanded ? <ChevronDown className="h-4 w-4 text-slate-400"/> : <ChevronRight className="h-4 w-4 text-slate-400"/>}<span className="flex-1 font-medium">{module.name}</span><span className="text-sm text-slate-400">{Math.round(module.completionRate)}%</span><span className="rounded-full bg-white/5 px-2 py-0.5 text-xs text-slate-300">{module.complete}/{module.total}</span></button>{expanded && <div className="border-t border-white/5 px-4 py-2">{requirements.length ? requirements.map((requirement) => <Requirement key={requirement.id} requirement={requirement} expanded={expandedRequirements.has(requirement.id)} onToggle={() => setExpandedRequirements((current) => toggle(current, requirement.id))}/>) : <p className="py-2 text-sm text-slate-500">No requirements match this filter.</p>}</div>}</div>;
  })}</div>;
}

function matches(requirement: RequirementItem, filter: string, query: string): boolean {
  if (filter !== "all" && requirement.liveStatus !== filter) return false;
  return !query || `${requirement.id} ${requirement.title}`.toLowerCase().includes(query.toLowerCase());
}
function toggle(current: Set<string>, value: string): Set<string> { const next = new Set(current); if (next.has(value)) next.delete(value); else next.add(value); return next; }
function Requirement({ requirement, expanded, onToggle }: { requirement: RequirementItem; expanded: boolean; onToggle(): void }) {
  const validations = requirement.validations ?? [];
  return <div className="py-1"><button type="button" onClick={onToggle} className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-sm hover:bg-white/[0.03]">{validations.length ? expanded ? <ChevronDown className="h-3 w-3 text-slate-500"/> : <ChevronRight className="h-3 w-3 text-slate-500"/> : <span className="w-3"/>}<Status status={requirement.liveStatus}/><span className="flex-1 text-slate-200"><span className="text-slate-500">{requirement.id}</span><span className="mx-2 text-slate-600">|</span>{requirement.title}</span><span className="text-xs text-slate-500">{validations.length} validation{validations.length === 1 ? "" : "s"}</span></button>{expanded && validations.length > 0 && <div className="ml-6 mt-1 space-y-1 border-l border-white/5 pl-4">{validations.map((validation, index) => <Validation key={`${validation.ref}-${index}`} validation={validation}/>)}</div>}</div>;
}
function Validation({ validation }: { validation: ValidationItem }) { return <div className="flex gap-2 text-xs text-slate-400"><Status status={validation.liveStatus}/><span>{validation.type}: {validation.ref}{validation.phase ? ` · ${validation.phase}` : ""}</span></div>; }
function Status({ status }: { status: string }) { const Icon = status === "passed" ? Check : status === "failed" ? X : Circle; return <Icon className={cn("mt-0.5 h-3.5 w-3.5", status === "passed" ? "text-emerald-400" : status === "failed" ? "text-rose-400" : "text-slate-500")}/>; }
