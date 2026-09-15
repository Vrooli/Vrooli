import { timestampDate } from "@bufbuild/protobuf/wkt";

import { Button } from "../../components/ui/button";
import type { ScoreRow } from "../../api/scoring";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";

interface FleetTableProps {
  rows: ScoreRow[];
  hasNextPage: boolean;
  loading: boolean;
  onNextPage: () => void;
}

export function FleetTable({ rows, hasNextPage, loading, onNextPage }: FleetTableProps) {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.scoring.fleet.card}
      aria-label={t(strings.scoring.fleet.title)}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 className="text-sm font-semibold uppercase text-app-muted-foreground">
          {t(strings.scoring.fleet.title)}
        </h3>
        <Button
          data-testid={selectors.scoring.fleet.next}
          type="button"
          variant="outline"
          disabled={!hasNextPage || loading}
          onClick={onNextPage}
        >
          {t(strings.scoring.fleet.next)}
        </Button>
      </div>
      {rows.length === 0 ? (
        <p data-testid={selectors.scoring.fleet.empty} className="mt-3 text-sm text-app-muted-foreground">
          {t(strings.scoring.fleet.empty)}
        </p>
      ) : (
        <div className="mt-3 overflow-x-auto">
          <table data-testid={selectors.scoring.fleet.table} className="w-full min-w-[760px] table-fixed text-sm">
            <thead>
              <tr className="border-b border-app-border text-app-muted-foreground">
                <th scope="col" className="w-52 py-2 text-start">{t(strings.scoring.fleet.scenario)}</th>
                <th scope="col" className="w-20 py-2 text-end">{t(strings.scoring.fleet.score)}</th>
                <th scope="col" className="w-44 py-2 text-start">{t(strings.scoring.fleet.rung)}</th>
                <th scope="col" className="w-28 py-2 text-end">{t(strings.scoring.fleet.priority)}</th>
                <th scope="col" className="w-40 py-2 text-start">{t(strings.scoring.fleet.calculated)}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr key={`${row.scenario}-${row.digest}`} className="border-b border-app-border/60">
                  <td className="truncate py-2 pe-3 font-medium">{row.scenario}</td>
                  <td className="py-2 text-end">{row.score}/100</td>
                  <td className="truncate py-2 pe-3">{row.workingRung}</td>
                  <td className="py-2 text-end">{row.priority.toFixed(2)}</td>
                  <td className="py-2">
                    {row.calculatedAt
                      ? formatDate(timestampDate(row.calculatedAt), { dateStyle: "medium" })
                      : t(strings.scoring.fleet.unknownDate)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
