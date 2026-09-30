import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { StatusBadge } from "../findings/StatusBadge";
import { FindingStatus } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";
import type { FindingHit } from "@vrooli/proto-types/web-search/v1/findings/findings_pb";

/**
 * FindingHitList renders the learnings-corpus search results: each hit's claim,
 * its semantic-search score, a status badge, and a weak-match flag when the
 * read path marked the hit weak.
 */
export function FindingHitList({ hits }: { hits: FindingHit[] }) {
  const { t } = useTranslation();

  return (
    <ul data-testid={selectors.search.findingHits} className="space-y-2">
      {hits.map((hit, i) => {
        const finding = hit.finding;
        const status = finding?.status ?? FindingStatus.UNSPECIFIED;
        return (
          <li
            key={finding?.id ?? `hit-${i}`}
            data-testid={selectors.search.findingHit}
            className="rounded-panel border border-app-border bg-app-surface p-4"
          >
            <div className="flex items-start justify-between gap-3">
              <p className="text-sm text-app-foreground">{finding?.claim ?? ""}</p>
              <StatusBadge status={status} />
            </div>
            <p className="mt-2 text-xs text-app-muted-foreground">
              {t(strings.search.resultScore, { score: hit.score.toFixed(3) })}
              {hit.weak && (
                <span className="ms-2 rounded-pill border border-app-border px-1.5 py-0.5 align-middle">
                  {t(strings.search.weak)}
                </span>
              )}
            </p>
          </li>
        );
      })}
    </ul>
  );
}
