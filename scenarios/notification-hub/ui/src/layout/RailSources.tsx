import { Link, useLocation } from "react-router-dom";

import { strings } from "../consts/strings";
import { preserveRoutedTestMode } from "../features/routedTestMode";
import { useTranslation } from "../i18n";

const MAX_SOURCES = 6;

/**
 * The rail's "By source" list: the senders the Hub counted for the waiting
 * queue, each a link into that filtered queue. The counts come from the same
 * one-row read the Needs-you badge already uses, never a local guess.
 */
export function RailSources({ senders, total }: { senders: readonly { value: string; count: number }[]; total: number }) {
  const { t } = useTranslation();
  const { search } = useLocation();
  if (senders.length === 0) return null;
  return (
    <div className="mt-3">
      <div className="flex items-baseline justify-between gap-2 px-2.5 pb-2">
        <span className="font-mono text-[9.5px] font-semibold uppercase tracking-[0.16em] text-app-muted-foreground">
          {t(strings.layout.sources.title)}
        </span>
        <Link
          to={preserveRoutedTestMode("/asks", search)}
          className="font-mono text-[10px] text-app-muted-foreground hover:text-app-primary"
        >
          {t(strings.layout.sources.all, { count: total })}
        </Link>
      </div>
      <nav aria-label={t(strings.layout.sources.label)} className="flex flex-col gap-0.5">
        {senders.slice(0, MAX_SOURCES).map((facet) => (
          <Link
            key={facet.value}
            to={preserveRoutedTestMode(`/asks?sender=${encodeURIComponent(facet.value)}`, search)}
            className="flex min-h-11 items-center gap-2.5 rounded-control px-2.5 text-[12.5px] text-app-muted-foreground hover:bg-app-surface hover:text-app-foreground"
          >
            <span aria-hidden="true" className="h-[5px] w-[5px] shrink-0 rounded-full bg-app-border" />
            <span className="min-w-0 flex-1 truncate">{facet.value}</span>
            <b className="font-mono text-[10.5px] font-semibold text-app-muted-foreground">{facet.count}</b>
          </Link>
        ))}
      </nav>
    </div>
  );
}
