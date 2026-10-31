import { Link } from "react-router-dom";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { Table } from "@vrooli/react-component-library/Table/1";
import { DataState } from "../components/feedback/DataState";
import { useApiQuery } from "../hooks/useApiQuery";
import { useBoundedList } from "../hooks/useBoundedList";
import { ShowMoreButton } from "../components/patterns/ShowMoreButton";
import { apiRequest } from "../hooks/useApiTransport";
import { formatStandardRelativeTime } from "../lib/dateTime";
import { formatCode, pluralize, shortenIds } from "../lib/format";
import { statusTone } from "../lib/display";
import type { TypedInvestigation } from "../types";

// The list can hold hundreds of rows; show a page at a time.
const LIST_PAGE_SIZE = 20;

/** Short columns keep their words whole; the table scrolls sideways before it breaks "Applicability" mid-word. */
const NO_BREAK = "whitespace-nowrap break-normal";

function coverageSummary(item: TypedInvestigation): string {
  const coverage = item.result?.coverage ?? [];
  if (coverage.length === 0) return "Not yet available";
  return coverage.map((plane) => `${formatCode(plane.plane)}: ${formatCode(plane.state)}`).join(" · ");
}

function diagnosisSummary(item: TypedInvestigation): string {
  const diagnosis = item.result?.diagnosis;
  return diagnosis ? `${formatCode(diagnosis.condition)} · ${formatCode(diagnosis.disposition ?? "inconclusive")}` : "Pending";
}

function InvestigationRows({ items: all }: { items: TypedInvestigation[] }) {
  const { visibleItems: items, total, showMore } = useBoundedList(all, LIST_PAGE_SIZE);
  return (
    <>
    <Table>
      <table className="min-w-[56rem]">
        <caption className="sr-only">Durable investigations</caption>
        <thead><tr><th className={NO_BREAK}>Status</th><th className="min-w-[10rem] break-normal">Subject</th><th className="min-w-[10rem] break-normal">Diagnosis</th><th className="min-w-[12rem] break-normal">Coverage</th><th className={NO_BREAK}>Applicability</th><th className={NO_BREAK}>Updated</th></tr></thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.investigationId} data-testid={`typed-investigation-${item.investigationId}`}>
              <td className={NO_BREAK}><StatusBadge tone={statusTone(item.operationStatus)}>{formatCode(item.operationStatus)}</StatusBadge></td>
              <td className="min-w-[10rem] break-normal">
                {formatCode(item.request?.subject?.kind ?? "subject")}: {shortenIds(item.request?.subject?.ref ?? item.investigationId)}
                <div className="text-xs text-muted-foreground">{pluralize(item.request?.subject?.runIds?.length ?? 0, "run")}</div>
              </td>
              <td className="min-w-[10rem] break-normal">{diagnosisSummary(item)}</td>
              <td className="min-w-[12rem] break-normal">{coverageSummary(item)}</td>
              <td className={NO_BREAK}>{item.result?.applicability?.state ? formatCode(item.result.applicability.state) : "Not yet available"}</td>
              <td className={NO_BREAK}>{formatStandardRelativeTime(item.updatedAt)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </Table>
    <ShowMoreButton shown={items.length} total={total} onShowMore={showMore} testId="investigations-show-more" />
    </>
  );
}

export function InvestigationsPage() {
  const lifecycle = useApiQuery<TypedInvestigation[]>(
    ["investigations"],
    async (signal) => (await apiRequest<{ investigations?: TypedInvestigation[] }>("/investigations?limit=50", { signal })).investigations ?? [],
    { fallback: [] },
  );

  const items = lifecycle.data ?? [];
  return (
    <section className="h-full overflow-y-auto overflow-x-hidden p-4 sm:p-6" data-testid="investigations-page"><div className="mx-auto flex w-full max-w-6xl flex-col gap-4">
      <PageHeader
        title="Investigations"
        description="A completed subject run is not the same as a completed investigation. Review state, evidence coverage and current applicability here."
        actions={<Link to="/observability" className="inline-flex min-h-11 items-center text-sm font-medium text-primary hover:underline sm:min-h-8">Health →</Link>}
      />
      <section aria-label="Durable investigation lifecycle" className="space-y-2">
        <DataState
          subject="Investigations"
          loading={lifecycle.loading}
          error={lifecycle.error}
          isEmpty={items.length === 0}
          onRetry={lifecycle.refetch}
          empty={<EmptyState density="compact" title="No investigations yet" description="Start one from a run's Investigate action." />}
        >
          <InvestigationRows items={items} />
        </DataState>
      </section>
    </div></section>
  );
}
