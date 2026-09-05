import { ExternalLink } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { SearchResult } from "@vrooli/proto-types/web-search/v1/livesearch/livesearch_pb";

/**
 * SnippetCard renders one live web-search result: title (linking out), snippet,
 * and the engine + raw relevance score as provenance.
 */
export function SnippetCard({ result }: { result: SearchResult }) {
  const { t } = useTranslation();

  return (
    <li
      data-testid={selectors.search.result}
      className="rounded-panel border border-app-border bg-app-surface p-4"
    >
      <a
        href={result.url}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={t(strings.search.openResult)}
        className="inline-flex items-center gap-1 font-medium text-app-primary hover:underline"
      >
        {result.title || result.url}
        <ExternalLink aria-hidden="true" className="h-3.5 w-3.5" />
      </a>
      {result.snippet && (
        <p className="mt-1 text-sm text-app-foreground">{result.snippet}</p>
      )}
      <p className="mt-2 text-xs text-app-muted-foreground">
        {result.engine && <span>{t(strings.search.resultEngine, { engine: result.engine })}</span>}
        {result.engine && " · "}
        {t(strings.search.resultScore, { score: result.score.toFixed(3) })}
      </p>
    </li>
  );
}
