import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { PageHeader } from "@vrooli/react-component-library/PageHeader/2";
import { BookOpenCheck } from "lucide-react";
import { Link } from "react-router-dom";

import { topicsUnknown, useReviews } from "../api/operator";
import { IconTile, StatusPill } from "../components/daylight";
import { QueryView } from "../components/QueryView";
import { Unreadable } from "../components/Unreadable";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

const TONES = ["lavender", "peach", "mint", "blue"] as const;

/** Review sessions written by the research-review skill, as a quiet card grid. */
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
              <ul className="rd-topic-grid">
                {reviews.map((r, index) => (
                  <li key={r.id} className="rd-topic-card">
                    <Link to={`/reviews/${encodeURIComponent(r.id)}`} className="rd-topic-card-main">
                      <span className="rd-topic-top">
                        <IconTile tone={TONES[index % TONES.length] ?? "lavender"}><BookOpenCheck /></IconTile>
                        <StatusPill tone="quiet">
                          {topicsUnknown(r)
                            ? t(strings.reviews.summaryTopicsUnknown, { verdicts: t(strings.reviews.verdictsCount, { count: r.verdicts.length }) })
                            : t(strings.reviews.summaryCombined, { topics: t(strings.reviews.topicsCount, { count: r.topics.length }), verdicts: t(strings.reviews.verdictsCount, { count: r.verdicts.length }) })}
                        </StatusPill>
                      </span>
                      <span className="rd-topic-title">{r.title?.value || r.id}</span>
                      <span className="rd-topic-meta">{r.id}</span>
                    </Link>
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
