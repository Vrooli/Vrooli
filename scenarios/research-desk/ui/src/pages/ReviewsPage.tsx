import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { Link } from "react-router-dom";

import { topicsUnknown, useReviews } from "../api/operator";
import { QueryView } from "../components/QueryView";
import { Unreadable } from "../components/Unreadable";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/** Review sessions written by the research-review skill. */
export function ReviewsPage() {
  const { t } = useTranslation();
  const query = useReviews();
  return (
    <div data-testid={selectors.pages.reviews} className="rd-page">
      <PageHeader headingId="reviews-heading" title={t(strings.reviews.title)} description={t(strings.reviews.description)} />
      <QueryView query={query}>
        {({ reviews, unreadable }) => (
          <>
            <Unreadable evidence={unreadable} />
            {reviews.length === 0 && unreadable.length === 0 ? (
              <EmptyState title={t(strings.reviews.emptyTitle)} description={t(strings.reviews.empty)} />
            ) : (
              <ul className="rd-rows">
                {reviews.map((r) => (
                  <li key={r.id} className="rd-row">
                    <Link to={`/reviews/${encodeURIComponent(r.id)}`} className="rd-row-link">
                      <span className="block font-semibold">{r.title?.value || r.id}</span>
                      <span className="rd-id block">{r.id}</span>
                    </Link>
                    <span className="text-body-sm text-app-muted-foreground">
                      {topicsUnknown(r)
                        ? t(strings.reviews.summaryTopicsUnknown, { verdicts: r.verdicts.length })
                        : t(strings.reviews.summary, { topics: r.topics.length, verdicts: r.verdicts.length })}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </QueryView>
    </div>
  );
}
