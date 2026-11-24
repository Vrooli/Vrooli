import { useId } from "react";
import { TriangleAlert } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { ParsedHtml } from "./htmlImport";

/** Parsed document summary and parser warnings. */
export function HtmlImportOverview({ parsed }: { parsed: ParsedHtml }) {
  const { t } = useTranslation();
  const warningId = useId();

  return (
    <>
      <div className="hi-summary">
        <span className="hi-doc-title" data-testid="html-import-title">
          {parsed.title}
        </span>
        <span className="hi-counts" data-testid="html-import-counts">
          {t("workbench.studio.htmlImportCounts", {
            defaultValue: "{{layers}} layers · {{regions}} regions",
            layers: parsed.layers,
            regions: parsed.regions.length,
          })}
        </span>
      </div>
      {parsed.warnings.length > 0 ? (
        <section className="hi-warnings" aria-labelledby={warningId}>
          <h3 id={warningId} className="hi-warnings-title">
            <TriangleAlert aria-hidden size={12} />
            {t("workbench.studio.htmlImportWarnings", { defaultValue: "Warnings" })}
          </h3>
          <ul data-testid="html-import-warnings">
            {parsed.warnings.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </section>
      ) : null}
    </>
  );
}
