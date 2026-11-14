/**
 * RecordsPage — list + semantic search across records.
 */

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { recordsService } from "../services/records-service";
import { RecordsList } from "../components/records/RecordsList";
import { RecordSearchBox } from "../components/records/RecordSearchBox";
import { RecordCaptureForm } from "../components/records/RecordCaptureForm";
import { PageLoadingState } from "../components/ui/loading-states";
import { ErrorState } from "../components/ui/error-state";
import type { RecordKind } from "../types";

export function RecordsPage() {
  const [kindFilter, setKindFilter] = useState<RecordKind | "">("");
  const [scenarioFilter, setScenarioFilter] = useState("");
  const [includeStubs, setIncludeStubs] = useState(false);

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["records", "list", { includeStubs }],
    queryFn: () => recordsService.list({ includeStubs, limit: 200 }),
  });

  const capture = async (input: Parameters<typeof recordsService.capture>[0]) => {
    const result = await recordsService.capture(input);
    // Published records become visible in the regular private-draft-excluding
    // list immediately. Drafts remain intentionally absent from that list.
    if (result.disposition === "published") await refetch();
    return result;
  };

  if (isLoading) return <PageLoadingState label="Loading records…" />;
  if (error)
    return (
      <ErrorState
        title="Failed to load records"
        message={error instanceof Error ? error.message : String(error)}
        onRetry={() => refetch()}
      />
    );

  return (
    <div className="mx-auto flex max-w-5xl flex-col gap-6 p-6">
      <header>
        <h1 className="text-2xl font-semibold text-slate-100">Records</h1>
        <p className="mt-1 text-sm text-slate-400">
          Narrative artifacts of completed work — the recursive-learning loop's write side.
        </p>
      </header>

      <section>
        <h2 className="mb-2 text-sm font-medium uppercase tracking-wide text-slate-400">Capture work</h2>
        <div className="rounded border border-slate-700 bg-slate-900/40 p-4">
          <RecordCaptureForm onSubmit={capture} />
        </div>
      </section>

      <section>
        <h2 className="mb-2 text-sm font-medium uppercase tracking-wide text-slate-400">Search</h2>
        <RecordSearchBox scenario={scenarioFilter || undefined} />
      </section>

      <section>
        <h2 className="mb-2 text-sm font-medium uppercase tracking-wide text-slate-400">Browse</h2>
        <RecordsList
          records={data ?? []}
          kindFilter={kindFilter}
          scenarioFilter={scenarioFilter}
          includeStubs={includeStubs}
          onKindFilterChange={setKindFilter}
          onScenarioFilterChange={setScenarioFilter}
          onIncludeStubsChange={setIncludeStubs}
        />
      </section>
    </div>
  );
}
