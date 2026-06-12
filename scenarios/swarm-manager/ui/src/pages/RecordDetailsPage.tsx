/**
 * RecordDetailsPage — single record view with narrative editor (for stubs)
 * and supersede-chain view.
 */

import { useCallback, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import { recordsService } from "../services/records-service";
import { RecordNarrativeEditor } from "../components/records/RecordNarrativeEditor";
import { SupersedeChainView } from "../components/records/SupersedeChainView";
import { PageLoadingState } from "../components/ui/loading-states";
import { ErrorState } from "../components/ui/error-state";
import type { RecordNarrativeInput } from "../types";

// Shared style for the in-app reference links (backlog / initiative / supersede
// chain), mirroring the emerald accent used elsewhere on this page.
const refLinkClass = "text-emerald-400 underline-offset-4 hover:underline";

export function RecordDetailsPage() {
  const { recordId } = useParams<{ recordId: string }>();
  const queryClient = useQueryClient();
  const [showChain, setShowChain] = useState(false);

  const { data: record, isLoading, error, refetch } = useQuery({
    queryKey: ["record", recordId],
    queryFn: () => recordsService.get(recordId ?? ""),
    enabled: !!recordId,
  });

  const handleFill = useCallback(
    async (input: RecordNarrativeInput) => {
      if (!recordId) return;
      await recordsService.fillNarrative(recordId, input);
      await queryClient.invalidateQueries({ queryKey: ["record", recordId] });
      await queryClient.invalidateQueries({ queryKey: ["records", "list"] });
    },
    [recordId, queryClient],
  );

  if (isLoading) return <PageLoadingState label="Loading record…" />;
  if (error)
    return (
      <ErrorState
        title="Failed to load record"
        message={error instanceof Error ? error.message : String(error)}
        onRetry={() => refetch()}
      />
    );
  if (!record) return <ErrorState title="Record not found" message={`No record with id ${recordId}`} />;

  return (
    <div className="mx-auto flex max-w-4xl flex-col gap-6 p-6">
      <header className="flex items-center gap-2">
        <h1 className="text-xl font-semibold text-slate-100">Record {record.id}</h1>
        {record.stub ? (
          <span className="rounded bg-amber-900/60 px-2 py-1 text-xs text-amber-200">stub</span>
        ) : null}
      </header>

      <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm">
        <dt className="text-slate-400">Kind</dt>
        <dd className="text-slate-100">{record.kind}</dd>
        <dt className="text-slate-400">Scenario</dt>
        <dd className="text-slate-100">{record.scenario}</dd>
        <dt className="text-slate-400">Outcome</dt>
        <dd className="text-slate-100">{record.outcome}</dd>
        {record.backlogRef ? (
          <>
            <dt className="text-slate-400">Backlog</dt>
            <dd>
              <Link to={`/backlog/${record.backlogRef}`} className={refLinkClass} data-testid="record-backlog-link">
                {record.backlogRef}
              </Link>
            </dd>
          </>
        ) : null}
        {record.initiativeId ? (
          <>
            <dt className="text-slate-400">Initiative</dt>
            <dd>
              <Link
                to={`/initiatives/${record.initiativeId}`}
                className={refLinkClass}
                data-testid="record-initiative-link"
              >
                {record.initiativeId}
              </Link>
            </dd>
          </>
        ) : null}
        {record.commit ? (
          <>
            <dt className="text-slate-400">Commit</dt>
            <dd className="font-mono text-slate-100">{record.commit}</dd>
          </>
        ) : null}
        {record.createdAt ? (
          <>
            <dt className="text-slate-400">Created</dt>
            <dd className="text-slate-100">{record.createdAt}</dd>
          </>
        ) : null}
        {record.supersedes ? (
          <>
            <dt className="text-slate-400">Supersedes</dt>
            <dd>
              <Link to={`/records/${record.supersedes}`} className={refLinkClass} data-testid="record-supersedes-link">
                {record.supersedes}
              </Link>
            </dd>
          </>
        ) : null}
        {record.supersededBy ? (
          <>
            <dt className="text-slate-400">Superseded by</dt>
            <dd>
              <Link
                to={`/records/${record.supersededBy}`}
                className={refLinkClass}
                data-testid="record-superseded-by-link"
              >
                {record.supersededBy}
              </Link>
            </dd>
          </>
        ) : null}
      </dl>

      {record.stub ? (
        <section>
          <h2 className="mb-2 text-sm font-medium uppercase tracking-wide text-slate-400">Fill narrative</h2>
          <RecordNarrativeEditor record={record} onSubmit={handleFill} />
        </section>
      ) : (
        <section>
          <h2 className="mb-2 text-sm font-medium uppercase tracking-wide text-slate-400">Narrative</h2>
          <div className="rounded border border-slate-700 bg-slate-900/60 p-3 text-sm text-slate-100">
            <p className="font-medium">Trigger</p>
            <p className="mb-2 whitespace-pre-wrap">{record.trigger}</p>
            <p className="font-medium">Approach</p>
            <p className="mb-2 whitespace-pre-wrap">{record.approach}</p>
            {record.ruledOut.length > 0 ? (
              <>
                <p className="font-medium">Ruled out</p>
                <ul className="mb-2 list-disc pl-6">
                  {record.ruledOut.map((ro, i) => (
                    <li key={i}>{ro}</li>
                  ))}
                </ul>
              </>
            ) : null}
            {record.filesChanged.length > 0 ? (
              <>
                <p className="font-medium">Files changed</p>
                <ul className="list-disc pl-6 font-mono text-xs text-slate-300">
                  {record.filesChanged.map((f, i) => (
                    <li key={i}>{f}</li>
                  ))}
                </ul>
              </>
            ) : null}
          </div>
        </section>
      )}

      <section>
        <button
          type="button"
          onClick={() => setShowChain((v) => !v)}
          className="text-sm text-emerald-400 underline-offset-4 hover:underline"
          data-testid="toggle-chain"
        >
          {showChain ? "Hide" : "View"} supersede chain
        </button>
        {showChain ? (
          <div className="mt-3">
            <SupersedeChainView rootId={record.id} />
          </div>
        ) : null}
      </section>
    </div>
  );
}
