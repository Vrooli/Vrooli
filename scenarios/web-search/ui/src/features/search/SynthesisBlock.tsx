import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { Synthesis } from "@vrooli/proto-types/web-search/v1/livesearch/livesearch_pb";

/**
 * SynthesisBlock renders the optional L1 synthesis over the live results: the
 * cited answer text plus its source citations, or an explicit abstention note
 * when the sources were insufficient or in conflict.
 */
export function SynthesisBlock({ synthesis }: { synthesis: Synthesis }) {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.search.synthesis}
      aria-labelledby="synthesis-heading"
      className="rounded-panel border border-app-border bg-app-surface-muted p-4"
    >
      <h3 id="synthesis-heading" className="text-sm font-semibold uppercase text-app-muted-foreground">
        {t(strings.search.synthesisHeading)}
      </h3>
      {synthesis.abstained || !synthesis.text ? (
        <p className="mt-2 text-sm text-app-muted-foreground">
          {t(strings.search.synthesisAbstained)}
        </p>
      ) : (
        <>
          <p className="mt-2 text-sm text-app-foreground">{synthesis.text}</p>
          {synthesis.citations.length > 0 && (
            <div className="mt-3">
              <p className="text-xs font-semibold uppercase text-app-muted-foreground">
                {t(strings.search.synthesisCitations)}
              </p>
              <ol className="mt-1 list-inside list-decimal text-sm">
                {synthesis.citations.map((c, i) => (
                  <li key={`${c.url}/${i}`}>
                    <a
                      href={c.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-app-primary hover:underline"
                    >
                      {c.title || c.url}
                    </a>
                  </li>
                ))}
              </ol>
            </div>
          )}
        </>
      )}
    </section>
  );
}
