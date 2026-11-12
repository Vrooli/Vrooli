import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { useSearchParams } from "react-router-dom";

import { CommissionState, useCommissions, type CommissionSummary } from "../api/activity";
import { CommissionRow } from "../components/CommissionRow";
import { QueryView } from "../components/QueryView";
import { StateBadge } from "../components/StateBadge";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { STATE_META, STATE_ORDER } from "../lib/states";

/** States that rarely need a look; folded so the open ones stay on screen. */
const FOLDED = new Set([CommissionState.ENDED, CommissionState.UNKNOWN, CommissionState.STOPPED]);

/**
 * Every commission the desk recognises, grouped by derived state. Unknown is
 * a state, not an error: older layouts show what can be read (I6).
 */
export function CommissionsPage() {
  const { t } = useTranslation();
  const query = useCommissions();
  const [params] = useSearchParams();
  const only: CommissionState = Number(params.get("state"));
  return (
    <div data-testid={selectors.pages.commissions} className="rd-page">
      <PageHeader headingId="commissions-heading" title={t(strings.commissions.title)} description={t(strings.commissions.description)} />
      <QueryView query={query}>
        {({ commissions }) => {
          if (commissions.length === 0) return <EmptyState title={t(strings.commissions.emptyTitle)} description={t(strings.commissions.empty)} />;
          const groups = STATE_ORDER.map((state) => ({ state, list: sortByWrite(commissions.filter((c) => stateOf(c) === state)) })).filter((g) => g.list.length > 0);
          return groups.map(({ state, list }) => {
            const body = (
              <ul className="rd-rows">
                {list.map((c) => <CommissionRow key={c.id} commission={c} />)}
              </ul>
            );
            const heading = (
              <span className="inline-flex items-center gap-space-xs">
                <StateBadge state={state} />
                <span>{t(strings.commissions.count, { count: list.length })}</span>
              </span>
            );
            return FOLDED.has(state) && only !== state ? (
              <details key={state} className="rd-fold">
                <summary>{heading}</summary>
                {body}
              </details>
            ) : (
              <section key={state} className="rd-section" aria-label={t(STATE_META[state].label)}>
                <h2 className="text-subheading font-semibold">{heading}</h2>
                {body}
              </section>
            );
          });
        }}
      </QueryView>
    </div>
  );
}

function stateOf(c: CommissionSummary): CommissionState {
  return c.state === CommissionState.UNSPECIFIED ? CommissionState.UNKNOWN : c.state;
}

function sortByWrite(list: CommissionSummary[]): CommissionSummary[] {
  const at = (c: CommissionSummary) => Number(c.lastWrite?.value?.seconds ?? 0n);
  return [...list].sort((a, b) => at(b) - at(a));
}
