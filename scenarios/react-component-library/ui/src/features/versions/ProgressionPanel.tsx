/** @vrooliComponentSource visualization.cartesian-charts */
import { useQuery } from "@tanstack/react-query";
import { listVersionLedger } from "../../api/versionLedger";
import { CartesianCharts, type CartesianPoint } from "../../components/CartesianCharts";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function ProgressionPanel({ libraryId }: { libraryId: string }) {
  const { t } = useTranslation();
  const ledger = useQuery({
    queryKey: ["version-ledger", libraryId],
    queryFn: () => listVersionLedger(libraryId),
    enabled: Boolean(libraryId),
  });
  const rows = ledger.data ?? [];
  const points: CartesianPoint[] = rows.map((row) => ({
    id: row.version,
    label: row.version,
    value: Math.round(row.testPassRate * 100),
    detail: `${row.adoptionCurrent} adopters · ${row.linesOfCode} LOC${row.lifecycleState === "retired" ? " · retired" : ""}`,
  }));
  return (
    <section data-testid={selectors.versions.progressionPanel} className="space-y-space-sm">
      {ledger.isLoading ? (
        <p role="status">{t(strings.componentDetail.progression.loading)}</p>
      ) : ledger.isError ? (
        <p role="alert">{t(strings.componentDetail.progression.error)}</p>
      ) : (
        <CartesianCharts
          title={t(strings.componentDetail.progression.title)}
          description={t(strings.componentDetail.progression.description)}
          data={points}
        />
      )}
    </section>
  );
}
