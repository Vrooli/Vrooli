import { useMemo, useState } from "react";
import { Layers } from "lucide-react";
import {
  ProposalState,
  type Proposal,
} from "@vrooli/proto-types/git-control-tower/v1/proposals/proposals_pb";
import { Badge } from "../../components/ui/badge";
import { useCombineProposals } from "../../lib/hooks-proposals";
import {
  hasPriorWork,
  effortLabel,
  freshnessMeta,
  epochsLabel,
  proposalEpochs,
} from "./model";

export function ProposalInbox({
  proposals,
  isLoading,
  error,
  repoId,
  onOpen,
}: {
  proposals: Proposal[];
  isLoading: boolean;
  error: Error | null;
  repoId: string | null;
  onOpen: (id: string) => void;
}) {
  const [filter, setFilter] = useState<"open" | "closed">("open");
  const [selected, setSelected] = useState<string[]>([]);
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [combining, setCombining] = useState(false);
  const [combineSources, setCombineSources] = useState<Proposal[]>([]);
  const [actionError, setActionError] = useState<string | null>(null);
  const combine = useCombineProposals(repoId);
  const visible = useMemo(
    () =>
      proposals.filter((p) =>
        filter === "open"
          ? p.state === ProposalState.OPEN
          : p.state !== ProposalState.OPEN,
      ),
    [proposals, filter],
  );
  const grouped = useMemo(() => {
    const result = new Map<string, Proposal[]>();
    visible.forEach((p) => {
      const key = p.work?.effortRef || "Other & combined work";
      const values = result.get(key) ?? [];
      values.push(p);
      result.set(key, values);
    });
    return result;
  }, [visible]);
  const submit = async () => {
    setActionError(null);
    try {
      const sources = combineSources;
      const result = await combine.mutateAsync({ sources, subject, body });
      if (result) onOpen(result.id);
    } catch (cause) {
      setActionError(cause instanceof Error ? cause.message : String(cause));
    }
  };
  return (
    <main
      className="min-h-0 flex-1 overflow-auto px-4 py-5 sm:px-6"
      data-testid="proposal-inbox"
    >
      <div className="mx-auto max-w-5xl">
        <p className="max-w-2xl text-sm leading-relaxed text-slate-400">
          Review agent-proposed work before creating a commit. Closing a
          proposal keeps all code and staged changes.
        </p>
        {proposals.length === 200 && (
          <p className="mt-2 text-xs text-amber-300">
            Showing the first 200 proposals. Use CLI state/effort filters to
            inspect older reviews.
          </p>
        )}
        <div className="my-5 flex flex-wrap items-center justify-between gap-3">
          <div
            className="flex gap-1 rounded-lg border border-slate-800 p-1"
            aria-label="Proposal state filter"
          >
            {(["open", "closed"] as const).map((state) => (
              <button
                key={state}
                className={`min-h-10 rounded-md px-4 text-sm capitalize ${filter === state ? "bg-slate-800 text-slate-100" : "text-slate-400"}`}
                onClick={() => {
                  setFilter(state);
                  setSelected([]);
                  setCombining(false);
                }}
                aria-pressed={filter === state}
              >
                {state} (
                {
                  proposals.filter((p) =>
                    state === "open"
                      ? p.state === ProposalState.OPEN
                      : p.state !== ProposalState.OPEN,
                  ).length
                }
                )
              </button>
            ))}
          </div>
          {filter === "open" && (
            <button
              className="inline-flex min-h-10 items-center gap-2 rounded-lg border border-slate-700 px-4 text-sm text-slate-200 disabled:opacity-40"
              disabled={selected.length < 2 || selected.length > 20}
              onClick={() => {
                setSubject("Review combined work");
                setCombineSources(
                  proposals.filter((p) => selected.includes(p.id)),
                );
                setCombining(true);
              }}
              data-testid="proposal-combine"
            >
              <Layers className="h-4 w-4" />
              Combine selected {selected.length > 0 && `(${selected.length})`}
            </button>
          )}
        </div>
        {error && (
          <p role="alert" className="text-sm text-red-300">
            Proposals unavailable: {error.message}
          </p>
        )}
        {isLoading && (
          <p className="text-sm text-slate-400">Loading proposals…</p>
        )}
        {!isLoading && !error && visible.length === 0 && (
          <p className="rounded-lg border border-dashed border-slate-800 px-6 py-12 text-center text-sm text-slate-500">
            No {filter} proposals.
          </p>
        )}
        {Array.from(grouped, ([effort, items]) => (
          <section key={effort} className="mb-6">
            <h2 className="mb-2 break-words text-xs font-medium text-slate-500">
              {effortLabel(effort)}
            </h2>
            <ul className="divide-y divide-slate-800 rounded-lg border border-slate-800 bg-slate-900/30">
              {items.map((p) => {
                const uncertain = p.files.filter((f) => hasPriorWork(f)).length;
                const status = freshnessMeta(p.freshness?.state);
                return (
                  <li key={p.id} className="flex items-center gap-3 px-3 py-2">
                    {filter === "open" && (
                      <input
                        type="checkbox"
                        className="h-4 w-4 shrink-0"
                        aria-label={`Select ${p.message?.subject}`}
                        checked={selected.includes(p.id)}
                        onChange={(event) =>
                          setSelected(
                            event.target.checked
                              ? [...selected, p.id]
                              : selected.filter((id) => id !== p.id),
                          )
                        }
                      />
                    )}
                    <button
                      className="min-h-14 min-w-0 flex-1 py-2 text-left"
                      onClick={() => onOpen(p.id)}
                      data-testid="proposal-card"
                    >
                      <span className="block text-sm font-medium leading-relaxed text-slate-100">
                        {p.message?.subject || p.id}
                      </span>
                      <span className="mt-1 block text-xs text-slate-500">
                        {p.files.length} files ·{" "}
                        {epochsLabel(proposalEpochs(p)) || "Work proposal"}
                        {uncertain
                          ? ` · ${uncertain} files include earlier work`
                          : ""}
                      </span>
                    </button>
                    <Badge
                      variant={
                        filter === "closed"
                          ? "default"
                          : p.captured
                            ? status.tone
                            : "warning"
                      }
                    >
                      {filter === "closed"
                        ? ProposalState[p.state]
                            .replace(/^.*?_/, "")
                            .toLowerCase()
                        : p.captured
                          ? status.label
                          : "Needs file review"}
                    </Badge>
                  </li>
                );
              })}
            </ul>
          </section>
        ))}
        {combining && (
          <section
            className="mb-5 space-y-3 rounded-xl border border-blue-900 bg-blue-950/20 p-4"
            aria-label="Combine review"
            data-testid="proposal-combine-preview"
          >
            <h2 className="text-sm font-semibold">Create one review package</h2>
            <p className="text-xs leading-relaxed text-slate-400">
              Combines the exact selected revisions. Shared files appear once.
              Different file versions or proposals without saved contents cannot
              be combined. Working files are never included automatically.
              Sources remain linked in history.
            </p>
            <ul className="space-y-1 text-xs text-slate-300">
              {combineSources.map((p) => (
                <li key={p.id}>
                  {p.message?.subject} · r{p.revision} · {p.files.length} files
                </li>
              ))}
            </ul>
            <input
              className="w-full rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm"
              aria-label="Combined commit subject"
              value={subject}
              onChange={(event) => setSubject(event.target.value)}
            />
            <textarea
              className="w-full rounded border border-slate-700 bg-slate-900 px-3 py-2 text-sm"
              rows={3}
              aria-label="Combined commit body"
              value={body}
              onChange={(event) => setBody(event.target.value)}
            />
            {actionError && (
              <p role="alert" className="text-xs text-red-300">
                {actionError}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <button
                className="min-h-11 px-3 text-sm text-slate-400"
                onClick={() => setCombining(false)}
              >
                Cancel
              </button>
              <button
                className="min-h-11 rounded-lg bg-blue-600 px-4 text-sm font-medium text-white disabled:opacity-40"
                onClick={() => void submit()}
                disabled={combine.isPending || !subject.trim()}
                data-testid="proposal-combine-confirm"
              >
                {combine.isPending ? "Combining…" : "Create review package"}
              </button>
            </div>
          </section>
        )}
      </div>
    </main>
  );
}
