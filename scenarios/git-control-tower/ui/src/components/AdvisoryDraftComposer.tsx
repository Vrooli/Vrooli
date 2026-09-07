import { useMemo, useState } from "react";
import { FileEdit, ShieldCheck } from "lucide-react";

type DraftKind = "commit-draft" | "pr-draft" | "release-draft";

interface AdvisoryDraftComposerProps {
  scenarioSlug: string;
  repoId?: string | null;
}

const labels: Record<DraftKind, string> = {
  "commit-draft": "Commit message",
  "pr-draft": "Pull request",
  "release-draft": "Release notes",
};

export function AdvisoryDraftComposer({ scenarioSlug, repoId }: AdvisoryDraftComposerProps) {
  const [kind, setKind] = useState<DraftKind>("pr-draft");
  const [text, setText] = useState("");
  const title = useMemo(() => labels[kind], [kind]);
  return (
    <section aria-label="Advisory draft composer" className="rounded-lg border border-slate-800 bg-slate-900/50 p-4">
      <div className="flex items-center justify-between gap-3 mb-2">
        <h3 className="flex items-center gap-2 text-xs font-medium text-slate-300"><FileEdit className="h-3.5 w-3.5 text-sky-400" aria-hidden="true" />{title}</h3>
        <select aria-label="Draft kind" value={kind} onChange={(event) => setKind(event.target.value as DraftKind)} className="rounded border border-slate-700 bg-slate-950 px-2 py-1 text-[11px] text-slate-300">
          {Object.entries(labels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </div>
      <p className="mb-2 text-[11px] text-slate-500">Local editable preview for repository {repoId ?? "without an exact identity"}, scope {scenarioSlug}.</p>
      <textarea aria-label="Draft text" value={text} onChange={(event) => setText(event.target.value)} placeholder="Write or paste an evidence-backed draft…" rows={4} className="w-full resize-y rounded border border-slate-700 bg-slate-950 p-2 text-xs text-slate-200 outline-none focus:border-sky-500" />
      <div className="mt-2 flex items-center gap-2 text-[11px] text-emerald-300"><ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />Draft only: no staging, commit, checkout, fetch, or publication.</div>
    </section>
  );
}
