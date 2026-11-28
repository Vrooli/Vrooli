import { Shapes } from "lucide-react";
import { useId } from "react";

import { useTranslation } from "../../i18n";
import { Chip } from "../primitives";
import type { TemplateNode } from "../templates/spec";
import { extractionCandidates } from "./extract";

/**
 * System-lens section: repeated custom structures worth extracting.
 *
 * The list is deterministic data from `extractionCandidates`; selecting an
 * entry is the only interaction, so the operator can inspect the example
 * before deciding to extract anything.
 */
export function StudioExtract({
  spec,
  onSelect,
}: {
  spec: TemplateNode;
  onSelect: (nodeId: string) => void;
}) {
  const { t } = useTranslation();
  const headingId = useId();
  const candidates = extractionCandidates(spec);

  return (
    <section className="studio-system-group" aria-labelledby={headingId}>
      <header className="studio-system-head">
        <h3 id={headingId}>
          {t("workbench.studio.extractionCandidates", {
            defaultValue: "Extraction candidates",
          })}
        </h3>
        <small>
          {t("workbench.studio.extractionCount", {
            defaultValue: "{{count}} found",
            count: candidates.length,
          })}
        </small>
      </header>
      <p className="studio-note">
        {t("workbench.studio.extractionExplain", {
          defaultValue:
            "Repeated custom structures that are the same shape, ranked by duplication cost.",
        })}
      </p>

      {candidates.length === 0 ? (
        <>
          <p className="studio-note">
            {t("workbench.studio.extractionEmpty", {
              defaultValue: "No repeated custom structure in this frame.",
            })}
          </p>
          <p className="studio-note">
            {t("workbench.studio.extractionEmptyHint", {
              defaultValue: "RCL primitives, pages and slots are excluded.",
            })}
          </p>
        </>
      ) : (
        <ul className="studio-extract-list">
          {candidates.map((candidate) => {
            const exampleId = candidate.exampleIds[0];
            if (!exampleId) return null;
            return (
              <li key={candidate.signature} className="studio-extract-item">
                <div className="studio-extract-head">
                  <Shapes aria-hidden size={12} className="studio-system-icon" />
                  <span className="studio-extract-label" title={candidate.label}>
                    {candidate.label}
                  </span>
                  <Chip tone="ice">
                    {t("workbench.studio.extractionScore", {
                      defaultValue: "cost {{score}}",
                      score: candidate.score,
                    })}
                  </Chip>
                </div>
                <div className="studio-extract-meta">
                  <span>
                    {t("workbench.studio.extractionUsage", {
                      defaultValue: "×{{occurrences}} · {{count}} layers",
                      occurrences: candidate.occurrences,
                      count: candidate.nodeCount,
                    })}
                  </span>
                  <button
                    type="button"
                    className="studio-link-btn"
                    aria-label={t("workbench.studio.extractionSelectLabel", {
                      defaultValue: "Select example of {{label}}",
                      label: candidate.label,
                    })}
                    onClick={() => onSelect(exampleId)}
                  >
                    {t("workbench.studio.extractionSelect", {
                      defaultValue: "Select example",
                    })}
                  </button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
